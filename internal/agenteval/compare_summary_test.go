package agenteval

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestComparePlanRepeatRequiresOriginalInteger(t *testing.T) {
	for _, marshal := range []func(any) ([]byte, error){json.Marshal, yaml.Marshal} {
		raw, err := marshal(pairPlan())
		if err != nil {
			t.Fatal(err)
		}
		needle := `"repeat":1`
		if !strings.Contains(string(raw), needle) {
			needle = "repeat: 1"
		}
		for _, token := range []string{"1.5", "1.0", "1e0", `"1"`, "null", "true", "0", "17", "-1"} {
			replacement := strings.TrimSuffix(needle, "1") + token
			input := strings.Replace(string(raw), needle, replacement, 1)
			if p, err := DecodeCompare([]byte(input)); err == nil || p != nil {
				t.Errorf("repeat %s admitted", token)
			}
		}
		for _, token := range []string{"1", "16"} {
			input := strings.Replace(string(raw), needle, strings.TrimSuffix(needle, "1")+token, 1)
			if p, err := DecodeCompare([]byte(input)); err != nil || p == nil {
				t.Fatal("valid integer rejected", err)
			}
		}
	}
	raw, err := yaml.Marshal(pairPlan())
	if err != nil {
		t.Fatal(err)
	}
	if p, err := DecodeCompare([]byte(strings.Replace(string(raw), "repeat: 1", "", 1))); err == nil || p != nil {
		t.Fatal("missing repeat accepted")
	}
}

func TestTaskSummariesFromRealEvidence(t *testing.T) {
	root, p := t.TempDir(), pairPlan()
	p.Pairs = nil
	_, load := kit(t)
	bundleBytes := rawBundle(t)
	for i, id := range []string{"close-issue", "already-closed", "close-issue", "read-issue"} {
		pair := PlannedPair{TaskID: id, Repeat: i + 1,
			Baseline:  PlannedTrial{TrialID: fmt.Sprintf("base-%d", i), Artifact: fmt.Sprintf("base-%d/terminal.json", i)},
			Candidate: PlannedTrial{TrialID: fmt.Sprintf("cand-%d", i), Artifact: fmt.Sprintf("cand-%d/terminal.json", i)}}
		p.Pairs = append(p.Pairs, pair)
		if i == 3 {
			continue
		} // Entirely missing tasks must still have a summary.
		for _, side := range []struct {
			trial     PlannedTrial
			model     string
			candidate bool
		}{
			{pair.Baseline, p.BaselineModel, false}, {pair.Candidate, p.CandidateModel, true},
		} {
			ta, w := load(id)
			if i == 2 && side.candidate {
				w.Calls = nil
			}
			cfg := mockConfig(side.trial.TrialID)
			cfg.Model = side.model
			if _, err := RecordMock(context.Background(), root, side.trial.TrialID, ta, bundleBytes, cfg, mockWitness(w)); err != nil {
				t.Fatal(err)
			}
		}
	}
	r, err := Compare(context.Background(), root, p)
	if err != nil {
		t.Fatal(err)
	}
	if r.SummaryPolicy != CompareSummaryPolicy || len(r.TaskSummaries) != 3 || r.Decision != "regression" || r.UpgradeAllowed {
		t.Fatalf("%+v", r)
	}
	full := Denominators{Planned: 2, Started: 2, Terminal: 2, ValidlyEvaluated: 2}
	want := []TaskSummary{
		{TaskID: "close-issue", PlannedPairs: 2, Decision: "regression", DecisionCounts: DecisionCounts{Regression: 1, NoRegressionObserved: 1}, Baseline: TaskCohort{Counts: full, Outcomes: OutcomeCounts{Success: 2}}, Candidate: TaskCohort{Counts: full, Outcomes: OutcomeCounts{Success: 1, TaskFailed: 1}}},
		{TaskID: "already-closed", PlannedPairs: 1, Decision: "no_regression_observed", DecisionCounts: DecisionCounts{NoRegressionObserved: 1}, Baseline: TaskCohort{Counts: Denominators{Planned: 1, Started: 1, Terminal: 1, ValidlyEvaluated: 1}, Outcomes: OutcomeCounts{ExpectedAbstention: 1}}, Candidate: TaskCohort{Counts: Denominators{Planned: 1, Started: 1, Terminal: 1, ValidlyEvaluated: 1}, Outcomes: OutcomeCounts{ExpectedAbstention: 1}}},
		{TaskID: "read-issue", PlannedPairs: 1, Decision: "inconclusive", DecisionCounts: DecisionCounts{Inconclusive: 1}, Baseline: TaskCohort{Counts: Denominators{Planned: 1, NotStarted: 1}, Outcomes: OutcomeCounts{Unscored: 1}}, Candidate: TaskCohort{Counts: Denominators{Planned: 1, NotStarted: 1}, Outcomes: OutcomeCounts{Unscored: 1}}},
	}
	if !reflect.DeepEqual(r.TaskSummaries, want) {
		t.Fatalf("summaries: %+v", r.TaskSummaries)
	}
	if r.BaselineOutcomes != (OutcomeCounts{Success: 2, ExpectedAbstention: 1, Unscored: 1}) || r.CandidateOutcomes != (OutcomeCounts{Success: 1, ExpectedAbstention: 1, TaskFailed: 1, Unscored: 1}) {
		t.Fatal("global outcome totals", r)
	}
	assertSummaryAccounting(t, r)
	for _, line := range []string{
		"| close-issue | 2 | regression | 1 | 0 | 0 | 1 |",
		"| read-issue | Candidate | 1 | 1 | 0 | 0 | 0 | 0 |",
		"| close-issue | Candidate | 1 | 0 | 1 | 0 | 0 |",
		"| already-closed | Baseline | 0 | 1 | 0 | 0 | 0 |",
		"| All planned tasks | Candidate | 1 | 1 | 1 | 0 | 1 |",
		"## Pair details", "not success rates",
	} {
		if !strings.Contains(r.Markdown(), line) {
			t.Fatalf("missing markdown %q", line)
		}
	}
}

func assertSummaryAccounting(t *testing.T, r *Comparison) {
	t.Helper()
	for _, candidate := range []bool{false, true} {
		var counts Denominators
		var outcomes OutcomeCounts
		for _, s := range r.TaskSummaries {
			c := s.Baseline
			if candidate {
				c = s.Candidate
			}
			d, o := c.Counts, c.Outcomes
			if d.Planned != d.NotStarted+d.Started || d.Started != d.Incomplete+d.Terminal || d.Planned != s.PlannedPairs || d.ValidlyEvaluated > d.Terminal || d.ValidlyEvaluated != o.Success+o.ExpectedAbstention+o.TaskFailed+o.PolicyViolation || d.Planned != d.ValidlyEvaluated+o.Unscored {
				t.Fatal("invalid task denominator", s)
			}
			dec := s.DecisionCounts
			if s.PlannedPairs != dec.Regression+dec.Incomparable+dec.Inconclusive+dec.NoRegressionObserved {
				t.Fatal("lost pair", s)
			}
			counts.Planned += d.Planned
			counts.NotStarted += d.NotStarted
			counts.Started += d.Started
			counts.Incomplete += d.Incomplete
			counts.Terminal += d.Terminal
			counts.ValidlyEvaluated += d.ValidlyEvaluated
			outcomes.Success += o.Success
			outcomes.ExpectedAbstention += o.ExpectedAbstention
			outcomes.TaskFailed += o.TaskFailed
			outcomes.PolicyViolation += o.PolicyViolation
			outcomes.Unscored += o.Unscored
		}
		wantCounts, wantOutcomes := r.BaselineCounts, r.BaselineOutcomes
		if candidate {
			wantCounts, wantOutcomes = r.CandidateCounts, r.CandidateOutcomes
		}
		if counts != wantCounts || outcomes != wantOutcomes {
			t.Fatal("task totals disagree with global totals")
		}
	}
}

func TestTaskSummaryUnscoredEvidence(t *testing.T) {
	for _, kind := range []string{"invalid", "partial", "identity_mismatch"} {
		t.Run(kind, func(t *testing.T) {
			root, p := t.TempDir(), pairPlan()
			recordComparisonTrial(t, root, "baseline", p.BaselineModel, false, false)
			recordComparisonTrial(t, root, "candidate", p.CandidateModel, false, false)
			file := filepath.Join(root, "candidate", "terminal.json")
			raw, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			e, err := DecodeEvidence(raw)
			if err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "partial":
				e.Episode.EvidenceStatus = "partial"
			case "identity_mismatch":
				e.Episode.Definition.Config.TrialID = "another-trial"
			}
			raw, err = json.Marshal(e)
			if err != nil {
				t.Fatal(err)
			}
			if kind == "invalid" {
				raw = []byte("{")
			}
			if err := os.WriteFile(file, raw, 0600); err != nil {
				t.Fatal(err)
			}
			r, err := Compare(context.Background(), root, p)
			if err != nil {
				t.Fatal(err)
			}
			if r.Pairs[0].Candidate.Validation != kind || r.CandidateOutcomes != (OutcomeCounts{Unscored: 1}) {
				t.Fatal("unsafe claimed success counted", r)
			}
			assertSummaryAccounting(t, r)
		})
	}
}

func TestTaskSummaryMaximumPlanAndUnknownValues(t *testing.T) {
	p := pairPlan()
	p.Pairs = nil
	for i := 0; i < 32; i++ {
		p.Pairs = append(p.Pairs, PlannedPair{TaskID: fmt.Sprintf("task-%d", i), Repeat: 1, Baseline: PlannedTrial{TrialID: fmt.Sprintf("base-%d", i), Artifact: fmt.Sprintf("base-%d/terminal.json", i)}, Candidate: PlannedTrial{TrialID: fmt.Sprintf("cand-%d", i), Artifact: fmt.Sprintf("cand-%d/terminal.json", i)}})
	}
	r, err := Compare(context.Background(), t.TempDir(), p)
	if err != nil || len(r.TaskSummaries) != 32 || r.CandidateOutcomes.Unscored != 32 {
		t.Fatal(r, err)
	}
	assertSummaryAccounting(t, r)
	for _, pair := range []PairResult{
		{TaskID: "task", Decision: "unknown"},
		{TaskID: "task", Decision: "inconclusive", Baseline: TrialResult{Validation: "valid", Outcome: "unknown"}},
	} {
		r := &Comparison{Pairs: []PairResult{pair}}
		if err := r.summarizeTasks(); err == nil || err.Error() != "COMPARE_SUMMARY_INVALID" {
			t.Fatal("unknown summary admitted", err)
		}
	}
}
