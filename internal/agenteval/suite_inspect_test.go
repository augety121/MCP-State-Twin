package agenteval

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/augety121/mcp-state-twin/internal/agenthost"
)

func auditFixture(t *testing.T) (string, *SuiteReport) {
	t.Helper()
	root, p := suiteFixture(t, "close-issue")
	prepared, err := PrepareSuite(context.Background(), root, suiteBytes(t, p))
	if err != nil {
		t.Fatal(err)
	}
	r, err := RunSuite(context.Background(), root, "suite", prepared)
	if err != nil {
		t.Fatal(err)
	}
	return root, r
}

// Compare owned fixture bytes directly; do not introduce file hash manifests.
func suiteSnapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	r := map[string]string{}
	err := filepath.WalkDir(root, func(name string, e os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, name)
		if err != nil {
			return err
		}
		if e.IsDir() {
			r[rel+"/"] = ""
			return nil
		}
		raw, err := os.ReadFile(name)
		if err != nil {
			return err
		}
		r[rel] = string(raw)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestSuiteAuditReplayAndTamper(t *testing.T) {
	source, original := auditFixture(t)
	files := suiteSnapshot(t, filepath.Join(source, "suite"))
	for _, tc := range []struct {
		name, state, verification string
		mutate                    func(*testing.T, string)
	}{
		{"complete", "published", "matched", nil},
		{"pending residue", "published_with_residue", "matched", func(t *testing.T, root string) { writeTestJSON(t, root, "suite/report.pending.json", original) }},
		{"pending only", "incomplete_or_running", "unverified", func(t *testing.T, root string) { renameAudit(t, root, "report.json", "report.pending.json") }},
		{"no report", "incomplete_or_running", "absent", func(t *testing.T, root string) { removeAudit(t, root, "report.json") }},
		{"forged decision", "invalid", "mismatch", func(t *testing.T, root string) {
			mutateAuditReport(t, root, func(r *SuiteReport) { r.Comparison.Decision = "regression" })
		}},
		{"forged count", "invalid", "mismatch", func(t *testing.T, root string) {
			mutateAuditReport(t, root, func(r *SuiteReport) { r.Comparison.Counts.ValidlyEvaluated = 0 })
		}},
		{"forged outcome", "invalid", "", func(t *testing.T, root string) {
			mutateAuditReport(t, root, func(r *SuiteReport) { r.Trials[0].Outcome = "task_failed" })
		}},
		{"deleted evidence", "invalid", "", func(t *testing.T, root string) { removeAudit(t, root, "candidate-01/terminal.json") }},
		{"broken evidence", "invalid", "", func(t *testing.T, root string) { writeAuditRaw(t, root, "candidate-01/terminal.json", `{"broken":`) }},
		{"unknown root member", "invalid", "", func(t *testing.T, root string) { writeAuditRaw(t, root, "private-unexpected.txt", "private sentinel") }},
		{"unknown trial member", "invalid", "", func(t *testing.T, root string) { writeAuditRaw(t, root, "baseline-01/private.txt", "private sentinel") }},
		{"publication conflict", "invalid", "", func(t *testing.T, root string) {
			writeTestJSON(t, root, "suite/report.pending.json", original)
			mutateAuditReport(t, root, func(r *SuiteReport) { r.Trials[0].Outcome = "task_failed" })
		}},
		{"claim limits", "invalid", "", func(t *testing.T, root string) {
			var c SuitePreflight
			readAuditJSON(t, root, "claim.json", &c)
			c.DeadlineSeconds++
			writeTestJSON(t, root, "suite/claim.json", c)
		}},
		{"plan substitution", "invalid", "", func(t *testing.T, root string) {
			var p ComparePlan
			readAuditJSON(t, root, "plan.json", &p)
			p.Pairs[0].Repeat++
			writeTestJSON(t, root, "suite/plan.json", p)
		}},
		{"trial claim identity", "invalid", "", func(t *testing.T, root string) {
			var c RunConfig
			readAuditJSON(t, root, "baseline-01/claim.json", &c)
			c.Model = "mock-other"
			writeTestJSON(t, root, "suite/baseline-01/claim.json", c)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			for name, raw := range files {
				if strings.HasSuffix(name, "/") {
					continue
				}
				file := filepath.Join(root, "suite", name)
				if err := os.MkdirAll(filepath.Dir(file), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(file, []byte(raw), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if tc.mutate != nil {
				tc.mutate(t, root)
			}
			before := suiteSnapshot(t, root)
			r, err := InspectSuite(context.Background(), root, "suite")
			if err != nil || r.State != tc.state || (tc.verification != "" && r.ReportVerification != tc.verification) {
				t.Fatalf("%+v %v", r, err)
			}
			if r.RegressionGatePassed != (tc.name == "complete") || r.UpgradeAllowed || r.ResumeAllowed || r.SnapshotAtomic {
				t.Fatalf("unsafe gate: %+v", r)
			}
			if tc.name == "complete" && (r.PlannedTrials != 2 || r.CompleteTrials != 2 || !same(r.Comparison, original.Comparison)) {
				t.Fatal("replay disagreement")
			}
			if !reflect.DeepEqual(before, suiteSnapshot(t, root)) {
				t.Fatal("audit modified files")
			}
			raw, _ := json.Marshal(r)
			for _, forbidden := range []string{root, "private sentinel", "private-unexpected", "private.txt", "objective"} {
				if strings.Contains(string(raw), forbidden) || strings.Contains(r.Markdown(), forbidden) {
					t.Fatal("audit exposed content", forbidden)
				}
			}
		})
	}
}

func TestSuiteAuditMetadataAdmission(t *testing.T) {
	root, _ := auditFixture(t)
	raw, err := os.ReadFile(filepath.Join(root, "suite", "report.json"))
	if err != nil {
		t.Fatal(err)
	}
	var p ComparePlan
	readAuditJSON(t, root, "plan.json", &p)
	for _, mutate := range []func(map[string]any){
		func(m map[string]any) { delete(m, "upgradeAllowed") },
		func(m map[string]any) { m["upgradeAllowed"] = nil },
		func(m map[string]any) { m["upgradeAllowed"] = true },
		func(m map[string]any) { m["plannedTrials"] = 2.5 },
		func(m map[string]any) { m["plannedTrials"] = nil },
		func(m map[string]any) { m["unknown"] = true },
		func(m map[string]any) { m["comparison"] = nil },
		func(m map[string]any) { m["comparisonStatus"] = "not_attempted" },
		func(m map[string]any) { m["failureCode"] = "SUITE_COMPARISON_FAILED" },
		func(m map[string]any) { m["executionStatus"] = "secret-private-content" },
		func(m map[string]any) { m["trials"].([]any)[0].(map[string]any)["trialId"] = "candidate-01" },
		func(m map[string]any) { m["trials"].([]any)[0].(map[string]any)["state"] = "not_started" },
	} {
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil {
			t.Fatal(err)
		}
		mutate(m)
		bad, _ := json.Marshal(m)
		if got, err := decodeSuiteReport(bad, &p); got != nil || err == nil || err.Error() != "SUITE_METADATA_INVALID" {
			t.Fatal("bad metadata admitted", err)
		}
	}
	for _, bad := range [][]byte{append(raw, []byte(" {}")...), []byte(strings.Replace(string(raw), `"upgradeAllowed": false`, `"upgradeAllowed": false, "upgradeAllowed": false`, 1)), []byte(strings.Replace(string(raw), `"plannedTrials": 2`, `"plannedTrials": 2.0`, 1))} {
		if _, err := decodeSuiteReport(bad, &p); err == nil {
			t.Fatal("invalid JSON admitted")
		}
	}
	var target struct {
		Value string `json:"value"`
	}
	private := `{"value":"\u0073\u006b-` + strings.Repeat("x", 30) + `"}`
	if err := decodeSuiteMetadata([]byte(private), MaxSuitePlanBytes, &target); err == nil || err.Error() != "SUITE_METADATA_INVALID" {
		t.Fatal("escaped sensitive content admitted", err)
	}
}

func TestSuiteAuditRegressionAndInterruptedRuns(t *testing.T) {
	for _, mode := range []string{"regression", "partial", "canceled"} {
		t.Run(mode, func(t *testing.T) {
			root, p := suiteFixture(t, "close-issue", "read-issue")
			if mode != "canceled" {
				response := mockResponse(mockFinal(map[string]any{}))
				if mode == "partial" {
					response = mockResponse(map[string]any{"type": "unsupported-item"})
				}
				writeTestJSON(t, root, "bad.json", &agenthost.MockScript{Kind: "MockResponses", SyntheticOnly: true, Responses: []json.RawMessage{response}})
				p.Pairs[0].CandidateResponses = "bad.json"
			}
			prepared, err := PrepareSuite(context.Background(), root, suiteBytes(t, p))
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			open := func(root string) (evidenceFS, error) {
				fs, err := openRootedEvidenceFS(root)
				if err != nil {
					return nil, err
				}
				return &failingEvidenceFS{evidenceFS: fs, hook: func(op string) {
					if mode == "canceled" && op == "link:terminal.json" {
						cancel()
					}
				}}, nil
			}
			_, runErr := runSuite(ctx, root, "suite", prepared, open, maxSuiteWriteBytes)
			if (mode == "regression") != (runErr == nil) {
				t.Fatal("fixture unexpected result", runErr)
			}
			before := suiteSnapshot(t, root)
			r, err := InspectSuite(context.Background(), root, "suite")
			if err != nil || r.State == "invalid" || r.RegressionGatePassed {
				t.Fatal(r, err)
			}
			if mode == "regression" {
				if r.ReportVerification != "matched" || r.Comparison.Decision != "regression" || r.CompleteTrials != 4 {
					t.Fatal(r)
				}
			} else if r.ReportVerification != "unverified" || r.State != "published_unverified" || r.Comparison.Counts.Planned != 4 || r.Comparison.Counts.NotStarted == 0 {
				t.Fatal(r)
			}
			if !reflect.DeepEqual(before, suiteSnapshot(t, root)) {
				t.Fatal("changed interrupted evidence")
			}
		})
	}
}

func TestSuiteAuditIncompleteBoundsAndCancellation(t *testing.T) {
	root, _ := auditFixture(t)
	if _, err := inspectSuite(context.Background(), root, "suite", 1); err == nil || err.Error() != "SUITE_INSPECT_RESOURCE_LIMIT" {
		t.Fatal("size budget", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if r, err := InspectSuite(ctx, root, "suite"); r != nil || !errors.Is(err, context.Canceled) {
		t.Fatal("cancellation", err)
	}
	if _, err := InspectSuite(context.Background(), root, "../outside"); err == nil {
		t.Fatal("unsafe path admitted")
	}
	for _, out := range []string{"missing", "empty"} {
		if out == "empty" {
			if err := os.Mkdir(filepath.Join(root, out), 0700); err != nil {
				t.Fatal(err)
			}
		}
		r, err := InspectSuite(context.Background(), root, out)
		if err != nil || r.RegressionGatePassed || r.State == "invalid" {
			t.Fatal(r, err)
		}
	}
	for _, name := range []string{"report.json", "baseline-01/claim.json", "baseline-01/terminal.json", "candidate-01/claim.json", "candidate-01/terminal.json"} {
		removeAudit(t, root, name)
	}
	for _, name := range []string{"baseline-01", "candidate-01"} {
		removeAudit(t, root, name)
	}
	r, err := InspectSuite(context.Background(), root, "suite")
	if err != nil || r.State != "incomplete_or_running" || r.Comparison.Counts.NotStarted != 2 {
		t.Fatal(r, err)
	}
	removeAudit(t, root, "plan.json")
	r, err = InspectSuite(context.Background(), root, "suite")
	if err != nil || r.State != "incomplete_or_running" || r.Comparison != nil || r.PlannedTrials != 2 {
		t.Fatal(r, err)
	}
}

func TestSuiteAuditSymlinkAndOversizedMetadata(t *testing.T) {
	root, _ := auditFixture(t)
	if err := os.Symlink(filepath.Join(root, "suite"), filepath.Join(root, "linked")); err == nil {
		if _, err := InspectSuite(context.Background(), root, "linked"); err == nil {
			t.Fatal("symlink directory admitted")
		}
	} else {
		t.Log("symlink creation unavailable; other cases still run")
	}
	f, err := os.OpenFile(filepath.Join(root, "suite", "report.json"), os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(maxSuiteReportBytes + 1); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	r, err := InspectSuite(context.Background(), root, "suite")
	if err != nil || r.State != "invalid" {
		t.Fatal(r, err)
	}
}

func readAuditJSON(t *testing.T, root, name string, v any) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, "suite", filepath.FromSlash(name)))
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(raw, v); err != nil {
		t.Fatal(err)
	}
}
func mutateAuditReport(t *testing.T, root string, mutate func(*SuiteReport)) {
	t.Helper()
	var r SuiteReport
	readAuditJSON(t, root, "report.json", &r)
	mutate(&r)
	writeTestJSON(t, root, "suite/report.json", r)
}
func writeAuditRaw(t *testing.T, root, name, raw string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, "suite", filepath.FromSlash(name)), []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
}
func removeAudit(t *testing.T, root, name string) {
	t.Helper()
	if err := os.Remove(filepath.Join(root, "suite", filepath.FromSlash(name))); err != nil {
		t.Fatal(err)
	}
}
func renameAudit(t *testing.T, root, from, to string) {
	t.Helper()
	if err := os.Rename(filepath.Join(root, "suite", from), filepath.Join(root, "suite", to)); err != nil {
		t.Fatal(err)
	}
}
