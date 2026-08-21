package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/augety121/mcp-state-twin/internal/canonical"
	"github.com/augety121/mcp-state-twin/internal/limits"
	"github.com/augety121/mcp-state-twin/internal/world"
	_ "modernc.org/sqlite"
)

var (
	ErrBranchNotFound   = errors.New("branch not found")
	ErrSnapshotNotFound = errors.New("snapshot not found")
	ErrBranchConflict   = errors.New("branch head conflict")
	ErrClockRegression  = errors.New("virtual clock must move forward")
	ErrClockLimit       = errors.New("virtual clock advance exceeds limit")
	ErrFaultNotFound    = errors.New("fault plan not found")
	ErrFaultInvalid     = errors.New("invalid fault plan")
	ErrResourceLimit    = errors.New("resource limit exceeded")
)

var idPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

const (
	applicationID   = 0x5354574e // ASCII "STWN"
	schemaVersion   = 4
	MaxClockAdvance = 10 * 365 * 24 * time.Hour
	MaxFaultPlans   = limits.MaxFaultRules
	MaxFaultRepeats = 1000
	MaxFaultMessage = 4096
)

const (
	FaultPhaseBeforeValidation          = "before-validation"
	FaultPhaseAfterCommitBeforeResponse = "after-commit-before-response"
)

type Store struct {
	db *sql.DB
}

type Branch struct {
	ID          string       `json:"id"`
	SpecDigest  string       `json:"specDigest"`
	State       *world.State `json:"state"`
	StateDigest string       `json:"stateDigest"`
	Clock       time.Time    `json:"clock"`
	CallCount   int64        `json:"callCount"`
	HeadVersion int64        `json:"headVersion"`
}

type Snapshot struct {
	ID                   string       `json:"id"`
	Name                 string       `json:"name"`
	SpecDigest           string       `json:"specDigest"`
	State                *world.State `json:"state"`
	StateDigest          string       `json:"stateDigest"`
	Clock                time.Time    `json:"clock"`
	SourceHeadVersion    int64        `json:"sourceHeadVersion"`
	StorageSchemaVersion int          `json:"storageSchemaVersion"`
}

type CallOutcome struct {
	Result      any
	ErrorClass  string
	CommitState bool
}

type ApplyResult struct {
	Result       any    `json:"result"`
	ErrorClass   string `json:"errorClass,omitempty"`
	BeforeDigest string `json:"beforeDigest"`
	AfterDigest  string `json:"afterDigest"`
	CallIndex    int64  `json:"callIndex"`
	FaultID      string `json:"faultId,omitempty"`
	FaultPhase   string `json:"faultPhase,omitempty"`
}

type FaultPlan struct {
	ID             string `json:"id"`
	BranchID       string `json:"branchId"`
	ToolName       string `json:"toolName"`
	Phase          string `json:"phase"`
	ErrorClass     string `json:"errorClass"`
	Message        string `json:"message"`
	RemainingCount int64  `json:"remainingCount"`
	FiredCount     int64  `json:"firedCount"`
	CreatedAt      string `json:"createdAt"`
}

type FaultEvent struct {
	ID           int64  `json:"id"`
	BranchID     string `json:"branchId"`
	FaultID      string `json:"faultId"`
	CallIndex    int64  `json:"callIndex"`
	Phase        string `json:"phase"`
	ErrorClass   string `json:"errorClass"`
	BeforeDigest string `json:"beforeDigest"`
	AfterDigest  string `json:"afterDigest"`
	CreatedAt    string `json:"createdAt"`
}

type ControlAuditEntry struct {
	ID           int64  `json:"id"`
	Operation    string `json:"operation"`
	BranchID     string `json:"branchId,omitempty"`
	SnapshotName string `json:"snapshotName,omitempty"`
	BeforeDigest string `json:"beforeDigest,omitempty"`
	AfterDigest  string `json:"afterDigest,omitempty"`
	CreatedAt    string `json:"createdAt"`
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open SQLite: %w", err)
	}
	// v0.1 uses deterministic serial transitions. A single connection also
	// keeps :memory: databases coherent across all calls.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	for _, pragma := range []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA busy_timeout = 5000",
		"PRAGMA journal_mode = WAL",
	} {
		if _, err := db.Exec(pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("configure SQLite: %w", err)
		}
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate() error {
	var appID int
	if err := s.db.QueryRow(`PRAGMA application_id`).Scan(&appID); err != nil {
		return fmt.Errorf("read SQLite application_id: %w", err)
	}
	if appID != 0 && appID != applicationID {
		return fmt.Errorf("database application_id %d does not belong to MCP State Twin", appID)
	}
	var version int
	if err := s.db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		return fmt.Errorf("read SQLite user_version: %w", err)
	}
	if version > schemaVersion {
		return fmt.Errorf("database schema version %d is newer than supported version %d", version, schemaVersion)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin SQLite migration: %w", err)
	}
	defer tx.Rollback()
	const schema = `
CREATE TABLE IF NOT EXISTS branches (
  id TEXT PRIMARY KEY,
  spec_digest TEXT NOT NULL,
  state_json BLOB NOT NULL,
  state_digest TEXT NOT NULL,
  clock TEXT NOT NULL,
  call_count INTEGER NOT NULL DEFAULT 0,
  head_version INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS snapshots (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL UNIQUE,
  spec_digest TEXT NOT NULL,
  state_json BLOB NOT NULL,
  state_digest TEXT NOT NULL,
  clock TEXT NOT NULL,
  source_head_version INTEGER NOT NULL DEFAULT 0,
  storage_schema_version INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS audit (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  branch_id TEXT NOT NULL,
  call_index INTEGER NOT NULL,
  tool_name TEXT NOT NULL,
  input_json BLOB NOT NULL,
  result_json BLOB NOT NULL,
  error_class TEXT NOT NULL,
  before_digest TEXT NOT NULL,
  after_digest TEXT NOT NULL,
  created_at TEXT NOT NULL,
  FOREIGN KEY(branch_id) REFERENCES branches(id) ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS control_audit (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  operation TEXT NOT NULL,
  branch_id TEXT NOT NULL DEFAULT '',
  snapshot_name TEXT NOT NULL DEFAULT '',
  before_digest TEXT NOT NULL DEFAULT '',
  after_digest TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS fault_plans (
  id TEXT NOT NULL,
  branch_id TEXT NOT NULL,
  tool_name TEXT NOT NULL,
  phase TEXT NOT NULL,
  error_class TEXT NOT NULL,
  message TEXT NOT NULL,
  remaining_count INTEGER NOT NULL CHECK(remaining_count >= 0 AND remaining_count <= 1000),
  fired_count INTEGER NOT NULL DEFAULT 0 CHECK(fired_count >= 0 AND fired_count <= 1000),
  created_at TEXT NOT NULL,
  CHECK(remaining_count + fired_count >= 1 AND remaining_count + fired_count <= 1000),
  CHECK(phase IN ('before-validation', 'after-commit-before-response')),
  CHECK(error_class IN ('RATE_LIMITED', 'TIMEOUT_BEFORE_EFFECT', 'TIMEOUT_AFTER_EFFECT')),
  CHECK((phase = 'after-commit-before-response' AND error_class = 'TIMEOUT_AFTER_EFFECT') OR
        (phase = 'before-validation' AND error_class IN ('RATE_LIMITED', 'TIMEOUT_BEFORE_EFFECT'))),
  PRIMARY KEY(branch_id, id),
  FOREIGN KEY(branch_id) REFERENCES branches(id) ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS fault_events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  branch_id TEXT NOT NULL,
  fault_id TEXT NOT NULL,
  call_index INTEGER NOT NULL,
  phase TEXT NOT NULL,
  error_class TEXT NOT NULL,
  before_digest TEXT NOT NULL,
  after_digest TEXT NOT NULL,
  created_at TEXT NOT NULL,
  FOREIGN KEY(branch_id) REFERENCES branches(id) ON DELETE CASCADE
);`
	if _, err := tx.Exec(schema); err != nil {
		return fmt.Errorf("migrate SQLite schema: %w", err)
	}
	hasStorageSchemaVersion, err := tableHasColumn(tx, "snapshots", "storage_schema_version")
	if err != nil {
		return err
	}
	if !hasStorageSchemaVersion {
		if _, err := tx.Exec(`ALTER TABLE snapshots ADD COLUMN storage_schema_version INTEGER NOT NULL DEFAULT 1`); err != nil {
			return fmt.Errorf("add snapshot storage schema version: %w", err)
		}
	}
	hasSourceHeadVersion, err := tableHasColumn(tx, "snapshots", "source_head_version")
	if err != nil {
		return err
	}
	if !hasSourceHeadVersion {
		if _, err := tx.Exec(`ALTER TABLE snapshots ADD COLUMN source_head_version INTEGER NOT NULL DEFAULT 0`); err != nil {
			return fmt.Errorf("add snapshot source head version: %w", err)
		}
	}
	hasBranchHeadVersion, err := tableHasColumn(tx, "branches", "head_version")
	if err != nil {
		return err
	}
	if !hasBranchHeadVersion {
		if _, err := tx.Exec(`ALTER TABLE branches ADD COLUMN head_version INTEGER NOT NULL DEFAULT 0`); err != nil {
			return fmt.Errorf("add branch head version: %w", err)
		}
	}
	// Early development databases briefly used a uniqueness constraint on
	// (branch_id, call_index). Reset intentionally rewinds call_index while the
	// audit ledger remains append-only, so that legacy constraint must be
	// removed without discarding records.
	var auditSQL string
	if err := tx.QueryRow(`SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'audit'`).Scan(&auditSQL); err != nil {
		return fmt.Errorf("inspect audit schema: %w", err)
	}
	compactAuditSQL := strings.ToLower(strings.Join(strings.Fields(auditSQL), ""))
	if strings.Contains(compactAuditSQL, "unique(branch_id,call_index)") {
		if _, err := tx.Exec(`
ALTER TABLE audit RENAME TO audit_legacy;
CREATE TABLE audit (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  branch_id TEXT NOT NULL,
  call_index INTEGER NOT NULL,
  tool_name TEXT NOT NULL,
  input_json BLOB NOT NULL,
  result_json BLOB NOT NULL,
  error_class TEXT NOT NULL,
  before_digest TEXT NOT NULL,
  after_digest TEXT NOT NULL,
  created_at TEXT NOT NULL,
  FOREIGN KEY(branch_id) REFERENCES branches(id) ON DELETE CASCADE
);
INSERT INTO audit(id, branch_id, call_index, tool_name, input_json, result_json, error_class, before_digest, after_digest, created_at)
SELECT id, branch_id, call_index, tool_name, input_json, result_json, error_class, before_digest, after_digest, created_at FROM audit_legacy;
DROP TABLE audit_legacy;`); err != nil {
			return fmt.Errorf("upgrade legacy audit schema: %w", err)
		}
	}
	if _, err := tx.Exec(fmt.Sprintf(`PRAGMA application_id = %d`, applicationID)); err != nil {
		return fmt.Errorf("set SQLite application_id: %w", err)
	}
	if _, err := tx.Exec(fmt.Sprintf(`PRAGMA user_version = %d`, schemaVersion)); err != nil {
		return fmt.Errorf("set SQLite user_version: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit SQLite migration: %w", err)
	}
	return nil
}

func tableHasColumn(tx *sql.Tx, table, column string) (bool, error) {
	rows, err := tx.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return false, fmt.Errorf("inspect %s schema: %w", table, err)
	}
	defer rows.Close()
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, columnType string
		var defaultValue any
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			return false, fmt.Errorf("scan %s schema: %w", table, err)
		}
		if name == column {
			return true, nil
		}
	}
	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("iterate %s schema: %w", table, err)
	}
	return false, nil
}

func validateID(kind, value string) error {
	if !idPattern.MatchString(value) {
		return fmt.Errorf("invalid %s %q", kind, value)
	}
	return nil
}

func (s *Store) InitializeBranch(ctx context.Context, id, specDigest string, state *world.State, clock time.Time) error {
	if err := validateID("branch ID", id); err != nil {
		return err
	}
	if err := state.ValidateBudget(); err != nil {
		return fmt.Errorf("%w: %v", ErrResourceLimit, err)
	}
	stateJSON, err := canonical.JSON(state)
	if err != nil {
		return err
	}
	stateDigest, err := canonical.Digest(state)
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin branch initialization: %w", err)
	}
	defer tx.Rollback()
	var exists int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM branches WHERE id = ?`, id).Scan(&exists); err != nil {
		return fmt.Errorf("check branch identity: %w", err)
	}
	if exists != 0 {
		return nil
	}
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM branches`).Scan(&count); err != nil {
		return fmt.Errorf("count branches: %w", err)
	}
	if count >= limits.MaxForks {
		return fmt.Errorf("%w: branch limit is %d", ErrResourceLimit, limits.MaxForks)
	}
	_, err = tx.ExecContext(ctx, `
INSERT INTO branches(id, spec_digest, state_json, state_digest, clock, call_count, head_version)
VALUES(?, ?, ?, ?, ?, 0, 0)`, id, specDigest, stateJSON, stateDigest, clock.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("initialize branch: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit branch initialization: %w", err)
	}
	return nil
}

func (s *Store) Branch(ctx context.Context, id string) (*Branch, error) {
	row := s.db.QueryRowContext(ctx, `SELECT spec_digest, state_json, state_digest, clock, call_count, head_version FROM branches WHERE id = ?`, id)
	return scanBranch(id, row)
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanBranch(id string, row rowScanner) (*Branch, error) {
	var specDigest, stateDigest, clockRaw string
	var stateJSON []byte
	var callCount, headVersion int64
	if err := row.Scan(&specDigest, &stateJSON, &stateDigest, &clockRaw, &callCount, &headVersion); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrBranchNotFound
		}
		return nil, fmt.Errorf("read branch: %w", err)
	}
	clock, err := time.Parse(time.RFC3339Nano, clockRaw)
	if err != nil {
		return nil, fmt.Errorf("parse branch clock: %w", err)
	}
	var state world.State
	if err := json.Unmarshal(stateJSON, &state); err != nil {
		return nil, fmt.Errorf("decode branch state: %w", err)
	}
	state.Normalize()
	return &Branch{ID: id, SpecDigest: specDigest, State: &state, StateDigest: stateDigest, Clock: clock, CallCount: callCount, HeadVersion: headVersion}, nil
}

func validateFaultPlan(plan FaultPlan) error {
	if err := validateID("fault ID", plan.ID); err != nil {
		return fmt.Errorf("%w: %v", ErrFaultInvalid, err)
	}
	if err := validateID("branch ID", plan.BranchID); err != nil {
		return fmt.Errorf("%w: %v", ErrFaultInvalid, err)
	}
	if err := validateID("tool name", plan.ToolName); err != nil {
		return fmt.Errorf("%w: %v", ErrFaultInvalid, err)
	}
	switch plan.Phase {
	case FaultPhaseBeforeValidation, FaultPhaseAfterCommitBeforeResponse:
	default:
		return fmt.Errorf("%w: unsupported phase %q", ErrFaultInvalid, plan.Phase)
	}
	switch plan.ErrorClass {
	case "RATE_LIMITED", "TIMEOUT_BEFORE_EFFECT", "TIMEOUT_AFTER_EFFECT":
	default:
		return fmt.Errorf("%w: unsupported error class %q", ErrFaultInvalid, plan.ErrorClass)
	}
	if plan.Phase == FaultPhaseBeforeValidation && plan.ErrorClass == "TIMEOUT_AFTER_EFFECT" {
		return fmt.Errorf("%w: TIMEOUT_AFTER_EFFECT requires phase %s", ErrFaultInvalid, FaultPhaseAfterCommitBeforeResponse)
	}
	if plan.Phase == FaultPhaseAfterCommitBeforeResponse && plan.ErrorClass != "TIMEOUT_AFTER_EFFECT" {
		return fmt.Errorf("%w: phase %s requires TIMEOUT_AFTER_EFFECT", ErrFaultInvalid, FaultPhaseAfterCommitBeforeResponse)
	}
	if plan.RemainingCount < 1 || plan.RemainingCount > MaxFaultRepeats {
		return fmt.Errorf("%w: remainingCount must be between 1 and %d", ErrFaultInvalid, MaxFaultRepeats)
	}
	if strings.TrimSpace(plan.Message) == "" || len(plan.Message) > MaxFaultMessage {
		return fmt.Errorf("%w: message must contain 1..%d bytes", ErrFaultInvalid, MaxFaultMessage)
	}
	return nil
}

// InstallFault installs a bounded, branch-local deterministic plan. Installing
// a plan is a privileged environment mutation and therefore advances the
// monotonic branch head even though it does not change world state.
func (s *Store) InstallFault(ctx context.Context, plan FaultPlan, expectedHeadVersion *int64) (*FaultPlan, error) {
	if err := validateFaultPlan(plan); err != nil {
		return nil, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin fault install transaction: %w", err)
	}
	defer tx.Rollback()
	branch, err := scanBranch(plan.BranchID, tx.QueryRowContext(ctx, `SELECT spec_digest, state_json, state_digest, clock, call_count, head_version FROM branches WHERE id = ?`, plan.BranchID))
	if err != nil {
		return nil, err
	}
	if expectedHeadVersion != nil && *expectedHeadVersion != branch.HeadVersion {
		return nil, fmt.Errorf("%w: branch %s expected head %d, current %d", ErrBranchConflict, plan.BranchID, *expectedHeadVersion, branch.HeadVersion)
	}
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM fault_plans WHERE branch_id = ?`, plan.BranchID).Scan(&count); err != nil {
		return nil, fmt.Errorf("count fault plans: %w", err)
	}
	if count >= MaxFaultPlans {
		return nil, fmt.Errorf("%w: branch fault plan limit is %d", ErrFaultInvalid, MaxFaultPlans)
	}
	var exists int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM fault_plans WHERE branch_id = ? AND id = ?`, plan.BranchID, plan.ID).Scan(&exists); err != nil {
		return nil, fmt.Errorf("check fault plan identity: %w", err)
	}
	if exists != 0 {
		return nil, fmt.Errorf("%w: fault ID %q already exists on branch %s", ErrFaultInvalid, plan.ID, plan.BranchID)
	}
	plan.CreatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	_, err = tx.ExecContext(ctx, `
INSERT INTO fault_plans(id, branch_id, tool_name, phase, error_class, message, remaining_count, fired_count, created_at)
VALUES(?, ?, ?, ?, ?, ?, ?, 0, ?)`, plan.ID, plan.BranchID, plan.ToolName, plan.Phase, plan.ErrorClass, plan.Message, plan.RemainingCount, plan.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("install fault plan: %w", err)
	}
	updated, err := tx.ExecContext(ctx, `UPDATE branches SET head_version = head_version + 1 WHERE id = ? AND head_version = ?`, plan.BranchID, branch.HeadVersion)
	if err != nil {
		return nil, fmt.Errorf("advance branch head for fault install: %w", err)
	}
	rows, err := updated.RowsAffected()
	if err != nil || rows != 1 {
		return nil, fmt.Errorf("%w: branch %s expected head %d", ErrBranchConflict, plan.BranchID, branch.HeadVersion)
	}
	if err := appendControlAudit(ctx, tx, "fault.install", plan.BranchID, plan.ID, branch.StateDigest, branch.StateDigest); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit fault install: %w", err)
	}
	return &plan, nil
}

func (s *Store) RemoveFault(ctx context.Context, branchID, faultID string, expectedHeadVersion *int64) error {
	if err := validateID("branch ID", branchID); err != nil {
		return err
	}
	if err := validateID("fault ID", faultID); err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin fault removal transaction: %w", err)
	}
	defer tx.Rollback()
	branch, err := scanBranch(branchID, tx.QueryRowContext(ctx, `SELECT spec_digest, state_json, state_digest, clock, call_count, head_version FROM branches WHERE id = ?`, branchID))
	if err != nil {
		return err
	}
	if expectedHeadVersion != nil && *expectedHeadVersion != branch.HeadVersion {
		return fmt.Errorf("%w: branch %s expected head %d, current %d", ErrBranchConflict, branchID, *expectedHeadVersion, branch.HeadVersion)
	}
	result, err := tx.ExecContext(ctx, `DELETE FROM fault_plans WHERE branch_id = ? AND id = ?`, branchID, faultID)
	if err != nil {
		return fmt.Errorf("remove fault plan: %w", err)
	}
	removed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if removed != 1 {
		return ErrFaultNotFound
	}
	result, err = tx.ExecContext(ctx, `UPDATE branches SET head_version = head_version + 1 WHERE id = ? AND head_version = ?`, branchID, branch.HeadVersion)
	if err != nil {
		return fmt.Errorf("advance branch head for fault removal: %w", err)
	}
	updated, err := result.RowsAffected()
	if err != nil || updated != 1 {
		return fmt.Errorf("%w: branch %s expected head %d", ErrBranchConflict, branchID, branch.HeadVersion)
	}
	if err := appendControlAudit(ctx, tx, "fault.remove", branchID, faultID, branch.StateDigest, branch.StateDigest); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit fault removal: %w", err)
	}
	return nil
}

func (s *Store) FaultPlans(ctx context.Context, branchID string) ([]FaultPlan, error) {
	if _, err := s.Branch(ctx, branchID); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, branch_id, tool_name, phase, error_class, message, remaining_count, fired_count, created_at
FROM fault_plans WHERE branch_id = ? ORDER BY id`, branchID)
	if err != nil {
		return nil, fmt.Errorf("query fault plans: %w", err)
	}
	defer rows.Close()
	plans := make([]FaultPlan, 0)
	for rows.Next() {
		var plan FaultPlan
		if err := rows.Scan(&plan.ID, &plan.BranchID, &plan.ToolName, &plan.Phase, &plan.ErrorClass, &plan.Message, &plan.RemainingCount, &plan.FiredCount, &plan.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan fault plan: %w", err)
		}
		plans = append(plans, plan)
	}
	return plans, rows.Err()
}

func (s *Store) FaultPlanDigest(ctx context.Context, branchID string) (string, error) {
	plans, err := s.FaultPlans(ctx, branchID)
	if err != nil {
		return "", err
	}
	type identity struct {
		ID          string `json:"id"`
		ToolName    string `json:"toolName"`
		Phase       string `json:"phase"`
		ErrorClass  string `json:"errorClass"`
		Message     string `json:"message"`
		RepeatCount int64  `json:"repeatCount"`
	}
	identities := make([]identity, 0, len(plans))
	for _, plan := range plans {
		identities = append(identities, identity{
			ID: plan.ID, ToolName: plan.ToolName, Phase: plan.Phase,
			ErrorClass: plan.ErrorClass, Message: plan.Message,
			RepeatCount: plan.RemainingCount + plan.FiredCount,
		})
	}
	return canonical.Digest(map[string]any{"format": "statetwin.dev/fault-plan/v1alpha1", "plans": identities})
}

func (s *Store) FaultEvents(ctx context.Context, branchID string) ([]FaultEvent, error) {
	if _, err := s.Branch(ctx, branchID); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, branch_id, fault_id, call_index, phase, error_class, before_digest, after_digest, created_at
FROM fault_events WHERE branch_id = ? ORDER BY id`, branchID)
	if err != nil {
		return nil, fmt.Errorf("query fault events: %w", err)
	}
	defer rows.Close()
	events := make([]FaultEvent, 0)
	for rows.Next() {
		var event FaultEvent
		if err := rows.Scan(&event.ID, &event.BranchID, &event.FaultID, &event.CallIndex, &event.Phase, &event.ErrorClass, &event.BeforeDigest, &event.AfterDigest, &event.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan fault event: %w", err)
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func takeFault(ctx context.Context, tx *sql.Tx, branchID, toolName, phase string) (*FaultPlan, error) {
	var plan FaultPlan
	err := tx.QueryRowContext(ctx, `
SELECT id, branch_id, tool_name, phase, error_class, message, remaining_count, fired_count, created_at
FROM fault_plans
WHERE branch_id = ? AND tool_name = ? AND phase = ? AND remaining_count > 0
ORDER BY id LIMIT 1`, branchID, toolName, phase).Scan(
		&plan.ID, &plan.BranchID, &plan.ToolName, &plan.Phase, &plan.ErrorClass,
		&plan.Message, &plan.RemainingCount, &plan.FiredCount, &plan.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("select fault plan: %w", err)
	}
	result, err := tx.ExecContext(ctx, `
UPDATE fault_plans SET remaining_count = remaining_count - 1, fired_count = fired_count + 1
WHERE branch_id = ? AND id = ? AND remaining_count = ?`, branchID, plan.ID, plan.RemainingCount)
	if err != nil {
		return nil, fmt.Errorf("consume fault plan: %w", err)
	}
	updated, err := result.RowsAffected()
	if err != nil || updated != 1 {
		return nil, fmt.Errorf("consume fault plan %s atomically", plan.ID)
	}
	plan.RemainingCount--
	plan.FiredCount++
	return &plan, nil
}

func faultOutcome(plan *FaultPlan, commit bool) CallOutcome {
	return CallOutcome{
		Result:     map[string]any{"error": map[string]any{"code": plan.ErrorClass, "message": plan.Message}},
		ErrorClass: plan.ErrorClass, CommitState: commit,
	}
}

func (s *Store) ApplyCall(
	ctx context.Context,
	branchID, expectedSpecDigest, toolName string,
	input any,
	apply func(state *world.State, clock time.Time, callIndex int64) (CallOutcome, error),
) (*ApplyResult, error) {
	if err := limits.ValidateJSON(input, limits.MaxInputBytes); err != nil {
		return nil, fmt.Errorf("%w: tool input: %v", ErrResourceLimit, err)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transition: %w", err)
	}
	defer tx.Rollback()

	branch, err := scanBranch(branchID, tx.QueryRowContext(ctx, `SELECT spec_digest, state_json, state_digest, clock, call_count, head_version FROM branches WHERE id = ?`, branchID))
	if err != nil {
		return nil, err
	}
	if branch.SpecDigest != expectedSpecDigest {
		return nil, fmt.Errorf("SPEC_DRIFT: branch is bound to %s, runtime loaded %s", branch.SpecDigest, expectedSpecDigest)
	}
	working, err := branch.State.Clone()
	if err != nil {
		return nil, err
	}
	callIndex := branch.CallCount + 1
	fired, err := takeFault(ctx, tx, branchID, toolName, FaultPhaseBeforeValidation)
	var outcome CallOutcome
	if err != nil {
		return nil, err
	}
	if fired != nil {
		outcome = faultOutcome(fired, false)
	} else {
		outcome, err = apply(working, branch.Clock, callIndex)
		if err != nil {
			return nil, err
		}
		if outcome.ErrorClass == "" {
			fired, err = takeFault(ctx, tx, branchID, toolName, FaultPhaseAfterCommitBeforeResponse)
			if err != nil {
				return nil, err
			}
			if fired != nil {
				outcome = faultOutcome(fired, true)
			}
		}
	}
	if !outcome.CommitState {
		working = branch.State
	}
	if err := working.ValidateBudget(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrResourceLimit, err)
	}

	stateJSON, err := canonical.JSON(working)
	if err != nil {
		return nil, err
	}
	afterDigest, err := canonical.Digest(working)
	if err != nil {
		return nil, err
	}
	resultJSON, err := canonical.JSON(outcome.Result)
	if err != nil {
		return nil, fmt.Errorf("canonicalize tool result: %w", err)
	}
	inputJSON, err := canonical.JSON(input)
	if err != nil {
		return nil, fmt.Errorf("canonicalize tool input: %w", err)
	}
	if len(inputJSON)+len(resultJSON) > limits.MaxAuditEventBytes {
		return nil, fmt.Errorf("%w: audit payload bytes %d exceed limit %d", ErrResourceLimit, len(inputJSON)+len(resultJSON), limits.MaxAuditEventBytes)
	}

	update, err := tx.ExecContext(ctx, `
UPDATE branches
SET state_json = ?, state_digest = ?, call_count = ?, head_version = head_version + 1
WHERE id = ? AND head_version = ?`, stateJSON, afterDigest, callIndex, branchID, branch.HeadVersion)
	if err != nil {
		return nil, fmt.Errorf("commit branch head: %w", err)
	}
	updated, err := update.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("read branch head update count: %w", err)
	}
	if updated != 1 {
		return nil, fmt.Errorf("%w: branch %s expected head %d", ErrBranchConflict, branchID, branch.HeadVersion)
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO audit(branch_id, call_index, tool_name, input_json, result_json, error_class, before_digest, after_digest, created_at)
VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)`, branchID, callIndex, toolName, inputJSON, resultJSON, outcome.ErrorClass, branch.StateDigest, afterDigest, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		return nil, fmt.Errorf("append audit record: %w", err)
	}
	if fired != nil {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO fault_events(branch_id, fault_id, call_index, phase, error_class, before_digest, after_digest, created_at)
VALUES(?, ?, ?, ?, ?, ?, ?, ?)`, branchID, fired.ID, callIndex, fired.Phase, fired.ErrorClass, branch.StateDigest, afterDigest, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
			return nil, fmt.Errorf("append fault event: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transition: %w", err)
	}
	result := &ApplyResult{Result: outcome.Result, ErrorClass: outcome.ErrorClass, BeforeDigest: branch.StateDigest, AfterDigest: afterDigest, CallIndex: callIndex}
	if fired != nil {
		result.FaultID = fired.ID
		result.FaultPhase = fired.Phase
	}
	return result, nil
}

func (s *Store) CreateSnapshot(ctx context.Context, name, branchID string) (*Snapshot, error) {
	if err := validateID("snapshot name", name); err != nil {
		return nil, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin snapshot transaction: %w", err)
	}
	defer tx.Rollback()
	var snapshotCount int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM snapshots`).Scan(&snapshotCount); err != nil {
		return nil, fmt.Errorf("count snapshots: %w", err)
	}
	if snapshotCount >= limits.MaxSnapshots {
		return nil, fmt.Errorf("%w: snapshot limit is %d", ErrResourceLimit, limits.MaxSnapshots)
	}
	branch, err := scanBranch(branchID, tx.QueryRowContext(ctx, `SELECT spec_digest, state_json, state_digest, clock, call_count, head_version FROM branches WHERE id = ?`, branchID))
	if err != nil {
		return nil, err
	}
	id, err := canonical.Digest(map[string]any{
		"name":                 name,
		"specDigest":           branch.SpecDigest,
		"stateDigest":          branch.StateDigest,
		"clock":                branch.Clock.UTC().Format(time.RFC3339Nano),
		"sourceHeadVersion":    branch.HeadVersion,
		"storageSchemaVersion": schemaVersion,
	})
	if err != nil {
		return nil, err
	}
	stateJSON, err := canonical.JSON(branch.State)
	if err != nil {
		return nil, err
	}
	_, err = tx.ExecContext(ctx, `
INSERT INTO snapshots(id, name, spec_digest, state_json, state_digest, clock, source_head_version, storage_schema_version, created_at)
VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)`, id, name, branch.SpecDigest, stateJSON, branch.StateDigest, branch.Clock.UTC().Format(time.RFC3339Nano), branch.HeadVersion, schemaVersion, time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return nil, fmt.Errorf("create snapshot: %w", err)
	}
	if err := appendControlAudit(ctx, tx, "snapshot.create", branchID, name, branch.StateDigest, branch.StateDigest); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit snapshot transaction: %w", err)
	}
	return &Snapshot{ID: id, Name: name, SpecDigest: branch.SpecDigest, State: branch.State, StateDigest: branch.StateDigest, Clock: branch.Clock, SourceHeadVersion: branch.HeadVersion, StorageSchemaVersion: schemaVersion}, nil
}

func (s *Store) snapshotByName(ctx context.Context, name string) (*Snapshot, error) {
	return scanSnapshot(name, s.db.QueryRowContext(ctx, `SELECT id, spec_digest, state_json, state_digest, clock, source_head_version, storage_schema_version FROM snapshots WHERE name = ?`, name))
}

func scanSnapshot(name string, row rowScanner) (*Snapshot, error) {
	var result Snapshot
	var stateJSON []byte
	var clockRaw string
	err := row.Scan(&result.ID, &result.SpecDigest, &stateJSON, &result.StateDigest, &clockRaw, &result.SourceHeadVersion, &result.StorageSchemaVersion)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSnapshotNotFound
		}
		return nil, fmt.Errorf("read snapshot: %w", err)
	}
	result.Name = name
	result.Clock, err = time.Parse(time.RFC3339Nano, clockRaw)
	if err != nil {
		return nil, fmt.Errorf("parse snapshot clock: %w", err)
	}
	var state world.State
	if err := json.Unmarshal(stateJSON, &state); err != nil {
		return nil, fmt.Errorf("decode snapshot state: %w", err)
	}
	state.Normalize()
	result.State = &state
	return &result, nil
}

func (s *Store) Fork(ctx context.Context, snapshotName, branchID string) error {
	if err := validateID("branch ID", branchID); err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin fork transaction: %w", err)
	}
	defer tx.Rollback()
	var branchCount int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM branches`).Scan(&branchCount); err != nil {
		return fmt.Errorf("count branches: %w", err)
	}
	if branchCount >= limits.MaxForks {
		return fmt.Errorf("%w: branch limit is %d", ErrResourceLimit, limits.MaxForks)
	}
	snapshot, err := scanSnapshot(snapshotName, tx.QueryRowContext(ctx, `SELECT id, spec_digest, state_json, state_digest, clock, source_head_version, storage_schema_version FROM snapshots WHERE name = ?`, snapshotName))
	if err != nil {
		return err
	}
	stateJSON, err := canonical.JSON(snapshot.State)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
INSERT INTO branches(id, spec_digest, state_json, state_digest, clock, call_count, head_version)
VALUES(?, ?, ?, ?, ?, 0, 0)`, branchID, snapshot.SpecDigest, stateJSON, snapshot.StateDigest, snapshot.Clock.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("fork snapshot: %w", err)
	}
	if err := appendControlAudit(ctx, tx, "branch.fork", branchID, snapshotName, "", snapshot.StateDigest); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit fork transaction: %w", err)
	}
	return nil
}

func (s *Store) Reset(ctx context.Context, snapshotName, branchID string) error {
	if err := validateID("branch ID", branchID); err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin reset transaction: %w", err)
	}
	defer tx.Rollback()
	snapshot, err := scanSnapshot(snapshotName, tx.QueryRowContext(ctx, `SELECT id, spec_digest, state_json, state_digest, clock, source_head_version, storage_schema_version FROM snapshots WHERE name = ?`, snapshotName))
	if err != nil {
		return err
	}
	branch, err := scanBranch(branchID, tx.QueryRowContext(ctx, `SELECT spec_digest, state_json, state_digest, clock, call_count, head_version FROM branches WHERE id = ?`, branchID))
	if err != nil {
		return err
	}
	stateJSON, err := canonical.JSON(snapshot.State)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `
UPDATE branches
SET spec_digest = ?, state_json = ?, state_digest = ?, clock = ?, call_count = 0, head_version = head_version + 1
WHERE id = ? AND head_version = ?`, snapshot.SpecDigest, stateJSON, snapshot.StateDigest, snapshot.Clock.UTC().Format(time.RFC3339Nano), branchID, branch.HeadVersion)
	if err != nil {
		return fmt.Errorf("reset branch: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("%w: branch %s expected head %d", ErrBranchConflict, branchID, branch.HeadVersion)
	}
	if err := appendControlAudit(ctx, tx, "branch.reset", branchID, snapshotName, branch.StateDigest, snapshot.StateDigest); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit reset transaction: %w", err)
	}
	return nil
}

// AdvanceClock moves a branch's virtual clock forward and records the
// privileged mutation. Host wall time is never used as the new world time.
// expectedHeadVersion, when non-nil, provides an optimistic concurrency check
// for a control-plane caller that read the branch first.
func (s *Store) AdvanceClock(ctx context.Context, branchID string, target time.Time, expectedHeadVersion *int64) error {
	target = target.UTC()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin clock transaction: %w", err)
	}
	defer tx.Rollback()
	branch, err := scanBranch(branchID, tx.QueryRowContext(ctx, `SELECT spec_digest, state_json, state_digest, clock, call_count, head_version FROM branches WHERE id = ?`, branchID))
	if err != nil {
		return err
	}
	if expectedHeadVersion != nil && *expectedHeadVersion != branch.HeadVersion {
		return fmt.Errorf("%w: branch %s expected head %d, current %d", ErrBranchConflict, branchID, *expectedHeadVersion, branch.HeadVersion)
	}
	if !target.After(branch.Clock) {
		return fmt.Errorf("%w: current=%s target=%s", ErrClockRegression, branch.Clock.UTC().Format(time.RFC3339Nano), target.Format(time.RFC3339Nano))
	}
	if target.Sub(branch.Clock) > MaxClockAdvance {
		return fmt.Errorf("%w: maximum advance is %s", ErrClockLimit, MaxClockAdvance)
	}
	result, err := tx.ExecContext(ctx, `
UPDATE branches SET clock = ?, head_version = head_version + 1
WHERE id = ? AND head_version = ?`, target.Format(time.RFC3339Nano), branchID, branch.HeadVersion)
	if err != nil {
		return fmt.Errorf("advance virtual clock: %w", err)
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read clock update count: %w", err)
	}
	if updated != 1 {
		return fmt.Errorf("%w: branch %s expected head %d", ErrBranchConflict, branchID, branch.HeadVersion)
	}
	if err := appendControlAudit(ctx, tx, "clock.advance", branchID, "", branch.StateDigest, branch.StateDigest); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit clock transaction: %w", err)
	}
	return nil
}

type sqlExecer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

func appendControlAudit(ctx context.Context, execer sqlExecer, operation, branchID, snapshotName, beforeDigest, afterDigest string) error {
	_, err := execer.ExecContext(ctx, `
INSERT INTO control_audit(operation, branch_id, snapshot_name, before_digest, after_digest, created_at)
VALUES(?, ?, ?, ?, ?, ?)`, operation, branchID, snapshotName, beforeDigest, afterDigest, time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("append control audit: %w", err)
	}
	return nil
}

func (s *Store) ControlAudit(ctx context.Context) ([]ControlAuditEntry, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, operation, branch_id, snapshot_name, before_digest, after_digest, created_at
FROM control_audit ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("query control audit: %w", err)
	}
	defer rows.Close()
	var entries []ControlAuditEntry
	for rows.Next() {
		var entry ControlAuditEntry
		if err := rows.Scan(&entry.ID, &entry.Operation, &entry.BranchID, &entry.SnapshotName, &entry.BeforeDigest, &entry.AfterDigest, &entry.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan control audit: %w", err)
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate control audit: %w", err)
	}
	return entries, nil
}
