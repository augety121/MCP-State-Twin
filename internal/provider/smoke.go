package provider

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/augety121/mcp-state-twin/internal/canonical"
)

const (
	APIVersion       = "statetwin.dev/v1alpha1"
	Format           = "statetwin.dev/provider-smoke-report/v1alpha1"
	maxProviderBody  = 8 << 20
	defaultOpenAIURL = "https://api.openai.com/v1"
	defaultClaudeURL = "https://api.anthropic.com/v1"
)

type Request struct {
	Provider         string
	Model            string
	RuntimeVersion   string
	RuntimeRevision  string
	Prompt           string
	MCPServerURL     string
	MCPAuthorization string
	APIKey           string
	BaseURL          string
	Timeout          time.Duration
	PollInterval     time.Duration
	HTTPClient       *http.Client
	SyntheticOnly    bool
	allowHTTPForTest bool
}

type Capabilities struct {
	BackgroundExecution string `json:"backgroundExecution"`
	Retrieve            string `json:"retrieve"`
	RemoteCancel        string `json:"remoteCancel"`
	CancelIdempotent    string `json:"cancelIdempotent"`
	RequestID           string `json:"requestId"`
	RemoteMCPTools      string `json:"remoteMcpTools"`
	ProviderIdempotency string `json:"providerIdempotency"`
}

type Report struct {
	APIVersion              string       `json:"apiVersion"`
	Kind                    string       `json:"kind"`
	Format                  string       `json:"format"`
	CreatedAt               string       `json:"createdAt"`
	Provider                string       `json:"provider"`
	Model                   string       `json:"model"`
	RuntimeVersion          string       `json:"runtimeVersion"`
	RuntimeRevision         string       `json:"runtimeRevision"`
	PromptDigest            string       `json:"promptDigest"`
	MCPServerURLDigest      string       `json:"mcpServerUrlDigest"`
	ProviderRequestIDDigest string       `json:"providerRequestIdDigest"`
	ResponseDigest          string       `json:"responseDigest"`
	ProviderStatus          string       `json:"providerStatus"`
	Outcome                 string       `json:"outcome"`
	MCPListToolsObserved    bool         `json:"mcpListToolsObserved"`
	MCPToolCalls            int          `json:"mcpToolCalls"`
	MCPToolResults          int          `json:"mcpToolResults"`
	MCPToolErrors           int          `json:"mcpToolErrors"`
	DurationMS              int64        `json:"durationMs"`
	Capabilities            Capabilities `json:"capabilities"`
	SyntheticOnly           bool         `json:"syntheticOnly"`
	SecretsPersisted        bool         `json:"secretsPersisted"`
}

func Run(ctx context.Context, request Request) (*Report, error) {
	if request.Provider != "openai" && request.Provider != "anthropic" {
		return nil, errors.New("provider must be openai or anthropic")
	}
	if strings.TrimSpace(request.Model) == "" || strings.TrimSpace(request.RuntimeVersion) == "" || !validRevision(request.RuntimeRevision) || strings.TrimSpace(request.Prompt) == "" || request.APIKey == "" {
		return nil, errors.New("model, runtime identity, prompt, and provider API key are required")
	}
	if !request.SyntheticOnly {
		return nil, errors.New("provider smoke requires an explicit synthetic-only attestation")
	}
	if err := validateRemoteURL(request.MCPServerURL, request.allowHTTPForTest); err != nil {
		return nil, err
	}
	if request.Timeout <= 0 || request.Timeout > 30*time.Minute {
		return nil, errors.New("provider timeout must be within 1ns..30m")
	}
	if request.PollInterval <= 0 {
		request.PollInterval = 2 * time.Second
	}
	if request.HTTPClient == nil {
		request.HTTPClient = &http.Client{Timeout: request.Timeout}
	}
	started := time.Now()
	var report *Report
	var err error
	if request.Provider == "openai" {
		report, err = runOpenAI(ctx, request)
	} else {
		report, err = runAnthropic(ctx, request)
	}
	if report != nil {
		report.DurationMS = time.Since(started).Milliseconds()
	}
	return report, err
}

func runOpenAI(ctx context.Context, request Request) (*Report, error) {
	baseURL := strings.TrimRight(request.BaseURL, "/")
	if baseURL == "" {
		baseURL = defaultOpenAIURL
	}
	tool := map[string]any{
		"type": "mcp", "server_label": "statetwin", "server_description": "Synthetic MCP State Twin evaluation server",
		"server_url": request.MCPServerURL, "require_approval": "never",
	}
	if request.MCPAuthorization != "" {
		tool["authorization"] = request.MCPAuthorization
	}
	payload := map[string]any{"model": request.Model, "input": request.Prompt, "background": true, "tools": []any{tool}}
	data, headerID, err := providerJSON(ctx, request.HTTPClient, http.MethodPost, baseURL+"/responses", request.APIKey, "openai", payload)
	if err != nil {
		return nil, err
	}
	var response openAIResponse
	if err := json.Unmarshal(data, &response); err != nil || response.ID == "" || response.Status == "" {
		return nil, errors.New("OpenAI response envelope is invalid")
	}
	for response.Status == "queued" || response.Status == "in_progress" {
		select {
		case <-ctx.Done():
			cancelCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			_, _, _ = providerJSON(cancelCtx, request.HTTPClient, http.MethodPost, baseURL+"/responses/"+url.PathEscape(response.ID)+"/cancel", request.APIKey, "openai", map[string]any{})
			cancel()
			return nil, ctx.Err()
		case <-time.After(request.PollInterval):
		}
		data, headerID, err = providerJSON(ctx, request.HTTPClient, http.MethodGet, baseURL+"/responses/"+url.PathEscape(response.ID), request.APIKey, "openai", nil)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(data, &response); err != nil {
			return nil, errors.New("OpenAI poll response is invalid")
		}
	}
	report := baseReport(request, response.ID, headerID, data)
	report.ProviderStatus = response.Status
	// The Responses API documents a cancellation endpoint, but that alone does
	// not establish the semantics of repeating a cancellation request. Keep the
	// idempotency capability unknown until first-party documentation and live
	// evidence establish it for this exact provider profile.
	report.Capabilities = Capabilities{BackgroundExecution: "supported", Retrieve: "supported", RemoteCancel: "supported", CancelIdempotent: "unknown", RequestID: observedRequestID(response.ID, headerID), RemoteMCPTools: "supported", ProviderIdempotency: "unknown"}
	for _, raw := range response.Output {
		var item struct {
			Type  string          `json:"type"`
			Error json.RawMessage `json:"error"`
		}
		if json.Unmarshal(raw, &item) != nil {
			continue
		}
		switch item.Type {
		case "mcp_list_tools":
			report.MCPListToolsObserved = true
		case "mcp_call":
			report.MCPToolCalls++
			if len(item.Error) != 0 && string(item.Error) != "null" {
				report.MCPToolErrors++
			}
		}
	}
	if response.Status == "completed" && report.MCPListToolsObserved && report.MCPToolCalls > 0 && report.MCPToolErrors == 0 {
		report.Outcome = "completed"
		return report, nil
	}
	report.Outcome = "failed"
	return report, errors.New("OpenAI live smoke did not complete a successful MCP tool call")
}

func runAnthropic(ctx context.Context, request Request) (*Report, error) {
	baseURL := strings.TrimRight(request.BaseURL, "/")
	if baseURL == "" {
		baseURL = defaultClaudeURL
	}
	server := map[string]any{"type": "url", "url": request.MCPServerURL, "name": "statetwin"}
	if request.MCPAuthorization != "" {
		server["authorization_token"] = request.MCPAuthorization
	}
	payload := map[string]any{
		"model": request.Model, "max_tokens": 2048,
		"messages":    []any{map[string]any{"role": "user", "content": request.Prompt}},
		"mcp_servers": []any{server},
		"tools":       []any{map[string]any{"type": "mcp_toolset", "mcp_server_name": "statetwin"}},
	}
	data, headerID, err := providerJSON(ctx, request.HTTPClient, http.MethodPost, baseURL+"/messages", request.APIKey, "anthropic", payload)
	if err != nil {
		return nil, err
	}
	var response struct {
		ID         string `json:"id"`
		StopReason string `json:"stop_reason"`
		Content    []struct {
			Type    string `json:"type"`
			IsError bool   `json:"is_error"`
		} `json:"content"`
	}
	if err := json.Unmarshal(data, &response); err != nil || response.ID == "" {
		return nil, errors.New("Anthropic response envelope is invalid")
	}
	report := baseReport(request, response.ID, headerID, data)
	report.ProviderStatus = response.StopReason
	report.Capabilities = Capabilities{BackgroundExecution: "unsupported", Retrieve: "unsupported", RemoteCancel: "disconnect-only", CancelIdempotent: "unknown", RequestID: observedRequestID(response.ID, headerID), RemoteMCPTools: "beta", ProviderIdempotency: "unknown"}
	for _, item := range response.Content {
		switch item.Type {
		case "mcp_tool_use":
			report.MCPToolCalls++
		case "mcp_tool_result":
			report.MCPToolResults++
			if item.IsError {
				report.MCPToolErrors++
			}
		}
	}
	if report.MCPToolCalls > 0 && report.MCPToolResults > 0 && report.MCPToolErrors == 0 && response.StopReason != "" {
		report.Outcome = "completed"
		return report, nil
	}
	report.Outcome = "failed"
	return report, errors.New("Anthropic live smoke did not complete a successful MCP tool call")
}

type openAIResponse struct {
	ID     string            `json:"id"`
	Status string            `json:"status"`
	Output []json.RawMessage `json:"output"`
}

func providerJSON(ctx context.Context, client *http.Client, method, endpoint, apiKey, provider string, payload any) ([]byte, string, error) {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return nil, "", err
		}
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if provider == "openai" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	} else {
		req.Header.Set("x-api-key", apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")
		req.Header.Set("anthropic-beta", "mcp-client-2025-11-20")
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("%s provider request failed", provider)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxProviderBody+1))
	if err != nil {
		return nil, "", fmt.Errorf("read %s provider response", provider)
	}
	if len(data) > maxProviderBody {
		return nil, "", fmt.Errorf("%s provider response exceeds limit", provider)
	}
	requestID := resp.Header.Get("x-request-id")
	if requestID == "" {
		requestID = resp.Header.Get("request-id")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, requestID, fmt.Errorf("%s provider returned HTTP %d", provider, resp.StatusCode)
	}
	return data, requestID, nil
}

func baseReport(request Request, bodyID, headerID string, response []byte) *Report {
	created := time.Now().UTC().Format(time.RFC3339)
	return &Report{
		APIVersion: APIVersion, Kind: "ProviderSmokeReport", Format: Format, CreatedAt: created,
		Provider: request.Provider, Model: request.Model,
		RuntimeVersion: request.RuntimeVersion, RuntimeRevision: request.RuntimeRevision,
		PromptDigest: digestString(request.Prompt), MCPServerURLDigest: digestString(request.MCPServerURL),
		ProviderRequestIDDigest: digestString(bodyID + "\x00" + headerID), ResponseDigest: digestBytes(response),
		SyntheticOnly: true, SecretsPersisted: false,
	}
}

func (r *Report) Digest() (string, error) { return canonical.Digest(r) }

func validateRemoteURL(value string, allowHTTP bool) error {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return errors.New("MCP server URL must be absolute without credentials, query, or fragment")
	}
	if parsed.Scheme != "https" && !(allowHTTP && parsed.Scheme == "http") {
		return errors.New("MCP server URL must use HTTPS")
	}
	return nil
}

func digestString(value string) string { return digestBytes([]byte(value)) }

func validRevision(value string) bool {
	if len(value) != 40 && len(value) != 64 {
		return false
	}
	for _, character := range value {
		if !((character >= '0' && character <= '9') || (character >= 'a' && character <= 'f')) {
			return false
		}
	}
	return true
}

func observedRequestID(bodyID, headerID string) string {
	switch {
	case bodyID != "" && headerID != "":
		return "both"
	case bodyID != "":
		return "body"
	case headerID != "":
		return "header"
	default:
		return "unavailable"
	}
}

func digestBytes(value []byte) string {
	sum := sha256.Sum256(value)
	return "sha256:" + hex.EncodeToString(sum[:])
}
