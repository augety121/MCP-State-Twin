package agenteval

import (
	"context"
	"errors"
	"sort"

	"github.com/augety121/mcp-state-twin/internal/bundle"
	"github.com/augety121/mcp-state-twin/internal/limits"
	"github.com/augety121/mcp-state-twin/internal/logging"
	"github.com/augety121/mcp-state-twin/internal/task"
	"github.com/augety121/mcp-state-twin/internal/world"
)

type CaseExpected struct {
	Outcome      string   `json:"outcome"`
	FailedChecks []string `json:"failedChecks"`
}
type TaskCase struct {
	CaseID    string         `json:"caseId"`
	TaskID    string         `json:"taskId"`
	Role      string         `json:"role"`
	Witness   string         `json:"witness"`
	Expected  CaseExpected   `json:"expected"`
	Mutations []ViewMutation `json:"mutations,omitempty"`
}

// ViewMutation only replaces an existing string field in a private grading
// view. It cannot dispatch tools, modify a world, or introduce executable code.
type ViewMutation struct {
	Entity string `json:"entity"`
	Key    string `json:"key"`
	Field  string `json:"field"`
	Value  string `json:"value"`
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
	return prepareCasesWith(ctx, name, func(n string, limit int) ([]byte, error) { return task.ReadFile(root, n, limit) }, bundle.OpenBytes)
}

func prepareCasesWith(ctx context.Context, name string, readFile inputReader, openBundle bundleReader) (*preparedCases, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	raw, err := readFile(name, 256<<10)
	if err != nil {
		return nil, errors.New("CASE_MANIFEST_INVALID")
	}
	var m CaseManifest
	if decodeSuiteMetadata(raw, 256<<10, &m) != nil || !((m.Format == "statetwin.dev/task-cases/v1alpha1" && m.Profile == "synthetic-witness-cases-v1") || (m.Format == "statetwin.dev/task-cases/v1alpha2" && m.Profile == "synthetic-oracle-mutation-cases-v1")) || len(m.Tasks) < 1 || len(m.Tasks) > 16 || len(m.Cases) < 1 || len(m.Cases) > 64 {
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
		b, err := readFile(name, limit)
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
			b, err = openBundle(raw)
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
			valid = (c.Expected.Outcome == "evaluator_error" || (m.Profile == "synthetic-oracle-mutation-cases-v1" && c.Expected.Outcome == "not_evaluated")) && len(failed) > 0
		}
		if !valid {
			return nil, errors.New("CASE_MANIFEST_INVALID")
		}
		if len(c.Mutations) > 0 {
			if m.Profile != "synthetic-oracle-mutation-cases-v1" || (c.Role != "policy-negative" && c.Role != "goal-negative") || len(c.Mutations) > 4 {
				return nil, errors.New("CASE_MANIFEST_INVALID")
			}
			b := bundles[t.Bundle]
			state, err := world.DecodeStrict(b.Files[b.Manifest.Fixture])
			if err != nil {
				return nil, errors.New("CASE_INPUT_INVALID")
			}
			seen := map[string]bool{}
			for _, v := range c.Mutations {
				key := v.Entity + "\x00" + v.Key + "\x00" + v.Field
				previous, ok := state.Entities[v.Entity][v.Key][v.Field].(string)
				// A witness may create the target record. Its entity and string
				// field must already be represented in the admitted fixture;
				// the exact target must exist after execution or the case fails.
				knownField := ok
				if state.Entities[v.Entity][v.Key] == nil {
					for _, record := range state.Entities[v.Entity] {
						if _, stringField := record[v.Field].(string); stringField {
							knownField = true
						}
					}
				}
				if !knownField || len(v.Entity) == 0 || len(v.Key) == 0 || len(v.Field) == 0 || len(v.Entity) > 128 || len(v.Key) > 128 || len(v.Field) > 128 || len(v.Value) > 1024 || (ok && v.Value == previous) || seen[key] {
					return nil, errors.New("CASE_INPUT_INVALID")
				}
				seen[key] = true
			}
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
