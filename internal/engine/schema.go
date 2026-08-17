package engine

import (
	"encoding/json"
	"fmt"
	"math"
)

func validateInput(schema map[string]any, input map[string]any) error {
	if schema == nil {
		return nil
	}
	if value, ok := schema["type"].(string); ok && value != "object" {
		return fmt.Errorf("root input schema type must be object")
	}
	required, _ := schema["required"].([]any)
	for _, raw := range required {
		name, ok := raw.(string)
		if !ok {
			continue
		}
		if _, exists := input[name]; !exists {
			return fmt.Errorf("missing required property %q", name)
		}
	}
	properties, _ := schema["properties"].(map[string]any)
	allowAdditional := true
	if value, ok := schema["additionalProperties"].(bool); ok {
		allowAdditional = value
	}
	for name, value := range input {
		rawProperty, exists := properties[name]
		if !exists {
			if !allowAdditional {
				return fmt.Errorf("unknown property %q", name)
			}
			continue
		}
		property, _ := rawProperty.(map[string]any)
		if err := validateType(name, property, value); err != nil {
			return err
		}
	}
	return nil
}

func validateType(name string, schema map[string]any, value any) error {
	want, _ := schema["type"].(string)
	valid := true
	switch want {
	case "", "null":
		valid = want == "" || value == nil
	case "string":
		_, valid = value.(string)
	case "boolean":
		_, valid = value.(bool)
	case "object":
		_, valid = value.(map[string]any)
	case "array":
		_, valid = value.([]any)
	case "number":
		switch value.(type) {
		case float64, float32, int, int64, int32, json.Number:
			valid = true
		default:
			valid = false
		}
	case "integer":
		switch number := value.(type) {
		case int, int64, int32:
			valid = true
		case float64:
			valid = !math.IsNaN(number) && !math.IsInf(number, 0) && math.Trunc(number) == number
		case json.Number:
			_, err := number.Int64()
			valid = err == nil
		default:
			valid = false
		}
	}
	if !valid {
		return fmt.Errorf("property %q must be %s", name, want)
	}
	if enum, ok := schema["enum"].([]any); ok {
		matched := false
		for _, candidate := range enum {
			if fmt.Sprint(candidate) == fmt.Sprint(value) {
				matched = true
				break
			}
		}
		if !matched {
			return fmt.Errorf("property %q is not in the declared enum", name)
		}
	}
	return nil
}
