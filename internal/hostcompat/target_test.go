package hostcompat

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func integer(n int) *int { return &n }

// Only synthetic unit tests derive targets from their reports. Production CLI
// requires an independent operator input, never copies evidence into a target.
func targetForReport(r *Report) *Target {
	rt, host, mcp := r.Runtime, r.Host, r.MCP
	l := r.Trial.Limits
	return &Target{
		APIVersion: APIVersion, Kind: TargetKind, Format: TargetFormat,
		Runtime: &rt, Host: &host, MCP: &mcp, ProcedureDigest: r.Claim.ProcedureDigest,
		Trial: &TargetTrial{ScenarioDigest: r.Trial.ScenarioDigest, PromptDigest: r.Trial.PromptDigest, ToolPolicyDigest: r.Trial.ToolPolicyDigest,
			Limits: &TargetLimits{ProviderRequests: integer(l.ProviderRequests), ToolCalls: integer(l.ToolCalls), WallTimeMS: integer(l.WallTimeMS), MaxTraceBytes: integer(l.MaxTraceBytes), RetriesPerProviderRequest: integer(l.RetriesPerProviderRequest), RetriesPerToolCall: integer(l.RetriesPerToolCall), RepeatedIdenticalCalls: integer(l.RepeatedIdenticalCalls)}},
		RedactionPolicy: r.Redaction.Policy,
	}
}

func TestTargetRoundTripAllProfiles(t *testing.T) {
	for _, profile := range []string{"generic-mcp", "custom-mcp", "openai-api-mcp", "anthropic-api-mcp", "chatgpt-mcp", "claude-code-mcp"} {
		want := targetForReport(reportForProfile(profile))
		for _, marshal := range []func(any) ([]byte, error){json.Marshal, yaml.Marshal} {
			raw, err := marshal(want)
			if err != nil {
				t.Fatal(err)
			}
			got, err := DecodeTarget(raw)
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Fatal(profile, err)
			}
		}
	}
}

func TestTargetRequiresEveryBudgetAndObject(t *testing.T) {
	want := targetForReport(validGenericReport())
	raw, _ := json.Marshal(want)
	fields := reflect.TypeOf(TargetLimits{})
	for i := 0; i < fields.NumField(); i++ {
		name := fields.Field(i).Tag.Get("json")
		for _, omit := range []bool{true, false} {
			var value map[string]any
			if err := json.Unmarshal(raw, &value); err != nil {
				t.Fatal(err)
			}
			limits := value["trial"].(map[string]any)["limits"].(map[string]any)
			if omit {
				delete(limits, name)
			} else {
				limits[name] = nil
			}
			b, _ := json.Marshal(value)
			if r, err := DecodeTarget(b); err == nil || r != nil {
				t.Fatalf("missing/null %s admitted", name)
			}
		}
	}
	for _, key := range []string{"runtime", "host", "mcp", "trial", "procedureDigest", "redactionPolicy", "apiVersion", "kind", "format"} {
		var value map[string]any
		if err := json.Unmarshal(raw, &value); err != nil {
			t.Fatal(err)
		}
		delete(value, key)
		b, _ := json.Marshal(value)
		if r, err := DecodeTarget(b); err == nil || r != nil {
			t.Fatalf("missing %s admitted", key)
		}
	}
	if err := (*Target)(nil).Validate(); err == nil {
		t.Fatal("nil target admitted")
	}
	want.Trial.Limits = nil
	if err := want.Validate(); err == nil {
		t.Fatal("missing limits object admitted")
	}
}

func TestTargetRejectsUnsupportedOrSensitiveDocuments(t *testing.T) {
	base := targetForReport(validGenericReport())
	raw, _ := yaml.Marshal(base)
	for _, suffix := range []string{"adapter: synthetic-v1\n", "claim: {level: verified}\n", "unknown: true\n", "---\n{}\n", "host: {}\n"} {
		if r, err := DecodeTarget(append(append([]byte{}, raw...), []byte(suffix)...)); err == nil || r != nil {
			t.Fatal("unsupported target admitted")
		}
	}
	for _, raw := range []string{"", "null", "[]", "host: &h {}\n", "host: !!map {}\n", `host: {name: "\x73\x6b-` + strings.Repeat("a", 24) + `"}`, `"\x73\x6b-` + strings.Repeat("a", 24) + `": true`} {
		if target, err := DecodeTarget([]byte(raw)); err == nil || target != nil || strings.Contains(err.Error(), strings.Repeat("a", 24)) {
			t.Fatal("unsafe input accepted or echoed")
		}
	}
	base.Host.Name = "sensitive-marker"
	raw, _ = yaml.Marshal(base)
	raw = []byte(strings.Replace(string(raw), "sensitive-marker", `"\x73\x6b-`+strings.Repeat("a", 24)+`"`, 1))
	if target, err := DecodeTarget(raw); err == nil || target != nil || err.Error() != "HOST_TARGET_SENSITIVE_CONTENT" {
		t.Fatal("decoded-value bypass", err)
	}
	base.Host.Name = "sk-" + strings.Repeat("a", 24)
	if err := base.Validate(); err == nil || err.Error() != "HOST_TARGET_SENSITIVE_CONTENT" {
		t.Fatal("direct-call bypass", err)
	}
	base.Host.Name = strings.Repeat("a", MaxTargetBytes)
	if err := base.Validate(); err == nil || err.Error() != "HOST_TARGET_RESOURCE_LIMIT" {
		t.Fatal("typed size bypass", err)
	}
	if target, err := DecodeTarget([]byte(strings.Repeat(" ", MaxTargetBytes+1))); err == nil || target != nil {
		t.Fatal("raw size bypass")
	}
}

func TestTargetIdentityAdmission(t *testing.T) {
	for _, mutate := range []func(*Target){
		func(s *Target) { s.Runtime.Revision = "main" },
		func(s *Target) { s.Host.Version = "unknown" },
		func(s *Target) { s.Host.Name = " padded" },
		func(s *Target) { s.MCP.NegotiatedVersion = "2026-02-30" },
		func(s *Target) { s.MCP.ObservedSurfaceDigest = digest("f") },
		func(s *Target) { s.MCP.EndpointTrust = "public" },
		func(s *Target) { s.MCP.SurfaceStatus = "modified" },
		func(s *Target) { s.Trial.Limits.ToolCalls = integer(0) },
		func(s *Target) { s.Trial.Limits.ProviderRequests = integer(-1) },
		func(s *Target) { s.Trial.Limits.RetriesPerToolCall = integer(-1) },
		func(s *Target) { s.RedactionPolicy = "unknown" },
	} {
		target := targetForReport(validGenericReport())
		mutate(target)
		if err := target.Validate(); err == nil {
			t.Fatal("invalid target identity admitted")
		}
	}
}

func TestTargetBudgetNumbersAreIntegers(t *testing.T) {
	raw, err := yaml.Marshal(targetForReport(validGenericReport()))
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"32.5", "32.0", "3.2e1", `"32"`, "true", "null"} {
		t.Run(value, func(t *testing.T) {
			input := strings.Replace(string(raw), "toolCalls: 32", "toolCalls: "+value, 1)
			if r, err := DecodeTarget([]byte(input)); err == nil || r != nil {
				t.Fatal("non-integer budget silently coerced")
			}
		})
	}
}

func TestReportRequiresExplicitIntegerFieldsInJSON(t *testing.T) {
	raw, _ := json.Marshal(validGenericReport())
	paths := [][]string{{"trial", "index"}, {"evidence", "assertionSummary", "passed"}, {"evidence", "assertionSummary", "failed"}}
	for _, name := range limitFields {
		paths = append(paths, []string{"trial", "limits", name})
	}
	for _, path := range paths {
		for _, bad := range []any{nil, 0.5, "0", false, "omit"} {
			var value map[string]any
			if err := json.Unmarshal(raw, &value); err != nil {
				t.Fatal(err)
			}
			parent := value
			for _, part := range path[:len(path)-1] {
				parent = parent[part].(map[string]any)
			}
			key := path[len(path)-1]
			if bad == "omit" {
				delete(parent, key)
			} else {
				parent[key] = bad
			}
			input, _ := json.Marshal(value)
			if r, err := Decode(input); err == nil || r != nil {
				t.Fatal("invalid integer field", path)
			}
		}
	}
}

func TestTargetFileAdmission(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "target.yaml")
	raw, _ := yaml.Marshal(targetForReport(validGenericReport()))
	if err := os.WriteFile(good, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadTarget(good); err != nil {
		t.Fatal(err)
	}
	for _, file := range []string{dir, filepath.Join(dir, "missing")} {
		if target, err := LoadTarget(file); err == nil || target != nil || strings.Contains(err.Error(), dir) {
			t.Fatal("unsafe file admitted or echoed")
		}
	}
	large := filepath.Join(dir, "large")
	f, err := os.Create(large)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(MaxTargetBytes + 1); err != nil {
		f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if target, err := LoadTarget(large); err == nil || target != nil {
		t.Fatal("oversized file admitted")
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(good, link); err != nil {
		if runtime.GOOS == "windows" {
			t.Skip("Windows symlink privileges unavailable")
		}
		t.Fatal(err)
	}
	if target, err := LoadTarget(link); err == nil || target != nil {
		t.Fatal("symlink target admitted")
	}
}
