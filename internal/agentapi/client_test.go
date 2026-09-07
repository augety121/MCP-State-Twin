package agentapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/augety121/mcp-state-twin/internal/agenthost"
	"github.com/augety121/mcp-state-twin/internal/task"
)

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func plan(t *testing.T) *Plan {
	t.Helper()
	data, err := os.ReadFile("../../examples/issue-tracker/agent-tasks/close-issue.json")
	if err != nil {
		t.Fatal(err)
	}
	ta, err := task.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	return &Plan{Format: PlanFormat, ID: "contract", Provider: "openai", Profile: Profile, Model: "test-model-not-a-product", Task: ta, BundleDigest: "sha256:" + strings.Repeat("a", 64), IssuedAt: now.Add(-time.Minute).Format(time.RFC3339Nano), ExpiresAt: now.Add(time.Hour).Format(time.RFC3339Nano), MaxRequests: 2, MaxOutputTokens: 128, CostPolicy: CostPolicy, Approved: true, SyntheticDataApproved: true, UnknownCostApproved: true}
}
func request(p *Plan) []byte {
	var tools []agenthost.Function
	for _, name := range p.Task.Tools {
		tools = append(tools, agenthost.Function{Type: "function", Name: name, Parameters: map[string]any{"type": "object"}})
	}
	b, _ := json.Marshal(map[string]any{"model": p.Model, "instructions": agenthost.Instructions, "input": []any{map[string]any{"role": "user", "content": "Synthetic fixture only"}}, "tools": tools, "store": false, "stream": false, "parallel_tool_calls": false, "max_output_tokens": p.MaxOutputTokens})
	return b
}
func response(body string, status int) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}
}

func TestPlanClosedApprovalAndNumericBoundary(t *testing.T) {
	p := plan(t)
	data, _ := json.Marshal(p)
	if _, err := DecodePlan(data); err != nil {
		t.Fatal(err)
	}
	for name, change := range map[string]func(*Plan){
		"model": func(p *Plan) { p.Model = "mock-test" }, "nil task": func(p *Plan) { p.Task = nil }, "request cap": func(p *Plan) { p.MaxRequests = p.Task.Budgets.ModelRequests + 1 }, "output cap": func(p *Plan) { p.MaxOutputTokens = 8193 }, "lifetime": func(p *Plan) { p.ExpiresAt = time.Now().Add(48 * time.Hour).Format(time.RFC3339Nano) }, "currency promise": func(p *Plan) { p.CostPolicy = "free" }, "endpoint provider": func(p *Plan) { p.Provider = "custom" },
	} {
		t.Run(name, func(t *testing.T) {
			p := plan(t)
			change(p)
			if p.Validate() == nil {
				t.Fatal("invalid plan accepted")
			}
		})
	}
	for _, field := range []string{"approved", "syntheticDataApproved", "unknownCostApproved"} {
		var v map[string]any
		_ = json.Unmarshal(data, &v)
		v[field] = false
		raw, _ := json.Marshal(v)
		p, err := DecodePlan(raw)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = New(p, "contract-credential", true); err == nil {
			t.Fatal("unapproved client accepted")
		}
	}
	for _, bad := range []string{strings.TrimSuffix(string(data), "}") + `,"endpoint":"https://example.invalid"}`, strings.TrimSuffix(string(data), "}") + `,"approved":true}`} {
		if _, err := DecodePlan([]byte(bad)); err == nil {
			t.Fatal("open/duplicate fields")
		}
	}
	if _, err := New(p, "contract-credential", false); err == nil {
		t.Fatal("missing explicit live flag")
	}
	if _, err := New(p, "", true); err == nil {
		t.Fatal("missing credential")
	}
	if p.Authorize(time.Now().Add(2*time.Hour), true) == nil || p.Authorize(time.Now().Add(-2*time.Hour), true) == nil {
		t.Fatal("outside approval window")
	}
}

func TestTransportFixedRoutePrivacyAndCap(t *testing.T) {
	p := plan(t)
	c, err := New(p, "contract-credential", true)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	transport := c.client.Transport.(*http.Transport)
	if transport.Proxy != nil || !transport.DisableKeepAlives || !transport.DisableCompression || c.client.CheckRedirect(nil, nil) != http.ErrUseLastResponse {
		t.Fatal("transport isolation changed")
	}
	if _, err := transport.DialContext(context.Background(), "tcp", "example.invalid:443"); err == nil {
		t.Fatal("alternate dial target")
	}
	posts := 0
	c.client.Transport = roundTrip(func(r *http.Request) (*http.Response, error) {
		posts++
		if r.URL.String() != Endpoint || r.Method != "POST" || r.GetBody != nil || r.Header.Get("Authorization") != "Bearer contract-credential" {
			t.Fatal("route/header/replay boundary")
		}
		if _, ok := r.Context().Deadline(); !ok {
			t.Fatal("missing request deadline")
		}
		return response(`{"model":"test-model-not-a-product","status":"completed","output":[],"usage":{"input_tokens":10,"output_tokens":3,"total_tokens":13}}`, 200), nil
	})
	body := request(p)
	p.MaxRequests = 16
	p.Task.Budgets.RequestSeconds = 30 // detached approval
	for i := 1; i <= 2; i++ {
		raw, receipt, err := c.Exchange(context.Background(), body)
		if err != nil || len(raw) == 0 || receipt.Sequence != i || receipt.Tokens == nil || receipt.Tokens.Total != 13 || receipt.Cost != "unknown" {
			t.Fatalf("%+v %v", receipt, err)
		}
	}
	if _, receipt, err := c.Exchange(context.Background(), body); err == nil || receipt.Sequence != 0 || posts != 2 {
		t.Fatal("request count escaped approval")
	}
}

func TestTransportFailureTaxonomyNeverRetries(t *testing.T) {
	for _, tc := range []struct {
		name, body, want string
		status           int
	}{
		{"redirect", `secret body not retained`, "PROVIDER_HTTP_ERROR", 302},
		{"auth", `secret body not retained`, "PROVIDER_HTTP_ERROR", 401},
		{"rate limit", `secret body not retained`, "PROVIDER_HTTP_ERROR", 429},
		{"server", `secret body not retained`, "PROVIDER_HTTP_ERROR", 500},
		{"duplicate", `{"model":"x","model":"y"}`, "PROVIDER_RESPONSE_INVALID", 200},
		{"bad json", `{"model":`, "PROVIDER_RESPONSE_INVALID", 200},
		{"partial usage", `{"model":"x","usage":{"input_tokens":1}}`, "PROVIDER_USAGE_INVALID", 200},
		{"bad usage", `{"model":"x","usage":{"input_tokens":1,"output_tokens":2,"total_tokens":4}}`, "PROVIDER_USAGE_INVALID", 200},
		{"private content", `{"model":"x","text":"api_key=synthetic-private-sentinel"}`, "DATA_POLICY_REJECTED", 200},
		{"oversized", strings.Repeat("x", (1<<20)+1), "PROVIDER_RESPONSE_LIMIT", 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := plan(t)
			c, err := New(p, "contract-credential", true)
			if err != nil {
				t.Fatal(err)
			}
			defer c.Close()
			posts := 0
			c.client.Transport = roundTrip(func(*http.Request) (*http.Response, error) {
				posts++
				r := response(tc.body, tc.status)
				r.Header.Set("Location", "https://example.invalid/redirect")
				return r, nil
			})
			raw, receipt, err := c.Exchange(context.Background(), request(p))
			if err == nil || err.Error() != tc.want || raw != nil || posts != 1 {
				t.Fatalf("%+v %v posts=%d", receipt, err, posts)
			}
			data, _ := json.Marshal(receipt)
			if strings.Contains(string(data), "sentinel") || strings.Contains(string(data), "secret") {
				t.Fatal("failure content leaked")
			}
			if _, _, err := c.Exchange(context.Background(), request(p)); err == nil || posts != 1 {
				t.Fatal("retried failed request")
			}
		})
	}
}

func TestTransportUnknownAcceptanceAndMissingUsage(t *testing.T) {
	p := plan(t)
	c, err := New(p, "contract-credential", true)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	c.client.Transport = roundTrip(func(*http.Request) (*http.Response, error) {
		return response(`{"model":"test-model-not-a-product","usage":null}`, 200), nil
	})
	_, receipt, err := c.Exchange(context.Background(), request(p))
	if err != nil || receipt.Tokens != nil || receipt.Cost != "unknown" {
		t.Fatal("unknown usage became measured zero")
	}
	c.client.Transport = roundTrip(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("private DNS/header/credential failure")
	})
	_, receipt, err = c.Exchange(context.Background(), request(p))
	if err == nil || err.Error() != "PROVIDER_ACCEPTANCE_UNKNOWN" || receipt.Outcome != "acceptance_unknown" || receipt.Sequence != 2 {
		t.Fatalf("%+v %v", receipt, err)
	}
}

func TestNoPostForCancellationExpiryOrInvalidBody(t *testing.T) {
	p := plan(t)
	c, err := New(p, "contract-credential", true)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	c.client.Transport = roundTrip(func(*http.Request) (*http.Response, error) { t.Fatal("unexpected network attempt"); return nil, nil })
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, r, e := c.Exchange(ctx, request(p)); e == nil || r.Sequence != 0 {
		t.Fatal("canceled admission")
	}
	for _, bad := range []string{strings.Replace(string(request(p)), `"store":false`, `"store":true`, 1), strings.TrimSuffix(string(request(p)), "}") + `,"background":true}`} {
		if _, r, e := c.Exchange(context.Background(), []byte(bad)); e == nil || r.Sequence != 0 {
			t.Fatal("unsafe request accepted")
		}
	}
	c.plan.ExpiresAt = time.Now().Add(-time.Second).Format(time.RFC3339Nano)
	if _, r, e := c.Exchange(context.Background(), request(p)); e == nil || r.Sequence != 0 {
		t.Fatal("expired approval")
	}
}

type failedReader struct{}

func (failedReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }
func (failedReader) Close() error             { return nil }

func TestTransportTruncationMediaAndInFlightCancel(t *testing.T) {
	for _, kind := range []string{"truncated", "media", "compressed", "cancel"} {
		t.Run(kind, func(t *testing.T) {
			p := plan(t)
			c, err := New(p, "contract-credential", true)
			if err != nil {
				t.Fatal(err)
			}
			defer c.Close()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			posts := 0
			c.client.Transport = roundTrip(func(r *http.Request) (*http.Response, error) {
				posts++
				if kind == "cancel" {
					cancel()
					<-r.Context().Done()
					return nil, r.Context().Err()
				}
				resp := response(`{"model":"test-model-not-a-product"}`, 200)
				switch kind {
				case "truncated":
					resp.Body = failedReader{}
				case "media":
					resp.Header.Set("Content-Type", "text/html")
				case "compressed":
					resp.Header.Set("Content-Encoding", "gzip")
				}
				return resp, nil
			})
			raw, receipt, err := c.Exchange(ctx, request(p))
			if err == nil || raw != nil || posts != 1 || receipt.Sequence != 1 || receipt.Outcome != "acceptance_unknown" {
				t.Fatalf("%+v %v", receipt, err)
			}
			if _, _, err = c.Exchange(context.Background(), request(p)); err == nil || posts != 1 {
				t.Fatal("late retry after failure")
			}
		})
	}
}
