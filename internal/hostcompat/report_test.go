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
	if _, err := Decode(encoded); err == nil || !strings.Contains(err.Error(), "field unknownField not found") {
		t.Fatalf("unknown field error = %v", err)
	}
}

func TestReportLoadRejectsOversizedFileBeforeDecode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "oversized.yaml")
	if err := os.WriteFile(path, []byte(strings.Repeat("x", MaxReportBytes+1)), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil || !strings.Contains(err.Error(), "exceeds") {
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
