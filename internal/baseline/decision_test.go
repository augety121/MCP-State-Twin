package baseline

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/augety121/mcp-state-twin/internal/baselinepack"
	"github.com/augety121/mcp-state-twin/internal/testfixture"
)

func fixture(t *testing.T) (string, *Prepared) {
	t.Helper()
	root := testfixture.Baseline(t)
	p, e := Prepare(context.Background(), root, "baseline-pilot.json")
	if e != nil {
		t.Fatal(e)
	}
	return root, p
}
func outcomes(f Frozen, improve bool) []Observation {
	result := []Observation{}
	families := map[string]int{}
	for _, t := range f.Trials {
		if _, ok := families[t.FamilyID]; !ok {
			families[t.FamilyID] = len(families)
		}
		n := families[t.FamilyID]
		success := true
		if t.ConfigID == f.Plan.Configs[0].ID && n%3 == 0 {
			success = false
		}
		if !improve && t.ConfigID == f.Plan.Configs[1].ID && n%2 == 0 {
			success = false
		}
		result = append(result, Observation{TrialID: t.ID, State: "completed", Verified: true, Cleanup: "complete", Success: success, ObservedModel: "unknown", Usage: Usage{Status: "unavailable"}})
	}
	return result
}
func TestBaselinePlanIdentityAndFreeze(t *testing.T) {
	root, p := fixture(t)
	if len(p.Frozen.Trials) != 144 || p.Frozen.Trials[2].ConfigID != p.Frozen.Plan.Configs[1].ID {
		t.Fatal("inventory/order")
	}
	ctx := context.Background()
	f, e := Freeze(ctx, root, "baseline-pilot.json", "frozen")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = Freeze(ctx, root, "baseline-pilot.json", "frozen"); e == nil {
		t.Fatal("overwrite")
	}
	if e = os.WriteFile(filepath.Join(root, "baseline-pack.json"), []byte(`{}`), 0600); e != nil {
		t.Fatal(e)
	}
	loaded, e := Load(ctx, filepath.Join(root, "frozen"), "baseline-pilot.json")
	if e != nil || !baselinepack.Equal(*f, loaded.Frozen) {
		t.Fatal(e)
	}
	report, e := Assess(ctx, filepath.Join(root, "frozen"), "baseline-pilot.json")
	if e != nil || report.Decision != "invalid" || len(report.Trials) != 144 {
		t.Fatal(report, e)
	}
	for _, m := range report.Metrics {
		if m.Planned != 72 || m.Missing != 72 || m.CostStatus != "unknown" || m.Cost != nil {
			t.Fatal(m)
		}
	}
}
func TestBaselineDecisionPolicy(t *testing.T) {
	_, p := fixture(t)
	f := p.Frozen
	good := outcomes(f, true)
	r, e := Decide(f, good)
	if e != nil || r.Decision != "pass" {
		t.Fatal(r, e)
	}
	bad := outcomes(f, false)
	r, e = Decide(f, bad)
	if e != nil || r.Decision != "fail" {
		t.Fatal(r, e)
	}
	all := outcomes(f, true)
	for i := range all {
		all[i].Success = true
	}
	r, _ = Decide(f, all)
	if r.Decision != "inconclusive" || r.Interval.Status != "degenerate" {
		t.Fatal(r)
	}
	r, _ = Decide(f, good[1:])
	if r.Decision != "invalid" || r.Metrics[f.Trials[0].ConfigID].Planned != 72 {
		t.Fatal(r)
	}
	changed := append([]Observation{}, good...)
	for i, t := range f.Trials {
		if t.ConfigID == f.Plan.Configs[1].ID {
			changed[i].PolicyFailures = 1
			break
		}
	}
	r, _ = Decide(f, changed)
	if r.Decision != "fail" || r.Reason != "policy-or-required-task" {
		t.Fatal(r)
	}
	if _, e = Decide(f, append(good, good[0])); e == nil {
		t.Fatal("duplicate evidence")
	}
	// Timeouts cannot enter successful latency statistics; unknown cost remains unknown.
	latency := 0.001
	good[0].State = "timeout"
	good[0].LatencySeconds = &latency
	r, _ = Decide(f, good)
	if r.Decision != "invalid" || len(r.Metrics[f.Trials[0].ConfigID].SuccessfulLatencies) != 0 {
		t.Fatal(r)
	}
}
func TestBaselineStatisticalCounterexamples(t *testing.T) {
	clusters := []Cluster{}
	for i := 0; i < 24; i++ {
		domain := "issue-tracker"
		if i >= 12 {
			domain = "package-registry"
		}
		v := 0.0
		if i%3 == 0 {
			v = 1
		}
		clusters = append(clusters, Cluster{string(rune('a' + i)), domain, v})
	}
	result := Bootstrap(clusters, 3, 78)
	if result.Status != "estimated" || result.Lower == nil || result.Upper == nil {
		t.Fatal(result)
	}
	// Independent binomial quantiles for 24 Bernoulli(p=1/3) clusters:
	// 2.5% -> 4/24 and 97.5% -> 13/24. Fixed seeded Monte Carlo is
	// checked against this exact distribution as well as reproducibility.
	if *result.Lower != 4.0/24 || *result.Upper != 13.0/24 {
		t.Fatal(*result.Lower, *result.Upper)
	}
	if !baselinepack.Equal(result, Bootstrap(clusters, 3, 78)) {
		t.Fatal("nondeterministic")
	}
	if Bootstrap(clusters[:19], 3, 78).Status != "insufficient-sample" || Bootstrap(clusters, 2, 78).Status != "insufficient-sample" {
		t.Fatal("insufficient sample")
	}
	for i := range clusters {
		clusters[i].Difference = 0
	}
	if Bootstrap(clusters, 3, 78).Status != "degenerate" {
		t.Fatal("zero width")
	}
	clusters[1].Family = clusters[0].Family
	if Bootstrap(clusters, 3, 78).Status != "invalid" {
		t.Fatal("pseudo replication")
	}
}
func TestBaselineShardBudgetsAndNoEffects(t *testing.T) {
	root, p := fixture(t)
	bad := p.Frozen.Plan
	bad.TrialsPerShard = 13
	raw, _ := json.Marshal(bad)
	os.WriteFile(filepath.Join(root, "bad.json"), raw, 0600)
	if _, err := Freeze(context.Background(), root, "bad.json", "bad-output"); err == nil {
		t.Fatal("budget bypass")
	}
	if _, e := os.Stat(filepath.Join(root, "bad-output")); !os.IsNotExist(e) {
		t.Fatal("preflight effect")
	}
	bad = p.Frozen.Plan
	bad.Policy.Delta = nil
	raw, _ = json.Marshal(bad)
	os.WriteFile(filepath.Join(root, "bad.json"), raw, 0600)
	if _, e := Prepare(context.Background(), root, "bad.json"); e == nil {
		t.Fatal("implicit delta")
	}
}
