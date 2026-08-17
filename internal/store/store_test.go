package store

import (
	"context"
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
}
