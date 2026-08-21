package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/augety121/mcp-state-twin/internal/limits"
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
	if branch.HeadVersion != 1 {
		t.Fatalf("head version = %d, want 1", branch.HeadVersion)
	}
}

func TestDeterministicFaultBeforeValidationDoesNotRunTransition(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	initial := world.New()
	initial.Entities["item"] = map[string]map[string]any{"a": {"id": "a", "state": "open"}}
	if err := s.InitializeBranch(ctx, "main", "sha256:spec", initial, time.Unix(0, 0)); err != nil {
		t.Fatal(err)
	}
	plan, err := s.InstallFault(ctx, FaultPlan{
		ID: "timeout-next", BranchID: "main", ToolName: "close",
		Phase: FaultPhaseBeforeValidation, ErrorClass: "TIMEOUT_BEFORE_EFFECT",
		Message: "synthetic request timeout", RemainingCount: 1,
	}, int64Pointer(0))
	if err != nil {
		t.Fatal(err)
	}
	if plan.FiredCount != 0 || plan.RemainingCount != 1 {
		t.Fatalf("installed plan = %#v", plan)
	}
	planDigest, err := s.FaultPlanDigest(ctx, "main")
	if err != nil {
		t.Fatal(err)
	}
	called := false
	result, err := s.ApplyCall(ctx, "main", "sha256:spec", "close", map[string]any{}, func(state *world.State, _ time.Time, _ int64) (CallOutcome, error) {
		called = true
		state.Entities["item"]["a"]["state"] = "closed"
		return CallOutcome{Result: map[string]any{"ok": true}, CommitState: true}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("before-validation fault executed the transition callback")
	}
	if result.ErrorClass != "TIMEOUT_BEFORE_EFFECT" || result.FaultID != "timeout-next" || result.BeforeDigest != result.AfterDigest {
		t.Fatalf("fault result = %#v", result)
	}
	branch, err := s.Branch(ctx, "main")
	if err != nil {
		t.Fatal(err)
	}
	if branch.State.Entities["item"]["a"]["state"] != "open" || branch.CallCount != 1 || branch.HeadVersion != 2 {
		t.Fatalf("branch after pre-effect fault = %#v", branch)
	}
	events, err := s.FaultEvents(ctx, "main")
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].FaultID != "timeout-next" || events[0].BeforeDigest != events[0].AfterDigest {
		t.Fatalf("fault events = %#v", events)
	}
	plans, err := s.FaultPlans(ctx, "main")
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) != 1 || plans[0].RemainingCount != 0 || plans[0].FiredCount != 1 {
		t.Fatalf("consumed plans = %#v", plans)
	}
	afterDigest, err := s.FaultPlanDigest(ctx, "main")
	if err != nil {
		t.Fatal(err)
	}
	if afterDigest != planDigest {
		t.Fatalf("fault plan identity changed when counter advanced: before %s after %s", planDigest, afterDigest)
	}
}

func TestDeterministicFaultAfterCommitHidesSuccessfulResult(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	initial := world.New()
	initial.Entities["item"] = map[string]map[string]any{"a": {"id": "a", "state": "open"}}
	if err := s.InitializeBranch(ctx, "main", "sha256:spec", initial, time.Unix(0, 0)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.InstallFault(ctx, FaultPlan{
		ID: "lost-response", BranchID: "main", ToolName: "close",
		Phase: FaultPhaseAfterCommitBeforeResponse, ErrorClass: "TIMEOUT_AFTER_EFFECT",
		Message: "synthetic response loss after commit", RemainingCount: 1,
	}, nil); err != nil {
		t.Fatal(err)
	}
	result, err := s.ApplyCall(ctx, "main", "sha256:spec", "close", map[string]any{}, func(state *world.State, _ time.Time, _ int64) (CallOutcome, error) {
		state.Entities["item"]["a"]["state"] = "closed"
		return CallOutcome{Result: map[string]any{"ok": true}, CommitState: true}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ErrorClass != "TIMEOUT_AFTER_EFFECT" || result.FaultPhase != FaultPhaseAfterCommitBeforeResponse || result.BeforeDigest == result.AfterDigest {
		t.Fatalf("post-commit fault result = %#v", result)
	}
	branch, err := s.Branch(ctx, "main")
	if err != nil {
		t.Fatal(err)
	}
	if branch.State.Entities["item"]["a"]["state"] != "closed" {
		t.Fatal("after-commit fault rolled back the committed business state")
	}
	second, err := s.ApplyCall(ctx, "main", "sha256:spec", "close", map[string]any{}, func(state *world.State, _ time.Time, _ int64) (CallOutcome, error) {
		return CallOutcome{Result: map[string]any{"ok": true}, CommitState: true}, nil
	})
	if err != nil || second.ErrorClass != "" || second.FaultID != "" {
		t.Fatalf("exhausted one-shot fault fired again: result=%#v err=%v", second, err)
	}
}

func TestFaultPlanValidationDigestAndBranchLocality(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	initial := world.New()
	initial.Entities["item"] = map[string]map[string]any{}
	for _, branch := range []string{"main", "sibling"} {
		if err := s.InitializeBranch(ctx, branch, "sha256:spec", initial, time.Unix(0, 0)); err != nil {
			t.Fatal(err)
		}
	}
	before, err := s.FaultPlanDigest(ctx, "main")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.InstallFault(ctx, FaultPlan{
		ID: "limited", BranchID: "main", ToolName: "create",
		Phase: FaultPhaseBeforeValidation, ErrorClass: "RATE_LIMITED",
		Message: "synthetic quota", RemainingCount: 2,
	}, nil); err != nil {
		t.Fatal(err)
	}
	after, err := s.FaultPlanDigest(ctx, "main")
	if err != nil {
		t.Fatal(err)
	}
	if before == after {
		t.Fatal("fault plan digest did not bind installed configuration")
	}
	sibling, err := s.FaultPlans(ctx, "sibling")
	if err != nil {
		t.Fatal(err)
	}
	if len(sibling) != 0 {
		t.Fatalf("fault plan leaked to sibling: %#v", sibling)
	}
	if _, err := s.InstallFault(ctx, FaultPlan{
		ID: "invalid", BranchID: "main", ToolName: "create",
		Phase: "random", ErrorClass: "MADE_UP", Message: "bad", RemainingCount: MaxFaultRepeats + 1,
	}, nil); !errors.Is(err, ErrFaultInvalid) {
		t.Fatalf("invalid plan error = %v", err)
	}
	if _, err := s.db.Exec(`
INSERT INTO fault_plans(id, branch_id, tool_name, phase, error_class, message, remaining_count, fired_count, created_at)
VALUES('tampered', 'main', 'create', 'before-validation', 'TIMEOUT_AFTER_EFFECT', 'invalid combination', 1, 0, '2026-08-21T00:00:00Z')`); err == nil {
		t.Fatal("SQLite constraints accepted an invalid phase/error combination")
	}
}

func TestFaultPlanAndCountersPersistAcrossReopen(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "faults.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	initial := world.New()
	initial.Entities["item"] = map[string]map[string]any{}
	if err := s.InitializeBranch(ctx, "main", "sha256:spec", initial, time.Unix(0, 0)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.InstallFault(ctx, FaultPlan{
		ID: "durable", BranchID: "main", ToolName: "create",
		Phase: FaultPhaseBeforeValidation, ErrorClass: "RATE_LIMITED",
		Message: "persisted synthetic quota", RemainingCount: 2,
	}, nil); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	result, err := s.ApplyCall(ctx, "main", "sha256:spec", "create", map[string]any{}, func(*world.State, time.Time, int64) (CallOutcome, error) {
		t.Fatal("persisted pre-validation fault did not fire")
		return CallOutcome{}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.FaultID != "durable" || result.ErrorClass != "RATE_LIMITED" {
		t.Fatalf("reopened fault result = %#v", result)
	}
	plans, err := s.FaultPlans(ctx, "main")
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) != 1 || plans[0].RemainingCount != 1 || plans[0].FiredCount != 1 {
		t.Fatalf("reopened fault counters = %#v", plans)
	}
}

func TestFaultSelectionUsesStablePlanIDOrder(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	initial := world.New()
	initial.Entities["item"] = map[string]map[string]any{}
	if err := s.InitializeBranch(ctx, "main", "sha256:spec", initial, time.Unix(0, 0)); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"z-last", "a-first"} {
		if _, err := s.InstallFault(ctx, FaultPlan{
			ID: id, BranchID: "main", ToolName: "create",
			Phase: FaultPhaseBeforeValidation, ErrorClass: "RATE_LIMITED",
			Message: id, RemainingCount: 1,
		}, nil); err != nil {
			t.Fatal(err)
		}
	}
	for index, want := range []string{"a-first", "z-last"} {
		result, err := s.ApplyCall(ctx, "main", "sha256:spec", "create", map[string]any{}, func(*world.State, time.Time, int64) (CallOutcome, error) {
			t.Fatal("ordered pre-validation fault did not fire")
			return CallOutcome{}, nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if result.FaultID != want {
			t.Fatalf("fault %d = %q, want %q", index, result.FaultID, want)
		}
	}
}

func TestDiffFailsClosedAtEntryLimit(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	initial := world.New()
	initial.Entities["item"] = map[string]map[string]any{}
	if err := s.InitializeBranch(ctx, "main", "sha256:spec", initial, time.Unix(0, 0)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateSnapshot(ctx, "base", "main"); err != nil {
		t.Fatal(err)
	}
	if err := s.Fork(ctx, "base", "changed"); err != nil {
		t.Fatal(err)
	}
	_, err := s.ApplyCall(ctx, "changed", "sha256:spec", "bulk", map[string]any{}, func(state *world.State, _ time.Time, _ int64) (CallOutcome, error) {
		for i := 0; i <= limits.MaxDiffEntries; i++ {
			state.Entities["item"][fmt.Sprintf("item-%05d", i)] = map[string]any{"id": i}
		}
		return CallOutcome{Result: map[string]any{"ok": true}, CommitState: true}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.DiffBranches(ctx, "main", "changed"); !errors.Is(err, ErrResourceLimit) {
		t.Fatalf("diff limit error = %v", err)
	}
}

func int64Pointer(value int64) *int64 { return &value }

func TestSnapshotForkIsolationAndDiff(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	initial := world.New()
	initial.Entities["item"] = map[string]map[string]any{"a": {"id": "a", "state": "open"}}
	if err := s.InitializeBranch(ctx, "main", "sha256:spec", initial, time.Unix(0, 0)); err != nil {
		t.Fatal(err)
	}
	snapshot, err := s.CreateSnapshot(ctx, "base", "main")
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.SourceHeadVersion != 0 {
		t.Fatalf("snapshot source head = %d, want 0", snapshot.SourceHeadVersion)
	}
	if err := s.Fork(ctx, "base", "run-a"); err != nil {
		t.Fatal(err)
	}
	if err := s.Fork(ctx, "base", "run-b"); err != nil {
		t.Fatal(err)
	}
	_, err = s.ApplyCall(ctx, "run-a", "sha256:spec", "close", map[string]any{}, func(state *world.State, _ time.Time, _ int64) (CallOutcome, error) {
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
	if branch.HeadVersion != 2 {
		t.Fatalf("head version after call+reset = %d, want 2", branch.HeadVersion)
	}

	// The reset rewinds the branch call index. A later call must still be
	// accepted while the pre-reset audit history remains append-only.
	if _, err := s.ApplyCall(ctx, "main", "sha256:spec", "close", map[string]any{}, closeItem); err != nil {
		t.Fatalf("call after reset: %v", err)
	}
	branch, err = s.Branch(ctx, "main")
	if err != nil {
		t.Fatal(err)
	}
	if branch.HeadVersion != 3 {
		t.Fatalf("head version after post-reset call = %d, want 3", branch.HeadVersion)
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
	if legacy.ID != "legacy-id" || legacy.StorageSchemaVersion != 1 || legacy.SourceHeadVersion != 0 {
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

func TestVersionTwoBranchMigratesWithMonotonicHead(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(fmt.Sprintf(`PRAGMA application_id = %d`, applicationID)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`PRAGMA user_version = 2`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
CREATE TABLE branches (
  id TEXT PRIMARY KEY,
  spec_digest TEXT NOT NULL,
  state_json BLOB NOT NULL,
  state_digest TEXT NOT NULL,
  clock TEXT NOT NULL,
  call_count INTEGER NOT NULL DEFAULT 0
);
INSERT INTO branches(id, spec_digest, state_json, state_digest, clock, call_count)
VALUES('main', 'sha256:spec', '{"entities":{},"sequences":{}}', 'sha256:state', '2026-08-01T00:00:00Z', 7);`); err != nil {
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
	branch, err := s.Branch(context.Background(), "main")
	if err != nil {
		t.Fatal(err)
	}
	if branch.HeadVersion != 0 || branch.CallCount != 7 {
		t.Fatalf("migrated branch = %#v", branch)
	}
	result, err := s.ApplyCall(context.Background(), "main", "sha256:spec", "read", map[string]any{}, func(state *world.State, _ time.Time, _ int64) (CallOutcome, error) {
		return CallOutcome{Result: map[string]any{"ok": true}, CommitState: true}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.CallIndex != 8 {
		t.Fatalf("migrated call index = %d, want 8", result.CallIndex)
	}
	branch, err = s.Branch(context.Background(), "main")
	if err != nil {
		t.Fatal(err)
	}
	if branch.HeadVersion != 1 {
		t.Fatalf("migrated head version after call = %d, want 1", branch.HeadVersion)
	}
	if _, err := s.InstallFault(context.Background(), FaultPlan{
		ID: "post-migration", BranchID: "main", ToolName: "read",
		Phase: FaultPhaseBeforeValidation, ErrorClass: "RATE_LIMITED",
		Message: "migration fixture", RemainingCount: 1,
	}, int64Pointer(1)); err != nil {
		t.Fatalf("schema-v4 fault tables unavailable after migration: %v", err)
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
