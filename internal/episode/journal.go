package episode

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/augety121/mcp-state-twin/internal/bundle"
	"github.com/augety121/mcp-state-twin/internal/canonical"
	"github.com/augety121/mcp-state-twin/internal/limits"
	_ "modernc.org/sqlite"
)

const (
	journalApplicationID = 0x45504a4c // ASCII "EPJL"
	journalSchemaVersion = 2
	JournalFormat        = "statetwin.dev/episode-journal/v1alpha1"
)

var (
	ErrEpisodeNotFound   = errors.New("episode not found")
	ErrEpisodeConflict   = errors.New("episode identity conflict")
	ErrEpisodeIncomplete = errors.New("episode is incomplete")
	ErrEpisodeFailed     = errors.New("episode ended without reusable evidence")
)

type Journal struct {
	db *sql.DB
}

type journalMigrationStage string

const (
	journalMigrationSchemaApplied journalMigrationStage = "schema-applied"
	journalMigrationMetadataSet   journalMigrationStage = "metadata-set"
)

type journalMigrationHook func(journalMigrationStage) error

type Request struct {
	EpisodeID       string
	BundleDigest    string
	ScenarioPath    string
	RuntimeVersion  string
	RuntimeRevision string
}

type Record struct {
	APIVersion           string          `json:"apiVersion"`
	Kind                 string          `json:"kind"`
	Format               string          `json:"format"`
	StorageSchemaVersion int             `json:"storageSchemaVersion"`
	EpisodeID            string          `json:"episodeId"`
	RequestDigest        string          `json:"requestDigest"`
	BundleDigest         string          `json:"bundleDigest"`
	ScenarioPath         string          `json:"scenarioPath"`
	Runtime              RuntimeIdentity `json:"runtime"`
	Status               Status          `json:"status"`
	Sequence             int             `json:"sequence"`
	Incomplete           bool            `json:"incomplete"`
	Outcome              string          `json:"outcome,omitempty"`
	EvidenceDigest       string          `json:"evidenceDigest,omitempty"`
	Evidence             *Envelope       `json:"evidence,omitempty"`
	ErrorClass           string          `json:"errorClass,omitempty"`
	CreatedAt            string          `json:"createdAt"`
	UpdatedAt            string          `json:"updatedAt"`
	Events               []Event         `json:"events"`
}

func OpenJournal(path string) (*Journal, error) {
	return openJournalWithMigrationHook(path, nil)
}

func openJournalWithMigrationHook(path string, hook journalMigrationHook) (*Journal, error) {
	if path == "" {
		return nil, errors.New("episode journal path is required")
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open episode journal: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	journal := &Journal{db: db}
	if err := journal.migrate(hook); err != nil {
		_ = db.Close()
		return nil, err
	}
	for _, pragma := range []string{"PRAGMA foreign_keys = ON", "PRAGMA busy_timeout = 5000", "PRAGMA journal_mode = WAL"} {
		if _, err := db.Exec(pragma); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("configure episode journal: %w", err)
		}
	}
	return journal, nil
}

func (j *Journal) Close() error {
	if j == nil || j.db == nil {
		return nil
	}
	return j.db.Close()
}

func (j *Journal) migrate(hook journalMigrationHook) error {
	var applicationID, version int
	if err := j.db.QueryRow(`PRAGMA application_id`).Scan(&applicationID); err != nil {
		return fmt.Errorf("read episode journal application_id: %w", err)
	}
	if err := j.db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		return fmt.Errorf("read episode journal user_version: %w", err)
	}
	if applicationID != 0 && applicationID != journalApplicationID {
		return fmt.Errorf("SQLite application_id %d does not belong to an Episode Journal", applicationID)
	}
	if version > journalSchemaVersion {
		return fmt.Errorf("episode journal schema version %d is newer than supported version %d", version, journalSchemaVersion)
	}
	if applicationID == 0 {
		var tables int
		if err := j.db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'`).Scan(&tables); err != nil {
			return fmt.Errorf("inspect episode journal: %w", err)
		}
		if tables != 0 {
			return errors.New("unidentified non-empty SQLite database is not an Episode Journal")
		}
	}
	tx, err := j.db.Begin()
	if err != nil {
		return fmt.Errorf("begin episode journal migration: %w", err)
	}
	defer tx.Rollback()
	const schema = `
CREATE TABLE IF NOT EXISTS episodes (
  id TEXT PRIMARY KEY,
  request_digest TEXT NOT NULL,
  bundle_digest TEXT NOT NULL,
  scenario_path TEXT NOT NULL,
  runtime_version TEXT NOT NULL,
  runtime_revision TEXT NOT NULL,
  status TEXT NOT NULL,
  sequence INTEGER NOT NULL CHECK(sequence >= 0),
  outcome TEXT NOT NULL DEFAULT '',
  evidence_json BLOB,
  evidence_digest TEXT NOT NULL DEFAULT '',
  error_class TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  CHECK(status IN ('CREATED','PROVISIONING','READY','RUNNING','EVALUATING','SUCCEEDED','ASSERTION_FAILED','RUNTIME_ERROR','CANCELLED'))
);
CREATE TABLE IF NOT EXISTS episode_events (
  episode_id TEXT NOT NULL,
  sequence INTEGER NOT NULL CHECK(sequence >= 0),
  from_status TEXT NOT NULL DEFAULT '',
  to_status TEXT NOT NULL,
  created_at TEXT NOT NULL,
  PRIMARY KEY(episode_id, sequence),
  FOREIGN KEY(episode_id) REFERENCES episodes(id) ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS episode_tasks (
  episode_id TEXT PRIMARY KEY,
  bundle_bytes BLOB NOT NULL,
  effect_profile TEXT NOT NULL,
  max_attempts INTEGER NOT NULL CHECK(max_attempts >= 1 AND max_attempts <= 16),
  task_state TEXT NOT NULL,
  attempt_count INTEGER NOT NULL DEFAULT 0 CHECK(attempt_count >= 0 AND attempt_count <= 16),
  fencing_token INTEGER NOT NULL DEFAULT 0 CHECK(fencing_token >= 0),
  active_attempt_id TEXT NOT NULL DEFAULT '',
  cancel_requested INTEGER NOT NULL DEFAULT 0 CHECK(cancel_requested IN (0, 1)),
  final_attempt_id TEXT NOT NULL DEFAULT '',
  final_evidence_digest TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  CHECK(effect_profile IN ('hermetic', 'external')),
  CHECK(task_state IN ('QUEUED','LEASED','COMPLETED','CANCELLED','FAILED','COMMIT_UNKNOWN')),
  FOREIGN KEY(episode_id) REFERENCES episodes(id) ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS episode_attempts (
  attempt_id TEXT PRIMARY KEY,
  episode_id TEXT NOT NULL,
  attempt_number INTEGER NOT NULL CHECK(attempt_number >= 1 AND attempt_number <= 16),
  worker_id TEXT NOT NULL,
  fencing_token INTEGER NOT NULL CHECK(fencing_token >= 1),
  state TEXT NOT NULL,
  commit_state TEXT NOT NULL,
  lease_until TEXT NOT NULL,
  error_class TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE(episode_id, attempt_number),
  CHECK(state IN ('LEASED','COMPLETED','FAILED','CANCELLED','EXPIRED','COMMIT_UNKNOWN')),
  CHECK(commit_state IN ('NOT_STARTED','NO_EFFECT','COMMITTED','UNKNOWN')),
  FOREIGN KEY(episode_id) REFERENCES episodes(id) ON DELETE CASCADE
);`
	if _, err := tx.Exec(schema); err != nil {
		return fmt.Errorf("create episode journal schema: %w", err)
	}
	if hook != nil {
		if err := hook(journalMigrationSchemaApplied); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(fmt.Sprintf(`PRAGMA application_id = %d`, journalApplicationID)); err != nil {
		return fmt.Errorf("set episode journal application_id: %w", err)
	}
	if _, err := tx.Exec(fmt.Sprintf(`PRAGMA user_version = %d`, journalSchemaVersion)); err != nil {
		return fmt.Errorf("set episode journal user_version: %w", err)
	}
	if hook != nil {
		if err := hook(journalMigrationMetadataSet); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit episode journal migration: %w", err)
	}
	return nil
}

func resolveRequest(artifact *bundle.Artifact, episodeID, scenarioPath, runtimeVersion, runtimeRevision string) (Request, error) {
	if artifact == nil {
		return Request{}, errors.New("TwinBundle artifact is required")
	}
	if !episodeIDPattern.MatchString(episodeID) {
		return Request{}, errors.New("episode ID must match ^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$")
	}
	if runtimeVersion == "" {
		return Request{}, errors.New("runtime version is required")
	}
	if runtimeRevision == "" {
		runtimeRevision = "unknown"
	}
	selected, err := selectScenario(artifact.Manifest, scenarioPath)
	if err != nil {
		return Request{}, err
	}
	return Request{EpisodeID: episodeID, BundleDigest: artifact.Digest, ScenarioPath: selected, RuntimeVersion: runtimeVersion, RuntimeRevision: runtimeRevision}, nil
}

func (request Request) Digest() (string, error) {
	return canonical.Digest(map[string]any{
		"format": JournalFormat, "episodeId": request.EpisodeID,
		"bundleDigest": request.BundleDigest, "scenarioPath": request.ScenarioPath,
		"runtimeVersion": request.RuntimeVersion, "runtimeRevision": request.RuntimeRevision,
	})
}

// RunPersistent reserves one immutable Episode identity, persists every
// non-terminal lifecycle transition, and atomically attaches terminal evidence.
// Repeating the exact completed request returns the existing evidence without
// executing the Scenario again. An existing non-terminal record is surfaced as
// incomplete and is never silently retried.
func (j *Journal) RunPersistent(ctx context.Context, artifact *bundle.Artifact, episodeID, scenarioPath, runtimeVersion, runtimeRevision string) (*Envelope, bool, error) {
	request, err := resolveRequest(artifact, episodeID, scenarioPath, runtimeVersion, runtimeRevision)
	if err != nil {
		return nil, false, err
	}
	record, created, err := j.createOrLoad(ctx, request)
	if err != nil {
		return nil, false, err
	}
	if !created {
		if record.Evidence != nil && (record.Status == StatusSucceeded || record.Status == StatusAssertionFailed) {
			return record.Evidence, true, nil
		}
		if !IsTerminal(record.Status) {
			return nil, true, fmt.Errorf("%w: %s is %s at sequence %d", ErrEpisodeIncomplete, record.EpisodeID, record.Status, record.Sequence)
		}
		return nil, true, fmt.Errorf("%w: %s ended as %s", ErrEpisodeFailed, record.EpisodeID, record.Status)
	}

	sequence := 0
	observer := func(event Event) error {
		if IsTerminal(event.To) {
			return nil
		}
		if err := j.appendTransition(ctx, request.EpisodeID, request, sequence, event); err != nil {
			return err
		}
		sequence = event.Sequence
		return nil
	}
	envelope, runErr := run(ctx, artifact, request.EpisodeID, request.ScenarioPath, request.RuntimeVersion, request.RuntimeRevision, observer)
	if runErr != nil {
		_ = j.fail(ctx, request.EpisodeID, request, sequence, "RUNTIME_ERROR")
		return nil, false, runErr
	}
	if err := j.complete(ctx, request, sequence, envelope); err != nil {
		return nil, false, err
	}
	return envelope, false, nil
}

func (j *Journal) createOrLoad(ctx context.Context, request Request) (*Record, bool, error) {
	digest, err := request.Digest()
	if err != nil {
		return nil, false, err
	}
	tx, err := j.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, false, fmt.Errorf("begin Episode reservation: %w", err)
	}
	defer tx.Rollback()
	record, err := scanRecord(tx.QueryRowContext(ctx, `SELECT id, request_digest, bundle_digest, scenario_path, runtime_version, runtime_revision, status, sequence, outcome, evidence_json, evidence_digest, error_class, created_at, updated_at FROM episodes WHERE id = ?`, request.EpisodeID))
	if err == nil {
		if record.RequestDigest != digest {
			return nil, false, fmt.Errorf("%w: Episode ID %q is already bound to a different request", ErrEpisodeConflict, request.EpisodeID)
		}
		if err := loadEvents(ctx, tx, record); err != nil {
			return nil, false, err
		}
		return record, false, nil
	}
	if !errors.Is(err, ErrEpisodeNotFound) {
		return nil, false, err
	}
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM episodes`).Scan(&count); err != nil {
		return nil, false, fmt.Errorf("count Episode records: %w", err)
	}
	if count >= limits.MaxEpisodeRecords {
		return nil, false, fmt.Errorf("RESOURCE_LIMIT: Episode record limit is %d", limits.MaxEpisodeRecords)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := tx.ExecContext(ctx, `INSERT INTO episodes(id, request_digest, bundle_digest, scenario_path, runtime_version, runtime_revision, status, sequence, created_at, updated_at) VALUES(?, ?, ?, ?, ?, ?, ?, 0, ?, ?)`, request.EpisodeID, digest, request.BundleDigest, request.ScenarioPath, request.RuntimeVersion, request.RuntimeRevision, StatusCreated, now, now); err != nil {
		return nil, false, fmt.Errorf("reserve Episode: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO episode_events(episode_id, sequence, from_status, to_status, created_at) VALUES(?, 0, '', ?, ?)`, request.EpisodeID, StatusCreated, now); err != nil {
		return nil, false, fmt.Errorf("record Episode creation: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, false, fmt.Errorf("commit Episode reservation: %w", err)
	}
	return &Record{APIVersion: APIVersion, Kind: "EpisodeRecord", Format: JournalFormat, StorageSchemaVersion: journalSchemaVersion, EpisodeID: request.EpisodeID, RequestDigest: digest, BundleDigest: request.BundleDigest, ScenarioPath: request.ScenarioPath, Runtime: RuntimeIdentity{Version: request.RuntimeVersion, Revision: request.RuntimeRevision}, Status: StatusCreated, Sequence: 0, Incomplete: true, CreatedAt: now, UpdatedAt: now, Events: []Event{{Sequence: 0, To: StatusCreated}}}, true, nil
}

func (j *Journal) appendTransition(ctx context.Context, episodeID string, request Request, expectedSequence int, event Event) error {
	if event.Sequence != expectedSequence+1 || !allowedTransition(event.From, event.To) || IsTerminal(event.To) {
		return fmt.Errorf("invalid durable Episode transition %+v", event)
	}
	tx, err := j.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	requestDigest, err := request.Digest()
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `UPDATE episodes SET status = ?, sequence = ?, updated_at = ? WHERE id = ? AND request_digest = ? AND status = ? AND sequence = ?`, event.To, event.Sequence, time.Now().UTC().Format(time.RFC3339Nano), episodeID, requestDigest, event.From, expectedSequence)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil || changed != 1 {
		return fmt.Errorf("%w: Episode %s transition lost compare-and-swap", ErrEpisodeConflict, episodeID)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO episode_events(episode_id, sequence, from_status, to_status, created_at) VALUES(?, ?, ?, ?, ?)`, episodeID, event.Sequence, event.From, event.To, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		return err
	}
	return tx.Commit()
}

func (j *Journal) complete(ctx context.Context, request Request, expectedSequence int, envelope *Envelope) error {
	if envelope == nil || envelope.Evidence == nil || !IsTerminal(envelope.Evidence.Lifecycle.Current) || envelope.Evidence.Lifecycle.Current == StatusRuntimeError || envelope.Evidence.Lifecycle.Current == StatusCancelled {
		return errors.New("completed Episode evidence is invalid")
	}
	if envelope.Evidence.EpisodeID != request.EpisodeID || envelope.Evidence.BundleDigest != request.BundleDigest || envelope.Evidence.ScenarioPath != request.ScenarioPath {
		return errors.New("completed Episode evidence identity mismatch")
	}
	actualDigest, err := envelope.Evidence.Digest()
	if err != nil || actualDigest != envelope.EvidenceDigest {
		return errors.New("completed Episode evidence digest mismatch")
	}
	encoded, err := canonical.JSON(envelope)
	if err != nil {
		return err
	}
	if len(encoded) > limits.MaxReportBytes {
		return fmt.Errorf("RESOURCE_LIMIT: Episode evidence exceeds %d bytes", limits.MaxReportBytes)
	}
	terminal := envelope.Evidence.Lifecycle.Current
	event := envelope.Evidence.Lifecycle.Events[len(envelope.Evidence.Lifecycle.Events)-1]
	if event.Sequence != expectedSequence+1 || event.To != terminal || event.From != StatusEvaluating {
		return errors.New("completed Episode lifecycle does not match durable journal")
	}
	tx, err := j.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	requestDigest, err := request.Digest()
	if err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	result, err := tx.ExecContext(ctx, `UPDATE episodes SET status = ?, sequence = ?, outcome = ?, evidence_json = ?, evidence_digest = ?, updated_at = ? WHERE id = ? AND request_digest = ? AND status = ? AND sequence = ?`, terminal, event.Sequence, envelope.Evidence.Outcome, encoded, envelope.EvidenceDigest, now, request.EpisodeID, requestDigest, StatusEvaluating, expectedSequence)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil || changed != 1 {
		return fmt.Errorf("%w: Episode %s completion lost compare-and-swap", ErrEpisodeConflict, request.EpisodeID)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO episode_events(episode_id, sequence, from_status, to_status, created_at) VALUES(?, ?, ?, ?, ?)`, request.EpisodeID, event.Sequence, event.From, event.To, now); err != nil {
		return err
	}
	return tx.Commit()
}

func (j *Journal) fail(ctx context.Context, episodeID string, request Request, expectedSequence int, errorClass string) error {
	tx, err := j.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	requestDigest, err := request.Digest()
	if err != nil {
		return err
	}
	var current Status
	var sequence int
	if err := tx.QueryRowContext(ctx, `SELECT status, sequence FROM episodes WHERE id = ? AND request_digest = ?`, episodeID, requestDigest).Scan(&current, &sequence); err != nil {
		return err
	}
	if sequence != expectedSequence || IsTerminal(current) || !allowedTransition(current, StatusRuntimeError) {
		return nil
	}
	nextSequence := sequence + 1
	now := time.Now().UTC().Format(time.RFC3339Nano)
	result, err := tx.ExecContext(ctx, `UPDATE episodes SET status = ?, sequence = ?, error_class = ?, updated_at = ? WHERE id = ? AND request_digest = ? AND status = ? AND sequence = ?`, StatusRuntimeError, nextSequence, errorClass, now, episodeID, requestDigest, current, sequence)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil || changed != 1 {
		return fmt.Errorf("%w: Episode %s failure transition lost compare-and-swap", ErrEpisodeConflict, episodeID)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO episode_events(episode_id, sequence, from_status, to_status, created_at) VALUES(?, ?, ?, ?, ?)`, episodeID, nextSequence, current, StatusRuntimeError, now); err != nil {
		return err
	}
	return tx.Commit()
}

func (j *Journal) Get(ctx context.Context, episodeID string) (*Record, error) {
	if !episodeIDPattern.MatchString(episodeID) {
		return nil, errors.New("episode ID must match ^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$")
	}
	record, err := scanRecord(j.db.QueryRowContext(ctx, `SELECT id, request_digest, bundle_digest, scenario_path, runtime_version, runtime_revision, status, sequence, outcome, evidence_json, evidence_digest, error_class, created_at, updated_at FROM episodes WHERE id = ?`, episodeID))
	if err != nil {
		return nil, err
	}
	if err := loadEvents(ctx, j.db, record); err != nil {
		return nil, err
	}
	return record, nil
}

type rowScanner interface {
	Scan(...any) error
}

type queryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func scanRecord(row rowScanner) (*Record, error) {
	var record Record
	var evidenceJSON []byte
	if err := row.Scan(&record.EpisodeID, &record.RequestDigest, &record.BundleDigest, &record.ScenarioPath, &record.Runtime.Version, &record.Runtime.Revision, &record.Status, &record.Sequence, &record.Outcome, &evidenceJSON, &record.EvidenceDigest, &record.ErrorClass, &record.CreatedAt, &record.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrEpisodeNotFound
		}
		return nil, fmt.Errorf("read Episode record: %w", err)
	}
	record.APIVersion = APIVersion
	record.Kind = "EpisodeRecord"
	record.Format = JournalFormat
	record.StorageSchemaVersion = journalSchemaVersion
	record.Incomplete = !IsTerminal(record.Status)
	createdAt, err := time.Parse(time.RFC3339Nano, record.CreatedAt)
	if err != nil {
		return nil, errors.New("Episode createdAt is invalid")
	}
	updatedAt, err := time.Parse(time.RFC3339Nano, record.UpdatedAt)
	if err != nil {
		return nil, errors.New("Episode updatedAt is invalid")
	}
	if updatedAt.Before(createdAt) {
		return nil, errors.New("Episode operational timestamps are inconsistent")
	}
	requestDigest, err := (Request{EpisodeID: record.EpisodeID, BundleDigest: record.BundleDigest, ScenarioPath: record.ScenarioPath, RuntimeVersion: record.Runtime.Version, RuntimeRevision: record.Runtime.Revision}).Digest()
	if err != nil || requestDigest != record.RequestDigest {
		return nil, errors.New("Episode request digest mismatch")
	}
	if len(evidenceJSON) != 0 {
		if err := json.Unmarshal(evidenceJSON, &record.Evidence); err != nil {
			return nil, fmt.Errorf("decode Episode evidence: %w", err)
		}
		if record.Evidence == nil || record.Evidence.Evidence == nil ||
			record.Evidence.APIVersion != APIVersion || record.Evidence.Kind != "EpisodeEvidenceEnvelope" ||
			record.Evidence.EvidenceDigest != record.EvidenceDigest {
			return nil, errors.New("Episode evidence identity mismatch")
		}
		evidence := record.Evidence.Evidence
		if evidence.APIVersion != APIVersion || evidence.Kind != Kind || evidence.Format != EvidenceFormat ||
			evidence.EpisodeID != record.EpisodeID || evidence.BundleDigest != record.BundleDigest ||
			evidence.ScenarioPath != record.ScenarioPath || evidence.Runtime != record.Runtime ||
			evidence.Outcome != record.Outcome {
			return nil, errors.New("Episode evidence does not match Journal identity")
		}
		actual, err := record.Evidence.Evidence.Digest()
		if err != nil || actual != record.EvidenceDigest {
			return nil, errors.New("Episode evidence digest mismatch")
		}
		if record.Status != StatusSucceeded && record.Status != StatusAssertionFailed {
			return nil, errors.New("Episode status cannot carry terminal Evidence")
		}
		expectedOutcome := "succeeded"
		if record.Status == StatusAssertionFailed {
			expectedOutcome = "assertion_failed"
		}
		if record.Outcome != expectedOutcome || record.ErrorClass != "" {
			return nil, errors.New("Episode terminal outcome is inconsistent")
		}
	} else if record.Status == StatusSucceeded || record.Status == StatusAssertionFailed || record.EvidenceDigest != "" {
		return nil, errors.New("terminal Episode record is missing Evidence")
	} else if record.Outcome != "" {
		return nil, errors.New("Episode without Evidence cannot carry an outcome")
	} else if record.Status == StatusRuntimeError && record.ErrorClass != "RUNTIME_ERROR" {
		return nil, errors.New("runtime-error Episode is missing its typed error class")
	} else if !IsTerminal(record.Status) && record.ErrorClass != "" {
		return nil, errors.New("incomplete Episode cannot carry an error class")
	}
	return &record, nil
}

func loadEvents(ctx context.Context, source queryer, record *Record) error {
	rows, err := source.QueryContext(ctx, `SELECT sequence, from_status, to_status FROM episode_events WHERE episode_id = ? ORDER BY sequence`, record.EpisodeID)
	if err != nil {
		return fmt.Errorf("read Episode lifecycle: %w", err)
	}
	defer rows.Close()
	record.Events = make([]Event, 0, record.Sequence+1)
	for rows.Next() {
		var event Event
		if err := rows.Scan(&event.Sequence, &event.From, &event.To); err != nil {
			return fmt.Errorf("scan Episode lifecycle: %w", err)
		}
		record.Events = append(record.Events, event)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(record.Events) != record.Sequence+1 || len(record.Events) == 0 || record.Events[len(record.Events)-1].To != record.Status {
		return errors.New("Episode lifecycle journal is inconsistent")
	}
	for index, event := range record.Events {
		if event.Sequence != index {
			return errors.New("Episode lifecycle sequence is inconsistent")
		}
		if index == 0 {
			if event.From != "" || event.To != StatusCreated {
				return errors.New("Episode lifecycle creation event is invalid")
			}
			continue
		}
		previous := record.Events[index-1]
		if event.From != previous.To || !allowedTransition(event.From, event.To) {
			return errors.New("Episode lifecycle transition chain is invalid")
		}
	}
	if record.Evidence != nil {
		lifecycle := record.Evidence.Evidence.Lifecycle
		if lifecycle.Current != record.Status || !equalEvents(lifecycle.Events, record.Events) {
			return errors.New("Episode Evidence lifecycle does not match Journal")
		}
	}
	return nil
}

func equalEvents(left, right []Event) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
