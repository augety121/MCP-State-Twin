package baseline

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/augety121/mcp-state-twin/internal/agentapi"
	"github.com/augety121/mcp-state-twin/internal/baselinepack"
	"github.com/augety121/mcp-state-twin/internal/bundle"
	"github.com/augety121/mcp-state-twin/internal/testfixture"
)

func TestBaselineLiveReadinessZeroOutbound(t *testing.T) {
	root := testfixture.Baseline(t)
	pack, e := baselinepack.Prepare(context.Background(), root, "baseline-pack.json", "plugin-profile.json")
	if e != nil {
		t.Fatal(e)
	}
	entry := &pack.Entries[0]
	b, e := bundle.OpenBytes(entry.Bundle)
	if e != nil {
		t.Fatal(e)
	}
	now := time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC)
	plan := agentapi.Plan{Format: agentapi.PlanFormat, ID: "six-task-smoke-read", Provider: "openai", Profile: agentapi.Profile, Model: "contract-model", Task: entry.Task, BundleDigest: b.Digest, IssuedAt: now.Format(time.RFC3339), ExpiresAt: now.Add(time.Hour).Format(time.RFC3339), MaxRequests: 2, MaxOutputTokens: 128, CostPolicy: agentapi.CostPolicy, Approved: true, SyntheticDataApproved: true, UnknownCostApproved: true}
	r, e := CheckLive(&plan, entry, now, true, 0)
	if e != nil || r.Status != "ready-for-explicit-api-bridge-run" || r.ProviderRequests != 0 || r.PluginClaimEligible {
		t.Fatal(r, e)
	}
	for _, condition := range []string{"unapproved", "expired", "wrong-profile", "exhausted", "wrong-task", "flag-missing"} {
		t.Run(condition, func(t *testing.T) {
			p := plan
			at := now
			used := 0
			allow := true
			switch condition {
			case "unapproved":
				p.Approved = false
			case "expired":
				at = now.Add(time.Hour)
			case "wrong-profile":
				p.Profile = "inspect-live"
			case "exhausted":
				used = 2
			case "wrong-task":
				p.Task = pack.Entries[1].Task
			case "flag-missing":
				allow = false
			}
			r, e := CheckLive(&p, entry, at, allow, used)
			if e == nil || r.ProviderRequests != 0 || r.Status != "blocked" {
				t.Fatal(r, e)
			}
		})
	}
}

func TestBaselineClaimBindingFreshnessAndRevocation(t *testing.T) {
	now := time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC)
	binding := Binding{Runtime: "v1", Revision: "head", Host: "product", HostVersion: "exact", Adapter: "inspect-0.3.275", Protocol: "2025-11-25", PackID: "reference", PackRevision: "v2", Oracle: "task-v1", Statistics: Algorithm, Evidence: "plugin-v1"}
	c := Claim{Format: "statetwin.dev/baseline-host-claim/v1alpha1", Binding: binding, Kind: "product", ObservedAt: now.Format(time.RFC3339), ValidUntil: now.Add(30 * 24 * time.Hour).Format(time.RFC3339), Observation: "operator-observed", Outcome: "passed", EvidenceRef: "observation.json"}
	r, e := AssessClaim(c, binding, now)
	if e != nil || r.Status != "unverified" || r.PublicationAllowed {
		t.Fatal(r, e)
	}
	r, e = AssessClaim(c, binding, now.Add(14*24*time.Hour))
	if e != nil || r.Status != "stale" {
		t.Fatal(r, e)
	}
	for _, field := range []string{"host", "adapter", "protocol", "pack", "oracle", "evidence"} {
		changed := binding
		switch field {
		case "host":
			changed.HostVersion = "new"
		case "adapter":
			changed.Adapter = "new"
		case "protocol":
			changed.Protocol = "new"
		case "pack":
			changed.PackRevision = "new"
		case "oracle":
			changed.Oracle = "new"
		case "evidence":
			changed.Evidence = "new"
		}
		r, e = AssessClaim(c, changed, now)
		if e != nil || r.Status != "stale" {
			t.Fatal(field, r, e)
		}
	}
	c.Revoked = true
	c.RevocationReason = "regression"
	r, e = AssessClaim(c, binding, now)
	if e != nil || r.Status != "regressed" {
		t.Fatal(r, e)
	}
}

func TestBaselineLiveShardApprovalBeforeCredentialsAndClaim(t *testing.T) {
	root, p := fixture(t)
	plan := p.Frozen.Plan
	plan.Entries = plan.Entries[:1]
	plan.Repeats = 1
	plan.Mode = "api-bridge-live"
	plan.LivePlans = map[string]string{}
	for i := range plan.Configs {
		c := &plan.Configs[i]
		c.Provider = "openai"
		c.RequestedModel = "test-model-not-a-product"
		c.Adapter = agentapi.Profile
		c.PromptVersion = "blind-objective-v1"
		c.Host.Name = "api-bridge"
		c.Host.Version = "v1alpha1"
		c.Host.Model = c.RequestedModel
	}
	now := time.Now().UTC()
	entry := &p.Pack.Entries[0]
	world, e := bundle.OpenBytes(entry.Bundle)
	if e != nil {
		t.Fatal(e)
	}
	for _, id := range []string{"trial-001", "trial-002"} {
		live := agentapi.Plan{Format: agentapi.PlanFormat, ID: id, Provider: "openai", Profile: agentapi.Profile, Model: "test-model-not-a-product", Task: entry.Task, BundleDigest: world.Digest, IssuedAt: now.Add(-time.Minute).Format(time.RFC3339Nano), ExpiresAt: now.Add(time.Hour).Format(time.RFC3339Nano), MaxRequests: 2, MaxOutputTokens: 128, CostPolicy: agentapi.CostPolicy}
		name := id + ".live.json"
		plan.LivePlans[id] = name
		raw, _ := json.Marshal(live)
		os.WriteFile(filepath.Join(root, name), raw, 0600)
	}
	raw, _ := json.Marshal(plan)
	os.WriteFile(filepath.Join(root, "live-baseline.json"), raw, 0600)
	ctx := context.Background()
	if _, e = Freeze(ctx, root, "live-baseline.json", "live-frozen"); e != nil {
		t.Fatal(e)
	}
	frozen := filepath.Join(root, "live-frozen")
	reads := 0
	for _, allow := range []bool{false, true} {
		if _, e = RunLiveShard(ctx, frozen, "live-baseline.json", 1, allow, func() string { reads++; return "" }); e == nil {
			t.Fatal("approval bypass")
		}
	}
	if reads != 0 {
		t.Fatal("credential read before admission")
	}
	if _, e = os.Stat(filepath.Join(frozen, "inputs/runs")); !os.IsNotExist(e) {
		t.Fatal("claim before approval")
	}
	r, e := Assess(ctx, frozen, "live-baseline.json")
	if e != nil || r.Decision != "invalid" || r.Mode != "api-bridge-live" || r.HostObservation == "contract-test" {
		t.Fatal(r, e)
	}
}
