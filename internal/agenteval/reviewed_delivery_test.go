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

	"github.com/augety121/mcp-state-twin/internal/task"
)

func catalogFixture(t *testing.T, root string, p *SuitePlan) *taskCatalog {
	t.Helper()
	c := &taskCatalog{Format: catalogFormat, Profile: "offline-task-binding-v1", Tasks: []taskReference{}}
	seen := map[string]bool{}
	for _, pair := range p.Pairs {
		if seen[pair.TaskID] {
			continue
		}
		seen[pair.TaskID] = true
		raw, err := os.ReadFile(filepath.Join(root, pair.Task))
		if err != nil {
			t.Fatal(err)
		}
		name := "reviewed-" + pair.TaskID + ".json"
		if os.WriteFile(filepath.Join(root, name), raw, 0600) != nil {
			t.Fatal("write")
		}
		c.Tasks = append(c.Tasks, taskReference{pair.TaskID, name})
	}
	writeTestJSON(t, root, "catalog.json", c)
	writeTestJSON(t, root, "suite.json", p)
	expectationFor(t, root, p)
	return c
}
func TestReviewedTaskCoherentReplacement(t *testing.T) {
	ctx := context.Background()
	root, p := suiteFixture(t, "close-issue", "close-issue")
	catalogFixture(t, root, p)
	before := suiteSnapshot(t, root)
	review, err := ReviewSuite(ctx, root, "suite.json", "suite", "expected.json", "catalog.json")
	if err != nil || review.Decision != "matched" || review.CheckedPairs != 2 || review.ExecutionPerformed {
		t.Fatal(review, err)
	}
	if !reflect.DeepEqual(before, suiteSnapshot(t, root)) {
		t.Fatal("review wrote inputs")
	}
	raw, _ := os.ReadFile(filepath.Join(root, p.Pairs[0].Task))
	ta, err := task.Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	ta.Oracle[0].Expr = "true"
	writeTestJSON(t, root, p.Pairs[0].Task, ta)
	review, err = ReviewSuite(ctx, root, "suite.json", "suite", "expected.json", "catalog.json")
	if err != nil || review.Decision != "mismatched" || review.CheckedPairs != 2 || !same(review.Tasks[0].Differences, []string{"oracle"}) {
		t.Fatal(review, err)
	}
	runAssessmentFixture(t, root, p)
	before = suiteSnapshot(t, root)
	r, err := AssessReviewedSuite(ctx, root, "suite", "expected.json", "catalog.json", "both-pass-v1")
	if err != nil || r.Decision != "failed" || r.Assessment.Decision != "passed" || r.TaskBinding.Status != "mismatched" || r.TaskBinding.MismatchedTrials != 4 || r.TaskBinding.VerifiedTrials != 4 {
		t.Fatal(r, err)
	}
	old, err := AssessSuite(ctx, root, "suite", "expected.json", "both-pass-v1")
	if err != nil || !same(old, r.Assessment) {
		t.Fatal("old assessment changed", err)
	}
	if !reflect.DeepEqual(before, suiteSnapshot(t, root)) {
		t.Fatal("assessment wrote evidence")
	}
	// Match the chosen Task on a subsequent independent call, then remove an
	// observed sample without erasing the expected denominator.
	writeTestJSON(t, root, "reviewed-close-issue.json", ta)
	r, err = AssessReviewedSuite(ctx, root, "suite", "expected.json", "catalog.json", "candidate-pass-v1")
	if err != nil || r.Decision != "passed" || r.TaskBinding.MatchedTrials != 4 {
		t.Fatal(r, err)
	}
	removeAudit(t, root, "report.json")
	removeAudit(t, root, "baseline-01/terminal.json")
	r, err = AssessReviewedSuite(ctx, root, "suite", "expected.json", "catalog.json", "candidate-pass-v1")
	if err != nil || r.Decision != "failed" || r.TaskBinding.UnavailableTrials != 1 || r.TaskBinding.PlannedTrials != 4 {
		t.Fatal(r, err)
	}
}
func TestCatalogAdmissionAndTaskEquality(t *testing.T) {
	ctx := context.Background()
	root, p := suiteFixture(t, "close-issue")
	c := catalogFixture(t, root, p)
	original, _ := os.ReadFile(filepath.Join(root, c.Tasks[0].Task))
	valid, err := decodeCatalogTask(original)
	if err != nil {
		t.Fatal(err)
	}
	var object map[string]any
	json.Unmarshal(original, &object)
	object["context"] = ""
	object["faultTool"] = ""
	whitespace, _ := json.MarshalIndent(object, "", " ")
	if v, err := decodeCatalogTask(whitespace); err != nil || !same(v, valid) {
		t.Fatal("equivalent spelling", err)
	}
	for _, bad := range []string{strings.Replace(string(original), `"modelRequests":16`, `"modelRequests":16.0`, 1), strings.Replace(string(original), `"revision":`, `"Revision":`, 1), strings.Replace(string(original), `"revision":"v1"`, `"revision":null`, 1), strings.Replace(string(original), `"revision":`, `"extra":true,"revision":`, 1), strings.Replace(string(original), `"revision":`, `"revision":"v1","revision":`, 1)} {
		if _, err := decodeCatalogTask([]byte(bad)); err == nil {
			t.Fatal("bad catalog Task admitted", bad)
		}
	}
	changed := *valid
	changed.Objective = "Different objective"
	changed.Revision = "v2"
	changed.Tools = append([]string(nil), valid.Tools...)
	changed.Tools[0], changed.Tools[1] = changed.Tools[1], changed.Tools[0]
	if !same(taskDifferences(valid, &changed), []string{"identity", "goal", "tools"}) {
		t.Fatal(taskDifferences(valid, &changed))
	}
	for _, name := range []string{"suite/catalog.json", "SUITE/catalog.json", "../catalog.json"} {
		if _, err := loadCatalog(ctx, root, name, "suite"); err == nil {
			t.Fatal("unsafe path")
		}
	}
	c.Tasks[0].Task = p.Pairs[0].Task
	writeTestJSON(t, root, "catalog.json", c)
	if _, err := ReviewSuite(ctx, root, "suite.json", "suite", "expected.json", "catalog.json"); err == nil {
		t.Fatal("self-reference")
	}
	ctx, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := ReviewSuite(ctx, root, "suite.json", "suite", "expected.json", "catalog.json"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
func TestReviewPlanCoverageAndFrozenInputs(t *testing.T) {
	ctx := context.Background()
	root, p := suiteFixture(t, "close-issue", "read-issue")
	catalogFixture(t, root, p)
	e := expectationFor(t, root, p)
	c, err := loadCatalog(ctx, root, "catalog.json", "suite")
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := PrepareSuite(ctx, root, suiteBytes(t, p))
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(root, p.Pairs[0].Task), []byte("invalid changed input"), 0600)
	r, err := reviewPrepared(ctx, e, c, prepared, 1024)
	if err != nil || r.Decision != "matched" {
		t.Fatal(r, err)
	}
	delete(c.tasks, "close-issue")
	r, err = reviewPrepared(ctx, e, c, prepared, 1025)
	if err != nil || r.CheckedPairs != 1 || r.CatalogStatus != "mismatched" || r.PlanStatus != "mismatched" {
		t.Fatal(r, err)
	}
	e.Plan.CandidateModel = "mock-other"
	r, err = reviewPrepared(ctx, e, c, prepared, 1024)
	if err != nil || r.CheckedPairs != 0 || r.PlannedPairs != 2 {
		t.Fatal(r, err)
	}
}

func TestSuiteAuditVerifiesEachStableTrialOnce(t *testing.T) {
	ctx := context.Background()
	root, p := suiteFixture(t, "close-issue", "read-issue")
	runAssessmentFixture(t, root, p)
	fs, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer fs.Close()
	calls := map[string]int{}
	r, err := inspectSuiteViewVerified(ctx, diskReadRoot{fs}, "suite", maxSuiteWriteBytes, nil, nil, func(ctx context.Context, e *AgentEvidence, terminal bool) error {
		calls[e.Episode.Definition.Config.TrialID]++
		return replay(ctx, e, terminal)
	})
	if err != nil || r.State != "published" || len(calls) != 4 {
		t.Fatal(r, calls, err)
	}
	for id, n := range calls {
		if n != 1 {
			t.Fatal("duplicate replay", id, n)
		}
	}
}
