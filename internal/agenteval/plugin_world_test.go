package agenteval

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/augety121/mcp-state-twin/internal/task"
	"github.com/augety121/mcp-state-twin/internal/testfixture"
)

func TestPluginWorldAuthorityAndReplay(t *testing.T) {
	root := testfixture.Baseline(t)
	raw, _ := os.ReadFile(filepath.Join(root, "issue-tracker", "agent-tasks", "close-issue.json"))
	ta, err := task.Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(root, "issue-tracker", ta.Bundle))
	ctx := context.Background()
	w, err := NewPluginWorld(ctx, ta, b)
	if err != nil {
		t.Fatal(err)
	}
	denied, err := w.Call(ctx, Call{"close_issue", map[string]any{"owner": "other", "repository": "demo", "number": 1}})
	if err != nil || denied.Authorized || denied.Dispatched || denied.ErrorClass != "AUTHORITY_DENIED" {
		t.Fatal(denied, err)
	}
	terminal, err := w.Finish("completed", nil)
	if err != nil {
		t.Fatal(err)
	}
	if terminal.Evaluation.PolicyAttempts != 1 || terminal.Evaluation.Outcome == "success" {
		t.Fatal(terminal.Evaluation)
	}
	if err = VerifyPluginTerminal(ctx, terminal, ta, b); err != nil {
		t.Fatal(err)
	}
	if _, err = w.Call(ctx, Call{"close_issue", map[string]any{}}); err == nil {
		t.Fatal("late call")
	}
	copy, err := w.Finish("completed", "invented")
	if err != nil || !same(copy, terminal) {
		t.Fatal("terminal changed", err)
	}
	terminal.View.Events[0].Authorized = true
	if VerifyPluginTerminal(ctx, terminal, ta, b) == nil {
		t.Fatal("tamper accepted")
	}
}

func TestPluginWorldExistingWitnessEquality(t *testing.T) {
	root := testfixture.Baseline(t)
	ctx := context.Background()
	for _, domain := range []string{"issue-tracker", "package-registry"} {
		for _, group := range []string{"core", "extended"} {
			pr, _, err := prepareProject(newProjectSource(ctx, root, "", 256<<20), domain, "project-"+group+".json")
			if err != nil {
				t.Fatal(err)
			}
			seen := map[string]bool{}
			for i, c := range pr.cases.manifest.Cases {
				if c.Role != "positive" || seen[c.TaskID] {
					continue
				}
				seen[c.TaskID] = true
				in := pr.cases.inputs[i]
				raw, err := os.ReadFile(filepath.Join(root, domain, in.task.Bundle))
				if err != nil {
					t.Fatal(err)
				}
				t.Run(c.TaskID, func(t *testing.T) {
					w, err := NewPluginWorld(ctx, in.task, raw)
					if err != nil {
						t.Fatal(err)
					}
					for _, call := range in.witness.Calls {
						if _, err = w.Call(ctx, call); err != nil {
							t.Fatal(err)
						}
					}
					got, err := w.Finish("completed", in.witness.Answer)
					if err != nil {
						t.Fatal(err)
					}
					if got.Evaluation.Outcome != c.Expected.Outcome || got.Cleanup != "complete" {
						t.Fatal(got.Evaluation)
					}
					data, _ := json.Marshal(got)
					decoded, err := DecodePluginTerminal(data)
					if err != nil {
						t.Fatal(err)
					}
					if err = VerifyPluginTerminal(ctx, decoded, in.task, raw); err != nil {
						t.Fatal(err)
					}
				})
			}
		}
	}
}

func TestPluginConcurrentAdmissionBudgetAndFinish(t *testing.T) {
	root := testfixture.Baseline(t)
	raw, _ := os.ReadFile(filepath.Join(root, "issue-tracker/agent-tasks/read-issue.json"))
	ta, e := task.Decode(raw)
	if e != nil {
		t.Fatal(e)
	}
	ta.Budgets.ToolAttempts = 1
	b, _ := os.ReadFile(filepath.Join(root, "issue-tracker", ta.Bundle))
	w, e := NewPluginWorld(context.Background(), ta, b)
	if e != nil {
		t.Fatal(e)
	}
	call := Call{"get_issue", map[string]any{"owner": "octo", "repository": "demo", "number": 1}}
	var workers sync.WaitGroup
	for i := 0; i < 8; i++ {
		workers.Add(1)
		go func() { defer workers.Done(); _, _ = w.Call(context.Background(), call) }()
	}
	workers.Wait()
	terminal, e := w.Finish("completed", nil)
	if e != nil {
		t.Fatal(e)
	}
	if terminal.ToolAttempts != 1 || len(terminal.View.Events) != 1 || terminal.Execution == "completed" {
		t.Fatal(terminal.ToolAttempts, terminal.Execution)
	}
	before, _ := json.Marshal(terminal)
	if _, e = w.Call(context.Background(), call); e == nil {
		t.Fatal("finish barrier")
	}
	again, _ := w.Finish("completed", map[string]any{"score": 1})
	after, _ := json.Marshal(again)
	if string(before) != string(after) {
		t.Fatal("late mutation")
	}
}
