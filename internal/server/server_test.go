package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/augety121/mcp-state-twin/internal/engine"
	"github.com/augety121/mcp-state-twin/internal/spec"
	"github.com/augety121/mcp-state-twin/internal/store"
	"github.com/augety121/mcp-state-twin/internal/world"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func referenceRuntime(t *testing.T) (*engine.Runtime, *store.Store) {
	t.Helper()
	root := filepath.Join("..", "..")
	twin, err := spec.Load(filepath.Join(root, "examples", "issue-tracker", "twin.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	fixtureBytes, err := os.ReadFile(filepath.Join(root, "examples", "issue-tracker", "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	var initial world.State
	if err := json.Unmarshal(fixtureBytes, &initial); err != nil {
		t.Fatal(err)
	}
	stateStore, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stateStore.Close() })
	runtime, err := engine.New(twin, stateStore)
	if err != nil {
		t.Fatal(err)
	}
	if err := runtime.Initialize(context.Background(), "main", &initial); err != nil {
		t.Fatal(err)
	}
	return runtime, stateStore
}

func TestMCPDataPlaneListsOnlyBusinessToolsAndMutatesBranch(t *testing.T) {
	runtime, stateStore := referenceRuntime(t)
	httpServer := httptest.NewServer(NewDataPlane(runtime))
	t.Cleanup(httpServer.Close)

	ctx := context.Background()
	client := mcp.NewClient(&mcp.Implementation{Name: "statetwin-test", Version: "v0.1.0"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: httpServer.URL + "/mcp/main"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { session.Close() })

	listed, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{
		"add_comment": true, "close_issue": true, "create_issue": true,
		"get_issue": true, "get_repository": true, "list_issues": true,
	}
	if len(listed.Tools) != len(want) {
		t.Fatalf("listed %d tools, want %d", len(listed.Tools), len(want))
	}
	for _, tool := range listed.Tools {
		if !want[tool.Name] {
			t.Fatalf("unexpected agent-visible tool %q", tool.Name)
		}
		for _, forbidden := range []string{"snapshot", "fork", "reset", "diff", "fault", "state"} {
			if tool.Name == forbidden {
				t.Fatalf("control tool %q leaked into data plane", tool.Name)
			}
		}
	}

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "close_issue",
		Arguments: map[string]any{"owner": "octo", "repository": "demo", "number": 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("close_issue failed: %#v", result.Content)
	}
	branch, err := stateStore.Branch(ctx, "main")
	if err != nil {
		t.Fatal(err)
	}
	if got := branch.State.Entities["issue"]["octo/demo#1"]["state"]; got != "closed" {
		t.Fatalf("issue state = %v, want closed", got)
	}
}

func TestControlPlaneRequiresIndependentBearerToken(t *testing.T) {
	_, stateStore := referenceRuntime(t)
	httpServer := httptest.NewServer(NewControlPlane(stateStore, "test-secret"))
	t.Cleanup(httpServer.Close)

	request, _ := http.NewRequest(http.MethodGet, httpServer.URL+"/v1/branches/main", nil)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthorized status = %d", response.StatusCode)
	}

	request, _ = http.NewRequest(http.MethodGet, httpServer.URL+"/v1/branches/main", nil)
	request.Header.Set("Authorization", "test-secret")
	response, err = http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("token without Bearer scheme status = %d", response.StatusCode)
	}

	request, _ = http.NewRequest(http.MethodGet, httpServer.URL+"/v1/branches/main", nil)
	request.Header.Set("Authorization", "Bearer test-secret")
	response, err = http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("authorized status = %d", response.StatusCode)
	}
}

func TestControlPlaneRejectsOversizeBody(t *testing.T) {
	_, stateStore := referenceRuntime(t)
	httpServer := httptest.NewServer(NewControlPlane(stateStore, "test-secret"))
	t.Cleanup(httpServer.Close)
	body := bytes.Repeat([]byte(" "), (1<<20)+1)
	request, _ := http.NewRequest(http.MethodPost, httpServer.URL+"/v1/snapshots", bytes.NewReader(body))
	request.Header.Set("Authorization", "Bearer test-secret")
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("oversize body status = %d, want %d", response.StatusCode, http.StatusBadRequest)
	}
}

func TestControlPlaneAdvancesVirtualClockWithHeadAndAudit(t *testing.T) {
	_, stateStore := referenceRuntime(t)
	httpServer := httptest.NewServer(NewControlPlane(stateStore, "test-secret"))
	t.Cleanup(httpServer.Close)

	request, _ := http.NewRequest(http.MethodPost, httpServer.URL+"/v1/clock/advance", bytes.NewBufferString(`{"branch":"main","by":"1h","expectedHeadVersion":0}`))
	request.Header.Set("Authorization", "Bearer test-secret")
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("advance status = %d", response.StatusCode)
	}
	branch, err := stateStore.Branch(context.Background(), "main")
	if err != nil {
		t.Fatal(err)
	}
	if branch.HeadVersion != 1 || branch.Clock.Hour() != 1 {
		t.Fatalf("branch after advance = head %d clock %s", branch.HeadVersion, branch.Clock)
	}
	entries, err := stateStore.ControlAudit(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Operation != "clock.advance" {
		t.Fatalf("clock audit = %#v", entries)
	}

	request, _ = http.NewRequest(http.MethodPost, httpServer.URL+"/v1/clock/advance", bytes.NewBufferString(`{"branch":"main","by":"1h","expectedHeadVersion":0}`))
	request.Header.Set("Authorization", "Bearer test-secret")
	request.Header.Set("Content-Type", "application/json")
	response, err = http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusConflict {
		t.Fatalf("stale clock advance status = %d, want 409", response.StatusCode)
	}

	request, _ = http.NewRequest(http.MethodPost, httpServer.URL+"/v1/clock/advance", bytes.NewBufferString(`{"branch":"main","by":"-1h"}`))
	request.Header.Set("Authorization", "Bearer test-secret")
	request.Header.Set("Content-Type", "application/json")
	response, err = http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("backward clock status = %d, want 400", response.StatusCode)
	}
}

func TestControlPlaneClockValidationIsBounded(t *testing.T) {
	_, stateStore := referenceRuntime(t)
	httpServer := httptest.NewServer(NewControlPlane(stateStore, "test-secret"))
	t.Cleanup(httpServer.Close)

	tests := []string{
		`{"branch":"main"}`,
		`{"branch":"main","by":"1h","to":"2026-08-02T00:00:00Z"}`,
		`{"branch":"main","by":"3650d"}`,
		`{"branch":"main","to":"not-a-time"}`,
	}
	for _, body := range tests {
		request, err := http.NewRequest(http.MethodPost, httpServer.URL+"/v1/clock/advance", bytes.NewBufferString(body))
		if err != nil {
			t.Fatal(err)
		}
		request.Header.Set("Authorization", "Bearer test-secret")
		request.Header.Set("Content-Type", "application/json")
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != http.StatusBadRequest {
			t.Fatalf("clock body %s status = %d, want 400", body, response.StatusCode)
		}
	}
	branch, err := stateStore.Branch(context.Background(), "main")
	if err != nil {
		t.Fatal(err)
	}
	if branch.HeadVersion != 0 {
		t.Fatalf("invalid clock requests changed head to %d", branch.HeadVersion)
	}
}

func TestPrivateFaultControlCommitsThenReturnsDeterministicFailure(t *testing.T) {
	runtime, stateStore := referenceRuntime(t)
	controlServer := httptest.NewServer(NewControlPlane(stateStore, "test-secret", "close_issue"))
	t.Cleanup(controlServer.Close)
	dataServer := httptest.NewServer(NewDataPlane(runtime))
	t.Cleanup(dataServer.Close)

	unknown := `{"id":"unknown-tool","branch":"main","tool":"reset_world","phase":"before-validation","errorClass":"RATE_LIMITED","message":"must fail","repeatCount":1}`
	request, err := http.NewRequest(http.MethodPost, controlServer.URL+"/v1/faults", bytes.NewBufferString(unknown))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer test-secret")
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("unknown fault tool status = %d, want 400", response.StatusCode)
	}

	body := `{"id":"lose-close-response","branch":"main","tool":"close_issue","phase":"after-commit-before-response","errorClass":"TIMEOUT_AFTER_EFFECT","message":"synthetic response loss","repeatCount":1,"expectedHeadVersion":0}`
	request, err = http.NewRequest(http.MethodPost, controlServer.URL+"/v1/faults", bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer test-secret")
	request.Header.Set("Content-Type", "application/json")
	response, err = http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("fault install status = %d", response.StatusCode)
	}

	ctx := context.Background()
	client := mcp.NewClient(&mcp.Implementation{Name: "statetwin-fault-test", Version: "v0.1.0"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: dataServer.URL + "/mcp/main"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { session.Close() })
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "close_issue", Arguments: map[string]any{"owner": "octo", "repository": "demo", "number": 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("post-commit response-loss fault was not visible to MCP caller")
	}
	branch, err := stateStore.Branch(ctx, "main")
	if err != nil {
		t.Fatal(err)
	}
	if branch.State.Entities["issue"]["octo/demo#1"]["state"] != "closed" {
		t.Fatal("business effect was not committed before synthetic response loss")
	}

	request, _ = http.NewRequest(http.MethodGet, controlServer.URL+"/v1/fault-events?branch=main", nil)
	request.Header.Set("Authorization", "Bearer test-secret")
	response, err = http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var events struct {
		Events []store.FaultEvent `json:"events"`
	}
	if err := json.NewDecoder(response.Body).Decode(&events); err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || len(events.Events) != 1 || events.Events[0].FaultID != "lose-close-response" {
		t.Fatalf("fault events response = status %d body %#v", response.StatusCode, events)
	}
}

func TestMCPBranchComesOnlyFromURLAndForksStayIsolated(t *testing.T) {
	runtime, stateStore := referenceRuntime(t)
	ctx := context.Background()
	if _, err := stateStore.CreateSnapshot(ctx, "base", "main"); err != nil {
		t.Fatal(err)
	}
	if err := stateStore.Fork(ctx, "base", "run-a"); err != nil {
		t.Fatal(err)
	}
	if err := stateStore.Fork(ctx, "base", "run-b"); err != nil {
		t.Fatal(err)
	}
	httpServer := httptest.NewServer(NewDataPlane(runtime))
	t.Cleanup(httpServer.Close)

	client := mcp.NewClient(&mcp.Implementation{Name: "statetwin-test", Version: "v0.1.0"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: httpServer.URL + "/mcp/run-a"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { session.Close() })
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "close_issue",
		Arguments: map[string]any{"owner": "octo", "repository": "demo", "number": 1, "branch": "run-b"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("model-visible branch override must be rejected by input schema")
	}
	result, err = session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "close_issue",
		Arguments: map[string]any{"owner": "octo", "repository": "demo", "number": 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("valid close failed: %#v", result.Content)
	}
	a, err := stateStore.Branch(ctx, "run-a")
	if err != nil {
		t.Fatal(err)
	}
	b, err := stateStore.Branch(ctx, "run-b")
	if err != nil {
		t.Fatal(err)
	}
	if a.State.Entities["issue"]["octo/demo#1"]["state"] != "closed" {
		t.Fatal("run-a did not receive its own transition")
	}
	if b.State.Entities["issue"]["octo/demo#1"]["state"] != "open" {
		t.Fatal("run-a transition leaked into run-b")
	}

	response, err := http.Get(httpServer.URL + "/mcp/invalid%20branch")
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("invalid branch path status = %d", response.StatusCode)
	}
}
