package baselinepack

import (
	"context"
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
	quality, err := Qualify(ctx, filepath.Join(root, "expanded"), "baseline-pack.json", "plugin-profile.json")
	if err != nil || quality.Decision != "matched" || len(quality.Groups) != 12 {
		t.Fatal(quality, err)
	}
	planned := 0
	for _, g := range quality.Groups {
		planned += g.Report.Planned
		if g.Report.Matched != g.Report.Planned {
			t.Fatal(g)
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
