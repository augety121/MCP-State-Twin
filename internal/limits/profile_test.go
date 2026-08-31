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
	if profile.MaxCassetteBytes != 0 {
		t.Fatal("disabled cassette feature must use zero, not an accidental unlimited value")
	}
	if profile.MaxScheduledEvents != 1024 || profile.MaxScheduledDelivery != 256 {
		t.Fatal("enabled scheduler limits are not bound into the resource profile")
	}
	if profile.MaxEntropyStreams != 64 || profile.MaxEntropyBytes != 32 {
		t.Fatal("enabled entropy limits are not bound into the resource profile")
	}
	if profile.MaxBundleFiles != 128 || profile.MaxBundleCompressed != 32<<20 || profile.MaxBundleExtracted != 64<<20 {
		t.Fatal("enabled TwinBundle limits are not bound into the resource profile")
	}
	if profile.MaxBundleMember != 16<<20 {
		t.Fatal("enabled TwinBundle member limit is not bound into the resource profile")
	}
	if profile.MaxEpisodeRecords != 10_000 {
		t.Fatal("enabled Episode record limit is not bound into the resource profile")
	}
	if profile.MaxEpisodeAttempts != 16 || profile.MaxLeaseSeconds != 3_600 {
		t.Fatal("enabled remote Episode limits are not bound into the resource profile")
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
