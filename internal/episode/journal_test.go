package episode

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/augety121/mcp-state-twin/internal/bundle"
	_ "modernc.org/sqlite"
)

func TestJournalPersistsAndIdempotentlyReusesTerminalEvidence(t *testing.T) {
	ctx := context.Background()
	artifact := buildIssueTrackerArtifact(t)
	path := filepath.Join(t.TempDir(), "episodes.db")
	journal, err := OpenJournal(path)
	if err != nil {
		t.Fatal(err)
	}
	first, reused, err := journal.RunPersistent(ctx, artifact, "durable-001", "", "0.2.0-dev", "revision-1")
	if err != nil {
		t.Fatal(err)
	}
	if reused || first.EvidenceDigest == "" {
		t.Fatalf("unexpected first result: reused=%v evidence=%+v", reused, first)
	}
	record, err := journal.Get(ctx, "durable-001")
	if err != nil {
		t.Fatal(err)
	}
	if record.Status != StatusSucceeded || record.Incomplete || record.Sequence != 5 || len(record.Events) != 6 {
		t.Fatalf("unexpected durable record: %+v", record)
	}
	if record.EvidenceDigest != first.EvidenceDigest || record.Evidence == nil {
		t.Fatalf("terminal evidence was not persisted: %+v", record)
	}
	terminalUpdatedAt := record.UpdatedAt

	second, reused, err := journal.RunPersistent(ctx, artifact, "durable-001", "", "0.2.0-dev", "revision-1")
	if err != nil {
		t.Fatal(err)
	}
	if !reused || second.EvidenceDigest != first.EvidenceDigest {
		t.Fatalf("idempotent replay changed evidence: reused=%v first=%s second=%s", reused, first.EvidenceDigest, second.EvidenceDigest)
	}
	replayedRecord, err := journal.Get(ctx, "durable-001")
	if err != nil {
		t.Fatal(err)
	}
	if replayedRecord.UpdatedAt != terminalUpdatedAt || replayedRecord.Sequence != 5 || len(replayedRecord.Events) != 6 {
		t.Fatalf("idempotent replay mutated the Journal: before=%s after=%+v", terminalUpdatedAt, replayedRecord)
	}
	if err := journal.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := OpenJournal(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	record, err = reopened.Get(ctx, "durable-001")
	if err != nil || record.EvidenceDigest != first.EvidenceDigest || record.Status != StatusSucceeded {
		t.Fatalf("reopened record = %+v, err=%v", record, err)
	}
}

func TestJournalRejectsIdentityConflictAndDoesNotRetryIncompleteEpisode(t *testing.T) {
	ctx := context.Background()
	artifact := buildIssueTrackerArtifact(t)
	journal, err := OpenJournal(filepath.Join(t.TempDir(), "episodes.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer journal.Close()
	request, err := resolveRequest(artifact, "incomplete-001", "", "0.2.0-dev", "revision-1")
	if err != nil {
		t.Fatal(err)
	}
	if _, created, err := journal.createOrLoad(ctx, request); err != nil || !created {
		t.Fatalf("reserve Episode: created=%v err=%v", created, err)
	}
	if err := journal.appendTransition(ctx, request.EpisodeID, request, 0, Event{Sequence: 1, From: StatusCreated, To: StatusProvisioning}); err != nil {
		t.Fatal(err)
	}
	if err := journal.appendTransition(ctx, request.EpisodeID, request, 1, Event{Sequence: 2, From: StatusProvisioning, To: StatusReady}); err != nil {
		t.Fatal(err)
	}
	if _, reused, err := journal.RunPersistent(ctx, artifact, request.EpisodeID, "", request.RuntimeVersion, request.RuntimeRevision); !reused || !errors.Is(err, ErrEpisodeIncomplete) {
		t.Fatalf("expected incomplete refusal, reused=%v err=%v", reused, err)
	}
	record, err := journal.Get(ctx, request.EpisodeID)
	if err != nil {
		t.Fatal(err)
	}
	if !record.Incomplete || record.Status != StatusReady || record.Sequence != 2 || record.Evidence != nil {
		t.Fatalf("incomplete record was changed: %+v", record)
	}
	if _, _, err := journal.RunPersistent(ctx, artifact, request.EpisodeID, "", request.RuntimeVersion, "different-revision"); !errors.Is(err, ErrEpisodeConflict) {
		t.Fatalf("expected immutable request conflict, got %v", err)
	}
}

func TestJournalPersistsRuntimeErrorWithoutSensitiveMessage(t *testing.T) {
	ctx := context.Background()
	artifact := buildIssueTrackerArtifact(t)
	artifact.Files[artifact.Manifest.Spec] = []byte("apiVersion: invalid\nsecret: sk-aaaaaaaaaaaaaaaaaaaa\n")
	journal, err := OpenJournal(filepath.Join(t.TempDir(), "episodes.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer journal.Close()
	if _, _, err := journal.RunPersistent(ctx, artifact, "runtime-error-001", "", "0.2.0-dev", "revision-1"); err == nil {
		t.Fatal("tampered in-memory artifact unexpectedly ran")
	}
	record, err := journal.Get(ctx, "runtime-error-001")
	if err != nil {
		t.Fatal(err)
	}
	if record.Status != StatusRuntimeError || record.ErrorClass != "RUNTIME_ERROR" || record.Evidence != nil {
		t.Fatalf("unexpected runtime-error record: %+v", record)
	}
	encoded := record.ErrorClass + record.Outcome
	if strings.Contains(encoded, "sk-") || strings.Contains(encoded, "secret") {
		t.Fatalf("journal persisted raw failure details: %q", encoded)
	}
}

func TestJournalDetectsEvidenceTampering(t *testing.T) {
	ctx := context.Background()
	artifact := buildIssueTrackerArtifact(t)
	journal, err := OpenJournal(filepath.Join(t.TempDir(), "episodes.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer journal.Close()
	if _, _, err := journal.RunPersistent(ctx, artifact, "tamper-001", "", "0.2.0-dev", "revision-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := journal.db.Exec(`UPDATE episodes SET evidence_json = '{"apiVersion":"tampered"}' WHERE id = 'tamper-001'`); err != nil {
		t.Fatal(err)
	}
	if _, err := journal.Get(ctx, "tamper-001"); err == nil || !strings.Contains(err.Error(), "identity mismatch") {
		t.Fatalf("expected evidence tamper refusal, got %v", err)
	}
}

func TestJournalDetectsRequestAndLifecycleTampering(t *testing.T) {
	ctx := context.Background()
	artifact := buildIssueTrackerArtifact(t)
	journal, err := OpenJournal(filepath.Join(t.TempDir(), "episodes.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer journal.Close()
	if _, _, err := journal.RunPersistent(ctx, artifact, "chain-001", "", "0.2.0-dev", "revision-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := journal.db.Exec(`UPDATE episodes SET runtime_revision = 'changed' WHERE id = 'chain-001'`); err != nil {
		t.Fatal(err)
	}
	if _, err := journal.Get(ctx, "chain-001"); err == nil || !strings.Contains(err.Error(), "request digest mismatch") {
		t.Fatalf("expected request tamper refusal, got %v", err)
	}
	if _, err := journal.db.Exec(`UPDATE episodes SET runtime_revision = 'revision-1' WHERE id = 'chain-001'`); err != nil {
		t.Fatal(err)
	}
	if _, err := journal.db.Exec(`UPDATE episode_events SET from_status = 'READY' WHERE episode_id = 'chain-001' AND sequence = 1`); err != nil {
		t.Fatal(err)
	}
	if _, err := journal.Get(ctx, "chain-001"); err == nil || !strings.Contains(err.Error(), "transition chain") {
		t.Fatalf("expected lifecycle tamper refusal, got %v", err)
	}
}

func TestJournalCrossValidatesEvidenceAgainstRecord(t *testing.T) {
	ctx := context.Background()
	artifact := buildIssueTrackerArtifact(t)
	journal, err := OpenJournal(filepath.Join(t.TempDir(), "episodes.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer journal.Close()
	if _, _, err := journal.RunPersistent(ctx, artifact, "cross-bind-001", "", "0.2.0-dev", "revision-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := journal.db.Exec(`UPDATE episodes SET outcome = 'assertion_failed' WHERE id = 'cross-bind-001'`); err != nil {
		t.Fatal(err)
	}
	if _, err := journal.Get(ctx, "cross-bind-001"); err == nil || !strings.Contains(err.Error(), "does not match Journal identity") {
		t.Fatalf("expected evidence/record cross-binding refusal, got %v", err)
	}
}

func TestJournalRejectsInconsistentOperationalMetadata(t *testing.T) {
	ctx := context.Background()
	artifact := buildIssueTrackerArtifact(t)
	journal, err := OpenJournal(filepath.Join(t.TempDir(), "episodes.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer journal.Close()
	if _, _, err := journal.RunPersistent(ctx, artifact, "metadata-001", "", "0.2.0-dev", "revision-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := journal.db.Exec(`UPDATE episodes SET updated_at = '2000-01-01T00:00:00Z' WHERE id = 'metadata-001'`); err != nil {
		t.Fatal(err)
	}
	if _, err := journal.Get(ctx, "metadata-001"); err == nil || !strings.Contains(err.Error(), "timestamps are inconsistent") {
		t.Fatalf("expected timestamp-order refusal, got %v", err)
	}
}

func TestJournalRefusesForeignAndFutureSQLite(t *testing.T) {
	root := t.TempDir()
	foreignPath := filepath.Join(root, "foreign.db")
	db, err := sql.Open("sqlite", foreignPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE foreign_data(value TEXT)`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenJournal(foreignPath); err == nil || !strings.Contains(err.Error(), "unidentified non-empty") {
		t.Fatalf("expected foreign database refusal, got %v", err)
	}
	db, err = sql.Open("sqlite", foreignPath)
	if err != nil {
		t.Fatal(err)
	}
	var applicationID, version int
	var journalMode string
	if err := db.QueryRow(`PRAGMA application_id`).Scan(&applicationID); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`PRAGMA journal_mode`).Scan(&journalMode); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if applicationID != 0 || version != 0 || strings.EqualFold(journalMode, "wal") {
		t.Fatalf("foreign database was mutated: applicationID=%d version=%d journalMode=%s", applicationID, version, journalMode)
	}

	futurePath := filepath.Join(root, "future.db")
	journal, err := OpenJournal(futurePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := journal.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = sql.Open("sqlite", futurePath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`PRAGMA user_version = 3`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenJournal(futurePath); err == nil || !strings.Contains(err.Error(), "newer than supported") {
		t.Fatalf("expected future journal refusal, got %v", err)
	}
}

func buildIssueTrackerArtifact(t *testing.T) *bundle.Artifact {
	t.Helper()
	manifestPath := filepath.Join("..", "..", "examples", "issue-tracker", "bundle.yaml")
	bundlePath := filepath.Join(t.TempDir(), "issue-tracker.stb")
	if _, err := bundle.Build(manifestPath, bundlePath); err != nil {
		t.Fatal(err)
	}
	artifact, err := bundle.Open(bundlePath)
	if err != nil {
		t.Fatal(err)
	}
	return artifact
}
