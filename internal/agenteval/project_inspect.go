package agenteval

import (
	"context"
	"errors"
	"os"
	"path"
	"strings"
	"time"

	"github.com/augety121/mcp-state-twin/internal/task"
)

type ProjectInspection struct {
	Format                    string            `json:"format"`
	ProjectID                 string            `json:"projectId"`
	Verification              string            `json:"verification"`
	QualificationVerification string            `json:"qualificationVerification"`
	HistoricalDecision        string            `json:"historicalDecision"`
	Lifecycle                 string            `json:"lifecycle"`
	ReasonCodes               []string          `json:"reasonCodes"`
	CurrentAssessment         ProjectAssessment `json:"currentAssessment"`
	WorldBinding              WorldBinding      `json:"worldBinding"`
	Provenance                string            `json:"provenance"`
}

func InspectProject(parent context.Context, root, name, out string) (*ProjectInspection, error) {
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
	fs, err := os.OpenRoot(root)
	if err != nil {
		return nil, errors.New("PROJECT_INPUT_UNAVAILABLE")
	}
	defer fs.Close()
	return inspectPreparedProject(ctx, diskReadRoot{fs}, out, p)
}
func safeProjectDir(fs evidenceReadRoot, out string) error {
	if task.PortablePath(out) != nil {
		return errors.New("PROJECT_INPUT_INVALID")
	}
	parts := strings.Split(out, "/")
	for i := range parts {
		v, err := fs.Lstat(strings.Join(parts[:i+1], "/"))
		if err != nil {
			return err
		}
		if !v.IsDir() || v.Mode()&os.ModeSymlink != 0 {
			return errors.New("PROJECT_INPUT_INVALID")
		}
	}
	return nil
}
func inspectPreparedProject(ctx context.Context, fs evidenceReadRoot, out string, p *preparedProject) (*ProjectInspection, error) {
	r := &ProjectInspection{Format: "statetwin.dev/project-inspection/v1alpha1", ProjectID: p.manifest.ID, Verification: "incomplete", QualificationVerification: "recorded_only", HistoricalDecision: "unknown", Lifecycle: "partial", ReasonCodes: []string{}, CurrentAssessment: ProjectAssessment{Status: "not_checked", FailedChecks: []ProjectCheckFailure{}, PlannedTrials: len(p.suite.trials)}, WorldBinding: worldBindingEmpty(p), Provenance: "not-proven"}
	bad := func(code string) (*ProjectInspection, error) {
		r.Verification = "mismatched"
		r.ReasonCodes = append(r.ReasonCodes, code)
		return r, nil
	}
	if ctx.Err() != nil {
		return r, projectError(ctx.Err())
	}
	if err := safeProjectDir(fs, out); err != nil {
		if os.IsNotExist(err) {
			return r, nil
		}
		return bad("PROJECT_INPUT_INVALID")
	}
	entries, err := suiteEntries(fs, out, 6)
	if err != nil {
		return bad("PROJECT_INPUT_INVALID")
	}
	present := map[string]bool{}
	for _, e := range entries {
		if !oneOf(e.Name(), "project-claim.json", "quality.json", "suite", "project-report.pending.json", "project-report.json") {
			return bad("PROJECT_INPUT_INVALID")
		}
		present[e.Name()] = true
		info, statErr := fs.Lstat(path.Join(out, e.Name()))
		if statErr != nil || info.Mode()&os.ModeSymlink != 0 || (e.Name() == "suite" && !info.IsDir()) || (e.Name() != "suite" && !info.Mode().IsRegular()) {
			return bad("PROJECT_INPUT_INVALID")
		}
	}
	if !present["project-claim.json"] {
		return r, nil
	}
	remaining := 4 << 20
	read := func(n string, v any) error {
		b, err := readArtifact(fs, path.Join(out, n), min(projectReportLimit, remaining))
		if err != nil {
			return err
		}
		remaining -= len(b)
		return decodeSuiteMetadata(b, projectReportLimit, v)
	}
	var claim projectClaim
	if read("project-claim.json", &claim) != nil || !same(claim, claimProject(p)) {
		return bad("PROJECT_REFERENCE_MISMATCH")
	}
	var q TaskQualification
	if present["quality.json"] {
		if read("quality.json", &q) != nil || !recordedQualityValid(p, &q) {
			return bad("PROJECT_INPUT_INVALID")
		}
	}
	if !present["project-report.json"] {
		if present["project-report.pending.json"] {
			var pending ProjectReport
			if read("project-report.pending.json", &pending) != nil {
				return bad("PROJECT_INPUT_INVALID")
			}
		}
		return r, nil
	}
	var saved ProjectReport
	if read("project-report.json", &saved) != nil || !present["quality.json"] {
		return bad("PROJECT_INPUT_INVALID")
	}
	r.HistoricalDecision = saved.Decision
	want := newProjectReport(p)
	want.CaseCounts = caseCounts(q.CaseReport)
	projectQualityDiagnostics(want, &q)
	if q.Decision != "qualified" {
		if present["suite"] {
			return bad("PROJECT_INPUT_INVALID")
		}
		want.Decision = "failed"
		want.Stages[2].Status = "failed"
		want.Stages[2].ReasonCode = "PROJECT_QUALITY_NOT_QUALIFIED"
		want.ReasonCodes = append(want.ReasonCodes, "PROJECT_QUALITY_NOT_QUALIFIED")
	} else {
		if !present["suite"] {
			return r, nil
		}
		a, w, checks, err := assessProject(ctx, fs, path.Join(out, "suite"), p)
		if err != nil {
			return r, projectError(err)
		}
		r.CurrentAssessment = ProjectAssessment{Status: "checked", FailedChecks: checks, PlannedTrials: len(p.suite.trials), Result: a}
		r.WorldBinding = w
		var sr SuiteReport
		b, err := readArtifact(fs, path.Join(out, "suite/report.json"), maxSuiteReportBytes)
		if err != nil {
			return bad("PROJECT_INPUT_INVALID")
		}
		if decodeSuiteMetadata(b, maxSuiteReportBytes, &sr) != nil {
			return bad("PROJECT_INPUT_INVALID")
		}
		want.TrialCounts = trialCounts(&sr)
		want.TaskBinding = a.TaskBinding
		want.WorldBinding = w
		want.BaseAssessment = r.CurrentAssessment
		want.Stages[2].Status = "passed"
		want.Stages[3].Status = "passed"
		want.Stages[4].Status = "passed"
		want.Decision = "passed"
		if a.Decision != "passed" || w.Status != "matched" {
			want.Decision = "failed"
			want.Stages[4].Status = "failed"
			want.Stages[4].ReasonCode = "PROJECT_ASSESSMENT_FAILED"
			want.ReasonCodes = append(want.ReasonCodes, "PROJECT_ASSESSMENT_FAILED")
		}
	}
	want.Lifecycle = "published"
	want.Stages[5].Status = "passed"
	if !same(saved, want) {
		return bad("PROJECT_REFERENCE_MISMATCH")
	}
	if present["project-report.pending.json"] {
		var pending ProjectReport
		if read("project-report.pending.json", &pending) != nil || !same(saved, pending) {
			return bad("PROJECT_INPUT_INVALID")
		}
		r.Lifecycle = "published_with_residue"
		r.ReasonCodes = append(r.ReasonCodes, "PROJECT_CLEANUP_FAILED")
		return r, nil
	}
	r.Verification = "consistent"
	r.Lifecycle = "published"
	return r, projectError(ctx.Err())
}

// A structural check of a historical record, never re-certification of witness
// execution. QualificationVerification explicitly preserves that limitation.
func recordedQualityValid(p *preparedProject, q *TaskQualification) bool {
	c := q.CaseReport
	if c == nil || c.Format != "statetwin.dev/task-case-report/v1alpha1" || c.Source != "scripted-witness-cases" || c.Profile != "synthetic-witness-cases-v1" || c.Provenance != "not-proven" || c.Planned != len(p.cases.inputs) || len(c.Cases) != c.Planned {
		return false
	}
	matched, mismatched := 0, 0
	for i, row := range c.Cases {
		ex := p.cases.manifest.Cases[i]
		source := ""
		if len(ex.Mutations) > 0 {
			source = "synthetic-view-mutation"
		}
		if row.GradingSource != source {
			return false
		}
		if row.CaseID != ex.CaseID || row.TaskID != ex.TaskID || row.Role != ex.Role || row.ExpectedOutcome != ex.Expected.Outcome || !oneOf(row.State, "matched", "mismatched") || row.ExecutionStatus != "completed" || row.CleanupStatus != "complete" || row.FailureCode != "" || !oneOf(row.ActualOutcome, "success", "expected_abstention", "task_failed", "policy_violation", "evaluator_error", "not_evaluated") {
			return false
		}
		allowed := map[string]bool{}
		for _, a := range p.cases.tasks[row.TaskID].Oracle {
			allowed[a.ID] = true
		}
		for _, ids := range [][]string{row.FailedCheckIDs, row.ErrorCheckIDs} {
			seen := map[string]bool{}
			for _, id := range ids {
				if !allowed[id] || seen[id] {
					return false
				}
				seen[id] = true
			}
		}
		for _, id := range row.ErrorCheckIDs {
			if !containsID(row.FailedCheckIDs, id) {
				return false
			}
		}
		if oneOf(row.ActualOutcome, "evaluator_error", "not_evaluated") != (len(row.ErrorCheckIDs) > 0) {
			return false
		}
		exact := row.ActualOutcome == ex.Expected.Outcome && equalIDs(row.FailedCheckIDs, ex.Expected.FailedChecks) && (ex.Role != "unscorable-negative" || len(row.ErrorCheckIDs) > 0)
		if (row.State == "matched") != exact {
			return false
		}
		if exact {
			matched++
		} else {
			mismatched++
		}
	}
	decision := "matched"
	if mismatched > 0 {
		decision = "mismatched"
	}
	if c.Matched != matched || c.Mismatched != mismatched || c.Failed != 0 || c.NotStarted != 0 || c.Decision != decision {
		return false
	}
	return same(q, qualifyCases(p.cases, c))
}
