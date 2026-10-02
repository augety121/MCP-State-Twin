package agenteval

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/augety121/mcp-state-twin/internal/bundle"
	"github.com/augety121/mcp-state-twin/internal/evaluator"
	"github.com/augety121/mcp-state-twin/internal/task"
)

func TestCaseCountBoundary(t *testing.T) {
	root, m := caseFixture(t)
	seed := m.Cases[0]
	m.Cases = nil
	for i := 0; i < 64; i++ {
		c := seed
		c.CaseID = fmt.Sprintf("case-%02d", i)
		m.Cases = append(m.Cases, c)
	}
	writeTestJSON(t, root, "cases.json", m)
	if p, err := prepareCases(context.Background(), root, "cases.json"); err != nil || len(p.inputs) != 64 {
		t.Fatal(p, err)
	}
	seed.CaseID = "overflow"
	m.Cases = append(m.Cases, seed)
	writeTestJSON(t, root, "cases.json", m)
	if _, err := prepareCases(context.Background(), root, "cases.json"); err == nil {
		t.Fatal("65 cases accepted")
	}
}

func caseFixture(t *testing.T) (string, *CaseManifest) {
	t.Helper()
	root, p := suiteFixture(t, "close-issue")
	_, load := kit(t)
	_, w := load("close-issue")
	writeTestJSON(t, root, "witness.json", w)
	m := &CaseManifest{Format: "statetwin.dev/task-cases/v1alpha1", Profile: "synthetic-witness-cases-v1", Tasks: []taskReference{{"close-issue", p.Pairs[0].Task}}, Cases: []TaskCase{{CaseID: "positive", TaskID: "close-issue", Role: "positive", Witness: "witness.json", Expected: CaseExpected{Outcome: "success", FailedChecks: []string{}}}}}
	writeTestJSON(t, root, "cases.json", m)
	return root, m
}
func TestCaseRunnerIsolationFreezeAndInfrastructure(t *testing.T) {
	ctx := context.Background()
	root, m := caseFixture(t)
	var empty Witness
	raw, _ := os.ReadFile(filepath.Join(root, "witness.json"))
	json.Unmarshal(raw, &empty)
	empty.Calls = []Call{}
	writeTestJSON(t, root, "empty.json", &empty)
	m.Cases = append(m.Cases, TaskCase{CaseID: "omission", TaskID: "close-issue", Role: "goal-negative", Witness: "empty.json", Expected: CaseExpected{Outcome: "task_failed", FailedChecks: []string{"objective"}}})
	writeTestJSON(t, root, "cases.json", m)
	p, err := prepareCases(ctx, root, "cases.json")
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(root, "witness.json"), []byte("changed"), 0600)
	r, err := runCases(ctx, p, RunWitness)
	if err != nil || r.Matched != 2 || r.Decision != "matched" {
		t.Fatal(r, err)
	}
	// Make the expected positive run omit work. A mismatch must not stop the next case.
	p.inputs[0].witness = &empty
	r, err = runCases(ctx, p, RunWitness)
	if err != nil || r.Mismatched != 1 || r.Matched != 1 {
		t.Fatal(r, err)
	}
	called := 0
	r, err = runCases(ctx, p, func(context.Context, *task.Task, *bundle.Artifact, *Witness) (*Report, error) {
		called++
		return &Report{ExecutionStatus: "completed", CleanupStatus: "failed"}, errors.New("private failure")
	})
	if err == nil || called != 1 || r.Failed != 1 || r.NotStarted != 1 || r.Decision != "incomplete" || r.Cases[0].FailureCode != "CASE_CLEANUP_FAILED" {
		t.Fatal(r, err)
	}
	cancelCtx, cancel := context.WithCancel(ctx)
	cancel()
	r, err = runCases(cancelCtx, p, RunWitness)
	if !errors.Is(err, context.Canceled) || r.NotStarted != 2 {
		t.Fatal(r, err)
	}
}
func TestCaseManifestClosedAdmission(t *testing.T) {
	ctx := context.Background()
	root, m := caseFixture(t)
	for _, mutate := range []func(*CaseManifest){func(m *CaseManifest) { m.Cases[0].Expected.FailedChecks = []string{"unknown"} }, func(m *CaseManifest) { m.Cases[0].Role = "anything" }, func(m *CaseManifest) { m.Cases[0].Expected.Outcome = "not_evaluated" }, func(m *CaseManifest) { m.Cases = append(m.Cases, m.Cases[0]) }, func(m *CaseManifest) { m.Cases[0].Witness = "../private" }, func(m *CaseManifest) { m.Cases[0].TaskID = "other" }} {
		raw, _ := json.Marshal(m)
		var copy CaseManifest
		json.Unmarshal(raw, &copy)
		mutate(&copy)
		writeTestJSON(t, root, "bad.json", copy)
		if _, err := prepareCases(ctx, root, "bad.json"); err == nil {
			t.Fatal("bad manifest admitted")
		}
	}
	m.Cases = append(m.Cases, TaskCase{CaseID: "last-bad", TaskID: "close-issue", Role: "positive", Witness: "missing.json", Expected: CaseExpected{Outcome: "success", FailedChecks: []string{}}})
	writeTestJSON(t, root, "bad.json", m)
	if r, err := RunCases(ctx, root, "bad.json"); err == nil || r != nil {
		t.Fatal("partial preflight executed", r, err)
	}
}
func TestQualificationCoverageRequiresRealNegativeChecks(t *testing.T) {
	p := &preparedCases{manifest: CaseManifest{Tasks: []taskReference{{TaskID: "t"}}}, tasks: map[string]*task.Task{"t": {Oracle: []task.Assertion{{ID: "goal", Category: "goal"}, {ID: "policy", Category: "policy"}}}}}
	report := &CaseReport{Decision: "matched", Cases: []CaseRow{{TaskID: "t", CaseID: "positive", Role: "positive", State: "matched"}, {TaskID: "t", CaseID: "goal-negative", Role: "goal-negative", State: "matched", FailedCheckIDs: []string{"goal"}}, {TaskID: "t", CaseID: "policy-negative", Role: "policy-negative", State: "matched", FailedCheckIDs: []string{"policy"}}}}
	if r := qualifyCases(p, report); r.Decision != "qualified" {
		t.Fatal(r)
	}
	report.Cases[2].ErrorCheckIDs = []string{"policy"}
	if r := qualifyCases(p, report); r.Decision != "not_qualified" || r.Tasks[0].Checks[1].Status != "uncovered" {
		t.Fatal(r)
	}
	report.Cases[2].FailedCheckIDs = nil
	report.Cases[2].ErrorCheckIDs = nil
	if r := qualifyCases(p, report); r.Decision != "not_qualified" {
		t.Fatal("policy attempts alone qualified")
	}
	report.Cases[0].State = "mismatched"
	report.Decision = "mismatched"
	if r := qualifyCases(p, report); r.Reasons[0].Code != "case_run_not_matched" {
		t.Fatal(r)
	}
}
func TestUnscorableCaseCannotSwallowInfrastructureFailure(t *testing.T) {
	p := &preparedCases{manifest: CaseManifest{Cases: []TaskCase{{CaseID: "error", Role: "unscorable-negative", Expected: CaseExpected{Outcome: "evaluator_error", FailedChecks: []string{"goal"}}}}}, inputs: []caseInput{{}}}
	run := func(context.Context, *task.Task, *bundle.Artifact, *Witness) (*Report, error) {
		return &Report{ExecutionStatus: "completed", CleanupStatus: "complete", Evaluation: &evaluator.Result{Outcome: "evaluator_error", Checks: []evaluator.Check{{ID: "goal", Passed: false, Error: "bounded"}}}}, nil
	}
	r, err := runCases(context.Background(), p, run)
	if err != nil || r.Matched != 1 {
		t.Fatal(r, err)
	}
}
