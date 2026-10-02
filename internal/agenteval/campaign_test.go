package agenteval

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/augety121/mcp-state-twin/internal/bundle"
	"github.com/augety121/mcp-state-twin/internal/task"
)

func campaignFixture(t *testing.T, n int) (string, *CampaignManifest) {
	t.Helper()
	root, project := projectFixture(t)
	m := &CampaignManifest{Format: CampaignFormat, Profile: CampaignProfile, ID: "campaign", Projects: []CampaignProject{}}
	for i := 0; i < n; i++ {
		p := *project
		p.ID = fmt.Sprintf("project-%d", i)
		name := p.ID + ".json"
		writeTestJSON(t, root, name, p)
		m.Projects = append(m.Projects, CampaignProject{p.ID, ".", name})
	}
	writeTestJSON(t, root, "campaign.json", m)
	return root, m
}

func TestCampaignAllInputPreflight(t *testing.T) {
	for _, n := range []int{1, 2, 3, 4, 5} {
		root, m := campaignFixture(t, n)
		r, err := CheckCampaign(context.Background(), root, "campaign.json")
		if n < 2 || n > 4 {
			if err == nil {
				t.Fatal("group count", n)
			}
			continue
		}
		if err != nil || r.ExecutionPerformed {
			t.Fatal(r, err)
		}
		os.WriteFile(filepath.Join(root, m.Projects[n-1].Project), []byte("{}"), 0600)
		before := suiteSnapshot(t, root)
		if _, err := RunCampaign(context.Background(), root, "campaign.json", "delivery"); err == nil {
			t.Fatal("late bad reference accepted")
		}
		if !reflect.DeepEqual(before, suiteSnapshot(t, root)) {
			t.Fatal("wrote before whole preflight")
		}
	}
	for _, variant := range []string{"duplicate-id", "duplicate-reference", "escape-root", "bad-id", "nested-output"} {
		t.Run(variant, func(t *testing.T) {
			root, m := campaignFixture(t, 2)
			switch variant {
			case "duplicate-id":
				m.Projects[1].ID = m.Projects[0].ID
			case "duplicate-reference":
				m.Projects[1].Project = m.Projects[0].Project
			case "escape-root":
				m.Projects[1].Root = "../outside"
			case "bad-id":
				m.Projects[1].ID = "bad|id"
			case "nested-output":
				m.Projects[1].Root = "delivery"
			}
			writeTestJSON(t, root, "campaign.json", m)
			if _, err := RunCampaign(context.Background(), root, "campaign.json", "delivery"); err == nil {
				t.Fatal("invalid admitted")
			}
		})
	}
}

func TestCampaignStopPolicy(t *testing.T) {
	for _, mode := range []string{"quality-gate", "task-gate", "cleanup", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			root, _ := campaignFixture(t, 3)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			p, _, err := prepareCampaign(ctx, root, "campaign.json", "delivery")
			if err != nil {
				t.Fatal(err)
			}
			ops := defaultProjectOperations()
			switch mode {
			case "quality-gate":
				copy := *p.projects[0].cases.inputs[0].witness
				copy.Calls = []Call{}
				p.projects[0].cases.inputs[0].witness = &copy
			case "task-gate":
				raw, _ := json.Marshal(mockWitness(&Witness{Kind: "TaskWitness", TaskID: "close-issue", SyntheticOnly: true, Calls: []Call{}}))
				p.projects[0].suite.trials[1].responses = raw
			default:
				count := 0
				ops.witness = func(ctx context.Context, t *task.Task, b *bundle.Artifact, w *Witness) (*Report, error) {
					count++
					if count == 5 {
						if mode == "cancel" {
							cancel()
							return nil, ctx.Err()
						}
						return &Report{CleanupStatus: "failed"}, errors.New("synthetic cleanup failure")
					}
					return RunWitness(ctx, t, b, w)
				}
			}
			r, err := runPreparedCampaign(ctx, root, "delivery", p, ops)
			if r.PlannedProjects != 3 || r.TrialCounts.Planned != 6 || r.CaseCounts.Planned != 12 || len(r.Projects) != 3 {
				t.Fatal("denominator", r)
			}
			if mode == "quality-gate" || mode == "task-gate" {
				if err != nil || r.Lifecycle != "published" || r.Decision != "failed" || r.CompletedProjects != 3 || r.Projects[2].Decision != "passed" {
					t.Fatal(r, err)
				}
			} else {
				if err == nil || r.Lifecycle != "partial" || r.NotStartedProjects != 1 || r.Projects[2].Lifecycle != "not_started" || r.CompletedProjects != 1 {
					t.Fatal(r, err)
				}
			}
		})
	}
}

func TestCampaignDenominators(t *testing.T) {
	root, _ := campaignFixture(t, 2)
	ctx := context.Background()
	r, err := RunCampaign(ctx, root, "campaign.json", "delivery")
	if err != nil || r.TrialCounts.Completed != 4 {
		t.Fatal(r, err)
	}
	a, err := InspectCampaign(ctx, root, "campaign.json", "delivery")
	if err != nil || a.Verification != "consistent" {
		t.Fatal(a, err)
	}
	if _, err := RunCampaign(ctx, root, "campaign.json", "delivery"); err == nil {
		t.Fatal("overwrite")
	}
	original, _ := os.ReadFile(filepath.Join(root, "delivery", "campaign-report.json"))
	for _, mode := range []string{"drop", "count", "duplicate", "pending", "missing"} {
		t.Run(mode, func(t *testing.T) {
			var c CampaignReport
			json.Unmarshal(original, &c)
			var restore func()
			switch mode {
			case "drop":
				c.Projects = c.Projects[:1]
			case "count":
				c.TrialCounts.Planned--
			case "duplicate":
				c.Projects[1] = c.Projects[0]
			case "pending":
				os.WriteFile(filepath.Join(root, "delivery", "campaign-report.pending.json"), original, 0600)
				restore = func() { os.Remove(filepath.Join(root, "delivery", "campaign-report.pending.json")) }
			case "missing":
				name := filepath.Join(root, "delivery", "projects", "project-1", "project-report.json")
				raw, _ := os.ReadFile(name)
				os.Remove(name)
				restore = func() { os.WriteFile(name, raw, 0600) }
			}
			if restore != nil {
				defer restore()
			}
			writeTestJSON(t, root, "delivery/campaign-report.json", c)
			a, err := InspectCampaign(ctx, root, "campaign.json", "delivery")
			if err != nil || a.Verification == "consistent" {
				t.Fatal(a, err)
			}
			os.WriteFile(filepath.Join(root, "delivery", "campaign-report.json"), original, 0600)
		})
	}
}

func TestCampaignPublicationFaults(t *testing.T) {
	root, _ := campaignFixture(t, 2)
	p, _, err := prepareCampaign(context.Background(), root, "campaign.json", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, point := range []struct{ op, leaf string }{{"mkdir", "projects"}, {"open", "campaign-claim.json"}, {"write", "campaign-report.pending.json"}, {"short", "campaign-report.pending.json"}, {"sync", "campaign-report.pending.json"}, {"close", "campaign-report.pending.json"}, {"link", "campaign-report.json"}, {"remove", "campaign-report.pending.json"}} {
		t.Run(point.op, func(t *testing.T) {
			ops := defaultProjectOperations()
			ops.witness = failQualityRunner
			ops.open = func(root string) (evidenceFS, error) {
				fs, e := openRootedEvidenceFS(root)
				if e != nil {
					return nil, e
				}
				return &failingEvidenceFS{evidenceFS: fs, op: point.op, leaf: point.leaf}, nil
			}
			r, err := runPreparedCampaign(context.Background(), root, "fault-"+point.op, p, ops)
			if err == nil || r.Lifecycle == "published" || r.TrialCounts.Planned != 4 {
				t.Fatal(r, err)
			}
			if point.op == "remove" && r.Lifecycle != "published_with_residue" {
				t.Fatal(r)
			}
		})
	}
}

func TestCampaignFrozenInputs(t *testing.T) {
	root, _ := campaignFixture(t, 2)
	p, _, err := prepareCampaign(context.Background(), root, "campaign.json", "delivery")
	if err != nil {
		t.Fatal(err)
	}
	ops := defaultProjectOperations()
	started := false
	ops.witness = func(ctx context.Context, t *task.Task, b *bundle.Artifact, w *Witness) (*Report, error) {
		if !started {
			started = true
			os.WriteFile(filepath.Join(root, "positive.json"), []byte("changed while first project runs"), 0600)
			os.WriteFile(filepath.Join(root, "project-1.json"), []byte("{}"), 0600)
		}
		return RunWitness(ctx, t, b, w)
	}
	r, err := runPreparedCampaign(context.Background(), root, "delivery", p, ops)
	if err != nil || r.Decision != "passed" || r.CompletedProjects != 2 {
		t.Fatal(r, err)
	}
}
