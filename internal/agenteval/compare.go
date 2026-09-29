package agenteval

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/augety121/mcp-state-twin/internal/limits"
	"github.com/augety121/mcp-state-twin/internal/strictyaml"
	"github.com/augety121/mcp-state-twin/internal/task"
	"gopkg.in/yaml.v3"
)

const CompareFormat = "statetwin.dev/agent-compare-offline/v1alpha1"
const CompareDecisionPolicy = "offline-regression-v2"
const CompareVerificationProfile = "offline-compare-v1"
const maxCompareEvidenceBytes = 128 << 20
const compareTimeout = 120 * time.Second

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
	TrialID             string   `json:"trialId"`
	State               string   `json:"state"`
	Validation          string   `json:"validation"`
	ExecutionStatus     string   `json:"executionStatus,omitempty"`
	Outcome             string   `json:"outcome,omitempty"`
	PolicyAttempts      int      `json:"policyAttempts"`
	BlockedAttempts     int      `json:"blockedAttempts"`
	CommittedViolations int      `json:"committedViolations"`
	FailedPolicyChecks  []string `json:"failedPolicyChecks"`
}
type PairResult struct {
	TaskID            string      `json:"taskId"`
	Repeat            int         `json:"repeat"`
	Baseline          TrialResult `json:"baseline"`
	Candidate         TrialResult `json:"candidate"`
	Decision          string      `json:"decision"`
	Reasons           []string    `json:"reasons"`
	NewPolicyFailures []string    `json:"newPolicyFailures"`
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
	Format              string        `json:"format"`
	Source              string        `json:"source"`
	Decision            string        `json:"decision"`
	UpgradeAllowed      bool          `json:"upgradeAllowed"`
	Counts              Denominators  `json:"counts"`
	Pairs               []PairResult  `json:"pairs"`
	Cost                string        `json:"cost"`
	DecisionPolicy      string        `json:"decisionPolicy"`
	VerificationProfile string        `json:"verificationProfile"`
	BaselineModel       string        `json:"baselineModel"`
	CandidateModel      string        `json:"candidateModel"`
	PlanBinding         string        `json:"planBinding"`
	BaselineCounts      Denominators  `json:"baselineCounts"`
	CandidateCounts     Denominators  `json:"candidateCounts"`
	SummaryPolicy       string        `json:"summaryPolicy"`
	BaselineOutcomes    OutcomeCounts `json:"baselineOutcomes"`
	CandidateOutcomes   OutcomeCounts `json:"candidateOutcomes"`
	TaskSummaries       []TaskSummary `json:"taskSummaries"`
}

func DecodeCompare(data []byte) (*ComparePlan, error) {
	var p ComparePlan
	if strictyaml.DecodeOneWithDepth(data, 64<<10, 16, "ComparePlan", &p) != nil {
		return nil, errors.New("COMPARE_PLAN_INVALID")
	}
	// Inspect original tokens only after bounded strict decoding. yaml.v3 can
	// otherwise truncate 1.5 into repeat 1, silently changing the fixed plan.
	var tokens struct {
		Pairs []struct {
			Repeat yaml.Node `yaml:"repeat"`
		} `yaml:"pairs"`
	}
	if yaml.Unmarshal(data, &tokens) != nil || len(tokens.Pairs) != len(p.Pairs) {
		return nil, errors.New("COMPARE_PLAN_INVALID")
	}
	for _, pair := range tokens.Pairs {
		if pair.Repeat.Kind != yaml.ScalarNode || pair.Repeat.Tag != "!!int" {
			return nil, errors.New("COMPARE_PLAN_INVALID")
		}
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
	return compareWith(ctx, root, p, VerifyEvidence, maxCompareEvidenceBytes)
}

// The private verifier/budget seam permits deterministic interruption tests.
// Public callers always use full replay and the fixed production budget.
func compareWith(ctx context.Context, root string, p *ComparePlan, verify func(context.Context, *AgentEvidence) error, byteLimit int) (*Comparison, error) {
	return compareObserved(ctx, root, p, verify, byteLimit, nil)
}

// Observers receive only identity-bound, replay-verified definitions, including
// unscorable results. They must not mutate evidence and never change old reports.
type definitionObserver func(taskID, trialID string, definition RunDefinition) error

func compareObserved(ctx context.Context, root string, p *ComparePlan, verify func(context.Context, *AgentEvidence) error, byteLimit int, observe definitionObserver) (*Comparison, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, compareTimeout)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	fs, err := os.OpenRoot(root)
	if err != nil {
		return nil, errors.New("COMPARE_ROOT_UNAVAILABLE")
	}
	defer fs.Close()
	r := &Comparison{
		Format: CompareFormat, Source: "mock-responses", Decision: "no_regression_observed",
		Cost: "mock-no-provider-call", Pairs: []PairResult{}, DecisionPolicy: CompareDecisionPolicy,
		BaselineModel: p.BaselineModel, CandidateModel: p.CandidateModel,
		PlanBinding:         "validated-input-not-preregistration-proof",
		VerificationProfile: CompareVerificationProfile,
	}
	for _, pair := range p.Pairs {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		base, b, err := inspectTrial(ctx, fs, pair.Baseline, pair.TaskID, p.BaselineModel, verify, &byteLimit)
		if err != nil {
			return nil, err
		}
		candidate, c, err := inspectTrial(ctx, fs, pair.Candidate, pair.TaskID, p.CandidateModel, verify, &byteLimit)
		if err != nil {
			return nil, err
		}
		if observe != nil {
			for _, sample := range []struct {
				row      TrialResult
				evidence *AgentEvidence
			}{{base, b}, {candidate, c}} {
				if sample.evidence != nil && oneOf(sample.row.Validation, "valid", "not_evaluated") {
					if err := observe(pair.TaskID, sample.row.TrialID, sample.evidence.Episode.Definition); err != nil {
						return nil, err
					}
				}
			}
		}
		row := PairResult{
			TaskID: pair.TaskID, Repeat: pair.Repeat, Baseline: base, Candidate: candidate,
			Decision: "inconclusive", Reasons: []string{"evidence_unavailable_or_unscorable"},
			NewPolicyFailures: []string{},
		}
		r.BaselineCounts.observe(base)
		r.CandidateCounts.observe(candidate)
		for _, trial := range []TrialResult{base, candidate} {
			r.Counts.observe(trial)
		}
		if b != nil && c != nil && base.Validation == "valid" && candidate.Validation == "valid" {
			bd, cd := b.Episode.Definition, c.Episode.Definition
			bd.Config.Model = ""
			cd.Config.Model = ""
			bd.Config.TrialID = ""
			cd.Config.TrialID = ""
			if !same(bd, cd) {
				row.Decision = "incomparable"
				row.Reasons = []string{"definition_mismatch"}
			} else {
				row.Decision = "no_regression_observed"
				row.Reasons, row.NewPolicyFailures = regressionReasons(base, candidate)
				if len(row.Reasons) > 0 {
					row.Decision = "regression"
				}
			}
		}
		if base.Validation == "identity_mismatch" || candidate.Validation == "identity_mismatch" {
			row.Decision = "incomparable"
			row.Reasons = []string{"identity_mismatch"}
		}
		r.Pairs = append(r.Pairs, row)
		if decisionPriority(row.Decision) > decisionPriority(r.Decision) {
			r.Decision = row.Decision
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := r.summarizeTasks(); err != nil {
		return nil, err
	}
	return r, nil
}

func (d *Denominators) observe(trial TrialResult) {
	d.Planned++
	if trial.State == "not_started" {
		d.NotStarted++
	} else {
		d.Started++
		if trial.State == "terminal" {
			d.Terminal++
		} else {
			d.Incomplete++
		}
	}
	if trial.Validation == "valid" {
		d.ValidlyEvaluated++
	}
}

func regressionReasons(base, candidate TrialResult) ([]string, []string) {
	reasons, newFailures := []string{}, []string{}
	if passing(base.Outcome) && !passing(candidate.Outcome) {
		reasons = append(reasons, "task_success_lost")
	}
	if candidate.PolicyAttempts > base.PolicyAttempts {
		reasons = append(reasons, "policy_attempts_increased")
	}
	if candidate.CommittedViolations > base.CommittedViolations {
		reasons = append(reasons, "committed_violations_increased")
	}
	failed := map[string]bool{}
	for _, id := range base.FailedPolicyChecks {
		failed[id] = true
	}
	for _, id := range candidate.FailedPolicyChecks {
		if !failed[id] {
			newFailures = append(newFailures, id)
		}
	}
	if len(newFailures) > 0 {
		reasons = append(reasons, "new_policy_failure")
	}
	if candidate.Outcome == "policy_violation" && base.Outcome != "policy_violation" {
		reasons = append(reasons, "policy_outcome_worsened")
	}
	return reasons, newFailures
}

func inspectTrial(ctx context.Context, fs *os.Root, p PlannedTrial, taskID, model string, verify func(context.Context, *AgentEvidence) error, remaining *int) (TrialResult, *AgentEvidence, error) {
	r := TrialResult{TrialID: p.TrialID, State: "incomplete", Validation: "invalid", FailedPolicyChecks: []string{}}
	if err := ctx.Err(); err != nil {
		return r, nil, err
	}
	parent := path.Dir(p.Artifact)
	if parent != "." {
		parts := strings.Split(parent, "/")
		for i := range parts {
			info, err := fs.Lstat(strings.Join(parts[:i+1], "/"))
			if os.IsNotExist(err) {
				r.State, r.Validation = "not_started", "missing"
				return r, nil, nil
			}
			if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
				return r, nil, nil
			}
		}
	}
	raw, err := readArtifact(fs, p.Artifact, limits.MaxReportBytes)
	if ctx.Err() != nil {
		return r, nil, ctx.Err()
	}
	if err != nil {
		return r, nil, nil
	}
	if len(raw) > *remaining {
		return r, nil, errors.New("COMPARE_RESOURCE_LIMIT")
	}
	*remaining -= len(raw)
	e, err := DecodeEvidence(raw)
	if ctx.Err() != nil {
		return r, nil, ctx.Err()
	}
	if err != nil {
		return r, nil, nil
	}
	ep := e.Episode
	switch ep.ExecutionStatus {
	case "completed", "canceled", "timed_out", "budget_exhausted", "host_error":
		r.State = "terminal"
	default:
		return r, nil, nil
	}
	r.ExecutionStatus = ep.ExecutionStatus
	if ep.Definition.Config.TrialID != p.TrialID || ep.Definition.Config.Model != model || ep.Definition.Task.ID != taskID {
		r.Validation = "identity_mismatch"
		return r, e, nil
	}
	err = verify(ctx, e)
	if ctx.Err() != nil {
		return r, nil, ctx.Err()
	}
	if err != nil {
		if ep.EvidenceStatus == "partial" {
			r.Validation = "partial"
		}
		return r, e, nil
	}
	r.Validation = "valid"
	r.Outcome = ep.Evaluation.Outcome
	r.PolicyAttempts = ep.Evaluation.PolicyAttempts
	r.BlockedAttempts = ep.Evaluation.BlockedAttempts
	r.CommittedViolations = ep.Evaluation.CommittedViolations
	if !passing(r.Outcome) && r.Outcome != "task_failed" && r.Outcome != "policy_violation" {
		r.Validation = "not_evaluated"
	}
	for _, check := range ep.Evaluation.Checks {
		if check.Error != "" {
			r.Validation = "not_evaluated"
		}
		if check.Category == "policy" && !check.Passed {
			r.FailedPolicyChecks = append(r.FailedPolicyChecks, check.ID)
		}
	}
	return r, e, nil
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
	fmt.Fprintf(&out, "Verification profile: `%s`.\n\n", r.VerificationProfile)
	fmt.Fprintf(&out, "# Offline Agent comparison\n\nDecision: `%s`. Automatic upgrade: **not authorized**.\n\nSynthetic mock evidence only; no provider capability or pricing inference.\n\nPolicy: `%s`. Baseline: `%s`. Candidate: `%s`.\n\nPlan binding: `%s`. Validly evaluated is not a success count.\n\nPlanned %d = not started %d + started %d. Started %d = incomplete %d + terminal %d. Validly evaluated: %d.\n\n| Cohort | Planned | Not started | Started | Incomplete | Terminal | Validly evaluated |\n|---|---|---|---|---|---|---|\n", r.Decision, r.DecisionPolicy, r.BaselineModel, r.CandidateModel, r.PlanBinding, r.Counts.Planned, r.Counts.NotStarted, r.Counts.Started, r.Counts.Started, r.Counts.Incomplete, r.Counts.Terminal, r.Counts.ValidlyEvaluated)
	for _, side := range []struct {
		name   string
		counts Denominators
	}{{"Baseline", r.BaselineCounts}, {"Candidate", r.CandidateCounts}} {
		c := side.counts
		fmt.Fprintf(&out, "| %s | %d | %d | %d | %d | %d | %d |\n", side.name, c.Planned, c.NotStarted, c.Started, c.Incomplete, c.Terminal, c.ValidlyEvaluated)
	}
	r.writeTaskSummaries(&out)
	out.WriteString("\n| Task / repeat | Baseline trial / validation / outcome | Candidate trial / validation / outcome | Decision | Reasons | New policy failures |\n|---|---|---|---|---|---|\n")
	for _, p := range r.Pairs {
		fmt.Fprintf(&out, "| %s / %d | %s / %s / %s | %s / %s / %s | %s | %s | %s |\n", p.TaskID, p.Repeat, p.Baseline.TrialID, p.Baseline.Validation, p.Baseline.Outcome, p.Candidate.TrialID, p.Candidate.Validation, p.Candidate.Outcome, p.Decision, strings.Join(p.Reasons, ", "), strings.Join(p.NewPolicyFailures, ", "))
	}
	return out.String()
}
