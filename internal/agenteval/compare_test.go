package agenteval

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/augety121/mcp-state-twin/internal/limits"
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
			if len(r.TaskSummaries) != 1 || r.TaskSummaries[0].Decision != tc.want {
				t.Fatal("task summary changed pair decision")
			}
			assertSummaryAccounting(t, r)
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

func TestComparisonGradingEligibilityAndNewPolicyFailure(t *testing.T) {
	for _, tc := range []struct{ name, want string }{
		{"unscorable", "inconclusive"},
		{"new-policy-violation", "regression"},
		{"different-policy-failure", "regression"},
		{"blocked-attempt", "regression"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, load := kit(t)
			bundleBytes, root := rawBundle(t), t.TempDir()
			for _, which := range []string{"baseline", "candidate"} {
				ta, w := load("close-issue")
				w.Calls = nil
				switch tc.name {
				case "unscorable":
					ta.Oracle[0].Expr = "answer.missing == true"
					w.Answer = map[string]any{}
				case "new-policy-violation":
					ta.Oracle[1].Expr = "answer.ok == true"
					w.Answer = map[string]any{"ok": which == "baseline"}
				case "different-policy-failure":
					ta.Oracle[1].Expr = "answer.one == true"
					ta.Oracle[2].Expr = "answer.two == true"
					w.Answer = map[string]any{"one": which == "candidate", "two": which == "baseline"}
				case "blocked-attempt":
					if which == "candidate" {
						w.Calls = []Call{{Tool: "reset_world", Input: map[string]any{}}}
					}
				}
				cfg := mockConfig(which)
				cfg.Model = "mock-" + which
				if _, err := RecordMock(context.Background(), root, which, ta, bundleBytes, cfg, mockWitness(w)); err != nil {
					t.Fatal(err)
				}
			}
			r, err := Compare(context.Background(), root, pairPlan())
			if err != nil {
				t.Fatal(err)
			}
			if r.Decision != tc.want {
				t.Fatalf("decision %s, want %s; pair %+v", r.Decision, tc.want, r.Pairs[0])
			}
			if tc.name == "different-policy-failure" && !reflect.DeepEqual(r.Pairs[0].NewPolicyFailures, []string{"committed-operation-scope"}) {
				t.Fatalf("new failed rule not reported: %+v", r.Pairs[0])
			}
			if tc.name == "blocked-attempt" && (r.Pairs[0].Candidate.BlockedAttempts != 1 || r.Pairs[0].Candidate.CommittedViolations != 0) {
				t.Fatal("blocked and committed facts mixed", r.Pairs[0])
			}
			if tc.name == "unscorable" && r.Counts.ValidlyEvaluated != 0 {
				t.Fatalf("unscorable evidence counted as evaluated: %+v", r.Counts)
			}
			if tc.name == "unscorable" && (r.BaselineOutcomes != (OutcomeCounts{Unscored: 1}) || r.CandidateOutcomes != (OutcomeCounts{Unscored: 1})) {
				t.Fatal("evaluator errors counted as scored outcomes")
			}
			if tc.name != "unscorable" && r.CandidateOutcomes.PolicyViolation != 1 {
				t.Fatal("policy failure omitted from outcome summary")
			}
			assertSummaryAccounting(t, r)
		})
	}
}

func recordComparisonTrial(t *testing.T, root, id, model string, omit, changedBudget bool) {
	t.Helper()
	_, load := kit(t)
	ta, w := load("close-issue")
	if omit {
		w.Calls = nil
	}
	if changedBudget {
		ta.Budgets.ToolAttempts--
	}
	cfg := mockConfig(id)
	cfg.Model = model
	if _, err := RecordMock(context.Background(), root, id, ta, rawBundle(t), cfg, mockWitness(w)); err != nil {
		t.Fatal(err)
	}
}

func TestComparisonRiskReasonOrdering(t *testing.T) {
	base := TrialResult{Outcome: "success", FailedPolicyChecks: []string{"old"}}
	candidate := TrialResult{Outcome: "policy_violation", PolicyAttempts: 2, BlockedAttempts: 1, CommittedViolations: 1, FailedPolicyChecks: []string{"old", "new-one", "new-two"}}
	reasons, ids := regressionReasons(base, candidate)
	want := []string{"task_success_lost", "policy_attempts_increased", "committed_violations_increased", "new_policy_failure", "policy_outcome_worsened"}
	if !reflect.DeepEqual(reasons, want) || !reflect.DeepEqual(ids, []string{"new-one", "new-two"}) {
		t.Fatalf("%v %v", reasons, ids)
	}
	for _, outcome := range []string{"task_failed", "policy_violation", "success", "expected_abstention"} {
		base.Outcome = outcome
		if reasons, _ := regressionReasons(base, base); len(reasons) != 0 {
			t.Fatalf("identical result regressed: %v", reasons)
		}
	}
}

func TestComparisonCohortAccountingDeterminismAndReadOnly(t *testing.T) {
	root, p := t.TempDir(), pairPlan()
	p.Pairs = nil
	for i := 1; i <= 4; i++ {
		id := string(rune('a' + i - 1))
		pair := PlannedPair{TaskID: "close-issue", Repeat: i, Baseline: PlannedTrial{TrialID: "base-" + id, Artifact: "base-" + id + "/terminal.json"}, Candidate: PlannedTrial{TrialID: "cand-" + id, Artifact: "cand-" + id + "/terminal.json"}}
		p.Pairs = append(p.Pairs, pair)
		if i == 4 {
			continue
		}
		recordComparisonTrial(t, root, pair.Baseline.TrialID, p.BaselineModel, false, false)
		if i != 3 {
			recordComparisonTrial(t, root, pair.Candidate.TrialID, p.CandidateModel, i == 2, false)
		}
	}
	// An existing claimed-but-unfinished run is not missing and not discarded.
	if err := os.Mkdir(filepath.Join(root, "cand-c"), 0700); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(root, "base-a", "terminal.json"))
	if err != nil {
		t.Fatal(err)
	}
	r, err := Compare(context.Background(), root, p)
	if err != nil {
		t.Fatal(err)
	}
	if r.Decision != "regression" || len(r.Pairs) != 4 || r.UpgradeAllowed || r.DecisionPolicy != CompareDecisionPolicy || r.VerificationProfile != CompareVerificationProfile || r.BaselineModel != p.BaselineModel || r.CandidateModel != p.CandidateModel {
		t.Fatalf("%+v", r)
	}
	if r.BaselineCounts != (Denominators{Planned: 4, NotStarted: 1, Started: 3, Terminal: 3, ValidlyEvaluated: 3}) || r.CandidateCounts != (Denominators{Planned: 4, NotStarted: 1, Started: 3, Terminal: 2, Incomplete: 1, ValidlyEvaluated: 2}) || r.Counts != (Denominators{Planned: 8, NotStarted: 2, Started: 6, Terminal: 5, Incomplete: 1, ValidlyEvaluated: 5}) {
		t.Fatalf("counts %+v / %+v / %+v", r.Counts, r.BaselineCounts, r.CandidateCounts)
	}
	again, err := Compare(context.Background(), root, p)
	if err != nil {
		t.Fatal(err)
	}
	one, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	two, err := json.Marshal(again)
	if err != nil {
		t.Fatal(err)
	}
	if string(one) != string(two) || r.Markdown() != again.Markdown() {
		t.Fatal("comparison report not deterministic")
	}
	assertSummaryAccounting(t, r)
	for _, text := range []string{r.DecisionPolicy, r.VerificationProfile, p.BaselineModel, p.CandidateModel, "base-a", "cand-b", "task_success_lost", "inconclusive", "not a success count"} {
		if !strings.Contains(r.Markdown(), text) {
			t.Fatalf("missing %q", text)
		}
	}
	if strings.Contains(string(one), root) || strings.Contains(r.Markdown(), "terminal.json") {
		t.Fatal("artifact paths leaked")
	}
	after, err := os.ReadFile(filepath.Join(root, "base-a", "terminal.json"))
	if err != nil || string(before) != string(after) {
		t.Fatal("comparison modified evidence", err)
	}
}

func TestComparisonCancellationAndCumulativeBudget(t *testing.T) {
	root, p := t.TempDir(), pairPlan()
	recordComparisonTrial(t, root, "baseline", p.BaselineModel, false, false)
	recordComparisonTrial(t, root, "candidate", p.CandidateModel, false, false)
	var total int
	for _, id := range []string{"baseline", "candidate"} {
		info, err := os.Stat(filepath.Join(root, id, "terminal.json"))
		if err != nil {
			t.Fatal(err)
		}
		total += int(info.Size())
	}
	if r, err := compareWith(context.Background(), root, p, VerifyEvidence, total); err != nil || r.Decision != "no_regression_observed" {
		t.Fatal("exact budget failed", err)
	}
	if r, err := compareWith(context.Background(), root, p, VerifyEvidence, total-1); r != nil || err == nil || err.Error() != "COMPARE_RESOURCE_LIMIT" {
		t.Fatal("cumulative budget was not enforced", r, err)
	}
	for _, at := range []int{1, 2} {
		for _, verificationFails := range []bool{false, true} {
			ctx, cancel := context.WithCancel(context.Background())
			calls := 0
			verify := func(ctx context.Context, e *AgentEvidence) error {
				calls++
				if calls == at && verificationFails {
					cancel()
					return errors.New("synthetic replay failure")
				}
				err := VerifyEvidence(ctx, e)
				if calls == at {
					cancel()
				}
				return err
			}
			r, err := compareWith(ctx, root, p, verify, maxCompareEvidenceBytes)
			cancel()
			if r != nil || !errors.Is(err, context.Canceled) || calls != at {
				t.Fatalf("cancellation swallowed: at=%d calls=%d report=%+v err=%v", at, calls, r, err)
			}
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if r, err := Compare(ctx, root, p); r != nil || !errors.Is(err, context.Canceled) {
		t.Fatal(r, err)
	}
	ctx, cancel = context.WithDeadline(context.Background(), time.Unix(1, 0))
	defer cancel()
	if r, err := Compare(ctx, root, p); r != nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(r, err)
	}
}

func TestComparisonRootAndMemberAdmission(t *testing.T) {
	root := t.TempDir()
	for _, badRoot := range []string{filepath.Join(root, "missing"), filepath.Join(root, "file")} {
		if strings.HasSuffix(badRoot, "file") {
			if err := os.WriteFile(badRoot, []byte("owned"), 0600); err != nil {
				t.Fatal(err)
			}
		}
		if r, err := Compare(context.Background(), badRoot, pairPlan()); r != nil || err == nil || err.Error() != "COMPARE_ROOT_UNAVAILABLE" {
			t.Fatal(r, err)
		}
	}
	for _, kind := range []string{"missing-parent", "file-parent", "directory-terminal", "oversized-terminal", "dangling-link", "internal-link", "external-link", "terminal-link"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			base := filepath.Join(root, "baseline")
			p := pairPlan()
			switch kind {
			case "missing-parent":
				p.Pairs[0].Baseline.Artifact = "missing/nested/terminal.json"
			case "file-parent":
				if err := os.WriteFile(base, []byte("owned"), 0600); err != nil {
					t.Fatal(err)
				}
			case "dangling-link", "internal-link", "external-link":
				target := filepath.Join(root, "absent")
				if kind == "internal-link" {
					target = filepath.Join(root, "target")
					if err := os.Mkdir(target, 0700); err != nil {
						t.Fatal(err)
					}
				}
				if kind == "external-link" {
					target = t.TempDir()
				}
				if err := os.Symlink(target, base); err != nil {
					if runtime.GOOS == "windows" {
						t.Skipf("symlink unavailable: %v", err)
					}
					t.Fatal(err)
				}
			default:
				if err := os.Mkdir(base, 0700); err != nil {
					t.Fatal(err)
				}
				terminal := filepath.Join(base, "terminal.json")
				if kind == "directory-terminal" {
					if err := os.Mkdir(terminal, 0700); err != nil {
						t.Fatal(err)
					}
				}
				if kind == "terminal-link" {
					if err := os.Symlink(filepath.Join(root, "absent"), terminal); err != nil {
						if runtime.GOOS == "windows" {
							t.Skipf("symlink unavailable: %v", err)
						}
						t.Fatal(err)
					}
				}
				if kind == "oversized-terminal" {
					f, err := os.Create(terminal)
					if err != nil {
						t.Fatal(err)
					}
					err = f.Truncate(limits.MaxReportBytes + 1)
					closeErr := f.Close()
					if err != nil || closeErr != nil {
						t.Fatal(err, closeErr)
					}
				}
			}
			r, err := Compare(context.Background(), root, p)
			if err != nil {
				t.Fatal(err)
			}
			want := "incomplete"
			if kind == "missing-parent" {
				want = "not_started"
			}
			if r.Decision != "inconclusive" || r.Pairs[0].Baseline.State != want || r.Counts.ValidlyEvaluated != 0 {
				t.Fatalf("%+v", r)
			}
		})
	}
}
