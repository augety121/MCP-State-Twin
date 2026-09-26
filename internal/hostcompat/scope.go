package hostcompat

const ScopePolicy = "host-report-scope-v1"

type ScopeAssessment struct {
	Policy               string   `json:"policy"`
	Mismatches           []string `json:"mismatches"`
	ClaimCurrentEligible bool     `json:"claimCurrentEligible"`
}

// AssessAgainst compares declared identities only; it does not discover a live
// configuration or authenticate either input. Temporal policy stays independent.
func AssessAgainst(r *Report, target *Target, at string) (*Assessment, error) {
	a, err := Assess(r, at)
	if err != nil {
		return nil, err
	}
	if err := target.Validate(); err != nil {
		return nil, err
	}
	limits, _ := target.Trial.Limits.values() // admission above requires every value
	mismatches := []string{}
	for _, field := range []struct {
		name               string
		observed, expected any
	}{
		{"runtime.version", r.Runtime.Version, target.Runtime.Version},
		{"runtime.revision", r.Runtime.Revision, target.Runtime.Revision},
		{"runtime.specDigest", r.Runtime.SpecDigest, target.Runtime.SpecDigest},
		{"runtime.surfaceDigest", r.Runtime.SurfaceDigest, target.Runtime.SurfaceDigest},
		{"runtime.snapshotDigest", r.Runtime.SnapshotDigest, target.Runtime.SnapshotDigest},
		{"host.profile", r.Host.Profile, target.Host.Profile},
		{"host.name", r.Host.Name, target.Host.Name},
		{"host.version", r.Host.Version, target.Host.Version},
		{"host.provider", r.Host.Provider, target.Host.Provider},
		{"host.requestedModel", r.Host.RequestedModel, target.Host.RequestedModel},
		{"host.model", r.Host.Model, target.Host.Model},
		{"mcp.configuredVersion", r.MCP.ConfiguredVersion, target.MCP.ConfiguredVersion},
		{"mcp.negotiatedVersion", r.MCP.NegotiatedVersion, target.MCP.NegotiatedVersion},
		{"mcp.transport", r.MCP.Transport, target.MCP.Transport},
		{"mcp.endpointTrust", r.MCP.EndpointTrust, target.MCP.EndpointTrust},
		{"mcp.deploymentProfileDigest", r.MCP.DeploymentProfileDigest, target.MCP.DeploymentProfileDigest},
		{"mcp.observedSurfaceDigest", r.MCP.ObservedSurfaceDigest, target.MCP.ObservedSurfaceDigest},
		{"mcp.surfaceStatus", r.MCP.SurfaceStatus, target.MCP.SurfaceStatus},
		{"claim.procedureDigest", r.Claim.ProcedureDigest, target.ProcedureDigest},
		{"trial.scenarioDigest", r.Trial.ScenarioDigest, target.Trial.ScenarioDigest},
		{"trial.promptDigest", r.Trial.PromptDigest, target.Trial.PromptDigest},
		{"trial.toolPolicyDigest", r.Trial.ToolPolicyDigest, target.Trial.ToolPolicyDigest},
		{"trial.limits.providerRequests", r.Trial.Limits.ProviderRequests, limits.ProviderRequests},
		{"trial.limits.toolCalls", r.Trial.Limits.ToolCalls, limits.ToolCalls},
		{"trial.limits.wallTimeMs", r.Trial.Limits.WallTimeMS, limits.WallTimeMS},
		{"trial.limits.maxTraceBytes", r.Trial.Limits.MaxTraceBytes, limits.MaxTraceBytes},
		{"trial.limits.retriesPerProviderRequest", r.Trial.Limits.RetriesPerProviderRequest, limits.RetriesPerProviderRequest},
		{"trial.limits.retriesPerToolCall", r.Trial.Limits.RetriesPerToolCall, limits.RetriesPerToolCall},
		{"trial.limits.repeatedIdenticalCalls", r.Trial.Limits.RepeatedIdenticalCalls, limits.RepeatedIdenticalCalls},
		{"redaction.policy", r.Redaction.Policy, target.RedactionPolicy},
	} {
		if field.observed != field.expected {
			mismatches = append(mismatches, field.name)
		}
	}
	a.ScopeStatus = "matched"
	if len(mismatches) != 0 {
		a.ScopeStatus = "mismatched"
	}
	a.Scope = &ScopeAssessment{Policy: ScopePolicy, Mismatches: mismatches, ClaimCurrentEligible: a.ClaimTimeEligible && len(mismatches) == 0}
	return a, nil
}
