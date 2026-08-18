package spec

import (
	"strings"
	"testing"
)

func minimalSpec() *TwinSpec {
	return &TwinSpec{
		APIVersion: APIVersion,
		Kind:       Kind,
		Metadata: Metadata{
			Name:     "example",
			Upstream: UpstreamMetadata{Protocol: "mcp", Status: "unbound"},
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

func TestSurfaceDigestIsOrderIndependent(t *testing.T) {
	s := minimalSpec()
	s.Tools = append(s.Tools, ToolSpec{
		Name: "delete_item", Description: "Delete an item.",
		InputSchema: map[string]any{"type": "object"},
		Effects:     []Effect{{Op: "delete", Entity: "item", Key: "input.id"}},
	})
	a, err := s.SurfaceDigest()
	if err != nil {
		t.Fatal(err)
	}
	s.Tools[0], s.Tools[1] = s.Tools[1], s.Tools[0]
	b, err := s.SurfaceDigest()
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatalf("tool order changed surface digest: %s != %s", a, b)
	}
}

func TestSurfaceDigestCoversDescriptionSchemaAndAnnotations(t *testing.T) {
	base := minimalSpec()
	original, err := base.SurfaceDigest()
	if err != nil {
		t.Fatal(err)
	}

	mutations := []func(*TwinSpec){
		func(s *TwinSpec) { s.Tools[0].Description = "Changed model-facing description." },
		func(s *TwinSpec) { s.Tools[0].InputSchema["additionalProperties"] = false },
		func(s *TwinSpec) {
			s.Tools[0].Effects = []Effect{{Op: "update", Entity: "item", Key: "input.id", Value: "input"}}
		},
	}
	for i, mutate := range mutations {
		candidate := minimalSpec()
		mutate(candidate)
		digest, digestErr := candidate.SurfaceDigest()
		if digestErr != nil {
			t.Fatal(digestErr)
		}
		if digest == original {
			t.Fatalf("mutation %d did not change surface digest", i)
		}
	}
}

func TestDecodeRejectsMultipleDocuments(t *testing.T) {
	_, err := Decode([]byte("apiVersion: statetwin.dev/v1alpha1\n---\nkind: Twin\n"))
	if err == nil || !strings.Contains(err.Error(), "multiple YAML documents") {
		t.Fatalf("expected multiple document rejection, got %v", err)
	}
}

func TestValidateRejectsMalformedSurfaceDigest(t *testing.T) {
	s := minimalSpec()
	s.Metadata.Upstream = UpstreamMetadata{Protocol: "mcp", Status: "current", SurfaceDigest: "sha256:not-a-digest"}
	if err := s.Validate(); err == nil || !strings.Contains(err.Error(), "lowercase sha256") {
		t.Fatalf("expected surface digest validation error, got %v", err)
	}
}

func TestValidateRejectsOversizedAndDeepSchemas(t *testing.T) {
	tooLarge := minimalSpec()
	tooLarge.Tools[0].InputSchema["description"] = strings.Repeat("x", MaxSchemaBytes)
	if err := tooLarge.Validate(); err == nil || !strings.Contains(err.Error(), "canonical size limit") {
		t.Fatalf("expected schema size rejection, got %v", err)
	}

	tooDeep := minimalSpec()
	var nested any = "leaf"
	for i := 0; i < MaxSchemaDepth+2; i++ {
		nested = map[string]any{"nested": nested}
	}
	tooDeep.Tools[0].InputSchema["nested"] = nested
	if err := tooDeep.Validate(); err == nil || !strings.Contains(err.Error(), "nesting limit") {
		t.Fatalf("expected schema depth rejection, got %v", err)
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
