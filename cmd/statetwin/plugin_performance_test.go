package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/augety121/mcp-state-twin/internal/plugin"
)

func TestPluginPerformanceCLIAndSchema(t *testing.T) {
	root := t.TempDir()
	run := plugin.PerformanceRun{Format: plugin.PerformanceFormat, RunID: "synthetic-cli", Source: "mock-contract", ProcessScope: "full-go-stdio-contract-v1", BaselineScope: "preloaded-run-witness-v1", Sessions: 100, GoroutinesBefore: 2, GoroutinesAfter: 3,
		Environment: plugin.PerformanceEnvironment{OS: "test", Arch: "test", CPU: "synthetic", Go: "go1.26", BuildLabel: "test", RuntimeVersion: "dev", RuntimeRevision: "unknown", GOMAXPROCS: 1},
		Workload:    plugin.PerformanceWorkload{PackID: "pack", PackRevision: "v1", EntryID: "read", TaskID: "read", Profile: "local-stdio-tools-v1", ProtocolProfile: "2025-11-25"}}
	for i := 0; i < 30; i++ {
		run.Pairs = append(run.Pairs, plugin.PerformancePair{ID: i + 1, ProcessSeconds: 1.5, BaselineSeconds: 1, Metrics: plugin.ContractMetrics{StartupSeconds: .1, TotalSeconds: 1.4, RSSStatus: "unavailable"}})
	}
	args := []string{"assess-performance", "--root", root, "--performance-report", "measurements.json"}
	for _, decision := range []string{"passed", "failed", "unavailable"} {
		t.Run(decision, func(t *testing.T) {
			if decision == "failed" {
				for i := range run.Pairs {
					run.Pairs[i].ProcessSeconds = 10
				}
			}
			raw, err := json.Marshal(run)
			if err != nil {
				t.Fatal(err)
			}
			if decision == "unavailable" {
				raw = []byte(`{"format":"statetwin.dev/plugin-performance/v1alpha1"}`)
			}
			if err = os.WriteFile(filepath.Join(root, "measurements.json"), raw, 0600); err != nil {
				t.Fatal(err)
			}
			var out bytes.Buffer
			err = runPluginTo(context.Background(), args, &out)
			if (err == nil) != (decision == "passed") {
				t.Fatal(err, out.String())
			}
			var result plugin.PerformanceAssessment
			if json.Unmarshal(out.Bytes(), &result) != nil || result.Decision != decision {
				t.Fatal(out.String())
			}
		})
	}
	for _, kind := range []string{"performance", "performance-assessment"} {
		var out bytes.Buffer
		if err := runPluginTo(context.Background(), []string{"schema", "--type", kind}, &out); err != nil || !strings.Contains(out.String(), `"additionalProperties": false`) {
			t.Fatal(err, out.String())
		}
	}
	for _, bad := range [][]string{
		{"assess-performance"},
		append(append([]string{}, args...), "--session-plan", "plan.json"),
		{"schema", "--type", "performance", "--performance-report", "measurements.json"},
		{"assess-performance", "--root", root, "--performance-report", "../escape.json"},
	} {
		var out bytes.Buffer
		if err := runPluginTo(context.Background(), bad, &out); err == nil {
			t.Fatal(bad)
		}
	}
}
