package agenteval

import (
	"context"
	"errors"
	"os"
	"path"
	"time"

	"github.com/augety121/mcp-state-twin/internal/task"
)

type CampaignProjectInspection struct {
	ProjectID                 string   `json:"projectId"`
	Verification              string   `json:"verification"`
	HistoricalDecision        string   `json:"historicalDecision"`
	QualificationVerification string   `json:"qualificationVerification"`
	ReasonCodes               []string `json:"reasonCodes"`
}
type CampaignInspection struct {
	Format             string                      `json:"format"`
	CampaignID         string                      `json:"campaignId"`
	Verification       string                      `json:"verification"`
	HistoricalDecision string                      `json:"historicalDecision"`
	PlannedProjects    int                         `json:"plannedProjects"`
	CheckedProjects    int                         `json:"checkedProjects"`
	Projects           []CampaignProjectInspection `json:"projects"`
	ReasonCodes        []string                    `json:"reasonCodes"`
	Provenance         string                      `json:"provenance"`
}

func InspectCampaign(parent context.Context, root, name, out string) (*CampaignInspection, error) {
	ctx, cancel := context.WithTimeout(parent, 480*time.Second)
	defer cancel()
	if ctx.Err() != nil {
		return nil, campaignError(ctx.Err())
	}
	if task.PortablePath(out) != nil {
		return nil, errors.New("CAMPAIGN_INPUT_INVALID")
	}
	p, _, err := prepareCampaign(ctx, root, name, out)
	if err != nil {
		return nil, err
	}
	fs, err := os.OpenRoot(root)
	if err != nil {
		return nil, errors.New("CAMPAIGN_INPUT_UNAVAILABLE")
	}
	defer fs.Close()
	return inspectPreparedCampaign(ctx, diskReadRoot{fs}, out, p)
}
func inspectPreparedCampaign(ctx context.Context, fs evidenceReadRoot, out string, p *preparedCampaign) (*CampaignInspection, error) {
	r := &CampaignInspection{Format: "statetwin.dev/campaign-inspection/v1alpha1", CampaignID: p.manifest.ID, Verification: "incomplete", HistoricalDecision: "unknown", PlannedProjects: len(p.projects), Projects: []CampaignProjectInspection{}, ReasonCodes: []string{}, Provenance: "not-proven"}
	for _, v := range p.projects {
		r.Projects = append(r.Projects, CampaignProjectInspection{ProjectID: v.manifest.ID, Verification: "not_checked", HistoricalDecision: "unknown", QualificationVerification: "recorded_only", ReasonCodes: []string{}})
	}
	bad := func(code string) (*CampaignInspection, error) {
		r.Verification = "mismatched"
		r.ReasonCodes = append(r.ReasonCodes, code)
		return r, nil
	}
	if ctx.Err() != nil {
		return r, campaignError(ctx.Err())
	}
	if err := safeProjectDir(fs, out); err != nil {
		if os.IsNotExist(err) {
			return r, nil
		}
		return bad("CAMPAIGN_INPUT_INVALID")
	}
	entries, err := suiteEntries(fs, out, 5)
	if err != nil {
		return bad("CAMPAIGN_INPUT_INVALID")
	}
	present := map[string]bool{}
	for _, e := range entries {
		if !oneOf(e.Name(), "campaign-claim.json", "projects", "campaign-report.pending.json", "campaign-report.json") {
			return bad("CAMPAIGN_INPUT_INVALID")
		}
		present[e.Name()] = true
		info, statErr := fs.Lstat(path.Join(out, e.Name()))
		if statErr != nil || info.Mode()&os.ModeSymlink != 0 || (e.Name() == "projects" && !info.IsDir()) || (e.Name() != "projects" && !info.Mode().IsRegular()) {
			return bad("CAMPAIGN_INPUT_INVALID")
		}
	}
	left := 4 << 20
	read := func(n string, v any) error {
		b, err := readArtifact(fs, path.Join(out, n), min(left, projectReportLimit))
		if err != nil {
			return err
		}
		left -= len(b)
		return decodeSuiteMetadata(b, projectReportLimit, v)
	}
	if !present["campaign-claim.json"] {
		return r, nil
	}
	var claim campaignClaim
	if read("campaign-claim.json", &claim) != nil || !same(claim, claimCampaign(p)) {
		return bad("CAMPAIGN_REFERENCE_MISMATCH")
	}
	if !present["projects"] {
		return r, nil
	}
	if err := safeProjectDir(fs, path.Join(out, "projects")); err != nil {
		return bad("CAMPAIGN_INPUT_INVALID")
	}
	children, err := suiteEntries(fs, path.Join(out, "projects"), 5)
	if err != nil {
		return bad("CAMPAIGN_INPUT_INVALID")
	}
	allowed := map[string]bool{}
	for _, v := range p.projects {
		allowed[v.manifest.ID] = true
	}
	for _, c := range children {
		if !allowed[c.Name()] {
			return bad("CAMPAIGN_INPUT_INVALID")
		}
	}
	want := newCampaignReport(p)
	complete := true
	for i, v := range p.projects {
		if ctx.Err() != nil {
			return r, campaignError(ctx.Err())
		}
		child, cancel := context.WithTimeout(ctx, 120*time.Second)
		pr, err := inspectPreparedProject(child, fs, path.Join(out, "projects", v.manifest.ID), v)
		cancel()
		if err != nil {
			return r, campaignError(err)
		}
		r.CheckedProjects++
		r.Projects[i] = CampaignProjectInspection{ProjectID: v.manifest.ID, Verification: pr.Verification, HistoricalDecision: pr.HistoricalDecision, QualificationVerification: "recorded_only", ReasonCodes: pr.ReasonCodes}
		if pr.Verification == "mismatched" {
			return bad("CAMPAIGN_REFERENCE_MISMATCH")
		}
		if pr.Verification != "consistent" {
			complete = false
			continue
		}
		// Already independently inspected. Read only the bounded report for projection.
		raw, err := readArtifact(fs, path.Join(out, "projects", v.manifest.ID, "project-report.json"), projectReportLimit)
		if err != nil {
			return bad("CAMPAIGN_INPUT_INVALID")
		}
		var saved ProjectReport
		if decodeSuiteMetadata(raw, projectReportLimit, &saved) != nil {
			return bad("CAMPAIGN_INPUT_INVALID")
		}
		want.Projects[i] = CampaignRow{ProjectID: saved.ProjectID, Lifecycle: saved.Lifecycle, Decision: saved.Decision, CaseCounts: saved.CaseCounts, TrialCounts: saved.TrialCounts, ReasonCodes: saved.ReasonCodes}
	}
	if !present["campaign-report.json"] {
		if present["campaign-report.pending.json"] {
			var pending CampaignReport
			if read("campaign-report.pending.json", &pending) != nil {
				return bad("CAMPAIGN_INPUT_INVALID")
			}
		}
		return r, nil
	}
	var saved CampaignReport
	if read("campaign-report.json", &saved) != nil {
		return bad("CAMPAIGN_INPUT_INVALID")
	}
	r.HistoricalDecision = saved.Decision
	if !complete {
		return bad("CAMPAIGN_REFERENCE_MISMATCH")
	}
	want.totals()
	want.Lifecycle = "published"
	if !same(saved, want) {
		return bad("CAMPAIGN_REFERENCE_MISMATCH")
	}
	if present["campaign-report.pending.json"] {
		var pending CampaignReport
		if read("campaign-report.pending.json", &pending) != nil || !same(saved, pending) {
			return bad("CAMPAIGN_INPUT_INVALID")
		}
		r.ReasonCodes = append(r.ReasonCodes, "CAMPAIGN_CLEANUP_FAILED")
		return r, nil
	}
	r.Verification = "consistent"
	return r, nil
}
