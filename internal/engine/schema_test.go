package engine

import (
	"context"
	"strings"
	"testing"

	"github.com/augety121/mcp-state-twin/internal/store"
	"github.com/augety121/mcp-state-twin/internal/world"
)

func TestJSONSchema202012NestedAndFormatValidation(t *testing.T) {
	schema := map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []any{"metadata"},
		"properties": map[string]any{
			"metadata": map[string]any{
				"type":     "object",
				"required": []any{"owner", "createdAt"},
				"properties": map[string]any{
					"owner":     map[string]any{"type": "string", "minLength": 3},
					"createdAt": map[string]any{"type": "string", "format": "date-time"},
				},
			},
		},
	}
	compiled, err := compileJSONSchema("nested", "input", schema)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateJSON(map[string]any{"metadata": map[string]any{"owner": "ab", "createdAt": "not-a-date"}}, compiled); err == nil {
		t.Fatal("nested constraints and asserted format must reject invalid input")
	}
	valid := map[string]any{"metadata": map[string]any{"owner": "octo", "createdAt": "2026-08-17T00:00:00Z"}}
	if err := validateJSON(valid, compiled); err != nil {
		t.Fatalf("valid nested input rejected: %v", err)
	}
}

func TestJSONSchemaExternalReferenceFailsClosed(t *testing.T) {
	_, err := compileJSONSchema("external", "input", map[string]any{"$ref": "https://example.com/schema.json"})
	if err == nil || !strings.Contains(err.Error(), "external JSON Schema resource is not allowed") {
		t.Fatalf("external reference error = %v", err)
	}
}

func TestInvalidDeclaredOutputRollsBackState(t *testing.T) {
	ctx := context.Background()
	twin := testSpec()
	twin.Tools[0].OutputSchema = map[string]any{
		"type":     "object",
		"required": []any{"item"},
		"properties": map[string]any{
			"item": map[string]any{
				"type":     "object",
				"required": []any{"impossible"},
			},
		},
	}
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
	if result.ErrorClass != "INTERNAL_TWIN_ERROR" {
		t.Fatalf("error class = %q, want INTERNAL_TWIN_ERROR", result.ErrorClass)
	}
	branch, err := s.Branch(ctx, "main")
	if err != nil {
		t.Fatal(err)
	}
	if len(branch.State.Entities["item"]) != 0 {
		t.Fatal("state committed despite invalid declared output")
	}
}
