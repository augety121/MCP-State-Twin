package agenteval

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/augety121/mcp-state-twin/internal/bundle"
	"github.com/augety121/mcp-state-twin/internal/task"
)

func kit(t *testing.T) (*bundle.Artifact, func(string) (*task.Task, *Witness)) {
	t.Helper()
	root := filepath.Join("..", "..", "examples", "issue-tracker")
	out := filepath.Join(t.TempDir(), "world.stb")
	if _, err := bundle.Build(filepath.Join(root, "bundle-agent.yaml"), out); err != nil {
		t.Fatal(err)
	}
	b, err := bundle.Open(out)
	if err != nil {
		t.Fatal(err)
	}
	return b, func(id string) (*task.Task, *Witness) {
		data, err := os.ReadFile(filepath.Join(root, "agent-tasks", id+".json"))
		if err != nil {
			t.Fatal(err)
		}
		taskSpec, err := task.Decode(data)
		if err != nil {
			t.Fatal(err)
		}
		data, err = os.ReadFile(filepath.Join(root, "agent-witnesses", id+".json"))
		if err != nil {
			t.Fatal(err)
		}
		w, err := DecodeWitness(data)
		if err != nil {
			t.Fatal(err)
		}
		return taskSpec, w
	}
}

func TestSixWitnessesThroughOfflineMCP(t *testing.T) {
	b, load := kit(t)
	for _, id := range []string{"read-issue", "create-issue", "close-issue", "already-closed", "scope-protection", "after-commit-confirm"} {
		t.Run(id, func(t *testing.T) {
			spec, w := load(id)
			r, err := RunWitness(context.Background(), spec, b, w)
			if err != nil {
				t.Fatal(err)
			}
			if r.Source != "scripted-witness" || r.Evaluation.Outcome != spec.ExpectedOutcome || r.CleanupStatus != "complete" {
				t.Fatalf("%+v %+v", r, r.Evaluation)
			}
			if id == "after-commit-confirm" {
				e := r.View.Events[0]
				if !e.EffectCommitted || e.ErrorClass != "TIMEOUT_AFTER_EFFECT" {
					t.Fatalf("effect ambiguity lost: %+v", e)
				}
			}
		})
	}
}

func TestWitnessScoringAndAdmissionFailures(t *testing.T) {
	b, load := kit(t)
	for _, tc := range []struct {
		name, want string
		change     func(*task.Task, *Witness)
	}{
		{"omit", "task_failed", func(s *task.Task, w *Witness) { w.Calls = nil }},
		{"wrong target", "policy_violation", func(s *task.Task, w *Witness) { w.Calls[0].Input["number"] = float64(2) }},
		{"extra write", "policy_violation", func(s *task.Task, w *Witness) {
			w.Calls = append(w.Calls, Call{Tool: "add_comment", Input: map[string]any{"owner": "octo", "repository": "demo", "number": 1, "body": "extra"}})
		}},
		{"extra legal read", "success", func(s *task.Task, w *Witness) {
			w.Calls = append(w.Calls, Call{Tool: "get_issue", Input: map[string]any{"owner": "octo", "repository": "demo", "number": 1}})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, w := load("close-issue")
			tc.change(s, w)
			r, err := RunWitness(context.Background(), s, b, w)
			if err != nil || r.Evaluation.Outcome != tc.want {
				t.Fatalf("%+v %v", r, err)
			}
		})
	}
	s, w := load("close-issue")
	s.Tools = append(s.Tools, "get_comments")
	if _, err := RunWitness(context.Background(), s, b, w); err == nil {
		t.Fatal("missing affordance accepted")
	}
	s, w = load("close-issue")
	w.Answer = "api_key=synthetic-secret-sentinel"
	if _, err := RunWitness(context.Background(), s, b, w); err == nil {
		t.Fatal("secret admitted")
	}
	s, w = load("close-issue")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := RunWitness(ctx, s, b, w); err == nil {
		t.Fatal("cancel ignored")
	}
}

func TestFaultRequiresDeliveredConfirmation(t *testing.T) {
	b, load := kit(t)
	s, w := load("after-commit-confirm")
	w.Calls = w.Calls[:1]
	r, err := RunWitness(context.Background(), s, b, w)
	if err != nil || r.Evaluation.Outcome != "task_failed" {
		t.Fatalf("%+v %v", r, err)
	}
	// A client-produced result is not trusted without the bounded MCP execution.
	bad := []byte(`{"kind":"TaskWitness","taskId":"close-issue","syntheticOnly":true,"calls":[],"answer":null,"passed":true}`)
	if _, err := DecodeWitness(bad); err == nil {
		t.Fatal("self-graded witness admitted")
	}
	s, w = load("close-issue")
	original, _ := json.Marshal(s)
	_, _ = RunWitness(context.Background(), s, b, w)
	after, _ := json.Marshal(s)
	if string(original) != string(after) {
		t.Fatal("run mutated task")
	}
}
