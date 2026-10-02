package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/augety121/mcp-state-twin/internal/agenteval"
)

func runEvaluationProject(parent context.Context, command string, args []string) error {
	return runEvaluationProjectTo(parent, command, args, os.Stdout)
}
func runEvaluationProjectTo(parent context.Context, command string, args []string, output io.Writer) error {
	prefix := "PROJECT"
	isCampaign := strings.HasPrefix(command, "campaign-")
	if isCampaign {
		prefix = "CAMPAIGN"
	}
	f := flag.NewFlagSet(command, flag.ContinueOnError)
	f.SetOutput(io.Discard)
	root := f.String("root", ".", "trusted local input root")
	format := f.String("format", "json", "json or markdown")
	refName := "project"
	if isCampaign {
		refName = "campaign"
	}
	name := f.String(refName, "", "relative manifest")
	out := ""
	if !strings.HasSuffix(command, "-check") {
		f.StringVar(&out, "out", "", "new output directory or directory to inspect")
	}
	if f.Parse(args) != nil || f.NArg() != 0 || *name == "" || (*format != "json" && *format != "markdown") || !strings.HasSuffix(command, "-check") && out == "" {
		return errors.New(prefix + "_INPUT_INVALID")
	}
	ctx, stop := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
	defer stop()
	var value any
	var err error
	gate := false
	switch command {
	case "project-check":
		r, e := agenteval.CheckProject(ctx, *root, *name)
		err = e
		if r != nil {
			value = r
			gate = r.Status == "statically_valid"
		}
	case "project-run":
		r, e := agenteval.RunProject(ctx, *root, *name, out)
		err = e
		if r != nil {
			value = r
			gate = r.Decision == "passed" && r.Lifecycle == "published"
		}
	case "project-inspect":
		r, e := agenteval.InspectProject(ctx, *root, *name, out)
		err = e
		if r != nil {
			value = r
			gate = r.Verification == "consistent"
		}
	case "campaign-check":
		r, e := agenteval.CheckCampaign(ctx, *root, *name)
		err = e
		if r != nil {
			value = r
			gate = r.Status == "statically_valid"
		}
	case "campaign-run":
		r, e := agenteval.RunCampaign(ctx, *root, *name, out)
		err = e
		if r != nil {
			value = r
			gate = r.Decision == "passed" && r.Lifecycle == "published"
		}
	case "campaign-inspect":
		r, e := agenteval.InspectCampaign(ctx, *root, *name, out)
		err = e
		if r != nil {
			value = r
			gate = r.Verification == "consistent"
		}
	default:
		return errors.New(prefix + "_INPUT_INVALID")
	}
	if value != nil {
		raw, e := json.MarshalIndent(value, "", "  ")
		if e != nil || len(raw) > 1<<20 {
			return errors.New(prefix + "_RESOURCE_LIMIT")
		}
		text := string(raw) + "\n"
		if *format == "markdown" {
			text = projectMarkdown(command, value, string(raw))
		}
		n, e := io.WriteString(output, text)
		if e != nil || n != len(text) {
			return errors.New(prefix + "_OUTPUT_FAILED")
		}
	}
	if err != nil {
		return err
	}
	if !gate {
		return errors.New(prefix + "_ASSESSMENT_FAILED")
	}
	return nil
}
func projectMarkdown(command string, value any, raw string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", command)
	counts := func(label string, c agenteval.ProjectCounts) {
		fmt.Fprintf(&b, "| %s | %d | %d | %d | %d | %d |\n", label, c.Planned, c.Started, c.Completed, c.Failed, c.NotStarted)
	}
	header := func() {
		b.WriteString("\n| Scope | Planned | Started | Completed | Failed | Not started |\n|---|---|---|---|---|---|\n")
	}
	stages := func(rows []agenteval.ProjectStage) {
		b.WriteString("\n| Stage | Status | Reason |\n|---|---|---|\n")
		for _, r := range rows {
			fmt.Fprintf(&b, "| %s | %s | %s |\n", r.Stage, r.Status, r.ReasonCode)
		}
	}
	switch r := value.(type) {
	case *agenteval.ProjectReport:
		fmt.Fprintf(&b, "Decision: **%s**. Publication: **%s**.\n", r.Decision, r.Lifecycle)
		header()
		counts("Cases", r.CaseCounts)
		counts("Trials", r.TrialCounts)
		stages(r.Stages)
		b.WriteString("\n| Case | Task | State | Expected | Actual | Failed checks |\n|---|---|---|---|---|---|\n")
		for _, row := range r.CaseFailures {
			fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s |\n", markdownCell(row.CaseID), markdownCell(row.TaskID), row.State, row.ExpectedOutcome, row.ActualOutcome, markdownCell(strings.Join(row.FailedCheckIDs, ", ")))
		}
		b.WriteString("\n| Task | Trial | Failed assertion | Category |\n|---|---|---|---|\n")
		for _, row := range r.BaseAssessment.FailedChecks {
			fmt.Fprintf(&b, "| %s | %s | %s | %s |\n", markdownCell(row.TaskID), markdownCell(row.TrialID), markdownCell(row.CheckID), markdownCell(row.Category))
		}
		if r.BaseAssessment.Result != nil {
			b.WriteString("\n| Task | Trial | Binding | Differences |\n|---|---|---|---|\n")
			for _, row := range r.TaskBinding.Trials {
				fmt.Fprintf(&b, "| %s | %s | %s | %s |\n", row.TaskID, row.TrialID, row.Status, strings.Join(row.Differences, ", "))
			}
		}
	case *agenteval.CampaignReport:
		fmt.Fprintf(&b, "Decision: **%s**. Publication: **%s**. Projects: %d planned, %d completed, %d not started.\n", r.Decision, r.Lifecycle, r.PlannedProjects, r.CompletedProjects, r.NotStartedProjects)
		header()
		counts("Cases", r.CaseCounts)
		counts("Trials", r.TrialCounts)
		b.WriteString("\n| Project | Publication | Decision | Reasons |\n|---|---|---|---|\n")
		for _, row := range r.Projects {
			fmt.Fprintf(&b, "| %s | %s | %s | %s |\n", row.ProjectID, row.Lifecycle, row.Decision, strings.Join(row.ReasonCodes, ", "))
		}
	case *agenteval.ProjectCheck:
		fmt.Fprintf(&b, "Admission: **%s**. Planned cases: %d. Planned trials: %d.\n", r.Status, r.PlannedCases, r.PlannedTrials)
		stages(r.Stages)
	case *agenteval.CampaignCheck:
		fmt.Fprintf(&b, "Admission: **%s**. Planned projects: %d.\n", r.Status, r.PlannedProjects)
		for _, row := range r.Projects {
			fmt.Fprintf(&b, "\nProject: %s — %s\n", row.ProjectID, row.Status)
			stages(row.Stages)
		}
	case *agenteval.ProjectInspection:
		fmt.Fprintf(&b, "Verification: **%s**. Historical decision: **%s**. Qualification: **recorded_only**.\n", r.Verification, r.HistoricalDecision)
	case *agenteval.CampaignInspection:
		fmt.Fprintf(&b, "Verification: **%s**. Historical decision: **%s**. Projects: %d/%d checked. Qualification: **recorded_only**.\n", r.Verification, r.HistoricalDecision, r.CheckedProjects, r.PlannedProjects)
	}
	b.WriteString("\nSynthetic offline inputs; provenance not proven. Case matches are not Agent successes. Content binding is not source authentication. Inspection does not rerun qualification.\n\n## Complete diagnostic\n\n```json\n")
	b.WriteString(strings.ReplaceAll(raw, "`", `\u0060`))
	b.WriteString("\n```\n")
	return b.String()
}

func markdownCell(s string) string {
	return strings.NewReplacer("\\", "\\\\", "|", "\\|", "\r", " ", "\n", " ", "`", "&#96;", "<", "&lt;", ">", "&gt;").Replace(s)
}
