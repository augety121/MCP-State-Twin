package baseline

import (
	"context"
	"errors"
	"fmt"
	"path"

	"github.com/augety121/mcp-state-twin/internal/agentapi"
	"github.com/augety121/mcp-state-twin/internal/agenteval"
	"github.com/augety121/mcp-state-twin/internal/baselinepack"
	"github.com/augety121/mcp-state-twin/internal/limits"
	"github.com/augety121/mcp-state-twin/internal/task"
)

func inspectLiveTrial(ctx context.Context, root string, p *Prepared, t Trial) (Observation, error) {
	o := Observation{TrialID: t.ID, State: "missing", ObservedModel: "unknown", Usage: Usage{Status: "unavailable"}}
	expected, e := agentapi.DecodePlan(p.Pack.Reader.Inputs[p.Frozen.Plan.LivePlans[t.ID]].Raw)
	if e != nil {
		return o, e
	}
	raw, e := task.ReadFile(root, path.Join("runs", fmt.Sprintf("shard-%02d", t.Shard), t.ID+".plan.json"), task.MaxBytes+(16<<10))
	if e != nil {
		return o, e
	}
	actual, e := agentapi.DecodePlan(raw)
	if e != nil || !baselinepack.Equal(expected, actual) {
		return o, errors.New("BASELINE_LIVE_BINDING_INVALID")
	}
	raw, e = task.ReadFile(root, path.Join(expected.OutputDirectory(), "terminal.json"), limits.MaxReportBytes)
	if e != nil {
		return o, e
	}
	evidence, e := agenteval.DecodeLiveEvidence(raw)
	if e != nil {
		return o, e
	}
	o.State = evidence.Episode.ExecutionStatus
	o.Cleanup = evidence.Episode.CleanupStatus
	if !baselinepack.Equal(evidence.Plan, expected) || evidence.Episode.Source != "provider-live" || agenteval.VerifyLiveEvidence(ctx, evidence) != nil {
		return o, errors.New("BASELINE_EVIDENCE_INVALID")
	}
	grade := evidence.Episode.Evaluation
	if grade == nil {
		return o, errors.New("BASELINE_EVIDENCE_INVALID")
	}
	o.Verified = true
	o.Success = grade.Outcome == expected.Task.ExpectedOutcome
	o.IllegalAttempts = grade.PolicyAttempts
	o.IllegalEffects = grade.CommittedViolations
	for _, c := range grade.Checks {
		if c.Category == "policy" && !c.Passed {
			o.PolicyFailures++
		}
	}
	input, output := 0, 0
	complete := true
	for _, receipt := range evidence.Receipts {
		if receipt.Tokens == nil {
			complete = false
			break
		}
		input += int(receipt.Tokens.Input)
		output += int(receipt.Tokens.Output)
	}
	if complete && len(evidence.Receipts) > 0 {
		o.Usage.Status = "reported"
		o.Usage.InputTokens = &input
		o.Usage.OutputTokens = &output
	}
	// Receipts do not authenticate provider origin or establish a model snapshot.
	// No price source is bound, so cost remains unknown even with reported usage.
	return o, nil
}
