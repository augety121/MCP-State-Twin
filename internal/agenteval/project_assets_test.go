package agenteval

import (
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/augety121/mcp-state-twin/internal/bundle"
)

// Copy only committed-format source assets. Generated outputs never become inputs.
func campaignAssets(t testing.TB) string {
	t.Helper()
	root := t.TempDir()
	source := filepath.Join("..", "..", "examples")
	err := filepath.WalkDir(source, func(n string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, e := filepath.Rel(source, n)
		if e != nil {
			return e
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return os.MkdirAll(filepath.Join(root, "examples", rel), 0700)
		}
		if !strings.HasSuffix(n, ".json") && !strings.HasSuffix(n, ".yaml") {
			return nil
		}
		b, e := os.ReadFile(n)
		if e != nil {
			return e
		}
		return os.WriteFile(filepath.Join(root, "examples", rel), b, 0600)
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, domain := range []string{"issue-tracker", "package-registry"} {
		worlds := []string{"agent"}
		if domain == "package-registry" {
			worlds = append(worlds, "project")
		}
		dir := filepath.Join(root, "examples", domain)
		if err := os.Mkdir(filepath.Join(dir, ".statetwin"), 0700); err != nil {
			t.Fatal(err)
		}
		for _, w := range worlds {
			for _, reviewed := range []bool{false, true} {
				src := filepath.Join(dir, "bundle-"+w+".yaml")
				out := w + "-world.stb"
				if reviewed {
					src = filepath.Join(dir, "reviewed", "world", w, "bundle-"+w+".yaml")
					out = "reviewed-" + out
				}
				if _, err := bundle.Build(src, filepath.Join(dir, ".statetwin", out)); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	return root
}

func TestCampaignAssetsQualification(t *testing.T) {
	root := campaignAssets(t)
	ctx := context.Background()
	p, c, err := prepareCampaign(ctx, root, "examples/evaluation-campaign.json", "")
	if err != nil {
		b, _ := json.Marshal(c)
		t.Fatal(string(b), err)
	}
	tasks, rows, alternates := 0, 0, 0
	witnessFiles := map[string]bool{}
	for _, project := range p.projects {
		t.Run(project.manifest.ID, func(t *testing.T) {
			report, err := runCases(ctx, project.cases, RunWitness)
			if err != nil {
				t.Fatal(err)
			}
			quality := qualifyCases(project.cases, report)
			for _, row := range report.Cases {
				if row.State != "matched" {
					t.Errorf("case mismatch: %+v", row)
				}
			}
			if quality.Decision != "qualified" {
				t.Fatalf("quality reasons: %+v", quality.Reasons)
			}
			for _, task := range quality.Tasks {
				if len(task.PositiveCaseIDs) > 1 {
					var first *Witness
					distinct := false
					for i, c := range project.cases.manifest.Cases {
						if c.TaskID == task.TaskID && c.Role == "positive" {
							w := project.cases.inputs[i].witness
							if first == nil {
								first = w
							} else if !reflect.DeepEqual(first.Calls, w.Calls) {
								distinct = true
							}
						}
					}
					if !distinct {
						t.Fatal("alternate positive repeats the same trajectory", task.TaskID)
					}
					alternates++
				}
			}
		})
		tasks += len(project.cases.tasks)
		rows += len(project.cases.inputs)
		for _, c := range project.cases.manifest.Cases {
			witnessFiles[project.cases.tasks[c.TaskID].Domain+"/"+c.Witness] = true
		}
	}
	if tasks != 24 || rows != 103 || len(witnessFiles) != 97 || alternates != 18 {
		t.Fatalf("tasks=%d rows=%d witnesses=%d alternates=%d", tasks, rows, len(witnessFiles), alternates)
	}
}

func TestFourProjectCampaign(t *testing.T) {
	root := campaignAssets(t)
	ctx := context.Background()
	r, err := RunCampaign(ctx, root, "examples/evaluation-campaign.json", "delivery")
	if err != nil || r.Decision != "passed" || r.TrialCounts.Completed != 48 || r.CompletedProjects != 4 {
		b, _ := json.Marshal(r)
		t.Fatal(string(b), err)
	}
	a, err := InspectCampaign(ctx, root, "examples/evaluation-campaign.json", "delivery")
	if err != nil || a.Verification != "consistent" || a.CheckedProjects != 4 {
		b, _ := json.Marshal(a)
		t.Fatal(string(b), err)
	}
}
