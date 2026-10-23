package plugin

import (
	"bytes"
	"encoding/json"
	"math"
	"testing"
)

func performanceFixture() PerformanceRun {
	r := PerformanceRun{Format: PerformanceFormat, RunID: "synthetic-test", Source: "mock-contract", ProcessScope: "full-go-stdio-contract-v1", BaselineScope: "preloaded-run-witness-v1", Sessions: 100, GoroutinesBefore: 2, GoroutinesAfter: 3,
		Environment: PerformanceEnvironment{OS: "test", Arch: "test", CPU: "synthetic", Go: "go1.26", BuildLabel: "fixture", RuntimeVersion: "dev", RuntimeRevision: "unknown", GOMAXPROCS: 1},
		Workload:    PerformanceWorkload{PackID: "pack", PackRevision: "v1", EntryID: "read", TaskID: "read", Profile: "local-stdio-tools-v1", ProtocolProfile: "2025-11-25"}}
	for i := 0; i < 30; i++ {
		r.Pairs = append(r.Pairs, PerformancePair{ID: i + 1, ProcessSeconds: 1.5, BaselineSeconds: 1,
			Metrics: ContractMetrics{PreflightSeconds: .1, StartupSeconds: .2, ToolsSeconds: .1, GradingAndSealSeconds: .1, CleanupSeconds: .1, InspectionSeconds: .1, TotalSeconds: 1.4, RSSStatus: "unavailable"}})
	}
	return r
}

func assessFixture(t *testing.T, run PerformanceRun) *PerformanceAssessment {
	t.Helper()
	raw, err := json.Marshal(run)
	if err != nil {
		t.Fatal(err)
	}
	return AssessPerformance(raw)
}

func TestPerformanceOriginalGatesAndNearestRank(t *testing.T) {
	run := performanceFixture()
	r := assessFixture(t, run)
	if r.Status != "valid" || r.Decision != "passed" || r.Summary.RatioOfP95 != 1.5 || len(r.Gates) != 3 || r.SourceTrust != "local-operator-asserted" {
		t.Fatal(r)
	}
	// Nearest rank excludes one slow observation from p95 but keeps it in max.
	run.Pairs[29].ProcessSeconds = 100
	r = assessFixture(t, run)
	if r.Decision != "passed" || r.Summary.ProcessSeconds.Max != 100 || r.Summary.ProcessSeconds.P95 != 1.5 {
		t.Fatal(r)
	}
	run.Pairs[28].ProcessSeconds = 100
	r = assessFixture(t, run)
	if r.Decision != "failed" || r.Summary.ProcessSeconds.P95 != 100 || r.Gates[1].Passed || r.Gates[2].Passed {
		t.Fatal(r)
	}
	for _, tt := range []struct {
		name                       string
		process, baseline, startup float64
		gate                       int
	}{
		{"ratio", 1.500001, 1, .2, 1}, {"overhead", 8.1, 6, .2, 2}, {"startup", 3, 3, 2.01, 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			run := performanceFixture()
			for i := range run.Pairs {
				p := &run.Pairs[i]
				p.ProcessSeconds, p.BaselineSeconds = tt.process, tt.baseline
				p.Metrics.StartupSeconds, p.Metrics.TotalSeconds = tt.startup, tt.process
			}
			r := assessFixture(t, run)
			if r.Status != "valid" || r.Decision != "failed" || r.Gates[tt.gate].Passed {
				t.Fatal(r)
			}
		})
	}
}

func TestPerformancePairingIsPreservedButDoesNotReplaceGate(t *testing.T) {
	run := performanceFixture()
	for i := range run.Pairs {
		p := &run.Pairs[i]
		p.ProcessSeconds, p.BaselineSeconds = 10, 1
		if i >= 15 {
			p.ProcessSeconds, p.BaselineSeconds = 1, 10
		}
		p.Metrics.TotalSeconds = p.ProcessSeconds
	}
	r := assessFixture(t, run)
	if r.Decision != "passed" || r.Summary.RatioOfP95 != 1 || r.Summary.DifferenceOfP95Seconds != 0 || r.Summary.PairedOverheadSeconds.P95 != 9 {
		t.Fatal(r)
	}
}

func TestPerformanceRejectsIncompleteOrContradictoryMeasurements(t *testing.T) {
	cases := map[string]func(*PerformanceRun){
		"too-few":                       func(r *PerformanceRun) { r.Pairs = r.Pairs[:29] },
		"duplicate-pair":                func(r *PerformanceRun) { r.Pairs[1].ID = 1 },
		"shuffled-pair":                 func(r *PerformanceRun) { r.Pairs[0], r.Pairs[1] = r.Pairs[1], r.Pairs[0] },
		"zero-baseline":                 func(r *PerformanceRun) { r.Pairs[0].BaselineSeconds = 0 },
		"negative-baseline":             func(r *PerformanceRun) { r.Pairs[0].BaselineSeconds = -1 },
		"negative-phase":                func(r *PerformanceRun) { r.Pairs[0].Metrics.ToolsSeconds = -1 },
		"unmeasured-startup":            func(r *PerformanceRun) { r.Pairs[0].Metrics.StartupSeconds = 0 },
		"total-exceeds-wall":            func(r *PerformanceRun) { r.Pairs[0].Metrics.TotalSeconds = 2 },
		"phases-exceed-total":           func(r *PerformanceRun) { r.Pairs[0].Metrics.TotalSeconds = .3 },
		"fake-rss-zero":                 func(r *PerformanceRun) { r.Pairs[0].Metrics.RSSStatus = "measured" },
		"fake-rss-unavailable":          func(r *PerformanceRun) { r.Pairs[0].Metrics.PeakRSSBytes = 100 },
		"missing-cpu":                   func(r *PerformanceRun) { r.Environment.CPU = " " },
		"race":                          func(r *PerformanceRun) { r.Environment.Race = true },
		"wrong-scope":                   func(r *PerformanceRun) { r.ProcessScope = "tools-only" },
		"wrong-source":                  func(r *PerformanceRun) { r.Source = "provider-authenticated" },
		"sample-count-exceeds-sessions": func(r *PerformanceRun) { r.Sessions = 29 },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			run := performanceFixture()
			mutate(&run)
			r := assessFixture(t, run)
			if r.Status != "invalid" || r.Decision != "unavailable" || r.Summary != nil || len(r.Gates) != 0 || r.Samples != 0 {
				t.Fatal(r)
			}
		})
	}
}

func TestPerformanceStrictWireAndLegacy(t *testing.T) {
	raw, err := json.Marshal(performanceFixture())
	if err != nil {
		t.Fatal(err)
	}
	for name, bad := range map[string][]byte{
		"null":             bytes.Replace(raw, []byte(`"baselineSeconds":1`), []byte(`"baselineSeconds":null`), 1),
		"missing":          bytes.Replace(raw, []byte(`"startupSeconds":0.2,`), nil, 1),
		"duplicate":        bytes.Replace(raw, []byte(`"baselineSeconds":1`), []byte(`"baselineSeconds":1,"baselineSeconds":1`), 1),
		"case-alias":       bytes.Replace(raw, []byte(`"baselineSeconds":1`), []byte(`"BaselineSeconds":1`), 1),
		"caller-threshold": append(append([]byte{}, raw[:len(raw)-1]...), []byte(`,"ratioLimit":100}`)...),
		"infinity":         bytes.Replace(raw, []byte(`"baselineSeconds":1`), []byte(`"baselineSeconds":1e309`), 1),
		"oversize":         bytes.Repeat([]byte(" "), PerformanceLimit+1),
	} {
		t.Run(name, func(t *testing.T) {
			if bytes.Equal(raw, bad) {
				t.Fatal("mutation not applied")
			}
			r := AssessPerformance(bad)
			if r.Status != "invalid" || r.Summary != nil {
				t.Fatal(r)
			}
		})
	}
	for _, format := range []string{"statetwin.dev/plugin-performance/v1alpha1", "statetwin.dev/inspect-performance/v1alpha1"} {
		r := AssessPerformance([]byte(`{"format":"` + format + `","ratioP95":1}`))
		if r.Status != "incomplete" || r.Decision != "unavailable" || r.Summary != nil {
			t.Fatal(r)
		}
	}
}

func TestPerformanceResourceBoundaries(t *testing.T) {
	run := performanceFixture()
	for len(run.Pairs) < 1000 {
		p := run.Pairs[0]
		p.ID = len(run.Pairs) + 1
		run.Pairs = append(run.Pairs, p)
	}
	run.Sessions = 10000
	if r := assessFixture(t, run); r.Status != "valid" {
		t.Fatal(r)
	}
	run.Sessions++
	if r := assessFixture(t, run); r.Status != "invalid" {
		t.Fatal(r)
	}
	run.Sessions--
	p := run.Pairs[0]
	p.ID = 1001
	run.Pairs = append(run.Pairs, p)
	if r := assessFixture(t, run); r.Status != "invalid" {
		t.Fatal(r)
	}
	run = performanceFixture()
	run.Pairs[0].ProcessSeconds = 3600
	if r := assessFixture(t, run); r.Status != "valid" {
		t.Fatal(r)
	}
	run.Pairs[0].ProcessSeconds = math.Nextafter(3600, math.Inf(1))
	if r := assessFixture(t, run); r.Status != "invalid" {
		t.Fatal(r)
	}
}
