package agenteval

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/augety121/mcp-state-twin/internal/agenthost"
	"gopkg.in/yaml.v3"
)

func suiteFixture(t *testing.T, ids ...string) (string, *SuitePlan) {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "world.stb"), rawBundle(t), 0600); err != nil {
		t.Fatal(err)
	}
	_, load := kit(t)
	p := &SuitePlan{Format: SuiteFormat, BaselineModel: "mock-baseline", CandidateModel: "mock-candidate", MaxOutputTokens: 1024}
	for i, id := range ids {
		ta, w := load(id)
		ta.Bundle = "world.stb"
		writeTestJSON(t, root, "task-"+id+".json", ta)
		writeTestJSON(t, root, "mock-"+id+".json", mockWitness(w))
		p.Pairs = append(p.Pairs, SuitePair{TaskID: id, Task: "task-" + id + ".json", Repeat: i + 1, BaselineResponses: "mock-" + id + ".json", CandidateResponses: "mock-" + id + ".json"})
	}
	return root, p
}

func suiteBytes(t *testing.T, p *SuitePlan) []byte {
	t.Helper()
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestSuiteSixTasksAndFrozenInputs(t *testing.T) {
	root, p := suiteFixture(t, "read-issue", "close-issue", "create-issue", "already-closed", "scope-protection", "after-commit-confirm")
	prepared, err := PrepareSuite(context.Background(), root, suiteBytes(t, p))
	if err != nil {
		t.Fatal(err)
	}
	summary := prepared.Summary()
	if summary.PlannedTrials != 12 || summary.Plan.Validate() != nil {
		t.Fatal(summary)
	}
	summary.Plan.Pairs[0].Baseline.TrialID = "tampered"
	if prepared.Summary().Plan.Pairs[0].Baseline.TrialID != "baseline-01" {
		t.Fatal("summary mutated frozen plan")
	}
	// Replace source content after preparation; execution must use frozen bytes.
	if err := os.WriteFile(filepath.Join(root, "world.stb"), []byte("invalid after preflight"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, pair := range p.Pairs {
		for _, name := range []string{pair.Task, pair.BaselineResponses} {
			if err := os.WriteFile(filepath.Join(root, name), []byte("invalid after preflight"), 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
	r, err := RunSuite(context.Background(), root, "suite", prepared)
	if err != nil {
		t.Fatal(err)
	}
	if r.ExecutionStatus != "completed" || r.ComparisonStatus != "complete" || r.Comparison.Decision != "no_regression_observed" || r.UpgradeAllowed || len(r.Trials) != 12 || r.Comparison.Counts.ValidlyEvaluated != 12 || len(r.Comparison.TaskSummaries) != 6 {
		t.Fatalf("%+v", r)
	}
	for _, trial := range r.Trials {
		if trial.State != "completed" || trial.ExecutionStatus != "completed" || trial.EvidenceStatus != "complete" || trial.CleanupStatus != "complete" {
			t.Fatal(trial)
		}
	}
	planRaw, err := os.ReadFile(filepath.Join(root, "suite", "plan.json"))
	if err != nil {
		t.Fatal(err)
	}
	plan, err := DecodeCompare(planRaw)
	if err != nil {
		t.Fatal(err)
	}
	again, err := Compare(context.Background(), filepath.Join(root, "suite"), plan)
	if err != nil || !reflect.DeepEqual(again, r.Comparison) {
		t.Fatal("saved plan did not reproduce comparison", err)
	}
	reportRaw, err := os.ReadFile(filepath.Join(root, "suite", "report.json"))
	if err != nil {
		t.Fatal(err)
	}
	var saved SuiteReport
	if err := json.Unmarshal(reportRaw, &saved); err != nil || !reflect.DeepEqual(&saved, r) {
		t.Fatal("saved report mismatch", err)
	}
	if _, err := os.Stat(filepath.Join(root, "suite", "report.pending.json")); !os.IsNotExist(err) {
		t.Fatal("owned staging left behind")
	}
	if second, err := RunSuite(context.Background(), root, "suite", prepared); err == nil || second != nil {
		t.Fatal("suite overwritten")
	}
	after, _ := os.ReadFile(filepath.Join(root, "suite", "report.json"))
	if string(after) != string(reportRaw) {
		t.Fatal("existing report changed")
	}
	for _, forbidden := range []string{"task-read-issue.json", "mock-read-issue.json", "objective", "world.stb"} {
		if strings.Contains(string(planRaw), forbidden) || strings.Contains(string(reportRaw), forbidden) {
			t.Fatal("private suite input exposed", forbidden)
		}
	}
}

func TestSuiteFailureContinuationAndPartialStop(t *testing.T) {
	for _, partial := range []bool{false, true} {
		t.Run(map[bool]string{false: "goal failure continues", true: "host failure stops"}[partial], func(t *testing.T) {
			root, p := suiteFixture(t, "close-issue", "read-issue")
			m := &agenthost.MockScript{Kind: "MockResponses", SyntheticOnly: true, Responses: []json.RawMessage{mockResponse(mockFinal(map[string]any{}))}}
			if partial {
				m.Responses = []json.RawMessage{mockResponse(map[string]any{"type": "unsupported-item"})}
			}
			writeTestJSON(t, root, "bad.json", m)
			p.Pairs[0].CandidateResponses = "bad.json"
			prepared, err := PrepareSuite(context.Background(), root, suiteBytes(t, p))
			if err != nil {
				t.Fatal(err)
			}
			r, err := RunSuite(context.Background(), root, "suite", prepared)
			if !partial {
				if err != nil || r.ExecutionStatus != "completed" || r.Comparison.Decision != "regression" || r.Trials[3].State != "completed" || r.Trials[1].Outcome != "task_failed" {
					t.Fatal(r, err)
				}
			} else {
				if err == nil || r == nil || r.ExecutionStatus != "stopped" || r.Trials[1].State != "failed" || r.Trials[2].State != "not_started" || r.Comparison.Decision != "inconclusive" || r.Comparison.Counts.Planned != 4 || r.Comparison.Counts.NotStarted != 2 {
					t.Fatal(r, err)
				}
				if _, err := os.Stat(filepath.Join(root, "suite", "baseline-02")); !os.IsNotExist(err) {
					t.Fatal("trial started after stop")
				}
			}
		})
	}
}

func TestSuitePreflightAdmissionAndNoWrites(t *testing.T) {
	root, p := suiteFixture(t, "close-issue", "read-issue")
	raw := suiteBytes(t, p)
	before, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*SuitePlan){
		func(p *SuitePlan) { p.Pairs[1].Task = "missing.json" },
		func(p *SuitePlan) { p.Pairs[1].CandidateResponses = "missing.json" },
		func(p *SuitePlan) { p.Pairs[1].TaskID = "different-task" },
		func(p *SuitePlan) { p.Pairs[0].Task = "../outside.json" },
		func(p *SuitePlan) { p.Pairs = append(p.Pairs, p.Pairs[0]) },
		func(p *SuitePlan) { p.BaselineModel = "real-provider" },
		func(p *SuitePlan) { p.CandidateModel = p.BaselineModel },
		func(p *SuitePlan) { p.MaxOutputTokens = 8193 },
		func(p *SuitePlan) { p.MaxOutputTokens = 0 },
		func(p *SuitePlan) { p.Pairs[0].Repeat = 17 },
		func(p *SuitePlan) { p.Pairs = nil },
		func(p *SuitePlan) {
			for len(p.Pairs) <= maxSuitePairs {
				p.Pairs = append(p.Pairs, p.Pairs[0])
			}
		},
	} {
		var bad SuitePlan
		if err := json.Unmarshal(raw, &bad); err != nil {
			t.Fatal(err)
		}
		mutate(&bad)
		if got, err := PrepareSuite(context.Background(), root, suiteBytes(t, &bad)); err == nil || got != nil || strings.Contains(err.Error(), root) {
			t.Fatal("invalid suite admitted or path exposed", err)
		}
	}
	for _, field := range []string{"repeat", "maxOutputTokens"} {
		for _, value := range []any{nil, "1", false} {
			var object map[string]any
			if err := json.Unmarshal(raw, &object); err != nil {
				t.Fatal(err)
			}
			target := object
			if field == "repeat" {
				target = object["pairs"].([]any)[0].(map[string]any)
			}
			target[field] = value
			for _, missing := range []bool{false, true} {
				if missing {
					delete(target, field)
				}
				bad, err := json.Marshal(object)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := DecodeSuite(bad); err == nil {
					t.Fatal("noninteger/missing number admitted", field, value, missing)
				}
			}
		}
	}
	if _, err := DecodeSuite(append([]byte("unknown: true\n"), []byte("format: invalid\n")...)); err == nil {
		t.Fatal("unknown field admitted")
	}
	for _, marshal := range []func(any) ([]byte, error){json.Marshal, yaml.Marshal} {
		data, err := marshal(p)
		if err != nil {
			t.Fatal(err)
		}
		for _, field := range []string{"repeat", "maxOutputTokens"} {
			needle := field + ": 1"
			replacement := field + ": 1.5"
			if field == "maxOutputTokens" {
				needle = field + ": 1024"
				replacement = field + ": 1024.5"
			}
			if json.Valid(data) {
				needle = `"` + field + `":1`
				replacement = `"` + field + `":1.5`
				if field == "maxOutputTokens" {
					needle = `"` + field + `":1024`
					replacement = `"` + field + `":1024.5`
				}
			}
			if got, err := DecodeSuite([]byte(strings.Replace(string(data), needle, replacement, 1))); err == nil || got != nil {
				t.Fatal("float suite number admitted", field)
			}
		}
	}
	for _, limits := range [][2]int{{1, maxSuiteInputBytes}, {maxSuiteInputBytes, 1}} {
		if got, err := prepareSuite(context.Background(), root, raw, limits[0], limits[1]); err == nil || got != nil || err.Error() != "SUITE_RESOURCE_LIMIT" {
			t.Fatal("input budget not enforced", err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if got, err := PrepareSuite(ctx, root, raw); got != nil || !errors.Is(err, context.Canceled) {
		t.Fatal("cancellation lost", err)
	}
	after, err := os.ReadDir(root)
	if err != nil || len(before) != len(after) {
		t.Fatal("preflight wrote files")
	}
	for i := range before {
		if before[i].Name() != after[i].Name() {
			t.Fatal("preflight changed directory")
		}
	}
}

func TestSuiteSensitiveScriptsFailBeforeExecution(t *testing.T) {
	root, p := suiteFixture(t, "close-issue")
	sentinel := "sk-" + strings.Repeat("x", 30)
	m := &agenthost.MockScript{Kind: "MockResponses", SyntheticOnly: true, Responses: []json.RawMessage{mockResponse(mockFinal(sentinel))}}
	writeTestJSON(t, root, "secret.json", m)
	p.Pairs[0].CandidateResponses = "secret.json"
	if got, err := PrepareSuite(context.Background(), root, suiteBytes(t, p)); got != nil || err == nil || strings.Contains(err.Error(), sentinel) {
		t.Fatal("secret admitted or echoed", err)
	}
}

func TestSuiteCancellationAndStorageFailures(t *testing.T) {
	for _, tc := range []struct {
		name, op, leaf string
		after, cancel  bool
	}{
		{"cancel after first", "", "", false, true},
		{"plan sync", "sync", "plan.json", false, false},
		{"report short write", "short", "report.pending.json", false, false},
		{"report sync", "sync", "report.pending.json", false, false},
		{"report close", "close", "report.pending.json", false, false},
		{"report link", "link", "report.json", false, false},
		{"after report link", "link", "report.json", true, false},
		{"report cleanup", "remove", "report.pending.json", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, p := suiteFixture(t, "close-issue")
			prepared, err := PrepareSuite(context.Background(), root, suiteBytes(t, p))
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			open := func(root string) (evidenceFS, error) {
				fs, err := openRootedEvidenceFS(root)
				if err != nil {
					return nil, err
				}
				return &failingEvidenceFS{evidenceFS: fs, op: tc.op, leaf: tc.leaf, after: tc.after, hook: func(op string) {
					if tc.cancel && op == "link:terminal.json" {
						cancel()
					}
				}}, nil
			}
			r, err := runSuite(ctx, root, "suite", prepared, open, maxSuiteWriteBytes)
			if err == nil {
				t.Fatal("storage/cancellation error swallowed")
			}
			if tc.cancel {
				if !errors.Is(err, context.Canceled) || r == nil || r.ExecutionStatus != "canceled" || r.Trials[1].State != "not_started" || r.ComparisonStatus != "not_attempted" {
					t.Fatal(r, err)
				}
				if _, err := os.Stat(filepath.Join(root, "suite", "candidate-01")); !os.IsNotExist(err) {
					t.Fatal("new trial after cancellation")
				}
			}
		})
	}
	root, p := suiteFixture(t, "close-issue")
	prepared, err := PrepareSuite(context.Background(), root, suiteBytes(t, p))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runSuite(context.Background(), root, "bounded", prepared, openRootedEvidenceFS, 1); err == nil || err.Error() != "SUITE_RESOURCE_LIMIT" {
		t.Fatal("write limit lost", err)
	}
}
