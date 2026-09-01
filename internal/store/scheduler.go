package store

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"time"

	"github.com/augety121/mcp-state-twin/internal/canonical"
	"github.com/augety121/mcp-state-twin/internal/limits"
	"github.com/augety121/mcp-state-twin/internal/world"
)

var (
	ErrSchedulerNotFound   = errors.New("scheduled event not found")
	ErrSchedulerInvalid    = errors.New("invalid scheduled event")
	ErrSchedulerConflict   = errors.New("scheduled event lifecycle conflict")
	ErrSchedulerEmpty      = errors.New("scheduler has no pending events")
	ErrSchedulerCursor     = errors.New("invalid scheduler cursor")
	ErrSchedulerAction     = errors.New("scheduled action requires a compatible runtime")
	scheduledToolPattern   = regexp.MustCompile(`^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$`)
	scheduledDigestPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
)

const (
	SchedulerFormat       = "statetwin.dev/scheduler-state/v1alpha1"
	SchedulerPolicy       = "deterministic-queue-v2"
	SchedulerKindSignal   = world.SchedulerKindSignal
	SchedulerKindAction   = world.SchedulerKindAction
	SchedulerPending      = world.SchedulerPending
	SchedulerDelivered    = world.SchedulerDelivered
	SchedulerCanceled     = world.SchedulerCanceled
	SchedulerCompleted    = world.SchedulerCompleted
	SchedulerFailed       = world.SchedulerFailed
	MinSchedulerPriority  = -1000
	MaxSchedulerPriority  = 1000
	SchedulerCursorFormat = "statetwin.dev/scheduler-cursor/v1"
)

type ScheduleRequest struct {
	ID       string                 `json:"id"`
	BranchID string                 `json:"branch"`
	DueAt    string                 `json:"dueAt"`
	Priority int                    `json:"priority"`
	Kind     string                 `json:"kind"`
	Payload  any                    `json:"payload"`
	Action   *world.ScheduledAction `json:"action,omitempty"`
}

type ClockAdvanceResult struct {
	BranchID        string                 `json:"branch"`
	Clock           string                 `json:"clock"`
	HeadVersion     int64                  `json:"headVersion"`
	Delivered       []world.ScheduledEvent `json:"delivered"`
	SchedulerDigest string                 `json:"schedulerDigest"`
}

type ScheduledEventQuery struct {
	BranchID string
	Status   string
	Limit    int
	Cursor   string
}

type ScheduledEventPage struct {
	Format          string                 `json:"format"`
	Policy          string                 `json:"policy"`
	BranchID        string                 `json:"branch"`
	HeadVersion     int64                  `json:"headVersion"`
	SchedulerDigest string                 `json:"schedulerDigest"`
	Digest          string                 `json:"digest"`
	Status          string                 `json:"status,omitempty"`
	Events          []world.ScheduledEvent `json:"events"`
	NextCursor      string                 `json:"nextCursor,omitempty"`
}

type NextDueBatch struct {
	Format          string `json:"format"`
	Policy          string `json:"policy"`
	BranchID        string `json:"branch"`
	Clock           string `json:"clock"`
	HeadVersion     int64  `json:"headVersion"`
	SchedulerDigest string `json:"schedulerDigest"`
	DueAt           string `json:"dueAt"`
	PendingAtDueAt  int    `json:"pendingAtDueAt"`
	PendingActions  int    `json:"pendingActions"`
	BatchSize       int    `json:"batchSize"`
	BatchActions    int    `json:"batchActions"`
	RequiresDrain   bool   `json:"requiresDrain"`
}

type SchedulerStepResult struct {
	BranchID         string                 `json:"branch"`
	Clock            string                 `json:"clock"`
	ClockAdvanced    bool                   `json:"clockAdvanced"`
	HeadVersion      int64                  `json:"headVersion"`
	Delivered        []world.ScheduledEvent `json:"delivered"`
	Processed        []world.ScheduledEvent `json:"processed"`
	DeliveredSignals int                    `json:"deliveredSignals"`
	ExecutedActions  int                    `json:"executedActions"`
	MoreAtInstant    bool                   `json:"moreAtInstant"`
	NextDueAt        string                 `json:"nextDueAt,omitempty"`
	SchedulerDigest  string                 `json:"schedulerDigest"`
}

type schedulerCursor struct {
	Format           string `json:"format"`
	BranchID         string `json:"branch"`
	HeadVersion      int64  `json:"headVersion"`
	SchedulerDigest  string `json:"schedulerDigest"`
	Status           string `json:"status,omitempty"`
	DueAt            string `json:"dueAt"`
	Priority         int    `json:"priority"`
	CreationSequence int64  `json:"creationSequence"`
	ID               string `json:"id"`
}

type schedulerIdentity struct {
	Format string                `json:"format"`
	Policy string                `json:"policy"`
	State  *world.SchedulerState `json:"state,omitempty"`
}

type ScheduledActionApply func(state *world.State, clock time.Time, callIndex int64, tool string, input map[string]any) (CallOutcome, error)

func (s *Store) ScheduleEvent(ctx context.Context, request ScheduleRequest, expectedHeadVersion *int64) (*world.ScheduledEvent, error) {
	if err := validateID("scheduled event id", request.ID); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSchedulerInvalid, err)
	}
	if err := validateID("branch id", request.BranchID); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSchedulerInvalid, err)
	}
	switch request.Kind {
	case SchedulerKindSignal:
		if request.Action != nil {
			return nil, fmt.Errorf("%w: signal must not contain action", ErrSchedulerInvalid)
		}
	case SchedulerKindAction:
		if request.Payload != nil || request.Action == nil || !scheduledToolPattern.MatchString(request.Action.Tool) ||
			request.Action.Input == nil || !scheduledDigestPattern.MatchString(request.Action.SpecDigest) {
			return nil, fmt.Errorf("%w: tool-call requires a valid action envelope and no payload", ErrSchedulerInvalid)
		}
		if err := limits.ValidateJSON(request.Action.Input, limits.MaxInputBytes); err != nil {
			return nil, fmt.Errorf("%w: action input: %v", ErrSchedulerInvalid, err)
		}
	default:
		return nil, fmt.Errorf("%w: kind must be %q or %q", ErrSchedulerInvalid, SchedulerKindSignal, SchedulerKindAction)
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
	if request.Kind == SchedulerKindAction && request.Action.SpecDigest != branch.SpecDigest {
		return nil, fmt.Errorf("%w: action spec digest %s does not match branch %s", ErrSchedulerConflict, request.Action.SpecDigest, branch.SpecDigest)
	}
	if !dueAt.After(branch.Clock) {
		return nil, fmt.Errorf("%w: dueAt must be after current virtual clock", ErrSchedulerInvalid)
	}
	if dueAt.Sub(branch.Clock) > MaxClockAdvance {
		return nil, fmt.Errorf("%w: dueAt exceeds maximum horizon %s", ErrSchedulerInvalid, MaxClockAdvance)
	}
	scheduler := branch.State.EnsureScheduler()
	if _, exists := scheduler.Events[request.ID]; exists {
		return nil, fmt.Errorf("%w: event %s already exists", ErrSchedulerConflict, request.ID)
	}
	if len(scheduler.Events) >= limits.MaxScheduledEvents {
		return nil, fmt.Errorf("%w: scheduled event limit is %d", ErrResourceLimit, limits.MaxScheduledEvents)
	}
	pendingAtInstant := 0
	for _, existing := range scheduler.Events {
		if existing.Status == SchedulerPending && existing.DueAt == dueAt.Format(time.RFC3339Nano) {
			pendingAtInstant++
		}
	}
	if pendingAtInstant >= limits.MaxScheduledAtInstant {
		return nil, fmt.Errorf("%w: pending events at %s are limited to %d", ErrResourceLimit, dueAt.Format(time.RFC3339Nano), limits.MaxScheduledAtInstant)
	}
	if scheduler.NextCreationSequence == int64(^uint64(0)>>1) {
		return nil, fmt.Errorf("%w: creation sequence exhausted", ErrSchedulerInvalid)
	}
	scheduler.NextCreationSequence++
	event := world.ScheduledEvent{
		ID: request.ID, DueAt: dueAt.Format(time.RFC3339Nano), Priority: request.Priority,
		CreationSequence: scheduler.NextCreationSequence, Kind: request.Kind,
		Payload: request.Payload, Action: request.Action, Status: SchedulerPending,
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

// ScheduledEventPage returns a head-and-digest-bound page. A continuation
// cursor is valid only while the branch snapshot and filter remain unchanged.
func (s *Store) ScheduledEventPage(ctx context.Context, query ScheduledEventQuery) (*ScheduledEventPage, error) {
	if err := validateID("branch id", query.BranchID); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSchedulerInvalid, err)
	}
	if !validSchedulerStatusFilter(query.Status) {
		return nil, fmt.Errorf("%w: unsupported status filter %q", ErrSchedulerInvalid, query.Status)
	}
	if query.Limit < 1 || query.Limit > limits.MaxSchedulerPageSize {
		return nil, fmt.Errorf("%w: page limit must be 1..%d", ErrSchedulerInvalid, limits.MaxSchedulerPageSize)
	}
	if len(query.Cursor) > limits.MaxSchedulerCursor {
		return nil, fmt.Errorf("%w: cursor exceeds %d bytes", ErrSchedulerCursor, limits.MaxSchedulerCursor)
	}
	branch, err := s.Branch(ctx, query.BranchID)
	if err != nil {
		return nil, err
	}
	digest, err := schedulerDigest(branch.State)
	if err != nil {
		return nil, err
	}
	events := scheduledEvents(branch.State)
	filtered := make([]world.ScheduledEvent, 0, len(events))
	for _, event := range events {
		if query.Status == "" || event.Status == query.Status {
			filtered = append(filtered, event)
		}
	}
	start := 0
	if query.Cursor != "" {
		cursor, decodeErr := decodeSchedulerCursor(query.Cursor)
		if decodeErr != nil {
			return nil, decodeErr
		}
		if cursor.BranchID != query.BranchID || cursor.HeadVersion != branch.HeadVersion || cursor.SchedulerDigest != digest || cursor.Status != query.Status {
			return nil, fmt.Errorf("%w: cursor no longer matches scheduler snapshot", ErrSchedulerConflict)
		}
		found := false
		for index, event := range filtered {
			if cursorMatchesEvent(cursor, event) {
				start = index + 1
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("%w: cursor boundary is absent", ErrSchedulerConflict)
		}
	}
	end := start + query.Limit
	if end > len(filtered) {
		end = len(filtered)
	}
	pageEvents := append(make([]world.ScheduledEvent, 0, end-start), filtered[start:end]...)
	page := &ScheduledEventPage{
		Format: SchedulerFormat, Policy: SchedulerPolicy, BranchID: query.BranchID,
		HeadVersion: branch.HeadVersion, SchedulerDigest: digest, Status: query.Status,
		Digest: digest, Events: pageEvents,
	}
	if end < len(filtered) {
		cursor, encodeErr := encodeSchedulerCursor(schedulerCursorForEvent(query.BranchID, branch.HeadVersion, digest, query.Status, pageEvents[len(pageEvents)-1]))
		if encodeErr != nil {
			return nil, encodeErr
		}
		page.NextCursor = cursor
	}
	return page, nil
}

func (s *Store) NextDueBatch(ctx context.Context, branchID string) (*NextDueBatch, error) {
	if err := validateID("branch id", branchID); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSchedulerInvalid, err)
	}
	branch, err := s.Branch(ctx, branchID)
	if err != nil {
		return nil, err
	}
	events := pendingScheduledEvents(branch.State)
	if len(events) == 0 {
		return nil, ErrSchedulerEmpty
	}
	dueAt := events[0].DueAt
	count := 0
	actionCount := 0
	for _, event := range events {
		if event.DueAt != dueAt {
			break
		}
		count++
		if event.Kind == SchedulerKindAction {
			actionCount++
		}
	}
	digest, err := schedulerDigest(branch.State)
	if err != nil {
		return nil, err
	}
	batchSize, batchActions := scheduledBatchSize(events)
	return &NextDueBatch{
		Format: SchedulerFormat, Policy: SchedulerPolicy, BranchID: branchID,
		Clock: branch.Clock.UTC().Format(time.RFC3339Nano), HeadVersion: branch.HeadVersion,
		SchedulerDigest: digest, DueAt: dueAt, PendingAtDueAt: count, PendingActions: actionCount,
		BatchSize: batchSize, BatchActions: batchActions, RequiresDrain: count > batchSize,
	}, nil
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
			left, leftErr := events[i].DueTime()
			right, rightErr := events[j].DueTime()
			if leftErr == nil && rightErr == nil && !left.Equal(right) {
				return left.Before(right)
			}
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

func pendingScheduledEvents(state *world.State) []world.ScheduledEvent {
	events := scheduledEvents(state)
	pending := make([]world.ScheduledEvent, 0, len(events))
	for _, event := range events {
		if event.Status == SchedulerPending {
			pending = append(pending, event)
		}
	}
	return pending
}

func scheduledBatchSize(pending []world.ScheduledEvent) (int, int) {
	if len(pending) == 0 {
		return 0, 0
	}
	dueAt := pending[0].DueAt
	count := 0
	actions := 0
	for count < len(pending) && pending[count].DueAt == dueAt && count < limits.MaxScheduledDelivery {
		if pending[count].Kind == SchedulerKindAction {
			if actions >= limits.MaxScheduledActions {
				break
			}
			actions++
		}
		count++
	}
	return count, actions
}

func validSchedulerStatusFilter(status string) bool {
	switch status {
	case "", SchedulerPending, SchedulerDelivered, SchedulerCanceled, SchedulerCompleted, SchedulerFailed:
		return true
	default:
		return false
	}
}

func schedulerCursorForEvent(branchID string, headVersion int64, digest, status string, event world.ScheduledEvent) schedulerCursor {
	return schedulerCursor{
		Format: SchedulerCursorFormat, BranchID: branchID, HeadVersion: headVersion, SchedulerDigest: digest, Status: status,
		DueAt: event.DueAt, Priority: event.Priority, CreationSequence: event.CreationSequence, ID: event.ID,
	}
}

func cursorMatchesEvent(cursor schedulerCursor, event world.ScheduledEvent) bool {
	return cursor.DueAt == event.DueAt && cursor.Priority == event.Priority &&
		cursor.CreationSequence == event.CreationSequence && cursor.ID == event.ID
}

func encodeSchedulerCursor(cursor schedulerCursor) (string, error) {
	data, err := canonical.JSON(cursor)
	if err != nil {
		return "", fmt.Errorf("encode scheduler cursor: %w", err)
	}
	encoded := base64.RawURLEncoding.EncodeToString(data)
	if len(encoded) > limits.MaxSchedulerCursor {
		return "", fmt.Errorf("%w: encoded cursor exceeds %d bytes", ErrSchedulerCursor, limits.MaxSchedulerCursor)
	}
	return encoded, nil
}

func decodeSchedulerCursor(encoded string) (schedulerCursor, error) {
	var cursor schedulerCursor
	data, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return cursor, fmt.Errorf("%w: malformed base64url", ErrSchedulerCursor)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cursor); err != nil {
		return cursor, fmt.Errorf("%w: malformed payload", ErrSchedulerCursor)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return cursor, fmt.Errorf("%w: payload must contain one JSON value", ErrSchedulerCursor)
	}
	if cursor.Format != SchedulerCursorFormat || cursor.BranchID == "" || cursor.HeadVersion < 0 || cursor.SchedulerDigest == "" ||
		!validSchedulerStatusFilter(cursor.Status) || cursor.DueAt == "" || cursor.CreationSequence < 1 || cursor.ID == "" {
		return cursor, fmt.Errorf("%w: incomplete or unsupported payload", ErrSchedulerCursor)
	}
	canonicalData, err := canonical.JSON(cursor)
	if err != nil || !bytes.Equal(data, canonicalData) {
		return cursor, fmt.Errorf("%w: payload is not canonical", ErrSchedulerCursor)
	}
	return cursor, nil
}

func schedulerDigest(state *world.State) (string, error) {
	var scheduler *world.SchedulerState
	if state != nil {
		scheduler = state.Scheduler
	}
	return canonical.Digest(schedulerIdentity{Format: SchedulerFormat, Policy: SchedulerPolicy, State: scheduler})
}
