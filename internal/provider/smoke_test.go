package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestOpenAISmokeUsesBackgroundPollingAndMCPContract(t *testing.T) {
	var polled atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer synthetic-openai-key" {
			t.Fatal("missing OpenAI bearer authentication")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("x-request-id", "req_synthetic")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/responses":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body["background"] != true || body["model"] != "gpt-test" {
				t.Fatalf("unexpected OpenAI body: %+v", body)
			}
			_, _ = w.Write([]byte(`{"id":"resp_synthetic","status":"in_progress","output":[]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/responses/resp_synthetic":
			polled.Add(1)
			_, _ = w.Write([]byte(`{"id":"resp_synthetic","status":"completed","output":[{"type":"mcp_list_tools","tools":[{"name":"get_issue"}]},{"type":"mcp_call","name":"get_issue","error":null,"output":"{}"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	report, err := Run(context.Background(), Request{
		Provider: "openai", Model: "gpt-test", RuntimeVersion: "test", RuntimeRevision: strings.Repeat("a", 40), Prompt: "Use get_issue exactly once.",
		MCPServerURL: server.URL + "/mcp", APIKey: "synthetic-openai-key", BaseURL: server.URL + "/v1",
		Timeout: time.Second, PollInterval: time.Millisecond, HTTPClient: server.Client(), allowHTTPForTest: true,
		SyntheticOnly: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if polled.Load() != 1 || report.Outcome != "completed" || !report.MCPListToolsObserved || report.MCPToolCalls != 1 || report.Capabilities.RemoteCancel != "supported" || report.Capabilities.CancelIdempotent != "unknown" {
		t.Fatalf("OpenAI report = %+v", report)
	}
	encoded, _ := json.Marshal(report)
	if strings.Contains(string(encoded), "synthetic-openai-key") {
		t.Fatal("provider report persisted API key")
	}
}

func TestAnthropicSmokeUsesCurrentMCPConnectorBetaShape(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/messages" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("x-api-key") != "synthetic-anthropic-key" || r.Header.Get("anthropic-version") != "2023-06-01" || r.Header.Get("anthropic-beta") != "mcp-client-2025-11-20" {
			t.Fatal("Anthropic version/beta/auth headers are incomplete")
		}
		var body struct {
			MCPServers []map[string]any `json:"mcp_servers"`
			Tools      []map[string]any `json:"tools"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if len(body.MCPServers) != 1 || body.MCPServers[0]["type"] != "url" || len(body.Tools) != 1 || body.Tools[0]["type"] != "mcp_toolset" {
			t.Fatalf("unexpected Anthropic MCP body: %+v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("request-id", "req_anthropic_synthetic")
		_, _ = w.Write([]byte(`{"id":"msg_synthetic","stop_reason":"end_turn","content":[{"type":"mcp_tool_use","id":"m1"},{"type":"mcp_tool_result","tool_use_id":"m1","is_error":false}]}`))
	}))
	defer server.Close()
	report, err := Run(context.Background(), Request{
		Provider: "anthropic", Model: "claude-test", RuntimeVersion: "test", RuntimeRevision: strings.Repeat("b", 40), Prompt: "Use get_issue exactly once.",
		MCPServerURL: server.URL + "/mcp", APIKey: "synthetic-anthropic-key", BaseURL: server.URL + "/v1",
		Timeout: time.Second, HTTPClient: server.Client(), allowHTTPForTest: true,
		SyntheticOnly: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.Outcome != "completed" || report.MCPToolCalls != 1 || report.MCPToolResults != 1 || report.Capabilities.RemoteCancel != "disconnect-only" || report.Capabilities.RemoteMCPTools != "beta" {
		t.Fatalf("Anthropic report = %+v", report)
	}
}

func TestOpenAICancellationCallsRemoteCancelAndDoesNotExposeErrorBody(t *testing.T) {
	var cancelled atomic.Bool
	created := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/responses":
			_, _ = w.Write([]byte(`{"id":"resp_cancel","status":"in_progress","output":[]}`))
			close(created)
		case r.Method == http.MethodPost && r.URL.Path == "/v1/responses/resp_cancel/cancel":
			cancelled.Store(true)
			_, _ = w.Write([]byte(`{"id":"resp_cancel","status":"cancelled","output":[]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		_, err := Run(ctx, Request{
			Provider: "openai", Model: "gpt-test", RuntimeVersion: "test", RuntimeRevision: strings.Repeat("c", 40), Prompt: "cancel", MCPServerURL: server.URL + "/mcp",
			APIKey: "synthetic-key", BaseURL: server.URL + "/v1", Timeout: time.Second,
			PollInterval: time.Second, HTTPClient: server.Client(), allowHTTPForTest: true,
			SyntheticOnly: true,
		})
		result <- err
	}()
	<-created
	time.Sleep(20 * time.Millisecond)
	cancel()
	err := <-result
	if err == nil || !cancelled.Load() {
		t.Fatalf("cancellation error=%v remoteCancel=%v", err, cancelled.Load())
	}
}

func TestProviderHTTPErrorDoesNotIncludeBodyOrCredential(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"synthetic-private-provider-detail"}`))
	}))
	defer server.Close()
	_, err := Run(context.Background(), Request{
		Provider: "anthropic", Model: "claude-test", RuntimeVersion: "test", RuntimeRevision: strings.Repeat("d", 40), Prompt: "test", MCPServerURL: server.URL + "/mcp",
		APIKey: "synthetic-key", BaseURL: server.URL, Timeout: time.Second, HTTPClient: server.Client(), allowHTTPForTest: true,
		SyntheticOnly: true,
	})
	if err == nil || strings.Contains(err.Error(), "synthetic-private-provider-detail") || strings.Contains(err.Error(), "synthetic-key") {
		t.Fatalf("unsafe provider error = %v", err)
	}
}

func TestProviderSmokeRequiresImmutableSourceRevision(t *testing.T) {
	_, err := Run(context.Background(), Request{
		Provider: "openai", Model: "gpt-test", RuntimeVersion: "test", RuntimeRevision: "unknown",
		Prompt: "test", MCPServerURL: "https://synthetic.example/mcp", APIKey: "synthetic-key",
		Timeout: time.Second, SyntheticOnly: true,
	})
	if err == nil || !strings.Contains(err.Error(), "runtime identity") {
		t.Fatalf("immutable-revision error = %v", err)
	}
}
