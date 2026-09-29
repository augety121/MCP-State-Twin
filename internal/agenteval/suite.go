package agenteval

import (
	"context"
	"errors"
	"path"
	"path/filepath"
	"time"

	"github.com/augety121/mcp-state-twin/internal/agenthost"
	"github.com/augety121/mcp-state-twin/internal/task"
)

const SuiteReportFormat = "statetwin.dev/agent-suite-report/v1alpha1"

type SuiteTrialResult struct {
	TrialID         string `json:"trialId"`
	State           string `json:"state"`
	ExecutionStatus string `json:"executionStatus,omitempty"`
	EvidenceStatus  string `json:"evidenceStatus,omitempty"`
	CleanupStatus   string `json:"cleanupStatus,omitempty"`
	Outcome         string `json:"outcome,omitempty"`
	FailureCode     string `json:"failureCode,omitempty"`
}

type SuiteReport struct {
	Format           string             `json:"format"`
	Profile          string             `json:"profile"`
	ExecutionStatus  string             `json:"executionStatus"`
	PlannedTrials    int                `json:"plannedTrials"`
	Trials           []SuiteTrialResult `json:"trials"`
	FailureCode      string             `json:"failureCode,omitempty"`
	ComparisonStatus string             `json:"comparisonStatus"`
	Comparison       *Comparison        `json:"comparison,omitempty"`
	UpgradeAllowed   bool               `json:"upgradeAllowed"`
}

func RunSuite(ctx context.Context, root, out string, prepared *PreparedSuite) (*SuiteReport, error) {
	return runSuite(ctx, root, out, prepared, openRootedEvidenceFS, maxSuiteWriteBytes)
}

func runSuite(parent context.Context, root, out string, prepared *PreparedSuite, open openEvidenceFS, writeLimit int) (*SuiteReport, error) {
	if prepared == nil || len(prepared.trials) == 0 || len(prepared.trials) != len(prepared.plan.Pairs)*2 || prepared.plan.Validate() != nil {
		return nil, errors.New("SUITE_PLAN_INVALID")
	}
	if task.PortablePath(out) != nil {
		return nil, errors.New("SUITE_OUTPUT_UNAVAILABLE")
	}
	ctx, cancel := context.WithTimeout(parent, 120*time.Second)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	budget := &suiteWriteBudget{remaining: writeLimit}
	limitedOpen := func(root string) (evidenceFS, error) {
		fs, err := open(root)
		if err != nil {
			return nil, err
		}
		return &suiteBudgetFS{evidenceFS: fs, budget: budget}, nil
	}
	claim := struct {
		Format string `json:"format"`
		SuitePreflight
	}{"statetwin.dev/agent-suite-claim/v1alpha1", prepared.Summary()}
	w, err := claimEvidence(root, out, claim, limitedOpen)
	if err != nil {
		if budget.exhausted {
			return nil, errors.New("SUITE_RESOURCE_LIMIT")
		}
		return nil, errors.New("SUITE_OUTPUT_UNAVAILABLE")
	}
	defer w.fs.Close()
	if err := writeSuiteJSON(w, "plan.json", prepared.plan); err != nil {
		if budget.exhausted {
			return nil, errors.New("SUITE_RESOURCE_LIMIT")
		}
		return nil, err
	}
	r := &SuiteReport{Format: SuiteReportFormat, Profile: SuiteProfile, ExecutionStatus: "completed", PlannedTrials: len(prepared.trials), ComparisonStatus: "not_attempted", Trials: make([]SuiteTrialResult, len(prepared.trials))}
	for i, trial := range prepared.trials {
		r.Trials[i] = SuiteTrialResult{TrialID: trial.config.TrialID, State: "not_started"}
	}
	var cause error
	for i, trial := range prepared.trials {
		if err := ctx.Err(); err != nil {
			cause = err
			break
		}
		row := &r.Trials[i]
		row.State = "failed"
		t, decodeErr := task.Decode(trial.taskBytes)
		m, mockErr := agenthost.DecodeMock(trial.responses)
		if decodeErr != nil || mockErr != nil {
			cause = errors.New("SUITE_INPUT_INVALID")
			row.FailureCode = "SUITE_INPUT_INVALID"
			break
		}
		cfg := trial.config
		ep, runErr := recordMockWithStorage(ctx, root, path.Join(out, cfg.TrialID), t, trial.bundleBytes, &cfg, m, limitedOpen)
		if ep != nil {
			row.ExecutionStatus, row.EvidenceStatus, row.CleanupStatus = ep.ExecutionStatus, ep.EvidenceStatus, ep.CleanupStatus
			if ep.Evaluation != nil {
				row.Outcome = ep.Evaluation.Outcome
			}
		}
		if runErr != nil || ep == nil || ep.ExecutionStatus != "completed" || ep.EvidenceStatus != "complete" || ep.CleanupStatus != "complete" {
			row.FailureCode = "SUITE_TRIAL_FAILED"
			cause = errors.New(row.FailureCode)
			break
		}
		row.State = "completed"
	}
	if err := ctx.Err(); err != nil {
		cause = err
		r.ExecutionStatus, r.FailureCode = "canceled", "SUITE_CANCELED"
		if errors.Is(err, context.DeadlineExceeded) {
			r.ExecutionStatus, r.FailureCode = "timed_out", "SUITE_DEADLINE_EXCEEDED"
		}
	} else if cause != nil {
		r.ExecutionStatus, r.FailureCode = "stopped", "SUITE_TRIAL_FAILED"
	}
	if budget.exhausted {
		r.ExecutionStatus, r.FailureCode = "stopped", "SUITE_RESOURCE_LIMIT"
		cause = errors.Join(cause, errors.New("SUITE_RESOURCE_LIMIT"))
	}
	if ctx.Err() == nil {
		comparison, compareErr := Compare(ctx, filepath.Join(root, filepath.FromSlash(out)), &prepared.plan)
		if compareErr != nil {
			r.ComparisonStatus = "failed"
			if cause == nil {
				r.FailureCode = "SUITE_COMPARISON_FAILED"
				cause = errors.New(r.FailureCode)
			}
		} else {
			r.ComparisonStatus, r.Comparison = "complete", comparison
		}
	}
	// Cancellation during comparison must remain cancellation, not a generic
	// comparison failure or a completed suite claim.
	if err := ctx.Err(); err != nil {
		cause = err
		r.ExecutionStatus, r.FailureCode = "canceled", "SUITE_CANCELED"
		if errors.Is(err, context.DeadlineExceeded) {
			r.ExecutionStatus, r.FailureCode = "timed_out", "SUITE_DEADLINE_EXCEEDED"
		}
	}
	if err := publishSuiteReport(w, r); err != nil {
		if budget.exhausted {
			err = errors.New("SUITE_RESOURCE_LIMIT")
		}
		return r, errors.Join(cause, err)
	}
	return r, cause
}

func writeSuiteJSON(w *evidenceWriter, name string, value any) error {
	if name != "plan.json" && name != "report.pending.json" {
		return errors.New("SUITE_REPORT_WRITE_FAILED")
	}
	raw, err := prepareEvidence(value)
	if err != nil {
		return errors.New("SUITE_REPORT_WRITE_FAILED")
	}
	if writeExclusiveBytes(w.fs, path.Join(w.out, name), raw) != nil {
		return errors.New("SUITE_REPORT_WRITE_FAILED")
	}
	return nil
}

func publishSuiteReport(w *evidenceWriter, r *SuiteReport) error {
	if err := writeSuiteJSON(w, "report.pending.json", r); err != nil {
		return err
	}
	if w.fs.Link(path.Join(w.out, "report.pending.json"), path.Join(w.out, "report.json")) != nil {
		return errors.New("SUITE_REPORT_PUBLISH_FAILED")
	}
	if w.fs.Remove(path.Join(w.out, "report.pending.json")) != nil {
		return errors.New("SUITE_REPORT_CLEANUP_FAILED")
	}
	return nil
}

// All suite and trial writes are synchronous and share this conservative
// allowance, including staging writes that are later removed.
type suiteWriteBudget struct {
	remaining int
	exhausted bool
}
type suiteBudgetFS struct {
	evidenceFS
	budget *suiteWriteBudget
}
type suiteBudgetFile struct {
	evidenceFile
	budget *suiteWriteBudget
}

func (f *suiteBudgetFS) CreateExclusive(name string) (evidenceFile, error) {
	file, err := f.evidenceFS.CreateExclusive(name)
	if err != nil {
		return nil, err
	}
	return &suiteBudgetFile{evidenceFile: file, budget: f.budget}, nil
}

func (f *suiteBudgetFile) Write(p []byte) (int, error) {
	if len(p) > f.budget.remaining {
		f.budget.exhausted = true
		return 0, errors.New("SUITE_RESOURCE_LIMIT")
	}
	n, err := f.evidenceFile.Write(p)
	f.budget.remaining -= n
	return n, err
}
