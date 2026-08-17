package spec

import "testing"

func minimalSpec() *TwinSpec {
	return &TwinSpec{
		APIVersion: APIVersion,
		Kind:       Kind,
		Metadata: Metadata{
			Name:     "example",
			Upstream: UpstreamMetadata{Protocol: "mcp", Status: "current", SurfaceDigest: "sha256:test"},
			Fidelity: FidelityMetadata{Level: "L1", Status: "unverified"},
		},
		Clock: ClockSpec{Mode: "virtual", Initial: "2026-08-01T00:00:00Z"},
		State: StateSpec{Entities: map[string]EntitySpec{"item": {Key: []string{"id"}}}},
		Tools: []ToolSpec{{
			Name: "get_item", Description: "Get an item.",
			InputSchema: map[string]any{"type": "object"},
		}},
	}
}

func TestValidateMinimal(t *testing.T) {
	if err := minimalSpec().Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestRejectsVerifiedL1(t *testing.T) {
	s := minimalSpec()
	s.Metadata.Fidelity.Status = "verified"
	if err := s.Validate(); err == nil {
		t.Fatal("expected invalid fidelity status")
	}
}

func TestDigestStable(t *testing.T) {
	s := minimalSpec()
	a, err := s.Digest()
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.Digest()
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatalf("digest changed: %s != %s", a, b)
	}
}
