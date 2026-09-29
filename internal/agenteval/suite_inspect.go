package agenteval

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/augety121/mcp-state-twin/internal/limits"
	"github.com/augety121/mcp-state-twin/internal/task"
)

const SuiteInspectionFormat = "statetwin.dev/agent-suite-inspection/v1alpha1"

type SuiteTrialInspection struct {
	TrialID    string      `json:"trialId"`
	Inspection *Inspection `json:"inspection"`
}

type SuiteInspection struct {
	Format               string                 `json:"format"`
	State                string                 `json:"state"`
	Problem              string                 `json:"problem,omitempty"`
	PlannedTrials        int                    `json:"plannedTrials"`
	CompleteTrials       int                    `json:"completeTrials"`
	StagingResidue       bool                   `json:"stagingResidue"`
	ReportVerification   string                 `json:"reportVerification"`
	Trials               []SuiteTrialInspection `json:"trials"`
	Comparison           *Comparison            `json:"comparison,omitempty"`
	RegressionGatePassed bool                   `json:"regressionGatePassed"`
	UpgradeAllowed       bool                   `json:"upgradeAllowed"`
	ResumeAllowed        bool                   `json:"resumeAllowed"`
	SnapshotAtomic       bool                   `json:"snapshotAtomic"`
}

// InspectSuite never promotes or repairs artifacts. The root and its writers
// must be trusted and quiescent across inventory, inspection and comparison.
func InspectSuite(ctx context.Context, root, out string) (*SuiteInspection, error) {
	return inspectSuite(ctx, root, out, maxSuiteWriteBytes)
}

func inspectSuite(parent context.Context, root, out string, sizeLimit int64) (*SuiteInspection, error) {
	return inspectSuiteObserved(parent, root, out, sizeLimit, nil, nil)
}

func inspectSuiteObserved(parent context.Context, root, out string, sizeLimit int64, planObserver func(ComparePlan), observe definitionObserver) (*SuiteInspection, error) {
	if task.PortablePath(out) != nil {
		return nil, errors.New("SUITE_INSPECT_PATH_INVALID")
	}
	ctx, cancel := context.WithTimeout(parent, 120*time.Second)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r := &SuiteInspection{Format: SuiteInspectionFormat, State: "incomplete_or_running", ReportVerification: "absent", Trials: []SuiteTrialInspection{}}
	invalid := func(problem string) (*SuiteInspection, error) {
		r.State, r.Problem, r.RegressionGatePassed = "invalid", problem, false
		return r, nil
	}
	fs, err := os.OpenRoot(root)
	if err != nil {
		return nil, errors.New("SUITE_INSPECT_ROOT_UNAVAILABLE")
	}
	defer fs.Close()
	parts := strings.Split(out, "/")
	for i := range parts {
		info, err := fs.Lstat(strings.Join(parts[:i+1], "/"))
		if os.IsNotExist(err) {
			r.State = "not_started"
			return r, nil
		}
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil, errors.New("SUITE_INSPECT_PATH_INVALID")
		}
	}
	entries, err := suiteEntries(fs, out, 4+2*maxSuitePairs+1)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return r, nil
	}
	metadata := map[string]int{"claim.json": MaxSuitePlanBytes, "plan.json": MaxSuitePlanBytes, "report.json": maxSuiteReportBytes, "report.pending.json": maxSuiteReportBytes}
	present := map[string]bool{}
	for _, e := range entries {
		present[e.Name()] = true
	}
	readMeta := func(name string, target any) error {
		raw, err := readArtifact(fs, path.Join(out, name), metadata[name])
		if err != nil {
			return errors.New("SUITE_METADATA_INVALID")
		}
		return decodeSuiteMetadata(raw, metadata[name], target)
	}
	claimRaw, err := readArtifact(fs, path.Join(out, "claim.json"), MaxSuitePlanBytes)
	if err != nil {
		return invalid("claim_missing_or_invalid")
	}
	claim, err := decodeSuiteClaim(claimRaw)
	if err != nil {
		return invalid("claim_missing_or_invalid")
	}
	r.PlannedTrials = claim.PlannedTrials
	allowedDirs := map[string]bool{}
	for _, pair := range claim.Plan.Pairs {
		allowedDirs[pair.Baseline.TrialID], allowedDirs[pair.Candidate.TrialID] = true, true
	}
	for _, entry := range entries {
		info, err := fs.Lstat(path.Join(out, entry.Name()))
		if err != nil || info.Mode()&os.ModeSymlink != 0 {
			return invalid("unexpected_or_unsafe_member")
		}
		if limit, ok := metadata[entry.Name()]; ok {
			if !info.Mode().IsRegular() || info.Size() > int64(limit) {
				return invalid("unexpected_or_unsafe_member")
			}
		} else if !allowedDirs[entry.Name()] || !info.IsDir() {
			return invalid("unexpected_or_unsafe_member")
		}
	}
	if !present["plan.json"] {
		if len(entries) != 1 {
			return invalid("plan_missing_or_invalid")
		}
		return r, nil
	}
	var plan ComparePlan
	if readMeta("plan.json", &plan) != nil || validateSuiteComparisonPlan(&plan) != nil || !same(plan, claim.Plan) {
		return invalid("plan_missing_or_invalid")
	}
	if planObserver != nil {
		planObserver(plan)
	}
	// Inspect the complete inventory before any replay work. Enumeration is
	// bounded, and file-size admission never opens an unexpected FIFO/device.
	var total int64
	for _, pair := range plan.Pairs {
		for _, trial := range []PlannedTrial{pair.Baseline, pair.Candidate} {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			if !present[trial.TrialID] {
				continue
			}
			members, err := suiteEntries(fs, path.Join(out, trial.TrialID), len(artifactNames)+1)
			if err != nil {
				return nil, err
			}
			for _, member := range members {
				if !oneOf(member.Name(), artifactNames...) {
					return invalid("unexpected_or_unsafe_trial_member")
				}
				info, err := fs.Lstat(path.Join(out, trial.TrialID, member.Name()))
				if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() > limits.MaxReportBytes {
					return invalid("unexpected_or_unsafe_trial_member")
				}
				if info.Size() > sizeLimit-total {
					return nil, errors.New("SUITE_INSPECT_RESOURCE_LIMIT")
				}
				total += info.Size()
			}
		}
	}
	var published, pending *SuiteReport
	for _, name := range []string{"report.pending.json", "report.json"} {
		if !present[name] {
			continue
		}
		raw, err := readArtifact(fs, path.Join(out, name), maxSuiteReportBytes)
		if err != nil {
			return invalid("report_invalid")
		}
		report, err := decodeSuiteReport(raw, &plan)
		if err != nil {
			return invalid("report_invalid")
		}
		if name == "report.json" {
			published = report
		} else {
			pending = report
			r.StagingResidue = true
		}
		r.ReportVerification = "unverified"
	}
	if published != nil && pending != nil && !same(published, pending) {
		return invalid("report_publication_conflict")
	}
	for _, pair := range plan.Pairs {
		for j, trial := range []PlannedTrial{pair.Baseline, pair.Candidate} {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			i, err := InspectDirectory(ctx, root, path.Join(out, trial.TrialID))
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			if err != nil {
				if err.Error() == "EVIDENCE_INSPECT_CANCELED_OR_TIMED_OUT" {
					return nil, context.DeadlineExceeded
				}
				return nil, errors.New("SUITE_INSPECT_TRIAL_FAILED")
			}
			r.Trials = append(r.Trials, SuiteTrialInspection{TrialID: trial.TrialID, Inspection: i})
			r.StagingResidue = r.StagingResidue || i.StagingResidue
			if i.State == "invalid" {
				return invalid("trial_invalid")
			}
			if i.TrialID != "" {
				model := plan.BaselineModel
				if j == 1 {
					model = plan.CandidateModel
				}
				raw, err := readArtifact(fs, path.Join(out, trial.TrialID, "claim.json"), 16<<10)
				if err != nil {
					return invalid("trial_claim_mismatch")
				}
				c, err := DecodeRun(raw)
				if err != nil || i.Lane != "offline" || c.TrialID != trial.TrialID || c.Model != model {
					return invalid("trial_claim_mismatch")
				}
			}
			if i.EvidenceComplete {
				r.CompleteTrials++
			}
		}
	}
	r.Comparison, err = compareObserved(ctx, filepath.Join(root, filepath.FromSlash(out)), &plan, VerifyEvidence, maxCompareEvidenceBytes, observe)
	if err != nil {
		if errors.Is(err, errAssessmentResourceLimit) {
			return nil, err
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, errors.New("SUITE_INSPECT_COMPARISON_FAILED")
	}
	for _, report := range []*SuiteReport{pending, published} {
		if report == nil {
			continue
		}
		if report.ComparisonStatus == "complete" && !same(report.Comparison, r.Comparison) {
			r.ReportVerification = "mismatch"
			return invalid("report_comparison_mismatch")
		}
		if report.ExecutionStatus != "completed" {
			continue
		}
		if r.CompleteTrials != r.PlannedTrials {
			return invalid("report_trial_mismatch")
		}
		for i, row := range report.Trials {
			pair := r.Comparison.Pairs[i/2]
			trial := pair.Baseline
			if i%2 == 1 {
				trial = pair.Candidate
			}
			if !oneOf(trial.Validation, "valid", "not_evaluated") || trial.ExecutionStatus != row.ExecutionStatus || trial.Outcome != row.Outcome {
				return invalid("report_trial_mismatch")
			}
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if published != nil {
		r.State = "published_unverified"
		if published.ExecutionStatus == "completed" && published.ComparisonStatus == "complete" {
			r.State, r.ReportVerification = "published", "matched"
			if r.StagingResidue {
				r.State = "published_with_residue"
			}
			r.RegressionGatePassed = !r.StagingResidue && r.Comparison.Decision == "no_regression_observed"
		}
	}
	return r, nil
}

func suiteEntries(fs *os.Root, dir string, limit int) ([]os.DirEntry, error) {
	f, err := fs.Open(dir)
	if err != nil {
		return nil, errors.New("SUITE_INSPECT_READ_FAILED")
	}
	entries, readErr := f.ReadDir(limit)
	closeErr := f.Close()
	if (readErr != nil && !errors.Is(readErr, io.EOF)) || closeErr != nil {
		return nil, errors.New("SUITE_INSPECT_READ_FAILED")
	}
	return entries, nil
}

func (r *SuiteInspection) Markdown() string {
	var out strings.Builder
	fmt.Fprintf(&out, "# Offline suite evidence inspection\n\nState: `%s`. Report verification: `%s`. Regression gate: `%t`.\n\nPlanned trials: %d. Complete evidence: %d. Staging residue: %t.\n\nRead-only observations; not an atomic snapshot or proof of process liveness. No resume or automatic upgrade authorization. Synthetic evidence only; provenance not proven.\n", r.State, r.ReportVerification, r.RegressionGatePassed, r.PlannedTrials, r.CompleteTrials, r.StagingResidue)
	if r.Problem != "" {
		fmt.Fprintf(&out, "\nProblem: `%s`.\n", r.Problem)
	}
	out.WriteString("\n| Trial | Directory state | Evidence complete | Staging residue |\n|---|---|---|---|\n")
	for _, trial := range r.Trials {
		i := trial.Inspection
		fmt.Fprintf(&out, "| %s | %s | %t | %t |\n", trial.TrialID, i.State, i.EvidenceComplete, i.StagingResidue)
	}
	if r.Comparison != nil {
		out.WriteString("\n" + r.Comparison.Markdown())
	}
	return out.String()
}
