package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/augety121/mcp-state-twin/internal/agenteval"
)

func TestAssessmentRepeatExampleCLI(t *testing.T) {
	root := prepareOfflineCLI(t)
	for _, file := range []string{"agent-repeat-expectation.json", "agent-suite-repeat-negative.json", "agent-tasks/close-issue-revised.json"} {
		raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "issue-tracker", filepath.FromSlash(file)))
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(root, filepath.FromSlash(file)), raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := captureAgentEval(t, []string{"suite", "--root", root, "--suite", "agent-suite-repeat-negative.json", "--out", ".statetwin/repeats"}); err != nil {
		t.Fatal(err)
	}
	if _, err := captureAgentEval(t, []string{"suite-verify", "--root", root, "--out", ".statetwin/repeats"}); err != nil {
		t.Fatal("old gate must retain its meaning", err)
	}
	for _, policy := range []string{"candidate-pass-v1", "both-pass-v1"} {
		for _, format := range []string{"json", "markdown"} {
			data, err := captureAgentEval(t, []string{"suite-assess", "--root", root, "--out", ".statetwin/repeats", "--expect", "agent-repeat-expectation.json", "--policy", policy, "--format", format})
			if err == nil || err.Error() != "ASSESSMENT_GATE_NOT_SATISFIED" {
				t.Fatal("heterogeneous repeat passed", err)
			}
			if format == "json" {
				var r agenteval.SuiteAssessment
				if json.Unmarshal(data, &r) != nil || r.Decision != "failed" || r.Expectation.Status != "matched" || r.Consistency[0].IdentityStatus != "heterogeneous" {
					t.Fatalf("%s", data)
				}
			}
		}
	}
}

func TestAssessmentCLIAdmission(t *testing.T) {
	for _, args := range [][]string{
		{"suite-assess"},
		{"suite-assess", "--out", "missing", "--expect", "missing.json", "--policy", "unknown"},
		{"suite-assess", "--out", "missing", "--expect", "missing.json", "--policy", "both-pass-v1", "--format", "html"},
		{"suite-assess", "--out", "missing", "--expect", "missing.json", "--policy", "both-pass-v1", "extra"},
		{"suite-assess", "--endpoint", "https://example.invalid"},
	} {
		data, err := captureAgentEval(t, args)
		if err == nil || len(data) != 0 {
			t.Fatal("arguments reached assessment", err)
		}
	}
}
