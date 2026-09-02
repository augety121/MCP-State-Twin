package engine

import (
	"context"
	"strings"
	"testing"

	"github.com/augety121/mcp-state-twin/internal/canonical"
	"github.com/augety121/mcp-state-twin/internal/limits"
	"github.com/augety121/mcp-state-twin/internal/store"
	"github.com/augety121/mcp-state-twin/internal/world"
	"google.golang.org/protobuf/types/known/structpb"
)

// These vectors are reviewed compatibility examples, not a claim of exhaustive
// equivalence between CEL releases. Run them before and after a module upgrade.
func TestExpressionCompatibilityVectors(t *testing.T) {
	r, err := New(testSpec(), nil)
	if err != nil {
		t.Fatal(err)
	}
	activation := map[string]any{
		"input": map[string]any{"name": "synthetic", "n": int64(3)},
		"clock": "2026-08-01T00:00:00Z", "call_index": int64(7),
		"state": map[string]any{}, "vars": map[string]any{},
	}
	for _, test := range []struct{ expr, json string }{
		{"input.n * 2 + call_index", "13"},
		{"[1, 2, 3].map(x, x * x)", "[1,4,9]"},
		{"{'name': input.name, 'time': clock, 'present': has(input.name)}", `{"name":"synthetic","present":true,"time":"2026-08-01T00:00:00Z"}`},
		{"[1, 2, 3].all(x, x > 0)", "true"},
		{"has(input.missing) ? input.missing : null", "null"},
		{"{'value': null, 'nested': [null, {'value': null}], 'zero': 0}", `{"nested":[null,{"value":null}],"value":null,"zero":0}`},
		{"input.name.startsWith('syn') && size(input.name) == 9", "true"},
	} {
		t.Run(test.expr, func(t *testing.T) {
			if err := r.compile(test.expr); err != nil {
				t.Fatal(err)
			}
			got, err := r.eval(test.expr, activation)
			if err != nil {
				t.Fatal(err)
			}
			data, err := canonical.JSON(got)
			if err != nil || string(data) != test.json {
				t.Fatalf("result = %s, err = %v; want %s", data, err, test.json)
			}
		})
	}
	for _, expression := range []string{"input.missing", "1 / 0", "{1: 'non-string key'}"} {
		if err := r.compile(expression); err != nil {
			t.Fatal(err)
		}
		if _, err := r.eval(expression, activation); err == nil {
			t.Errorf("%s must fail, not manufacture a result", expression)
		}
	}
}

func TestExpressionNullSurvivesSchemaAndStoredState(t *testing.T) {
	twin := testSpec()
	twin.Tools[0].Effects[1].Value = "{'id': string(vars.id), 'name': input.name, 'optional': null}"
	twin.Tools[0].Result = "{'value': vars.created.optional}"
	twin.Tools[0].OutputSchema = map[string]any{
		"type": "object", "required": []any{"value"}, "additionalProperties": false,
		"properties": map[string]any{"value": map[string]any{"type": "null"}},
	}
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	r, err := New(twin, s)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := r.Initialize(ctx, "main", world.New()); err != nil {
		t.Fatal(err)
	}
	result, err := r.Call(ctx, "main", "create_item", map[string]any{"name": "synthetic"})
	if err != nil {
		t.Fatal(err)
	}
	if result.ErrorClass != "" {
		t.Fatalf("nullable output schema rejected result: %#v", result)
	}
	branch, err := s.Branch(ctx, "main")
	if err != nil {
		t.Fatal(err)
	}
	value, exists := branch.State.Entities["item"]["1"]["optional"]
	if !exists || value != nil {
		t.Fatalf("persisted null = %#v, exists=%v", value, exists)
	}
	if _, err := normalizeNative(structpb.NullValue(1)); err == nil {
		t.Fatal("unknown protobuf null enum must fail")
	}
}

func TestExpressionCompatibilityKeepsBoundsAndNoExternalFunctions(t *testing.T) {
	r, err := New(testSpec(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, expression := range []string{
		"readFile('/synthetic')", "http.get('https://example.invalid')",
		"exec('synthetic')", "now()", "random()", "input[",
		strings.Repeat("x", MaxExpressionBytes+1),
	} {
		if err := r.compile(expression); err == nil {
			t.Errorf("unexpected accepted expression: %.80s", expression)
		}
	}
	expression := "input.values.map(x, x + 1)"
	if err := r.compile(expression); err != nil {
		t.Fatal(err)
	}
	values := make([]int64, limits.MaxExpressionCost+1)
	_, err = r.eval(expression, map[string]any{"input": map[string]any{"values": values}})
	if err == nil || !strings.Contains(err.Error(), "cost limit") {
		t.Fatalf("evaluation must stop at the CEL cost limit, got %v", err)
	}
}
