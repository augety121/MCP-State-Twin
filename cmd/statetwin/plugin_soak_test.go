package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/augety121/mcp-state-twin/internal/agenteval"
	"github.com/augety121/mcp-state-twin/internal/baselinepack"
	"github.com/augety121/mcp-state-twin/internal/bundle"
	"github.com/augety121/mcp-state-twin/internal/plugin"
	"github.com/augety121/mcp-state-twin/internal/server"
	"github.com/augety121/mcp-state-twin/internal/testfixture"
)

// Opt-in release qualification: always record measurements; thresholds are
// assessed separately rather than hiding a regression with a flaky CI skip.
func TestPluginReleaseSoakAndPerformance(t *testing.T) {
	if os.Getenv("STATETWIN_PLUGIN_SOAK") != "1" {
		t.Skip("release qualification: set STATETWIN_PLUGIN_SOAK=1")
	}
	cpu := os.Getenv("STATETWIN_PERFORMANCE_CPU")
	build := os.Getenv("STATETWIN_PERFORMANCE_BUILD")
	if cpu == "" || build == "" {
		t.Fatal("set STATETWIN_PERFORMANCE_CPU and STATETWIN_PERFORMANCE_BUILD to identify the measured environment")
	}
	if performanceRaceEnabled {
		t.Fatal("performance qualification requires a non-race build")
	}
	root := testfixture.Baseline(t)
	ctx := context.Background()
	binary, e := os.Executable()
	if e != nil {
		t.Fatal(e)
	}
	raw, e := os.ReadFile(filepath.Join(root, "baseline-pack.json"))
	if e != nil {
		t.Fatal(e)
	}
	var pack baselinepack.Pack
	if e = json.Unmarshal(raw, &pack); e != nil {
		t.Fatal(e)
	}
	pack.Entries = pack.Entries[:1]
	raw, e = json.Marshal(pack)
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(filepath.Join(root, "baseline-pack.json"), raw, 0600); e != nil {
		t.Fatal(e)
	}
	prepared, e := baselinepack.Prepare(ctx, root, "baseline-pack.json", "plugin-profile.json")
	if e != nil {
		t.Fatal(e)
	}
	entry := prepared.Entries[0]
	raw = prepared.Reader.Inputs["issue-tracker/agent-witnesses/read-issue.json"].Raw
	w, e := agenteval.DecodeWitness(raw)
	if e != nil {
		t.Fatal(e)
	}
	world, e := bundle.OpenBytes(entry.Bundle)
	if e != nil {
		t.Fatal(e)
	}
	before := runtime.NumGoroutine()
	result := plugin.PerformanceRun{Format: plugin.PerformanceFormat, RunID: "soak-" + time.Now().UTC().Format("20060102-150405"), Source: "mock-contract", ProcessScope: "full-go-stdio-contract-v1", BaselineScope: "preloaded-run-witness-v1", Sessions: 100, GoroutinesBefore: before,
		Environment: plugin.PerformanceEnvironment{OS: runtime.GOOS, Arch: runtime.GOARCH, CPU: cpu, Go: runtime.Version(), BuildLabel: build, RuntimeVersion: server.Version, RuntimeRevision: server.Revision, GOMAXPROCS: runtime.GOMAXPROCS(0), Race: performanceRaceEnabled},
		Workload:    plugin.PerformanceWorkload{PackID: prepared.Pack.ID, PackRevision: prepared.Pack.Revision, EntryID: entry.Entry.ID, TaskID: entry.Task.ID, Profile: prepared.Profile.ID, ProtocolProfile: prepared.Profile.ProtocolProfile}, Pairs: []plugin.PerformancePair{}}
	for i := 0; i < 100; i++ {
		plan := plugin.Plan{Format: plugin.PlanFormat, ID: "soak", Pack: "baseline-pack.json", Profile: "plugin-profile.json", EntryID: entry.Entry.ID, Host: plugin.Host{Name: "mcp-client", Version: "go-sdk-1.8.0", Model: "mock/statetwin", FreshSession: true, ToolsOnly: true}, Output: "soak-" + time.Now().Format("150405.000000000"), Mode: "offline-contract", DeadlineSeconds: 60}
		raw, e = json.Marshal(plan)
		if e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(filepath.Join(root, "soak.json"), raw, 0600); e != nil {
			t.Fatal(e)
		}
		started := time.Now()
		r, measurement, e := plugin.RunContractMeasured(ctx, []string{binary, "-test.run=^TestPluginProcessHelper$", "--"}, root, "soak.json", w)
		elapsed := time.Since(started).Seconds()
		if e != nil || r.Decision != "passed" {
			t.Fatal(i, r, e)
		}
		if i < 30 {
			started = time.Now()
			r, e := agenteval.RunWitness(ctx, entry.Task, world, w)
			if e != nil || r.Evaluation.Outcome != "success" {
				t.Fatal(r, e)
			}
			result.Pairs = append(result.Pairs, plugin.PerformancePair{ID: i + 1, ProcessSeconds: elapsed, BaselineSeconds: time.Since(started).Seconds(), Metrics: measurement})
		}
	}
	runtime.GC()
	after := runtime.NumGoroutine()
	if after > before+4 {
		t.Fatal("goroutine growth", before, after)
	}
	result.GoroutinesAfter = after
	raw, e = json.MarshalIndent(result, "", "  ")
	if e != nil {
		t.Fatal(e)
	}
	assessment := plugin.AssessPerformance(raw)
	if assessment.Status != "valid" {
		t.Fatal("invalid measurement record", assessment.Reason)
	}
	summary, e := json.MarshalIndent(assessment, "", "  ")
	if e != nil {
		t.Fatal(e)
	}
	t.Log(string(summary))
	if destination := os.Getenv("STATETWIN_PLUGIN_PERFORMANCE_OUT"); destination != "" {
		if e = os.WriteFile(destination, raw, 0600); e != nil {
			t.Fatal(e)
		}
	}
}
