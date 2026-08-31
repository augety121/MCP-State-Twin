package episode

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/augety121/mcp-state-twin/internal/bundle"
)

func TestRemoteHermeticLeaseRecoveryFencesStaleWorkerAndCompletesOnce(t *testing.T) {
	ctx := context.Background()
	journal, err := OpenJournal(filepath.Join(t.TempDir(), "episodes.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer journal.Close()
	data := buildIssueTrackerBundleBytes(t)
	if _, created, err := journal.Submit(ctx, data, "remote-001", "", "0.2.0-dev", "revision-1", EffectHermetic, 3); err != nil || !created {
		t.Fatalf("submit: created=%v err=%v", created, err)
	}
	first, err := journal.Claim(ctx, "worker-a", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := journal.db.Exec(`UPDATE episode_attempts SET lease_until = '2000-01-01T00:00:00Z' WHERE attempt_id = ?`, first.AttemptID); err != nil {
		t.Fatal(err)
	}
	if recovered, err := journal.RecoverExpired(ctx); err != nil || recovered != 1 {
		t.Fatalf("recover expired = %d, %v", recovered, err)
	}
	second, err := journal.Claim(ctx, "worker-b", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if second.AttemptNumber != 2 || second.FencingToken <= first.FencingToken {
		t.Fatalf("recovery did not advance attempt/fence: first=%+v second=%+v", first, second)
	}
	if _, err := journal.Heartbeat(ctx, first.EpisodeID, first.AttemptID, first.FencingToken, time.Minute); !errors.Is(err, ErrStaleAttempt) {
		t.Fatalf("stale heartbeat error = %v", err)
	}
	artifact, err := bundle.OpenBytes(second.Bundle)
	if err != nil {
		t.Fatal(err)
	}
	evidence, err := Run(ctx, artifact, second.EpisodeID, second.ScenarioPath, second.Runtime.Version, second.Runtime.Revision)
	if err != nil {
		t.Fatal(err)
	}
	accepted, reused, err := journal.CompleteClaim(ctx, second.EpisodeID, second.AttemptID, second.FencingToken, evidence)
	if err != nil || reused || accepted.EvidenceDigest != evidence.EvidenceDigest {
		t.Fatalf("complete: reused=%v accepted=%+v err=%v", reused, accepted, err)
	}
	again, reused, err := journal.CompleteClaim(ctx, second.EpisodeID, second.AttemptID, second.FencingToken, evidence)
	if err != nil || !reused || again.EvidenceDigest != evidence.EvidenceDigest {
		t.Fatalf("duplicate completion: reused=%v accepted=%+v err=%v", reused, again, err)
	}
	task, err := journal.GetTask(ctx, second.EpisodeID)
	if err != nil {
		t.Fatal(err)
	}
	if task.State != TaskCompleted || task.Episode.Status != StatusSucceeded || len(task.Attempts) != 2 || task.Attempts[0].State != AttemptExpired || task.Attempts[1].State != AttemptCompleted {
		t.Fatalf("unexpected completed task: %+v", task)
	}
}

func TestRemoteExternalExpiryFailsClosedAsCommitUnknown(t *testing.T) {
	ctx := context.Background()
	journal, err := OpenJournal(filepath.Join(t.TempDir(), "episodes.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer journal.Close()
	if _, _, err := journal.Submit(ctx, buildIssueTrackerBundleBytes(t), "external-001", "", "0.2.0-dev", "revision-1", EffectExternal, 3); err != nil {
		t.Fatal(err)
	}
	claim, err := journal.Claim(ctx, "provider-worker", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := journal.db.Exec(`UPDATE episode_attempts SET lease_until = '2000-01-01T00:00:00Z' WHERE attempt_id = ?`, claim.AttemptID); err != nil {
		t.Fatal(err)
	}
	if _, err := journal.RecoverExpired(ctx); err != nil {
		t.Fatal(err)
	}
	task, err := journal.GetTask(ctx, claim.EpisodeID)
	if err != nil {
		t.Fatal(err)
	}
	if task.State != TaskCommitUnknown || task.Attempts[0].CommitState != CommitUnknown || task.Attempts[0].State != AttemptCommitUnknown {
		t.Fatalf("external ambiguity was collapsed: %+v", task)
	}
	if _, err := journal.Claim(ctx, "provider-worker-2", time.Minute); !errors.Is(err, ErrNoClaimableEpisode) {
		t.Fatalf("ambiguous external attempt was retried: %v", err)
	}
}

func TestRemoteCancellationIsDurableAndCooperative(t *testing.T) {
	ctx := context.Background()
	journal, err := OpenJournal(filepath.Join(t.TempDir(), "episodes.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer journal.Close()
	data := buildIssueTrackerBundleBytes(t)
	if _, _, err := journal.Submit(ctx, data, "cancel-queued", "", "0.2.0-dev", "revision-1", EffectHermetic, 2); err != nil {
		t.Fatal(err)
	}
	queued, err := journal.CancelTask(ctx, "cancel-queued")
	if err != nil || queued.State != TaskCancelled || queued.Episode.Status != StatusCancelled {
		t.Fatalf("queued cancellation = %+v, %v", queued, err)
	}
	if _, _, err := journal.Submit(ctx, data, "cancel-leased", "", "0.2.0-dev", "revision-1", EffectHermetic, 2); err != nil {
		t.Fatal(err)
	}
	claim, err := journal.Claim(ctx, "worker-cancel", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	leasing, err := journal.CancelTask(ctx, claim.EpisodeID)
	if err != nil || leasing.State != TaskLeased || !leasing.CancelRequested {
		t.Fatalf("leased cancellation = %+v, %v", leasing, err)
	}
	heartbeat, err := journal.Heartbeat(ctx, claim.EpisodeID, claim.AttemptID, claim.FencingToken, time.Minute)
	if err != nil || !heartbeat.CancelRequested {
		t.Fatalf("heartbeat cancellation = %+v, %v", heartbeat, err)
	}
	artifact, err := bundle.OpenBytes(claim.Bundle)
	if err != nil {
		t.Fatal(err)
	}
	evidence, err := Run(ctx, artifact, claim.EpisodeID, claim.ScenarioPath, claim.Runtime.Version, claim.Runtime.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := journal.CompleteClaim(ctx, claim.EpisodeID, claim.AttemptID, claim.FencingToken, evidence); !errors.Is(err, ErrCancelRequested) {
		t.Fatalf("completion after cancellation error = %v", err)
	}
	cancelled, err := journal.FailClaim(ctx, claim.EpisodeID, claim.AttemptID, claim.FencingToken, CommitNoEffect, "CANCELLED")
	if err != nil || cancelled.State != TaskCancelled || cancelled.Episode.Status != StatusCancelled {
		t.Fatalf("cancel acknowledgement = %+v, %v", cancelled, err)
	}
}

func TestRemoteConcurrentClaimHasOneOwner(t *testing.T) {
	ctx := context.Background()
	journal, err := OpenJournal(filepath.Join(t.TempDir(), "episodes.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer journal.Close()
	if _, _, err := journal.Submit(ctx, buildIssueTrackerBundleBytes(t), "claim-race", "", "0.2.0-dev", "revision-1", EffectHermetic, 2); err != nil {
		t.Fatal(err)
	}
	type result struct {
		claim *Claim
		err   error
	}
	results := make(chan result, 2)
	var group sync.WaitGroup
	for _, worker := range []string{"worker-one", "worker-two"} {
		group.Add(1)
		go func(id string) {
			defer group.Done()
			claim, err := journal.Claim(ctx, id, time.Minute)
			results <- result{claim: claim, err: err}
		}(worker)
	}
	group.Wait()
	close(results)
	winners, empty := 0, 0
	for got := range results {
		if got.err == nil && got.claim != nil {
			winners++
		} else if errors.Is(got.err, ErrNoClaimableEpisode) {
			empty++
		} else {
			t.Fatalf("unexpected claim result: %+v", got)
		}
	}
	if winners != 1 || empty != 1 {
		t.Fatalf("claim winners=%d no-task=%d", winners, empty)
	}
}

func TestGetTaskRejectsTamperedAttemptAccounting(t *testing.T) {
	ctx := context.Background()
	journal, err := OpenJournal(filepath.Join(t.TempDir(), "episodes.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer journal.Close()
	if _, _, err := journal.Submit(ctx, buildIssueTrackerBundleBytes(t), "tampered-task", "", "0.2.0-dev", "revision-1", EffectHermetic, 2); err != nil {
		t.Fatal(err)
	}
	if _, err := journal.db.Exec(`UPDATE episode_tasks SET attempt_count = 1 WHERE episode_id = 'tampered-task'`); err != nil {
		t.Fatal(err)
	}
	if _, err := journal.GetTask(ctx, "tampered-task"); err == nil || !strings.Contains(err.Error(), "attempt accounting") {
		t.Fatalf("tampered task error = %v", err)
	}
}

func TestJournalSchemaV1MigratesToV2WithoutLosingEpisode(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "episodes.db")
	journal, err := OpenJournal(path)
	if err != nil {
		t.Fatal(err)
	}
	artifact := buildIssueTrackerArtifact(t)
	request, err := resolveRequest(artifact, "legacy-episode", "", "0.2.0-dev", "revision-1")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := journal.createOrLoad(ctx, request); err != nil {
		t.Fatal(err)
	}
	if err := journal.Close(); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DROP TABLE episode_attempts; DROP TABLE episode_tasks; PRAGMA user_version = 1`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	migrated, err := OpenJournal(path)
	if err != nil {
		t.Fatal(err)
	}
	defer migrated.Close()
	record, err := migrated.Get(ctx, request.EpisodeID)
	if err != nil || record.RequestDigest == "" || record.Status != StatusCreated {
		t.Fatalf("migrated Episode = %+v, %v", record, err)
	}
	var version int
	if err := migrated.db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil || version != 2 {
		t.Fatalf("schema version = %d, %v", version, err)
	}
}

func TestPublishedJournalV1FixtureMigratesWithoutDataLoss(t *testing.T) {
	path := filepath.Join(t.TempDir(), "published-v1.db")
	installJournalFixture(t, path)
	journal, err := OpenJournal(path)
	if err != nil {
		t.Fatal(err)
	}
	defer journal.Close()
	record, err := journal.Get(context.Background(), "published-v1")
	if err != nil {
		t.Fatal(err)
	}
	if record.RequestDigest != "sha256:163a58a7c15adc635d51bd3f63bb7038085f2ddb8893d6e7a8b6be6cb71cb927" || record.Status != StatusCreated || record.Sequence != 0 {
		t.Fatalf("published Journal record changed: %+v", record)
	}
	var version int
	if err := journal.db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil || version != journalSchemaVersion {
		t.Fatalf("Journal schema version = %d, %v", version, err)
	}
}

func TestInterruptedJournalV1MigrationRecoversAfterProcessExit(t *testing.T) {
	const crashCode = 86
	if os.Getenv("STATETWIN_TEST_JOURNAL_MIGRATION_CRASH") == "1" {
		path := os.Getenv("STATETWIN_TEST_JOURNAL_MIGRATION_PATH")
		want := journalMigrationStage(os.Getenv("STATETWIN_TEST_JOURNAL_MIGRATION_STAGE"))
		_, err := openJournalWithMigrationHook(path, func(got journalMigrationStage) error {
			if got == want {
				os.Exit(crashCode)
			}
			return nil
		})
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(87)
		}
		os.Exit(88)
	}
	for _, stage := range []journalMigrationStage{journalMigrationSchemaApplied, journalMigrationMetadataSet} {
		t.Run(string(stage), func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "interrupted.db")
			installJournalFixture(t, path)
			command := exec.Command(os.Args[0], "-test.run=^TestInterruptedJournalV1MigrationRecoversAfterProcessExit$")
			command.Env = append(os.Environ(),
				"STATETWIN_TEST_JOURNAL_MIGRATION_CRASH=1",
				"STATETWIN_TEST_JOURNAL_MIGRATION_PATH="+path,
				"STATETWIN_TEST_JOURNAL_MIGRATION_STAGE="+string(stage),
			)
			err := command.Run()
			exitError, ok := err.(*exec.ExitError)
			if !ok || exitError.ExitCode() != crashCode {
				t.Fatalf("migration subprocess error = %v", err)
			}
			journal, err := OpenJournal(path)
			if err != nil {
				t.Fatal(err)
			}
			defer journal.Close()
			if _, err := journal.Get(context.Background(), "published-v1"); err != nil {
				t.Fatal(err)
			}
			var integrity string
			if err := journal.db.QueryRow(`PRAGMA integrity_check`).Scan(&integrity); err != nil || integrity != "ok" {
				t.Fatalf("integrity_check = %q, %v", integrity, err)
			}
		})
	}
}

func installJournalFixture(t *testing.T, path string) {
	t.Helper()
	script, err := os.ReadFile(filepath.Join("testdata", "b8e836b-journal-v1.sql"))
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(string(script)); err != nil {
		_ = db.Close()
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
}

func buildIssueTrackerBundleBytes(t *testing.T) []byte {
	t.Helper()
	manifestPath := filepath.Join("..", "..", "examples", "issue-tracker", "bundle.yaml")
	path := filepath.Join(t.TempDir(), "issue-tracker.stb")
	if _, err := bundle.Build(manifestPath, path); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
