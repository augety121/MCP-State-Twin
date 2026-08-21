package limits

import (
	"strings"
	"testing"
)

func TestDefaultProfileDigestIsStableAndBindsSemanticLimits(t *testing.T) {
	first, err := Digest()
	if err != nil {
		t.Fatal(err)
	}
	second, err := Digest()
	if err != nil {
		t.Fatal(err)
	}
	if first != second || !strings.HasPrefix(first, "sha256:") {
		t.Fatalf("profile digest is not stable: %q %q", first, second)
	}
	profile := Default()
	if profile.Format != Format || profile.Version != Version {
		t.Fatalf("profile identity = %#v", profile)
	}
	if profile.MaxScheduledEvents != 0 || profile.MaxBundleFiles != 0 {
		t.Fatal("disabled features must use zero, not an accidental unlimited value")
	}
}

func TestValidateJSONBoundsBytesDepthMembersAndDomain(t *testing.T) {
	if err := ValidateJSON(map[string]any{"ok": true}, 64); err != nil {
		t.Fatal(err)
	}
	if err := ValidateJSON(map[string]any{"payload": strings.Repeat("x", 64)}, 32); err == nil || !strings.Contains(err.Error(), "bytes") {
		t.Fatalf("byte limit error = %v", err)
	}
	value := any("leaf")
	for range MaxJSONDepth + 1 {
		value = []any{value}
	}
	if err := ValidateJSON(value, MaxStateBytes); err == nil || !strings.Contains(err.Error(), "depth") {
		t.Fatalf("depth limit error = %v", err)
	}
	many := make([]any, MaxJSONMembers+1)
	if err := ValidateJSON(many, MaxStateBytes*8); err == nil || !strings.Contains(err.Error(), "members") {
		t.Fatalf("member limit error = %v", err)
	}
}
