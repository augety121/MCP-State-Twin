package agenteval

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/augety121/mcp-state-twin/internal/limits"
	"github.com/augety121/mcp-state-twin/internal/strictyaml"
	"github.com/augety121/mcp-state-twin/internal/task"
)

const CompareFormat = "statetwin.dev/agent-compare-offline/v1alpha1"

type PlannedTrial struct {
	TrialID  string `json:"trialId" yaml:"trialId"`
	Artifact string `json:"artifact" yaml:"artifact"`
}
type PlannedPair struct {
	TaskID    string       `json:"taskId" yaml:"taskId"`
	Repeat    int          `json:"repeat" yaml:"repeat"`
	Baseline  PlannedTrial `json:"baseline" yaml:"baseline"`
	Candidate PlannedTrial `json:"candidate" yaml:"candidate"`
}
type ComparePlan struct {
	Format             string        `json:"format" yaml:"format"`
	BaselineModel      string        `json:"baselineModel" yaml:"baselineModel"`
	CandidateModel     string        `json:"candidateModel" yaml:"candidateModel"`
	AllowedDifferences []string      `json:"allowedDifferences" yaml:"allowedDifferences"`
	Pairs              []PlannedPair `json:"pairs" yaml:"pairs"`
}
type TrialResult struct {
	TrialID         string `json:"trialId"`
	State           string `json:"state"`
	Validation      string `json:"validation"`
	ExecutionStatus string `json:"executionStatus,omitempty"`
	Outcome         string `json:"outcome,omitempty"`
	PolicyAttempts  int    `json:"policyAttempts"`
}
type PairResult struct {
	TaskID    string      `json:"taskId"`
	Repeat    int         `json:"repeat"`
	Baseline  TrialResult `json:"baseline"`
	Candidate TrialResult `json:"candidate"`
	Decision  string      `json:"decision"`
}
type Denominators struct {
	Planned          int `json:"planned"`
	NotStarted       int `json:"notStarted"`
	Started          int `json:"started"`
	Incomplete       int `json:"incomplete"`
	Terminal         int `json:"terminal"`
	ValidlyEvaluated int `json:"validlyEvaluated"`
}
type Comparison struct {
	Format         string       `json:"format"`
	Source         string       `json:"source"`
	Decision       string       `json:"decision"`
	UpgradeAllowed bool         `json:"upgradeAllowed"`
	Counts         Denominators `json:"counts"`
	Pairs          []PairResult `json:"pairs"`
	Cost           string       `json:"cost"`
}

func DecodeCompare(data []byte) (*ComparePlan, error) {
	var p ComparePlan
	if strictyaml.DecodeOneWithDepth(data, 64<<10, 16, "ComparePlan", &p) != nil {
		return nil, errors.New("COMPARE_PLAN_INVALID")
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return &p, nil
}
func (p *ComparePlan) Validate() error {
	if p == nil || p.Format != CompareFormat || len(p.Pairs) == 0 || len(p.Pairs) > 32 || len(p.AllowedDifferences) != 1 || p.AllowedDifferences[0] != "model" || p.BaselineModel == p.CandidateModel {
		return errors.New("COMPARE_PLAN_INVALID")
	}
	for _, model := range []string{p.BaselineModel, p.CandidateModel} {
		if !strings.HasPrefix(model, "mock-") || !runID.MatchString(model) {
			return errors.New("COMPARE_PLAN_INVALID")
		}
	}
	ids, files, pairs := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, pair := range p.Pairs {
		key := fmt.Sprintf("%s/%d", pair.TaskID, pair.Repeat)
		if !runID.MatchString(pair.TaskID) || pair.Repeat < 1 || pair.Repeat > 16 || pairs[key] {
			return errors.New("COMPARE_PLAN_INVALID")
		}
		pairs[key] = true
		for _, trial := range []PlannedTrial{pair.Baseline, pair.Candidate} {
			if !runID.MatchString(trial.TrialID) || ids[trial.TrialID] || files[strings.ToLower(trial.Artifact)] || task.PortablePath(trial.Artifact) != nil || path.Base(trial.Artifact) != "terminal.json" {
				return errors.New("COMPARE_PLAN_INVALID")
			}
			ids[trial.TrialID] = true
			files[strings.ToLower(trial.Artifact)] = true
		}
	}
	return nil
}

func Compare(ctx context.Context, root string, p *ComparePlan) (*Comparison, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	r := &Comparison{Format: CompareFormat, Source: "mock-responses", Decision: "no_regression_observed", Cost: "mock-no-provider-call", Pairs: []PairResult{}}
	for _, pair := range p.Pairs {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		base, b := inspectTrial(ctx, root, pair.Baseline, pair.TaskID, p.BaselineModel)
		candidate, c := inspectTrial(ctx, root, pair.Candidate, pair.TaskID, p.CandidateModel)
		row := PairResult{TaskID: pair.TaskID, Repeat: pair.Repeat, Baseline: base, Candidate: candidate, Decision: "inconclusive"}
		for _, trial := range []TrialResult{base, candidate} {
			r.Counts.Planned++
			if trial.State == "not_started" {
				r.Counts.NotStarted++
			} else {
				r.Counts.Started++
				if trial.State == "terminal" {
					r.Counts.Terminal++
				} else {
					r.Counts.Incomplete++
				}
			}
			if trial.Validation == "valid" {
				r.Counts.ValidlyEvaluated++
			}
		}
		if b != nil && c != nil && base.Validation == "valid" && candidate.Validation == "valid" {
			bd, cd := b.Episode.Definition, c.Episode.Definition
			bd.Config.Model = ""
			cd.Config.Model = ""
			bd.Config.TrialID = ""
			cd.Config.TrialID = ""
			if !same(bd, cd) {
				row.Decision = "incomparable"
			} else {
				row.Decision = "no_regression_observed"
				if (passing(base.Outcome) && !passing(candidate.Outcome)) || candidate.PolicyAttempts > base.PolicyAttempts {
					row.Decision = "regression"
				}
			}
		}
		if base.Validation == "identity_mismatch" || candidate.Validation == "identity_mismatch" {
			row.Decision = "incomparable"
		}
		r.Pairs = append(r.Pairs, row)
		if decisionPriority(row.Decision) > decisionPriority(r.Decision) {
			r.Decision = row.Decision
		}
	}
	return r, nil
}

func inspectTrial(ctx context.Context, root string, p PlannedTrial, taskID, model string) (TrialResult, *AgentEvidence) {
	r := TrialResult{TrialID: p.TrialID, State: "incomplete", Validation: "invalid"}
	raw, err := task.ReadFile(root, p.Artifact, limits.MaxReportBytes)
	if err != nil {
		fs, openErr := os.OpenRoot(root)
		if openErr == nil {
			defer fs.Close()
			_, e := fs.Stat(path.Dir(p.Artifact))
			if os.IsNotExist(e) {
				r.State = "not_started"
				r.Validation = "missing"
			}
		}
		return r, nil
	}
	e, err := DecodeEvidence(raw)
	if err != nil {
		return r, nil
	}
	ep := e.Episode
	switch ep.ExecutionStatus {
	case "completed", "canceled", "timed_out", "budget_exhausted", "host_error":
		r.State = "terminal"
	default:
		return r, nil
	}
	r.ExecutionStatus = ep.ExecutionStatus
	if ep.Definition.Config.TrialID != p.TrialID || ep.Definition.Config.Model != model || ep.Definition.Task.ID != taskID {
		r.Validation = "identity_mismatch"
		return r, e
	}
	if err = VerifyEvidence(ctx, e); err != nil {
		if ep.EvidenceStatus == "partial" {
			r.Validation = "partial"
		}
		return r, e
	}
	r.Validation = "valid"
	r.Outcome = ep.Evaluation.Outcome
	r.PolicyAttempts = ep.Evaluation.PolicyAttempts
	return r, e
}
func passing(s string) bool { return s == "success" || s == "expected_abstention" }
func decisionPriority(s string) int {
	switch s {
	case "regression":
		return 3
	case "incomparable":
		return 2
	case "inconclusive":
		return 1
	default:
		return 0
	}
}

func (r *Comparison) Markdown() string {
	var out strings.Builder
	fmt.Fprintf(&out, "# Offline Agent comparison\n\nDecision: `%s`. Automatic upgrade: **not authorized**.\n\nSynthetic mock evidence only; no provider capability or pricing inference.\n\nPlanned %d = not started %d + started %d. Started %d = incomplete %d + terminal %d. Validly evaluated: %d.\n\n| Task / repeat | Baseline | Candidate | Decision |\n|---|---|---|---|\n", r.Decision, r.Counts.Planned, r.Counts.NotStarted, r.Counts.Started, r.Counts.Started, r.Counts.Incomplete, r.Counts.Terminal, r.Counts.ValidlyEvaluated)
	for _, p := range r.Pairs {
		fmt.Fprintf(&out, "| %s / %d | %s / %s | %s / %s | %s |\n", p.TaskID, p.Repeat, p.Baseline.Validation, p.Baseline.Outcome, p.Candidate.Validation, p.Candidate.Outcome, p.Decision)
	}
	return out.String()
}
