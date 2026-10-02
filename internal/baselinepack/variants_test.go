package baselinepack

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/augety121/mcp-state-twin/internal/testfixture"
)

func TestBaselineVariantQualification(t *testing.T) {
	root := testfixture.Baseline(t)
	ctx := context.Background()
	summary, err := GenerateVariants(ctx, root, "baseline-pack.json", "plugin-profile.json", "expanded")
	if err != nil {
		t.Fatal(err)
	}
	if summary.Entries != 72 || summary.Families != 24 {
		t.Fatal(summary)
	}
	p, err := Prepare(ctx, filepath.Join(root, "expanded"), "baseline-pack.json", "plugin-profile.json")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	counts := map[string]int{}
	for _, e := range p.Entries {
		counts[e.Entry.Family]++
		if e.Entry.Split == "evaluation" && e.Entry.Disclosure != "public-evaluation-split" {
			t.Fatal(e.Entry)
		}
		key := filepath.Join(e.Entry.Root, e.Entry.Cases)
		if seen[key] {
			continue
		}
		seen[key] = true
	}
	for family, n := range counts {
		if n != 3 {
			t.Fatal(family, n)
		}
	}
	if len(seen) != 12 {
		t.Fatal("expected four case groups in each split", len(seen))
	}
	planned := 0
	// Each public operation retains its 120s production deadline under race.
	// Partition by declared split; the aggregate still covers all 309 cases.
	for _, split := range []string{"dev", "regression", "evaluation"} {
		manifest := p.Pack
		manifest.Entries = nil
		for _, e := range p.Pack.Entries {
			if e.Split == split {
				manifest.Entries = append(manifest.Entries, e)
			}
		}
		name := "quality-" + split + ".json"
		raw, err := json.Marshal(manifest)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(root, "expanded", name), raw, 0600); err != nil {
			t.Fatal(err)
		}
		quality, err := Qualify(ctx, filepath.Join(root, "expanded"), name, "plugin-profile.json")
		if err != nil || quality.Decision != "matched" || len(quality.Groups) != 4 {
			t.Fatalf("split %s: %+v %v", split, quality, err)
		}
		for _, g := range quality.Groups {
			planned += g.Report.Planned
			if g.Report.Matched != g.Report.Planned {
				t.Fatal(g)
			}
		}
	}
	if planned != 309 {
		t.Fatal("quality inventory", planned)
	}
	if _, err = GenerateVariants(ctx, root, "baseline-pack.json", "plugin-profile.json", "expanded"); err == nil {
		t.Fatal("overwritten")
	}
	if _, err = variantText([]byte("octo"), "arbitrary-script"); err == nil {
		t.Fatal("unbounded parameter")
	}
}
