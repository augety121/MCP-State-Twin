package main

import (
	"context"
	"errors"
	"flag"
	"fmt"

	"github.com/augety121/mcp-state-twin/internal/agenteval"
	"github.com/augety121/mcp-state-twin/internal/agenthost"
	"github.com/augety121/mcp-state-twin/internal/limits"
	"github.com/augety121/mcp-state-twin/internal/task"
)

func runAgentEval(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return errors.New("eval requires preflight, mock, verify, compare, live-plan, live-preflight, live, or live-verify")
	}
	command := args[0]
	if command == "live-plan" || command == "live-preflight" || command == "live" || command == "live-verify" {
		return runAgentLive(ctx, args)
	}
	if command != "preflight" && command != "mock" && command != "verify" && command != "compare" {
		return errors.New("unsupported eval command")
	}
	f := flag.NewFlagSet("eval "+command, flag.ContinueOnError)
	root := f.String("root", ".", "trusted artifact root")
	var taskName, configName, responsesName, outName, evidenceName, planName, format string
	switch command {
	case "preflight", "mock":
		f.StringVar(&taskName, "task", "", "relative AgentTask")
		f.StringVar(&configName, "config", "", "relative offline run configuration")
		if command == "mock" {
			f.StringVar(&responsesName, "responses", "", "relative synthetic Responses script")
			f.StringVar(&outName, "out", "", "new relative evidence directory; parent must exist")
		}
	case "verify":
		f.StringVar(&evidenceName, "evidence", "", "relative terminal artifact")
	case "compare":
		f.StringVar(&planName, "plan", "", "relative fixed comparison plan")
		f.StringVar(&format, "format", "json", "json or markdown")
	}
	if err := f.Parse(args[1:]); err != nil {
		return err
	}
	if f.NArg() != 0 {
		return errors.New("unexpected eval arguments")
	}
	if command == "verify" {
		data, err := task.ReadFile(*root, evidenceName, limits.MaxReportBytes)
		if err != nil {
			return err
		}
		e, err := agenteval.DecodeEvidence(data)
		if err != nil {
			return err
		}
		if err = agenteval.VerifyEvidence(ctx, e); err != nil {
			return err
		}
		return printJSON(map[string]any{"source": "mock-responses", "structure": "valid", "worldReplay": "matched", "grading": "matched", "providerProvenance": "not-proven", "trialId": e.Episode.Definition.Config.TrialID})
	}
	if command == "compare" {
		if format != "json" && format != "markdown" {
			return errors.New("unsupported report format")
		}
		data, err := task.ReadFile(*root, planName, 64<<10)
		if err != nil {
			return err
		}
		p, err := agenteval.DecodeCompare(data)
		if err != nil {
			return err
		}
		r, err := agenteval.Compare(ctx, *root, p)
		if err != nil {
			return err
		}
		if format == "markdown" {
			_, err = fmt.Print(r.Markdown())
		} else {
			err = printJSON(r)
		}
		if err != nil {
			return err
		}
		if r.Decision != "no_regression_observed" {
			return errors.New("offline comparison gate not satisfied")
		}
		return nil
	}
	t, b, err := agenteval.Load(*root, taskName)
	if err != nil {
		return err
	}
	data, err := task.ReadFile(*root, configName, 16<<10)
	if err != nil {
		return err
	}
	c, err := agenteval.DecodeRun(data)
	if err != nil {
		return err
	}
	if err = agenteval.Preflight(t, b, c); err != nil {
		return err
	}
	if command == "preflight" {
		return printJSON(map[string]any{"status": "offline-statically-valid", "live": false, "modelAvailability": "not-checked", "trialId": c.TrialID})
	}
	data, err = task.ReadFile(*root, responsesName, 8<<20)
	if err != nil {
		return err
	}
	m, err := agenthost.DecodeMock(data)
	if err != nil {
		return err
	}
	bundleBytes, err := task.ReadFile(*root, t.Bundle, limits.MaxBundleCompressed)
	if err != nil {
		return err
	}
	r, err := agenteval.RecordMock(ctx, *root, outName, t, bundleBytes, c, m)
	if err != nil {
		return err
	}
	if err = printJSON(map[string]any{"source": r.Source, "trialId": c.TrialID, "executionStatus": r.ExecutionStatus, "evidenceStatus": r.EvidenceStatus, "cleanupStatus": r.CleanupStatus, "evaluation": r.Evaluation, "usage": r.Usage}); err != nil {
		return err
	}
	if r.ExecutionStatus != "completed" || r.EvidenceStatus != "complete" || r.CleanupStatus != "complete" || r.Evaluation == nil || (r.Evaluation.Outcome != "success" && r.Evaluation.Outcome != "expected_abstention") {
		return errors.New("offline Agent evaluation did not pass")
	}
	return nil
}
