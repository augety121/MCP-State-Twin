package agenteval

import (
	"context"
	"errors"
	"sort"

	"github.com/augety121/mcp-state-twin/internal/bundle"
	"github.com/augety121/mcp-state-twin/internal/limits"
	"github.com/augety121/mcp-state-twin/internal/logging"
	"github.com/augety121/mcp-state-twin/internal/task"
)

type CaseExpected struct {
	Outcome      string   `json:"outcome"`
	FailedChecks []string `json:"failedChecks"`
}
type TaskCase struct {
	CaseID   string       `json:"caseId"`
	TaskID   string       `json:"taskId"`
	Role     string       `json:"role"`
	Witness  string       `json:"witness"`
	Expected CaseExpected `json:"expected"`
}
type CaseManifest struct {
	Format  string          `json:"format"`
	Profile string          `json:"profile"`
	Tasks   []taskReference `json:"tasks"`
	Cases   []TaskCase      `json:"cases"`
}
type caseInput struct {
	task    *task.Task
	bundle  *bundle.Artifact
	witness *Witness
}
type preparedCases struct {
	manifest CaseManifest
	inputs   []caseInput
	tasks    map[string]*task.Task
}

func prepareCases(ctx context.Context, root, name string) (*preparedCases, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	raw, err := task.ReadFile(root, name, 256<<10)
	if err != nil {
		return nil, errors.New("CASE_MANIFEST_INVALID")
	}
	var m CaseManifest
	if decodeSuiteMetadata(raw, 256<<10, &m) != nil || m.Format != "statetwin.dev/task-cases/v1alpha1" || m.Profile != "synthetic-witness-cases-v1" || len(m.Tasks) < 1 || len(m.Tasks) > 16 || len(m.Cases) < 1 || len(m.Cases) > 64 {
		return nil, errors.New("CASE_MANIFEST_INVALID")
	}
	p := &preparedCases{manifest: m, tasks: map[string]*task.Task{}}
	bundles := map[string]*bundle.Artifact{}
	cache := map[string][]byte{}
	remaining, extracted := 64<<20, 64<<20
	read := func(name string, limit int) ([]byte, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if b, ok := cache[name]; ok {
			return b, nil
		}
		b, err := task.ReadFile(root, name, limit)
		if err != nil {
			return nil, errors.New("CASE_INPUT_INVALID")
		}
		if len(b) > remaining {
			return nil, errors.New("CASE_RESOURCE_LIMIT")
		}
		remaining -= len(b)
		cache[name] = b
		return b, nil
	}
	for _, ref := range m.Tasks {
		if !validLabel(ref.TaskID) || p.tasks[ref.TaskID] != nil {
			return nil, errors.New("CASE_MANIFEST_INVALID")
		}
		raw, err := read(ref.Task, task.MaxBytes)
		if err != nil {
			return nil, err
		}
		t, err := task.Decode(raw)
		if err != nil || t.ID != ref.TaskID {
			return nil, errors.New("CASE_INPUT_INVALID")
		}
		b := bundles[t.Bundle]
		if b == nil {
			raw, err := read(t.Bundle, limits.MaxBundleCompressed)
			if err != nil {
				return nil, err
			}
			b, err = bundle.OpenBytes(raw)
			if err != nil {
				return nil, errors.New("CASE_INPUT_INVALID")
			}
			names := make([]string, 0, len(b.Files))
			for n := range b.Files {
				names = append(names, n)
			}
			sort.Strings(names)
			for _, n := range names {
				v := b.Files[n]
				if len(v) > extracted {
					return nil, errors.New("CASE_RESOURCE_LIMIT")
				}
				extracted -= len(v)
				if logging.ContainsSensitive(string(v)) {
					return nil, errors.New("DATA_POLICY_REJECTED")
				}
			}
			bundles[t.Bundle] = b
		}
		if Admit(t, b) != nil {
			return nil, errors.New("CASE_INPUT_INVALID")
		}
		p.tasks[t.ID] = t
	}
	ids, used := map[string]bool{}, map[string]bool{}
	for _, c := range m.Cases {
		t := p.tasks[c.TaskID]
		if !validLabel(c.CaseID) || ids[c.CaseID] || t == nil || len(c.Expected.FailedChecks) > 64 {
			return nil, errors.New("CASE_MANIFEST_INVALID")
		}
		ids[c.CaseID] = true
		used[c.TaskID] = true
		assertions := map[string]string{}
		for _, a := range t.Oracle {
			assertions[a.ID] = a.Category
		}
		failed := map[string]bool{}
		hasGoal := false
		for _, id := range c.Expected.FailedChecks {
			if assertions[id] == "" || failed[id] {
				return nil, errors.New("CASE_MANIFEST_INVALID")
			}
			failed[id] = true
			hasGoal = hasGoal || assertions[id] == "goal"
		}
		valid := false
		switch c.Role {
		case "positive":
			valid = c.Expected.Outcome == t.ExpectedOutcome && len(failed) == 0
		case "goal-negative":
			valid = c.Expected.Outcome == "task_failed" && hasGoal
		case "policy-negative":
			valid = c.Expected.Outcome == "policy_violation"
		case "unscorable-negative":
			valid = c.Expected.Outcome == "evaluator_error" && len(failed) > 0
		}
		if !valid {
			return nil, errors.New("CASE_MANIFEST_INVALID")
		}
		raw, err := read(c.Witness, task.MaxBytes)
		if err != nil {
			return nil, err
		}
		w, err := DecodeWitness(raw)
		if err != nil || w.TaskID != t.ID || len(w.Calls) > t.Budgets.ToolAttempts {
			return nil, errors.New("CASE_INPUT_INVALID")
		}
		p.inputs = append(p.inputs, caseInput{t, bundles[t.Bundle], w})
	}
	if len(used) != len(p.tasks) {
		return nil, errors.New("CASE_MANIFEST_INVALID")
	}
	return p, ctx.Err()
}
