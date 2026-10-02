package baseline

import (
	"errors"
	"time"

	"github.com/augety121/mcp-state-twin/internal/agentapi"
	"github.com/augety121/mcp-state-twin/internal/agenteval"
	"github.com/augety121/mcp-state-twin/internal/baselinepack"
	"github.com/augety121/mcp-state-twin/internal/bundle"
)

type LiveReadiness struct {
	Format              string `json:"format"`
	Profile             string `json:"profile"`
	Status              string `json:"status"`
	Reason              string `json:"reason"`
	ProviderRequests    int    `json:"providerRequests"`
	CostPolicy          string `json:"costPolicy"`
	PluginClaimEligible bool   `json:"pluginClaimEligible"`
}

// CheckLive delegates authorization and resource semantics to the accepted
// API bridge. It never reads credentials or constructs a network client.
// The bridge is not an Inspect/stdio compatibility observation.
func CheckLive(p *agentapi.Plan, entry *baselinepack.PreparedEntry, now time.Time, allow bool, usedRequests int) (LiveReadiness, error) {
	r := LiveReadiness{Format: "statetwin.dev/baseline-live-readiness/v1alpha1", Status: "blocked", Reason: "LIVE_PLAN_INVALID", CostPolicy: agentapi.CostPolicy}
	if p == nil || entry == nil || usedRequests < 0 {
		return r, errors.New(r.Reason)
	}
	r.Profile = p.Profile
	if p.Profile != agentapi.Profile {
		r.Reason = "LIVE_PROFILE_UNSUPPORTED"
		return r, errors.New(r.Reason)
	}
	if err := p.Authorize(now, allow); err != nil {
		r.Reason = err.Error()
		return r, err
	}
	if usedRequests >= p.MaxRequests {
		r.Reason = "BUDGET_EXHAUSTED"
		return r, errors.New(r.Reason)
	}
	b, err := bundle.OpenBytes(entry.Bundle)
	if err != nil || !baselinepack.Equal(p.Task, entry.Task) || agenteval.PreflightLive(p, b) != nil {
		r.Reason = "LIVE_PLAN_BINDING_MISMATCH"
		return r, errors.New(r.Reason)
	}
	r.Status = "ready-for-explicit-api-bridge-run"
	r.Reason = "separate-stdio-and-product-evidence-required"
	return r, nil
}

type Binding struct {
	Runtime      string `json:"runtime"`
	Revision     string `json:"revision"`
	Host         string `json:"host"`
	HostVersion  string `json:"hostVersion"`
	Adapter      string `json:"adapter"`
	Protocol     string `json:"protocol"`
	PackID       string `json:"packId"`
	PackRevision string `json:"packRevision"`
	Oracle       string `json:"oracle"`
	Statistics   string `json:"statistics"`
	Evidence     string `json:"evidence"`
}
type Claim struct {
	Format           string  `json:"format"`
	Binding          Binding `json:"binding"`
	Kind             string  `json:"kind"`
	ObservedAt       string  `json:"observedAt"`
	ValidUntil       string  `json:"validUntil"`
	Observation      string  `json:"observation"`
	Outcome          string  `json:"outcome"`
	Revoked          bool    `json:"revoked"`
	RevocationReason string  `json:"revocationReason"`
	EvidenceRef      string  `json:"evidenceRef"`
}
type ClaimAssessment struct {
	Status             string `json:"status"`
	Reason             string `json:"reason"`
	SourceTrust        string `json:"sourceTrust"`
	PublicationAllowed bool   `json:"publicationAllowed"`
}

// AssessClaim is declaration assessment only; unsigned local declarations are
// never upgraded to independently verified host observations.
func AssessClaim(c Claim, current Binding, at time.Time) (ClaimAssessment, error) {
	r := ClaimAssessment{Status: "unverified", Reason: "independent-evidence-not-verified", SourceTrust: "local-operator-asserted"}
	if c.Format != "statetwin.dev/baseline-host-claim/v1alpha1" || (c.Kind != "api" && c.Kind != "product") || c.Binding.Host == "" || c.Binding.HostVersion == "" || c.Binding.PackID == "" || c.EvidenceRef == "" {
		return r, errors.New("BASELINE_CLAIM_INVALID")
	}
	observed, e := time.Parse(time.RFC3339Nano, c.ObservedAt)
	expiry, f := time.Parse(time.RFC3339Nano, c.ValidUntil)
	if e != nil || f != nil || !expiry.After(observed) {
		return r, errors.New("BASELINE_CLAIM_INVALID")
	}
	ttl := 30 * 24 * time.Hour
	if c.Kind == "product" {
		ttl = 14 * 24 * time.Hour
	}
	if expiry.After(observed.Add(ttl)) {
		expiry = observed.Add(ttl)
	}
	if c.Revoked {
		if c.RevocationReason == "" {
			return r, errors.New("BASELINE_CLAIM_INVALID")
		}
		r.Status = "regressed"
		r.Reason = "revoked"
		return r, nil
	}
	if !baselinepack.Equal(c.Binding, current) || !at.Before(expiry) || at.Before(observed) {
		r.Status = "stale"
		r.Reason = "binding-or-time-changed"
		return r, nil
	}
	if c.Outcome == "failed" {
		r.Status = "regressed"
		r.Reason = "observed-regression"
		return r, nil
	}
	if c.Outcome != "passed" || (c.Observation != "contract-test" && c.Observation != "operator-observed") {
		return r, errors.New("BASELINE_CLAIM_INVALID")
	}
	if c.Observation == "contract-test" {
		r.Status = "experimental"
		r.Reason = "contract-evidence-only"
	}
	return r, nil
}
