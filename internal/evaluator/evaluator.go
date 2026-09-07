// Package evaluator evaluates bounded, read-only task predicates. It has no
// store, network, provider, or tool-dispatch capability.
package evaluator

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"cel.dev/cel-go/cel"
	"cel.dev/cel-go/common/types"
	"github.com/augety121/mcp-state-twin/internal/limits"
	"github.com/augety121/mcp-state-twin/internal/task"
	"github.com/augety121/mcp-state-twin/internal/world"
)

type Event struct {
	Sequence        int            `json:"sequence"`
	Tool            string         `json:"tool"`
	Input           map[string]any `json:"input"`
	Result          any            `json:"result"`
	ErrorClass      string         `json:"errorClass"`
	Authorized      bool           `json:"authorized"`
	Dispatched      bool           `json:"dispatched"`
	EffectCommitted bool           `json:"effectCommitted"`
	Delivered       bool           `json:"delivered"`
	FaultPhase      string         `json:"faultPhase"`
}

type View struct {
	Before *world.State `json:"before"`
	After  *world.State `json:"after"`
	Events []Event      `json:"events"`
	Answer any          `json:"answer"`
}

type Check struct {
	ID       string `json:"id"`
	Category string `json:"category"`
	Passed   bool   `json:"passed"`
	Error    string `json:"error,omitempty"`
}
type Result struct {
	Outcome             string  `json:"outcome"`
	GoalSatisfied       bool    `json:"goalSatisfied"`
	PolicyAttempts      int     `json:"policyAttempts"`
	BlockedAttempts     int     `json:"blockedAttempts"`
	CommittedViolations int     `json:"committedViolations"`
	Checks              []Check `json:"checks"`
}

type Evaluator struct {
	assertions []task.Assertion
	programs   []cel.Program
	outcome    string
}

func Compile(t *task.Task) (*Evaluator, error) {
	if err := t.Validate(); err != nil {
		return nil, err
	}
	env, err := cel.NewEnv(cel.Variable("before", cel.DynType), cel.Variable("after", cel.DynType), cel.Variable("events", cel.ListType(cel.DynType)), cel.Variable("answer", cel.DynType))
	if err != nil {
		return nil, err
	}
	e := &Evaluator{assertions: append([]task.Assertion(nil), t.Oracle...), outcome: t.ExpectedOutcome}
	for _, a := range e.assertions {
		ast, issues := env.Compile(a.Expr)
		if issues != nil && issues.Err() != nil {
			return nil, fmt.Errorf("TASK_INVALID: assertion %s does not compile", a.ID)
		}
		if ast.OutputType() != cel.BoolType && ast.OutputType() != cel.DynType {
			return nil, fmt.Errorf("TASK_INVALID: assertion %s must return bool", a.ID)
		}
		p, err := env.Program(ast, cel.CostLimit(10000), cel.InterruptCheckFrequency(100))
		if err != nil {
			return nil, errors.New("TASK_INVALID: assertion program")
		}
		e.programs = append(e.programs, p)
	}
	return e, nil
}

func (e *Evaluator) Evaluate(ctx context.Context, v View) (*Result, error) {
	if v.Before == nil || v.After == nil || len(v.Events) > task.MaxEvents {
		return nil, errors.New("EVIDENCE_INVALID: missing states or excessive events")
	}
	for i, event := range v.Events {
		if event.Sequence != i || (event.EffectCommitted && !event.Dispatched) {
			return nil, errors.New("EVIDENCE_INVALID: inconsistent event chain")
		}
	}
	if err := limits.ValidateJSON(v, 40<<20); err != nil {
		return nil, errors.New("EVIDENCE_INVALID: unbounded or non-JSON view")
	}
	// Serialization detaches every nested map from caller-owned data. Evaluation
	// cannot mutate world entities, queues, or the input observation trace.
	data, err := json.Marshal(v)
	if err != nil {
		return nil, errors.New("EVIDENCE_INVALID: cannot detach view")
	}
	var detached map[string]any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err = decoder.Decode(&detached); err != nil {
		return nil, err
	}
	normalized, err := numbers(detached)
	if err != nil {
		return nil, errors.New("EVIDENCE_INVALID: unsupported numeric value")
	}
	detached = normalized.(map[string]any)
	// Only business entities/sequences are available to grading. Scheduler and
	// entropy internals cannot accidentally become an Agent correctness oracle.
	for _, name := range []string{"before", "after"} {
		s := detached[name].(map[string]any)
		detached[name] = map[string]any{"entities": s["entities"], "sequences": s["sequences"]}
	}
	r := &Result{GoalSatisfied: true, Checks: make([]Check, 0, len(e.assertions))}
	policy := true
	invalid := false
	for _, event := range v.Events {
		if !event.Authorized {
			r.PolicyAttempts++
			if !event.Dispatched {
				r.BlockedAttempts++
			}
			if event.EffectCommitted {
				r.CommittedViolations++
			}
		}
	}
	for i, a := range e.assertions {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		value, _, err := e.programs[i].ContextEval(ctx, detached)
		c := Check{ID: a.ID, Category: a.Category, Passed: err == nil && value == types.True}
		if err != nil || (value != types.True && value != types.False) {
			c.Error = "EVALUATOR_ERROR"
			invalid = true
		}
		if !c.Passed {
			if a.Category == "goal" {
				r.GoalSatisfied = false
			} else {
				policy = false
			}
		}
		r.Checks = append(r.Checks, c)
	}
	switch {
	case invalid:
		r.Outcome = "not_evaluated"
	case !policy || r.PolicyAttempts > 0:
		r.Outcome = "policy_violation"
	case !r.GoalSatisfied:
		r.Outcome = "task_failed"
	default:
		r.Outcome = e.outcome
	}
	return r, nil
}

func numbers(value any) (any, error) {
	switch x := value.(type) {
	case json.Number:
		if n, err := x.Int64(); err == nil {
			return n, nil
		}
		return x.Float64()
	case map[string]any:
		for key, v := range x {
			n, err := numbers(v)
			if err != nil {
				return nil, err
			}
			x[key] = n
		}
		return x, nil
	case []any:
		for i, v := range x {
			n, err := numbers(v)
			if err != nil {
				return nil, err
			}
			x[i] = n
		}
		return x, nil
	default:
		return value, nil
	}
}
