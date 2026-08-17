package engine

import (
	"fmt"
	"strings"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

type compiledSchemas struct {
	input  *jsonschema.Schema
	output *jsonschema.Schema
}

// denyExternalSchemaLoader makes TwinSpec compilation hermetic. Local
// fragments and $defs remain supported because they resolve inside the
// in-memory resource; any reference that needs another resource fails closed.
type denyExternalSchemaLoader struct{}

func (denyExternalSchemaLoader) Load(url string) (any, error) {
	return nil, fmt.Errorf("external JSON Schema resource is not allowed: %s", url)
}

func compileJSONSchema(toolName, direction string, document map[string]any) (*jsonschema.Schema, error) {
	if document == nil {
		return nil, nil
	}
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	compiler.AssertFormat()
	compiler.UseLoader(denyExternalSchemaLoader{})
	resource := "urn:statetwin:tool:" + toolName + ":" + direction
	if err := compiler.AddResource(resource, document); err != nil {
		return nil, fmt.Errorf("add schema resource: %w", err)
	}
	compiled, err := compiler.Compile(resource)
	if err != nil {
		return nil, fmt.Errorf("compile schema: %w", err)
	}
	return compiled, nil
}

func validateJSON(instance any, schema *jsonschema.Schema) error {
	if schema == nil {
		return nil
	}
	if err := schema.Validate(instance); err != nil {
		return fmt.Errorf("JSON Schema validation failed: %s", strings.TrimSpace(err.Error()))
	}
	return nil
}
