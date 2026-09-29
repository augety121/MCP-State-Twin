package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/augety121/mcp-state-twin/internal/hostcompat"
)

func TestCompatibilityRejectsUnknownRedactionBeforeOutput(t *testing.T) {
	r, file := compatibilityFixture(t)
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	for _, replacement := range []string{`"secretsDetected":null`, `"secretsDetected":"off"`} {
		input := strings.Replace(string(raw), `"secretsDetected":false`, replacement, 1)
		if err := os.WriteFile(file, []byte(input), 0600); err != nil {
			t.Fatal(err)
		}
		for _, command := range []string{"validate", "assess"} {
			args := []string{command, "--report", file}
			if command == "assess" {
				args = append(args, "--at", r.Metadata.CreatedAt, "--target", filepath.Join("..", "..", "internal", "hostcompat", "testdata", "synthetic-target.yaml"), "--require-current")
			}
			var output bytes.Buffer
			if err := runCompatibilityTo(args, &output); err == nil || output.Len() != 0 {
				t.Fatal("unknown redaction produced successful CLI output", command)
			}
		}
		after, err := os.ReadFile(file)
		if err != nil || string(after) != input {
			t.Fatal("rejected input modified", err)
		}
	}
}

func compatibilityFixture(t *testing.T) (*hostcompat.Report, string) {
	t.Helper()
	r, err := hostcompat.Load(filepath.Join("..", "..", "internal", "hostcompat", "testdata", "synthetic-report.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	return r, filepath.Join(t.TempDir(), "report.json")
}

func TestCompatibilityAssessmentCLI(t *testing.T) {
	for _, tt := range []struct {
		name, level, at   string
		gate, wantFailure bool
		freshness         string
	}{
		{"current", "verified", "2026-08-24T00:00:00Z", true, false, "within_window"},
		{"expired diagnostic", "verified", "2026-09-23T00:00:00Z", false, false, "expired"},
		{"expired gate", "verified", "2026-09-23T00:00:00Z", true, true, "expired"},
		{"future", "verified", "2026-08-23T00:00:00Z", true, true, "not_yet_valid"},
		{"experimental", "experimental", "2026-08-24T00:00:00Z", true, true, "within_window"},
		{"regressed", "regressed", "2026-08-24T00:00:00Z", true, true, "within_window"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r, file := compatibilityFixture(t)
			r.Claim.Level = tt.level
			raw, _ := json.Marshal(r)
			if err := os.WriteFile(file, raw, 0600); err != nil {
				t.Fatal(err)
			}
			args := []string{"assess", "--report", file, "--at", tt.at}
			if tt.gate {
				args = append(args, "--require-fresh")
			}
			var output bytes.Buffer
			err := runCompatibilityTo(args, &output)
			if (err != nil) != tt.wantFailure {
				t.Fatal("incorrect exit", err)
			}
			if tt.wantFailure && err.Error() != "HOST_REPORT_NOT_TIME_ELIGIBLE" {
				t.Fatal(err)
			}
			var a hostcompat.Assessment
			if err := json.Unmarshal(output.Bytes(), &a); err != nil {
				t.Fatal(err)
			}
			if a.Freshness != tt.freshness || a.PublicationAllowed || a.Provenance != "not_verified" || a.ScopeStatus != "not_checked" {
				t.Fatalf("bad report: %+v", a)
			}
			after, err := os.ReadFile(file)
			if err != nil || !bytes.Equal(raw, after) {
				t.Fatal("source changed", err)
			}
		})
	}
}

func TestCompatibilityCLIAdmissionAndOutputFailure(t *testing.T) {
	r, file := compatibilityFixture(t)
	raw, _ := json.Marshal(r)
	if err := os.WriteFile(file, raw, 0600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"assess"}, {"assess", "--report", file},
		{"assess", "--report", file, "--at", "now"},
		{"assess", "--report", file, "--at", r.Metadata.CreatedAt, "extra"},
		{"validate", "--report", file, "--require-fresh"},
		{"assess", "--report", file, "--at", r.Metadata.CreatedAt, "--network"},
	} {
		var output bytes.Buffer
		if err := runCompatibilityTo(args, &output); err == nil || output.Len() != 0 {
			t.Fatal("invalid CLI emitted assessment")
		}
	}
	for _, command := range []string{"validate", "assess"} {
		args := []string{command, "--report", file}
		if command == "assess" {
			args = append(args, "--at", r.Metadata.CreatedAt)
		}
		if err := runCompatibilityTo(args, failingCompatibilityWriter{}); !errors.Is(err, io.ErrClosedPipe) {
			t.Fatal("write failure lost", err)
		}
	}
	var output bytes.Buffer
	if err := runCompatibilityTo([]string{"validate", "--report", file}, &output); err != nil {
		t.Fatal(err)
	}
	var validation map[string]any
	if err := json.Unmarshal(output.Bytes(), &validation); err != nil {
		t.Fatal(err)
	}
	if validation["valid"] != true || validation["validationScope"] != "structure-only" || validation["publicationAllowed"] != false || validation["provenance"] != "not_verified" {
		t.Fatal("validation overclaim")
	}
	// A still-in-window date must not bypass structural admission.
	r.MCP.ObservedSurfaceDigest = r.Runtime.SpecDigest
	raw, _ = json.Marshal(r)
	if err := os.WriteFile(file, raw, 0600); err != nil {
		t.Fatal(err)
	}
	output.Reset()
	if err := runCompatibilityTo([]string{"assess", "--report", file, "--at", r.Metadata.CreatedAt}, &output); err == nil || output.Len() != 0 {
		t.Fatal("invalid report assessed")
	}
}

type failingCompatibilityWriter struct{}

func (failingCompatibilityWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func compatibilityTargetFixture(t *testing.T) (*hostcompat.Target, string) {
	t.Helper()
	target, err := hostcompat.LoadTarget(filepath.Join("..", "..", "internal", "hostcompat", "testdata", "synthetic-target.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	return target, filepath.Join(t.TempDir(), "target.json")
}

func TestCompatibilityScopedCLI(t *testing.T) {
	for _, tt := range []struct {
		name, level, at        string
		change, current, fresh bool
		wantError              string
	}{
		{"matching current", "verified", "2026-08-24T00:00:00Z", false, true, false, ""},
		{"in-window drift", "verified", "2026-08-24T00:00:00Z", true, true, false, "HOST_REPORT_NOT_CURRENT"},
		{"expired match", "verified", "2026-09-26T00:00:00Z", false, true, false, "HOST_REPORT_NOT_CURRENT"},
		{"future match", "verified", "2026-08-23T00:00:00Z", false, true, false, "HOST_REPORT_NOT_CURRENT"},
		{"experimental match", "experimental", "2026-08-24T00:00:00Z", false, true, false, "HOST_REPORT_NOT_CURRENT"},
		{"regressed match", "regressed", "2026-08-24T00:00:00Z", false, true, false, "HOST_REPORT_NOT_CURRENT"},
		{"diagnostic drift", "verified", "2026-08-24T00:00:00Z", true, false, false, ""},
		{"legacy time gate", "verified", "2026-08-24T00:00:00Z", true, false, true, ""},
		{"both gates stale", "verified", "2026-09-26T00:00:00Z", false, true, true, "HOST_REPORT_NOT_CURRENT"},
		{"both gates drift", "verified", "2026-08-24T00:00:00Z", true, true, true, "HOST_REPORT_NOT_CURRENT"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r, reportFile := compatibilityFixture(t)
			target, targetFile := compatibilityTargetFixture(t)
			r.Claim.Level = tt.level
			if tt.change {
				target.Runtime.Version = "synthetic-v2"
			}
			reportBytes, _ := json.Marshal(r)
			targetBytes, _ := json.Marshal(target)
			if err := os.WriteFile(reportFile, reportBytes, 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(targetFile, targetBytes, 0600); err != nil {
				t.Fatal(err)
			}
			args := []string{"assess", "--report", reportFile, "--target", targetFile, "--at", tt.at}
			if tt.current {
				args = append(args, "--require-current")
			}
			if tt.fresh {
				args = append(args, "--require-fresh")
			}
			var output bytes.Buffer
			err := runCompatibilityTo(args, &output)
			if tt.wantError == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || err.Error() != tt.wantError {
				t.Fatal("gate exit mismatch", err)
			}
			var a hostcompat.Assessment
			if err := json.Unmarshal(output.Bytes(), &a); err != nil {
				t.Fatal(err)
			}
			if a.Scope == nil || a.PublicationAllowed || a.Provenance != "not_verified" {
				t.Fatal("scope absent or overclaim")
			}
			if tt.change && (a.ScopeStatus != "mismatched" || a.Scope.ClaimCurrentEligible || len(a.Scope.Mismatches) != 1 || a.Scope.Mismatches[0] != "runtime.version") {
				t.Fatal("drift not reported")
			}
			for file, before := range map[string][]byte{reportFile: reportBytes, targetFile: targetBytes} {
				after, err := os.ReadFile(file)
				if err != nil || !bytes.Equal(before, after) {
					t.Fatal("source changed", err)
				}
			}
		})
	}
}

func TestCompatibilityScopedCLIRefusalAndOutputFailure(t *testing.T) {
	r, reportFile := compatibilityFixture(t)
	target, targetFile := compatibilityTargetFixture(t)
	for file, value := range map[string]any{reportFile: r, targetFile: target} {
		raw, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	var output bytes.Buffer
	err := runCompatibilityTo([]string{"assess", "--report", "does-not-exist", "--at", r.Metadata.CreatedAt, "--require-current"}, &output)
	if err == nil || err.Error() != "--target is required for --require-current" || output.Len() != 0 {
		t.Fatal("missing target did not fail before IO", err)
	}
	args := []string{"assess", "--report", reportFile, "--target", targetFile, "--at", r.Metadata.CreatedAt, "--require-current"}
	if err := runCompatibilityTo(args, failingCompatibilityWriter{}); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatal("lost output error", err)
	}
	if err := os.WriteFile(targetFile, []byte(`{"adapter":"unknown"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := runCompatibilityTo(args, &output); err == nil || output.Len() != 0 {
		t.Fatal("invalid target assessed")
	}
	for _, args := range [][]string{
		{"validate", "--report", reportFile, "--target", targetFile},
		{"assess", "--report", reportFile, "--target", targetFile, "--require-current"},
	} {
		if err := runCompatibilityTo(args, &output); err == nil || output.Len() != 0 {
			t.Fatal("invalid CLI accepted")
		}
	}
}
