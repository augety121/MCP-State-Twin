package baselinepack

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/augety121/mcp-state-twin/internal/agenteval"
	"github.com/augety121/mcp-state-twin/internal/testfixture"
)

func TestQualityRunsEachRelativeRoot(t *testing.T) {
	root := testfixture.Baseline(t)
	ctx := context.Background()
	p, err := Prepare(ctx, root, "baseline-pack.json", "plugin-profile.json")
	if err != nil {
		t.Fatal(err)
	}
	first := p.Pack.Entries[0]
	// Both entries reach the same manifest, but its Task/world/witness paths
	// resolve against two independent roots. Copies are not hard links.
	if err := os.CopyFS(root, os.DirFS(filepath.Join(root, first.Root))); err != nil {
		t.Fatal(err)
	}
	shadow := first
	shadow.ID, shadow.Root = "shadow-read-issue", "."
	shadow.Cases = filepath.ToSlash(filepath.Join(first.Root, first.Cases))
	p.Pack.Entries = []Entry{first, shadow}
	writeJSON := func(name string, value any) {
		t.Helper()
		raw, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, name), raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	writeJSON("root-quality.json", p.Pack)
	// This witness is syntactically admitted but cannot satisfy the read
	// oracle. The original root retains its successful witness.
	witnessName := "agent-witnesses/read-issue.json"
	raw, err := os.ReadFile(filepath.Join(root, witnessName))
	if err != nil {
		t.Fatal(err)
	}
	witness, err := agenteval.DecodeWitness(raw)
	if err != nil {
		t.Fatal(err)
	}
	witness.Answer = map[string]any{"number": 999}
	writeJSON(witnessName, witness)
	if _, err := Prepare(ctx, root, "root-quality.json", "plugin-profile.json"); err != nil {
		t.Fatalf("both roots must pass static admission: %v", err)
	}
	quality, err := Qualify(ctx, root, "root-quality.json", "plugin-profile.json")
	if err == nil || quality == nil || quality.Decision != "partial" || len(quality.Groups) != 2 {
		t.Fatalf("distinct quality world skipped: %+v, %v", quality, err)
	}
	if quality.Groups[0].State != "matched" || quality.Groups[1].State != "failed" {
		t.Fatalf("wrong root result: %+v", quality.Groups)
	}
}
