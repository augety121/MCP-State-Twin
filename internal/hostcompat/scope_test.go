package hostcompat

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestScopeAllProfilesAndSeparateTimeEligibility(t *testing.T) {
	for _, profile := range []string{"generic-mcp", "custom-mcp", "openai-api-mcp", "anthropic-api-mcp", "chatgpt-mcp", "claude-code-mcp"} {
		for _, level := range []string{"verified", "experimental", "regressed"} {
			r := reportForProfile(profile)
			r.Claim.Level = level
			target := targetForReport(r)
			for _, at := range []string{r.Metadata.CreatedAt, "2026-09-26T00:00:00Z", "2026-08-23T00:00:00Z"} {
				a, err := AssessAgainst(r, target, at)
				if err != nil {
					t.Fatal(err)
				}
				want := level == "verified" && at == r.Metadata.CreatedAt
				if a.ScopeStatus != "matched" || a.Scope == nil || a.Scope.Mismatches == nil || len(a.Scope.Mismatches) != 0 || a.Scope.ClaimCurrentEligible != want || a.Scope.Policy != ScopePolicy || a.PublicationAllowed || a.Provenance != "not_verified" {
					t.Fatalf("incorrect scoped claim: %+v", a)
				}
			}
		}
	}
}

func TestScopeRejectsEveryChangedDimension(t *testing.T) {
	for _, tt := range []struct {
		path    string
		mutate  func(*Target)
		invalid bool
	}{
		{"runtime.version", func(s *Target) { s.Runtime.Version = "synthetic-v2" }, false},
		{"runtime.revision", func(s *Target) { s.Runtime.Revision = strings.Repeat("b", 40) }, false},
		{"runtime.specDigest", func(s *Target) { s.Runtime.SpecDigest = digest("f") }, false},
		{"runtime.surfaceDigest", func(s *Target) { s.Runtime.SurfaceDigest = digest("f"); s.MCP.ObservedSurfaceDigest = digest("f") }, false},
		{"runtime.snapshotDigest", func(s *Target) { s.Runtime.SnapshotDigest = digest("f") }, false},
		{"host.profile", func(s *Target) { s.Host.Profile = "custom-mcp" }, false},
		{"host.name", func(s *Target) { s.Host.Name = "other-host" }, false},
		{"host.version", func(s *Target) { s.Host.Version = "synthetic-v2" }, false},
		{"host.provider", func(s *Target) { s.Host.Provider = "anthropic"; s.Host.Profile = "anthropic-api-mcp" }, false},
		{"host.requestedModel", func(s *Target) { s.Host.RequestedModel = "synthetic-requested" }, false},
		{"host.model", func(s *Target) { s.Host.Model = "synthetic-model-v2" }, false},
		{"mcp.configuredVersion", func(s *Target) { s.MCP.ConfiguredVersion = "2025-11-25" }, false},
		{"mcp.negotiatedVersion", func(s *Target) { s.MCP.NegotiatedVersion = "2025-11-25" }, false},
		{"mcp.transport", func(s *Target) { s.MCP.Transport = "stdio" }, true},
		{"mcp.endpointTrust", func(s *Target) { s.MCP.EndpointTrust = "public" }, false},
		{"mcp.deploymentProfileDigest", func(s *Target) { s.MCP.DeploymentProfileDigest = digest("f") }, false},
		{"mcp.observedSurfaceDigest", func(s *Target) { s.MCP.ObservedSurfaceDigest = digest("f"); s.Runtime.SurfaceDigest = digest("f") }, false},
		{"mcp.surfaceStatus", func(s *Target) { s.MCP.SurfaceStatus = "modified" }, true},
		{"claim.procedureDigest", func(s *Target) { s.ProcedureDigest = digest("f") }, false},
		{"trial.scenarioDigest", func(s *Target) { s.Trial.ScenarioDigest = digest("f") }, false},
		{"trial.promptDigest", func(s *Target) { s.Trial.PromptDigest = digest("f") }, false},
		{"trial.toolPolicyDigest", func(s *Target) { s.Trial.ToolPolicyDigest = digest("f") }, false},
		{"trial.limits.providerRequests", func(s *Target) { *s.Trial.Limits.ProviderRequests++ }, false},
		{"trial.limits.toolCalls", func(s *Target) { *s.Trial.Limits.ToolCalls++ }, false},
		{"trial.limits.wallTimeMs", func(s *Target) { *s.Trial.Limits.WallTimeMS++ }, false},
		{"trial.limits.maxTraceBytes", func(s *Target) { *s.Trial.Limits.MaxTraceBytes++ }, false},
		{"trial.limits.retriesPerProviderRequest", func(s *Target) { *s.Trial.Limits.RetriesPerProviderRequest++ }, false},
		{"trial.limits.retriesPerToolCall", func(s *Target) { *s.Trial.Limits.RetriesPerToolCall++ }, false},
		{"trial.limits.repeatedIdenticalCalls", func(s *Target) { *s.Trial.Limits.RepeatedIdenticalCalls++ }, false},
		{"redaction.policy", func(s *Target) { s.RedactionPolicy = "unknown" }, true},
	} {
		t.Run(tt.path, func(t *testing.T) {
			r := reportForProfile("openai-api-mcp")
			target := targetForReport(r)
			tt.mutate(target)
			a, err := AssessAgainst(r, target, r.Metadata.CreatedAt)
			if tt.invalid {
				if err == nil || a != nil {
					t.Fatal("unsupported target admitted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !a.ClaimTimeEligible || a.Scope.ClaimCurrentEligible || a.ScopeStatus != "mismatched" {
				t.Fatal("scope change silently inherited old evidence")
			}
			found := false
			for _, field := range a.Scope.Mismatches {
				if field == tt.path {
					found = true
				}
			}
			if !found {
				t.Fatalf("missing reason %s: %v", tt.path, a.Scope.Mismatches)
			}
		})
	}
}

func TestScopeOrderingPrivacyAndNoMutation(t *testing.T) {
	r := validGenericReport()
	target := targetForReport(r)
	target.Runtime.Version, target.Host.Name = "do-not-echo-runtime", "do-not-echo-host"
	target.Host.Profile = "custom-mcp"
	target.MCP.DeploymentProfileDigest = digest("f") // empty does not mean wildcard
	*target.Trial.Limits.ProviderRequests = 1        // zero does not mean wildcard
	target.Host.RequestedModel = "do-not-echo-requested-model"
	rawReport, _ := json.Marshal(r)
	rawTarget, _ := json.Marshal(target)
	a, err := AssessAgainst(r, target, r.Metadata.CreatedAt)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"runtime.version", "host.profile", "host.name", "host.requestedModel", "mcp.deploymentProfileDigest", "trial.limits.providerRequests"}
	if !reflect.DeepEqual(a.Scope.Mismatches, want) {
		t.Fatal(a.Scope.Mismatches)
	}
	b, err := AssessAgainst(r, target, r.Metadata.CreatedAt)
	if err != nil {
		t.Fatal(err)
	}
	left, _ := json.Marshal(a)
	right, _ := json.Marshal(b)
	if !bytes.Equal(left, right) {
		t.Fatal("nondeterministic result")
	}
	for _, forbidden := range []string{"do-not-echo", digest("f"), "expected", "observedValue"} {
		if strings.Contains(string(left), forbidden) {
			t.Fatal("raw target value echoed")
		}
	}
	afterReport, _ := json.Marshal(r)
	afterTarget, _ := json.Marshal(target)
	if !bytes.Equal(rawReport, afterReport) || !bytes.Equal(rawTarget, afterTarget) {
		t.Fatal("input mutated")
	}
	if a, err := AssessAgainst(r, nil, r.Metadata.CreatedAt); err == nil || a != nil {
		t.Fatal("missing target admitted")
	}
	if a, err := AssessAgainst(nil, target, r.Metadata.CreatedAt); err == nil || a != nil {
		t.Fatal("missing report admitted")
	}
}

func TestScopeKeepsOutcomesAndTimeSeparate(t *testing.T) {
	r := validGenericReport()
	target := targetForReport(r)
	r.Claim.Level = "experimental"
	r.MCP.SurfaceStatus, r.MCP.ObservedSurfaceDigest = "modified", digest("f")
	r.Trial.Index = 7
	r.Trial.Outcome = "assertion_failed"
	r.Evidence.AssertionSummary.Failed = 1
	a, err := AssessAgainst(r, target, r.Metadata.CreatedAt)
	if err != nil {
		t.Fatal(err)
	}
	if a.Scope.ClaimCurrentEligible || !reflect.DeepEqual(a.Scope.Mismatches, []string{"mcp.observedSurfaceDigest", "mcp.surfaceStatus"}) {
		t.Fatal("outcome ignored or surface missed")
	}
	r = reportForProfile("openai-api-mcp")
	target = targetForReport(r)
	target.Host.Profile = "chatgpt-mcp"
	a, err = AssessAgainst(r, target, "2026-09-10T00:00:00Z")
	if err != nil || !a.ClaimTimeEligible || a.Scope.ClaimCurrentEligible {
		t.Fatal("API report promoted to product claim", err)
	}
}
