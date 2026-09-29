package hostcompat

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestValidGenericMCPReport(t *testing.T) {
	report := validGenericReport()
	if err := report.Validate(); err != nil {
		t.Fatal(err)
	}
	first, err := report.Digest()
	if err != nil {
		t.Fatal(err)
	}
	second, err := report.Digest()
	if err != nil {
		t.Fatal(err)
	}
	if first != second || !strings.HasPrefix(first, "sha256:") {
		t.Fatalf("report digest is not deterministic: %q %q", first, second)
	}

	encoded, err := yaml.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := Decode(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Host.Profile != "generic-mcp" {
		t.Fatalf("decoded profile = %q", decoded.Host.Profile)
	}
}

func TestValidOpenAIAPIReportRequiresRemoteEvidence(t *testing.T) {
	report := validGenericReport()
	report.Host = Host{
		Profile: "openai-api-mcp", Name: "responses-api", Version: "2026-08-24",
		Provider: "openai", RequestedModel: "gpt-test", Model: "gpt-test-2026-08-24",
	}
	report.MCP.EndpointTrust = "public"
	report.MCP.DeploymentProfileDigest = digest("d")
	report.Trial.Limits.ProviderRequests = 2
	report.Evidence.ProviderRequestIDDigest = digest("e")
	report.Evidence.Checks = []string{
		"tool-discovery", "read-only-call", "state-changing-scenario", "domain-error",
		"terminal-assertions", "bounded-termination", "artifact-redaction",
	}
	if err := report.Validate(); err != nil {
		t.Fatal(err)
	}

	report.MCP.EndpointTrust = "loopback"
	report.MCP.DeploymentProfileDigest = ""
	if err := report.Validate(); err == nil || !strings.Contains(err.Error(), "cannot claim a loopback endpoint") {
		t.Fatalf("loopback provider report error = %v", err)
	}
}

func TestReportFailsClosedOnClaimsEvidenceAndSecrets(t *testing.T) {
	tests := []struct {
		name       string
		mutate     func(*Report)
		wantSubstr string
	}{
		{name: "mutable revision", mutate: func(r *Report) { r.Runtime.Revision = "main" }, wantSubstr: "immutable"},
		{name: "expired before run", mutate: func(r *Report) { r.Claim.ValidUntil = "2026-08-23T00:00:00Z" }, wantSubstr: "must be after"},
		{name: "modified verified surface", mutate: func(r *Report) { r.MCP.SurfaceStatus = "modified" }, wantSubstr: "exact observed"},
		{name: "failed verified assertion", mutate: func(r *Report) { r.Evidence.AssertionSummary.Failed = 1 }, wantSubstr: "failed assertions"},
		{name: "missing check", mutate: func(r *Report) { r.Evidence.Checks = r.Evidence.Checks[:1] }, wantSubstr: "missing"},
		{name: "secret flag", mutate: func(r *Report) { r.Redaction.SecretsDetected = true }, wantSubstr: "must be false"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			report := validGenericReport()
			test.mutate(report)
			err := report.Validate()
			if err == nil || !strings.Contains(err.Error(), test.wantSubstr) {
				t.Fatalf("Validate error = %v, want substring %q", err, test.wantSubstr)
			}
		})
	}

	secret := "api_key: sk-" + strings.Repeat("a", 24) + "\n"
	if _, err := Decode([]byte(secret)); err == nil || !strings.Contains(err.Error(), "credential-like") {
		t.Fatalf("secret admission error = %v", err)
	}
}

func TestReportDecoderRejectsUnknownFields(t *testing.T) {
	report := validGenericReport()
	encoded, err := yaml.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	encoded = append(encoded, []byte("unknownField: true\n")...)
	if _, err := Decode(encoded); err == nil || err.Error() != "HOST_REPORT_DECODE_INVALID" {
		t.Fatalf("unknown field error = %v", err)
	}
}

func TestReportCrossFieldAdmission(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Report)
	}{
		{"exact surface mismatch", func(r *Report) { r.MCP.ObservedSurfaceDigest = digest("f") }},
		{"zero successful assertions", func(r *Report) { r.Evidence.AssertionSummary.Passed = 0 }},
		{"contradictory cancellation", func(r *Report) { r.Evidence.Checks = append(r.Evidence.Checks, "cancellation") }},
		{"invalid calendar version", func(r *Report) { r.MCP.NegotiatedVersion = "2026-02-30" }},
		{"padded placeholder version", func(r *Report) { r.Host.Version = " unknown " }},
		{"unknown runtime version", func(r *Report) { r.Runtime.Version = "unknown" }},
		{"empty custom provider", func(r *Report) { r.Host.Profile = "custom-mcp"; r.Host.Provider = "" }},
		{"experimental backward expiry", func(r *Report) { r.Claim.Level = "experimental"; r.Claim.ValidUntil = r.Metadata.CreatedAt }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := validGenericReport()
			tt.mutate(r)
			if err := r.Validate(); err == nil {
				t.Fatal("inconsistent report admitted")
			}
		})
	}
}

func TestReportRejectsDecodedYAMLCredentials(t *testing.T) {
	r := validGenericReport()
	r.Host.Name = "synthetic-marker"
	raw, err := yaml.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	// The YAML spelling is harmless to a raw scan; the decoded value is not.
	secret := "sk-" + strings.Repeat("a", 24)
	raw = []byte(strings.Replace(string(raw), "synthetic-marker", `"\x73\x6b-`+strings.Repeat("a", 24)+`"`, 1))
	if _, err := Decode(raw); err == nil || strings.Contains(err.Error(), secret) {
		t.Fatal("decoded credential admitted or echoed")
	}
	for _, raw := range []string{
		`"\x73\x6b-` + strings.Repeat("a", 24) + `": true`,
		"trial:\n  index: \"\\x73\\x6b-" + strings.Repeat("a", 24) + "\"\n",
	} {
		if _, err := Decode([]byte(raw)); err == nil || err.Error() != "HOST_REPORT_DECODE_INVALID" {
			t.Fatal("parser error must not expose a decoded key or value")
		}
	}
}

func TestReportIdentityAndDecodeBounds(t *testing.T) {
	for _, value := range []string{"", " padded", "padded ", "line\nbreak", "tab\there", strings.Repeat("a", 257)} {
		r := validGenericReport()
		r.Host.Name = value
		if err := r.Validate(); err == nil {
			t.Fatal("invalid identity accepted")
		}
	}
	r := validGenericReport()
	r.Host.Name = strings.Repeat("a", 256)
	if err := r.Validate(); err != nil {
		t.Fatal("maximum identity rejected", err)
	}
	for _, value := range []string{"AUTO", "latest", "unknown", "none"} {
		r := reportForProfile("openai-api-mcp")
		r.Host.Model = value
		if err := r.Validate(); err == nil {
			t.Fatal("placeholder model accepted")
		}
	}
	for _, identity := range []Host{
		{Provider: "unknown", Model: "synthetic-v1"},
		{Provider: "synthetic-provider", Model: "unknown"},
	} {
		r := reportForProfile("custom-mcp")
		r.Host.Provider, r.Host.Model = identity.Provider, identity.Model
		if err := r.Validate(); err == nil {
			t.Fatal("custom provider placeholder accepted")
		}
	}
	if _, err := Decode([]byte(strings.Repeat(" ", MaxReportBytes+1))); err == nil {
		t.Fatal("oversized decode accepted")
	}
	r = validGenericReport()
	r.Host.Name = "sk-" + strings.Repeat("a", 24)
	if err := r.Validate(); err == nil || strings.Contains(err.Error(), r.Host.Name) {
		t.Fatal("direct caller privacy bypass")
	}
}

func TestReportAllowsExplicitExperimentalDifferences(t *testing.T) {
	r := validGenericReport()
	r.Claim.Level, r.Claim.ValidUntil = "experimental", ""
	r.MCP.SurfaceStatus, r.MCP.ObservedSurfaceDigest = "modified", digest("f")
	r.MCP.ConfiguredVersion, r.MCP.NegotiatedVersion = "2024-02-29", "2025-11-25"
	r.Evidence.AssertionSummary = Summary{}
	r.Evidence.Checks = append(r.Evidence.Checks, "additional-synthetic-check")
	if err := r.Validate(); err != nil {
		t.Fatal("explicit experimental difference rejected", err)
	}
}

func TestReportMalformedBoundariesStayContentFree(t *testing.T) {
	for _, input := range []string{"", "null", "[]", "apiVersion: one\napiVersion: two\n", "---\n---\n", "host: &h {}\n", "host: !!map {}\n"} {
		if r, err := Decode([]byte(input)); err == nil || r != nil {
			t.Fatal("malformed document admitted")
		}
	}
	r := validGenericReport()
	r.Evidence.Checks = append(r.Evidence.Checks, "bad check")
	if err := r.Validate(); err == nil {
		t.Fatal("non-identifier check admitted")
	}
	r.Evidence.Checks[len(r.Evidence.Checks)-1] = strings.Repeat("a", 129)
	if err := r.Validate(); err == nil {
		t.Fatal("oversized check admitted")
	}
	r.Host.Name = strings.Repeat("a", MaxReportBytes)
	if err := r.Validate(); err == nil || err.Error() != "HOST_REPORT_RESOURCE_LIMIT" {
		t.Fatal("oversized typed report admitted", err)
	}
}

func TestReportRejectsNumericCoercion(t *testing.T) {
	raw, err := yaml.Marshal(validGenericReport())
	if err != nil {
		t.Fatal(err)
	}
	for _, pair := range [][2]string{{"failed: 0", "failed: 0.5"}, {"passed: 12", "passed: 12.5"}, {"index: 0", "index: 0.5"}, {"toolCalls: 32", "toolCalls: 32.5"}} {
		t.Run(pair[1], func(t *testing.T) {
			input := strings.Replace(string(raw), pair[0], pair[1], 1)
			if r, err := Decode([]byte(input)); err == nil || r != nil {
				t.Fatal("fractional report counter/budget silently coerced")
			}
		})
	}
}

func TestReportLoadRejectsOversizedFileBeforeDecode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "oversized.yaml")
	if err := os.WriteFile(path, []byte(strings.Repeat("x", MaxReportBytes+1)), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil || err.Error() != "HOST_REPORT_FILE_INVALID" {
		t.Fatalf("oversized report error = %v", err)
	}
}

func validGenericReport() *Report {
	return &Report{
		APIVersion: APIVersion,
		Kind:       Kind,
		Format:     Format,
		Metadata: Metadata{
			Name: "generic-go-sdk", CreatedAt: "2026-08-24T00:00:00Z",
		},
		Claim: Claim{
			Level: "verified", ValidUntil: "2026-09-24T00:00:00Z", ProcedureDigest: digest("1"),
		},
		Runtime: Runtime{
			Version: "0.1.0-dev", Revision: strings.Repeat("a", 40), SpecDigest: digest("2"),
			SurfaceDigest: digest("3"), SnapshotDigest: digest("4"),
		},
		Host: Host{
			Profile: "generic-mcp", Name: "official-go-sdk", Version: "v1.7.0", Provider: "none", Model: "none",
		},
		MCP: MCP{
			ConfiguredVersion: "2026-07-28", NegotiatedVersion: "2026-07-28", Transport: "streamable-http",
			EndpointTrust: "loopback", ObservedSurfaceDigest: digest("3"), SurfaceStatus: "exact",
		},
		Trial: Trial{
			ScenarioDigest: digest("5"), PromptDigest: digest("6"), ToolPolicyDigest: digest("7"), Index: 0,
			Limits: Limits{
				ProviderRequests: 0, ToolCalls: 32, WallTimeMS: 60000, MaxTraceBytes: 1 << 20,
				RetriesPerProviderRequest: 0, RetriesPerToolCall: 1, RepeatedIdenticalCalls: 3,
			},
			Outcome: "completed",
		},
		Evidence: Evidence{
			EnvironmentDigest: digest("8"), TerminalStateDigest: digest("9"), TraceDigest: digest("a"),
			Checks: []string{
				"initialize", "ping", "tools-list", "read-only-call", "state-changing-call", "invalid-input",
				"domain-error", "unknown-tool", "control-tools-hidden", "surface-digest", "branch-isolation",
				"cancellation-unsupported",
			},
			AssertionSummary: Summary{Passed: 12, Failed: 0},
		},
		Redaction: Redaction{Policy: "synthetic-only-v1", SecretsDetected: false},
	}
}

func digest(character string) string {
	return "sha256:" + strings.Repeat(character, 64)
}
