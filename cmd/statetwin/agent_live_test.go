package main

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/augety121/mcp-state-twin/internal/agentapi"
	"github.com/augety121/mcp-state-twin/internal/bundle"
)

// Tests use a file rather than a pipe so bounded but large Task JSON cannot
// block the writer waiting for a concurrent stdout consumer.
func captureLivePlan(t *testing.T, args []string) ([]byte, error) {
	t.Helper()
	out, err := os.CreateTemp(t.TempDir(), "stdout-")
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	old := os.Stdout
	os.Stdout = out
	defer func() { os.Stdout = old }()
	err = runAgentEval(context.Background(), args)
	if _, seekErr := out.Seek(0, 0); seekErr != nil {
		t.Fatal(seekErr)
	}
	data, readErr := io.ReadAll(out)
	if readErr != nil {
		t.Fatal(readErr)
	}
	return data, err
}

func TestLiveCLIPlanPreflightAndApprovalRefusal(t *testing.T) {
	root := t.TempDir()
	source := "../../examples/issue-tracker"
	if err := os.MkdirAll(filepath.Join(root, ".statetwin", "live"), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := bundle.Build(filepath.Join(source, "bundle-agent.yaml"), filepath.Join(root, ".statetwin", "agent-world.stb")); err != nil {
		t.Fatal(err)
	}
	ta, err := os.ReadFile(filepath.Join(source, "agent-tasks", "close-issue.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(root, "task.json"), ta, 0600); err != nil {
		t.Fatal(err)
	}
	args := []string{"live-plan", "--root", root, "--task", "task.json", "--id", "operator-trial", "--model", "test-model-not-a-product", "--max-requests", "2", "--max-output-tokens", "512", "--valid-for", "1h"}
	data, err := captureLivePlan(t, args)
	if err != nil {
		t.Fatal(err)
	}
	p, err := agentapi.DecodePlan(data)
	if err != nil {
		t.Fatal(err)
	}
	if p.Approved || p.SyntheticDataApproved || p.UnknownCostApproved {
		t.Fatal("generator inferred approval")
	}
	planFile := filepath.Join(root, "plan.json")
	if err = os.WriteFile(planFile, data, 0600); err != nil {
		t.Fatal(err)
	}
	if err = runAgentEval(context.Background(), []string{"live-preflight", "--root", root, "--plan", "plan.json"}); err != nil {
		t.Fatal(err)
	}
	for _, extra := range [][]string{nil, {"--allow-live"}} {
		args := append([]string{"live", "--root", root, "--plan", "plan.json"}, extra...)
		if err = runAgentEval(context.Background(), args); err == nil || err.Error() != "LIVE_NOT_AUTHORIZED" {
			t.Fatal("unapproved live admitted", err)
		}
	}
	if _, err = os.Stat(filepath.Join(root, filepath.FromSlash(p.OutputDirectory()))); !os.IsNotExist(err) {
		t.Fatal("unapproved live claimed directory")
	}
	p.Approved = true
	p.SyntheticDataApproved = true
	p.UnknownCostApproved = true
	data, _ = json.Marshal(p)
	if err = os.WriteFile(planFile, data, 0600); err != nil {
		t.Fatal(err)
	}
	if err = runAgentEval(context.Background(), []string{"live", "--root", root, "--plan", "plan.json"}); err == nil {
		t.Fatal("approved plan bypassed CLI flag")
	}
	t.Setenv("OPENAI_API_KEY", "") // explicitly absent; no actual account/key/network
	if err = runAgentEval(context.Background(), []string{"live", "--root", root, "--plan", "plan.json", "--allow-live"}); err == nil || err.Error() != "PROVIDER_CREDENTIAL_MISSING_OR_INVALID" {
		t.Fatal(err)
	}
	claim, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(p.OutputDirectory()), "claim.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(claim), "OPENAI_API_KEY") {
		t.Fatal("credential reference retained")
	}
	for _, args := range [][]string{{"live-plan", "--root", root, "--task", "task.json"}, {"live", "--endpoint", "https://example.invalid"}, {"live-verify", "--root", root, "--evidence", "../outside.json"}} {
		if err = runAgentEval(context.Background(), args); err == nil {
			t.Fatal("unsafe CLI arguments admitted")
		}
	}
}
