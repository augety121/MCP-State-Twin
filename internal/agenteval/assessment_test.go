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
	"github.com/augety121/mcp-state-twin/internal/task"
)

func expectationFor(t *testing.T, root string, p *SuitePlan) *SuiteExpectation {
	t.Helper()
	e := &SuiteExpectation{Format: ExpectationFormat, Profile: SuiteProfile, MaxOutputTokens: p.MaxOutputTokens, Plan: p.comparisonPlan()}
	writeTestJSON(t, root, "expected.json", e)
	return e
}
func runAssessmentFixture(t *testing.T, root string, p *SuitePlan) {
	t.Helper()
	prepared, err := PrepareSuite(context.Background(), root, suiteBytes(t, p))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = RunSuite(context.Background(), root, "suite", prepared); err != nil {
		t.Fatal(err)
	}
}
func TestAssessmentPoliciesRealReplay(t *testing.T) {
	for _, mode := range []string{"pass", "baseline-fails", "both-fail", "regression", "abstention", "unscorable", "policy-violation"} {
		t.Run(mode, func(t *testing.T) {
			id := "close-issue"
			if mode == "abstention" {
				id = "already-closed"
			}
			root, p := suiteFixture(t, id)
			if oneOf(mode, "baseline-fails", "both-fail", "regression") {
				writeTestJSON(t, root, "omit.json", &agenthost.MockScript{Kind: "MockResponses", SyntheticOnly: true, Responses: []json.RawMessage{mockResponse(mockFinal(map[string]any{}))}})
				if mode != "regression" {
					p.Pairs[0].BaselineResponses = "omit.json"
				}
				if mode != "baseline-fails" {
					p.Pairs[0].CandidateResponses = "omit.json"
				}
			}
			if mode == "unscorable" || mode == "policy-violation" {
				var ta task.Task
				raw, err := os.ReadFile(filepath.Join(root, p.Pairs[0].Task))
				if err != nil {
					t.Fatal(err)
				}
				if err = json.Unmarshal(raw, &ta); err != nil {
					t.Fatal(err)
				}
				if mode == "unscorable" {
					ta.Oracle[0].Expr = "answer.missing == true"
				} else {
					ta.Oracle[1].Expr = "false"
				}
				writeTestJSON(t, root, p.Pairs[0].Task, ta)
			}
			expectationFor(t, root, p)
			runAssessmentFixture(t, root, p)
			before := suiteSnapshot(t, root)
			old, err := InspectSuite(context.Background(), root, "suite")
			if err != nil {
				t.Fatal(err)
			}
			for _, policy := range []string{"candidate-pass-v1", "both-pass-v1"} {
				r, err := AssessSuite(context.Background(), root, "suite", "expected.json", policy)
				if err != nil {
					t.Fatal(err)
				}
				wantPass := mode == "pass" || mode == "abstention" || (mode == "baseline-fails" && policy == "candidate-pass-v1")
				if (r.Decision == "passed") != wantPass || r.Expectation.Status != "matched" || r.Consistency[0].IdentityStatus != "single_pair" || r.UpgradeAllowed || r.Provenance != "not-proven" {
					t.Fatalf("%+v", r)
				}
				if !same(old, r.Audit) {
					t.Fatal("collector changed old audit")
				}
				if mode == "both-fail" && r.Audit.Comparison.Decision != "no_regression_observed" {
					t.Fatal("fixture does not reproduce equal failure")
				}
				if mode == "unscorable" && r.Consistency[0].VerifiedDefinitions != 2 {
					t.Fatal("replay-valid unscorable definition lost")
				}
				// Exercise deterministic rendering for both a pass and an ordered
				// failure, without replaying every policy matrix cell twice.
				if policy == "candidate-pass-v1" && (mode == "pass" || mode == "both-fail") {
					again, err := AssessSuite(context.Background(), root, "suite", "expected.json", policy)
					if err != nil || !same(r, again) || r.Markdown() != again.Markdown() {
						t.Fatal("nondeterministic assessment", err)
					}
				}
				text, _ := json.Marshal(r)
				if strings.Contains(string(text), "objective") || strings.Contains(r.Markdown(), root) {
					t.Fatal("private definition leaked")
				}
			}
			if !reflect.DeepEqual(before, suiteSnapshot(t, root)) {
				t.Fatal("assessment changed inputs")
			}
		})
	}
}

func TestAssessmentRepeatedDefinitions(t *testing.T) {
	for _, change := range []string{"none", "revision", "oracle", "authority", "budget", "output-budget"} {
		t.Run(change, func(t *testing.T) {
			root, p := suiteFixture(t, "close-issue", "close-issue")
			var ta task.Task
			raw, err := os.ReadFile(filepath.Join(root, p.Pairs[1].Task))
			if err != nil {
				t.Fatal(err)
			}
			if err = json.Unmarshal(raw, &ta); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "revision":
				ta.Revision = "v2"
			case "oracle":
				ta.Oracle[0].Expr = "false"
			case "authority":
				ta.Authority[0].Equals["owner"] = "other"
			case "budget":
				ta.Budgets.ModelRequests--
			}
			writeTestJSON(t, root, "second-task.json", ta)
			p.Pairs[1].Task = "second-task.json"
			e := expectationFor(t, root, p)
			prepared, err := PrepareSuite(context.Background(), root, suiteBytes(t, p))
			if err != nil {
				t.Fatal(err)
			}
			if change == "output-budget" {
				prepared.trials[2].config.MaxOutputTokens++
				prepared.trials[3].config.MaxOutputTokens++
			}
			if _, err := RunSuite(context.Background(), root, "suite", prepared); err != nil {
				t.Fatal(err)
			}
			r, err := AssessSuite(context.Background(), root, "suite", "expected.json", "candidate-pass-v1")
			if err != nil {
				t.Fatal(err)
			}
			if r.Audit.Comparison.Decision != "no_regression_observed" {
				t.Fatal("pair comparison changed")
			}
			g := r.Consistency[0]
			if g.VerifiedDefinitions != 4 || g.Coverage != "complete" {
				t.Fatal(g)
			}
			if change == "none" {
				if r.Decision != "passed" || g.IdentityStatus != "homogeneous" {
					t.Fatal(r)
				}
			} else if r.Decision != "failed" || g.IdentityStatus != "heterogeneous" || !reflect.DeepEqual(g.MismatchingTrialIDs, []string{"baseline-02", "candidate-02"}) {
				t.Fatal(r, g)
			}
			if change == "output-budget" && r.Expectation.Status != "mismatched" {
				t.Fatal("budget mismatch ignored")
			}
			if change == "none" {
				if _, err := assessPrepared(context.Background(), root, "suite", e, "candidate-pass-v1", 1); !errors.Is(err, errAssessmentResourceLimit) {
					t.Fatal("reference bound", err)
				}
			}
			// With a missing final trial, a proven difference stays heterogeneous.
			removeAudit(t, root, "report.json")
			removeAudit(t, root, "candidate-02/terminal.json")
			r, err = AssessSuite(context.Background(), root, "suite", "expected.json", "candidate-pass-v1")
			if err != nil {
				t.Fatal(err)
			}
			if r.Decision != "failed" || r.Consistency[0].UnavailableDefinitions != 1 {
				t.Fatal(r)
			}
			if change != "none" && r.Consistency[0].IdentityStatus != "heterogeneous" {
				t.Fatal("mismatch lost after missing evidence")
			}
		})
	}
}

func TestAssessmentExpectationAndAdmission(t *testing.T) {
	root, p := suiteFixture(t, "close-issue")
	e := expectationFor(t, root, p)
	runAssessmentFixture(t, root, p)
	for _, mutate := range []func(*SuiteExpectation){
		func(e *SuiteExpectation) { e.Plan.BaselineModel = "mock-other" },
		func(e *SuiteExpectation) { e.Plan.Pairs[0].Repeat = 2 },
		func(e *SuiteExpectation) { e.Plan.Pairs[0].TaskID = "other-task" },
		func(e *SuiteExpectation) { e.MaxOutputTokens++ },
		func(e *SuiteExpectation) {
			pair := e.Plan.Pairs[0]
			pair.Repeat = 2
			pair.Baseline = PlannedTrial{TrialID: "baseline-02", Artifact: "baseline-02/terminal.json"}
			pair.Candidate = PlannedTrial{TrialID: "candidate-02", Artifact: "candidate-02/terminal.json"}
			e.Plan.Pairs = append(e.Plan.Pairs, pair)
		},
	} {
		copy := *e
		copy.Plan.Pairs = append([]PlannedPair(nil), e.Plan.Pairs...)
		mutate(&copy)
		writeTestJSON(t, root, "changed.json", copy)
		r, err := AssessSuite(context.Background(), root, "suite", "changed.json", "candidate-pass-v1")
		if err != nil || r.Decision != "failed" || r.Expectation.Status != "mismatched" {
			t.Fatal(r, err)
		}
	}
	raw, _ := json.Marshal(e)
	for _, bad := range []string{
		strings.Replace(string(raw), `"maxOutputTokens":1024`, `"maxOutputTokens":1024.5`, 1),
		strings.Replace(string(raw), `"maxOutputTokens":1024`, `"maxOutputTokens":null`, 1),
		strings.Replace(string(raw), `"maxOutputTokens":1024,`, "", 1),
		strings.Replace(string(raw), `"repeat":1`, `"repeat":1.0`, 1),
		strings.Replace(string(raw), `"repeat":1`, `"repeat":1,"repeat":1`, 1),
		strings.Replace(string(raw), `"profile":`, `"unknown":true,"profile":`, 1),
		string(raw) + "{}", strings.Repeat(" ", MaxSuitePlanBytes+1),
	} {
		if got, err := DecodeExpectation([]byte(bad)); got != nil || err == nil {
			t.Fatal("invalid expectation accepted")
		}
	}
	for _, name := range []string{"suite/expected.json", "SUITE/expected.json", "../outside.json"} {
		if _, err := AssessSuite(context.Background(), root, "suite", name, "candidate-pass-v1"); err == nil {
			t.Fatal("unsafe expectation", name)
		}
	}
	if _, err := AssessSuite(context.Background(), root, "suite", "expected.json", "unknown"); err == nil {
		t.Fatal("policy accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if got, err := AssessSuite(ctx, root, "suite", "expected.json", "candidate-pass-v1"); got != nil || !errors.Is(err, context.Canceled) {
		t.Fatal("cancellation", err)
	}
	if r, err := AssessSuite(context.Background(), root, "missing", "expected.json", "candidate-pass-v1"); err != nil || r.Decision != "failed" || r.Expectation.Coverage.PlannedTrials != 2 || len(r.Consistency) != 0 {
		t.Fatal(r, err)
	}
	if err := os.Symlink(filepath.Join(root, "expected.json"), filepath.Join(root, "linked.json")); err == nil {
		if _, err := AssessSuite(context.Background(), root, "suite", "linked.json", "candidate-pass-v1"); err == nil {
			t.Fatal("symlink accepted")
		}
	} else {
		t.Log("symlinks unavailable")
	}
}

func TestDefinitionCollectorIdentityAndBounds(t *testing.T) {
	_, p := suiteFixture(t, "close-issue", "read-issue", "close-issue")
	e := &SuiteExpectation{Plan: p.comparisonPlan(), MaxOutputTokens: 1024}
	for _, field := range []string{"runtime", "revision", "snapshot", "bundle", "projection", "isolation"} {
		c := newAssessmentCollector(e, maxDefinitionReferences)
		c.plan(e.Plan)
		d := RunDefinition{Config: RunConfig{Model: "mock-a", TrialID: "baseline-01", MaxOutputTokens: 1024}}
		if err := c.observe("close-issue", "baseline-01", d); err != nil {
			t.Fatal(err)
		}
		d.Config.Model, d.Config.TrialID = "mock-b", "candidate-01"
		if err := c.observe("close-issue", "candidate-01", d); err != nil {
			t.Fatal(err)
		}
		switch field {
		case "runtime":
			d.RuntimeVersion = "other"
		case "revision":
			d.RuntimeRevision = "other"
		case "snapshot":
			d.ModelSnapshot = "other"
		case "bundle":
			d.BundleDigest = "other"
		case "projection":
			d.Projection = "other"
		case "isolation":
			d.Isolation = "other"
		}
		if err := c.observe("close-issue", "baseline-03", d); err != nil {
			t.Fatal(err)
		}
		_, g := c.results()
		if len(g) != 2 || g[0].IdentityStatus != "heterogeneous" || g[0].Coverage != "incomplete" || g[1].IdentityStatus != "unverifiable" || !reflect.DeepEqual(g[0].MismatchingTrialIDs, []string{"baseline-03"}) {
			t.Fatal(g)
		}
	}
}

func TestAssessmentRejectsUnverifiedRuntimeDefinitions(t *testing.T) {
	root, p := suiteFixture(t, "close-issue")
	expectationFor(t, root, p)
	runAssessmentFixture(t, root, p)
	file := filepath.Join(root, "suite", "baseline-01", "terminal.json")
	original, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*RunDefinition){
		func(d *RunDefinition) { d.RuntimeVersion = "other" }, func(d *RunDefinition) { d.RuntimeRevision = "other" },
		func(d *RunDefinition) { d.ModelSnapshot = "other" }, func(d *RunDefinition) { d.BundleDigest = "other" },
	} {
		var evidence AgentEvidence
		if err := json.Unmarshal(original, &evidence); err != nil {
			t.Fatal(err)
		}
		change(&evidence.Episode.Definition)
		writeTestJSON(t, root, "suite/baseline-01/terminal.json", evidence)
		r, err := AssessSuite(context.Background(), root, "suite", "expected.json", "candidate-pass-v1")
		if err != nil || r.Decision != "failed" || r.Audit.State != "invalid" || r.Expectation.Coverage.VerifiedTrials != 0 {
			t.Fatal(r, err)
		}
	}
}

func TestAssessmentCollectorMaximumPlanAndCancellation(t *testing.T) {
	_, p := suiteFixture(t, "close-issue")
	for i := 2; i <= 16; i++ {
		pair := p.Pairs[0]
		pair.Repeat = i
		p.Pairs = append(p.Pairs, pair)
	}
	e := &SuiteExpectation{Plan: p.comparisonPlan(), MaxOutputTokens: 1024}
	c := newAssessmentCollector(e, maxDefinitionReferences)
	c.plan(e.Plan)
	for _, pair := range e.Plan.Pairs {
		for _, trial := range []PlannedTrial{pair.Baseline, pair.Candidate} {
			if err := c.observe(pair.TaskID, trial.TrialID, RunDefinition{Config: RunConfig{TrialID: trial.TrialID, Model: "mock-label", MaxOutputTokens: 1024}}); err != nil {
				t.Fatal(err)
			}
		}
	}
	expected, groups := c.results()
	if expected.Status != "matched" || expected.Coverage.VerifiedTrials != 32 || groups[0].PlannedPairs != 16 || groups[0].IdentityStatus != "homogeneous" {
		t.Fatal(expected, groups)
	}
	root, p := suiteFixture(t, "close-issue")
	expectationFor(t, root, p)
	runAssessmentFixture(t, root, p)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r, err := inspectSuiteObserved(ctx, root, "suite", maxSuiteWriteBytes, func(ComparePlan) { cancel() }, nil)
	if r != nil || !errors.Is(err, context.Canceled) {
		t.Fatal("mid-audit cancellation lost", r, err)
	}
}

func TestAssessmentMissingRepeatsAndPlanChanges(t *testing.T) {
	root, p := suiteFixture(t, "close-issue", "close-issue")
	e := expectationFor(t, root, p)
	runAssessmentFixture(t, root, p)
	for _, drop := range []bool{false, true} {
		copy := *e
		copy.Plan.Pairs = append([]PlannedPair(nil), e.Plan.Pairs...)
		if drop {
			copy.Plan.Pairs = copy.Plan.Pairs[:1]
		} else {
			copy.Plan.Pairs[0].Repeat, copy.Plan.Pairs[1].Repeat = 2, 1
		}
		writeTestJSON(t, root, "changed.json", copy)
		r, err := AssessSuite(context.Background(), root, "suite", "changed.json", "both-pass-v1")
		if err != nil || r.Decision != "failed" || r.Expectation.Status != "mismatched" || r.Expectation.Coverage.VerifiedTrials != 0 {
			t.Fatal(r, err)
		}
	}
	removeAudit(t, root, "report.json")
	files := []string{"baseline-01/terminal.json", "candidate-01/terminal.json", "baseline-02/terminal.json", "candidate-02/terminal.json"}
	for _, file := range files {
		name := filepath.Join(root, "suite", filepath.FromSlash(file))
		raw, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		removeAudit(t, root, file)
		r, err := AssessSuite(context.Background(), root, "suite", "expected.json", "both-pass-v1")
		if err != nil || r.Decision != "failed" || r.Expectation.Coverage.VerifiedTrials != 3 || r.Consistency[0].IdentityStatus != "unverifiable" || r.Consistency[0].UnavailableDefinitions != 1 {
			t.Fatal(r, err)
		}
		if err = os.WriteFile(name, raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, file := range files {
		removeAudit(t, root, file)
	}
	r, err := AssessSuite(context.Background(), root, "suite", "expected.json", "both-pass-v1")
	if err != nil || r.Consistency[0].UnavailableDefinitions != 4 || r.Consistency[0].VerifiedDefinitions != 0 {
		t.Fatal(r, err)
	}
}
