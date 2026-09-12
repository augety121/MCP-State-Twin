package agenteval

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTestJSON(t *testing.T, root, name string, value any) {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(root, filepath.FromSlash(name)), raw, 0600); err != nil {
		t.Fatal(err)
	}
}
func TestInspectPublishedIsReadOnlyAndNotTaskPass(t *testing.T) {
	root, e := record(t)
	before, err := os.ReadFile(filepath.Join(root, "trial", "terminal.json"))
	if err != nil {
		t.Fatal(err)
	}
	r, err := InspectDirectory(context.Background(), root, "trial")
	if err != nil {
		t.Fatal(err)
	}
	if r.State != "published" || !r.EvidenceComplete || r.ResumeAllowed || r.SnapshotAtomic || r.ProviderProvenance != "not-proven" || r.TrialID != e.Episode.Definition.Config.TrialID {
		t.Fatalf("%+v", r)
	}
	after, _ := os.ReadFile(filepath.Join(root, "trial", "terminal.json"))
	if string(before) != string(after) {
		t.Fatal("inspection modified evidence")
	}
	files, err := os.ReadDir(filepath.Join(root, "trial"))
	if err != nil || len(files) != 2 {
		t.Fatal("inspection changed directory")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err = InspectDirectory(ctx, root, "trial"); err == nil {
		t.Fatal("canceled inspection completed")
	}
}

func TestInspectCanceledBeforeFilesystemAdmission(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	root := t.TempDir()
	for _, candidate := range []string{root, filepath.Join(root, "missing-root")} {
		if r, err := InspectDirectory(ctx, candidate, "not-started"); r != nil || err == nil || err.Error() != "EVIDENCE_INSPECT_CANCELED_OR_TIMED_OUT" {
			t.Fatalf("cancellation became a filesystem diagnostic: %+v %v", r, err)
		}
	}
}

func TestInspectLifecycleAndCorruption(t *testing.T) {
	_, original := record(t)
	data, _ := json.Marshal(original)
	for _, kind := range []string{"absent", "empty", "claim", "closure", "pending", "partial", "missing-claim", "wrong-claim", "corrupt", "conflict", "duplicate", "unknown-member", "unsafe-status"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			e, err := DecodeEvidence(data)
			if err != nil {
				t.Fatal(err)
			}
			if kind != "absent" {
				if err = os.Mkdir(filepath.Join(root, "trial"), 0700); err != nil {
					t.Fatal(err)
				}
			}
			if kind != "absent" && kind != "empty" && kind != "missing-claim" {
				writeTestJSON(t, root, "trial/claim.json", e.Episode.Definition.Config)
			}
			switch kind {
			case "closure":
				e.Episode.CleanupStatus = "pending"
				e.Episode.EvidenceStatus = "partial"
				e.Episode.WorldReplayable = false
				writeTestJSON(t, root, "trial/closure.json", e)
			case "pending":
				writeTestJSON(t, root, "trial/terminal.pending.json", e)
			case "partial":
				e.Episode.ExecutionStatus = "budget_exhausted"
				e.Episode.FailureCode = "BUDGET_EXHAUSTED"
				e.Episode.EvidenceStatus = "partial"
				e.Episode.WorldReplayable = false
				writeTestJSON(t, root, "trial/terminal.json", e)
			case "missing-claim":
				writeTestJSON(t, root, "trial/terminal.json", e)
			case "wrong-claim":
				c := e.Episode.Definition.Config
				c.TrialID = "other-trial"
				writeTestJSON(t, root, "trial/claim.json", c)
				writeTestJSON(t, root, "trial/terminal.json", e)
			case "corrupt":
				if err = os.WriteFile(filepath.Join(root, "trial", "terminal.json"), []byte(`{"broken":`), 0600); err != nil {
					t.Fatal(err)
				}
			case "conflict":
				writeTestJSON(t, root, "trial/terminal.json", e)
				e.Episode.View.After.Sequences["comment_id"]++
				writeTestJSON(t, root, "trial/terminal.pending.json", e)
			case "duplicate":
				writeTestJSON(t, root, "trial/terminal.json", e)
				writeTestJSON(t, root, "trial/terminal.pending.json", e)
			case "unknown-member":
				if err = os.WriteFile(filepath.Join(root, "trial", "private-sensitive-filename"), nil, 0600); err != nil {
					t.Fatal(err)
				}
			case "unsafe-status":
				e.Episode.ExecutionStatus = "private-status-sentinel"
				writeTestJSON(t, root, "trial/terminal.json", e)
			}
			r, err := InspectDirectory(context.Background(), root, "trial")
			if err != nil {
				t.Fatal(err)
			}
			want := "invalid"
			switch kind {
			case "absent":
				want = "not_started"
			case "empty", "claim", "closure", "pending", "partial":
				want = "incomplete_or_running"
			case "duplicate":
				want = "published_with_residue"
			}
			if r.State != want || r.EvidenceComplete != (kind == "duplicate") || r.ResumeAllowed {
				t.Fatalf("%s: %+v", kind, r)
			}
			encoded, _ := json.Marshal(r)
			if strings.Contains(string(encoded), "private-") {
				t.Fatal("diagnostic echoed private metadata")
			}
		})
	}
}

func TestInspectLocalAPIContractArtifactsAndBinding(t *testing.T) {
	p, raw, w := livePlan(t, "close-issue")
	root := liveRoot(t)
	posts := 0
	closed := false
	if _, err := recordLive(context.Background(), root, p, raw, true, "contract-test", fakeClient(liveScript(w), 0, &posts, &closed)); err != nil {
		t.Fatal(err)
	}
	r, err := InspectDirectory(context.Background(), root, p.OutputDirectory())
	if err != nil {
		t.Fatal(err)
	}
	if !r.EvidenceComplete || r.Lane != "local-api" || r.State != "published" {
		t.Fatalf("%+v", r)
	}
	p.MaxOutputTokens++ // valid plan shape, wrong approval content
	writeTestJSON(t, root, p.OutputDirectory()+"/claim.json", p)
	r, err = InspectDirectory(context.Background(), root, p.OutputDirectory())
	if err != nil || r.State != "invalid" || r.EvidenceComplete {
		t.Fatalf("%+v %v", r, err)
	}
}

func TestInspectPrivateOversizedAndSymlinkMembers(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "trial"), 0700); err != nil {
		t.Fatal(err)
	}
	writeTestJSON(t, root, "trial/claim.json", mockConfig("trial"))
	if err := os.WriteFile(filepath.Join(root, "trial", "closure.json"), []byte(`{"note":"api_key=synthetic-private-sentinel"}`), 0600); err != nil {
		t.Fatal(err)
	}
	r, err := InspectDirectory(context.Background(), root, "trial")
	if err != nil || r.State != "invalid" {
		t.Fatal("private content admitted")
	}
	encoded, _ := json.Marshal(r)
	if strings.Contains(string(encoded), "synthetic-private-sentinel") {
		t.Fatal("private content echoed")
	}
	// Only the owned test leaf is replaced. A sparse file tests admission before
	// reading/allocating its advertised size; it does not fill the real disk.
	f, err := os.OpenFile(filepath.Join(root, "trial", "closure.json"), os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if err = f.Truncate((32 << 20) + 1); err != nil {
		t.Fatal(err)
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}
	r, err = InspectDirectory(context.Background(), root, "trial")
	if err != nil || r.State != "invalid" {
		t.Fatal("oversized member admitted")
	}
	for _, name := range []string{"../outside", "/absolute", `..\outside`} {
		if _, err = InspectDirectory(context.Background(), root, name); err == nil {
			t.Fatal("path escaped")
		}
	}
	outside := t.TempDir()
	if err = os.Symlink(outside, filepath.Join(root, "linked")); err != nil {
		t.Log("symlink creation unavailable on this platform")
		return
	}
	if _, err = InspectDirectory(context.Background(), root, "linked"); err == nil {
		t.Fatal("symlink directory followed")
	}
}
