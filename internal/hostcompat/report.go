package hostcompat

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/augety121/mcp-state-twin/internal/canonical"
	"github.com/augety121/mcp-state-twin/internal/logging"
	"github.com/augety121/mcp-state-twin/internal/strictyaml"
)

const (
	APIVersion     = "statetwin.dev/v1alpha1"
	Kind           = "HostCompatibilityReport"
	Format         = "statetwin.dev/host-compatibility-report/v1alpha1"
	MaxReportBytes = 1 << 20
)

var (
	identifierPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$`)
	digestPattern     = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)
	revisionPattern   = regexp.MustCompile(`^(?:[a-f0-9]{40}|[a-f0-9]{64})$`)
	protocolPattern   = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}$`)
	utcPattern        = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,9})?Z$`)
)

type Report struct {
	APIVersion string    `json:"apiVersion" yaml:"apiVersion"`
	Kind       string    `json:"kind" yaml:"kind"`
	Format     string    `json:"format" yaml:"format"`
	Metadata   Metadata  `json:"metadata" yaml:"metadata"`
	Claim      Claim     `json:"claim" yaml:"claim"`
	Runtime    Runtime   `json:"runtime" yaml:"runtime"`
	Host       Host      `json:"host" yaml:"host"`
	MCP        MCP       `json:"mcp" yaml:"mcp"`
	Trial      Trial     `json:"trial" yaml:"trial"`
	Evidence   Evidence  `json:"evidence" yaml:"evidence"`
	Redaction  Redaction `json:"redaction" yaml:"redaction"`
}

type Metadata struct {
	Name      string `json:"name" yaml:"name"`
	CreatedAt string `json:"createdAt" yaml:"createdAt"`
}

type Claim struct {
	Level           string `json:"level" yaml:"level"`
	ValidUntil      string `json:"validUntil,omitempty" yaml:"validUntil,omitempty"`
	ProcedureDigest string `json:"procedureDigest" yaml:"procedureDigest"`
}

type Runtime struct {
	Version        string `json:"version" yaml:"version"`
	Revision       string `json:"revision" yaml:"revision"`
	SpecDigest     string `json:"specDigest" yaml:"specDigest"`
	SurfaceDigest  string `json:"surfaceDigest" yaml:"surfaceDigest"`
	SnapshotDigest string `json:"snapshotDigest" yaml:"snapshotDigest"`
}

type Host struct {
	Profile        string `json:"profile" yaml:"profile"`
	Name           string `json:"name" yaml:"name"`
	Version        string `json:"version" yaml:"version"`
	Provider       string `json:"provider" yaml:"provider"`
	RequestedModel string `json:"requestedModel,omitempty" yaml:"requestedModel,omitempty"`
	Model          string `json:"model" yaml:"model"`
}

type MCP struct {
	ConfiguredVersion       string `json:"configuredVersion" yaml:"configuredVersion"`
	NegotiatedVersion       string `json:"negotiatedVersion" yaml:"negotiatedVersion"`
	Transport               string `json:"transport" yaml:"transport"`
	EndpointTrust           string `json:"endpointTrust" yaml:"endpointTrust"`
	DeploymentProfileDigest string `json:"deploymentProfileDigest,omitempty" yaml:"deploymentProfileDigest,omitempty"`
	ObservedSurfaceDigest   string `json:"observedSurfaceDigest" yaml:"observedSurfaceDigest"`
	SurfaceStatus           string `json:"surfaceStatus" yaml:"surfaceStatus"`
}

type Trial struct {
	ScenarioDigest   string `json:"scenarioDigest" yaml:"scenarioDigest"`
	PromptDigest     string `json:"promptDigest" yaml:"promptDigest"`
	ToolPolicyDigest string `json:"toolPolicyDigest" yaml:"toolPolicyDigest"`
	Index            int    `json:"index" yaml:"index"`
	Limits           Limits `json:"limits" yaml:"limits"`
	Outcome          string `json:"outcome" yaml:"outcome"`
}

type Limits struct {
	ProviderRequests          int `json:"providerRequests" yaml:"providerRequests"`
	ToolCalls                 int `json:"toolCalls" yaml:"toolCalls"`
	WallTimeMS                int `json:"wallTimeMs" yaml:"wallTimeMs"`
	MaxTraceBytes             int `json:"maxTraceBytes" yaml:"maxTraceBytes"`
	RetriesPerProviderRequest int `json:"retriesPerProviderRequest" yaml:"retriesPerProviderRequest"`
	RetriesPerToolCall        int `json:"retriesPerToolCall" yaml:"retriesPerToolCall"`
	RepeatedIdenticalCalls    int `json:"repeatedIdenticalCalls" yaml:"repeatedIdenticalCalls"`
}

type Evidence struct {
	EnvironmentDigest       string   `json:"environmentDigest" yaml:"environmentDigest"`
	TerminalStateDigest     string   `json:"terminalStateDigest" yaml:"terminalStateDigest"`
	TraceDigest             string   `json:"traceDigest" yaml:"traceDigest"`
	ProviderRequestIDDigest string   `json:"providerRequestIdDigest,omitempty" yaml:"providerRequestIdDigest,omitempty"`
	Checks                  []string `json:"checks" yaml:"checks"`
	AssertionSummary        Summary  `json:"assertionSummary" yaml:"assertionSummary"`
}

type Summary struct {
	Passed int `json:"passed" yaml:"passed"`
	Failed int `json:"failed" yaml:"failed"`
}

type Redaction struct {
	Policy          string `json:"policy" yaml:"policy"`
	SecretsDetected bool   `json:"secretsDetected" yaml:"secretsDetected"`
}

func Load(path string) (*Report, error) {
	data, err := readDeclaration(path, MaxReportBytes, "HOST_REPORT")
	if err != nil {
		return nil, err
	}
	return Decode(data)
}

func Decode(data []byte) (*Report, error) {
	if len(data) > MaxReportBytes {
		return nil, errors.New("HOST_REPORT_RESOURCE_LIMIT")
	}
	if logging.ContainsSensitive(string(data)) {
		return nil, errors.New("host compatibility report contains a credential-like, private-key, or email pattern")
	}
	var report Report
	if err := strictyaml.DecodeOne(data, MaxReportBytes, "HostCompatibilityReport", &report); err != nil {
		// Parser diagnostics may echo a decoded YAML key or scalar before the
		// typed-value privacy scan can run. Keep admission failures content-free.
		return nil, errors.New("HOST_REPORT_DECODE_INVALID")
	}
	if err := explicitScalarFields(data, true); err != nil {
		return nil, err
	}
	if err := report.Validate(); err != nil {
		return nil, err
	}
	return &report, nil
}

func (r *Report) Digest() (string, error) {
	return canonical.Digest(r)
}

func (r *Report) Validate() error {
	if r == nil {
		return errors.New("HostCompatibilityReport is required")
	}
	// YAML quoted scalars may hide credential patterns from the raw-byte scan.
	// Check decoded values too; never include those values in the error.
	encoded, err := json.Marshal(r)
	if err != nil || len(encoded) > MaxReportBytes {
		return errors.New("HOST_REPORT_RESOURCE_LIMIT")
	}
	if logging.ContainsSensitive(string(encoded)) {
		return errors.New("HOST_REPORT_SENSITIVE_CONTENT")
	}
	var problems []string
	requireEqual(&problems, "apiVersion", r.APIVersion, APIVersion)
	requireEqual(&problems, "kind", r.Kind, Kind)
	requireEqual(&problems, "format", r.Format, Format)
	if !identifierPattern.MatchString(r.Metadata.Name) {
		problems = append(problems, "metadata.name is invalid")
	}
	createdAt, createdErr := requireUTC(&problems, "metadata.createdAt", r.Metadata.CreatedAt)

	if !oneOf(r.Claim.Level, "experimental", "verified", "regressed") {
		problems = append(problems, "claim.level must be experimental, verified, or regressed")
	}
	requireDigest(&problems, "claim.procedureDigest", r.Claim.ProcedureDigest)
	if r.Claim.Level == "verified" || r.Claim.ValidUntil != "" {
		validUntil, err := requireUTC(&problems, "claim.validUntil", r.Claim.ValidUntil)
		if err == nil && createdErr == nil && !validUntil.After(createdAt) {
			problems = append(problems, "claim.validUntil must be after metadata.createdAt")
		}
	}

	validateRuntime(&problems, r.Runtime, r.Claim.Level == "verified")
	for name, value := range map[string]string{
		"mcp.observedSurfaceDigest": r.MCP.ObservedSurfaceDigest,
		"trial.scenarioDigest":      r.Trial.ScenarioDigest, "trial.promptDigest": r.Trial.PromptDigest,
		"trial.toolPolicyDigest": r.Trial.ToolPolicyDigest, "evidence.environmentDigest": r.Evidence.EnvironmentDigest,
		"evidence.terminalStateDigest": r.Evidence.TerminalStateDigest, "evidence.traceDigest": r.Evidence.TraceDigest,
	} {
		requireDigest(&problems, name, value)
	}

	validateHost(&problems, r)
	validateMCP(&problems, r)
	validateTrial(&problems, r)
	validateEvidence(&problems, r)
	if r.Redaction.Policy != "synthetic-only-v1" {
		problems = append(problems, "redaction.policy must be synthetic-only-v1")
	}
	if r.Redaction.SecretsDetected {
		problems = append(problems, "redaction.secretsDetected must be false")
	}

	if len(problems) != 0 {
		sort.Strings(problems)
		return errors.New("invalid HostCompatibilityReport: " + strings.Join(problems, "; "))
	}
	return nil
}

func validateRuntime(problems *[]string, r Runtime, verified bool) {
	requireIdentity(problems, "runtime.version", r.Version)
	if verified && placeholder(r.Version) {
		*problems = append(*problems, "runtime.version must identify the observed runtime version")
	}
	if !revisionPattern.MatchString(r.Revision) {
		*problems = append(*problems, "runtime.revision must be an immutable 40- or 64-character lowercase hexadecimal revision")
	}
	requireDigest(problems, "runtime.specDigest", r.SpecDigest)
	requireDigest(problems, "runtime.surfaceDigest", r.SurfaceDigest)
	requireDigest(problems, "runtime.snapshotDigest", r.SnapshotDigest)
}

func validateHost(problems *[]string, r *Report) {
	profiles := map[string]string{
		"generic-mcp": "none", "openai-api-mcp": "openai", "chatgpt-mcp": "openai",
		"anthropic-api-mcp": "anthropic", "claude-code-mcp": "anthropic", "custom-mcp": "",
	}
	wantProvider, valid := profiles[r.Host.Profile]
	if !valid {
		*problems = append(*problems, "host.profile is not supported by this report version")
	}
	for name, value := range map[string]string{
		"host.name": r.Host.Name, "host.version": r.Host.Version,
		"host.provider": r.Host.Provider, "host.model": r.Host.Model,
	} {
		requireIdentity(problems, name, value)
	}
	if r.Host.RequestedModel != "" {
		requireIdentity(problems, "host.requestedModel", r.Host.RequestedModel)
	}
	if r.Claim.Level == "verified" && placeholder(r.Host.Version) {
		*problems = append(*problems, "host.version must identify the observed host version")
	}
	if valid && wantProvider != "" && r.Host.Provider != wantProvider {
		*problems = append(*problems, fmt.Sprintf("host.provider must be %q for profile %q", wantProvider, r.Host.Profile))
	}
	if r.Host.Profile == "generic-mcp" && (r.Host.Provider != "none" || r.Host.Model != "none") {
		*problems = append(*problems, "generic-mcp reports must use provider and model value none")
	}
	if oneOf(r.Host.Profile, "openai-api-mcp", "chatgpt-mcp", "anthropic-api-mcp", "claude-code-mcp") {
		if strings.TrimSpace(r.Host.Model) == "" || strings.EqualFold(r.Host.Model, "none") {
			*problems = append(*problems, "provider profiles must record a model identifier or unknown")
		} else if r.Claim.Level == "verified" && placeholder(r.Host.Model) {
			*problems = append(*problems, "provider profiles must record the resolved model identifier")
		}
	}
	if r.Host.Profile == "custom-mcp" && r.Claim.Level == "verified" && r.Host.Provider != "none" && (placeholder(r.Host.Provider) || placeholder(r.Host.Model)) {
		*problems = append(*problems, "verified custom provider reports require non-placeholder provider and model identities")
	}
}

func validateMCP(problems *[]string, r *Report) {
	for _, version := range []string{r.MCP.ConfiguredVersion, r.MCP.NegotiatedVersion} {
		if _, err := time.Parse(time.DateOnly, version); err != nil || !protocolPattern.MatchString(version) {
			*problems = append(*problems, "MCP versions must be calendar-valid YYYY-MM-DD dates")
			break
		}
	}
	if r.MCP.Transport != "streamable-http" {
		*problems = append(*problems, "mcp.transport must be streamable-http for the current profile")
	}
	if !oneOf(r.MCP.EndpointTrust, "loopback", "private", "public") {
		*problems = append(*problems, "mcp.endpointTrust must be loopback, private, or public")
	}
	remoteProfile := oneOf(r.Host.Profile, "openai-api-mcp", "chatgpt-mcp", "anthropic-api-mcp")
	if remoteProfile && r.MCP.EndpointTrust == "loopback" {
		*problems = append(*problems, "provider-hosted profiles cannot claim a loopback endpoint")
	}
	if r.MCP.EndpointTrust != "loopback" || r.MCP.DeploymentProfileDigest != "" {
		requireDigest(problems, "mcp.deploymentProfileDigest", r.MCP.DeploymentProfileDigest)
	}
	if !oneOf(r.MCP.SurfaceStatus, "exact", "modified") {
		*problems = append(*problems, "mcp.surfaceStatus must be exact or modified")
	}
	if r.Claim.Level == "verified" && r.MCP.SurfaceStatus != "exact" {
		*problems = append(*problems, "verified reports require an exact observed tool surface")
	}
	if r.MCP.SurfaceStatus == "exact" && r.MCP.ObservedSurfaceDigest != r.Runtime.SurfaceDigest {
		*problems = append(*problems, "exact observed surface must match runtime.surfaceDigest")
	}
}

func validateTrial(problems *[]string, r *Report) {
	if r.Trial.Index < 0 {
		*problems = append(*problems, "trial.index must not be negative")
	}
	if !oneOf(r.Trial.Outcome, "completed", "assertion_failed", "model_stopped", "tool_budget_exceeded", "provider_budget_exceeded", "timeout", "provider_error", "protocol_error", "runtime_error", "cancelled") {
		*problems = append(*problems, "trial.outcome is not a canonical outcome")
	}
	validateLimits(problems, r.Host.Profile, r.Trial.Limits)
	if r.Claim.Level == "verified" && r.Trial.Outcome != "completed" {
		*problems = append(*problems, "verified reports require trial.outcome completed")
	}
}

func validateLimits(problems *[]string, profile string, limits Limits) {
	if limits.ToolCalls <= 0 || limits.WallTimeMS <= 0 || limits.MaxTraceBytes <= 0 || limits.RepeatedIdenticalCalls <= 0 {
		*problems = append(*problems, "trial limits for tool calls, wall time, trace bytes, and repeated calls must be positive")
	}
	if limits.ProviderRequests < 0 || limits.RetriesPerProviderRequest < 0 || limits.RetriesPerToolCall < 0 {
		*problems = append(*problems, "trial provider and retry limits must not be negative")
	}
	providerAPI := oneOf(profile, "openai-api-mcp", "anthropic-api-mcp")
	if providerAPI && limits.ProviderRequests <= 0 {
		*problems = append(*problems, "API provider profiles require a positive provider request limit")
	}
}

func validateEvidence(problems *[]string, r *Report) {
	if r.Evidence.AssertionSummary.Passed < 0 || r.Evidence.AssertionSummary.Failed < 0 {
		*problems = append(*problems, "assertion counts must not be negative")
	}
	if r.Claim.Level == "verified" && r.Evidence.AssertionSummary.Failed != 0 {
		*problems = append(*problems, "verified reports cannot contain failed assertions")
	}
	if r.Claim.Level == "verified" && r.Evidence.AssertionSummary.Passed == 0 {
		*problems = append(*problems, "verified reports require successful assertions")
	}
	providerAPI := oneOf(r.Host.Profile, "openai-api-mcp", "anthropic-api-mcp")
	if providerAPI {
		requireDigest(problems, "evidence.providerRequestIdDigest", r.Evidence.ProviderRequestIDDigest)
	} else if r.Evidence.ProviderRequestIDDigest != "" {
		requireDigest(problems, "evidence.providerRequestIdDigest", r.Evidence.ProviderRequestIDDigest)
	}

	seen := make(map[string]struct{}, len(r.Evidence.Checks))
	for _, check := range r.Evidence.Checks {
		if len(check) > 128 || !identifierPattern.MatchString(check) {
			*problems = append(*problems, "evidence.checks must contain bounded identifiers")
		}
		if _, exists := seen[check]; exists {
			*problems = append(*problems, "evidence.checks must not contain duplicates")
		}
		seen[check] = struct{}{}
	}
	if _, yes := seen["cancellation"]; yes {
		if _, no := seen["cancellation-unsupported"]; no {
			*problems = append(*problems, "evidence.checks has contradictory cancellation claims")
		}
	}
	var required []string
	if r.Host.Profile == "generic-mcp" || r.Host.Profile == "custom-mcp" {
		required = []string{"initialize", "ping", "tools-list", "read-only-call", "state-changing-call", "invalid-input", "domain-error", "unknown-tool", "control-tools-hidden", "surface-digest", "branch-isolation"}
		if _, cancelled := seen["cancellation"]; !cancelled {
			if _, unsupported := seen["cancellation-unsupported"]; !unsupported {
				*problems = append(*problems, "generic MCP evidence requires cancellation or cancellation-unsupported")
			}
		}
	} else {
		required = []string{"tool-discovery", "read-only-call", "state-changing-scenario", "domain-error", "terminal-assertions", "bounded-termination", "artifact-redaction"}
	}
	for _, check := range required {
		if _, exists := seen[check]; !exists {
			*problems = append(*problems, "evidence.checks is missing "+check)
		}
	}
}

func requireEqual(problems *[]string, name, got, want string) {
	if got != want {
		*problems = append(*problems, fmt.Sprintf("%s must be %q", name, want))
	}
}

func requireDigest(problems *[]string, name, value string) {
	if !digestPattern.MatchString(value) {
		*problems = append(*problems, name+" must be a lowercase sha256 digest")
	}
}

func requireUTC(problems *[]string, name, value string) (time.Time, error) {
	// time.Parse alone accepts non-RFC3339 spellings and silently truncates
	// sub-nanosecond precision. Reject them before eligibility comparisons.
	if !utcPattern.MatchString(value) {
		*problems = append(*problems, name+" must be an RFC3339 UTC timestamp ending in Z with at most 9 fractional digits")
		return time.Time{}, errors.New("invalid timestamp")
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		*problems = append(*problems, name+" must be a calendar-valid RFC3339 UTC timestamp")
		return time.Time{}, errors.New("invalid timestamp")
	}
	return parsed, nil
}

func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

func requireIdentity(problems *[]string, name, value string) {
	if value == "" || len(value) > 256 || strings.TrimSpace(value) != value || strings.ContainsFunc(value, unicode.IsControl) {
		*problems = append(*problems, name+" must be a nonempty, unpadded identity of at most 256 bytes without controls")
	}
}

func placeholder(value string) bool {
	return oneOf(strings.ToLower(strings.TrimSpace(value)), "latest", "auto", "unknown", "none")
}
