package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
	"time"

	"github.com/augety121/mcp-state-twin/internal/agenteval"
	"github.com/augety121/mcp-state-twin/internal/baselinepack"
	"github.com/augety121/mcp-state-twin/internal/bundle"
	"github.com/augety121/mcp-state-twin/internal/plugin"
	"github.com/augety121/mcp-state-twin/internal/testfixture"
)

// Opt-in release qualification: always record measurements; thresholds are
// assessed separately rather than hiding a regression with a flaky CI skip.
func TestPluginReleaseSoakAndPerformance(t *testing.T) {
	if os.Getenv("STATETWIN_PLUGIN_SOAK") != "1" {
		t.Skip("release qualification: set STATETWIN_PLUGIN_SOAK=1")
	}
	root := testfixture.Baseline(t)
	ctx := context.Background()
	binary, e := os.Executable()
	if e != nil {
		t.Fatal(e)
	}
	raw, _ := os.ReadFile(filepath.Join(root, "baseline-pack.json"))
	var pack baselinepack.Pack
	json.Unmarshal(raw, &pack)
	pack.Entries = pack.Entries[:1]
	raw, _ = json.Marshal(pack)
	os.WriteFile(filepath.Join(root, "baseline-pack.json"), raw, 0600)
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
	process := []float64{}
	inprocess := []float64{}
	measurements := []plugin.ContractMetrics{}
	for i := 0; i < 100; i++ {
		plan := plugin.Plan{Format: plugin.PlanFormat, ID: "soak", Pack: "baseline-pack.json", Profile: "plugin-profile.json", EntryID: entry.Entry.ID, Host: plugin.Host{Name: "mcp-client", Version: "go-sdk-1.8.0", Model: "mock/statetwin", FreshSession: true, ToolsOnly: true}, Output: "soak-" + time.Now().Format("150405.000000000"), Mode: "offline-contract", DeadlineSeconds: 60}
		raw, _ = json.Marshal(plan)
		os.WriteFile(filepath.Join(root, "soak.json"), raw, 0600)
		started := time.Now()
		r, measurement, e := plugin.RunContractMeasured(ctx, []string{binary, "-test.run=^TestPluginProcessHelper$", "--"}, root, "soak.json", w)
		elapsed := time.Since(started).Seconds()
		if e != nil || r.Decision != "passed" {
			t.Fatal(i, r, e)
		}
		if i < 30 {
			measurements = append(measurements, measurement)
			process = append(process, elapsed)
			started = time.Now()
			r, e := agenteval.RunWitness(ctx, entry.Task, world, w)
			if e != nil || r.Evaluation.Outcome != "success" {
				t.Fatal(r, e)
			}
			inprocess = append(inprocess, time.Since(started).Seconds())
		}
	}
	runtime.GC()
	after := runtime.NumGoroutine()
	if after > before+4 {
		t.Fatal("goroutine growth", before, after)
	}
	sort.Float64s(process)
	sort.Float64s(inprocess)
	result := map[string]any{"format": "statetwin.dev/plugin-performance/v1alpha1", "os": runtime.GOOS, "arch": runtime.GOARCH, "go": runtime.Version(), "sessions": 100, "samples": 30, "processScope": "preflight + child stdio + scoring + verification + child wait", "inprocessScope": "existing RunWitness with preloaded Task and world", "processP50Seconds": process[14], "processP95Seconds": process[28], "processMaxSeconds": process[29], "inprocessP50Seconds": inprocess[14], "inprocessP95Seconds": inprocess[28], "inprocessMaxSeconds": inprocess[29], "goroutinesBefore": before, "goroutinesAfter": after, "rssStatus": "unavailable", "startupStatus": "not-separately-measured", "ratioP95": process[28] / inprocess[28], "overheadP95Seconds": process[28] - inprocess[28]}
	raw, _ = json.MarshalIndent(result, "", "  ")
	result["measurements"] = measurements
	starts := []float64{}
	peak := int64(0)
	for _, m := range measurements {
		starts = append(starts, m.StartupSeconds)
		if m.PeakRSSBytes > peak {
			peak = m.PeakRSSBytes
		}
	}
	sort.Float64s(starts)
	result["startupP95Seconds"] = starts[28]
	result["startupStatus"] = "measured"
	result["peakChildRssBytes"] = peak
	if peak > 0 {
		result["rssStatus"] = "measured"
	}
	raw, _ = json.MarshalIndent(result, "", "  ")
	t.Log(string(raw))
	if destination := os.Getenv("STATETWIN_PLUGIN_PERFORMANCE_OUT"); destination != "" {
		if e = os.WriteFile(destination, raw, 0600); e != nil {
			t.Fatal(e)
		}
	}
}
