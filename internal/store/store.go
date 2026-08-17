package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/augety121/mcp-state-twin/internal/canonical"
	"github.com/augety121/mcp-state-twin/internal/world"
	_ "modernc.org/sqlite"
)

var (
	ErrBranchNotFound   = errors.New("branch not found")
	ErrSnapshotNotFound = errors.New("snapshot not found")
)

var idPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

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
}

type Snapshot struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	SpecDigest  string       `json:"specDigest"`
	State       *world.State `json:"state"`
	StateDigest string       `json:"stateDigest"`
	Clock       time.Time    `json:"clock"`
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
	const schema = `
CREATE TABLE IF NOT EXISTS branches (
  id TEXT PRIMARY KEY,
  spec_digest TEXT NOT NULL,
  state_json BLOB NOT NULL,
  state_digest TEXT NOT NULL,
  clock TEXT NOT NULL,
  call_count INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS snapshots (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL UNIQUE,
  spec_digest TEXT NOT NULL,
  state_json BLOB NOT NULL,
  state_digest TEXT NOT NULL,
  clock TEXT NOT NULL,
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
);`
	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("migrate SQLite schema: %w", err)
	}
	return nil
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
	state.Normalize()
	stateJSON, err := canonical.JSON(state)
	if err != nil {
		return err
	}
	stateDigest, err := canonical.Digest(state)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
INSERT INTO branches(id, spec_digest, state_json, state_digest, clock, call_count)
VALUES(?, ?, ?, ?, ?, 0)
ON CONFLICT(id) DO NOTHING`, id, specDigest, stateJSON, stateDigest, clock.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("initialize branch: %w", err)
	}
	return nil
}

func (s *Store) Branch(ctx context.Context, id string) (*Branch, error) {
	row := s.db.QueryRowContext(ctx, `SELECT spec_digest, state_json, state_digest, clock, call_count FROM branches WHERE id = ?`, id)
	return scanBranch(id, row)
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanBranch(id string, row rowScanner) (*Branch, error) {
	var specDigest, stateDigest, clockRaw string
	var stateJSON []byte
	var callCount int64
	if err := row.Scan(&specDigest, &stateJSON, &stateDigest, &clockRaw, &callCount); err != nil {
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
	return &Branch{ID: id, SpecDigest: specDigest, State: &state, StateDigest: stateDigest, Clock: clock, CallCount: callCount}, nil
}

func (s *Store) ApplyCall(
	ctx context.Context,
	branchID, expectedSpecDigest, toolName string,
	input any,
	apply func(state *world.State, clock time.Time, callIndex int64) (CallOutcome, error),
) (*ApplyResult, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transition: %w", err)
	}
	defer tx.Rollback()

	branch, err := scanBranch(branchID, tx.QueryRowContext(ctx, `SELECT spec_digest, state_json, state_digest, clock, call_count FROM branches WHERE id = ?`, branchID))
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
	outcome, err := apply(working, branch.Clock, callIndex)
	if err != nil {
		return nil, err
	}
	if !outcome.CommitState {
		working = branch.State
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

	if _, err := tx.ExecContext(ctx, `UPDATE branches SET state_json = ?, state_digest = ?, call_count = ? WHERE id = ?`, stateJSON, afterDigest, callIndex, branchID); err != nil {
		return nil, fmt.Errorf("commit branch head: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO audit(branch_id, call_index, tool_name, input_json, result_json, error_class, before_digest, after_digest, created_at)
VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)`, branchID, callIndex, toolName, inputJSON, resultJSON, outcome.ErrorClass, branch.StateDigest, afterDigest, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		return nil, fmt.Errorf("append audit record: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transition: %w", err)
	}
	return &ApplyResult{Result: outcome.Result, ErrorClass: outcome.ErrorClass, BeforeDigest: branch.StateDigest, AfterDigest: afterDigest, CallIndex: callIndex}, nil
}

func (s *Store) CreateSnapshot(ctx context.Context, name, branchID string) (*Snapshot, error) {
	if err := validateID("snapshot name", name); err != nil {
		return nil, err
	}
	branch, err := s.Branch(ctx, branchID)
	if err != nil {
		return nil, err
	}
	id, err := canonical.Digest(map[string]any{
		"name":        name,
		"specDigest":  branch.SpecDigest,
		"stateDigest": branch.StateDigest,
		"clock":       branch.Clock.UTC().Format(time.RFC3339Nano),
	})
	if err != nil {
		return nil, err
	}
	stateJSON, err := canonical.JSON(branch.State)
	if err != nil {
		return nil, err
	}
	_, err = s.db.ExecContext(ctx, `
INSERT INTO snapshots(id, name, spec_digest, state_json, state_digest, clock, created_at)
VALUES(?, ?, ?, ?, ?, ?, ?)`, id, name, branch.SpecDigest, stateJSON, branch.StateDigest, branch.Clock.UTC().Format(time.RFC3339Nano), time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return nil, fmt.Errorf("create snapshot: %w", err)
	}
	return &Snapshot{ID: id, Name: name, SpecDigest: branch.SpecDigest, State: branch.State, StateDigest: branch.StateDigest, Clock: branch.Clock}, nil
}

func (s *Store) snapshotByName(ctx context.Context, name string) (*Snapshot, error) {
	var result Snapshot
	var stateJSON []byte
	var clockRaw string
	err := s.db.QueryRowContext(ctx, `SELECT id, spec_digest, state_json, state_digest, clock FROM snapshots WHERE name = ?`, name).Scan(&result.ID, &result.SpecDigest, &stateJSON, &result.StateDigest, &clockRaw)
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
	snapshot, err := s.snapshotByName(ctx, snapshotName)
	if err != nil {
		return err
	}
	stateJSON, err := canonical.JSON(snapshot.State)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
INSERT INTO branches(id, spec_digest, state_json, state_digest, clock, call_count)
VALUES(?, ?, ?, ?, ?, 0)`, branchID, snapshot.SpecDigest, stateJSON, snapshot.StateDigest, snapshot.Clock.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("fork snapshot: %w", err)
	}
	return nil
}

func (s *Store) Reset(ctx context.Context, snapshotName, branchID string) error {
	snapshot, err := s.snapshotByName(ctx, snapshotName)
	if err != nil {
		return err
	}
	stateJSON, err := canonical.JSON(snapshot.State)
	if err != nil {
		return err
	}
	result, err := s.db.ExecContext(ctx, `
UPDATE branches SET spec_digest = ?, state_json = ?, state_digest = ?, clock = ?, call_count = 0 WHERE id = ?`, snapshot.SpecDigest, stateJSON, snapshot.StateDigest, snapshot.Clock.UTC().Format(time.RFC3339Nano), branchID)
	if err != nil {
		return fmt.Errorf("reset branch: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return ErrBranchNotFound
	}
	return nil
}
