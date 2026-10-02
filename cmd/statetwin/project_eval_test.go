package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/augety121/mcp-state-twin/internal/agenteval"
	"github.com/augety121/mcp-state-twin/internal/bundle"
)

type projectShortWriter struct{ fail bool }

func (w projectShortWriter) Write(p []byte) (int, error) {
	if w.fail {
		return 0, errors.New("synthetic write failure")
	}
	return len(p) / 2, nil
}

func TestRegressionCampaignCLI(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join("..", "..", "examples")
	err := filepath.WalkDir(source, func(n string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, e := filepath.Rel(source, n)
		if e != nil {
			return e
		}
		dest := filepath.Join(root, "examples", rel)
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return os.MkdirAll(dest, 0700)
		}
		if !strings.HasSuffix(n, ".json") && !strings.HasSuffix(n, ".yaml") {
			return nil
		}
		raw, e := os.ReadFile(n)
		if e != nil {
			return e
		}
		return os.WriteFile(dest, raw, 0600)
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, domain := range []string{"issue-tracker", "package-registry"} {
		dir := filepath.Join(root, "examples", domain)
		os.Mkdir(filepath.Join(dir, ".statetwin"), 0700)
		worlds := []string{"agent"}
		if domain == "package-registry" {
			worlds = append(worlds, "project")
		}
		for _, world := range worlds {
			for _, reviewed := range []bool{false, true} {
				source := filepath.Join(dir, "bundle-"+world+".yaml")
				out := world + "-world.stb"
				if reviewed {
					source = filepath.Join(dir, "reviewed", "world", world, "bundle-"+world+".yaml")
					out = "reviewed-" + out
				}
				if _, err := bundle.Build(source, filepath.Join(dir, ".statetwin", out)); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	ctx := context.Background()
	var output bytes.Buffer
	args := []string{"--root", root, "--campaign", "examples/evaluation-regression.json"}
	if err := runEvaluationProjectTo(ctx, "campaign-check", args, &output); err != nil {
		t.Fatal(err, output.String())
	}
	args = append(args, "--out", "delivery")
	output.Reset()
	err = runEvaluationProjectTo(ctx, "campaign-run", args, &output)
	var report agenteval.CampaignReport
	if err == nil || err.Error() != "CAMPAIGN_ASSESSMENT_FAILED" || json.Unmarshal(output.Bytes(), &report) != nil || report.Lifecycle != "published" || report.Decision != "failed" || report.CompletedProjects != 4 || report.TrialCounts.Completed != 48 {
		t.Fatal(err, output.String())
	}
	for _, row := range report.Projects {
		want := "passed"
		if strings.HasSuffix(row.ProjectID, "regression") {
			want = "failed"
		}
		if row.Decision != want {
			t.Fatal(row)
		}
	}
	for _, group := range []struct{ id, task string }{{"issue-regression", "issue-close-with-comment"}, {"registry-regression", "pkg-publish-then-install"}} {
		raw, err := os.ReadFile(filepath.Join(root, "delivery", "projects", group.id, "project-report.json"))
		if err != nil {
			t.Fatal(err)
		}
		var p agenteval.ProjectReport
		if json.Unmarshal(raw, &p) != nil || len(p.BaseAssessment.FailedChecks) != 1 || p.BaseAssessment.FailedChecks[0].TaskID != group.task || p.BaseAssessment.FailedChecks[0].CheckID != "objective" {
			t.Fatal(string(raw))
		}
	}
	output.Reset()
	if err := runEvaluationProjectTo(ctx, "campaign-inspect", args, &output); err != nil {
		t.Fatal(err, output.String())
	}
	var inspected agenteval.CampaignInspection
	if json.Unmarshal(output.Bytes(), &inspected) != nil || inspected.Verification != "consistent" || inspected.HistoricalDecision != "failed" || inspected.CheckedProjects != 4 {
		t.Fatal(output.String())
	}
}

func TestProjectReportProjection(t *testing.T) {
	report := &agenteval.ProjectReport{Decision: "failed", Lifecycle: "published", CaseCounts: agenteval.ProjectCounts{Planned: 3, Completed: 3, Started: 3}, TrialCounts: agenteval.ProjectCounts{Planned: 2, NotStarted: 2}, CaseFailures: []agenteval.CaseRow{{CaseID: "case-one", TaskID: "task-one", State: "mismatched", FailedCheckIDs: []string{"unsafe|`\n<script>"}}}, BaseAssessment: agenteval.ProjectAssessment{FailedChecks: []agenteval.ProjectCheckFailure{{TaskID: "task-one", TrialID: "candidate-01", CheckID: "objective", Category: "goal"}}}}
	raw, _ := json.MarshalIndent(report, "", "  ")
	md := projectMarkdown("project-run", report, string(raw))
	for _, want := range []string{"**failed**", "**published**", "| Cases | 3 | 3 | 3 | 0 | 0 |", "| Trials | 2 | 0 | 0 | 0 | 2 |", "candidate-01", "objective", `unsafe\|&#96; &lt;script&gt;`, "provenance not proven"} {
		if !strings.Contains(md, want) {
			t.Fatal(want, md)
		}
	}
	if strings.Contains(md, "<script>") {
		t.Fatal("unsafe table")
	}
	for _, command := range []string{"project-check", "project-run", "project-inspect", "campaign-check", "campaign-run", "campaign-inspect"} {
		for _, args := range [][]string{{}, {"--unknown"}, {"--project", "p.json", "--out", "x", "--format", "bad"}} {
			if err := runEvaluationProjectTo(context.Background(), command, args, &bytes.Buffer{}); err == nil {
				t.Fatal(command, args)
			}
		}
	}
}

func TestProjectGuideCLI(t *testing.T) {
	root := preparePackageCLI(t)
	if _, err := bundle.Build(filepath.Join(root, "reviewed", "world", "agent", "bundle-agent.yaml"), filepath.Join(root, ".statetwin", "reviewed-agent-world.stb")); err != nil {
		t.Fatal(err)
	}
	inputNames := []string{"project-core.json", "agent-suite.json", "agent-cases.json", "reviewed/catalog-core.json", "reviewed/worlds-core.json"}
	inputs := map[string]string{}
	for _, n := range inputNames {
		b, err := os.ReadFile(filepath.Join(root, n))
		if err != nil {
			t.Fatal(err)
		}
		inputs[n] = string(b)
	}
	ctx := context.Background()
	var out bytes.Buffer
	if err := runEvaluationProjectTo(ctx, "project-check", []string{"--root", root, "--project", "project-core.json"}, &out); err != nil {
		t.Fatal(err, out.String())
	}
	var check agenteval.ProjectCheck
	if json.Unmarshal(out.Bytes(), &check) != nil || check.Status != "statically_valid" || check.ExecutionPerformed {
		t.Fatal(out.String())
	}
	out.Reset()
	if err := runEvaluationProjectTo(ctx, "project-run", []string{"--root", root, "--project", "project-core.json", "--out", "delivery", "--format", "markdown"}, &out); err != nil || !strings.Contains(out.String(), "**passed**") {
		t.Fatal(err, out.String())
	}
	out.Reset()
	if err := runEvaluationProjectTo(ctx, "project-inspect", []string{"--root", root, "--project", "project-core.json", "--out", "delivery"}, &out); err != nil {
		t.Fatal(err, out.String())
	}
	var inspection agenteval.ProjectInspection
	if json.Unmarshal(out.Bytes(), &inspection) != nil || inspection.Verification != "consistent" || inspection.QualificationVerification != "recorded_only" {
		t.Fatal(out.String())
	}
	for _, command := range []string{"suite-export", "suite-import"} {
		args := []string{"--root", root, "--archive", "suite.tar", "--out", "delivery/suite"}
		if command == "suite-import" {
			args[len(args)-1] = "restored"
		}
		if raw, err := captureDelivery(t, command, args...); err != nil {
			t.Fatal(string(raw), err)
		}
	}
	if raw, err := captureDelivery(t, "suite-verify", "--root", root, "--out", "restored"); err != nil {
		t.Fatal(string(raw), err)
	}
	for _, fail := range []bool{false, true} {
		err := runEvaluationProjectTo(ctx, "project-inspect", []string{"--root", root, "--project", "project-core.json", "--out", "delivery"}, projectShortWriter{fail})
		if err == nil || err.Error() != "PROJECT_OUTPUT_FAILED" {
			t.Fatal(err)
		}
	}
	// Output failure takes precedence even when admission already failed.
	if err := runEvaluationProjectTo(ctx, "project-check", []string{"--root", root, "--project", "missing.json"}, projectShortWriter{}); err == nil || err.Error() != "PROJECT_OUTPUT_FAILED" {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "delivery", "project-report.json")); err != nil {
		t.Fatal("stdout error removed published result")
	}
	for n, original := range inputs {
		b, err := os.ReadFile(filepath.Join(root, n))
		if err != nil || string(b) != original {
			t.Fatal("source changed", n, err)
		}
	}
}
