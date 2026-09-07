package agenteval

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func pairPlan() *ComparePlan {
	return &ComparePlan{Format: CompareFormat, BaselineModel: "mock-baseline", CandidateModel: "mock-candidate", AllowedDifferences: []string{"model"}, Pairs: []PlannedPair{{TaskID: "close-issue", Repeat: 1, Baseline: PlannedTrial{TrialID: "baseline", Artifact: "baseline/terminal.json"}, Candidate: PlannedTrial{TrialID: "candidate", Artifact: "candidate/terminal.json"}}}}
}

func TestComparisonDecisionsAndDenominators(t *testing.T) {
	for _, tc := range []struct {
		name, want         string
		omit, changeBudget bool
	}{{"equal", "no_regression_observed", false, false}, {"regression", "regression", true, false}, {"budget changes", "incomparable", false, true}} {
		t.Run(tc.name, func(t *testing.T) {
			_, load := kit(t)
			bundleBytes := rawBundle(t)
			root := t.TempDir()
			for _, which := range []string{"baseline", "candidate"} {
				ta, w := load("close-issue")
				cfg := mockConfig(which)
				cfg.Model = "mock-" + which
				if which == "candidate" && tc.omit {
					w.Calls = nil
				}
				if which == "candidate" && tc.changeBudget {
					ta.Budgets.ToolAttempts--
				}
				if _, err := RecordMock(context.Background(), root, which, ta, bundleBytes, cfg, mockWitness(w)); err != nil {
					t.Fatal(err)
				}
			}
			r, err := Compare(context.Background(), root, pairPlan())
			if err != nil {
				t.Fatal(err)
			}
			if r.Decision != tc.want || r.UpgradeAllowed || r.Counts != (Denominators{Planned: 2, Started: 2, Terminal: 2, ValidlyEvaluated: 2}) {
				t.Fatalf("%+v", r)
			}
			if !strings.Contains(r.Markdown(), tc.want) {
				t.Fatal("markdown omitted decision")
			}
		})
	}
}

func TestMissingAndIncompleteTrialsStayInPlan(t *testing.T) {
	root := t.TempDir()
	p := pairPlan()
	r, err := Compare(context.Background(), root, p)
	if err != nil {
		t.Fatal(err)
	}
	if r.Decision != "inconclusive" || r.Counts != (Denominators{Planned: 2, NotStarted: 2}) {
		t.Fatalf("%+v", r)
	}
	if err = os.Mkdir(filepath.Join(root, "baseline"), 0700); err != nil {
		t.Fatal(err)
	}
	r, err = Compare(context.Background(), root, p)
	if err != nil {
		t.Fatal(err)
	}
	if r.Counts != (Denominators{Planned: 2, NotStarted: 1, Started: 1, Incomplete: 1}) {
		t.Fatalf("%+v", r)
	}
	p.Pairs = append(p.Pairs, p.Pairs[0])
	if _, err = Compare(context.Background(), root, p); err == nil {
		t.Fatal("duplicate trial accepted")
	}
}

func TestComparisonRejectsTrialIdentitySubstitution(t *testing.T) {
	root, _ := record(t)
	p := pairPlan()
	p.Pairs[0].Baseline.Artifact = "trial/terminal.json"
	r, err := Compare(context.Background(), root, p)
	if err != nil {
		t.Fatal(err)
	}
	if r.Decision != "incomparable" || r.Counts.ValidlyEvaluated != 0 {
		t.Fatalf("%+v", r)
	}
}
