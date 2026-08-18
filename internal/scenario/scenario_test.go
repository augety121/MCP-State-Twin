package scenario

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/augety121/mcp-state-twin/internal/spec"
	"github.com/augety121/mcp-state-twin/internal/world"
)

func loadReference(t *testing.T) (*spec.TwinSpec, *world.State) {
	t.Helper()
	root := filepath.Join("..", "..")
	twin, err := spec.Load(filepath.Join(root, "examples", "issue-tracker", "twin.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "examples", "issue-tracker", "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	var state world.State
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatal(err)
	}
	return twin, &state
}

func TestDecodeRejectsUnknownFieldsAliasesAndInvalidPointers(t *testing.T) {
	base := `apiVersion: statetwin.dev/v1alpha1
kind: Scenario
metadata: {name: close-issue}
steps:
  - id: close
    tool: close_issue
    input: {number: 1}
    expect:
      errorClass: ""
      assertions:
        - {path: /entities/issue, operator: exists}
`
	if _, err := Decode([]byte(base + "unknown: true\n")); err == nil || !strings.Contains(err.Error(), "field unknown") {
		t.Fatalf("expected unknown-field rejection, got %v", err)
	}
	if _, err := Decode([]byte(strings.Replace(base, "input: {number: 1}", "input: &args {number: 1}\n    copy: *args", 1))); err == nil || !strings.Contains(err.Error(), "anchors and aliases") {
		t.Fatalf("expected alias rejection, got %v", err)
	}
	invalidPointer := strings.Replace(base, "/entities/issue", "/entities/~2issue", 1)
	if _, err := Decode([]byte(invalidPointer)); err == nil || !strings.Contains(err.Error(), "invalid JSON Pointer") {
		t.Fatalf("expected pointer rejection, got %v", err)
	}
	invalidClass := strings.Replace(base, `errorClass: ""`, "errorClass: MADE_UP", 1)
	if _, err := Decode([]byte(invalidClass)); err == nil || !strings.Contains(err.Error(), "canonical error class") {
		t.Fatalf("expected error-class rejection, got %v", err)
	}
	timestampValue := strings.Replace(base,
		"        - {path: /entities/issue, operator: exists}",
		"        - path: /entities/issue\n          operator: equals\n          value: 2026-08-18", 1)
	if _, err := Decode([]byte(timestampValue)); err == nil || !strings.Contains(err.Error(), "JSON value domain") {
		t.Fatalf("expected non-JSON YAML value rejection, got %v", err)
	}
}

func TestRunProducesDeterministicPassingReport(t *testing.T) {
	twin, state := loadReference(t)
	scenario := &Scenario{
		APIVersion: APIVersion, Kind: Kind, Metadata: Metadata{Name: "close-issue"},
		Steps: []Step{{
			ID: "close", Tool: "close_issue",
			Input: map[string]any{"owner": "octo", "repository": "demo", "number": 1},
			Expect: Expectation{Assertions: []Assertion{{
				Path: "/entities/issue/octo~1demo#1/state", Operator: "equals", Value: "closed",
			}}},
		}},
		FinalAssertions: []Assertion{{Path: "/entities/comment", Operator: "exists"}},
	}
	if err := scenario.Validate(); err != nil {
		t.Fatal(err)
	}
	report, err := Run(context.Background(), twin, state, scenario, "test")
	if err != nil {
		t.Fatal(err)
	}
	if !report.Passed || len(report.OrderedToolTrace) != 1 || len(report.StateDiff) == 0 {
		t.Fatalf("unexpected report: %#v", report)
	}
	if report.EnvironmentDigest == "" || report.InitialStateDigest == report.TerminalStateDigest {
		t.Fatalf("invalid report digests: %#v", report)
	}
	if state.Entities["issue"]["octo/demo#1"]["state"] != "open" {
		t.Fatal("scenario runner mutated the caller-owned initial state")
	}

	repeated, err := Run(context.Background(), twin, state, scenario, "test")
	if err != nil {
		t.Fatal(err)
	}
	if repeated.EnvironmentDigest != report.EnvironmentDigest || repeated.TerminalStateDigest != report.TerminalStateDigest {
		t.Fatal("same scenario did not reproduce environment and terminal digests")
	}
}

func TestRunTreatsExpectedDomainErrorAsPassing(t *testing.T) {
	twin, state := loadReference(t)
	scenario := &Scenario{
		APIVersion: APIVersion, Kind: Kind, Metadata: Metadata{Name: "expected-conflict"},
		Steps: []Step{
			{ID: "close", Tool: "close_issue", Input: map[string]any{"owner": "octo", "repository": "demo", "number": 1}},
			{ID: "close-again", Tool: "close_issue", Input: map[string]any{"owner": "octo", "repository": "demo", "number": 1}, Expect: Expectation{ErrorClass: "CONFLICT"}},
		},
	}
	report, err := Run(context.Background(), twin, state, scenario, "test")
	if err != nil {
		t.Fatal(err)
	}
	if !report.Passed || len(report.OrderedToolTrace) != 2 || report.OrderedToolTrace[1].ErrorClass != "CONFLICT" {
		t.Fatalf("expected modeled conflict to pass: %#v", report)
	}
}

func TestRunFailsClosedOnUnexpectedErrorClass(t *testing.T) {
	twin, state := loadReference(t)
	scenario := &Scenario{
		APIVersion: APIVersion, Kind: Kind, Metadata: Metadata{Name: "unexpected-error"},
		Steps: []Step{
			{ID: "missing", Tool: "get_issue", Input: map[string]any{"owner": "octo", "repository": "demo", "number": 404}},
			{ID: "must-not-run", Tool: "list_issues", Input: map[string]any{"owner": "octo", "repository": "demo"}},
		},
	}
	report, err := Run(context.Background(), twin, state, scenario, "test")
	if err != nil {
		t.Fatal(err)
	}
	if report.Passed || len(report.Failures) != 1 || len(report.OrderedToolTrace) != 1 || report.OrderedToolTrace[0].ErrorClass != "NOT_FOUND" {
		t.Fatalf("unexpected failure report: %#v", report)
	}
}
