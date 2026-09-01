package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/augety121/mcp-state-twin/internal/limits"
	"github.com/augety121/mcp-state-twin/internal/world"
)

func TestSchedulerOrdersDeliversAndIsolatesForks(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	initial := world.New()
	initial.Entities["item"] = map[string]map[string]any{}
	clock := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	if err := s.InitializeBranch(ctx, "main", "sha256:spec", initial, clock); err != nil {
		t.Fatal(err)
	}
	requests := []ScheduleRequest{
		{ID: "normal-first", BranchID: "main", DueAt: clock.Add(time.Hour).Format(time.RFC3339), Priority: 0, Kind: SchedulerKindSignal, Payload: map[string]any{"n": 1}},
		{ID: "high-priority", BranchID: "main", DueAt: clock.Add(time.Hour).Format(time.RFC3339), Priority: 10, Kind: SchedulerKindSignal, Payload: map[string]any{"n": 2}},
		{ID: "normal-second", BranchID: "main", DueAt: clock.Add(time.Hour).Format(time.RFC3339), Priority: 0, Kind: SchedulerKindSignal, Payload: map[string]any{"n": 3}},
	}
	for _, request := range requests {
		if _, err := s.ScheduleEvent(ctx, request, nil); err != nil {
			t.Fatal(err)
		}
	}
	events, beforeDigest, err := s.ScheduledEvents(ctx, "main")
	if err != nil {
		t.Fatal(err)
	}
	wantOrder := []string{"high-priority", "normal-first", "normal-second"}
	for index, want := range wantOrder {
		if events[index].ID != want {
			t.Fatalf("event %d = %q, want %q", index, events[index].ID, want)
		}
	}
	if _, err := s.CreateSnapshot(ctx, "queued", "main"); err != nil {
		t.Fatal(err)
	}
	if err := s.Fork(ctx, "queued", "run-a"); err != nil {
		t.Fatal(err)
	}
	if err := s.Fork(ctx, "queued", "run-b"); err != nil {
		t.Fatal(err)
	}
	result, err := s.AdvanceClock(ctx, "run-a", clock.Add(2*time.Hour), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Delivered) != 3 {
		t.Fatalf("delivered events = %d, want 3", len(result.Delivered))
	}
	for index, want := range wantOrder {
		if result.Delivered[index].ID != want || result.Delivered[index].Status != SchedulerDelivered {
			t.Fatalf("delivered[%d] = %#v", index, result.Delivered[index])
		}
	}
	_, afterDigest, err := s.ScheduledEvents(ctx, "run-a")
	if err != nil {
		t.Fatal(err)
	}
	if afterDigest == beforeDigest {
		t.Fatal("scheduler digest did not bind delivery lifecycle")
	}
	sibling, siblingDigest, err := s.ScheduledEvents(ctx, "run-b")
	if err != nil {
		t.Fatal(err)
	}
	if siblingDigest != beforeDigest {
		t.Fatalf("sibling scheduler digest changed: %s != %s", siblingDigest, beforeDigest)
	}
	for _, event := range sibling {
		if event.Status != SchedulerPending {
			t.Fatalf("delivery leaked into sibling: %#v", event)
		}
	}
}

func TestSchedulerCancellationAndLifecycleConflicts(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	clock := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	if err := s.InitializeBranch(ctx, "main", "sha256:spec", world.New(), clock); err != nil {
		t.Fatal(err)
	}
	created, err := s.ScheduleEvent(ctx, ScheduleRequest{
		ID: "cancel-me", BranchID: "main", DueAt: clock.Add(time.Hour).Format(time.RFC3339),
		Kind: SchedulerKindSignal, Payload: map[string]any{},
	}, int64Pointer(0))
	if err != nil || created.CreationSequence != 1 {
		t.Fatalf("created event = %#v err=%v", created, err)
	}
	if _, err := s.ScheduleEvent(ctx, ScheduleRequest{
		ID: "stale", BranchID: "main", DueAt: clock.Add(2 * time.Hour).Format(time.RFC3339),
		Kind: SchedulerKindSignal, Payload: map[string]any{},
	}, int64Pointer(0)); !errors.Is(err, ErrBranchConflict) {
		t.Fatalf("stale scheduler CAS error = %v", err)
	}
	if _, err := s.ScheduleEvent(ctx, ScheduleRequest{
		ID: "too-late", BranchID: "main", DueAt: clock.Format(time.RFC3339),
		Kind: SchedulerKindSignal, Payload: map[string]any{},
	}, nil); !errors.Is(err, ErrSchedulerInvalid) {
		t.Fatalf("non-future dueAt error = %v", err)
	}
	if _, err := s.ScheduleEvent(ctx, ScheduleRequest{
		ID: "cancel-me", BranchID: "main", DueAt: clock.Add(time.Hour).Format(time.RFC3339),
		Kind: SchedulerKindSignal, Payload: map[string]any{},
	}, nil); !errors.Is(err, ErrSchedulerConflict) {
		t.Fatalf("duplicate event error = %v", err)
	}
	canceled, err := s.CancelScheduledEvent(ctx, "main", "cancel-me", int64Pointer(1))
	if err != nil {
		t.Fatal(err)
	}
	if canceled.Status != SchedulerCanceled || canceled.CanceledAt != clock.Format(time.RFC3339Nano) {
		t.Fatalf("canceled event = %#v", canceled)
	}
	if _, err := s.CancelScheduledEvent(ctx, "main", "cancel-me", nil); !errors.Is(err, ErrSchedulerConflict) {
		t.Fatalf("second cancellation error = %v", err)
	}
	result, err := s.AdvanceClock(ctx, "main", clock.Add(2*time.Hour), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Delivered) != 0 {
		t.Fatalf("canceled event delivered: %#v", result.Delivered)
	}
}

func TestSchedulerDeliveryBudgetRollsBackClockAndState(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	clock := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	initial := world.New()
	scheduler := initial.EnsureScheduler()
	for index := 0; index <= limits.MaxScheduledDelivery; index++ {
		id := fmt.Sprintf("event-%03d", index)
		scheduler.NextCreationSequence++
		scheduler.Events[id] = world.ScheduledEvent{
			ID: id, DueAt: clock.Add(time.Hour).Format(time.RFC3339), Priority: 0,
			CreationSequence: scheduler.NextCreationSequence, Kind: SchedulerKindSignal,
			Payload: map[string]any{}, Status: SchedulerPending,
		}
	}
	if err := s.InitializeBranch(ctx, "main", "sha256:spec", initial, clock); err != nil {
		t.Fatal(err)
	}
	before, err := s.Branch(ctx, "main")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.AdvanceClock(ctx, "main", clock.Add(2*time.Hour), nil); !errors.Is(err, ErrResourceLimit) {
		t.Fatalf("advance error = %v", err)
	}
	after, err := s.Branch(ctx, "main")
	if err != nil {
		t.Fatal(err)
	}
	if !after.Clock.Equal(before.Clock) || after.HeadVersion != before.HeadVersion || after.StateDigest != before.StateDigest {
		t.Fatalf("budget failure mutated branch: before=%#v after=%#v", before, after)
	}
}

func TestSchedulerRejectsMalformedPersistedLifecycleOnRead(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	if err := s.InitializeBranch(ctx, "main", "sha256:spec", world.New(), time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	malformed := `{"entities":{},"sequences":{},"scheduler":{"nextCreationSequence":1,"events":{"bad":{"id":"bad","dueAt":"2026-08-01T01:00:00Z","priority":0,"creationSequence":1,"kind":"signal","payload":{},"status":"delivered"}}}}`
	if _, err := s.db.ExecContext(ctx, `UPDATE branches SET state_json = ? WHERE id = 'main'`, malformed); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Branch(ctx, "main"); err == nil || !strings.Contains(err.Error(), "validate branch state") {
		t.Fatalf("malformed persisted scheduler read error = %v", err)
	}
}

func TestSchedulerUsesTemporalRatherThanLexicalRFC3339Order(t *testing.T) {
	state := world.New()
	scheduler := state.EnsureScheduler()
	scheduler.NextCreationSequence = 2
	scheduler.Events["fractional"] = world.ScheduledEvent{
		ID: "fractional", DueAt: "2026-08-01T01:00:00.1Z", Priority: 0,
		CreationSequence: 1, Kind: SchedulerKindSignal, Payload: map[string]any{}, Status: SchedulerPending,
	}
	scheduler.Events["whole"] = world.ScheduledEvent{
		ID: "whole", DueAt: "2026-08-01T01:00:00Z", Priority: 0,
		CreationSequence: 2, Kind: SchedulerKindSignal, Payload: map[string]any{}, Status: SchedulerPending,
	}
	events := scheduledEvents(state)
	if events[0].ID != "whole" || events[1].ID != "fractional" {
		t.Fatalf("temporal order = %q, %q", events[0].ID, events[1].ID)
	}
}

func TestSchedulerPerInstantAdmissionPreventsNewUndrainableQueues(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	clock := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	dueAt := clock.Add(time.Hour).Format(time.RFC3339Nano)
	initial := scheduledSignalState(limits.MaxScheduledAtInstant-1, dueAt)
	if err := s.InitializeBranch(ctx, "main", "sha256:spec", initial, clock); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ScheduleEvent(ctx, ScheduleRequest{
		ID: "at-limit", BranchID: "main", DueAt: dueAt, Kind: SchedulerKindSignal, Payload: map[string]any{},
	}, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ScheduleEvent(ctx, ScheduleRequest{
		ID: "over-limit", BranchID: "main", DueAt: dueAt, Kind: SchedulerKindSignal, Payload: map[string]any{},
	}, nil); !errors.Is(err, ErrResourceLimit) {
		t.Fatalf("overfull instant admission error = %v", err)
	}
	if _, err := s.CancelScheduledEvent(ctx, "main", "event-000", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ScheduleEvent(ctx, ScheduleRequest{
		ID: "replacement", BranchID: "main", DueAt: dueAt, Kind: SchedulerKindSignal, Payload: map[string]any{},
	}, nil); err != nil {
		t.Fatalf("cancellation did not free pending-instant capacity: %v", err)
	}
}

func TestSchedulerPerInstantAdmissionSerializesConcurrentFinalSlot(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	clock := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	dueAt := clock.Add(time.Hour).Format(time.RFC3339Nano)
	if err := s.InitializeBranch(ctx, "main", "sha256:spec", scheduledSignalState(limits.MaxScheduledAtInstant-1, dueAt), clock); err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	results := make(chan error, 2)
	for _, id := range []string{"concurrent-a", "concurrent-b"} {
		go func(eventID string) {
			<-start
			_, err := s.ScheduleEvent(ctx, ScheduleRequest{
				ID: eventID, BranchID: "main", DueAt: dueAt, Kind: SchedulerKindSignal, Payload: map[string]any{},
			}, nil)
			results <- err
		}(id)
	}
	close(start)
	successes := 0
	resourceFailures := 0
	for range 2 {
		err := <-results
		switch {
		case err == nil:
			successes++
		case errors.Is(err, ErrResourceLimit):
			resourceFailures++
		default:
			t.Fatalf("unexpected concurrent admission error = %v", err)
		}
	}
	if successes != 1 || resourceFailures != 1 {
		t.Fatalf("concurrent results success=%d resource=%d", successes, resourceFailures)
	}
}

func TestAdvanceClockToNextDrainsLegacyOverfullInstantDeterministically(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	clock := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	dueAt := clock.Add(time.Hour).Format(time.RFC3339Nano)
	if err := s.InitializeBranch(ctx, "main", "sha256:spec", scheduledSignalState(limits.MaxScheduledDelivery+1, dueAt), clock); err != nil {
		t.Fatal(err)
	}
	preview, err := s.NextDueBatch(ctx, "main")
	if err != nil {
		t.Fatal(err)
	}
	if preview.PendingAtDueAt != limits.MaxScheduledDelivery+1 || preview.BatchSize != limits.MaxScheduledDelivery || !preview.RequiresDrain {
		t.Fatalf("next due preview = %#v", preview)
	}
	first, err := s.AdvanceClockToNext(ctx, "main", int64Pointer(0))
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Delivered) != limits.MaxScheduledDelivery || !first.ClockAdvanced || !first.MoreAtInstant || first.NextDueAt != dueAt {
		t.Fatalf("first scheduler step = %#v", first)
	}
	for index, event := range first.Delivered {
		want := fmt.Sprintf("event-%03d", index)
		if event.ID != want {
			t.Fatalf("delivered[%d] = %s, want %s", index, event.ID, want)
		}
	}
	second, err := s.AdvanceClockToNext(ctx, "main", int64Pointer(1))
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Delivered) != 1 || second.Delivered[0].ID != "event-256" || second.ClockAdvanced || second.MoreAtInstant || second.NextDueAt != "" {
		t.Fatalf("second scheduler step = %#v", second)
	}
	if _, err := s.AdvanceClockToNext(ctx, "main", int64Pointer(2)); !errors.Is(err, ErrSchedulerEmpty) {
		t.Fatalf("empty scheduler step error = %v", err)
	}
	branch, err := s.Branch(ctx, "main")
	if err != nil {
		t.Fatal(err)
	}
	if branch.HeadVersion != 2 || branch.Clock.Format(time.RFC3339Nano) != dueAt {
		t.Fatalf("branch after drain = %#v", branch)
	}
}

func TestSchedulerPaginationIsDigestBoundAndFiltered(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	clock := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	if err := s.InitializeBranch(ctx, "main", "sha256:spec", world.New(), clock); err != nil {
		t.Fatal(err)
	}
	for index := 0; index < 4; index++ {
		if _, err := s.ScheduleEvent(ctx, ScheduleRequest{
			ID: fmt.Sprintf("page-%d", index), BranchID: "main",
			DueAt: clock.Add(time.Duration(index+1) * time.Hour).Format(time.RFC3339Nano),
			Kind:  SchedulerKindSignal, Payload: map[string]any{},
		}, nil); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.CancelScheduledEvent(ctx, "main", "page-1", nil); err != nil {
		t.Fatal(err)
	}
	first, err := s.ScheduledEventPage(ctx, ScheduledEventQuery{BranchID: "main", Status: SchedulerPending, Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Events) != 2 || first.Events[0].ID != "page-0" || first.Events[1].ID != "page-2" || first.NextCursor == "" || first.Digest != first.SchedulerDigest {
		t.Fatalf("first scheduler page = %#v", first)
	}
	second, err := s.ScheduledEventPage(ctx, ScheduledEventQuery{BranchID: "main", Status: SchedulerPending, Limit: 2, Cursor: first.NextCursor})
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Events) != 1 || second.Events[0].ID != "page-3" || second.NextCursor != "" {
		t.Fatalf("second scheduler page = %#v", second)
	}
	if _, err := s.ScheduleEvent(ctx, ScheduleRequest{
		ID: "page-4", BranchID: "main", DueAt: clock.Add(5 * time.Hour).Format(time.RFC3339Nano),
		Kind: SchedulerKindSignal, Payload: map[string]any{},
	}, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ScheduledEventPage(ctx, ScheduledEventQuery{BranchID: "main", Status: SchedulerPending, Limit: 2, Cursor: first.NextCursor}); !errors.Is(err, ErrSchedulerConflict) {
		t.Fatalf("stale cursor error = %v", err)
	}
	if _, err := s.ScheduledEventPage(ctx, ScheduledEventQuery{BranchID: "main", Limit: 2, Cursor: "not-base64*"}); !errors.Is(err, ErrSchedulerCursor) {
		t.Fatalf("malformed cursor error = %v", err)
	}
	empty, err := s.ScheduledEventPage(ctx, ScheduledEventQuery{BranchID: "main", Status: SchedulerDelivered, Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if empty.Events == nil || len(empty.Events) != 0 {
		t.Fatalf("empty page must encode an array: %#v", empty.Events)
	}
}

func scheduledSignalState(count int, dueAt string) *world.State {
	state := world.New()
	scheduler := state.EnsureScheduler()
	for index := 0; index < count; index++ {
		id := fmt.Sprintf("event-%03d", index)
		scheduler.NextCreationSequence++
		scheduler.Events[id] = world.ScheduledEvent{
			ID: id, DueAt: dueAt, Priority: 0, CreationSequence: scheduler.NextCreationSequence,
			Kind: SchedulerKindSignal, Payload: map[string]any{}, Status: SchedulerPending,
		}
	}
	return state
}

const scheduledTestSpecDigest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func TestScheduledActionInfrastructureFailureRollsBackWholeStep(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	clock := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	if err := s.InitializeBranch(ctx, "main", scheduledTestSpecDigest, world.New(), clock); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"first", "second"} {
		if _, err := s.ScheduleEvent(ctx, ScheduleRequest{
			ID: id, BranchID: "main", DueAt: clock.Add(time.Hour).Format(time.RFC3339Nano), Kind: SchedulerKindAction,
			Action: &world.ScheduledAction{Tool: "create_item", Input: map[string]any{"id": id}, SpecDigest: scheduledTestSpecDigest},
		}, nil); err != nil {
			t.Fatal(err)
		}
	}
	_, err := s.AdvanceClockToNextWithActions(ctx, "main", int64Pointer(2), scheduledTestSpecDigest,
		func(state *world.State, _ time.Time, _ int64, _ string, input map[string]any) (CallOutcome, error) {
			if input["id"] == "second" {
				return CallOutcome{}, errors.New("synthetic executor crash")
			}
			state.Entities["item"] = map[string]map[string]any{"first": {"id": "first"}}
			return CallOutcome{Result: map[string]any{"ok": true}, CommitState: true}, nil
		})
	if err == nil || !strings.Contains(err.Error(), "synthetic executor crash") {
		t.Fatalf("step error = %v", err)
	}
	branch, err := s.Branch(ctx, "main")
	if err != nil {
		t.Fatal(err)
	}
	if branch.HeadVersion != 2 || branch.CallCount != 0 || !branch.Clock.Equal(clock) || len(branch.State.Entities) != 0 {
		t.Fatalf("failed batch mutated branch = %#v", branch)
	}
	for _, event := range branch.State.Scheduler.Events {
		if event.Status != SchedulerPending || event.Outcome != nil || event.AttemptCount != 0 {
			t.Fatalf("failed batch terminalized event = %#v", event)
		}
	}
	var auditCount int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit`).Scan(&auditCount); err != nil || auditCount != 0 {
		t.Fatalf("audit count=%d err=%v", auditCount, err)
	}
}

func TestScheduledActionRejectsSpecDriftAtAdmissionAndExecution(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	clock := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	otherDigest := "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	if err := s.InitializeBranch(ctx, "main", scheduledTestSpecDigest, world.New(), clock); err != nil {
		t.Fatal(err)
	}
	request := ScheduleRequest{
		ID: "bound-action", BranchID: "main", DueAt: clock.Add(time.Hour).Format(time.RFC3339Nano), Kind: SchedulerKindAction,
		Action: &world.ScheduledAction{Tool: "create_item", Input: map[string]any{}, SpecDigest: otherDigest},
	}
	if _, err := s.ScheduleEvent(ctx, request, nil); !errors.Is(err, ErrSchedulerConflict) {
		t.Fatalf("mismatched admission error = %v", err)
	}
	request.Action.SpecDigest = scheduledTestSpecDigest
	if _, err := s.ScheduleEvent(ctx, request, int64Pointer(0)); err != nil {
		t.Fatal(err)
	}
	_, err := s.AdvanceClockToNextWithActions(ctx, "main", int64Pointer(1), otherDigest,
		func(*world.State, time.Time, int64, string, map[string]any) (CallOutcome, error) {
			return CallOutcome{Result: map[string]any{"ok": true}, CommitState: true}, nil
		})
	if err == nil || !strings.Contains(err.Error(), "SPEC_DRIFT") {
		t.Fatalf("runtime drift error = %v", err)
	}
	branch, err := s.Branch(ctx, "main")
	if err != nil {
		t.Fatal(err)
	}
	if branch.HeadVersion != 1 || branch.CallCount != 0 || branch.State.Scheduler.Events["bound-action"].Status != SchedulerPending {
		t.Fatalf("spec drift mutated branch = %#v", branch)
	}
}

func TestScheduledActionDomainFailureCommitsTerminalEvidenceWithoutEffect(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	clock := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	if err := s.InitializeBranch(ctx, "main", scheduledTestSpecDigest, world.New(), clock); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ScheduleEvent(ctx, ScheduleRequest{
		ID: "missing", BranchID: "main", DueAt: clock.Add(time.Hour).Format(time.RFC3339Nano), Kind: SchedulerKindAction,
		Action: &world.ScheduledAction{Tool: "close_item", Input: map[string]any{"id": "missing"}, SpecDigest: scheduledTestSpecDigest},
	}, nil); err != nil {
		t.Fatal(err)
	}
	result, err := s.AdvanceClockToNextWithActions(ctx, "main", int64Pointer(1), scheduledTestSpecDigest,
		func(*world.State, time.Time, int64, string, map[string]any) (CallOutcome, error) {
			return CallOutcome{Result: map[string]any{"error": map[string]any{"code": "NOT_FOUND"}}, ErrorClass: "NOT_FOUND", CommitState: false}, nil
		})
	if err != nil {
		t.Fatal(err)
	}
	event := result.Processed[0]
	if event.Status != SchedulerFailed || event.Outcome == nil || event.Outcome.EffectCommitted || event.Outcome.ErrorClass != "NOT_FOUND" {
		t.Fatalf("failed event = %#v", event)
	}
	branch, err := s.Branch(ctx, "main")
	if err != nil {
		t.Fatal(err)
	}
	if branch.CallCount != 1 || branch.HeadVersion != 2 || len(branch.State.Entities) != 0 {
		t.Fatalf("domain failure branch = %#v", branch)
	}
}

func TestScheduledActionRejectsCommittedFailureWithoutAmbiguityEvidence(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	clock := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	if err := s.InitializeBranch(ctx, "main", scheduledTestSpecDigest, world.New(), clock); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ScheduleEvent(ctx, ScheduleRequest{
		ID: "ambiguous", BranchID: "main", DueAt: clock.Add(time.Hour).Format(time.RFC3339Nano), Kind: SchedulerKindAction,
		Action: &world.ScheduledAction{Tool: "create_item", Input: map[string]any{}, SpecDigest: scheduledTestSpecDigest},
	}, nil); err != nil {
		t.Fatal(err)
	}
	_, err := s.AdvanceClockToNextWithActions(ctx, "main", int64Pointer(1), scheduledTestSpecDigest,
		func(*world.State, time.Time, int64, string, map[string]any) (CallOutcome, error) {
			return CallOutcome{Result: map[string]any{"error": map[string]any{"code": "CONFLICT"}}, ErrorClass: "CONFLICT", CommitState: true}, nil
		})
	if !errors.Is(err, ErrSchedulerInvalid) {
		t.Fatalf("committed failure error = %v", err)
	}
	branch, err := s.Branch(ctx, "main")
	if err != nil {
		t.Fatal(err)
	}
	if branch.HeadVersion != 1 || branch.CallCount != 0 || branch.State.Scheduler.Events["ambiguous"].Status != SchedulerPending {
		t.Fatalf("invalid committed failure mutated branch = %#v", branch)
	}
}

func TestScheduledActionCancellationIsTerminalAndNeverExecutes(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	clock := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	if err := s.InitializeBranch(ctx, "main", scheduledTestSpecDigest, world.New(), clock); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ScheduleEvent(ctx, ScheduleRequest{
		ID: "cancel-action", BranchID: "main", DueAt: clock.Add(time.Hour).Format(time.RFC3339Nano), Kind: SchedulerKindAction,
		Action: &world.ScheduledAction{Tool: "create_item", Input: map[string]any{}, SpecDigest: scheduledTestSpecDigest},
	}, nil); err != nil {
		t.Fatal(err)
	}
	canceled, err := s.CancelScheduledEvent(ctx, "main", "cancel-action", int64Pointer(1))
	if err != nil {
		t.Fatal(err)
	}
	if canceled.Status != SchedulerCanceled || canceled.Outcome != nil || canceled.AttemptCount != 0 {
		t.Fatalf("canceled action = %#v", canceled)
	}
	called := false
	_, err = s.AdvanceClockToNextWithActions(ctx, "main", int64Pointer(2), scheduledTestSpecDigest,
		func(*world.State, time.Time, int64, string, map[string]any) (CallOutcome, error) {
			called = true
			return CallOutcome{}, nil
		})
	if !errors.Is(err, ErrSchedulerEmpty) || called {
		t.Fatalf("canceled action advance error=%v called=%v", err, called)
	}
}

func TestScheduledActionRejectsSchedulerCascadeAndRollsBack(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	clock := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	if err := s.InitializeBranch(ctx, "main", scheduledTestSpecDigest, world.New(), clock); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ScheduleEvent(ctx, ScheduleRequest{
		ID: "cascade", BranchID: "main", DueAt: clock.Add(time.Hour).Format(time.RFC3339Nano), Kind: SchedulerKindAction,
		Action: &world.ScheduledAction{Tool: "spawn_event", Input: map[string]any{}, SpecDigest: scheduledTestSpecDigest},
	}, nil); err != nil {
		t.Fatal(err)
	}
	_, err := s.AdvanceClockToNextWithActions(ctx, "main", int64Pointer(1), scheduledTestSpecDigest,
		func(state *world.State, due time.Time, _ int64, _ string, _ map[string]any) (CallOutcome, error) {
			scheduler := state.EnsureScheduler()
			scheduler.NextCreationSequence++
			scheduler.Events["injected"] = world.ScheduledEvent{
				ID: "injected", DueAt: due.Add(time.Hour).Format(time.RFC3339Nano), CreationSequence: scheduler.NextCreationSequence,
				Kind: SchedulerKindSignal, Payload: map[string]any{}, Status: SchedulerPending,
			}
			return CallOutcome{Result: map[string]any{"ok": true}, CommitState: true}, nil
		})
	if !errors.Is(err, ErrSchedulerInvalid) {
		t.Fatalf("cascade error = %v", err)
	}
	branch, err := s.Branch(ctx, "main")
	if err != nil {
		t.Fatal(err)
	}
	if branch.HeadVersion != 1 || branch.CallCount != 0 || len(branch.State.Scheduler.Events) != 1 || branch.State.Scheduler.Events["cascade"].Status != SchedulerPending {
		t.Fatalf("cascade failure mutated branch = %#v", branch)
	}
}

func TestScheduledActionRejectsSchedulerCascadeEvenWhenCallbackReportsFailure(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	clock := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	if err := s.InitializeBranch(ctx, "main", scheduledTestSpecDigest, world.New(), clock); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ScheduleEvent(ctx, ScheduleRequest{
		ID: "hidden-cascade", BranchID: "main", DueAt: clock.Add(time.Hour).Format(time.RFC3339Nano), Kind: SchedulerKindAction,
		Action: &world.ScheduledAction{Tool: "spawn_event", Input: map[string]any{}, SpecDigest: scheduledTestSpecDigest},
	}, nil); err != nil {
		t.Fatal(err)
	}
	_, err := s.AdvanceClockToNextWithActions(ctx, "main", int64Pointer(1), scheduledTestSpecDigest,
		func(state *world.State, due time.Time, _ int64, _ string, _ map[string]any) (CallOutcome, error) {
			scheduler := state.EnsureScheduler()
			scheduler.NextCreationSequence++
			scheduler.Events["injected"] = world.ScheduledEvent{
				ID: "injected", DueAt: due.Add(time.Hour).Format(time.RFC3339Nano), CreationSequence: scheduler.NextCreationSequence,
				Kind: SchedulerKindSignal, Payload: map[string]any{}, Status: SchedulerPending,
			}
			return CallOutcome{Result: map[string]any{"error": map[string]any{"code": "REJECTED"}}, ErrorClass: "REJECTED"}, nil
		})
	if !errors.Is(err, ErrSchedulerInvalid) {
		t.Fatalf("failed callback cascade error = %v", err)
	}
	branch, err := s.Branch(ctx, "main")
	if err != nil {
		t.Fatal(err)
	}
	if branch.HeadVersion != 1 || branch.CallCount != 0 || len(branch.State.Scheduler.Events) != 1 || branch.State.Scheduler.Events["hidden-cascade"].Status != SchedulerPending {
		t.Fatalf("failed callback cascade mutated branch = %#v", branch)
	}
}

func TestScheduledActionBudgetSplitsOnlyAtTotalOrderBoundary(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	clock := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	dueAt := clock.Add(time.Hour).Format(time.RFC3339Nano)
	initial := scheduledActionState(limits.MaxScheduledActions+1, dueAt)
	if err := s.InitializeBranch(ctx, "main", scheduledTestSpecDigest, initial, clock); err != nil {
		t.Fatal(err)
	}
	preview, err := s.NextDueBatch(ctx, "main")
	if err != nil {
		t.Fatal(err)
	}
	if preview.BatchSize != limits.MaxScheduledActions || preview.BatchActions != limits.MaxScheduledActions || !preview.RequiresDrain {
		t.Fatalf("action preview = %#v", preview)
	}
	executor := func(*world.State, time.Time, int64, string, map[string]any) (CallOutcome, error) {
		return CallOutcome{Result: map[string]any{"ok": true}, CommitState: true}, nil
	}
	first, err := s.AdvanceClockToNextWithActions(ctx, "main", int64Pointer(0), scheduledTestSpecDigest, executor)
	if err != nil {
		t.Fatal(err)
	}
	if first.ExecutedActions != limits.MaxScheduledActions || !first.MoreAtInstant || len(first.Processed) != limits.MaxScheduledActions {
		t.Fatalf("first action batch = %#v", first)
	}
	second, err := s.AdvanceClockToNextWithActions(ctx, "main", int64Pointer(1), scheduledTestSpecDigest, executor)
	if err != nil {
		t.Fatal(err)
	}
	if second.ExecutedActions != 1 || second.MoreAtInstant || second.ClockAdvanced {
		t.Fatalf("second action batch = %#v", second)
	}
	branch, err := s.Branch(ctx, "main")
	if err != nil {
		t.Fatal(err)
	}
	if branch.CallCount != int64(limits.MaxScheduledActions+1) || branch.HeadVersion != 2 {
		t.Fatalf("action batch branch = %#v", branch)
	}
}

func TestScheduledActionBudgetIncludesFollowingSignalWithoutExceedingActionCap(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	clock := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	dueAt := clock.Add(time.Hour).Format(time.RFC3339Nano)
	initial := scheduledActionState(limits.MaxScheduledActions, dueAt)
	scheduler := initial.EnsureScheduler()
	scheduler.NextCreationSequence++
	scheduler.Events["signal-after-actions"] = world.ScheduledEvent{
		ID: "signal-after-actions", DueAt: dueAt, CreationSequence: scheduler.NextCreationSequence,
		Kind: SchedulerKindSignal, Payload: map[string]any{"ready": true}, Status: SchedulerPending,
	}
	if err := s.InitializeBranch(ctx, "main", scheduledTestSpecDigest, initial, clock); err != nil {
		t.Fatal(err)
	}
	executor := func(*world.State, time.Time, int64, string, map[string]any) (CallOutcome, error) {
		return CallOutcome{Result: map[string]any{"ok": true}, CommitState: true}, nil
	}
	first, err := s.AdvanceClockToNextWithActions(ctx, "main", int64Pointer(0), scheduledTestSpecDigest, executor)
	if err != nil {
		t.Fatal(err)
	}
	if first.ExecutedActions != limits.MaxScheduledActions || first.DeliveredSignals != 1 || first.MoreAtInstant ||
		len(first.Delivered) != 1 || first.Delivered[0].ID != "signal-after-actions" {
		t.Fatalf("ordered-prefix batch = %#v", first)
	}
}

func TestScheduledActionOversizedResultRollsBackWholeStep(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	clock := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	if err := s.InitializeBranch(ctx, "main", scheduledTestSpecDigest, world.New(), clock); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ScheduleEvent(ctx, ScheduleRequest{
		ID: "oversized", BranchID: "main", DueAt: clock.Add(time.Hour).Format(time.RFC3339Nano), Kind: SchedulerKindAction,
		Action: &world.ScheduledAction{Tool: "create_item", Input: map[string]any{}, SpecDigest: scheduledTestSpecDigest},
	}, nil); err != nil {
		t.Fatal(err)
	}
	_, err := s.AdvanceClockToNextWithActions(ctx, "main", int64Pointer(1), scheduledTestSpecDigest,
		func(*world.State, time.Time, int64, string, map[string]any) (CallOutcome, error) {
			return CallOutcome{Result: map[string]any{"blob": strings.Repeat("x", limits.MaxOutputBytes)}, CommitState: true}, nil
		})
	if !errors.Is(err, ErrResourceLimit) {
		t.Fatalf("oversized result error = %v", err)
	}
	branch, err := s.Branch(ctx, "main")
	if err != nil {
		t.Fatal(err)
	}
	if branch.HeadVersion != 1 || branch.CallCount != 0 || branch.State.Scheduler.Events["oversized"].Status != SchedulerPending {
		t.Fatalf("oversized result mutated branch = %#v", branch)
	}
}

func scheduledActionState(count int, dueAt string) *world.State {
	state := world.New()
	scheduler := state.EnsureScheduler()
	for index := 0; index < count; index++ {
		id := fmt.Sprintf("action-%03d", index)
		scheduler.NextCreationSequence++
		scheduler.Events[id] = world.ScheduledEvent{
			ID: id, DueAt: dueAt, CreationSequence: scheduler.NextCreationSequence, Kind: SchedulerKindAction,
			Action: &world.ScheduledAction{Tool: "create_item", Input: map[string]any{"id": id}, SpecDigest: scheduledTestSpecDigest},
			Status: SchedulerPending,
		}
	}
	return state
}
