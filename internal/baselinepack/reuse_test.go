package baselinepack

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/augety121/mcp-state-twin/internal/bundle"
	"github.com/augety121/mcp-state-twin/internal/task"
	"github.com/augety121/mcp-state-twin/internal/testfixture"
)

func singleEntryPack(t *testing.T) (string, Pack) {
	t.Helper()
	root := testfixture.Baseline(t)
	raw, err := os.ReadFile(filepath.Join(root, "baseline-pack.json"))
	if err != nil {
		t.Fatal(err)
	}
	var pack Pack
	if err = json.Unmarshal(raw, &pack); err != nil {
		t.Fatal(err)
	}
	pack.Entries = pack.Entries[:1]
	writePackJSON(t, root, "baseline-pack.json", pack)
	return root, pack
}

func writePackJSON(t *testing.T, root, name string, value any) {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(root, name), raw, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestPackTaskReuseEquivalentAdmission(t *testing.T) {
	root, pack := singleEntryPack(t)
	ctx := context.Background()
	p, err := Prepare(ctx, root, "baseline-pack.json", "plugin-profile.json")
	if err != nil {
		t.Fatal(err)
	}
	e := pack.Entries[0]
	actual := filepath.ToSlash(filepath.Join(e.Root, e.Task))
	reviewed := filepath.ToSlash(filepath.Join(e.Root, e.ReviewedTask))
	if !bytes.Equal(p.Reader.Inputs[actual].Raw, p.Reader.Inputs[reviewed].Raw) {
		t.Fatal("fixture must exercise exact-byte Task reuse")
	}
	// Exact copies remain separate inputs in the operation's byte budget.
	total := 0
	for _, in := range p.Reader.Inputs {
		total += len(in.Raw)
	}
	if p.Reader.Left != InputLimit-total {
		t.Fatal("input debit did not include all frozen paths")
	}
	for _, budget := range []int{total, total - 1} {
		r := NewReader(ctx, root)
		r.Left = budget
		_, err := PrepareWith(r, "baseline-pack.json", "plugin-profile.json")
		if (err == nil) != (budget == total) {
			t.Fatalf("budget %d, required %d: %v", budget, total, err)
		}
	}
	// The slower comparison still accepts a reordered equivalent definition.
	var object map[string]json.RawMessage
	if err = json.Unmarshal(p.Reader.Inputs[reviewed].Raw, &object); err != nil {
		t.Fatal(err)
	}
	writePackJSON(t, root, reviewed, object)
	q, err := Prepare(ctx, root, "baseline-pack.json", "plugin-profile.json")
	if err != nil {
		t.Fatal("equivalent reviewed Task was rejected", err)
	}
	if bytes.Equal(q.Reader.Inputs[actual].Raw, q.Reader.Inputs[reviewed].Raw) || !Equal(p.Entries[0].Task, q.Entries[0].Task) {
		t.Fatal("reformatted review did not preserve semantic admission")
	}
	// Equal bytes cannot stand in for decoding a valid actual Task first.
	for _, name := range []string{actual, reviewed} {
		if err = os.WriteFile(filepath.Join(root, name), []byte(`{broken`), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := Prepare(ctx, root, "baseline-pack.json", "plugin-profile.json"); err == nil {
		t.Fatal("identically malformed actual/reviewed Task was accepted")
	}
}

func TestCaseCandidateRequiresFullDefinition(t *testing.T) {
	root, pack := singleEntryPack(t)
	e := &pack.Entries[0]
	raw, err := os.ReadFile(filepath.Join(root, e.Root, e.Task))
	if err != nil {
		t.Fatal(err)
	}
	selected, err := task.Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	selected.Objective = "A different objective under the same Task ID"
	// Keep the case manifest's original Task intact; its same-ID candidate
	// must not authorize these separate matching actual/reviewed definitions.
	e.Task, e.ReviewedTask = "selected-task.json", "selected-review.json"
	writePackJSON(t, root, filepath.Join(e.Root, e.Task), selected)
	writePackJSON(t, root, filepath.Join(e.Root, e.ReviewedTask), selected)
	writePackJSON(t, root, "baseline-pack.json", pack)
	if _, err = Prepare(context.Background(), root, "baseline-pack.json", "plugin-profile.json"); err == nil || err.Error() != "PLUGIN_QUALITY_REFERENCE_INVALID" {
		t.Fatalf("same-ID different definition was not rejected by case binding: %v", err)
	}
}

func TestSameWorldIdentityDoesNotReplaceContentComparison(t *testing.T) {
	a := &bundle.Artifact{Files: map[string][]byte{"fixture.json": []byte(`{"value":1}`)}}
	b := &bundle.Artifact{Files: map[string][]byte{"fixture.json": []byte(`{"value":1}`)}}
	if SameWorld(nil, nil) || SameWorld(a, nil) || SameWorld(nil, a) {
		t.Fatal("nil world was considered equal")
	}
	if !SameWorld(a, a) || !SameWorld(a, b) {
		t.Fatal("identical or independently equal world was rejected")
	}
	b.Files["fixture.json"] = []byte(`{"value":2}`)
	if SameWorld(a, b) {
		t.Fatal("different world members were considered equal")
	}
	b.Files["fixture.json"] = bytes.Clone(a.Files["fixture.json"])
	b.Manifest.Fixture = "different.json"
	if SameWorld(a, b) {
		t.Fatal("different world manifests were considered equal")
	}
}
