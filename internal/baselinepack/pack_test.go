package baselinepack

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/augety121/mcp-state-twin/internal/testfixture"
)

func TestBaselinePackAdmission(t *testing.T) {
	root := testfixture.Baseline(t)
	p, err := Prepare(context.Background(), root, "baseline-pack.json", "plugin-profile.json")
	if err != nil {
		t.Fatal(err)
	}
	if p.Summary().Entries != 24 || p.Summary().Families != 24 || p.Summary().ExecutionPerformed {
		t.Fatal(p.Summary())
	}
	raw, err := os.ReadFile(filepath.Join(root, "baseline-pack.json"))
	if err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string][]byte{"unknown": []byte(strings.Replace(string(raw), `"id":`, `"unknown":true,"id":`, 1)), "duplicate": []byte(strings.Replace(string(raw), `"id":`, `"id":"x","id":`, 1)), "null": []byte(strings.Replace(string(raw), `"entries": [`, `"entries": null,"other": [`, 1))} {
		t.Run(name, func(t *testing.T) {
			if err := os.WriteFile(filepath.Join(root, "bad.json"), data, 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := Prepare(context.Background(), root, "bad.json", "plugin-profile.json"); err == nil {
				t.Fatal("accepted invalid pack")
			}
		})
	}
	bad := p.Pack
	bad.Entries = append(bad.Entries, bad.Entries[0])
	data, _ := json.Marshal(bad)
	os.WriteFile(filepath.Join(root, "bad.json"), data, 0600)
	if _, err := Prepare(context.Background(), root, "bad.json", "plugin-profile.json"); err == nil {
		t.Fatal("duplicate")
	}
}

func TestCaseAdmissionKeepsRelativeRootAndOracle(t *testing.T) {
	root := testfixture.Baseline(t)
	ctx := context.Background()
	p, err := Prepare(ctx, root, "baseline-pack.json", "plugin-profile.json")
	if err != nil {
		t.Fatal(err)
	}
	first := p.Pack.Entries[0]
	copyFile := func(from, to string) {
		t.Helper()
		raw, err := os.ReadFile(filepath.Join(root, from))
		if err != nil {
			t.Fatal(err)
		}
		if err = os.MkdirAll(filepath.Dir(filepath.Join(root, to)), 0700); err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(root, to), raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	shadow := first
	shadow.ID, shadow.Root = "shadow-read-issue", "."
	shadow.Cases = filepath.ToSlash(filepath.Join(first.Root, first.Cases))
	copyFile(filepath.Join(first.Root, first.Task), first.Task)
	copyFile(filepath.Join(first.Root, first.ReviewedTask), first.ReviewedTask)
	// The same Task text under another root binds a different valid world.
	// Reusing only the canonical case filename would miss that distinction.
	copyFile("package-registry/.statetwin/agent-world.stb", ".statetwin/agent-world.stb")
	copyFile("package-registry/.statetwin/reviewed-agent-world.stb", shadow.ReviewedWorld)
	manifest := p.Pack
	manifest.Entries = []Entry{first, shadow}
	raw, _ := json.Marshal(manifest)
	if err = os.WriteFile(filepath.Join(root, "shadow.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = Prepare(ctx, root, "shadow.json", "plugin-profile.json"); err == nil {
		t.Fatal("case admission reused across different reference roots")
	}
	// A matching actual/reviewed Task is still subject to CEL admission.
	bad := Clone(p.Entries[0].Task)
	bad.Oracle[0].Expr = "missing_function()"
	raw, _ = json.Marshal(bad)
	for _, name := range []string{first.Task, first.ReviewedTask} {
		if err = os.WriteFile(filepath.Join(root, first.Root, name), raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = Prepare(ctx, root, "baseline-pack.json", "plugin-profile.json"); err == nil {
		t.Fatal("oracle compilation was skipped")
	}
}

func TestPluginReferenceIsolationAndFreeze(t *testing.T) {
	root := testfixture.Baseline(t)
	ctx := context.Background()
	p, err := Prepare(ctx, root, "baseline-pack.json", "plugin-profile.json")
	if err != nil {
		t.Fatal(err)
	}
	e := p.Entries[0]
	original := e.Task.Objective
	name := filepath.Join(root, e.Entry.Root, e.Entry.Task)
	if err := os.WriteFile(name, []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	if e.Task.Objective != original {
		t.Fatal("frozen definition mutated")
	}
	if _, err = Prepare(ctx, root, "baseline-pack.json", "plugin-profile.json"); err == nil {
		t.Fatal("stale cache")
	}
	if p.Reader.RejectOutput(e.Entry.Root) == nil {
		t.Fatal("output aliases source subtree")
	}
	if _, err = p.Reader.Read("../escape", 1024); err == nil {
		t.Fatal("path escape")
	}
	root = testfixture.Baseline(t)
	p, err = Prepare(ctx, root, "baseline-pack.json", "plugin-profile.json")
	if err != nil {
		t.Fatal(err)
	}
	a := p.Entries[0].Entry
	review := filepath.Join(root, a.Root, a.ReviewedTask)
	if err = os.Remove(review); err != nil {
		t.Fatal(err)
	}
	if err = os.Link(filepath.Join(root, a.Root, a.Task), review); err != nil {
		t.Fatal(err)
	}
	if _, err = Prepare(ctx, root, "baseline-pack.json", "plugin-profile.json"); err == nil {
		t.Fatal("hardlink accepted as independent reference")
	}
}
