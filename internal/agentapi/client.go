package agentapi

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/augety121/mcp-state-twin/internal/agenthost"
	"github.com/augety121/mcp-state-twin/internal/logging"
)

// Receipt is deliberately content-free. Request IDs and headers are private.
// Unknown token usage/cost is never encoded as a measured zero.
type Receipt struct {
	Sequence      int     `json:"sequence"`
	AdmittedAt    string  `json:"admittedAt,omitempty"`
	Outcome       string  `json:"outcome"`
	HTTPStatus    int     `json:"httpStatus,omitempty"`
	ReportedModel string  `json:"reportedModel,omitempty"`
	Tokens        *Tokens `json:"tokens,omitempty"`
	Cost          string  `json:"cost"`
}
type Tokens struct {
	Input  int64 `json:"input"`
	Output int64 `json:"output"`
	Total  int64 `json:"total"`
}

type Client struct {
	mu      sync.Mutex
	plan    Plan
	key     string
	client  *http.Client
	sent    int
	stopped bool
}

func New(p *Plan, key string, allowLive bool) (*Client, error) {
	if err := p.Authorize(time.Now(), allowLive); err != nil {
		return nil, err
	}
	if len(key) < 1 || len(key) > 1024 || strings.ContainsAny(key, "\r\n\t ") {
		return nil, errors.New("PROVIDER_CREDENTIAL_MISSING_OR_INVALID")
	}
	for _, r := range key {
		if r < 33 || r > 126 {
			return nil, errors.New("PROVIDER_CREDENTIAL_MISSING_OR_INVALID")
		}
	}
	timeout := time.Duration(p.Task.Budgets.RequestSeconds) * time.Second
	dialer := &net.Dialer{Timeout: timeout, KeepAlive: -1}
	transport := &http.Transport{Proxy: nil, DisableKeepAlives: true, DisableCompression: true, MaxConnsPerHost: 1, MaxResponseHeaderBytes: 16 << 10, TLSHandshakeTimeout: timeout, ResponseHeaderTimeout: timeout, TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12}}
	transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		if addr != "api.openai.com:443" {
			return nil, errors.New("PROVIDER_ROUTE_REFUSED")
		}
		return dialer.DialContext(ctx, network, addr)
	}
	client := &http.Client{Transport: transport, Timeout: timeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	// Detach caller-owned task/configuration before storing an approval.
	data, _ := json.Marshal(p)
	copyPlan, err := DecodePlan(data)
	if err != nil {
		return nil, err
	}
	return &Client{plan: *copyPlan, key: key, client: client}, nil
}

func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stopped = true
	c.key = ""
	c.client.CloseIdleConnections()
}

// Exchange performs at most one POST and never polls/retries/cancels remotely.
// The mutex serializes requests. This is a local count cap, not an account-wide
// monetary limit or an exactly-once provider contract.
func (c *Client) Exchange(ctx context.Context, body []byte) (raw []byte, receipt Receipt, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	receipt.Cost = "unknown"
	if c.stopped {
		return nil, receipt, errors.New("PROVIDER_STOPPED")
	}
	if err = c.plan.Authorize(time.Now(), true); err != nil {
		return nil, receipt, err
	}
	if err = ctx.Err(); err != nil {
		return nil, receipt, err
	}
	if c.sent >= c.plan.MaxRequests {
		c.stopped = true
		return nil, receipt, errors.New("BUDGET_EXHAUSTED")
	}
	var request struct {
		Model        string               `json:"model"`
		Instructions string               `json:"instructions"`
		Input        []json.RawMessage    `json:"input"`
		Tools        []agenthost.Function `json:"tools"`
		Store        *bool                `json:"store"`
		Stream       *bool                `json:"stream"`
		Parallel     *bool                `json:"parallel_tool_calls"`
		MaxTokens    int                  `json:"max_output_tokens"`
	}
	if agenthost.DecodeDocument(body, c.plan.Task.Budgets.TraceBytes, &request) != nil || request.Model != c.plan.Model || request.Instructions != agenthost.Instructions || request.Store == nil || *request.Store || request.Stream == nil || *request.Stream || request.Parallel == nil || *request.Parallel || request.MaxTokens != c.plan.MaxOutputTokens || len(request.Input) == 0 || len(request.Tools) != len(c.plan.Task.Tools) {
		return nil, receipt, errors.New("PROVIDER_REQUEST_REFUSED")
	}
	for i, tool := range request.Tools {
		if tool.Type != "function" || tool.Name != c.plan.Task.Tools[i] || tool.Strict {
			return nil, receipt, errors.New("PROVIDER_REQUEST_REFUSED")
		}
	}
	if logging.ContainsSensitive(string(body)) {
		return nil, receipt, errors.New("DATA_POLICY_REJECTED")
	}
	requestCtx, cancel := context.WithTimeout(ctx, time.Duration(c.plan.Task.Budgets.RequestSeconds)*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(requestCtx, http.MethodPost, Endpoint, bytes.NewReader(body))
	req.GetBody = nil // no body replay, even if a future transport changes defaults
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.key)
	// Parsing/privacy admission can consume time. Recheck at the actual POST
	// boundary, not just when Exchange was first entered.
	admittedAt := time.Now().UTC()
	if err = c.plan.Authorize(admittedAt, true); err != nil {
		return nil, receipt, err
	}
	if err = requestCtx.Err(); err != nil {
		return nil, receipt, err
	}
	c.sent++
	receipt.Sequence = c.sent
	receipt.AdmittedAt = admittedAt.Format(time.RFC3339Nano)
	receipt.Outcome = "acceptance_unknown"
	resp, requestErr := c.client.Do(req)
	if requestErr != nil {
		c.stopped = true
		return nil, receipt, errors.New("PROVIDER_ACCEPTANCE_UNKNOWN")
	}
	defer resp.Body.Close()
	receipt.HTTPStatus = resp.StatusCode
	if resp.StatusCode != http.StatusOK {
		c.stopped = true
		receipt.Outcome = "http_error"
		return nil, receipt, errors.New("PROVIDER_HTTP_ERROR")
	}
	media, _, mediaErr := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if mediaErr != nil || media != "application/json" || resp.Header.Get("Content-Encoding") != "" {
		c.stopped = true
		return nil, receipt, errors.New("PROVIDER_RESPONSE_INVALID")
	}
	raw, err = io.ReadAll(io.LimitReader(resp.Body, int64(c.plan.Task.Budgets.ResponseBytes)+1))
	if err != nil {
		c.stopped = true
		return nil, receipt, errors.New("PROVIDER_ACCEPTANCE_UNKNOWN")
	}
	if len(raw) > c.plan.Task.Budgets.ResponseBytes {
		c.stopped = true
		return nil, receipt, errors.New("PROVIDER_RESPONSE_LIMIT")
	}
	var response struct {
		Model string `json:"model"`
		Usage *struct {
			Input  *int64 `json:"input_tokens"`
			Output *int64 `json:"output_tokens"`
			Total  *int64 `json:"total_tokens"`
		} `json:"usage"`
	}
	// Provider envelopes have forward-compatible metadata; the codec strictly
	// validates executable output items separately. Duplicate JSON keys fail.
	var object map[string]any
	if agenthost.DecodeDocument(raw, c.plan.Task.Budgets.ResponseBytes, &object) != nil || json.Unmarshal(raw, &response) != nil || !ValidModel(response.Model) {
		c.stopped = true
		return nil, receipt, errors.New("PROVIDER_RESPONSE_INVALID")
	}
	if logging.ContainsSensitive(string(raw)) {
		c.stopped = true
		return nil, receipt, errors.New("DATA_POLICY_REJECTED")
	}
	receipt.ReportedModel = response.Model
	if response.Usage != nil {
		u := response.Usage
		if u.Input == nil || u.Output == nil || u.Total == nil {
			c.stopped = true
			return nil, receipt, errors.New("PROVIDER_USAGE_INVALID")
		}
		tokens := &Tokens{Input: *u.Input, Output: *u.Output, Total: *u.Total}
		if !ValidTokens(tokens) {
			c.stopped = true
			return nil, receipt, errors.New("PROVIDER_USAGE_INVALID")
		}
		receipt.Tokens = tokens
	}
	receipt.Outcome = "response_received"
	return raw, receipt, nil
}

func ValidTokens(t *Tokens) bool {
	return t == nil || (t.Input >= 0 && t.Output >= 0 && t.Input <= 1<<40 && t.Output <= 1<<40 && t.Total == t.Input+t.Output)
}
