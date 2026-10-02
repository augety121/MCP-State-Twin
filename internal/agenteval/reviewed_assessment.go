package agenteval

import (
	"context"
	"errors"
	"os"
	"time"
)

type TaskBinding struct {
	Status            string      `json:"status"`
	PlannedTrials     int         `json:"plannedTrials"`
	VerifiedTrials    int         `json:"verifiedTrials"`
	MatchedTrials     int         `json:"matchedTrials"`
	MismatchedTrials  int         `json:"mismatchedTrials"`
	UnavailableTrials int         `json:"unavailableTrials"`
	MissingTaskIDs    []string    `json:"missingTaskIds"`
	ExtraTaskIDs      []string    `json:"extraTaskIds"`
	Trials            []TaskMatch `json:"trials"`
}
type ReviewedAssessment struct {
	Format         string           `json:"format"`
	Profile        string           `json:"profile"`
	Policy         string           `json:"policy"`
	Decision       string           `json:"decision"`
	Reasons        []ReviewReason   `json:"reasons"`
	TaskBinding    TaskBinding      `json:"taskBinding"`
	Assessment     *SuiteAssessment `json:"assessment"`
	UpgradeAllowed bool             `json:"upgradeAllowed"`
	Provenance     string           `json:"provenance"`
}

func AssessReviewedSuite(parent context.Context, root, out, expect, catalog, policy string) (*ReviewedAssessment, error) {
	if !oneOf(policy, "candidate-pass-v1", "both-pass-v1") {
		return nil, errors.New("REVIEWED_ASSESSMENT_ARGUMENTS_INVALID")
	}
	ctx, cancel := context.WithTimeout(parent, 120*time.Second)
	defer cancel()
	e, c, err := reviewedInputs(ctx, root, out, expect, catalog)
	if err != nil {
		return nil, err
	}
	fs, err := os.OpenRoot(root)
	if err != nil {
		return nil, errors.New("SUITE_INSPECT_ROOT_UNAVAILABLE")
	}
	defer fs.Close()
	return assessReviewedView(ctx, diskReadRoot{fs}, out, e, c, policy, replay)
}

func assessReviewedView(ctx context.Context, fs evidenceReadRoot, out string, e *SuiteExpectation, c *frozenCatalog, policy string, verify func(context.Context, *AgentEvidence, bool) error) (*ReviewedAssessment, error) {
	collector := newAssessmentCollector(e, maxDefinitionReferences)
	r := &ReviewedAssessment{Format: "statetwin.dev/agent-reviewed-assessment/v1alpha1", Profile: "offline-reviewed-task-v1", Policy: policy, Decision: "passed", Reasons: []ReviewReason{}, Provenance: "not-proven"}
	b := &r.TaskBinding
	b.Status = "matched"
	b.PlannedTrials = 2 * len(e.Plan.Pairs)
	b.Trials = []TaskMatch{}
	b.MissingTaskIDs, b.ExtraTaskIDs = catalogCoverage(c, e.Plan)
	indices := map[string]int{}
	for _, pair := range e.Plan.Pairs {
		for _, trial := range []PlannedTrial{pair.Baseline, pair.Candidate} {
			indices[trial.TrialID] = len(b.Trials)
			b.Trials = append(b.Trials, TaskMatch{TaskID: pair.TaskID, Repeat: pair.Repeat, TrialID: trial.TrialID, Status: "unverifiable", Differences: []string{}})
		}
	}
	observe := func(taskID, trialID string, d RunDefinition) error {
		if err := collector.observe(taskID, trialID, d); err != nil {
			return err
		}
		if !collector.planMatches {
			return nil
		}
		b.VerifiedTrials++
		if ref := c.tasks[taskID]; ref != nil {
			row := &b.Trials[indices[trialID]]
			row.Differences = taskDifferences(d.Task, ref)
			row.Status = "matched"
			if len(row.Differences) > 0 {
				row.Status = "mismatched"
			}
		}
		return ctx.Err()
	}
	audit, err := inspectSuiteViewVerified(ctx, fs, out, maxSuiteWriteBytes, collector.plan, observe, verify)
	if err != nil {
		return nil, err
	}
	ex, groups := collector.results()
	r.Assessment = &SuiteAssessment{Format: AssessmentFormat, AssessmentProfile: "offline-acceptance-v1", Policy: policy, Decision: "passed", Reasons: []AssessmentReason{}, Expectation: ex, Consistency: groups, Audit: audit, Provenance: "not-proven"}
	r.Assessment.evaluate(e)
	if r.Assessment.Decision != "passed" {
		r.Reasons = append(r.Reasons, ReviewReason{Code: "base_assessment_failed"})
	}
	if len(b.MissingTaskIDs)+len(b.ExtraTaskIDs) > 0 {
		r.Reasons = append(r.Reasons, ReviewReason{Code: "catalog_coverage_mismatch"})
		b.Status = "mismatched"
	}
	for _, row := range b.Trials {
		code := ""
		switch row.Status {
		case "matched":
			b.MatchedTrials++
		case "mismatched":
			b.MismatchedTrials++
			code = "reviewed_task_mismatch"
		default:
			b.UnavailableTrials++
			code = "reviewed_task_unverifiable"
		}
		if code != "" {
			r.Reasons = append(r.Reasons, ReviewReason{Code: code, TaskID: row.TaskID, TrialID: row.TrialID})
		}
	}
	if b.MismatchedTrials > 0 {
		b.Status = "mismatched"
	} else if b.UnavailableTrials > 0 && b.Status != "mismatched" {
		b.Status = "unverifiable"
	}
	if len(r.Reasons) > 0 {
		r.Decision = "failed"
	}
	return r, ctx.Err()
}
