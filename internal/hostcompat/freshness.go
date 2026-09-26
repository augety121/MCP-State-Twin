package hostcompat

import (
	"errors"
	"time"
)

const (
	AssessmentFormat = "statetwin.dev/host-report-assessment/v1alpha1"
	FreshnessPolicy  = "host-report-time-v1"
)

// Assessment is a time-policy assessment of a declaration, not live evidence
// verification. Time eligibility is necessary but never sufficient to publish.
type Assessment struct {
	Format              string           `json:"format"`
	Policy              string           `json:"policy"`
	Profile             string           `json:"profile"`
	DeclaredLevel       string           `json:"declaredLevel"`
	AssessedAt          string           `json:"assessedAt"`
	ObservedAt          string           `json:"observedAt"`
	DeclaredValidUntil  string           `json:"declaredValidUntil,omitempty"`
	EffectiveValidUntil string           `json:"effectiveValidUntil"`
	TTLSeconds          int64            `json:"ttlSeconds"`
	Freshness           string           `json:"freshness"`
	ClaimTimeEligible   bool             `json:"claimTimeEligible"`
	ScopeStatus         string           `json:"scopeStatus"`
	Provenance          string           `json:"provenance"`
	PublicationAllowed  bool             `json:"publicationAllowed"`
	Scope               *ScopeAssessment `json:"scope,omitempty"`
}

// Assess never reads the wall clock, discovers credentials, hashes files, checks
// other profiles, or changes the original report. Callers supply the audit time.
func Assess(r *Report, at string) (*Assessment, error) {
	var problems []string
	now, err := requireUTC(&problems, "at", at)
	if err != nil {
		return nil, errors.New("HOST_REPORT_ASSESSMENT_TIME_INVALID")
	}
	if err := r.Validate(); err != nil {
		return nil, err
	}
	created, _ := time.Parse(time.RFC3339Nano, r.Metadata.CreatedAt)
	ttl := 30 * 24 * time.Hour
	if oneOf(r.Host.Profile, "chatgpt-mcp", "claude-code-mcp") {
		ttl = 14 * 24 * time.Hour
	}
	expires := created.Add(ttl)
	if r.Claim.ValidUntil != "" {
		declared, _ := time.Parse(time.RFC3339Nano, r.Claim.ValidUntil)
		if declared.Before(expires) {
			expires = declared
		}
	}
	if expires.Year() > 9999 {
		return nil, errors.New("HOST_REPORT_EXPIRY_OUT_OF_RANGE")
	}
	state := "within_window"
	if now.Before(created) {
		state = "not_yet_valid"
	} else if !now.Before(expires) {
		state = "expired"
	}
	return &Assessment{
		Format: AssessmentFormat, Policy: FreshnessPolicy,
		Profile: r.Host.Profile, DeclaredLevel: r.Claim.Level,
		AssessedAt: now.Format(time.RFC3339Nano), ObservedAt: created.Format(time.RFC3339Nano),
		DeclaredValidUntil: r.Claim.ValidUntil, EffectiveValidUntil: expires.Format(time.RFC3339Nano),
		TTLSeconds: int64(ttl / time.Second), Freshness: state,
		ClaimTimeEligible: r.Claim.Level == "verified" && state == "within_window",
		ScopeStatus:       "not_checked", Provenance: "not_verified", PublicationAllowed: false,
	}, nil
}
