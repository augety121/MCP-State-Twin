package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/augety121/mcp-state-twin/internal/hostcompat"
)

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
