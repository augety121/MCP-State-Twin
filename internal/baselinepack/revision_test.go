package baselinepack

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/augety121/mcp-state-twin/internal/testfixture"
)

func TestBaselineRevisionLifecycle(t *testing.T) {
	root := testfixture.Baseline(t)
	ctx := context.Background()
	r, e := Register(ctx, root, "baseline-pack.json", "plugin-profile.json", "registry")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = Register(ctx, root, "baseline-pack.json", "plugin-profile.json", "registry"); e != nil {
		t.Fatal("idempotency", e)
	}
	raw, _ := os.ReadFile(filepath.Join(root, "baseline-pack.json"))
	var pack Pack
	json.Unmarshal(raw, &pack)
	pack.Entries[0].Family = "changed-family"
	raw, _ = json.Marshal(pack)
	os.WriteFile(filepath.Join(root, "baseline-pack.json"), raw, 0600)
	if _, e = Register(ctx, root, "baseline-pack.json", "plugin-profile.json", "registry"); e == nil {
		t.Fatal("same revision changed")
	}
	event := RevisionEvent{Format: "statetwin.dev/baseline-pack-event/v1alpha1", State: "deprecated", At: "2026-10-02T00:00:00Z", Reason: "replacement", Replacement: "v2", PreviewVersions: []string{"preview-one"}}
	if Transition(root, "registry", r.PackID, r.Revision, event) == nil {
		t.Fatal("deprecation window bypass")
	}
	event.PreviewVersions = append(event.PreviewVersions, "preview-two")
	if e = Transition(root, "registry", r.PackID, r.Revision, event); e != nil {
		t.Fatal(e)
	}
	if state, e := RevisionState(root, "registry", r.PackID, r.Revision); e != nil || state != "deprecated" {
		t.Fatal(state, e)
	}
	event.State = "revoked"
	event.Reason = "urgent regression"
	if e = Transition(root, "registry", r.PackID, r.Revision, event); e != nil {
		t.Fatal(e)
	}
	if state, e := RevisionState(root, "registry", r.PackID, r.Revision); e != nil || state != "revoked" {
		t.Fatal(state, e)
	}
	if _, e = os.Stat(filepath.Join(root, "registry", r.PackID, r.Revision, "revision.json")); e != nil {
		t.Fatal("history deleted")
	}
}

func TestRevisionStateRejectsInvalidSnapshot(t *testing.T) {
	root := testfixture.Baseline(t)
	packPath := filepath.Join(root, "baseline-pack.json")
	raw, err := os.ReadFile(packPath)
	if err != nil {
		t.Fatal(err)
	}
	var pack Pack
	if err = json.Unmarshal(raw, &pack); err != nil {
		t.Fatal(err)
	}
	pack.Entries = pack.Entries[:1]
	raw, err = json.Marshal(pack)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(packPath, raw, 0600); err != nil {
		t.Fatal(err)
	}
	r, err := Register(context.Background(), root, "baseline-pack.json", "plugin-profile.json", "registry")
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "registry", r.PackID, r.Revision)
	revisionPath := filepath.Join(dir, "revision.json")
	event := RevisionEvent{Format: "statetwin.dev/baseline-pack-event/v1alpha1", State: "deprecated", At: "2026-10-02T00:00:00Z", Reason: "replacement", Replacement: "v2", PreviewVersions: []string{"preview-one", "preview-two"}}
	for name, change := range map[string]func(*Revision){
		"format":              func(r *Revision) { r.Format = "unrecognized" },
		"pack-id":             func(r *Revision) { r.PackID = "another-pack" },
		"revision":            func(r *Revision) { r.Revision = "another-revision" },
		"profile":             func(r *Revision) { r.Profile.ProtocolProfile = "unsupported" },
		"review-independence": func(r *Revision) { r.ReviewIndependence = "independent" },
		"missing-task": func(r *Revision) {
			delete(r.Inputs, pack.Entries[0].Root+"/"+pack.Entries[0].Task)
		},
		"invalid-task": func(r *Revision) {
			r.Inputs[pack.Entries[0].Root+"/"+pack.Entries[0].Task] = []byte(`{}`)
		},
		"extra-input":    func(r *Revision) { r.Inputs["unused.json"] = []byte(`{}`) },
		"unsafe-input":   func(r *Revision) { r.Inputs["../outside"] = []byte("fixture") },
		"missing-inputs": func(r *Revision) { r.Inputs = map[string][]byte{} },
	} {
		t.Run(name, func(t *testing.T) {
			invalid := Clone(*r)
			change(&invalid)
			raw, err := json.Marshal(invalid)
			if err != nil {
				t.Fatal(err)
			}
			if err = os.WriteFile(revisionPath, raw, 0600); err != nil {
				t.Fatal(err)
			}
			if state, err := RevisionState(root, "registry", r.PackID, r.Revision); err == nil || state != "" {
				t.Fatalf("invalid snapshot reported state=%q err=%v", state, err)
			}
			if err := Transition(root, "registry", r.PackID, r.Revision, event); err == nil {
				t.Fatal("transition accepted invalid snapshot")
			}
			if _, err := os.Stat(filepath.Join(dir, "deprecated.json")); !os.IsNotExist(err) {
				t.Fatalf("invalid transition created event: %v", err)
			}
		})
	}
	for _, invalid := range []string{`{}`, `not-json`} {
		if err := os.WriteFile(revisionPath, []byte(invalid), 0600); err != nil {
			t.Fatal(err)
		}
		if state, err := RevisionState(root, "registry", r.PackID, r.Revision); err == nil || state != "" {
			t.Fatalf("invalid document reported state=%q err=%v", state, err)
		}
	}
	// Registry inspection uses the recorded bytes even after original sources
	// change. The missing-task case above also proves no live-file fallback.
	raw, err = json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(revisionPath, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(packPath, []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	if state, err := RevisionState(root, "registry", r.PackID, r.Revision); err != nil || state != "frozen" {
		t.Fatalf("frozen source was reread: state=%q err=%v", state, err)
	}
}
