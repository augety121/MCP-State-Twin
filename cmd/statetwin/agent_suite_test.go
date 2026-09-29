package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/augety121/mcp-state-twin/internal/agenteval"
)

func TestSuiteCLIOutputFailure(t *testing.T) {
	root := prepareOfflineCLI(t)
	p := agenteval.SuitePlan{Format: agenteval.SuiteFormat, BaselineModel: "mock-base", CandidateModel: "mock-candidate", MaxOutputTokens: 1024, Pairs: []agenteval.SuitePair{{TaskID: "close-issue", Task: "agent-tasks/close-issue.json", Repeat: 1, BaselineResponses: "agent-mocks/close-issue.json", CandidateResponses: "agent-mocks/close-issue.json"}}}
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(root, "suite.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	out, err := os.Create(filepath.Join(t.TempDir(), "closed-output"))
	if err != nil {
		t.Fatal(err)
	}
	if err = out.Close(); err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = out
	defer func() { os.Stdout = old }()
	for _, command := range []string{"suite-preflight", "suite"} {
		args := []string{command, "--root", root, "--suite", "suite.json"}
		if command == "suite" {
			args = append(args, "--out", ".statetwin/output-failure")
		}
		if err = runAgentEval(context.Background(), args); err == nil {
			t.Fatal("stdout failure swallowed", command)
		}
	}
	for _, command := range []string{"suite-inspect", "suite-verify"} {
		for _, format := range []string{"json", "markdown"} {
			if err := runAgentEval(context.Background(), []string{command, "--root", root, "--out", ".statetwin/output-failure", "--format", format}); err == nil || err.Error() == "SUITE_GATE_NOT_SATISFIED" {
				t.Fatal("audit stdout failure swallowed", command, format, err)
			}
		}
	}
}

func TestOfflineSuiteCLI(t *testing.T) {
	root := prepareOfflineCLI(t)
	source := filepath.Join("..", "..", "examples", "issue-tracker")
	copyFile := func(name string) {
		t.Helper()
		raw, err := os.ReadFile(filepath.Join(source, filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(root, filepath.FromSlash(name)), raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	copyFile("agent-suite.json")
	for _, id := range []string{"read-issue", "close-issue", "create-issue", "already-closed", "scope-protection", "after-commit-confirm"} {
		copyFile("agent-tasks/" + id + ".json")
		copyFile("agent-mocks/" + id + ".json")
	}
	raw, err := captureAgentEval(t, []string{"suite-preflight", "--root", root, "--suite", "agent-suite.json"})
	if err != nil {
		t.Fatal(err)
	}
	var pre agenteval.SuitePreflight
	if json.Unmarshal(raw, &pre) != nil || pre.PlannedTrials != 12 {
		t.Fatalf("preflight: %s", raw)
	}
	entries, err := os.ReadDir(filepath.Join(root, ".statetwin"))
	if err != nil || len(entries) != 1 || entries[0].Name() != "agent-world.stb" {
		t.Fatal("preflight wrote output", err)
	}
	raw, err = captureAgentEval(t, []string{"suite", "--root", root, "--suite", "agent-suite.json", "--out", ".statetwin/suite"})
	if err != nil {
		t.Fatal(err)
	}
	var report agenteval.SuiteReport
	if json.Unmarshal(raw, &report) != nil || report.Comparison == nil || len(report.Trials) != 12 || report.Comparison.Decision != "no_regression_observed" {
		t.Fatalf("report: %s", raw)
	}
	suiteRoot := filepath.Join(root, ".statetwin", "suite")
	checkSuiteAuditCLI(t, root, ".statetwin/suite", true)
	for _, row := range report.Trials {
		if row.State != "completed" {
			t.Fatal(row)
		}
		if _, err = captureAgentEval(t, []string{"verify", "--root", suiteRoot, "--evidence", row.TrialID + "/terminal.json"}); err != nil {
			t.Fatal(row.TrialID, err)
		}
	}
	for _, format := range []string{"json", "markdown"} {
		if _, err = captureAgentEval(t, []string{"compare", "--root", suiteRoot, "--plan", "plan.json", "--format", format}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = captureAgentEval(t, []string{"suite", "--root", root, "--suite", "agent-suite.json", "--out", ".statetwin/suite"}); err == nil {
		t.Fatal("overwrote suite")
	}
	planRaw, err := os.ReadFile(filepath.Join(root, "agent-suite.json"))
	if err != nil {
		t.Fatal(err)
	}
	var plan agenteval.SuitePlan
	if err = json.Unmarshal(planRaw, &plan); err != nil {
		t.Fatal(err)
	}
	plan.Pairs[1].CandidateResponses = "agent-mocks/omit-action.json"
	writePlan := func() {
		t.Helper()
		data, err := json.Marshal(plan)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(root, "agent-suite.json"), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	writePlan()
	raw, err = captureAgentEval(t, []string{"suite", "--root", root, "--suite", "agent-suite.json", "--out", ".statetwin/regression"})
	if err == nil || json.Unmarshal(raw, &report) != nil || report.Comparison.Decision != "regression" {
		t.Fatalf("regression gate: %s %v", raw, err)
	}
	for _, row := range report.Trials {
		if row.State != "completed" {
			t.Fatal("task failure stopped suite", row)
		}
	}
	plan.Pairs[5].CandidateResponses = "missing.json"
	checkSuiteAuditCLI(t, root, ".statetwin/regression", false)
	writePlan()
	if raw, err = captureAgentEval(t, []string{"suite", "--root", root, "--suite", "agent-suite.json", "--out", ".statetwin/invalid"}); err == nil || len(raw) != 0 {
		t.Fatal("late invalid input admitted")
	}
	if _, err = os.Stat(filepath.Join(root, ".statetwin", "invalid")); !os.IsNotExist(err) {
		t.Fatal("late rejection wrote evidence")
	}
	cleanReport, err := os.ReadFile(filepath.Join(suiteRoot, "report.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(suiteRoot, "report.pending.json"), cleanReport, 0600); err != nil {
		t.Fatal(err)
	}
	for _, command := range []string{"suite-inspect", "suite-verify"} {
		data, err := captureAgentEval(t, []string{command, "--root", root, "--out", ".statetwin/suite"})
		if (err == nil) != (command == "suite-inspect") || !strings.Contains(string(data), "published_with_residue") {
			t.Fatal("residue gate", command, err, string(data))
		}
	}
	// A saved plausible verdict must not bypass replay-backed verification.
	file := filepath.Join(root, ".statetwin", "regression", "report.json")
	report.Comparison.Decision = "no_regression_observed"
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, data, 0600); err != nil {
		t.Fatal(err)
	}
	for _, command := range []string{"suite-inspect", "suite-verify"} {
		data, err := captureAgentEval(t, []string{command, "--root", root, "--out", ".statetwin/regression"})
		if err == nil || err.Error() != "SUITE_DIRECTORY_INVALID" || !strings.Contains(string(data), "report_comparison_mismatch") {
			t.Fatal("tampered report admitted", err, string(data))
		}
	}
}

func checkSuiteAuditCLI(t *testing.T, root, out string, pass bool) {
	t.Helper()
	for _, command := range []string{"suite-inspect", "suite-verify"} {
		for _, format := range []string{"json", "markdown"} {
			data, err := captureAgentEval(t, []string{command, "--root", root, "--out", out, "--format", format})
			wantOK := command == "suite-inspect" || pass
			if (err == nil) != wantOK {
				t.Fatal(command, format, err)
			}
			if format == "json" {
				var r agenteval.SuiteInspection
				if json.Unmarshal(data, &r) != nil || r.ReportVerification != "matched" || r.CompleteTrials != 12 || r.RegressionGatePassed != pass {
					t.Fatalf("CLI audit: %s", data)
				}
			} else if !strings.Contains(string(data), "| baseline-06 | published | true | false |") || !strings.Contains(string(data), "# Offline Agent comparison") {
				t.Fatalf("markdown missing evidence: %s", data)
			}
		}
	}
}

func TestSuiteAuditCLIArgumentsAndMissing(t *testing.T) {
	root := t.TempDir()
	for _, command := range []string{"suite-inspect", "suite-verify"} {
		data, err := captureAgentEval(t, []string{command, "--root", root, "--out", "missing"})
		if (err == nil) != (command == "suite-inspect") || !strings.Contains(string(data), "not_started") {
			t.Fatal(command, err, string(data))
		}
		for _, args := range [][]string{{command}, {command, "--out", "missing", "--format", "html"}, {command, "--out", "missing", "positional"}, {command, "--endpoint", "https://example.invalid"}} {
			data, err := captureAgentEval(t, args)
			if err == nil || len(data) != 0 {
				t.Fatal("invalid CLI reached inspection", args, err)
			}
		}
	}
}
