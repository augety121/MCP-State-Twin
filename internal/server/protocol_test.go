package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testClientMeta = `"_meta":{
  "io.modelcontextprotocol/protocolVersion":"2026-07-28",
  "io.modelcontextprotocol/clientInfo":{"name":"statetwin-wire-test","version":"1.0.0"},
  "io.modelcontextprotocol/clientCapabilities":{}
}`

func TestProtocolEvidencePinsSDKAndProfiles(t *testing.T) {
	evidence := CurrentProtocolEvidence()
	if evidence.SDKVersion != MCPGoSDKVersion {
		t.Fatalf("MCP Go SDK version = %q, want %s", evidence.SDKVersion, MCPGoSDKVersion)
	}
	if len(evidence.TestedProfiles) != 2 ||
		evidence.TestedProfiles[0].Version != ModernProtocolVersion ||
		evidence.TestedProfiles[1].Version != LegacyProtocolVersion {
		t.Fatalf("unexpected tested profiles: %#v", evidence.TestedProfiles)
	}
	moduleFile, err := os.ReadFile(filepath.Join("..", "..", "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	wantPin := MCPGoSDKModule + " " + MCPGoSDKVersion
	if !strings.Contains(string(moduleFile), wantPin) {
		t.Fatalf("go.mod does not contain declared SDK pin %q", wantPin)
	}
}

func TestMCP20260728DiscoverWireContract(t *testing.T) {
	runtime, _ := referenceRuntime(t)
	httpServer := httptest.NewServer(NewDataPlane(runtime))
	t.Cleanup(httpServer.Close)

	body := `{"jsonrpc":"2.0","id":1,"method":"server/discover","params":{` + testClientMeta + `}}`
	response := postModern(t, httpServer.URL+"/mcp/main", "server/discover", "", body)
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("discover status = %d, body = %s", response.StatusCode, readBody(t, response))
	}
	if got := response.Header.Get("Mcp-Session-Id"); got != "" {
		t.Fatalf("modern discover returned forbidden session ID %q", got)
	}
	result := decodeRPCResult(t, response)
	if result["resultType"] != "complete" {
		t.Fatalf("discover resultType = %v, want complete", result["resultType"])
	}
	versions, ok := result["supportedVersions"].([]any)
	if !ok || !containsString(versions, ModernProtocolVersion) {
		t.Fatalf("discover supportedVersions = %#v, missing %s", result["supportedVersions"], ModernProtocolVersion)
	}
	meta, ok := result["_meta"].(map[string]any)
	if !ok {
		t.Fatalf("discover response metadata = %#v", result["_meta"])
	}
	serverInfo, ok := meta["io.modelcontextprotocol/serverInfo"].(map[string]any)
	if !ok || serverInfo["version"] != Version {
		t.Fatalf("discover serverInfo = %#v", meta["io.modelcontextprotocol/serverInfo"])
	}
}

func TestMCP20260728DirectFirstToolsList(t *testing.T) {
	runtime, _ := referenceRuntime(t)
	httpServer := httptest.NewServer(NewDataPlane(runtime))
	t.Cleanup(httpServer.Close)

	body := `{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{` + testClientMeta + `}}`
	response := postModern(t, httpServer.URL+"/mcp/main", "tools/list", "", body)
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("direct tools/list status = %d, body = %s", response.StatusCode, readBody(t, response))
	}
	if got := response.Header.Get("Mcp-Session-Id"); got != "" {
		t.Fatalf("modern tools/list returned forbidden session ID %q", got)
	}
	result := decodeRPCResult(t, response)
	if result["resultType"] != "complete" {
		t.Fatalf("tools/list resultType = %v, want complete", result["resultType"])
	}
	tools, ok := result["tools"].([]any)
	if !ok || len(tools) != len(runtime.Spec().Tools) {
		t.Fatalf("tools/list tools = %#v, want %d tools", result["tools"], len(runtime.Spec().Tools))
	}
}

func TestMCP20260728RejectsHeaderBodyVersionMismatch(t *testing.T) {
	runtime, _ := referenceRuntime(t)
	httpServer := httptest.NewServer(NewDataPlane(runtime))
	t.Cleanup(httpServer.Close)

	body := `{"jsonrpc":"2.0","id":3,"method":"tools/list","params":{"_meta":{` +
		`"io.modelcontextprotocol/protocolVersion":"2025-11-25",` +
		`"io.modelcontextprotocol/clientInfo":{"name":"statetwin-wire-test","version":"1.0.0"},` +
		`"io.modelcontextprotocol/clientCapabilities":{}}}}`
	response := postModern(t, httpServer.URL+"/mcp/main", "tools/list", "", body)
	defer response.Body.Close()
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("version mismatch status = %d, want 400; body = %s", response.StatusCode, readBody(t, response))
	}
}

func TestMCP20251125LegacyInitializeCompatibility(t *testing.T) {
	runtime, _ := referenceRuntime(t)
	httpServer := httptest.NewServer(NewDataPlane(runtime))
	t.Cleanup(httpServer.Close)

	body := `{"jsonrpc":"2.0","id":4,"method":"initialize","params":{` +
		`"protocolVersion":"2025-11-25","capabilities":{},` +
		`"clientInfo":{"name":"statetwin-legacy-test","version":"1.0.0"}}}`
	request, err := http.NewRequest(http.MethodPost, httpServer.URL+"/mcp/main", bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("legacy initialize status = %d, body = %s", response.StatusCode, readBody(t, response))
	}
	result := decodeRPCResult(t, response)
	if result["protocolVersion"] != LegacyProtocolVersion {
		t.Fatalf("legacy negotiated version = %v, want %s", result["protocolVersion"], LegacyProtocolVersion)
	}
}

func postModern(t *testing.T, endpoint, method, name, body string) *http.Response {
	t.Helper()
	request, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	request.Header.Set("Mcp-Protocol-Version", ModernProtocolVersion)
	request.Header.Set("Mcp-Method", method)
	if name != "" {
		request.Header.Set("Mcp-Name", name)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	return response
}

func decodeRPCResult(t *testing.T, response *http.Response) map[string]any {
	t.Helper()
	var envelope struct {
		Result map[string]any `json:"result"`
		Error  any            `json:"error"`
	}
	decoder := json.NewDecoder(response.Body)
	if err := decoder.Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Error != nil {
		t.Fatalf("unexpected JSON-RPC error: %#v", envelope.Error)
	}
	if envelope.Result == nil {
		t.Fatal("JSON-RPC result is missing")
	}
	return envelope.Result
}

func readBody(t *testing.T, response *http.Response) string {
	t.Helper()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func containsString(values []any, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
