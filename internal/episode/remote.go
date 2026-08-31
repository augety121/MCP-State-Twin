package episode

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/augety121/mcp-state-twin/internal/bundle"
	"github.com/augety121/mcp-state-twin/internal/canonical"
	"github.com/augety121/mcp-state-twin/internal/limits"
)

const CoordinatorFormat = "statetwin.dev/episode-coordinator/v1alpha1"

var (
	ErrNoClaimableEpisode = errors.New("no claimable episode")
	ErrStaleAttempt       = errors.New("stale or expired episode attempt")
	ErrCancelRequested    = errors.New("episode cancellation requested")
	ErrCommitUnknown      = errors.New("episode commit state is unknown")
)

type TaskState string

const (
	TaskQueued        TaskState = "QUEUED"
	TaskLeased        TaskState = "LEASED"
	TaskCompleted     TaskState = "COMPLETED"
	TaskCancelled     TaskState = "CANCELLED"
	TaskFailed        TaskState = "FAILED"
	TaskCommitUnknown TaskState = "COMMIT_UNKNOWN"
)

type AttemptState string

const (
	AttemptLeased        AttemptState = "LEASED"
	AttemptCompleted     AttemptState = "COMPLETED"
	AttemptFailed        AttemptState = "FAILED"
	AttemptCancelled     AttemptState = "CANCELLED"
	AttemptExpired       AttemptState = "EXPIRED"
	AttemptCommitUnknown AttemptState = "COMMIT_UNKNOWN"
)

type CommitState string

const (
	CommitNotStarted CommitState = "NOT_STARTED"
	CommitNoEffect   CommitState = "NO_EFFECT"
	CommitCommitted  CommitState = "COMMITTED"
	CommitUnknown    CommitState = "UNKNOWN"
)

type EffectProfile string

const (
	EffectHermetic EffectProfile = "hermetic"
	EffectExternal EffectProfile = "external"
)

type TaskRecord struct {
	APIVersion          string          `json:"apiVersion"`
	Kind                string          `json:"kind"`
	Format              string          `json:"format"`
	Episode             *Record         `json:"episode"`
	EffectProfile       EffectProfile   `json:"effectProfile"`
	MaxAttempts         int             `json:"maxAttempts"`
	State               TaskState       `json:"state"`
	AttemptCount        int             `json:"attemptCount"`
	FencingToken        int64           `json:"fencingToken"`
	ActiveAttemptID     string          `json:"activeAttemptId,omitempty"`
	CancelRequested     bool            `json:"cancelRequested"`
	FinalAttemptID      string          `json:"finalAttemptId,omitempty"`
	FinalEvidenceDigest string          `json:"finalEvidenceDigest,omitempty"`
	CreatedAt           string          `json:"createdAt"`
	UpdatedAt           string          `json:"updatedAt"`
	Attempts            []AttemptRecord `json:"attempts"`
}

type AttemptRecord struct {
	AttemptID    string       `json:"attemptId"`
	EpisodeID    string       `json:"episodeId"`
	Number       int          `json:"number"`
	WorkerID     string       `json:"workerId"`
	FencingToken int64        `json:"fencingToken"`
	State        AttemptState `json:"state"`
	CommitState  CommitState  `json:"commitState"`
	LeaseUntil   string       `json:"leaseUntil"`
	ErrorClass   string       `json:"errorClass,omitempty"`
	CreatedAt    string       `json:"createdAt"`
	UpdatedAt    string       `json:"updatedAt"`
}

type Claim struct {
	APIVersion      string          `json:"apiVersion"`
	Kind            string          `json:"kind"`
	Format          string          `json:"format"`
	EpisodeID       string          `json:"episodeId"`
	AttemptID       string          `json:"attemptId"`
	AttemptNumber   int             `json:"attemptNumber"`
	FencingToken    int64           `json:"fencingToken"`
	LeaseUntil      string          `json:"leaseUntil"`
	Bundle          []byte          `json:"bundle"`
	BundleDigest    string          `json:"bundleDigest"`
	ScenarioPath    string          `json:"scenarioPath"`
	Runtime         RuntimeIdentity `json:"runtime"`
	EffectProfile   EffectProfile   `json:"effectProfile"`
	CancelRequested bool            `json:"cancelRequested"`
}

type HeartbeatResult struct {
	LeaseUntil      string `json:"leaseUntil"`
	CancelRequested bool   `json:"cancelRequested"`
}

// Submit creates an immutable remotely claimable task after validating the
// complete TwinBundle. Credentials and worker configuration are intentionally
// absent from the durable request.
func (j *Journal) Submit(ctx context.Context, bundleBytes []byte, episodeID, scenarioPath, runtimeVersion, runtimeRevision string, effectProfile EffectProfile, maxAttempts int) (*TaskRecord, bool, error) {
	if effectProfile != EffectHermetic && effectProfile != EffectExternal {
		return nil, false, errors.New("effect profile must be hermetic or external")
	}
	if maxAttempts < 1 || maxAttempts > limits.MaxEpisodeAttempts {
		return nil, false, fmt.Errorf("max attempts must be within 1..%d", limits.MaxEpisodeAttempts)
	}
	artifact, err := bundle.OpenBytes(bundleBytes)
	if err != nil {
		return nil, false, err
	}
	request, err := resolveRequest(artifact, episodeID, scenarioPath, runtimeVersion, runtimeRevision)
	if err != nil {
		return nil, false, err
	}
	_, episodeCreated, err := j.createOrLoad(ctx, request)
	if err != nil {
		return nil, false, err
	}
	tx, err := j.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, false, err
	}
	defer tx.Rollback()
	var storedProfile EffectProfile
	var storedMax int
	err = tx.QueryRowContext(ctx, `SELECT effect_profile, max_attempts FROM episode_tasks WHERE episode_id = ?`, episodeID).Scan(&storedProfile, &storedMax)
	if err == nil {
		if storedProfile != effectProfile || storedMax != maxAttempts {
			return nil, false, fmt.Errorf("%w: Episode task policy is immutable", ErrEpisodeConflict)
		}
		if err := tx.Commit(); err != nil {
			return nil, false, err
		}
		record, getErr := j.GetTask(ctx, episodeID)
		return record, false, getErr
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, false, err
	}
	if !episodeCreated {
		episodeRecord, getErr := j.Get(ctx, episodeID)
		if getErr != nil {
			return nil, false, getErr
		}
		if IsTerminal(episodeRecord.Status) || episodeRecord.Sequence != 0 {
			return nil, false, fmt.Errorf("%w: existing Episode cannot become a remote task", ErrEpisodeConflict)
		}
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := tx.ExecContext(ctx, `INSERT INTO episode_tasks(episode_id, bundle_bytes, effect_profile, max_attempts, task_state, created_at, updated_at) VALUES(?, ?, ?, ?, ?, ?, ?)`, episodeID, bundleBytes, effectProfile, maxAttempts, TaskQueued, now, now); err != nil {
		return nil, false, fmt.Errorf("create Episode task: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, false, err
	}
	record, err := j.GetTask(ctx, episodeID)
	return record, true, err
}

func (j *Journal) Claim(ctx context.Context, workerID string, lease time.Duration) (*Claim, error) {
	return j.ClaimProfile(ctx, workerID, lease, "")
}

func (j *Journal) ClaimProfile(ctx context.Context, workerID string, lease time.Duration, profile EffectProfile) (*Claim, error) {
	if !episodeIDPattern.MatchString(workerID) {
		return nil, errors.New("worker ID must match ^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$")
	}
	if profile != "" && profile != EffectHermetic && profile != EffectExternal {
		return nil, errors.New("claim effect profile must be hermetic, external, or empty")
	}
	if lease <= 0 || lease > time.Duration(limits.MaxLeaseSeconds)*time.Second {
		return nil, fmt.Errorf("lease must be within 1s..%ds", limits.MaxLeaseSeconds)
	}
	now := time.Now().UTC()
	tx, err := j.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if _, err := recoverExpiredTx(ctx, tx, now); err != nil {
		return nil, err
	}
	var claim Claim
	var attemptCount int
	err = tx.QueryRowContext(ctx, `SELECT t.episode_id, t.bundle_bytes, e.bundle_digest, e.scenario_path, e.runtime_version, e.runtime_revision, t.effect_profile, t.attempt_count FROM episode_tasks t JOIN episodes e ON e.id = t.episode_id WHERE t.task_state = ? AND t.cancel_requested = 0 AND (? = '' OR t.effect_profile = ?) ORDER BY t.created_at, t.episode_id LIMIT 1`, TaskQueued, profile, profile).Scan(&claim.EpisodeID, &claim.Bundle, &claim.BundleDigest, &claim.ScenarioPath, &claim.Runtime.Version, &claim.Runtime.Revision, &claim.EffectProfile, &attemptCount)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNoClaimableEpisode
	}
	if err != nil {
		return nil, err
	}
	claim.AttemptID, err = newAttemptID()
	if err != nil {
		return nil, err
	}
	claim.AttemptNumber = attemptCount + 1
	claim.LeaseUntil = now.Add(lease).Format(time.RFC3339Nano)
	var fence int64
	if err := tx.QueryRowContext(ctx, `SELECT fencing_token + 1 FROM episode_tasks WHERE episode_id = ?`, claim.EpisodeID).Scan(&fence); err != nil {
		return nil, err
	}
	claim.FencingToken = fence
	stamp := now.Format(time.RFC3339Nano)
	result, err := tx.ExecContext(ctx, `UPDATE episode_tasks SET task_state = ?, attempt_count = ?, fencing_token = ?, active_attempt_id = ?, updated_at = ? WHERE episode_id = ? AND task_state = ?`, TaskLeased, claim.AttemptNumber, fence, claim.AttemptID, stamp, claim.EpisodeID, TaskQueued)
	if err != nil {
		return nil, err
	}
	if changed, _ := result.RowsAffected(); changed != 1 {
		return nil, ErrNoClaimableEpisode
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO episode_attempts(attempt_id, episode_id, attempt_number, worker_id, fencing_token, state, commit_state, lease_until, created_at, updated_at) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, claim.AttemptID, claim.EpisodeID, claim.AttemptNumber, workerID, fence, AttemptLeased, CommitNotStarted, claim.LeaseUntil, stamp, stamp); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	claim.APIVersion, claim.Kind, claim.Format = APIVersion, "EpisodeClaim", CoordinatorFormat
	return &claim, nil
}

func (j *Journal) Heartbeat(ctx context.Context, episodeID, attemptID string, fence int64, lease time.Duration) (*HeartbeatResult, error) {
	if lease <= 0 || lease > time.Duration(limits.MaxLeaseSeconds)*time.Second {
		return nil, fmt.Errorf("lease must be within 1s..%ds", limits.MaxLeaseSeconds)
	}
	now := time.Now().UTC()
	tx, err := j.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var cancel bool
	var leaseUntil string
	err = tx.QueryRowContext(ctx, `SELECT t.cancel_requested, a.lease_until FROM episode_tasks t JOIN episode_attempts a ON a.attempt_id = t.active_attempt_id WHERE t.episode_id = ? AND t.task_state = ? AND t.active_attempt_id = ? AND t.fencing_token = ? AND a.state = ?`, episodeID, TaskLeased, attemptID, fence, AttemptLeased).Scan(&cancel, &leaseUntil)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrStaleAttempt
	}
	if err != nil {
		return nil, err
	}
	parsed, err := time.Parse(time.RFC3339Nano, leaseUntil)
	if err != nil || !parsed.After(now) {
		return nil, ErrStaleAttempt
	}
	next := now.Add(lease).Format(time.RFC3339Nano)
	result, err := tx.ExecContext(ctx, `UPDATE episode_attempts SET lease_until = ?, updated_at = ? WHERE attempt_id = ? AND fencing_token = ? AND state = ?`, next, now.Format(time.RFC3339Nano), attemptID, fence, AttemptLeased)
	if err != nil {
		return nil, err
	}
	if err := requireOneRow(result, ErrStaleAttempt); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &HeartbeatResult{LeaseUntil: next, CancelRequested: cancel}, nil
}

func (j *Journal) CompleteClaim(ctx context.Context, episodeID, attemptID string, fence int64, envelope *Envelope) (*Envelope, bool, error) {
	if envelope == nil || envelope.Evidence == nil {
		return nil, false, errors.New("Episode Evidence is required")
	}
	tx, err := j.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, false, err
	}
	defer tx.Rollback()
	var state TaskState
	var active, finalAttempt, finalDigest string
	var storedFence int64
	var cancel bool
	err = tx.QueryRowContext(ctx, `SELECT task_state, active_attempt_id, final_attempt_id, final_evidence_digest, fencing_token, cancel_requested FROM episode_tasks WHERE episode_id = ?`, episodeID).Scan(&state, &active, &finalAttempt, &finalDigest, &storedFence, &cancel)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, ErrEpisodeNotFound
	}
	if err != nil {
		return nil, false, err
	}
	if state == TaskCompleted {
		if finalAttempt == attemptID && storedFence == fence && finalDigest == envelope.EvidenceDigest {
			record, err := scanRecord(tx.QueryRowContext(ctx, `SELECT id, request_digest, bundle_digest, scenario_path, runtime_version, runtime_revision, status, sequence, outcome, evidence_json, evidence_digest, error_class, created_at, updated_at FROM episodes WHERE id = ?`, episodeID))
			if err != nil {
				return nil, false, err
			}
			if err := loadEvents(ctx, tx, record); err != nil {
				return nil, false, err
			}
			return record.Evidence, true, nil
		}
		return nil, false, ErrStaleAttempt
	}
	if state != TaskLeased || active != attemptID || storedFence != fence {
		return nil, false, ErrStaleAttempt
	}
	if cancel {
		return nil, false, ErrCancelRequested
	}
	request, err := requestForEpisodeTx(ctx, tx, episodeID)
	if err != nil {
		return nil, false, err
	}
	if err := validateRemoteEnvelope(request, envelope); err != nil {
		return nil, false, err
	}
	var attemptState AttemptState
	var leaseUntil string
	if err := tx.QueryRowContext(ctx, `SELECT state, lease_until FROM episode_attempts WHERE attempt_id = ? AND episode_id = ? AND fencing_token = ?`, attemptID, episodeID, fence).Scan(&attemptState, &leaseUntil); err != nil {
		return nil, false, ErrStaleAttempt
	}
	if attemptState != AttemptLeased {
		return nil, false, ErrStaleAttempt
	}
	if parsed, parseErr := time.Parse(time.RFC3339Nano, leaseUntil); parseErr != nil || !parsed.After(time.Now().UTC()) {
		return nil, false, ErrStaleAttempt
	}
	encoded, err := canonical.JSON(envelope)
	if err != nil {
		return nil, false, err
	}
	if len(encoded) > limits.MaxReportBytes {
		return nil, false, fmt.Errorf("RESOURCE_LIMIT: Episode evidence exceeds %d bytes", limits.MaxReportBytes)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	for _, event := range envelope.Evidence.Lifecycle.Events[1:] {
		if _, err := tx.ExecContext(ctx, `INSERT INTO episode_events(episode_id, sequence, from_status, to_status, created_at) VALUES(?, ?, ?, ?, ?)`, episodeID, event.Sequence, event.From, event.To, now); err != nil {
			return nil, false, err
		}
	}
	terminal := envelope.Evidence.Lifecycle.Current
	result, err := tx.ExecContext(ctx, `UPDATE episodes SET status = ?, sequence = ?, outcome = ?, evidence_json = ?, evidence_digest = ?, updated_at = ? WHERE id = ? AND status = ? AND sequence = 0`, terminal, len(envelope.Evidence.Lifecycle.Events)-1, envelope.Evidence.Outcome, encoded, envelope.EvidenceDigest, now, episodeID, StatusCreated)
	if err != nil {
		return nil, false, err
	}
	if changed, _ := result.RowsAffected(); changed != 1 {
		return nil, false, ErrStaleAttempt
	}
	attemptResult, err := tx.ExecContext(ctx, `UPDATE episode_attempts SET state = ?, commit_state = ?, updated_at = ? WHERE attempt_id = ? AND state = ? AND fencing_token = ?`, AttemptCompleted, CommitCommitted, now, attemptID, AttemptLeased, fence)
	if err != nil {
		return nil, false, err
	}
	if err := requireOneRow(attemptResult, ErrStaleAttempt); err != nil {
		return nil, false, err
	}
	taskResult, err := tx.ExecContext(ctx, `UPDATE episode_tasks SET task_state = ?, active_attempt_id = '', final_attempt_id = ?, final_evidence_digest = ?, updated_at = ? WHERE episode_id = ? AND active_attempt_id = ? AND fencing_token = ? AND task_state = ?`, TaskCompleted, attemptID, envelope.EvidenceDigest, now, episodeID, attemptID, fence, TaskLeased)
	if err != nil {
		return nil, false, err
	}
	if err := requireOneRow(taskResult, ErrStaleAttempt); err != nil {
		return nil, false, err
	}
	if err := tx.Commit(); err != nil {
		return nil, false, err
	}
	return envelope, false, nil
}

func (j *Journal) FailClaim(ctx context.Context, episodeID, attemptID string, fence int64, commitState CommitState, errorClass string) (*TaskRecord, error) {
	if commitState != CommitNoEffect && commitState != CommitUnknown {
		return nil, errors.New("failed attempts must report NO_EFFECT or UNKNOWN")
	}
	if errorClass != "WORKER_ERROR" && errorClass != "PROVIDER_ERROR" && errorClass != "CANCELLED" {
		return nil, errors.New("unsupported typed worker error class")
	}
	tx, err := j.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var state TaskState
	var active string
	var storedFence int64
	var cancel bool
	var attempts, maxAttempts int
	err = tx.QueryRowContext(ctx, `SELECT task_state, active_attempt_id, fencing_token, cancel_requested, attempt_count, max_attempts FROM episode_tasks WHERE episode_id = ?`, episodeID).Scan(&state, &active, &storedFence, &cancel, &attempts, &maxAttempts)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrEpisodeNotFound
	}
	if err != nil {
		return nil, err
	}
	if state != TaskLeased || active != attemptID || storedFence != fence {
		return nil, ErrStaleAttempt
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	attemptState, taskState := AttemptFailed, TaskFailed
	if commitState == CommitUnknown {
		attemptState, taskState = AttemptCommitUnknown, TaskCommitUnknown
	} else if cancel {
		attemptState, taskState = AttemptCancelled, TaskCancelled
	} else if attempts < maxAttempts {
		taskState = TaskQueued
	}
	attemptResult, err := tx.ExecContext(ctx, `UPDATE episode_attempts SET state = ?, commit_state = ?, error_class = ?, updated_at = ? WHERE attempt_id = ? AND state = ? AND fencing_token = ?`, attemptState, commitState, errorClass, now, attemptID, AttemptLeased, fence)
	if err != nil {
		return nil, err
	}
	if err := requireOneRow(attemptResult, ErrStaleAttempt); err != nil {
		return nil, err
	}
	taskResult, err := tx.ExecContext(ctx, `UPDATE episode_tasks SET task_state = ?, active_attempt_id = '', updated_at = ? WHERE episode_id = ? AND active_attempt_id = ? AND fencing_token = ? AND task_state = ?`, taskState, now, episodeID, attemptID, fence, TaskLeased)
	if err != nil {
		return nil, err
	}
	if err := requireOneRow(taskResult, ErrStaleAttempt); err != nil {
		return nil, err
	}
	if taskState == TaskCancelled {
		if err := cancelParentTx(ctx, tx, episodeID, now); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return j.GetTask(ctx, episodeID)
}

func (j *Journal) CancelTask(ctx context.Context, episodeID string) (*TaskRecord, error) {
	tx, err := j.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var state TaskState
	if err := tx.QueryRowContext(ctx, `SELECT task_state FROM episode_tasks WHERE episode_id = ?`, episodeID).Scan(&state); errors.Is(err, sql.ErrNoRows) {
		return nil, ErrEpisodeNotFound
	} else if err != nil {
		return nil, err
	}
	if state == TaskCompleted {
		return nil, fmt.Errorf("%w: completed Episode cannot be cancelled", ErrEpisodeConflict)
	}
	if state == TaskCancelled || state == TaskFailed || state == TaskCommitUnknown {
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return j.GetTask(ctx, episodeID)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	next := state
	if state == TaskQueued {
		next = TaskCancelled
		if err := cancelParentTx(ctx, tx, episodeID, now); err != nil {
			return nil, err
		}
	}
	result, err := tx.ExecContext(ctx, `UPDATE episode_tasks SET cancel_requested = 1, task_state = ?, updated_at = ? WHERE episode_id = ? AND task_state = ?`, next, now, episodeID, state)
	if err != nil {
		return nil, err
	}
	if err := requireOneRow(result, ErrEpisodeConflict); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return j.GetTask(ctx, episodeID)
}

func (j *Journal) RecoverExpired(ctx context.Context) (int, error) {
	tx, err := j.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	count, err := recoverExpiredTx(ctx, tx, time.Now().UTC())
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return count, nil
}

func recoverExpiredTx(ctx context.Context, tx *sql.Tx, now time.Time) (int, error) {
	rows, err := tx.QueryContext(ctx, `SELECT t.episode_id, t.active_attempt_id, t.effect_profile, t.cancel_requested, t.attempt_count, t.max_attempts, t.fencing_token FROM episode_tasks t JOIN episode_attempts a ON a.attempt_id = t.active_attempt_id WHERE t.task_state = ? AND a.state = ? AND a.lease_until <= ?`, TaskLeased, AttemptLeased, now.Format(time.RFC3339Nano))
	if err != nil {
		return 0, err
	}
	type expired struct {
		episode, attempt string
		profile          EffectProfile
		cancel           bool
		count, max       int
		fence            int64
	}
	var found []expired
	for rows.Next() {
		var item expired
		if err := rows.Scan(&item.episode, &item.attempt, &item.profile, &item.cancel, &item.count, &item.max, &item.fence); err != nil {
			rows.Close()
			return 0, err
		}
		found = append(found, item)
	}
	if err := rows.Close(); err != nil {
		return 0, err
	}
	stamp := now.Format(time.RFC3339Nano)
	for _, item := range found {
		attemptState, commitState, taskState := AttemptExpired, CommitNoEffect, TaskFailed
		if item.profile == EffectExternal {
			attemptState, commitState, taskState = AttemptCommitUnknown, CommitUnknown, TaskCommitUnknown
		} else if item.cancel {
			attemptState, taskState = AttemptCancelled, TaskCancelled
		} else if item.count < item.max {
			taskState = TaskQueued
		}
		attemptResult, err := tx.ExecContext(ctx, `UPDATE episode_attempts SET state = ?, commit_state = ?, error_class = ?, updated_at = ? WHERE attempt_id = ? AND state = ? AND fencing_token = ?`, attemptState, commitState, "LEASE_EXPIRED", stamp, item.attempt, AttemptLeased, item.fence)
		if err != nil {
			return 0, err
		}
		if err := requireOneRow(attemptResult, ErrStaleAttempt); err != nil {
			return 0, err
		}
		taskResult, err := tx.ExecContext(ctx, `UPDATE episode_tasks SET task_state = ?, active_attempt_id = '', updated_at = ? WHERE episode_id = ? AND active_attempt_id = ? AND fencing_token = ? AND task_state = ?`, taskState, stamp, item.episode, item.attempt, item.fence, TaskLeased)
		if err != nil {
			return 0, err
		}
		if err := requireOneRow(taskResult, ErrStaleAttempt); err != nil {
			return 0, err
		}
		if taskState == TaskCancelled {
			if err := cancelParentTx(ctx, tx, item.episode, stamp); err != nil {
				return 0, err
			}
		}
	}
	return len(found), nil
}

func (j *Journal) GetTask(ctx context.Context, episodeID string) (*TaskRecord, error) {
	if !episodeIDPattern.MatchString(episodeID) {
		return nil, errors.New("episode ID is invalid")
	}
	task := &TaskRecord{APIVersion: APIVersion, Kind: "EpisodeTask", Format: CoordinatorFormat}
	err := j.db.QueryRowContext(ctx, `SELECT effect_profile, max_attempts, task_state, attempt_count, fencing_token, active_attempt_id, cancel_requested, final_attempt_id, final_evidence_digest, created_at, updated_at FROM episode_tasks WHERE episode_id = ?`, episodeID).Scan(&task.EffectProfile, &task.MaxAttempts, &task.State, &task.AttemptCount, &task.FencingToken, &task.ActiveAttemptID, &task.CancelRequested, &task.FinalAttemptID, &task.FinalEvidenceDigest, &task.CreatedAt, &task.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrEpisodeNotFound
	}
	if err != nil {
		return nil, err
	}
	task.Episode, err = j.Get(ctx, episodeID)
	if err != nil {
		return nil, err
	}
	rows, err := j.db.QueryContext(ctx, `SELECT attempt_id, episode_id, attempt_number, worker_id, fencing_token, state, commit_state, lease_until, error_class, created_at, updated_at FROM episode_attempts WHERE episode_id = ? ORDER BY attempt_number`, episodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var attempt AttemptRecord
		if err := rows.Scan(&attempt.AttemptID, &attempt.EpisodeID, &attempt.Number, &attempt.WorkerID, &attempt.FencingToken, &attempt.State, &attempt.CommitState, &attempt.LeaseUntil, &attempt.ErrorClass, &attempt.CreatedAt, &attempt.UpdatedAt); err != nil {
			return nil, err
		}
		if _, err := time.Parse(time.RFC3339Nano, attempt.LeaseUntil); err != nil {
			return nil, errors.New("Episode attempt lease timestamp is invalid")
		}
		task.Attempts = append(task.Attempts, attempt)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := validateTaskRecord(task); err != nil {
		return nil, err
	}
	return task, nil
}

func validateTaskRecord(task *TaskRecord) error {
	created, err := time.Parse(time.RFC3339Nano, task.CreatedAt)
	if err != nil {
		return errors.New("Episode task creation timestamp is invalid")
	}
	updated, err := time.Parse(time.RFC3339Nano, task.UpdatedAt)
	if err != nil || updated.Before(created) {
		return errors.New("Episode task update timestamp is invalid")
	}
	if task.AttemptCount != len(task.Attempts) || task.AttemptCount < 0 || task.AttemptCount > task.MaxAttempts || task.MaxAttempts < 1 || task.MaxAttempts > limits.MaxEpisodeAttempts {
		return errors.New("Episode task attempt accounting is invalid")
	}
	for index, attempt := range task.Attempts {
		if attempt.EpisodeID != task.Episode.EpisodeID || attempt.Number != index+1 || attempt.FencingToken <= 0 || attempt.FencingToken > task.FencingToken {
			return errors.New("Episode attempt lineage is invalid")
		}
		attemptCreated, createErr := time.Parse(time.RFC3339Nano, attempt.CreatedAt)
		attemptUpdated, updateErr := time.Parse(time.RFC3339Nano, attempt.UpdatedAt)
		if createErr != nil || updateErr != nil || attemptUpdated.Before(attemptCreated) || attemptCreated.Before(created) {
			return errors.New("Episode attempt timestamp lineage is invalid")
		}
	}
	switch task.State {
	case TaskQueued:
		if task.ActiveAttemptID != "" || task.FinalAttemptID != "" || task.FinalEvidenceDigest != "" {
			return errors.New("queued Episode task contains active or terminal identity")
		}
	case TaskLeased:
		if task.ActiveAttemptID == "" || len(task.Attempts) == 0 {
			return errors.New("leased Episode task has no active attempt")
		}
		last := task.Attempts[len(task.Attempts)-1]
		if last.AttemptID != task.ActiveAttemptID || last.State != AttemptLeased || last.FencingToken != task.FencingToken {
			return errors.New("leased Episode task active attempt is inconsistent")
		}
	case TaskCompleted:
		if task.ActiveAttemptID != "" || task.FinalAttemptID == "" || task.FinalEvidenceDigest == "" || task.Episode.Evidence == nil || task.Episode.EvidenceDigest != task.FinalEvidenceDigest {
			return errors.New("completed Episode task terminal evidence is inconsistent")
		}
		if len(task.Attempts) == 0 {
			return errors.New("completed Episode task has no final attempt")
		}
		last := task.Attempts[len(task.Attempts)-1]
		if last.AttemptID != task.FinalAttemptID || last.State != AttemptCompleted || last.CommitState != CommitCommitted {
			return errors.New("completed Episode task final attempt is inconsistent")
		}
	case TaskCancelled, TaskFailed, TaskCommitUnknown:
		if task.ActiveAttemptID != "" || task.FinalAttemptID != "" || task.FinalEvidenceDigest != "" {
			return errors.New("terminal Episode task contains active or completed evidence identity")
		}
	default:
		return errors.New("Episode task state is invalid")
	}
	return nil
}

func requireOneRow(result sql.Result, mismatch error) error {
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed != 1 {
		return mismatch
	}
	return nil
}

func requestForEpisodeTx(ctx context.Context, tx *sql.Tx, episodeID string) (Request, error) {
	var request Request
	err := tx.QueryRowContext(ctx, `SELECT id, bundle_digest, scenario_path, runtime_version, runtime_revision FROM episodes WHERE id = ?`, episodeID).Scan(&request.EpisodeID, &request.BundleDigest, &request.ScenarioPath, &request.RuntimeVersion, &request.RuntimeRevision)
	return request, err
}

func validateRemoteEnvelope(request Request, envelope *Envelope) error {
	if envelope.APIVersion != APIVersion || envelope.Kind != "EpisodeEvidenceEnvelope" || envelope.Evidence == nil {
		return errors.New("Episode Evidence envelope identity is invalid")
	}
	evidence := envelope.Evidence
	if evidence.APIVersion != APIVersion || evidence.Kind != Kind || evidence.Format != EvidenceFormat || evidence.EpisodeID != request.EpisodeID || evidence.BundleDigest != request.BundleDigest || evidence.ScenarioPath != request.ScenarioPath || evidence.Runtime != (RuntimeIdentity{Version: request.RuntimeVersion, Revision: request.RuntimeRevision}) {
		return errors.New("Episode Evidence does not match remote task identity")
	}
	if evidence.Lifecycle.Current != StatusSucceeded && evidence.Lifecycle.Current != StatusAssertionFailed {
		return errors.New("remote completion requires successful or assertion-failed Evidence")
	}
	if len(evidence.Lifecycle.Events) != 6 || evidence.Lifecycle.Events[0] != (Event{Sequence: 0, To: StatusCreated}) {
		return errors.New("remote Episode lifecycle is invalid")
	}
	for index := 1; index < len(evidence.Lifecycle.Events); index++ {
		event := evidence.Lifecycle.Events[index]
		if event.Sequence != index || event.From != evidence.Lifecycle.Events[index-1].To || !allowedTransition(event.From, event.To) {
			return errors.New("remote Episode lifecycle transition chain is invalid")
		}
	}
	digest, err := evidence.Digest()
	if err != nil || digest != envelope.EvidenceDigest {
		return errors.New("remote Episode Evidence digest mismatch")
	}
	return nil
}

func cancelParentTx(ctx context.Context, tx *sql.Tx, episodeID, now string) error {
	result, err := tx.ExecContext(ctx, `UPDATE episodes SET status = ?, sequence = 1, error_class = 'CANCELLED', updated_at = ? WHERE id = ? AND status = ? AND sequence = 0`, StatusCancelled, now, episodeID, StatusCreated)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed == 0 {
		var status Status
		if err := tx.QueryRowContext(ctx, `SELECT status FROM episodes WHERE id = ?`, episodeID).Scan(&status); err != nil {
			return err
		}
		if status == StatusCancelled {
			return nil
		}
		return ErrEpisodeConflict
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO episode_events(episode_id, sequence, from_status, to_status, created_at) VALUES(?, 1, ?, ?, ?)`, episodeID, StatusCreated, StatusCancelled, now)
	return err
}

func newAttemptID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", fmt.Errorf("generate attempt ID: %w", err)
	}
	return "att_" + hex.EncodeToString(value[:]), nil
}
