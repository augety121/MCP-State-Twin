package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/augety121/mcp-state-twin/internal/bundle"
)

func TestOfflineAgentCLIQuickstartAndRegression(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join("..", "..", "examples", "issue-tracker")
	if err := os.Mkdir(filepath.Join(root, ".statetwin"), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := bundle.Build(filepath.Join(source, "bundle-agent.yaml"), filepath.Join(root, ".statetwin", "agent-world.stb")); err != nil {
		t.Fatal(err)
	}
	files := []string{"agent-tasks/close-issue.json", "agent-runs/baseline.json", "agent-runs/candidate.json", "agent-mocks/close-issue.json", "agent-mocks/omit-action.json", "agent-comparison.json"}
	for _, name := range files {
		data, err := os.ReadFile(filepath.Join(source, filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		target := filepath.Join(root, filepath.FromSlash(name))
		if err = os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(target, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	ctx := context.Background()
	for _, args := range [][]string{
		{"preflight", "--root", root, "--task", "agent-tasks/close-issue.json", "--config", "agent-runs/baseline.json"},
		{"mock", "--root", root, "--task", "agent-tasks/close-issue.json", "--config", "agent-runs/baseline.json", "--responses", "agent-mocks/close-issue.json", "--out", ".statetwin/baseline"},
		{"verify", "--root", root, "--evidence", ".statetwin/baseline/terminal.json"},
	} {
		if err := runAgentEval(ctx, args); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
	}
	// A failed task still publishes complete, independently checkable evidence.
	if err := runAgentEval(ctx, []string{"mock", "--root", root, "--task", "agent-tasks/close-issue.json", "--config", "agent-runs/candidate.json", "--responses", "agent-mocks/omit-action.json", "--out", ".statetwin/candidate"}); err == nil {
		t.Fatal("task failure exit code")
	}
	if err := runAgentEval(ctx, []string{"verify", "--root", root, "--evidence", ".statetwin/candidate/terminal.json"}); err != nil {
		t.Fatal(err)
	}
	for _, format := range []string{"json", "markdown"} {
		if err := runAgentEval(ctx, []string{"compare", "--root", root, "--plan", "agent-comparison.json", "--format", format}); err == nil {
			t.Fatal("regression exit code")
		}
	}
	for _, args := range [][]string{{}, {"live"}, {"mock", "--endpoint", "https://example.invalid"}, {"verify", "--root", root, "--evidence", "../terminal.json"}, {"compare", "--root", root, "--plan", "agent-comparison.json", "--format", "html"}} {
		if err := runAgentEval(ctx, args); err == nil {
			t.Fatal("unsafe CLI accepted")
		}
	}
}
