package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/augety121/mcp-state-twin/internal/bundle"
)

func TestTaskCLIAdmissionAndWitness(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join("..", "..", "examples", "issue-tracker")
	if err := os.Mkdir(filepath.Join(root, ".statetwin"), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := bundle.Build(filepath.Join(source, "bundle-agent.yaml"), filepath.Join(root, ".statetwin", "agent-world.stb")); err != nil {
		t.Fatal(err)
	}
	for _, p := range []struct{ source, target string }{{"agent-tasks/close-issue.json", "task.json"}, {"agent-witnesses/close-issue.json", "witness.json"}} {
		data, err := os.ReadFile(filepath.Join(source, filepath.FromSlash(p.source)))
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(root, p.target), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, args := range [][]string{{"validate", "--root", root, "--task", "task.json"}, {"witness", "--root", root, "--task", "task.json", "--witness", "witness.json"}} {
		if err := runTask(context.Background(), args); err != nil {
			t.Fatal(err)
		}
	}
	for _, args := range [][]string{{}, {"run"}, {"validate", "--root", root, "--task", "../task.json"}, {"validate", "--root", root, "--task", "task.json", "--witness", "witness.json"}, {"witness", "--root", root, "--task", "task.json"}} {
		if err := runTask(context.Background(), args); err == nil {
			t.Fatalf("accepted bad CLI: %v", args)
		}
	}
}
