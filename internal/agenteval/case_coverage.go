package agenteval

import (
	"context"
	"errors"
	"time"
)

type CheckCoverage struct {
	ID                     string   `json:"id"`
	Category               string   `json:"category"`
	Status                 string   `json:"status"`
	MatchedNegativeCaseIDs []string `json:"matchedNegativeCaseIds"`
}
type TaskCoverage struct {
	TaskID          string          `json:"taskId"`
	PositiveCaseIDs []string        `json:"positiveCaseIds"`
	Checks          []CheckCoverage `json:"checks"`
}
type CoverageReason struct {
	Code    string `json:"code"`
	TaskID  string `json:"taskId,omitempty"`
	CheckID string `json:"checkId,omitempty"`
}
type TaskQualification struct {
	Format     string           `json:"format"`
	Profile    string           `json:"profile"`
	Decision   string           `json:"decision"`
	CaseReport *CaseReport      `json:"caseReport"`
	Tasks      []TaskCoverage   `json:"tasks"`
	Reasons    []CoverageReason `json:"reasons"`
	Provenance string           `json:"provenance"`
}

func QualifyTasks(parent context.Context, root, name string) (*TaskQualification, error) {
	ctx, cancel := context.WithTimeout(parent, 120*time.Second)
	defer cancel()
	p, err := prepareCases(ctx, root, name)
	if err != nil {
		return nil, err
	}
	report, err := runCases(ctx, p, RunWitness)
	if report == nil {
		return nil, err
	}
	r := qualifyCases(p, report)
	if safe(r, 1<<20) != nil {
		return nil, errors.New("CASE_RESOURCE_LIMIT")
	}
	return r, err
}
func qualifyCases(p *preparedCases, report *CaseReport) *TaskQualification {
	r := &TaskQualification{Format: "statetwin.dev/task-qualification/v1alpha1", Profile: "oracle-cases-v1", Decision: "qualified", CaseReport: report, Tasks: []TaskCoverage{}, Reasons: []CoverageReason{}, Provenance: "not-proven"}
	if report.Decision != "matched" {
		r.Reasons = append(r.Reasons, CoverageReason{Code: "case_run_not_matched"})
	}
	for _, ref := range p.manifest.Tasks {
		row := TaskCoverage{TaskID: ref.TaskID, PositiveCaseIDs: []string{}, Checks: []CheckCoverage{}}
		for _, a := range p.tasks[ref.TaskID].Oracle {
			row.Checks = append(row.Checks, CheckCoverage{ID: a.ID, Category: a.Category, Status: "uncovered", MatchedNegativeCaseIDs: []string{}})
		}
		for _, c := range report.Cases {
			if c.TaskID != ref.TaskID || c.State != "matched" {
				continue
			}
			if c.Role == "positive" {
				row.PositiveCaseIDs = append(row.PositiveCaseIDs, c.CaseID)
			}
			for i := range row.Checks {
				check := &row.Checks[i]
				if c.Role != check.Category+"-negative" || containsID(c.ErrorCheckIDs, check.ID) || !containsID(c.FailedCheckIDs, check.ID) {
					continue
				}
				check.Status = "covered"
				check.MatchedNegativeCaseIDs = append(check.MatchedNegativeCaseIDs, c.CaseID)
			}
		}
		if len(row.PositiveCaseIDs) == 0 {
			r.Reasons = append(r.Reasons, CoverageReason{Code: "positive_missing", TaskID: ref.TaskID})
		}
		for _, c := range row.Checks {
			if c.Status != "covered" {
				r.Reasons = append(r.Reasons, CoverageReason{Code: "negative_coverage_missing", TaskID: ref.TaskID, CheckID: c.ID})
			}
		}
		r.Tasks = append(r.Tasks, row)
	}
	if len(r.Reasons) > 0 {
		r.Decision = "not_qualified"
	}
	return r
}
func containsID(ids []string, id string) bool {
	for _, s := range ids {
		if s == id {
			return true
		}
	}
	return false
}
