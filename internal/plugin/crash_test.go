package plugin

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/augety121/mcp-state-twin/internal/agenteval"
	"github.com/augety121/mcp-state-twin/internal/baselinepack"
	"github.com/augety121/mcp-state-twin/internal/testfixture"
)

func TestPluginChildHelper(t *testing.T) {
	if os.Getenv("STATETWIN_PLUGIN_CONTROL") == "" {
		return
	}
	for i, arg := range os.Args {
		if arg == "--" {
			root, name := os.Args[i+4], os.Args[i+6]
			p, e := Prepare(context.Background(), root, name)
			if e != nil {
				os.Exit(2)
			}
			_, e = Serve(context.Background(), root, p, os.Stdin, os.Stdout, os.Getenv("STATETWIN_PLUGIN_CONTROL"), os.Getenv("STATETWIN_PLUGIN_CONTROL_TOKEN"))
			if e != nil {
				os.Exit(2)
			}
			os.Exit(0)
		}
	}
	os.Exit(3)
}
func TestPluginTenStagedCrashesReapOwnedChildren(t *testing.T) {
	root := testfixture.Baseline(t)
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
	raw, _ = os.ReadFile(filepath.Join(root, "issue-tracker/agent-witnesses/read-issue.json"))
	w, e := agenteval.DecodeWitness(raw)
	if e != nil {
		t.Fatal(e)
	}
	for _, stage := range []string{"started", "initialized", "ready", "called", "sealed"} {
		for repeat := 0; repeat < 2; repeat++ {
			t.Run(stage+string(rune('a'+repeat)), func(t *testing.T) {
				p := Plan{Format: PlanFormat, ID: "crash", Pack: "baseline-pack.json", Profile: "plugin-profile.json", EntryID: "read-issue", Host: Host{Name: "mcp-client", Version: "go-sdk-1.8.0", Model: "mock/statetwin", FreshSession: true, ToolsOnly: true}, Output: stage + string(rune('a'+repeat)), Mode: "offline-contract", DeadlineSeconds: 15}
				raw, _ := json.Marshal(p)
				os.WriteFile(filepath.Join(root, "crash.json"), raw, 0600)
				var child *exec.Cmd
				ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
				defer cancel()
				ctx = context.WithValue(ctx, contractStageKey{}, func(s string, c *exec.Cmd) {
					if s == stage {
						child = c
						if e := c.Process.Kill(); e != nil {
							t.Error(e)
						}
					}
				})
				_, e := RunContract(ctx, []string{binary, "-test.run=^TestPluginChildHelper$", "--"}, root, "crash.json", w)
				if e == nil || child == nil || child.ProcessState == nil {
					t.Fatal("missing crash/reap", e, child)
				}
				report, err := Inspect(context.Background(), root, "crash.json")
				if stage != "sealed" && (err == nil || report != nil && report.Decision != "invalid") {
					t.Fatal("crash became success", report, err)
				}
			})
		}
	}
}
