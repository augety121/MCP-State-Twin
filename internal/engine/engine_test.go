package engine

import (
	"context"
	"fmt"
	"testing"

	"github.com/augety121/mcp-state-twin/internal/spec"
	"github.com/augety121/mcp-state-twin/internal/store"
	"github.com/augety121/mcp-state-twin/internal/world"
)

func boolPtr(v bool) *bool { return &v }

func testSpec() *spec.TwinSpec {
	return &spec.TwinSpec{
		APIVersion: spec.APIVersion,
		Kind:       spec.Kind,
		Metadata: spec.Metadata{
			Name:     "items",
			Upstream: spec.UpstreamMetadata{Protocol: "mcp", Status: "current", SurfaceDigest: "sha256:surface"},
			Fidelity: spec.FidelityMetadata{Level: "L1", Status: "unverified"},
		},
		Clock:      spec.ClockSpec{Mode: "virtual", Initial: "2026-08-01T00:00:00Z"},
		State:      spec.StateSpec{Entities: map[string]spec.EntitySpec{"item": {Key: []string{"id"}}}},
		Invariants: []spec.InvariantSpec{{ID: "non-empty", Assert: "state.entities.item.all(k, state.entities.item[k].id != '')"}},
		Tools: []spec.ToolSpec{
			{
				Name: "create_item", Description: "Create an item.", Modeled: boolPtr(true),
				InputSchema: map[string]any{
					"type": "object", "required": []any{"name"}, "additionalProperties": false,
					"properties": map[string]any{"name": map[string]any{"type": "string"}},
				},
				Effects: []spec.Effect{
					{Op: "allocate", Sequence: "item", As: "id"},
					{Op: "insert", Entity: "item", Key: "string(vars.id)", Value: "{'id': string(vars.id), 'name': input.name}"},
				},
				Query:  &spec.Query{Entity: "item", Key: "string(vars.id)", As: "created"},
				Result: "{'item': vars.created}",
			},
		},
	}
}

func TestDeterministicCreateAndInvalidInput(t *testing.T) {
	ctx := context.Background()
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	twin := testSpec()
	if err := twin.Validate(); err != nil {
		t.Fatal(err)
	}
	runtime, err := New(twin, s)
	if err != nil {
		t.Fatal(err)
	}
	if err := runtime.Initialize(ctx, "main", world.New()); err != nil {
		t.Fatal(err)
	}
	created, err := runtime.Call(ctx, "main", "create_item", map[string]any{"name": "first"})
	if err != nil {
		t.Fatal(err)
	}
	if created.ErrorClass != "" {
		t.Fatalf("unexpected domain error: %#v", created.Result)
	}
	failed, err := runtime.Call(ctx, "main", "create_item", map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if failed.ErrorClass != "INVALID_INPUT" {
		t.Fatalf("error class = %q", failed.ErrorClass)
	}
	if failed.BeforeDigest != failed.AfterDigest {
		t.Fatal("invalid input changed state")
	}
}

func TestThousandCallCorpusReplaysToSameDigest(t *testing.T) {
	ctx := context.Background()
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	runtime, err := New(testSpec(), s)
	if err != nil {
		t.Fatal(err)
	}
	if err := runtime.Initialize(ctx, "main", world.New()); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateSnapshot(ctx, "base", "main"); err != nil {
		t.Fatal(err)
	}
	if err := s.Fork(ctx, "base", "replay-a"); err != nil {
		t.Fatal(err)
	}
	if err := s.Fork(ctx, "base", "replay-b"); err != nil {
		t.Fatal(err)
	}
	for i := range 1000 {
		input := map[string]any{"name": fmt.Sprintf("item-%04d", i)}
		a, err := runtime.Call(ctx, "replay-a", "create_item", input)
		if err != nil {
			t.Fatal(err)
		}
		b, err := runtime.Call(ctx, "replay-b", "create_item", input)
		if err != nil {
			t.Fatal(err)
		}
		if a.AfterDigest != b.AfterDigest {
			t.Fatalf("digest diverged at call %d: %s != %s", i+1, a.AfterDigest, b.AfterDigest)
		}
	}
	a, err := s.Branch(ctx, "replay-a")
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.Branch(ctx, "replay-b")
	if err != nil {
		t.Fatal(err)
	}
	if a.StateDigest != b.StateDigest {
		t.Fatalf("final state digest differs: %s != %s", a.StateDigest, b.StateDigest)
	}
}

func TestExplicitlyUnmodeledToolFailsWithoutStateChange(t *testing.T) {
	ctx := context.Background()
	twin := testSpec()
	twin.Tools[0].Modeled = boolPtr(false)
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	runtime, err := New(twin, s)
	if err != nil {
		t.Fatal(err)
	}
	if err := runtime.Initialize(ctx, "main", world.New()); err != nil {
		t.Fatal(err)
	}
	result, err := runtime.Call(ctx, "main", "create_item", map[string]any{"name": "first"})
	if err != nil {
		t.Fatal(err)
	}
	if result.ErrorClass != "UNMODELED_BEHAVIOR" {
		t.Fatalf("error class = %q, want UNMODELED_BEHAVIOR", result.ErrorClass)
	}
	if result.BeforeDigest != result.AfterDigest {
		t.Fatal("unmodeled behavior changed state")
	}
}
