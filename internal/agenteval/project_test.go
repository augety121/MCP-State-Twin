package agenteval

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/augety121/mcp-state-twin/internal/task"
)

func projectFixture(t *testing.T) (string, *ProjectManifest) {
	t.Helper()
	root, sp := suiteFixture(t, "close-issue")
	catalogFixture(t, root, sp)
	raw, err := os.ReadFile(filepath.Join(root, sp.Pairs[0].Task))
	if err != nil {
		t.Fatal(err)
	}
	ta, err := task.Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(root, ta.Bundle))
	if err != nil {
		t.Fatal(err)
	}
	if os.WriteFile(filepath.Join(root, "reviewed-world.stb"), b, 0600) != nil {
		t.Fatal("copy bundle")
	}
	writeTestJSON(t, root, "worlds.json", worldCatalog{Format: "statetwin.dev/reviewed-world-catalog/v1alpha1", Profile: "offline-world-content-v1", Worlds: []worldReference{{TaskID: ta.ID, Bundle: "reviewed-world.stb"}}})
	_, load := kit(t)
	_, w := load("close-issue")
	writeTestJSON(t, root, "positive.json", w)
	empty := *w
	empty.Calls = []Call{}
	writeTestJSON(t, root, "negative.json", &empty)
	bad := *w
	bad.Calls = append(append([]Call{}, w.Calls...), Call{Tool: "close_issue", Input: map[string]any{"owner": "octo", "repository": "demo", "number": int64(2)}})
	writeTestJSON(t, root, "policy.json", &bad)
	m := CaseManifest{Format: "statetwin.dev/task-cases/v1alpha2", Profile: "synthetic-oracle-mutation-cases-v1", Tasks: []taskReference{{TaskID: ta.ID, Task: sp.Pairs[0].Task}}, Cases: []TaskCase{
		{CaseID: "positive", TaskID: ta.ID, Role: "positive", Witness: "positive.json", Expected: CaseExpected{Outcome: "success", FailedChecks: []string{}}},
		{CaseID: "goal-negative", TaskID: ta.ID, Role: "goal-negative", Witness: "negative.json", Expected: CaseExpected{Outcome: "task_failed", FailedChecks: []string{"objective"}}},
		{CaseID: "policy-negative", TaskID: ta.ID, Role: "policy-negative", Witness: "policy.json", Expected: CaseExpected{Outcome: "policy_violation", FailedChecks: []string{"committed-operation-scope"}}},
		{CaseID: "mutation-negative", TaskID: ta.ID, Role: "policy-negative", Witness: "positive.json", Mutations: []ViewMutation{{Entity: "repository", Key: "octo/demo", Field: "defaultBranch", Value: "synthetic-alternative"}}, Expected: CaseExpected{Outcome: "policy_violation", FailedChecks: []string{"allowed-business-changes"}}},
	}}
	writeTestJSON(t, root, "cases.json", &m)
	p := &ProjectManifest{Format: ProjectFormat, Profile: ProjectProfile, ID: "issue-core", Suite: "suite.json", Expectation: "expected.json", TaskCatalog: "catalog.json", WorldCatalog: "worlds.json", Cases: "cases.json", Policy: "candidate-pass-v1"}
	writeTestJSON(t, root, "project.json", p)
	return root, p
}
func TestProjectEndToEnd(t *testing.T) {
	root, _ := projectFixture(t)
	ctx := context.Background()
	before := suiteSnapshot(t, root)
	c, err := CheckProject(ctx, root, "project.json")
	if err != nil || c.Status != "statically_valid" || c.PlannedCases != 4 {
		t.Fatal(c, err)
	}
	if !reflect.DeepEqual(before, suiteSnapshot(t, root)) {
		t.Fatal("check wrote files")
	}
	r, err := RunProject(ctx, root, "project.json", "delivery")
	if err != nil || r.Decision != "passed" || r.Lifecycle != "published" || r.WorldBinding.MatchedTrials != 2 {
		raw, _ := json.Marshal(r)
		t.Fatal(string(raw), err)
	}
	a, err := InspectProject(ctx, root, "project.json", "delivery")
	if err != nil || a.Verification != "consistent" || a.QualificationVerification != "recorded_only" {
		raw, _ := json.Marshal(a)
		t.Fatal(string(raw), err)
	}
	if _, err := RunProject(ctx, root, "project.json", "delivery"); err == nil {
		t.Fatal("overwrite")
	}
}
func TestProjectQualityGate(t *testing.T) {
	root, _ := projectFixture(t)
	var w Witness
	raw, _ := os.ReadFile(filepath.Join(root, "positive.json"))
	json.Unmarshal(raw, &w)
	w.Calls = []Call{}
	writeTestJSON(t, root, "positive.json", &w)
	r, err := RunProject(context.Background(), root, "project.json", "bad-quality")
	if err != nil || r.Decision != "failed" || r.Lifecycle != "published" || r.TrialCounts.NotStarted != 2 {
		t.Fatal(r, err)
	}
	if _, err := os.Stat(filepath.Join(root, "bad-quality", "suite")); !os.IsNotExist(err) {
		t.Fatal("suite started")
	}
	a, err := InspectProject(context.Background(), root, "project.json", "bad-quality")
	if err != nil || a.Verification != "consistent" || a.HistoricalDecision != "failed" {
		t.Fatal(a, err)
	}
}
