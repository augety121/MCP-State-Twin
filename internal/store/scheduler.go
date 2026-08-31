package store

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/augety121/mcp-state-twin/internal/canonical"
	"github.com/augety121/mcp-state-twin/internal/limits"
	"github.com/augety121/mcp-state-twin/internal/world"
)

var (
	ErrSchedulerNotFound = errors.New("scheduled event not found")
	ErrSchedulerInvalid  = errors.New("invalid scheduled event")
	ErrSchedulerConflict = errors.New("scheduled event lifecycle conflict")
)

const (
	SchedulerFormat      = "statetwin.dev/scheduler-state/v1alpha1"
	SchedulerPolicy      = "signal-queue-v1"
	SchedulerKindSignal  = world.SchedulerKindSignal
	SchedulerPending     = world.SchedulerPending
	SchedulerDelivered   = world.SchedulerDelivered
	SchedulerCanceled    = world.SchedulerCanceled
	MinSchedulerPriority = -1000
	MaxSchedulerPriority = 1000
)

type ScheduleRequest struct {
	ID       string `json:"id"`
	BranchID string `json:"branch"`
	DueAt    string `json:"dueAt"`
	Priority int    `json:"priority"`
	Kind     string `json:"kind"`
	Payload  any    `json:"payload"`
}

type ClockAdvanceResult struct {
	BranchID        string                 `json:"branch"`
	Clock           string                 `json:"clock"`
	HeadVersion     int64                  `json:"headVersion"`
	Delivered       []world.ScheduledEvent `json:"delivered"`
	SchedulerDigest string                 `json:"schedulerDigest"`
}

type schedulerIdentity struct {
	Format string                `json:"format"`
	Policy string                `json:"policy"`
	State  *world.SchedulerState `json:"state,omitempty"`
}

func (s *Store) ScheduleEvent(ctx context.Context, request ScheduleRequest, expectedHeadVersion *int64) (*world.ScheduledEvent, error) {
	if err := validateID("scheduled event id", request.ID); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSchedulerInvalid, err)
	}
	if err := validateID("branch id", request.BranchID); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSchedulerInvalid, err)
	}
	if request.Kind != SchedulerKindSignal {
		return nil, fmt.Errorf("%w: kind must be %q", ErrSchedulerInvalid, SchedulerKindSignal)
	}
	if request.Priority < MinSchedulerPriority || request.Priority > MaxSchedulerPriority {
		return nil, fmt.Errorf("%w: priority must be %d..%d", ErrSchedulerInvalid, MinSchedulerPriority, MaxSchedulerPriority)
	}
	if err := limits.ValidateJSON(request.Payload, limits.MaxInputBytes); err != nil {
		return nil, fmt.Errorf("%w: payload: %v", ErrSchedulerInvalid, err)
	}
	dueAt, err := time.Parse(time.RFC3339Nano, request.DueAt)
	if err != nil {
		return nil, fmt.Errorf("%w: dueAt must be RFC3339", ErrSchedulerInvalid)
	}
	dueAt = dueAt.UTC()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin schedule transaction: %w", err)
	}
	defer tx.Rollback()
	branch, err := scanBranch(request.BranchID, tx.QueryRowContext(ctx, `SELECT spec_digest, state_json, state_digest, clock, call_count, head_version FROM branches WHERE id = ?`, request.BranchID))
	if err != nil {
		return nil, err
	}
	if expectedHeadVersion != nil && *expectedHeadVersion != branch.HeadVersion {
		return nil, fmt.Errorf("%w: branch %s expected head %d, current %d", ErrBranchConflict, request.BranchID, *expectedHeadVersion, branch.HeadVersion)
	}
	if !dueAt.After(branch.Clock) {
		return nil, fmt.Errorf("%w: dueAt must be after current virtual clock", ErrSchedulerInvalid)
	}
	if dueAt.Sub(branch.Clock) > MaxClockAdvance {
		return nil, fmt.Errorf("%w: dueAt exceeds maximum horizon %s", ErrSchedulerInvalid, MaxClockAdvance)
	}
	scheduler := branch.State.EnsureScheduler()
	if len(scheduler.Events) >= limits.MaxScheduledEvents {
		return nil, fmt.Errorf("%w: scheduled event limit is %d", ErrResourceLimit, limits.MaxScheduledEvents)
	}
	if _, exists := scheduler.Events[request.ID]; exists {
		return nil, fmt.Errorf("%w: event %s already exists", ErrSchedulerConflict, request.ID)
	}
	if scheduler.NextCreationSequence == int64(^uint64(0)>>1) {
		return nil, fmt.Errorf("%w: creation sequence exhausted", ErrSchedulerInvalid)
	}
	scheduler.NextCreationSequence++
	event := world.ScheduledEvent{
		ID: request.ID, DueAt: dueAt.Format(time.RFC3339Nano), Priority: request.Priority,
		CreationSequence: scheduler.NextCreationSequence, Kind: request.Kind,
		Payload: request.Payload, Status: SchedulerPending,
	}
	scheduler.Events[event.ID] = event
	if err := branch.State.ValidateBudget(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrResourceLimit, err)
	}
	stateJSON, err := canonical.JSON(branch.State)
	if err != nil {
		return nil, fmt.Errorf("encode scheduled state: %w", err)
	}
	afterDigest, err := canonical.Digest(branch.State)
	if err != nil {
		return nil, fmt.Errorf("digest scheduled state: %w", err)
	}
	result, err := tx.ExecContext(ctx, `UPDATE branches SET state_json = ?, state_digest = ?, head_version = head_version + 1 WHERE id = ? AND head_version = ?`, stateJSON, afterDigest, request.BranchID, branch.HeadVersion)
	if err != nil {
		return nil, fmt.Errorf("persist scheduled event: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if count != 1 {
		return nil, fmt.Errorf("%w: branch %s expected head %d", ErrBranchConflict, request.BranchID, branch.HeadVersion)
	}
	if err := appendControlAudit(ctx, tx, "scheduler.event.create", request.BranchID, "", branch.StateDigest, afterDigest); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit schedule transaction: %w", err)
	}
	return &event, nil
}

func (s *Store) CancelScheduledEvent(ctx context.Context, branchID, eventID string, expectedHeadVersion *int64) (*world.ScheduledEvent, error) {
	if err := validateID("branch id", branchID); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSchedulerInvalid, err)
	}
	if err := validateID("scheduled event id", eventID); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSchedulerInvalid, err)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin cancellation transaction: %w", err)
	}
	defer tx.Rollback()
	branch, err := scanBranch(branchID, tx.QueryRowContext(ctx, `SELECT spec_digest, state_json, state_digest, clock, call_count, head_version FROM branches WHERE id = ?`, branchID))
	if err != nil {
		return nil, err
	}
	if expectedHeadVersion != nil && *expectedHeadVersion != branch.HeadVersion {
		return nil, fmt.Errorf("%w: branch %s expected head %d, current %d", ErrBranchConflict, branchID, *expectedHeadVersion, branch.HeadVersion)
	}
	if branch.State.Scheduler == nil {
		return nil, fmt.Errorf("%w: event %s", ErrSchedulerNotFound, eventID)
	}
	event, exists := branch.State.Scheduler.Events[eventID]
	if !exists {
		return nil, fmt.Errorf("%w: event %s", ErrSchedulerNotFound, eventID)
	}
	if event.Status != SchedulerPending {
		return nil, fmt.Errorf("%w: event %s is %s", ErrSchedulerConflict, eventID, event.Status)
	}
	event.Status = SchedulerCanceled
	event.CanceledAt = branch.Clock.UTC().Format(time.RFC3339Nano)
	branch.State.Scheduler.Events[eventID] = event
	stateJSON, err := canonical.JSON(branch.State)
	if err != nil {
		return nil, err
	}
	afterDigest, err := canonical.Digest(branch.State)
	if err != nil {
		return nil, err
	}
	result, err := tx.ExecContext(ctx, `UPDATE branches SET state_json = ?, state_digest = ?, head_version = head_version + 1 WHERE id = ? AND head_version = ?`, stateJSON, afterDigest, branchID, branch.HeadVersion)
	if err != nil {
		return nil, err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if count != 1 {
		return nil, fmt.Errorf("%w: branch %s expected head %d", ErrBranchConflict, branchID, branch.HeadVersion)
	}
	if err := appendControlAudit(ctx, tx, "scheduler.event.cancel", branchID, "", branch.StateDigest, afterDigest); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit cancellation transaction: %w", err)
	}
	return &event, nil
}

func (s *Store) ScheduledEvents(ctx context.Context, branchID string) ([]world.ScheduledEvent, string, error) {
	branch, err := s.Branch(ctx, branchID)
	if err != nil {
		return nil, "", err
	}
	events := scheduledEvents(branch.State)
	digest, err := schedulerDigest(branch.State)
	if err != nil {
		return nil, "", err
	}
	return events, digest, nil
}

func scheduledEvents(state *world.State) []world.ScheduledEvent {
	if state == nil || state.Scheduler == nil {
		return []world.ScheduledEvent{}
	}
	events := make([]world.ScheduledEvent, 0, len(state.Scheduler.Events))
	for _, event := range state.Scheduler.Events {
		events = append(events, event)
	}
	sort.Slice(events, func(i, j int) bool {
		if events[i].DueAt != events[j].DueAt {
			return events[i].DueAt < events[j].DueAt
		}
		if events[i].Priority != events[j].Priority {
			return events[i].Priority > events[j].Priority
		}
		if events[i].CreationSequence != events[j].CreationSequence {
			return events[i].CreationSequence < events[j].CreationSequence
		}
		return events[i].ID < events[j].ID
	})
	return events
}

func schedulerDigest(state *world.State) (string, error) {
	var scheduler *world.SchedulerState
	if state != nil {
		scheduler = state.Scheduler
	}
	return canonical.Digest(schedulerIdentity{Format: SchedulerFormat, Policy: SchedulerPolicy, State: scheduler})
}
