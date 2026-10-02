package agenteval

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/augety121/mcp-state-twin/internal/task"
)

type ReviewReason struct {
	Code    string `json:"code"`
	TaskID  string `json:"taskId,omitempty"`
	Repeat  int    `json:"repeat,omitempty"`
	TrialID string `json:"trialId,omitempty"`
}
type TaskMatch struct {
	TaskID      string   `json:"taskId"`
	Repeat      int      `json:"repeat"`
	TrialID     string   `json:"trialId,omitempty"`
	Status      string   `json:"status"`
	Differences []string `json:"differences"`
}
type SuiteReview struct {
	Format             string         `json:"format"`
	Profile            string         `json:"profile"`
	Decision           string         `json:"decision"`
	PlanStatus         string         `json:"planStatus"`
	CatalogStatus      string         `json:"catalogStatus"`
	PlannedPairs       int            `json:"plannedPairs"`
	CheckedPairs       int            `json:"checkedPairs"`
	Tasks              []TaskMatch    `json:"tasks"`
	ExtraTaskIDs       []string       `json:"extraTaskIds"`
	Reasons            []ReviewReason `json:"reasons"`
	ExecutionPerformed bool           `json:"executionPerformed"`
	Provenance         string         `json:"provenance"`
}

func reviewedInputs(ctx context.Context, root, out, expect, catalog string) (*SuiteExpectation, *frozenCatalog, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	if task.PortablePath(out) != nil || task.PortablePath(expect) != nil || beneath(expect, out) {
		return nil, nil, errors.New("ASSESSMENT_EXPECTATION_PATH_INVALID")
	}
	raw, err := task.ReadFile(root, expect, MaxSuitePlanBytes)
	if err != nil {
		return nil, nil, errors.New("ASSESSMENT_EXPECTATION_INVALID")
	}
	e, err := DecodeExpectation(raw)
	if err != nil {
		return nil, nil, err
	}
	c, err := loadCatalog(ctx, root, catalog, out)
	return e, c, err
}
func ReviewSuite(parent context.Context, root, suite, out, expect, catalog string) (*SuiteReview, error) {
	ctx, cancel := context.WithTimeout(parent, 120*time.Second)
	defer cancel()
	e, c, err := reviewedInputs(ctx, root, out, expect, catalog)
	if err != nil {
		return nil, err
	}
	raw, err := task.ReadFile(root, suite, MaxSuitePlanBytes)
	if err != nil {
		return nil, errors.New("SUITE_INPUT_INVALID")
	}
	p, err := DecodeSuite(raw)
	if err != nil {
		return nil, err
	}
	for _, ref := range c.entries {
		for _, pair := range p.Pairs {
			if strings.EqualFold(ref.Task, pair.Task) {
				return nil, errors.New("TASK_CATALOG_PATH_INVALID")
			}
		}
	}
	prepared, err := PrepareSuite(ctx, root, raw)
	if err != nil {
		return nil, err
	}
	return reviewPrepared(ctx, e, c, prepared, p.MaxOutputTokens)
}
func reviewPrepared(ctx context.Context, e *SuiteExpectation, c *frozenCatalog, p *PreparedSuite, tokens int) (*SuiteReview, error) {
	missing, extra := catalogCoverage(c, e.Plan)
	r := &SuiteReview{Format: "statetwin.dev/agent-suite-review/v1alpha1", Profile: "offline-task-review-v1", Decision: "matched", PlanStatus: "matched", CatalogStatus: "matched", PlannedPairs: len(e.Plan.Pairs), Tasks: []TaskMatch{}, ExtraTaskIDs: extra, Reasons: []ReviewReason{}, Provenance: "not-proven"}
	planMatches := same(e.Plan, p.plan)
	if !planMatches {
		r.PlanStatus = "mismatched"
		r.Reasons = append(r.Reasons, ReviewReason{Code: "plan_mismatch"})
	}
	if tokens != e.MaxOutputTokens {
		r.PlanStatus = "mismatched"
		r.Reasons = append(r.Reasons, ReviewReason{Code: "output_budget_mismatch"})
	}
	if len(missing)+len(extra) > 0 {
		r.CatalogStatus = "mismatched"
		r.Reasons = append(r.Reasons, ReviewReason{Code: "catalog_coverage_mismatch"})
	}
	for i, pair := range e.Plan.Pairs {
		row := TaskMatch{TaskID: pair.TaskID, Repeat: pair.Repeat, Status: "unverifiable", Differences: []string{}}
		if planMatches && c.tasks[pair.TaskID] != nil {
			actual, err := task.Decode(p.trials[2*i].taskBytes)
			if err != nil {
				return nil, errors.New("SUITE_INPUT_INVALID")
			}
			row.Differences = taskDifferences(actual, c.tasks[pair.TaskID])
			row.Status = "matched"
			r.CheckedPairs++
			if len(row.Differences) > 0 {
				row.Status = "mismatched"
			}
		}
		r.Tasks = append(r.Tasks, row)
		if row.Status != "matched" {
			code := "task_unverifiable"
			if row.Status == "mismatched" {
				code = "task_mismatch"
			}
			r.Reasons = append(r.Reasons, ReviewReason{Code: code, TaskID: row.TaskID, Repeat: row.Repeat})
		}
	}
	if len(r.Reasons) > 0 {
		r.Decision = "mismatched"
	}
	return r, ctx.Err()
}
