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
