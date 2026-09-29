package agenteval

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/augety121/mcp-state-twin/internal/agenthost"
	"github.com/augety121/mcp-state-twin/internal/logging"
)

const suiteClaimFormat = "statetwin.dev/agent-suite-claim/v1alpha1"
const maxSuiteReportBytes = 512 << 10

// A semantic round trip also rejects omitted required zero/false fields, null
// substitutions and case-insensitive field aliases accepted by encoding/json.
func decodeSuiteMetadata(raw []byte, limit int, target any) error {
	if agenthost.DecodeDocument(raw, limit, target) != nil || logging.ContainsSensitive(string(raw)) || safe(target, limit) != nil {
		return errors.New("SUITE_METADATA_INVALID")
	}
	var original any
	if json.Unmarshal(raw, &original) != nil || !same(original, target) {
		return errors.New("SUITE_METADATA_INVALID")
	}
	return nil
}

func validateSuiteComparisonPlan(p *ComparePlan) error {
	if p.Validate() != nil || len(p.Pairs) > maxSuitePairs {
		return errors.New("SUITE_METADATA_INVALID")
	}
	for i, pair := range p.Pairs {
		for j, trial := range []PlannedTrial{pair.Baseline, pair.Candidate} {
			id := fmt.Sprintf("%s-%02d", []string{"baseline", "candidate"}[j], i+1)
			if trial.TrialID != id || trial.Artifact != id+"/terminal.json" {
				return errors.New("SUITE_METADATA_INVALID")
			}
		}
	}
	return nil
}

func decodeSuiteClaim(raw []byte) (*SuitePreflight, error) {
	var c SuitePreflight
	if decodeSuiteMetadata(raw, MaxSuitePlanBytes, &c) != nil || validateSuiteComparisonPlan(&c.Plan) != nil {
		return nil, errors.New("SUITE_METADATA_INVALID")
	}
	p := PreparedSuite{plan: c.Plan, trials: make([]preparedSuiteTrial, 2*len(c.Plan.Pairs))}
	expected := p.Summary()
	expected.Format = suiteClaimFormat
	if !same(c, expected) {
		return nil, errors.New("SUITE_METADATA_INVALID")
	}
	return &c, nil
}

func decodeSuiteReport(raw []byte, p *ComparePlan) (*SuiteReport, error) {
	var r SuiteReport
	invalid := errors.New("SUITE_METADATA_INVALID")
	if decodeSuiteMetadata(raw, maxSuiteReportBytes, &r) != nil || r.Format != SuiteReportFormat || r.Profile != SuiteProfile || r.UpgradeAllowed || r.PlannedTrials != 2*len(p.Pairs) || len(r.Trials) != r.PlannedTrials {
		return nil, invalid
	}
	if !oneOf(r.ExecutionStatus, "completed", "stopped", "canceled", "timed_out") || !oneOf(r.ComparisonStatus, "complete", "failed", "not_attempted") || (r.ComparisonStatus == "complete") != (r.Comparison != nil) {
		return nil, invalid
	}
	if !oneOf(r.FailureCode, "", "SUITE_TRIAL_FAILED", "SUITE_RESOURCE_LIMIT", "SUITE_CANCELED", "SUITE_DEADLINE_EXCEEDED", "SUITE_COMPARISON_FAILED") {
		return nil, invalid
	}
	if r.ExecutionStatus == "completed" && ((r.ComparisonStatus == "complete" && r.FailureCode != "") || (r.ComparisonStatus == "failed" && r.FailureCode != "SUITE_COMPARISON_FAILED") || r.ComparisonStatus == "not_attempted") {
		return nil, invalid
	}
	if (r.ExecutionStatus == "canceled" && r.FailureCode != "SUITE_CANCELED") || (r.ExecutionStatus == "timed_out" && r.FailureCode != "SUITE_DEADLINE_EXCEEDED") || (r.ExecutionStatus == "stopped" && !oneOf(r.FailureCode, "SUITE_TRIAL_FAILED", "SUITE_RESOURCE_LIMIT")) || (r.ExecutionStatus == "completed" && !oneOf(r.FailureCode, "", "SUITE_COMPARISON_FAILED")) {
		return nil, invalid
	}
	stopped := false
	for i, row := range r.Trials {
		pair := p.Pairs[i/2]
		id := pair.Baseline.TrialID
		if i%2 == 1 {
			id = pair.Candidate.TrialID
		}
		if row.TrialID != id || !oneOf(row.State, "completed", "failed", "not_started") || !oneOf(row.Outcome, "", "success", "expected_abstention", "task_failed", "policy_violation", "evaluator_error", "not_evaluated") || !oneOf(row.FailureCode, "", "SUITE_TRIAL_FAILED", "SUITE_INPUT_INVALID") {
			return nil, invalid
		}
		if row.State == "not_started" {
			if row.ExecutionStatus != "" || row.EvidenceStatus != "" || row.CleanupStatus != "" || row.Outcome != "" || row.FailureCode != "" {
				return nil, invalid
			}
		} else {
			if stopped {
				return nil, invalid
			}
			if row.State == "completed" {
				if row.ExecutionStatus != "completed" || row.EvidenceStatus != "complete" || row.CleanupStatus != "complete" || row.FailureCode != "" {
					return nil, invalid
				}
			} else if row.FailureCode == "" {
				return nil, invalid
			}
			if row.ExecutionStatus != "" || row.EvidenceStatus != "" || row.CleanupStatus != "" {
				if !knownStatuses(&AgentEpisode{ExecutionStatus: row.ExecutionStatus, EvidenceStatus: row.EvidenceStatus, CleanupStatus: row.CleanupStatus}) {
					return nil, invalid
				}
			}
		}
		if row.State != "completed" {
			stopped = true
		}
		if r.ExecutionStatus == "completed" && stopped {
			return nil, invalid
		}
	}
	return &r, nil
}

func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}
