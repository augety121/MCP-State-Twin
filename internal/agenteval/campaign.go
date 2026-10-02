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

const CampaignFormat = "statetwin.dev/evaluation-campaign/v1alpha1"
const CampaignProfile = "offline-reviewed-campaign-v1"

type CampaignProject struct {
	ID      string `json:"id"`
	Root    string `json:"root"`
	Project string `json:"project"`
}
type CampaignManifest struct {
	Format   string            `json:"format"`
	Profile  string            `json:"profile"`
	ID       string            `json:"id"`
	Projects []CampaignProject `json:"projects"`
}
type CampaignCheck struct {
	Format             string          `json:"format"`
	CampaignID         string          `json:"campaignId"`
	Status             string          `json:"status"`
	PlannedProjects    int             `json:"plannedProjects"`
	Projects           []*ProjectCheck `json:"projects"`
	ExecutionPerformed bool            `json:"executionPerformed"`
	Provenance         string          `json:"provenance"`
}
type preparedCampaign struct {
	manifest CampaignManifest
	projects []*preparedProject
}

func prepareCampaign(ctx context.Context, root, name, out string) (*preparedCampaign, *CampaignCheck, error) {
	r := &CampaignCheck{Format: "statetwin.dev/campaign-check/v1alpha1", Status: "invalid", Projects: []*ProjectCheck{}, Provenance: "not-proven"}
	fail := func(err error) (*preparedCampaign, *CampaignCheck, error) { return nil, r, campaignError(err) }
	s := newProjectSource(ctx, root, out, 256<<20)
	raw, err := s.read(".", name, 64<<10)
	if err != nil {
		return fail(err)
	}
	var m CampaignManifest
	if decodeSuiteMetadata(raw, 64<<10, &m) != nil || m.Format != CampaignFormat || m.Profile != CampaignProfile || !validLabel(m.ID) || len(m.Projects) < 2 || len(m.Projects) > 4 {
		return fail(errors.New("PROJECT_INPUT_INVALID"))
	}
	r.CampaignID = m.ID
	r.PlannedProjects = len(m.Projects)
	ids, paths := map[string]bool{}, map[string]bool{}
	for _, ref := range m.Projects {
		if !validLabel(ref.ID) || ids[ref.ID] || ref.Root != "." && task.PortablePath(ref.Root) != nil || task.PortablePath(ref.Project) != nil {
			return fail(errors.New("PROJECT_INPUT_INVALID"))
		}
		n := strings.ToLower(path.Join(ref.Root, ref.Project))
		if paths[n] {
			return fail(errors.New("PROJECT_INPUT_INVALID"))
		}
		paths[n] = true
		ids[ref.ID] = true
		c := checkSkeleton()
		c.ProjectID = ref.ID
		c.Status = "not_checked"
		r.Projects = append(r.Projects, c)
	}
	p := &preparedCampaign{manifest: m}
	for i, ref := range m.Projects {
		v, check, err := prepareProject(s, ref.Root, ref.Project)
		r.Projects[i] = check
		if err != nil {
			return fail(err)
		}
		if v.manifest.ID != ref.ID {
			return fail(errors.New("PROJECT_REFERENCE_MISMATCH"))
		}
		p.projects = append(p.projects, v)
	}
	// A reviewed reference must remain independent of every actual input in
	// the campaign, including aliases crossing project roots.
	for i, reviewed := range p.projects {
		for j, actual := range p.projects {
			a, b := m.Projects[i].Root, m.Projects[j].Root
			for _, ref := range reviewed.catalog.entries {
				for _, input := range actual.actualTasks {
					if !s.separate(".", path.Join(a, ref.Task), path.Join(b, input)) {
						return fail(errors.New("PROJECT_REFERENCE_MISMATCH"))
					}
				}
			}
			for _, ref := range reviewed.worlds.references {
				for _, input := range actual.actualBundles {
					if !s.separate(".", path.Join(a, ref.Bundle), path.Join(b, input)) {
						return fail(errors.New("PROJECT_REFERENCE_MISMATCH"))
					}
				}
			}
		}
	}
	r.Status = "statically_valid"
	return p, r, nil
}
func campaignError(err error) error {
	if err == nil {
		return nil
	}
	return errors.New(strings.Replace(projectError(err).Error(), "PROJECT_", "CAMPAIGN_", 1))
}
func CheckCampaign(parent context.Context, root, name string) (*CampaignCheck, error) {
	ctx, cancel := context.WithTimeout(parent, 480*time.Second)
	defer cancel()
	_, r, err := prepareCampaign(ctx, root, name, "")
	return r, err
}

type CampaignRow struct {
	ProjectID   string        `json:"projectId"`
	Lifecycle   string        `json:"lifecycle"`
	Decision    string        `json:"decision"`
	CaseCounts  ProjectCounts `json:"caseCounts"`
	TrialCounts ProjectCounts `json:"trialCounts"`
	ReasonCodes []string      `json:"reasonCodes"`
}
type CampaignReport struct {
	Format             string        `json:"format"`
	Profile            string        `json:"profile"`
	CampaignID         string        `json:"campaignId"`
	Lifecycle          string        `json:"lifecycle"`
	Decision           string        `json:"decision"`
	PlannedProjects    int           `json:"plannedProjects"`
	CompletedProjects  int           `json:"completedProjects"`
	NotStartedProjects int           `json:"notStartedProjects"`
	Projects           []CampaignRow `json:"projects"`
	CaseCounts         ProjectCounts `json:"caseCounts"`
	TrialCounts        ProjectCounts `json:"trialCounts"`
	ReasonCodes        []string      `json:"reasonCodes"`
	Provenance         string        `json:"provenance"`
}
type campaignClaim struct {
	Format     string         `json:"format"`
	CampaignID string         `json:"campaignId"`
	Projects   []projectClaim `json:"projects"`
}

func claimCampaign(p *preparedCampaign) campaignClaim {
	c := campaignClaim{Format: "statetwin.dev/campaign-claim/v1alpha1", CampaignID: p.manifest.ID, Projects: []projectClaim{}}
	for _, v := range p.projects {
		c.Projects = append(c.Projects, claimProject(v))
	}
	return c
}
func newCampaignReport(p *preparedCampaign) *CampaignReport {
	r := &CampaignReport{Format: "statetwin.dev/campaign-report/v1alpha1", Profile: CampaignProfile, CampaignID: p.manifest.ID, Lifecycle: "partial", Decision: "incomplete", PlannedProjects: len(p.projects), Projects: []CampaignRow{}, ReasonCodes: []string{}, Provenance: "not-proven"}
	for _, v := range p.projects {
		t := newProjectReport(v)
		r.Projects = append(r.Projects, CampaignRow{ProjectID: v.manifest.ID, Lifecycle: "not_started", Decision: "not_started", CaseCounts: t.CaseCounts, TrialCounts: t.TrialCounts, ReasonCodes: []string{}})
	}
	r.totals()
	return r
}
func addCounts(a *ProjectCounts, b ProjectCounts) {
	a.Planned += b.Planned
	a.Started += b.Started
	a.Completed += b.Completed
	a.Failed += b.Failed
	a.NotStarted += b.NotStarted
}
func (r *CampaignReport) totals() {
	r.CompletedProjects = 0
	r.NotStartedProjects = 0
	r.CaseCounts = ProjectCounts{}
	r.TrialCounts = ProjectCounts{}
	r.Decision = "passed"
	for _, v := range r.Projects {
		addCounts(&r.CaseCounts, v.CaseCounts)
		addCounts(&r.TrialCounts, v.TrialCounts)
		if v.Lifecycle == "published" {
			r.CompletedProjects++
			if v.Decision != "passed" && r.Decision != "incomplete" {
				r.Decision = "failed"
			}
		} else {
			r.Decision = "incomplete"
			if v.Lifecycle == "not_started" {
				r.NotStartedProjects++
			}
		}
	}
}
func RunCampaign(parent context.Context, root, name, out string) (*CampaignReport, error) {
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
	return runPreparedCampaign(ctx, root, out, p, defaultProjectOperations())
}
func runPreparedCampaign(ctx context.Context, root, out string, p *preparedCampaign, ops projectOperations) (*CampaignReport, error) {
	r := newCampaignReport(p)
	fail := func(code string) (*CampaignReport, error) {
		if ctx.Err() != nil {
			code = "CAMPAIGN_CANCELED_OR_TIMED_OUT"
		}
		r.totals()
		r.ReasonCodes = append(r.ReasonCodes, code)
		return r, errors.New(code)
	}
	if ctx.Err() != nil {
		return fail("CAMPAIGN_CANCELED_OR_TIMED_OUT")
	}
	fs, err := ops.open(root)
	if err != nil {
		return fail("CAMPAIGN_INPUT_UNAVAILABLE")
	}
	defer fs.Close()
	if outputParent(fs, out) != nil {
		return fail("CAMPAIGN_INPUT_INVALID")
	}
	if !absent(fs, out) {
		return fail("CAMPAIGN_DEST_EXISTS")
	}
	if err := fs.Mkdir(out, 0700); err != nil {
		if os.IsExist(err) {
			return fail("CAMPAIGN_DEST_EXISTS")
		}
		return fail("CAMPAIGN_WRITE_FAILED")
	}
	left := 4 << 20
	write := func(n string, v any) error {
		return campaignError(writeProjectJSON(ctx, fs, path.Join(out, n), v, &left))
	}
	if err := write("campaign-claim.json", claimCampaign(p)); err != nil {
		return fail(err.Error())
	}
	if fs.Mkdir(path.Join(out, "projects"), 0700) != nil {
		return fail("CAMPAIGN_WRITE_FAILED")
	}
	for i, v := range p.projects {
		if ctx.Err() != nil {
			return fail("CAMPAIGN_CANCELED_OR_TIMED_OUT")
		}
		child, cancel := context.WithTimeout(ctx, 120*time.Second)
		pr, err := runPreparedProject(child, root, path.Join(out, "projects", v.manifest.ID), v, ops)
		cancel()
		if pr != nil {
			r.Projects[i] = CampaignRow{ProjectID: pr.ProjectID, Lifecycle: pr.Lifecycle, Decision: pr.Decision, CaseCounts: pr.CaseCounts, TrialCounts: pr.TrialCounts, ReasonCodes: pr.ReasonCodes}
		}
		if err != nil {
			return fail(campaignError(err).Error())
		}
	}
	r.totals()
	r.Lifecycle = "published"
	if err := write("campaign-report.pending.json", r); err != nil {
		r.Lifecycle = "partial"
		return fail(err.Error())
	}
	if ctx.Err() != nil {
		r.Lifecycle = "partial"
		return fail("CAMPAIGN_CANCELED_OR_TIMED_OUT")
	}
	if fs.Link(path.Join(out, "campaign-report.pending.json"), path.Join(out, "campaign-report.json")) != nil {
		r.Lifecycle = "partial"
		return fail("CAMPAIGN_PUBLISH_FAILED")
	}
	if fs.Remove(path.Join(out, "campaign-report.pending.json")) != nil {
		r.Lifecycle = "published_with_residue"
		return fail("CAMPAIGN_CLEANUP_FAILED")
	}
	return r, nil
}
