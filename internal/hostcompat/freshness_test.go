package hostcompat

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func reportForProfile(profile string) *Report {
	r := validGenericReport()
	r.Host.Profile = profile
	if profile == "generic-mcp" || profile == "custom-mcp" {
		return r
	}
	r.Host.Name = "synthetic-host"
	r.Host.Version = "synthetic-v1"
	r.Host.Model = "synthetic-model-v1"
	r.Host.Provider = "openai"
	if profile == "anthropic-api-mcp" || profile == "claude-code-mcp" {
		r.Host.Provider = "anthropic"
	}
	r.MCP.EndpointTrust = "private"
	r.MCP.DeploymentProfileDigest = digest("d")
	r.Trial.Limits.ProviderRequests = 2
	r.Evidence.ProviderRequestIDDigest = digest("e")
	r.Evidence.Checks = []string{"tool-discovery", "read-only-call", "state-changing-scenario", "domain-error", "terminal-assertions", "bounded-termination", "artifact-redaction"}
	return r
}

func TestAssessmentProfilesAndTimeBoundaries(t *testing.T) {
	for _, profile := range []string{"generic-mcp", "custom-mcp", "openai-api-mcp", "anthropic-api-mcp", "chatgpt-mcp", "claude-code-mcp"} {
		t.Run(profile, func(t *testing.T) {
			r := reportForProfile(profile)
			created, _ := time.Parse(time.RFC3339Nano, r.Metadata.CreatedAt)
			days := 30
			if profile == "chatgpt-mcp" || profile == "claude-code-mcp" {
				days = 14
			}
			expires := created.Add(time.Duration(days) * 24 * time.Hour)
			for _, tt := range []struct {
				at       time.Time
				state    string
				eligible bool
			}{
				{created.Add(-time.Nanosecond), "not_yet_valid", false},
				{created, "within_window", true},
				{expires.Add(-time.Nanosecond), "within_window", true},
				{expires, "expired", false},
				{expires.Add(time.Nanosecond), "expired", false},
			} {
				a, err := Assess(r, tt.at.Format(time.RFC3339Nano))
				if err != nil {
					t.Fatal(err)
				}
				if a.Freshness != tt.state || a.ClaimTimeEligible != tt.eligible || a.TTLSeconds != int64(days*24*3600) || a.EffectiveValidUntil != expires.Format(time.RFC3339Nano) {
					t.Fatalf("incorrect boundary: %+v", a)
				}
				if a.PublicationAllowed || a.Provenance != "not_verified" || a.ScopeStatus != "not_checked" || a.Policy != FreshnessPolicy || a.Format != AssessmentFormat {
					t.Fatalf("unearned claim: %+v", a)
				}
			}
		})
	}
}

func TestAssessmentDeclaredWindowAndNonVerified(t *testing.T) {
	r := validGenericReport()
	r.Metadata.CreatedAt = "2026-08-24T00:00:00.123456789Z"
	r.Claim.ValidUntil = "2026-08-25T00:00:00.123456789Z"
	a, err := Assess(r, "2026-08-25T00:00:00.123456788Z")
	if err != nil || !a.ClaimTimeEligible || a.EffectiveValidUntil != r.Claim.ValidUntil {
		t.Fatal(a, err)
	}
	a, err = Assess(r, r.Claim.ValidUntil)
	if err != nil || a.Freshness != "expired" || a.ClaimTimeEligible {
		t.Fatal(a, err)
	}
	for _, level := range []string{"experimental", "regressed"} {
		for _, expiry := range []string{"", "2026-08-25T00:00:00Z"} {
			r.Claim.Level, r.Claim.ValidUntil = level, expiry
			a, err := Assess(r, r.Metadata.CreatedAt)
			if err != nil || a.ClaimTimeEligible || a.PublicationAllowed || a.Freshness != "within_window" || a.DeclaredLevel != level {
				t.Fatal(a, err)
			}
		}
	}
}

func TestAssessmentInvalidInputAndOverflow(t *testing.T) {
	for _, at := range []string{"", "now", "2026-02-30T00:00:00Z", "2026-08-24", "2026-08-24T00:00:00+00:00", "2026-08-24T00:00:00Z\n"} {
		if a, err := Assess(validGenericReport(), at); err == nil || a != nil {
			t.Fatalf("invalid time %q accepted", at)
		}
	}
	if a, err := Assess(nil, "2026-08-24T00:00:00Z"); err == nil || a != nil {
		t.Fatal("nil report accepted")
	}
	r := validGenericReport()
	r.MCP.ObservedSurfaceDigest = digest("f")
	if a, err := Assess(r, r.Metadata.CreatedAt); err == nil || a != nil {
		t.Fatal("assessment bypassed admission")
	}
	r = validGenericReport()
	r.Claim.Level, r.Claim.ValidUntil = "experimental", ""
	r.Metadata.CreatedAt = "9999-12-31T00:00:00Z"
	if a, err := Assess(r, r.Metadata.CreatedAt); err == nil || a != nil {
		t.Fatal("unrepresentable expiry accepted")
	}
	r.Claim.ValidUntil = "9999-12-31T01:00:00Z"
	if a, err := Assess(r, r.Metadata.CreatedAt); err != nil || a.EffectiveValidUntil != r.Claim.ValidUntil {
		t.Fatal("earlier representable expiry rejected", a, err)
	}
}

func TestAssessmentIsReadOnlyDeterministicAndMinimal(t *testing.T) {
	r := validGenericReport()
	r.Host.Name = "do-not-echo-host"
	r.Host.Version = "do-not-echo-version"
	raw, err := yaml.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "report.yaml")
	if err := os.WriteFile(file, raw, 0600); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(file)
	if err != nil {
		t.Fatal(err)
	}
	a, err := Assess(loaded, loaded.Metadata.CreatedAt)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Assess(loaded, loaded.Metadata.CreatedAt)
	if err != nil {
		t.Fatal(err)
	}
	left, _ := json.Marshal(a)
	right, _ := json.Marshal(b)
	if !bytes.Equal(left, right) {
		t.Fatal("nondeterministic assessment")
	}
	after, err := os.ReadFile(file)
	if err != nil || !bytes.Equal(raw, after) {
		t.Fatal("source changed", err)
	}
	encoded, _ := yaml.Marshal(loaded)
	if !bytes.Equal(encoded, raw) {
		t.Fatal("in-memory report changed")
	}
	for _, forbidden := range []string{r.Host.Name, r.Host.Version, r.Runtime.SurfaceDigest, file, "reportDigest"} {
		if strings.Contains(string(left), forbidden) {
			t.Fatal("unnecessary report content exposed")
		}
	}
}
