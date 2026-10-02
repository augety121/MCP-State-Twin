package agenteval

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/augety121/mcp-state-twin/internal/bundle"
	"github.com/augety121/mcp-state-twin/internal/task"
)

func TestProjectFailureAccounting(t *testing.T) {
	root, p := preparedProjectFixture(t)
	// Both trials execute; a candidate that does nothing is a business failure.
	bad := mockWitness(&Witness{Kind: "TaskWitness", TaskID: "close-issue", SyntheticOnly: true, Calls: []Call{}})
	raw, _ := json.Marshal(bad)
	p.suite.trials[1].responses = raw
	r, err := runPreparedProject(context.Background(), root, "delivery", p, defaultProjectOperations())
	if err != nil || r.Decision != "failed" || r.Lifecycle != "published" || r.TrialCounts.Completed != 2 || r.TrialCounts.NotStarted != 0 {
		t.Fatal(r, err)
	}
	if len(r.BaseAssessment.FailedChecks) != 1 || r.BaseAssessment.FailedChecks[0].TrialID != "candidate-01" || r.BaseAssessment.FailedChecks[0].CheckID != "objective" {
		t.Fatal(r.BaseAssessment.FailedChecks)
	}
	a, err := InspectProject(context.Background(), root, "project.json", "delivery")
	if err != nil || a.Verification != "consistent" || a.HistoricalDecision != "failed" {
		t.Fatal(a, err)
	}
}

func TestCoherentWorldSubstitution(t *testing.T) {
	root, p := preparedProjectFixture(t)
	source := t.TempDir()
	for _, name := range []string{"twin.yaml", "agent-state.json", "scenario-close-issue.yaml", "bundle-agent.yaml"} {
		raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "issue-tracker", name))
		if err != nil {
			t.Fatal(err)
		}
		if name == "agent-state.json" {
			raw = []byte(strings.ReplaceAll(string(raw), "Target issue", "Synthetic substituted title"))
		}
		if err := os.WriteFile(filepath.Join(source, name), raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	out := filepath.Join(source, "altered.stb")
	if _, err := bundle.Build(filepath.Join(source, "bundle-agent.yaml"), out); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(out)
	for i := range p.suite.trials {
		p.suite.trials[i].bundleBytes = raw
	}
	if _, err := RunSuite(context.Background(), root, "substituted", p.suite); err != nil {
		t.Fatal(err)
	}
	fs, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer fs.Close()
	a, w, _, err := assessProject(context.Background(), diskReadRoot{fs}, "substituted", p)
	if err != nil || a.TaskBinding.Status != "matched" || w.Status != "mismatched" || w.MismatchedTrials != 2 {
		t.Fatal(a, w, err)
	}
}

func TestProjectVerifiedWorldBinding(t *testing.T) {
	root, p := preparedProjectFixture(t)
	if _, err := RunSuite(context.Background(), root, "suite", p.suite); err != nil {
		t.Fatal(err)
	}
	name := filepath.Join(root, "suite", "candidate-01", "terminal.json")
	original, _ := os.ReadFile(name)
	for _, mode := range []string{"runtime", "corrupt"} {
		var e AgentEvidence
		if err := json.Unmarshal(original, &e); err != nil {
			t.Fatal(err)
		}
		if mode == "runtime" {
			e.Episode.Definition.RuntimeVersion = "unsupported"
			writeTestJSON(t, root, "suite/candidate-01/terminal.json", e)
		} else {
			os.WriteFile(name, []byte("{}"), 0600)
		}
		fs, _ := os.OpenRoot(root)
		_, w, _, err := assessProject(context.Background(), diskReadRoot{fs}, "suite", p)
		fs.Close()
		if err != nil {
			t.Fatal(err)
		}
		if w.PlannedTrials != 2 || w.UnavailableTrials < 1 || w.Status == "matched" {
			t.Fatal(w)
		}
	}
	os.WriteFile(name, original, 0600)
	fs, _ := os.OpenRoot(root)
	_, w, _, err := assessProject(context.Background(), diskReadRoot{fs}, "suite", p)
	fs.Close()
	if err != nil || w.Status != "matched" {
		t.Fatal(w, err)
	}
}

func TestProjectInspectionScope(t *testing.T) {
	root, _ := projectFixture(t)
	if _, err := RunProject(context.Background(), root, "project.json", "delivery"); err != nil {
		t.Fatal(err)
	}
	before := suiteSnapshot(t, root)
	a, err := InspectProject(context.Background(), root, "project.json", "delivery")
	if err != nil || a.Verification != "consistent" || a.QualificationVerification != "recorded_only" {
		t.Fatal(a, err)
	}
	if !reflect.DeepEqual(before, suiteSnapshot(t, root)) {
		t.Fatal("inspect wrote files")
	}
	for _, mode := range []string{"report-count", "quality-row", "unknown", "missing-terminal", "pending"} {
		t.Run(mode, func(t *testing.T) {
			var restore func()
			switch mode {
			case "report-count":
				name := "delivery/project-report.json"
				raw, _ := os.ReadFile(filepath.Join(root, name))
				var r ProjectReport
				json.Unmarshal(raw, &r)
				r.TrialCounts.Planned--
				writeTestJSON(t, root, name, r)
				restore = func() { os.WriteFile(filepath.Join(root, name), raw, 0600) }
			case "quality-row":
				name := "delivery/quality.json"
				raw, _ := os.ReadFile(filepath.Join(root, name))
				var q TaskQualification
				json.Unmarshal(raw, &q)
				q.CaseReport.Cases[0].TaskID = "wrong"
				writeTestJSON(t, root, name, q)
				restore = func() { os.WriteFile(filepath.Join(root, name), raw, 0600) }
			case "unknown":
				name := filepath.Join(root, "delivery", "unknown.json")
				os.WriteFile(name, []byte("{}"), 0600)
				restore = func() { os.Remove(name) }
			case "missing-terminal":
				name := filepath.Join(root, "delivery", "suite", "candidate-01", "terminal.json")
				raw, _ := os.ReadFile(name)
				os.Remove(name)
				restore = func() { os.WriteFile(name, raw, 0600) }
			case "pending":
				raw, _ := os.ReadFile(filepath.Join(root, "delivery", "project-report.json"))
				name := filepath.Join(root, "delivery", "project-report.pending.json")
				os.WriteFile(name, raw, 0600)
				restore = func() { os.Remove(name) }
			}
			defer restore()
			a, err := InspectProject(context.Background(), root, "project.json", "delivery")
			if err != nil || a.Verification == "consistent" {
				t.Fatal(a, err)
			}
		})
	}
}

func failQualityRunner(ctx context.Context, t *task.Task, b *bundle.Artifact, w *Witness) (*Report, error) {
	copy := *w
	copy.Calls = []Call{}
	return RunWitness(ctx, t, b, &copy)
}

func TestProjectPublicationFaults(t *testing.T) {
	root, p := preparedProjectFixture(t)
	for _, point := range []struct{ op, leaf string }{{"mkdir", "target"}, {"open", "project-claim.json"}, {"write", "quality.json"}, {"short", "quality.json"}, {"sync", "project-report.pending.json"}, {"close", "project-report.pending.json"}, {"link", "project-report.json"}, {"remove", "project-report.pending.json"}} {
		t.Run(point.op, func(t *testing.T) {
			out := "fault-" + point.op
			leaf := point.leaf
			if leaf == "target" {
				leaf = out
			}
			ops := defaultProjectOperations()
			ops.witness = failQualityRunner
			ops.open = func(root string) (evidenceFS, error) {
				fs, err := openRootedEvidenceFS(root)
				if err != nil {
					return nil, err
				}
				return &failingEvidenceFS{evidenceFS: fs, op: point.op, leaf: leaf}, nil
			}
			r, err := runPreparedProject(context.Background(), root, out, p, ops)
			if err == nil || r.Lifecycle == "published" {
				t.Fatal(r, err)
			}
			if point.op == "remove" && r.Lifecycle != "published_with_residue" {
				t.Fatal(r)
			}
			a, err := InspectProject(context.Background(), root, "project.json", out)
			if err != nil || a.Verification == "consistent" {
				t.Fatal(a, err)
			}
		})
	}
}

func TestProjectCancellationStages(t *testing.T) {
	for _, stage := range []string{"read", "quality", "trial", "replay", "publication"} {
		t.Run(stage, func(t *testing.T) {
			root, p := preparedProjectFixture(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			ops := defaultProjectOperations()
			switch stage {
			case "read":
				cancel()
				s := newProjectSource(ctx, root, "", projectInputLimit)
				if _, _, err := prepareProject(s, ".", "project.json"); err == nil || err.Error() != "PROJECT_CANCELED_OR_TIMED_OUT" {
					t.Fatal(err)
				}
				return
			case "quality":
				ops.witness = func(ctx context.Context, t *task.Task, b *bundle.Artifact, w *Witness) (*Report, error) {
					cancel()
					return nil, ctx.Err()
				}
			case "trial":
				ops.suite = func(ctx context.Context, r, o string, p *PreparedSuite, f openEvidenceFS, n int) (*SuiteReport, error) {
					cancel()
					return nil, ctx.Err()
				}
			case "replay":
				if _, err := RunSuite(ctx, root, "suite", p.suite); err != nil {
					t.Fatal(err)
				}
				cancel()
				fs, _ := os.OpenRoot(root)
				defer fs.Close()
				_, _, _, err := assessProject(ctx, diskReadRoot{fs}, "suite", p)
				if err == nil {
					t.Fatal("replay ignored cancel")
				}
				return
			case "publication":
				ops.witness = failQualityRunner
				ops.open = func(root string) (evidenceFS, error) {
					fs, e := openRootedEvidenceFS(root)
					if e != nil {
						return nil, e
					}
					return &failingEvidenceFS{evidenceFS: fs, hook: func(op string) {
						if op == "close:project-report.pending.json" {
							cancel()
						}
					}}, nil
				}
			}
			r, err := runPreparedProject(ctx, root, "delivery", p, ops)
			if err == nil || err.Error() != "PROJECT_CANCELED_OR_TIMED_OUT" || r.Lifecycle != "partial" {
				t.Fatal(r, err)
			}
			if _, err := os.Stat(filepath.Join(root, "delivery", "project-report.json")); !os.IsNotExist(err) {
				t.Fatal("published after cancellation")
			}
		})
	}
}

func TestProjectProcessExit(t *testing.T) {
	if point := os.Getenv("STATETWIN_PROJECT_CRASH_POINT"); point != "" {
		root := os.Getenv("STATETWIN_PROJECT_CRASH_ROOT")
		p, _, err := prepareProject(newProjectSource(context.Background(), root, "delivery", projectInputLimit), ".", "project.json")
		if err != nil {
			os.Exit(78)
		}
		ops := defaultProjectOperations()
		ops.witness = failQualityRunner
		ops.open = func(root string) (evidenceFS, error) {
			fs, err := openRootedEvidenceFS(root)
			if err != nil {
				return nil, err
			}
			return &failingEvidenceFS{evidenceFS: fs, hook: func(op string) {
				if op == point {
					os.Exit(79)
				}
			}}, nil
		}
		runPreparedProject(context.Background(), root, "delivery", p, ops)
		os.Exit(80)
	}
	for _, point := range []string{"close:project-claim.json", "link:project-report.json"} {
		t.Run(point, func(t *testing.T) {
			root, _ := projectFixture(t)
			cmd := exec.Command(os.Args[0], "-test.run=^TestProjectProcessExit$")
			cmd.Env = append(os.Environ(), "STATETWIN_PROJECT_CRASH_POINT="+point, "STATETWIN_PROJECT_CRASH_ROOT="+root)
			err := cmd.Run()
			var exit *exec.ExitError
			if !errors.As(err, &exit) || exit.ExitCode() != 79 {
				t.Fatal(err)
			}
			a, err := InspectProject(context.Background(), root, "project.json", "delivery")
			if err != nil || a.Verification == "consistent" || a.Lifecycle != "partial" {
				t.Fatal(a, err)
			}
		})
	}
}

func TestMutationIsolationAndAdmission(t *testing.T) {
	root, p := preparedProjectFixture(t)
	in := p.cases.inputs[0]
	r, err := RunWitness(context.Background(), in.task, in.bundle, in.witness)
	if err != nil {
		t.Fatal(err)
	}
	before, _ := json.Marshal(r)
	altered, err := mutatedGrading(context.Background(), in.task, r, []ViewMutation{{"repository", "octo/demo", "defaultBranch", "synthetic-branch"}})
	after, _ := json.Marshal(r)
	if err != nil || altered.Evaluation.Outcome != "policy_violation" || string(before) != string(after) {
		t.Fatal("mutation escaped private view", err)
	}
	for _, variant := range []string{"positive", "v1", "unknown-field", "too-many", "missing-target"} {
		t.Run(variant, func(t *testing.T) {
			var m CaseManifest
			raw, _ := os.ReadFile(filepath.Join(root, "cases.json"))
			json.Unmarshal(raw, &m)
			switch variant {
			case "positive":
				m.Cases[3].Role = "positive"
			case "v1":
				m.Format = "statetwin.dev/task-cases/v1alpha1"
				m.Profile = "synthetic-witness-cases-v1"
			case "unknown-field":
				m.Cases[3].Mutations[0].Field = "not-a-field"
			case "too-many":
				for len(m.Cases[3].Mutations) < 5 {
					m.Cases[3].Mutations = append(m.Cases[3].Mutations, m.Cases[3].Mutations[0])
				}
			case "missing-target":
				m.Cases[3].Mutations[0].Key = "missing"
			}
			writeTestJSON(t, root, "mutation-cases.json", m)
			prep, err := prepareCases(context.Background(), root, "mutation-cases.json")
			if variant == "missing-target" {
				if err != nil {
					t.Fatal(err)
				}
				r, err := runCases(context.Background(), prep, RunWitness)
				if err == nil || r.Failed != 1 {
					t.Fatal(r, err)
				}
			} else if err == nil {
				t.Fatal("invalid mutation admitted")
			}
		})
	}
}

func TestRealUnscorableCaseDoesNotCoverGoal(t *testing.T) {
	root, p := preparedProjectFixture(t)
	ta := p.cases.tasks["close-issue"]
	ta.Oracle[0].Expr = "answer.missing"
	writeTestJSON(t, root, "task-close-issue.json", ta)
	m := p.cases.manifest
	m.Cases = []TaskCase{{CaseID: "unscorable", TaskID: ta.ID, Role: "unscorable-negative", Witness: "positive.json", Expected: CaseExpected{Outcome: "not_evaluated", FailedChecks: []string{"objective"}}}}
	writeTestJSON(t, root, "unscorable.json", m)
	prepared, err := prepareCases(context.Background(), root, "unscorable.json")
	if err != nil {
		t.Fatal(err)
	}
	r, err := runCases(context.Background(), prepared, RunWitness)
	if err != nil || r.Matched != 1 || len(r.Cases[0].ErrorCheckIDs) != 1 {
		t.Fatal(r, err)
	}
	q := qualifyCases(prepared, r)
	if q.Decision != "not_qualified" || q.Tasks[0].Checks[0].Status != "uncovered" {
		t.Fatal(q)
	}
	p.cases = prepared
	if !recordedQualityValid(p, q) {
		t.Fatal("honest not_evaluated record rejected")
	}
}
