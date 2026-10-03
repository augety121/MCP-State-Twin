package baselinepack

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
)

func requiredFields(raw []byte, dst any) error {
	var v any
	if json.Unmarshal(raw, &v) != nil {
		return errors.New("PLUGIN_PLAN_INVALID")
	}
	var visit func(reflect.Type, any) bool
	visit = func(t reflect.Type, v any) bool {
		for t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
		switch t.Kind() {
		case reflect.Struct:
			object, ok := v.(map[string]any)
			if !ok {
				return false
			}
			for i := 0; i < t.NumField(); i++ {
				f := t.Field(i)
				if !f.IsExported() {
					continue
				}
				tag := strings.Split(f.Tag.Get("json"), ",")
				if tag[0] == "-" {
					continue
				}
				name := tag[0]
				if name == "" {
					name = f.Name
				}
				value, exists := object[name]
				optional := len(tag) > 1 && tag[1] == "omitempty"
				if !exists {
					if !optional {
						return false
					}
					continue
				}
				if !visit(f.Type, value) {
					return false
				}
			}
		case reflect.Slice, reflect.Array:
			if t.Elem().Kind() == reflect.Uint8 {
				return true
			}
			items, ok := v.([]any)
			if !ok {
				return false
			}
			for _, item := range items {
				if !visit(t.Elem(), item) {
					return false
				}
			}
		case reflect.Map:
			object, ok := v.(map[string]any)
			if !ok {
				return false
			}
			for _, item := range object {
				if !visit(t.Elem(), item) {
					return false
				}
			}
		}
		return true
	}
	if !visit(reflect.TypeOf(dst), v) {
		return errors.New("PLUGIN_PLAN_INVALID")
	}
	return nil
}

// Schema exposes the exact closed field/type/required structure used by the
// decoder. Cross-file identity, resource limits and semantic enums are still
// enforced by admission; a schema-only pass never authorizes execution.
func Schema(value any) map[string]any {
	var build func(reflect.Type) map[string]any
	build = func(t reflect.Type) map[string]any {
		for t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
		s := map[string]any{}
		switch t.Kind() {
		case reflect.Struct:
			s["type"] = "object"
			s["additionalProperties"] = false
			properties := map[string]any{}
			required := []string{}
			for i := 0; i < t.NumField(); i++ {
				f := t.Field(i)
				if !f.IsExported() {
					continue
				}
				tag := strings.Split(f.Tag.Get("json"), ",")
				if tag[0] == "-" {
					continue
				}
				name := tag[0]
				if name == "" {
					name = f.Name
				}
				properties[name] = build(f.Type)
				if len(tag) == 1 || tag[1] != "omitempty" {
					required = append(required, name)
				}
			}
			s["properties"] = properties
			s["required"] = required
		case reflect.String:
			s["type"] = "string"
		case reflect.Bool:
			s["type"] = "boolean"
		case reflect.Int, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint64:
			s["type"] = "integer"
		case reflect.Float32, reflect.Float64:
			s["type"] = "number"
		case reflect.Slice, reflect.Array:
			if t.Elem().Kind() == reflect.Uint8 {
				s["type"] = "string"
				s["contentEncoding"] = "base64"
			} else {
				s["type"] = "array"
				s["items"] = build(t.Elem())
			}
		case reflect.Map:
			s["type"] = "object"
			s["additionalProperties"] = build(t.Elem())
		}
		return s
	}
	result := build(reflect.TypeOf(value))
	result["$schema"] = "https://json-schema.org/draft/2020-12/schema"
	return result
}
