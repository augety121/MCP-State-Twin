package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/augety121/mcp-state-twin/internal/bundle"
	"github.com/augety121/mcp-state-twin/internal/task"
)

func prepareOfflineCLI(t *testing.T) string {
	t.Helper()
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
	return root
}

func TestOfflineAgentCLIQuickstartAndRegression(t *testing.T) {
	root := prepareOfflineCLI(t)
	ctx := context.Background()
	for _, args := range [][]string{
		{"preflight", "--root", root, "--task", "agent-tasks/close-issue.json", "--config", "agent-runs/baseline.json"},
		{"mock", "--root", root, "--task", "agent-tasks/close-issue.json", "--config", "agent-runs/baseline.json", "--responses", "agent-mocks/close-issue.json", "--out", ".statetwin/baseline"},
		{"verify", "--root", root, "--evidence", ".statetwin/baseline/terminal.json"},
		{"inspect", "--root", root, "--out", ".statetwin/baseline"},
		{"inspect", "--root", root, "--out", ".statetwin/not-created"},
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
	if err := os.WriteFile(filepath.Join(root, ".statetwin", "baseline", "closure.json"), []byte(`{"broken":`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := runAgentEval(ctx, []string{"inspect", "--root", root, "--out", ".statetwin/baseline"}); err == nil || err.Error() != "EVIDENCE_DIRECTORY_INVALID" {
		t.Fatal("invalid inspection must fail CLI admission", err)
	}
	for _, args := range [][]string{{}, {"live"}, {"mock", "--endpoint", "https://example.invalid"}, {"verify", "--root", root, "--evidence", "../terminal.json"}, {"compare", "--root", root, "--plan", "agent-comparison.json", "--format", "html"}} {
		if err := runAgentEval(ctx, args); err == nil {
			t.Fatal("unsafe CLI accepted")
		}
	}
}

func TestOfflineComparisonCLIFailsClosedOnGradingAndPolicy(t *testing.T) {
	for _, mode := range []string{"unscorable", "new-policy-failure"} {
		t.Run(mode, func(t *testing.T) {
			root := prepareOfflineCLI(t)
			name := filepath.Join(root, "agent-tasks", "close-issue.json")
			raw, err := os.ReadFile(name)
			if err != nil {
				t.Fatal(err)
			}
			ta, err := task.Decode(raw)
			if err != nil {
				t.Fatal(err)
			}
			ta.Oracle[0].Expr = "false"
			ta.Oracle[1].Expr = "answer.ok == true"
			if mode == "unscorable" {
				ta.Oracle[0].Expr = "answer.missing == true"
			}
			raw, err = json.Marshal(ta)
			if err != nil {
				t.Fatal(err)
			}
			if err = os.WriteFile(name, raw, 0600); err != nil {
				t.Fatal(err)
			}
			for _, which := range []string{"baseline", "candidate"} {
				answer := `{"ok":true}`
				if which == "candidate" {
					answer = `{"ok":false}`
				}
				script := map[string]any{"kind": "MockResponses", "syntheticOnly": true, "responses": []any{map[string]any{"status": "completed", "output": []any{map[string]any{"type": "message", "role": "assistant", "content": []any{map[string]any{"type": "output_text", "text": answer}}}}}}}
				raw, err = json.Marshal(script)
				if err != nil {
					t.Fatal(err)
				}
				if err = os.WriteFile(filepath.Join(root, "case.json"), raw, 0600); err != nil {
					t.Fatal(err)
				}
				if err = runAgentEval(context.Background(), []string{"mock", "--root", root, "--task", "agent-tasks/close-issue.json", "--config", "agent-runs/" + which + ".json", "--responses", "case.json", "--out", ".statetwin/" + which}); err == nil {
					t.Fatal("fixture must not pass task grading")
				}
				if err = runAgentEval(context.Background(), []string{"verify", "--root", root, "--evidence", ".statetwin/" + which + "/terminal.json"}); err != nil {
					t.Fatal("fixture must remain replay-valid", err)
				}
			}
			for _, format := range []string{"json", "markdown"} {
				if err = runAgentEval(context.Background(), []string{"compare", "--root", root, "--plan", "agent-comparison.json", "--format", format}); err == nil {
					t.Fatal("CLI gate accepted an unscorable/unsafe pair")
				}
			}
		})
	}
}
