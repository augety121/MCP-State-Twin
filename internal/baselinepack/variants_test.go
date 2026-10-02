package baselinepack

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/augety121/mcp-state-twin/internal/agenteval"
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
		report, err := agenteval.RunCases(ctx, filepath.Join(root, "expanded", e.Entry.Root), e.Entry.Cases)
		if err != nil || report.Decision != "matched" || report.Matched != report.Planned {
			t.Fatalf("%s: %+v %v", key, report, err)
		}
	}
	for family, n := range counts {
		if n != 3 {
			t.Fatal(family, n)
		}
	}
	if len(seen) != 12 {
		t.Fatal("expected four case groups in each split", len(seen))
	}
	if _, err = GenerateVariants(ctx, root, "baseline-pack.json", "plugin-profile.json", "expanded"); err == nil {
		t.Fatal("overwritten")
	}
	if _, err = variantText([]byte("octo"), "arbitrary-script"); err == nil {
		t.Fatal("unbounded parameter")
	}
}
