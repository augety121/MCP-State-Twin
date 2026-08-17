package engine

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/augety121/mcp-state-twin/internal/spec"
	"github.com/augety121/mcp-state-twin/internal/store"
	"github.com/augety121/mcp-state-twin/internal/world"
	"github.com/google/cel-go/cel"
)

type Runtime struct {
	spec     *spec.TwinSpec
	digest   string
	store    *store.Store
	env      *cel.Env
	programs map[string]cel.Program
	tools    map[string]spec.ToolSpec
	schemas  map[string]compiledSchemas
}

func New(twin *spec.TwinSpec, stateStore *store.Store) (*Runtime, error) {
	digest, err := twin.Digest()
	if err != nil {
		return nil, err
	}
	env, err := cel.NewEnv(
		cel.Variable("input", cel.DynType),
		cel.Variable("state", cel.DynType),
		cel.Variable("vars", cel.DynType),
		cel.Variable("item", cel.DynType),
		cel.Variable("clock", cel.StringType),
		cel.Variable("call_index", cel.IntType),
	)
	if err != nil {
		return nil, fmt.Errorf("create CEL environment: %w", err)
	}
	r := &Runtime{
		spec: twin, digest: digest, store: stateStore, env: env,
		programs: make(map[string]cel.Program),
		tools:    make(map[string]spec.ToolSpec, len(twin.Tools)),
		schemas:  make(map[string]compiledSchemas, len(twin.Tools)),
	}
	for _, tool := range twin.Tools {
		r.tools[tool.Name] = tool
		inputSchema, err := compileJSONSchema(tool.Name, "input", tool.InputSchema)
		if err != nil {
			return nil, fmt.Errorf("tool %s inputSchema: %w", tool.Name, err)
		}
		outputSchema, err := compileJSONSchema(tool.Name, "output", tool.OutputSchema)
		if err != nil {
			return nil, fmt.Errorf("tool %s outputSchema: %w", tool.Name, err)
		}
		r.schemas[tool.Name] = compiledSchemas{input: inputSchema, output: outputSchema}
		for _, condition := range tool.Preconditions {
			if err := r.compile(condition.Expr); err != nil {
				return nil, fmt.Errorf("tool %s precondition: %w", tool.Name, err)
			}
		}
		for _, effect := range tool.Effects {
			for _, expression := range []string{effect.Key, effect.Value} {
				if expression != "" {
					if err := r.compile(expression); err != nil {
						return nil, fmt.Errorf("tool %s effect %s: %w", tool.Name, effect.Op, err)
					}
				}
			}
		}
		if tool.Query != nil {
			for _, expression := range []string{tool.Query.Key, tool.Query.Where} {
				if expression != "" {
					if err := r.compile(expression); err != nil {
						return nil, fmt.Errorf("tool %s query: %w", tool.Name, err)
					}
				}
			}
		}
		for _, condition := range tool.Postconditions {
			if err := r.compile(condition.Expr); err != nil {
				return nil, fmt.Errorf("tool %s postcondition: %w", tool.Name, err)
			}
		}
		if tool.Result != "" {
			if err := r.compile(tool.Result); err != nil {
				return nil, fmt.Errorf("tool %s result: %w", tool.Name, err)
			}
		}
	}
	for _, invariant := range twin.Invariants {
		if err := r.compile(invariant.Assert); err != nil {
			return nil, fmt.Errorf("invariant %s: %w", invariant.ID, err)
		}
	}
	return r, nil
}

func (r *Runtime) Digest() string { return r.digest }

func (r *Runtime) Spec() *spec.TwinSpec { return r.spec }

func (r *Runtime) compile(expression string) error {
	if _, exists := r.programs[expression]; exists {
		return nil
	}
	ast, issues := r.env.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return issues.Err()
	}
	program, err := r.env.Program(ast, cel.CostLimit(10_000))
	if err != nil {
		return err
	}
	r.programs[expression] = program
	return nil
}

func (r *Runtime) Initialize(ctx context.Context, branchID string, initial *world.State) error {
	clock, err := time.Parse(time.RFC3339Nano, r.spec.Clock.Initial)
	if err != nil {
		return fmt.Errorf("parse virtual clock: %w", err)
	}
	for entity := range r.spec.State.Entities {
		if initial.Entities[entity] == nil {
			initial.Entities[entity] = make(map[string]map[string]any)
		}
	}
	return r.store.InitializeBranch(ctx, branchID, r.digest, initial, clock)
}

func (r *Runtime) Call(ctx context.Context, branchID, toolName string, input map[string]any) (*store.ApplyResult, error) {
	tool, exists := r.tools[toolName]
	if !exists {
		return nil, fmt.Errorf("unknown tool %q", toolName)
	}
	return r.store.ApplyCall(ctx, branchID, r.digest, toolName, input, func(state *world.State, clock time.Time, callIndex int64) (store.CallOutcome, error) {
		return r.apply(tool, state, clock, callIndex, input), nil
	})
}

func (r *Runtime) apply(tool spec.ToolSpec, state *world.State, clock time.Time, callIndex int64, input map[string]any) store.CallOutcome {
	if tool.Modeled != nil && !*tool.Modeled {
		return failure("UNMODELED_BEHAVIOR", "tool behavior is not modeled")
	}
	toolSchemas := r.schemas[tool.Name]
	if err := validateJSON(input, toolSchemas.input); err != nil {
		return failure("INVALID_INPUT", err.Error())
	}
	vars := make(map[string]any)
	activation := func(item any) map[string]any {
		return map[string]any{
			"input": input, "state": state.CELValue(), "vars": vars,
			"item": item, "clock": clock.UTC().Format(time.RFC3339Nano), "call_index": callIndex,
		}
	}

	for _, condition := range tool.Preconditions {
		ok, err := r.evalBool(condition.Expr, activation(nil))
		if err != nil {
			return failure("INTERNAL_TWIN_ERROR", "precondition evaluation failed: "+err.Error())
		}
		if !ok {
			code := condition.Code
			if code == "" {
				code = "PRECONDITION_FAILED"
			}
			message := condition.Message
			if message == "" {
				message = "declared precondition failed"
			}
			return failure(code, message)
		}
	}

	for _, effect := range tool.Effects {
		switch effect.Op {
		case "allocate":
			state.Sequences[effect.Sequence]++
			vars[effect.As] = state.Sequences[effect.Sequence]
		case "insert", "update", "delete":
			keyValue, err := r.eval(effect.Key, activation(nil))
			if err != nil {
				return failure("INTERNAL_TWIN_ERROR", "effect key evaluation failed: "+err.Error())
			}
			key, ok := keyValue.(string)
			if !ok || key == "" {
				return failure("INTERNAL_TWIN_ERROR", "effect key must evaluate to a non-empty string")
			}
			entities := state.Entities[effect.Entity]
			if entities == nil {
				entities = make(map[string]map[string]any)
				state.Entities[effect.Entity] = entities
			}
			existing, found := entities[key]
			switch effect.Op {
			case "insert":
				if found {
					return failure("CONFLICT", effect.Entity+" already exists")
				}
				value, err := r.evalMap(effect.Value, activation(nil))
				if err != nil {
					return failure("INTERNAL_TWIN_ERROR", "insert value evaluation failed: "+err.Error())
				}
				entities[key] = value
				vars[effect.Entity+"_key"] = key
			case "update":
				if !found {
					return failure("NOT_FOUND", effect.Entity+" does not exist")
				}
				value, err := r.evalMap(effect.Value, activation(existing))
				if err != nil {
					return failure("INTERNAL_TWIN_ERROR", "update value evaluation failed: "+err.Error())
				}
				if effect.Merge {
					for field, fieldValue := range value {
						existing[field] = fieldValue
					}
				} else {
					entities[key] = value
				}
				vars[effect.Entity+"_key"] = key
			case "delete":
				if !found {
					return failure("NOT_FOUND", effect.Entity+" does not exist")
				}
				delete(entities, key)
			}
		}
	}

	if tool.Query != nil {
		query := tool.Query
		entities := state.Entities[query.Entity]
		if query.Many {
			keys := make([]string, 0, len(entities))
			for key := range entities {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			items := make([]any, 0, len(keys))
			for _, key := range keys {
				item := entities[key]
				if query.Where != "" {
					ok, err := r.evalBool(query.Where, activation(item))
					if err != nil {
						return failure("INTERNAL_TWIN_ERROR", "query predicate failed: "+err.Error())
					}
					if !ok {
						continue
					}
				}
				items = append(items, item)
			}
			vars[query.As] = items
		} else {
			keyValue, err := r.eval(query.Key, activation(nil))
			if err != nil {
				return failure("INTERNAL_TWIN_ERROR", "query key failed: "+err.Error())
			}
			key, ok := keyValue.(string)
			if !ok {
				return failure("INTERNAL_TWIN_ERROR", "query key must evaluate to string")
			}
			item, found := entities[key]
			if !found {
				return failure("NOT_FOUND", query.Entity+" does not exist")
			}
			vars[query.As] = item
		}
	}

	for _, condition := range tool.Postconditions {
		ok, err := r.evalBool(condition.Expr, activation(nil))
		if err != nil || !ok {
			message := condition.Message
			if message == "" {
				message = "declared postcondition failed"
			}
			if err != nil {
				message += ": " + err.Error()
			}
			return failure("INVARIANT_VIOLATION", message)
		}
	}
	for _, invariant := range r.spec.Invariants {
		ok, err := r.evalBool(invariant.Assert, activation(nil))
		if err != nil || !ok {
			message := "global invariant failed: " + invariant.ID
			if err != nil {
				message += ": " + err.Error()
			}
			return failure("INVARIANT_VIOLATION", message)
		}
	}

	result := any(map[string]any{"ok": true})
	if tool.Result != "" {
		value, err := r.eval(tool.Result, activation(nil))
		if err != nil {
			return failure("INTERNAL_TWIN_ERROR", "result evaluation failed: "+err.Error())
		}
		result = value
	}
	if err := validateJSON(result, toolSchemas.output); err != nil {
		return failure("INTERNAL_TWIN_ERROR", "declared outputSchema rejected tool result: "+err.Error())
	}
	return store.CallOutcome{Result: result, CommitState: true}
}

func failure(code, message string) store.CallOutcome {
	return store.CallOutcome{
		Result:      map[string]any{"error": map[string]any{"code": code, "message": message}},
		ErrorClass:  code,
		CommitState: false,
	}
}

func (r *Runtime) eval(expression string, activation map[string]any) (any, error) {
	program, ok := r.programs[expression]
	if !ok {
		return nil, fmt.Errorf("expression was not compiled")
	}
	value, _, err := program.Eval(activation)
	if err != nil {
		return nil, err
	}
	native, err := value.ConvertToNative(reflect.TypeOf((*any)(nil)).Elem())
	if err != nil {
		return nil, err
	}
	return normalizeNative(native)
}

func normalizeNative(value any) (any, error) {
	switch typed := value.(type) {
	case map[any]any:
		result := make(map[string]any, len(typed))
		for rawKey, rawValue := range typed {
			key, ok := rawKey.(string)
			if !ok {
				return nil, fmt.Errorf("object key must be string, got %T", rawKey)
			}
			converted, err := normalizeNative(rawValue)
			if err != nil {
				return nil, err
			}
			result[key] = converted
		}
		return result, nil
	case map[string]any:
		result := make(map[string]any, len(typed))
		for key, rawValue := range typed {
			converted, err := normalizeNative(rawValue)
			if err != nil {
				return nil, err
			}
			result[key] = converted
		}
		return result, nil
	case []any:
		result := make([]any, len(typed))
		for i, rawValue := range typed {
			converted, err := normalizeNative(rawValue)
			if err != nil {
				return nil, err
			}
			result[i] = converted
		}
		return result, nil
	default:
		return value, nil
	}
}

func (r *Runtime) evalBool(expression string, activation map[string]any) (bool, error) {
	value, err := r.eval(expression, activation)
	if err != nil {
		return false, err
	}
	result, ok := value.(bool)
	if !ok {
		return false, fmt.Errorf("expression must return boolean, got %T", value)
	}
	return result, nil
}

func (r *Runtime) evalMap(expression string, activation map[string]any) (map[string]any, error) {
	value, err := r.eval(expression, activation)
	if err != nil {
		return nil, err
	}
	result, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expression must return object, got %T", value)
	}
	return result, nil
}

func (r *Runtime) ToolNames() []string {
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func IsDomainError(result *store.ApplyResult) bool {
	return result != nil && strings.TrimSpace(result.ErrorClass) != ""
}
