package agenteval

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path"
	"strings"
	"time"

	"github.com/augety121/mcp-state-twin/internal/task"
)

type ProjectCounts struct {
	Planned    int `json:"planned"`
	Started    int `json:"started"`
	Completed  int `json:"completed"`
	Failed     int `json:"failed"`
	NotStarted int `json:"notStarted"`
}
type ProjectCheckFailure struct {
	TaskID   string `json:"taskId"`
	TrialID  string `json:"trialId"`
	CheckID  string `json:"checkId"`
	Category string `json:"category"`
}
type ProjectAssessment struct {
	FailedChecks  []ProjectCheckFailure `json:"failedChecks"`
	Status        string                `json:"status"`
	PlannedTrials int                   `json:"plannedTrials"`
	Result        *ReviewedAssessment   `json:"result"`
}
type ProjectReport struct {
	CaseFailures   []CaseRow         `json:"caseFailures"`
	QualityReasons []CoverageReason  `json:"qualityReasons"`
	Format         string            `json:"format"`
	Profile        string            `json:"profile"`
	ProjectID      string            `json:"projectId"`
	Lifecycle      string            `json:"lifecycle"`
	Decision       string            `json:"decision"`
	Stages         []ProjectStage    `json:"stages"`
	CaseCounts     ProjectCounts     `json:"caseCounts"`
	TrialCounts    ProjectCounts     `json:"trialCounts"`
	TaskBinding    TaskBinding       `json:"taskBinding"`
	WorldBinding   WorldBinding      `json:"worldBinding"`
	BaseAssessment ProjectAssessment `json:"baseAssessment"`
	ReasonCodes    []string          `json:"reasonCodes"`
	Provenance     string            `json:"provenance"`
}
type projectClaim struct {
	Format        string   `json:"format"`
	ProjectID     string   `json:"projectId"`
	PlannedCases  int      `json:"plannedCases"`
	PlannedPairs  int      `json:"plannedPairs"`
	PlannedTrials int      `json:"plannedTrials"`
	Policy        string   `json:"policy"`
	Stages        []string `json:"stages"`
}

var projectStages = []string{"preflight", "review", "quality", "suite", "assessment", "publication"}

func claimProject(p *preparedProject) projectClaim {
	return projectClaim{"statetwin.dev/project-claim/v1alpha1", p.manifest.ID, len(p.cases.inputs), len(p.suite.plan.Pairs), len(p.suite.trials), p.manifest.Policy, append([]string{}, projectStages...)}
}
func newProjectReport(p *preparedProject) *ProjectReport {
	w := worldBindingEmpty(p)
	t := TaskBinding{Status: "unverifiable", PlannedTrials: w.PlannedTrials, UnavailableTrials: w.PlannedTrials, MissingTaskIDs: []string{}, ExtraTaskIDs: []string{}, Trials: append([]TaskMatch{}, w.Trials...)}
	r := &ProjectReport{CaseFailures: []CaseRow{}, QualityReasons: []CoverageReason{}, Format: "statetwin.dev/project-report/v1alpha1", Profile: ProjectProfile, ProjectID: p.manifest.ID, Lifecycle: "partial", Decision: "incomplete", Stages: []ProjectStage{}, CaseCounts: ProjectCounts{Planned: len(p.cases.inputs), NotStarted: len(p.cases.inputs)}, TrialCounts: ProjectCounts{Planned: len(p.suite.trials), NotStarted: len(p.suite.trials)}, TaskBinding: t, WorldBinding: w, BaseAssessment: ProjectAssessment{Status: "not_checked", FailedChecks: []ProjectCheckFailure{}, PlannedTrials: len(p.suite.trials)}, ReasonCodes: []string{}, Provenance: "not-proven"}
	for _, s := range projectStages {
		r.Stages = append(r.Stages, ProjectStage{Stage: s, Status: "not_started"})
	}
	r.Stages[0].Status = "passed"
	r.Stages[1].Status = "passed"
	return r
}
func caseCounts(r *CaseReport) ProjectCounts {
	return ProjectCounts{Planned: r.Planned, Started: r.Planned - r.NotStarted, Completed: r.Matched + r.Mismatched, Failed: r.Failed, NotStarted: r.NotStarted}
}
func trialCounts(r *SuiteReport) ProjectCounts {
	c := ProjectCounts{Planned: r.PlannedTrials}
	for _, t := range r.Trials {
		switch t.State {
		case "completed":
			c.Started++
			c.Completed++
		case "failed":
			c.Started++
			c.Failed++
		default:
			c.NotStarted++
		}
	}
	return c
}

type projectOperations struct {
	open    openEvidenceFS
	witness witnessRunner
	suite   func(context.Context, string, string, *PreparedSuite, openEvidenceFS, int) (*SuiteReport, error)
}

func defaultProjectOperations() projectOperations {
	return projectOperations{openRootedEvidenceFS, RunWitness, runSuite}
}
func RunProject(parent context.Context, root, name, out string) (*ProjectReport, error) {
	ctx, cancel := context.WithTimeout(parent, 120*time.Second)
	defer cancel()
	if ctx.Err() != nil {
		return nil, projectError(ctx.Err())
	}
	if task.PortablePath(out) != nil {
		return nil, errors.New("PROJECT_INPUT_INVALID")
	}
	p, _, err := prepareProject(newProjectSource(ctx, root, out, projectInputLimit), ".", name)
	if err != nil {
		return nil, err
	}
	return runPreparedProject(ctx, root, out, p, defaultProjectOperations())
}
func runPreparedProject(ctx context.Context, root, out string, p *preparedProject, ops projectOperations) (*ProjectReport, error) {
	r := newProjectReport(p)
	stage := 0
	fail := func(code string) (*ProjectReport, error) {
		if ctx.Err() != nil {
			code = "PROJECT_CANCELED_OR_TIMED_OUT"
		}
		r.Stages[stage].Status = "failed"
		r.Stages[stage].ReasonCode = code
		r.ReasonCodes = append(r.ReasonCodes, code)
		return r, errors.New(code)
	}
	if ctx.Err() != nil {
		return fail("PROJECT_CANCELED_OR_TIMED_OUT")
	}
	fs, err := ops.open(root)
	if err != nil {
		return fail("PROJECT_INPUT_UNAVAILABLE")
	}
	defer fs.Close()
	if outputParent(fs, out) != nil {
		return fail("PROJECT_INPUT_INVALID")
	}
	if !absent(fs, out) {
		return fail("PROJECT_DEST_EXISTS")
	}
	if err := fs.Mkdir(out, 0700); err != nil {
		if os.IsExist(err) {
			return fail("PROJECT_DEST_EXISTS")
		}
		return fail("PROJECT_WRITE_FAILED")
	}
	remaining := 4 << 20
	write := func(name string, value any) error {
		return writeProjectJSON(ctx, fs, path.Join(out, name), value, &remaining)
	}
	if err := write("project-claim.json", claimProject(p)); err != nil {
		return fail(err.Error())
	}
	stage = 2
	cases, err := runCases(ctx, p.cases, ops.witness)
	if cases != nil {
		r.CaseCounts = caseCounts(cases)
	}
	if err != nil {
		code := "PROJECT_ASSESSMENT_FAILED"
		if cases != nil {
			for _, c := range cases.Cases {
				if c.CleanupStatus == "failed" {
					code = "PROJECT_CLEANUP_FAILED"
				}
			}
		}
		return fail(code)
	}
	quality := qualifyCases(p.cases, cases)
	projectQualityDiagnostics(r, quality)
	if err := write("quality.json", quality); err != nil {
		return fail(err.Error())
	}
	if quality.Decision != "qualified" {
		r.Stages[stage].Status = "failed"
		r.Stages[stage].ReasonCode = "PROJECT_QUALITY_NOT_QUALIFIED"
		r.Stages[3].ReasonCode = "PROJECT_QUALITY_NOT_QUALIFIED"
		r.Stages[4].ReasonCode = "PROJECT_QUALITY_NOT_QUALIFIED"
		r.ReasonCodes = append(r.ReasonCodes, "PROJECT_QUALITY_NOT_QUALIFIED")
		r.Decision = "failed"
	} else {
		r.Stages[stage].Status = "passed"
		stage = 3
		suite, err := ops.suite(ctx, root, path.Join(out, "suite"), p.suite, ops.open, maxSuiteWriteBytes)
		if suite != nil {
			r.TrialCounts = trialCounts(suite)
		}
		if err != nil {
			code := "PROJECT_ASSESSMENT_FAILED"
			if strings.Contains(err.Error(), "RESOURCE_LIMIT") {
				code = "PROJECT_RESOURCE_LIMIT"
			} else if strings.Contains(err.Error(), "CLEANUP") {
				code = "PROJECT_CLEANUP_FAILED"
			} else if strings.Contains(err.Error(), "WRITE") {
				code = "PROJECT_WRITE_FAILED"
			}
			return fail(code)
		}
		r.Stages[stage].Status = "passed"
		stage = 4
		disk, err := os.OpenRoot(root)
		if err != nil {
			return fail("PROJECT_INPUT_UNAVAILABLE")
		}
		a, w, checks, err := assessProject(ctx, diskReadRoot{disk}, path.Join(out, "suite"), p)
		disk.Close()
		if err != nil {
			code := "PROJECT_ASSESSMENT_FAILED"
			if strings.Contains(err.Error(), "RESOURCE_LIMIT") {
				code = "PROJECT_RESOURCE_LIMIT"
			}
			return fail(code)
		}
		r.WorldBinding = w
		r.TaskBinding = a.TaskBinding
		r.BaseAssessment.Status = "checked"
		r.BaseAssessment.Result = a
		r.BaseAssessment.FailedChecks = checks
		r.Decision = "passed"
		r.Stages[stage].Status = "passed"
		if a.Decision != "passed" || w.Status != "matched" {
			r.Decision = "failed"
			r.Stages[stage].Status = "failed"
			r.Stages[stage].ReasonCode = "PROJECT_ASSESSMENT_FAILED"
			r.ReasonCodes = append(r.ReasonCodes, "PROJECT_ASSESSMENT_FAILED")
		}
	}
	stage = 5
	if ctx.Err() != nil {
		return fail("PROJECT_CANCELED_OR_TIMED_OUT")
	}
	r.Lifecycle = "published"
	r.Stages[stage].Status = "passed"
	if err := write("project-report.pending.json", r); err != nil {
		r.Lifecycle = "partial"
		return fail(err.Error())
	}
	if ctx.Err() != nil {
		r.Lifecycle = "partial"
		return fail("PROJECT_CANCELED_OR_TIMED_OUT")
	}
	if fs.Link(path.Join(out, "project-report.pending.json"), path.Join(out, "project-report.json")) != nil {
		r.Lifecycle = "partial"
		return fail("PROJECT_PUBLISH_FAILED")
	}
	if fs.Remove(path.Join(out, "project-report.pending.json")) != nil {
		r.Lifecycle = "published_with_residue"
		return fail("PROJECT_CLEANUP_FAILED")
	}
	return r, nil
}
func writeProjectJSON(ctx context.Context, fs evidenceFS, name string, v any, remaining *int) error {
	if ctx.Err() != nil {
		return projectError(ctx.Err())
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return errors.New("PROJECT_WRITE_FAILED")
	}
	if len(b) > projectReportLimit || len(b) > *remaining {
		return errors.New("PROJECT_RESOURCE_LIMIT")
	}
	if safe(v, projectReportLimit) != nil {
		return errors.New("PROJECT_INPUT_INVALID")
	}
	*remaining -= len(b)
	if writeExclusiveBytes(fs, name, b) != nil {
		return errors.New("PROJECT_WRITE_FAILED")
	}
	return nil
}

func projectQualityDiagnostics(r *ProjectReport, q *TaskQualification) {
	r.QualityReasons = q.Reasons
	for _, row := range q.CaseReport.Cases {
		if row.State != "matched" {
			r.CaseFailures = append(r.CaseFailures, row)
		}
	}
}
