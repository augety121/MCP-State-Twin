package agenteval

import (
	"context"
	"errors"
	"time"

	"github.com/augety121/mcp-state-twin/internal/bundle"
	"github.com/augety121/mcp-state-twin/internal/evaluator"
	"github.com/augety121/mcp-state-twin/internal/task"
)

type CaseRow struct {
	GradingSource   string   `json:"gradingSource,omitempty"`
	CaseID          string   `json:"caseId"`
	TaskID          string   `json:"taskId"`
	Role            string   `json:"role"`
	State           string   `json:"state"`
	ExpectedOutcome string   `json:"expectedOutcome"`
	ActualOutcome   string   `json:"actualOutcome"`
	FailedCheckIDs  []string `json:"failedCheckIds"`
	ErrorCheckIDs   []string `json:"errorCheckIds"`
	ExecutionStatus string   `json:"executionStatus"`
	CleanupStatus   string   `json:"cleanupStatus"`
	FailureCode     string   `json:"failureCode"`
}
type CaseReport struct {
	Format     string    `json:"format"`
	Source     string    `json:"source"`
	Profile    string    `json:"profile"`
	Decision   string    `json:"decision"`
	Planned    int       `json:"planned"`
	Matched    int       `json:"matched"`
	Mismatched int       `json:"mismatched"`
	Failed     int       `json:"failed"`
	NotStarted int       `json:"notStarted"`
	Cases      []CaseRow `json:"cases"`
	Provenance string    `json:"provenance"`
}
type witnessRunner func(context.Context, *task.Task, *bundle.Artifact, *Witness) (*Report, error)

func RunCases(parent context.Context, root, name string) (*CaseReport, error) {
	ctx, cancel := context.WithTimeout(parent, 120*time.Second)
	defer cancel()
	p, err := prepareCases(ctx, root, name)
	if err != nil {
		return nil, err
	}
	return runCases(ctx, p, RunWitness)
}
func runCases(ctx context.Context, p *preparedCases, run witnessRunner) (*CaseReport, error) {
	r := &CaseReport{Format: "statetwin.dev/task-case-report/v1alpha1", Source: "scripted-witness-cases", Profile: "synthetic-witness-cases-v1", Decision: "matched", Planned: len(p.inputs), Cases: []CaseRow{}, Provenance: "not-proven"}
	for _, c := range p.manifest.Cases {
		r.Cases = append(r.Cases, CaseRow{CaseID: c.CaseID, TaskID: c.TaskID, Role: c.Role, ExpectedOutcome: c.Expected.Outcome, State: "not_started", FailedCheckIDs: []string{}, ErrorCheckIDs: []string{}})
	}
	var first error
	for i, in := range p.inputs {
		if ctx.Err() != nil {
			first = ctx.Err()
			break
		}
		row := &r.Cases[i]
		result, err := run(ctx, in.task, in.bundle, in.witness)
		if err == nil && result != nil && result.ExecutionStatus == "completed" && result.CleanupStatus == "complete" && len(p.manifest.Cases[i].Mutations) > 0 {
			result, err = mutatedGrading(ctx, in.task, result, p.manifest.Cases[i].Mutations)
			row.GradingSource = "synthetic-view-mutation"
		}
		if result != nil {
			row.ExecutionStatus = result.ExecutionStatus
			row.CleanupStatus = result.CleanupStatus
			if result.Evaluation != nil {
				row.ActualOutcome = result.Evaluation.Outcome
				for _, check := range result.Evaluation.Checks {
					if !check.Passed {
						row.FailedCheckIDs = append(row.FailedCheckIDs, check.ID)
					}
					if check.Error != "" {
						row.ErrorCheckIDs = append(row.ErrorCheckIDs, check.ID)
					}
				}
			}
		}
		if err != nil || ctx.Err() != nil || result == nil || result.Evaluation == nil || row.ExecutionStatus != "completed" || row.CleanupStatus != "complete" {
			row.State = "failed"
			row.FailureCode = "CASE_EXECUTION_FAILED"
			if row.CleanupStatus == "failed" {
				row.FailureCode = "CASE_CLEANUP_FAILED"
			}
			first = errors.New(row.FailureCode)
			if ctx.Err() != nil {
				first = ctx.Err()
				row.FailureCode = "CASE_CANCELED_OR_TIMED_OUT"
			}
			break
		}
		row.State = "mismatched"
		c := p.manifest.Cases[i]
		if row.ActualOutcome == c.Expected.Outcome && equalIDs(row.FailedCheckIDs, c.Expected.FailedChecks) && (c.Role != "unscorable-negative" || len(row.ErrorCheckIDs) > 0) {
			row.State = "matched"
		}
	}
	for _, row := range r.Cases {
		switch row.State {
		case "matched":
			r.Matched++
		case "mismatched":
			r.Mismatched++
		case "failed":
			r.Failed++
		default:
			r.NotStarted++
		}
	}
	if r.Mismatched > 0 {
		r.Decision = "mismatched"
	}
	if r.Failed+r.NotStarted > 0 {
		r.Decision = "incomplete"
	}
	if safe(r, 1<<20) != nil {
		return nil, errors.New("CASE_RESOURCE_LIMIT")
	}
	return r, first
}

func mutatedGrading(ctx context.Context, t *task.Task, r *Report, mutations []ViewMutation) (*Report, error) {
	copy := *r
	if r.View.After == nil {
		return nil, errors.New("CASE_EXECUTION_FAILED")
	}
	after, err := r.View.After.Clone()
	if err != nil {
		return nil, err
	}
	copy.View.After = after
	for _, m := range mutations {
		if _, ok := after.Entities[m.Entity][m.Key][m.Field].(string); !ok {
			return nil, errors.New("CASE_EXECUTION_FAILED")
		}
		after.Entities[m.Entity][m.Key][m.Field] = m.Value
	}
	grader, err := evaluator.Compile(t)
	if err != nil {
		return nil, err
	}
	copy.Evaluation, err = grader.Evaluate(ctx, copy.View)
	return &copy, err
}
func equalIDs(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	m := map[string]bool{}
	for _, s := range a {
		m[s] = true
	}
	for _, s := range b {
		if !m[s] {
			return false
		}
	}
	return len(m) == len(a)
}
