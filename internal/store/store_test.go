package store

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/augety121/mcp-state-twin/internal/world"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestFailedOutcomeDoesNotCommitState(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	initial := world.New()
	initial.Entities["item"] = map[string]map[string]any{"a": {"id": "a"}}
	if err := s.InitializeBranch(ctx, "main", "sha256:spec", initial, time.Unix(0, 0)); err != nil {
		t.Fatal(err)
	}
	result, err := s.ApplyCall(ctx, "main", "sha256:spec", "bad_call", map[string]any{}, func(state *world.State, _ time.Time, _ int64) (CallOutcome, error) {
		state.Entities["item"]["b"] = map[string]any{"id": "b"}
		return CallOutcome{Result: map[string]any{"error": "no"}, ErrorClass: "PRECONDITION_FAILED", CommitState: false}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.BeforeDigest != result.AfterDigest {
		t.Fatal("failed transition changed state digest")
	}
	branch, err := s.Branch(ctx, "main")
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := branch.State.Entities["item"]["b"]; exists {
		t.Fatal("failed transition leaked state")
	}
	if branch.CallCount != 1 {
		t.Fatalf("call count = %d, want 1", branch.CallCount)
	}
}

func TestSnapshotForkIsolationAndDiff(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	initial := world.New()
	initial.Entities["item"] = map[string]map[string]any{"a": {"id": "a", "state": "open"}}
	if err := s.InitializeBranch(ctx, "main", "sha256:spec", initial, time.Unix(0, 0)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateSnapshot(ctx, "base", "main"); err != nil {
		t.Fatal(err)
	}
	if err := s.Fork(ctx, "base", "run-a"); err != nil {
		t.Fatal(err)
	}
	if err := s.Fork(ctx, "base", "run-b"); err != nil {
		t.Fatal(err)
	}
	_, err := s.ApplyCall(ctx, "run-a", "sha256:spec", "close", map[string]any{}, func(state *world.State, _ time.Time, _ int64) (CallOutcome, error) {
		state.Entities["item"]["a"]["state"] = "closed"
		return CallOutcome{Result: map[string]any{"ok": true}, CommitState: true}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.Branch(ctx, "run-b")
	if err != nil {
		t.Fatal(err)
	}
	if b.State.Entities["item"]["a"]["state"] != "open" {
		t.Fatal("fork mutation leaked into sibling")
	}
	changes, err := s.DiffBranches(ctx, "run-b", "run-a")
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 1 || changes[0].Path != "/entities/item/a/state" {
		t.Fatalf("unexpected diff: %#v", changes)
	}
	entries, err := s.ControlAudit(ctx)
	if err != nil {
		t.Fatal(err)
	}
	operations := []string{"snapshot.create", "branch.fork", "branch.fork"}
	if len(entries) != len(operations) {
		t.Fatalf("control audit entries = %d, want %d", len(entries), len(operations))
	}
	for i, operation := range operations {
		if entries[i].Operation != operation {
			t.Fatalf("control audit[%d] = %q, want %q", i, entries[i].Operation, operation)
		}
	}
}

func TestResetRestoresSnapshotAndAllowsFurtherCalls(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	initial := world.New()
	initial.Entities["item"] = map[string]map[string]any{"a": {"id": "a", "state": "open"}}
	if err := s.InitializeBranch(ctx, "main", "sha256:spec", initial, time.Unix(0, 0)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateSnapshot(ctx, "base", "main"); err != nil {
		t.Fatal(err)
	}

	closeItem := func(state *world.State, _ time.Time, _ int64) (CallOutcome, error) {
		state.Entities["item"]["a"]["state"] = "closed"
		return CallOutcome{Result: map[string]any{"ok": true}, CommitState: true}, nil
	}
	if _, err := s.ApplyCall(ctx, "main", "sha256:spec", "close", map[string]any{}, closeItem); err != nil {
		t.Fatal(err)
	}
	if err := s.Reset(ctx, "base", "main"); err != nil {
		t.Fatal(err)
	}

	branch, err := s.Branch(ctx, "main")
	if err != nil {
		t.Fatal(err)
	}
	if got := branch.State.Entities["item"]["a"]["state"]; got != "open" {
		t.Fatalf("state after reset = %v, want open", got)
	}
	if branch.CallCount != 0 {
		t.Fatalf("call count after reset = %d, want 0", branch.CallCount)
	}

	// The reset rewinds the branch call index. A later call must still be
	// accepted while the pre-reset audit history remains append-only.
	if _, err := s.ApplyCall(ctx, "main", "sha256:spec", "close", map[string]any{}, closeItem); err != nil {
		t.Fatalf("call after reset: %v", err)
	}
	entries, err := s.ControlAudit(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[1].Operation != "branch.reset" {
		t.Fatalf("unexpected control audit: %#v", entries)
	}
	if entries[1].BeforeDigest == "" || entries[1].AfterDigest == "" {
		t.Fatal("reset audit must bind before and after digests")
	}
}

func TestDatabaseIdentityAndSchemaVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	var gotApplicationID, gotVersion int
	if err := s.db.QueryRow(`PRAGMA application_id`).Scan(&gotApplicationID); err != nil {
		t.Fatal(err)
	}
	if err := s.db.QueryRow(`PRAGMA user_version`).Scan(&gotVersion); err != nil {
		t.Fatal(err)
	}
	if gotApplicationID != applicationID || gotVersion != schemaVersion {
		t.Fatalf("database identity/version = %x/%d, want %x/%d", gotApplicationID, gotVersion, applicationID, schemaVersion)
	}
}

func TestDatabaseRejectsForeignIdentityAndNewerSchema(t *testing.T) {
	for _, test := range []struct {
		name       string
		pragmas    []string
		wantSubstr string
	}{
		{name: "foreign application", pragmas: []string{`PRAGMA application_id = 12345`}, wantSubstr: "does not belong"},
		{name: "newer schema", pragmas: []string{fmt.Sprintf(`PRAGMA application_id = %d`, applicationID), fmt.Sprintf(`PRAGMA user_version = %d`, schemaVersion+1)}, wantSubstr: "newer than supported"},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "state.db")
			db, err := sql.Open("sqlite", path)
			if err != nil {
				t.Fatal(err)
			}
			for _, pragma := range test.pragmas {
				if _, err := db.Exec(pragma); err != nil {
					t.Fatal(err)
				}
			}
			if err := db.Close(); err != nil {
				t.Fatal(err)
			}
			opened, err := Open(path)
			if opened != nil {
				opened.Close()
			}
			if err == nil || !strings.Contains(err.Error(), test.wantSubstr) {
				t.Fatalf("Open error = %v, want substring %q", err, test.wantSubstr)
			}
		})
	}
}

func TestVersionOneSnapshotSchemaMigratesWithoutDataLoss(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(fmt.Sprintf(`PRAGMA application_id = %d`, applicationID)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`PRAGMA user_version = 1`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
CREATE TABLE snapshots (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL UNIQUE,
  spec_digest TEXT NOT NULL,
  state_json BLOB NOT NULL,
  state_digest TEXT NOT NULL,
  clock TEXT NOT NULL,
  created_at TEXT NOT NULL
);
INSERT INTO snapshots(id, name, spec_digest, state_json, state_digest, clock, created_at)
VALUES('legacy-id', 'legacy', 'sha256:spec', '{"entities":{},"sequences":{}}', 'sha256:state', '2026-08-01T00:00:00Z', '2026-08-01T00:00:00Z');`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	legacy, err := s.snapshotByName(context.Background(), "legacy")
	if err != nil {
		t.Fatal(err)
	}
	if legacy.ID != "legacy-id" || legacy.StorageSchemaVersion != 1 {
		t.Fatalf("migrated snapshot = %#v", legacy)
	}
	var gotVersion int
	if err := s.db.QueryRow(`PRAGMA user_version`).Scan(&gotVersion); err != nil {
		t.Fatal(err)
	}
	if gotVersion != schemaVersion {
		t.Fatalf("migrated schema version = %d, want %d", gotVersion, schemaVersion)
	}
}

func TestHundredForksRemainIsolated(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	initial := world.New()
	initial.Entities["item"] = map[string]map[string]any{"a": {"id": "a", "owner": "base"}}
	if err := s.InitializeBranch(ctx, "main", "sha256:spec", initial, time.Unix(0, 0)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateSnapshot(ctx, "base", "main"); err != nil {
		t.Fatal(err)
	}

	const branches = 100
	errorsCh := make(chan error, branches)
	var group sync.WaitGroup
	for i := range branches {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			branchID := fmt.Sprintf("run-%03d", index)
			if err := s.Fork(ctx, "base", branchID); err != nil {
				errorsCh <- fmt.Errorf("fork %s: %w", branchID, err)
				return
			}
			_, err := s.ApplyCall(ctx, branchID, "sha256:spec", "claim", map[string]any{"owner": branchID}, func(state *world.State, _ time.Time, _ int64) (CallOutcome, error) {
				state.Entities["item"]["a"]["owner"] = branchID
				return CallOutcome{Result: map[string]any{"owner": branchID}, CommitState: true}, nil
			})
			if err != nil {
				errorsCh <- fmt.Errorf("mutate %s: %w", branchID, err)
			}
		}(i)
	}
	group.Wait()
	close(errorsCh)
	for err := range errorsCh {
		t.Error(err)
	}
	if t.Failed() {
		return
	}
	for i := range branches {
		branchID := fmt.Sprintf("run-%03d", i)
		branch, err := s.Branch(ctx, branchID)
		if err != nil {
			t.Fatal(err)
		}
		if got := branch.State.Entities["item"]["a"]["owner"]; got != branchID {
			t.Fatalf("branch %s observed owner %v", branchID, got)
		}
	}
	base, err := s.Branch(ctx, "main")
	if err != nil {
		t.Fatal(err)
	}
	if got := base.State.Entities["item"]["a"]["owner"]; got != "base" {
		t.Fatalf("base branch was mutated: owner=%v", got)
	}
}
