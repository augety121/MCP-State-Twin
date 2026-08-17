package server

import (
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
