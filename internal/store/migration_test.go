package store

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

const migrationCrashExitCode = 86

func TestTaggedAlphaDatabaseRemainsReadable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tagged-alpha.db")
	installSQLFixture(t, path, "v0.1.0-alpha.1-schema-v4.sql")

	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	assertStorageVersion(t, s, schemaVersion)
	branch, err := s.Branch(context.Background(), "main")
	if err != nil {
		t.Fatal(err)
	}
	if branch.SpecDigest != "sha256:tagged-spec" || branch.CallCount != 1 || branch.HeadVersion != 2 {
		t.Fatalf("tagged branch changed during reopen: %#v", branch)
	}
	snapshot, err := s.snapshotByName(context.Background(), "tagged-base")
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.ID != "tagged-snapshot-id" || snapshot.SourceHeadVersion != 2 || snapshot.StorageSchemaVersion != 4 {
		t.Fatalf("tagged snapshot changed during reopen: %#v", snapshot)
	}
	assertRowCount(t, s, "audit", 1)
	assertRowCount(t, s, "control_audit", 1)
	plans, err := s.FaultPlans(context.Background(), "main")
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) != 1 || plans[0].ID != "tagged-rate-limit" || plans[0].RemainingCount != 1 {
		t.Fatalf("tagged fault plans changed during reopen: %#v", plans)
	}
}

func TestSchemaV3FixtureMigratesWithoutDataLoss(t *testing.T) {
	path := filepath.Join(t.TempDir(), "schema-v3.db")
	installSQLFixture(t, path, "schema-v3-pre-fault.sql")

	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	assertMigratedV3State(t, s)
	if _, err := s.InstallFault(context.Background(), FaultPlan{
		ID: "post-v3-migration", BranchID: "main", ToolName: "get_item",
		Phase: FaultPhaseBeforeValidation, ErrorClass: "RATE_LIMITED",
		Message: "migration proof", RemainingCount: 1,
	}, int64Pointer(5)); err != nil {
		t.Fatalf("schema-v4 fault tables unavailable after v3 migration: %v", err)
	}
}

func TestInterruptedMigrationRecoversAfterProcessExit(t *testing.T) {
	if os.Getenv("STATETWIN_TEST_MIGRATION_CRASH") == "1" {
		path := os.Getenv("STATETWIN_TEST_MIGRATION_PATH")
		wantStage := migrationStage(os.Getenv("STATETWIN_TEST_MIGRATION_STAGE"))
		_, err := openWithMigrationHook(path, func(gotStage migrationStage) error {
			if gotStage == wantStage {
				os.Exit(migrationCrashExitCode)
			}
			return nil
		})
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(87)
		}
		os.Exit(88)
	}

	for _, stage := range []migrationStage{migrationStageSchemaApplied, migrationStageMetadataSet} {
		t.Run(string(stage), func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "interrupted.db")
			installSQLFixture(t, path, "schema-v3-pre-fault.sql")

			command := exec.Command(os.Args[0], "-test.run=^TestInterruptedMigrationRecoversAfterProcessExit$")
			command.Env = append(os.Environ(),
				"STATETWIN_TEST_MIGRATION_CRASH=1",
				"STATETWIN_TEST_MIGRATION_PATH="+path,
				"STATETWIN_TEST_MIGRATION_STAGE="+string(stage),
			)
			err := command.Run()
			exitError, ok := err.(*exec.ExitError)
			if !ok || exitError.ExitCode() != migrationCrashExitCode {
				t.Fatalf("migration subprocess error = %v, want exit code %d", err, migrationCrashExitCode)
			}

			s, err := Open(path)
			if err != nil {
				t.Fatalf("reopen after migration interruption at %s: %v", stage, err)
			}
			defer s.Close()
			assertMigratedV3State(t, s)
			var integrity string
			if err := s.db.QueryRow(`PRAGMA integrity_check`).Scan(&integrity); err != nil {
				t.Fatal(err)
			}
			if integrity != "ok" {
				t.Fatalf("integrity_check after migration interruption = %q", integrity)
			}
		})
	}
}

func installSQLFixture(t *testing.T, path, name string) {
	t.Helper()
	script, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(string(script)); err != nil {
		_ = db.Close()
		t.Fatalf("install fixture %s: %v", name, err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
}

func assertMigratedV3State(t *testing.T, s *Store) {
	t.Helper()
	assertStorageVersion(t, s, schemaVersion)
	branch, err := s.Branch(context.Background(), "main")
	if err != nil {
		t.Fatal(err)
	}
	if branch.SpecDigest != "sha256:v3-spec" || branch.CallCount != 3 || branch.HeadVersion != 5 {
		t.Fatalf("v3 branch changed during migration: %#v", branch)
	}
	snapshot, err := s.snapshotByName(context.Background(), "v3-base")
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.ID != "v3-snapshot-id" || snapshot.SourceHeadVersion != 5 || snapshot.StorageSchemaVersion != 3 {
		t.Fatalf("v3 snapshot changed during migration: %#v", snapshot)
	}
	assertRowCount(t, s, "audit", 1)
	assertRowCount(t, s, "control_audit", 1)
	assertRowCount(t, s, "fault_plans", 0)
	assertRowCount(t, s, "fault_events", 0)
}

func assertStorageVersion(t *testing.T, s *Store, want int) {
	t.Helper()
	var got int
	if err := s.db.QueryRow(`PRAGMA user_version`).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("storage schema version = %d, want %d", got, want)
	}
}

func assertRowCount(t *testing.T, s *Store, table string, want int) {
	t.Helper()
	var got int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("%s row count = %d, want %d", table, got, want)
	}
}
