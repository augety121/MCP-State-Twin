package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/augety121/mcp-state-twin/internal/agenteval"
	"github.com/augety121/mcp-state-twin/internal/baseline"
	"github.com/augety121/mcp-state-twin/internal/baselinepack"
	"github.com/augety121/mcp-state-twin/internal/plugin"
	"github.com/augety121/mcp-state-twin/internal/testfixture"
)

func TestPluginProcessHelper(t *testing.T) {
	if os.Getenv("STATETWIN_PLUGIN_CONTROL") == "" {
		return
	}
	for i, arg := range os.Args {
		if arg == "--" {
			if err := runPlugin(context.Background(), os.Args[i+2:]); err != nil {
				os.Exit(2)
			}
			os.Exit(0)
		}
	}
	os.Exit(3)
}

func TestPluginIndependentProcessProfilesAll24(t *testing.T) {
	root := testfixture.Baseline(t)
	ctx := context.Background()
	binary, e := os.Executable()
	if e != nil {
		t.Fatal(e)
	}
	for _, protocol := range []string{"2025-11-25", "2026-07-28"} {
		raw, _ := os.ReadFile(filepath.Join(root, "plugin-profile.json"))
		var profile baselinepack.Profile
		json.Unmarshal(raw, &profile)
		profile.ProtocolProfile = protocol
		raw, _ = json.Marshal(profile)
		os.WriteFile(filepath.Join(root, "plugin-profile.json"), raw, 0600)
		pack, e := baselinepack.Prepare(ctx, root, "baseline-pack.json", "plugin-profile.json")
		if e != nil {
			t.Fatal(e)
		}
		for _, entry := range pack.Entries {
			t.Run(protocol+"/"+entry.Entry.ID, func(t *testing.T) {
				var cases agenteval.CaseManifest
				raw := pack.Reader.Inputs[filepath.ToSlash(filepath.Join(entry.Entry.Root, entry.Entry.Cases))].Raw
				if baselinepack.Decode(raw, 1<<20, &cases) != nil {
					t.Fatal("cases")
				}
				var witness *agenteval.Witness
				for _, c := range cases.Cases {
					if c.TaskID == entry.Task.ID && c.Role == "positive" {
						witness, e = agenteval.DecodeWitness(pack.Reader.Inputs[filepath.ToSlash(filepath.Join(entry.Entry.Root, c.Witness))].Raw)
						if e != nil {
							t.Fatal(e)
						}
						break
					}
				}
				plan := plugin.Plan{Format: plugin.PlanFormat, ID: entry.Entry.ID, Pack: "baseline-pack.json", Profile: "plugin-profile.json", EntryID: entry.Entry.ID, Host: plugin.Host{Name: "mcp-client", Version: "go-sdk-1.8.0", Model: "mock/statetwin", FreshSession: true, ToolsOnly: true}, Output: entry.Entry.ID + "-" + protocol, Mode: "offline-contract", DeadlineSeconds: entry.Task.Budgets.EpisodeSeconds}
				raw, _ = json.Marshal(plan)
				name := entry.Entry.ID + ".plan.json"
				if e = os.WriteFile(filepath.Join(root, name), raw, 0600); e != nil {
					t.Fatal(e)
				}
				r, e := plugin.RunContract(ctx, []string{binary, "-test.run=^TestPluginProcessHelper$", "--"}, root, name, witness)
				if e != nil || r == nil || r.Decision != "passed" {
					t.Fatal(r, e)
				}
			})
		}
	}
}

type shortPluginWriter struct{}

func (shortPluginWriter) Write(p []byte) (int, error) { return len(p) / 2, nil }
func TestPluginCLIProjectionAndOutput(t *testing.T) {
	root := testfixture.Baseline(t)
	args := []string{"projection", "--root", root, "--session-plan", "plugin-session.json"}
	var out bytes.Buffer
	if e := runPluginTo(context.Background(), args, &out); e != nil {
		t.Fatal(e)
	}
	for _, hidden := range []string{"oracle", "expectedOutcome", "authority", "snapshot", "faultPlan"} {
		if bytes.Contains(out.Bytes(), []byte(hidden)) {
			t.Fatal(hidden)
		}
	}
	if e := runPluginTo(context.Background(), args, shortPluginWriter{}); e == nil {
		t.Fatal("short output accepted")
	}
	if e := runPluginTo(context.Background(), []string{"check", "--root", root, "--pack", "baseline-pack.json", "--profile", "plugin-profile.json"}, io.Discard); e != nil {
		t.Fatal(e)
	}
	if _, e := os.Stat(filepath.Join(root, "plugin-read-issue")); !os.IsNotExist(e) {
		t.Fatal("preflight started world")
	}
}

func TestBaselineRealShardAndFixedDenominators(t *testing.T) {
	root := testfixture.Baseline(t)
	ctx := context.Background()
	binary, e := os.Executable()
	if e != nil {
		t.Fatal(e)
	}
	raw, _ := os.ReadFile(filepath.Join(root, "baseline-pilot.json"))
	var plan baseline.Plan
	json.Unmarshal(raw, &plan)
	plan.Entries = plan.Entries[:2]
	plan.Repeats = 1
	plan.TrialsPerShard = 2
	raw, _ = json.Marshal(plan)
	os.WriteFile(filepath.Join(root, "small.json"), raw, 0600)
	if _, e = baseline.Freeze(ctx, root, "small.json", "experiment"); e != nil {
		t.Fatal(e)
	}
	frozen := filepath.Join(root, "experiment")
	run := func(c context.Context, r, n string, w *agenteval.Witness) (*plugin.Report, error) {
		return plugin.RunContract(c, []string{binary, "-test.run=^TestPluginProcessHelper$", "--"}, r, n, w)
	}
	result, e := baseline.RunShard(ctx, frozen, "small.json", 1, run)
	if e != nil || result.State != "complete" {
		t.Fatal(result, e)
	}
	assessed, e := baseline.Assess(ctx, frozen, "small.json")
	if e != nil || assessed.Decision != "invalid" || len(assessed.Trials) != 4 {
		t.Fatal(assessed, e)
	}
	if _, e = baseline.RunShard(ctx, frozen, "small.json", 1, run); e == nil {
		t.Fatal("resumed claimed shard")
	}
	result, e = baseline.RunShard(ctx, frozen, "small.json", 2, run)
	if e != nil || result.State != "complete" {
		t.Fatal(result, e)
	}
	assessed, e = baseline.Assess(ctx, frozen, "small.json")
	if e != nil || assessed.Decision != "inconclusive" {
		t.Fatal(assessed, e)
	}
	for _, m := range assessed.Metrics {
		if m.Planned != 2 || m.Successes != 2 || len(m.SuccessfulLatencies) != 2 {
			t.Fatal(m)
		}
	}
	// A lost final shard index invalidates the experiment even when individual
	// terminals still replay. Complete trial evidence cannot invent a seal.
	os.Rename(filepath.Join(frozen, "inputs/runs/shard-02/shard-report.json"), filepath.Join(frozen, "preserved-shard-report.json"))
	assessed, e = baseline.Assess(ctx, frozen, "small.json")
	if e != nil || assessed.Decision != "invalid" || assessed.Metrics["candidate"].Planned != 2 {
		t.Fatal(assessed, e)
	}
}
