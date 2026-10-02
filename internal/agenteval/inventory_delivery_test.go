package agenteval

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestInventoryAndRetentionBoundaries(t *testing.T) {
	ctx := context.Background()
	root, p := suiteFixture(t, "close-issue")
	runAssessmentFixture(t, root, p)
	reg := &SuiteRegistry{Format: "statetwin.dev/suite-registry/v1alpha1", Entries: []RegistryEntry{{"complete", "suite"}, {"missing", "missing"}}}
	writeTestJSON(t, root, "registry.json", reg)
	policy := RetentionPolicy{Format: "statetwin.dev/retention-intent/v1alpha1", Entries: []RetentionIntent{{ID: "complete", Intent: "review", References: []string{}}, {ID: "missing", Intent: "review", References: []string{"complete"}}}}
	writeTestJSON(t, root, "retention.json", policy)
	before := suiteSnapshot(t, root)
	inv, err := InventorySuites(ctx, root, "registry.json", "metadata")
	if err != nil || inv.Completion != "complete" || inv.Entries[0].ReportVerification != "not_checked" || inv.Entries[0].AuditState != "" || inv.Entries[1].ObservationState != "missing" {
		t.Fatal(inv, err)
	}
	replay, err := InventorySuites(ctx, root, "registry.json", "replay")
	if err != nil || replay.Entries[0].AuditState != "published" || replay.Entries[0].ReportVerification != "matched" {
		t.Fatal(replay, err)
	}
	retention, err := PreviewRetention(ctx, root, "registry.json", "retention.json")
	if err != nil || retention.Entries[0].Status != "protected" || retention.Entries[1].Status != "unknown" || retention.DeletionPerformed {
		t.Fatal(retention, err)
	}
	if !reflect.DeepEqual(before, suiteSnapshot(t, root)) {
		t.Fatal("read-only command modified source")
	}
	reg.Entries[1].Out = "suite/child"
	writeTestJSON(t, root, "bad.json", reg)
	if _, err := loadRegistry(root, "bad.json"); err == nil {
		t.Fatal("nested registry")
	}
	policy.Entries[0].References = []string{"not-registered"}
	writeTestJSON(t, root, "bad-policy.json", policy)
	if _, err := PreviewRetention(ctx, root, "registry.json", "bad-policy.json"); err == nil {
		t.Fatal("dangling reference")
	}
	os.WriteFile(filepath.Join(root, "suite", "claim.json"), []byte("broken"), 0600)
	inv, err = InventorySuites(ctx, root, "registry.json", "metadata")
	if err != nil || inv.Entries[0].ObservationState != "invalid" {
		t.Fatal(inv, err)
	}
}
func TestRetentionCyclesAndBudgetedView(t *testing.T) {
	reg := &SuiteRegistry{Entries: []RegistryEntry{{ID: "a"}, {ID: "b"}, {ID: "c"}}}
	p := &RetentionPolicy{Entries: []RetentionIntent{{ID: "a", Intent: "review", References: []string{"b"}}, {ID: "b", Intent: "review", References: []string{"a"}}, {ID: "c", Intent: "keep", References: []string{"a"}}}}
	inv := &SuiteInventory{Completion: "complete", Entries: []InventoryEntry{}}
	for i := 0; i < 3; i++ {
		inv.Entries = append(inv.Entries, InventoryEntry{ObservationState: "observed", Problem: "metadata_present", SizeComplete: true})
	}
	r := retentionResult(reg, p, inv)
	if r.Entries[0].Status != "protected" || r.Entries[1].Status != "protected" || !same(r.DependencyGroups, [][]string{{"a", "b"}, {"c"}}) {
		t.Fatal(r)
	}
	p.Entries[2].Intent = "review"
	r = retentionResult(reg, p, inv)
	if r.Entries[0].Status != "review_candidate" {
		t.Fatal(r)
	}
	budget := &inventoryBudget{bytes: 1, entries: 1}
	view := budgetReadRoot{frozenFiles{"one": []byte("123"), "two": []byte("4")}, budget}
	f, _ := view.Open("one")
	if _, err := f.Read(make([]byte, 4)); err == nil || !budget.exhausted {
		t.Fatal("byte budget bypass")
	}
	budget = &inventoryBudget{bytes: 100, entries: 1}
	view.budget = budget
	f, _ = view.Open(".")
	if _, err := f.ReadDir(5); err == nil || !budget.exhausted {
		t.Fatal("entry budget bypass")
	}
}
