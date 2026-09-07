package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"os"
	"time"

	"github.com/augety121/mcp-state-twin/internal/agentapi"
	"github.com/augety121/mcp-state-twin/internal/agenteval"
	"github.com/augety121/mcp-state-twin/internal/bundle"
	"github.com/augety121/mcp-state-twin/internal/limits"
	"github.com/augety121/mcp-state-twin/internal/task"
)

func runAgentLive(ctx context.Context, args []string) error {
	command := args[0]
	f := flag.NewFlagSet("eval "+command, flag.ContinueOnError)
	root := f.String("root", ".", "trusted local artifact root")
	var taskName, planName, evidenceName, id, model string
	var requests, tokens int
	var validFor time.Duration
	var allow bool
	switch command {
	case "live-plan":
		f.StringVar(&taskName, "task", "", "relative synthetic AgentTask")
		f.StringVar(&id, "id", "", "unique plan ID; output directory cannot be reused")
		f.StringVar(&model, "model", "", "explicit operator-selected Responses model; no default")
		f.IntVar(&requests, "max-requests", 0, "required maximum POST count")
		f.IntVar(&tokens, "max-output-tokens", 0, "required per-request output token cap")
		f.DurationVar(&validFor, "valid-for", 0, "required approval lifetime, up to 24h")
	case "live-preflight", "live":
		f.StringVar(&planName, "plan", "", "relative reviewed live plan")
		if command == "live" {
			f.BoolVar(&allow, "allow-live", false, "explicitly permit paid API requests under approved plan")
		}
	case "live-verify":
		f.StringVar(&evidenceName, "evidence", "", "relative live terminal evidence; replay is offline")
	}
	if err := f.Parse(args[1:]); err != nil {
		return err
	}
	if f.NArg() != 0 {
		return errors.New("unexpected eval arguments")
	}
	if command == "live-plan" {
		t, b, err := agenteval.Load(*root, taskName)
		if err != nil {
			return err
		}
		if validFor <= 0 || validFor > 24*time.Hour {
			return errors.New("LIVE_PLAN_INVALID")
		}
		now := time.Now().UTC()
		p := &agentapi.Plan{Format: agentapi.PlanFormat, ID: id, Provider: "openai", Profile: agentapi.Profile, Model: model, Task: t, BundleDigest: b.Digest, IssuedAt: now.Format(time.RFC3339Nano), ExpiresAt: now.Add(validFor).Format(time.RFC3339Nano), MaxRequests: requests, MaxOutputTokens: tokens, CostPolicy: agentapi.CostPolicy}
		if err = agenteval.PreflightLive(p, b); err != nil {
			return err
		}
		encoded, encodeErr := json.MarshalIndent(p, "", "  ")
		if encodeErr != nil || len(encoded)+1 > task.MaxBytes+(16<<10) {
			return errors.New("LIVE_PLAN_INVALID")
		}
		return printJSON(p) // approval flags intentionally false; no credentials read
	}
	if command == "live-verify" {
		data, err := task.ReadFile(*root, evidenceName, limits.MaxReportBytes)
		if err != nil {
			return err
		}
		e, err := agenteval.DecodeLiveEvidence(data)
		if err != nil {
			return err
		}
		if err = agenteval.VerifyLiveEvidence(ctx, e); err != nil {
			return err
		}
		return printJSON(map[string]any{"source": e.Episode.Source, "worldReplay": "matched", "grading": "matched", "providerProvenance": "not-proven", "modelSnapshot": "unknown", "cost": "unknown", "trialId": e.Plan.ID})
	}
	data, err := task.ReadFile(*root, planName, task.MaxBytes+(16<<10))
	if err != nil {
		return err
	}
	p, err := agentapi.DecodePlan(data)
	if err != nil {
		return err
	}
	data, err = task.ReadFile(*root, p.Task.Bundle, limits.MaxBundleCompressed)
	if err != nil {
		return err
	}
	b, err := bundle.OpenBytes(data)
	if err != nil {
		return err
	}
	if err = agenteval.PreflightLive(p, b); err != nil {
		return err
	}
	if command == "live-preflight" {
		return printJSON(map[string]any{"status": "locally-valid-not-provider-verified", "approvalCurrentlyValid": p.Authorize(time.Now(), true) == nil, "live": false, "modelAvailability": "not-checked", "cost": "unknown", "trialId": p.ID})
	}
	// Check explicit authority before reading a credential. Neither the plan
	// generator nor preflight accesses the environment or sends a request.
	if err = p.Authorize(time.Now(), allow); err != nil {
		return err
	}
	r, err := agenteval.RecordLive(ctx, *root, p, data, os.Getenv("OPENAI_API_KEY"), allow)
	if err != nil {
		return err
	}
	if err = printJSON(map[string]any{"source": r.Source, "trialId": p.ID, "executionStatus": r.ExecutionStatus, "failureCode": r.FailureCode, "evidenceStatus": r.EvidenceStatus, "cleanupStatus": r.CleanupStatus, "evaluation": r.Evaluation, "usage": r.Usage, "providerProvenance": "not-proven", "cost": "unknown"}); err != nil {
		return err
	}
	if r.ExecutionStatus != "completed" || r.EvidenceStatus != "complete" || r.CleanupStatus != "complete" || r.Evaluation == nil || (r.Evaluation.Outcome != "success" && r.Evaluation.Outcome != "expected_abstention") {
		return errors.New("live Agent evaluation did not pass")
	}
	return nil
}
