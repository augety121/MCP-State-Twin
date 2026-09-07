// Package task defines experimental goal-based tasks independently of Scenario.
package task

import (
	"errors"
	"fmt"
	"path"
	"reflect"
	"regexp"
	"strings"

	"github.com/augety121/mcp-state-twin/internal/limits"
	"github.com/augety121/mcp-state-twin/internal/spec"
	"github.com/augety121/mcp-state-twin/internal/strictyaml"
)

const (
	APIVersion    = "statetwin.dev/v1alpha1"
	Kind          = "AgentTask"
	Format        = "statetwin.dev/agent-task/v1alpha1"
	MaxBytes      = 256 << 10
	MaxAssertions = 64
	MaxEvents     = 512
)

var idPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$`)

type Task struct {
	APIVersion      string      `json:"apiVersion" yaml:"apiVersion"`
	Kind            string      `json:"kind" yaml:"kind"`
	Format          string      `json:"format" yaml:"format"`
	ID              string      `json:"id" yaml:"id"`
	Revision        string      `json:"revision" yaml:"revision"`
	Domain          string      `json:"domain" yaml:"domain"`
	Bundle          string      `json:"bundle" yaml:"bundle"`
	Objective       string      `json:"objective" yaml:"objective"`
	Context         string      `json:"context,omitempty" yaml:"context,omitempty"`
	Mode            string      `json:"mode" yaml:"mode"`
	Tools           []string    `json:"tools" yaml:"tools"`
	Authority       []Rule      `json:"authority" yaml:"authority"`
	Budgets         Budgets     `json:"budgets" yaml:"budgets"`
	Oracle          []Assertion `json:"oracle" yaml:"oracle"`
	ExpectedOutcome string      `json:"expectedOutcome" yaml:"expectedOutcome"`
	FaultTool       string      `json:"faultTool,omitempty" yaml:"faultTool,omitempty"`
}

type Rule struct {
	Tool   string         `json:"tool" yaml:"tool"`
	Equals map[string]any `json:"equals" yaml:"equals"`
}

type Assertion struct {
	ID       string `json:"id" yaml:"id"`
	Category string `json:"category" yaml:"category"`
	Expr     string `json:"expr" yaml:"expr"`
}

type Budgets struct {
	ModelRequests  int `json:"modelRequests" yaml:"modelRequests"`
	ToolAttempts   int `json:"toolAttempts" yaml:"toolAttempts"`
	RequestSeconds int `json:"requestSeconds" yaml:"requestSeconds"`
	EpisodeSeconds int `json:"episodeSeconds" yaml:"episodeSeconds"`
	CleanupSeconds int `json:"cleanupSeconds" yaml:"cleanupSeconds"`
	ResponseBytes  int `json:"responseBytes" yaml:"responseBytes"`
	TraceBytes     int `json:"traceBytes" yaml:"traceBytes"`
}

func (b Budgets) Validate() error {
	values := []struct{ value, max int }{
		{b.ModelRequests, 16}, {b.ToolAttempts, 32}, {b.RequestSeconds, 30},
		{b.EpisodeSeconds, 180}, {b.CleanupSeconds, 10},
		{b.ResponseBytes, 1 << 20}, {b.TraceBytes, 8 << 20},
	}
	for _, v := range values {
		if v.value < 1 || v.value > v.max {
			return errors.New("TASK_INVALID: budget outside supported bounds")
		}
	}
	if b.TraceBytes < 64<<10 || b.RequestSeconds > b.EpisodeSeconds {
		return errors.New("TASK_INVALID: inconsistent budget")
	}
	return nil
}

func Decode(data []byte) (*Task, error) {
	var t Task
	if err := strictyaml.DecodeOneWithDepth(data, MaxBytes, 32, "AgentTask", &t); err != nil {
		return nil, errors.New("TASK_INVALID: strict document admission failed")
	}
	if err := t.Validate(); err != nil {
		return nil, err
	}
	return &t, nil
}

func (t *Task) Validate() error {
	if t == nil {
		return errors.New("TASK_INVALID: task is required")
	}
	if t.APIVersion != APIVersion || t.Kind != Kind || t.Format != Format {
		return errors.New("TASK_INVALID: unsupported task format")
	}
	if !idPattern.MatchString(t.ID) || len(t.ID) > 128 || !idPattern.MatchString(t.Domain) || len(t.Domain) > 128 || !idPattern.MatchString(t.Revision) || len(t.Revision) > 128 {
		return errors.New("TASK_INVALID: invalid identity")
	}
	if err := PortablePath(t.Bundle); err != nil {
		return err
	}
	if strings.TrimSpace(t.Objective) == "" || len(t.Objective) > 8<<10 || len(t.Context) > 32<<10 || t.Mode != "blind" {
		return errors.New("TASK_INVALID: unsupported objective or mode")
	}
	if t.ExpectedOutcome != "success" && t.ExpectedOutcome != "expected_abstention" {
		return errors.New("TASK_INVALID: unsupported expected outcome")
	}
	if err := t.Budgets.Validate(); err != nil {
		return err
	}
	if len(t.Tools) < 1 || len(t.Tools) > 32 || len(t.Authority) < 1 || len(t.Authority) > 64 {
		return errors.New("TASK_INVALID: tool or authority count")
	}
	seen := map[string]bool{}
	for _, tool := range t.Tools {
		if !idPattern.MatchString(tool) || len(tool) > 128 || seen[tool] {
			return errors.New("TASK_INVALID: invalid or duplicate tool")
		}
		seen[tool] = true
	}
	if t.FaultTool != "" && !seen[t.FaultTool] {
		return errors.New("TASK_INVALID: fault requires a visible tool")
	}
	for _, rule := range t.Authority {
		if !seen[rule.Tool] || len(rule.Equals) == 0 || len(rule.Equals) > 32 {
			return errors.New("TASK_INVALID: authority must constrain a visible tool")
		}
		for key, value := range rule.Equals {
			if !idPattern.MatchString(key) || len(key) > 128 || !scalar(value) {
				return errors.New("TASK_INVALID: authority requires scalar equalities")
			}
		}
	}
	if len(t.Oracle) < 2 || len(t.Oracle) > MaxAssertions {
		return errors.New("TASK_INVALID: oracle count")
	}
	ids := map[string]bool{}
	goal, policy := false, false
	for _, assertion := range t.Oracle {
		if !idPattern.MatchString(assertion.ID) || len(assertion.ID) > 128 || ids[assertion.ID] || strings.TrimSpace(assertion.Expr) == "" || len(assertion.Expr) > 4096 {
			return errors.New("TASK_INVALID: assertion identity or size")
		}
		ids[assertion.ID] = true
		switch assertion.Category {
		case "goal":
			goal = true
		case "policy":
			policy = true
		default:
			return errors.New("TASK_INVALID: assertion category")
		}
	}
	if !goal || !policy {
		return errors.New("TASK_INVALID: goal and policy assertions required")
	}
	if err := limits.ValidateJSON(t, MaxBytes); err != nil {
		return errors.New("TASK_INVALID: non-JSON or oversized task")
	}
	return nil
}

// AdmitSurface rejects missing or unmodeled tools without changing the TwinSpec.
// This is structural admission, not proof of task solvability or model ability.
func (t *Task) AdmitSurface(twin *spec.TwinSpec) error {
	if err := t.Validate(); err != nil {
		return err
	}
	if twin == nil || twin.Metadata.Name != t.Domain {
		return errors.New("TASK_INVALID: domain mismatch")
	}
	tools := map[string]spec.ToolSpec{}
	for _, tool := range twin.Tools {
		tools[tool.Name] = tool
	}
	for _, name := range t.Tools {
		tool, ok := tools[name]
		if !ok || (tool.Modeled != nil && !*tool.Modeled) {
			return fmt.Errorf("TASK_INVALID: unavailable declared tool %s", name)
		}
	}
	for _, rule := range t.Authority {
		properties, ok := tools[rule.Tool].InputSchema["properties"].(map[string]any)
		if !ok {
			return errors.New("TASK_INVALID: authority requires object properties")
		}
		for key := range rule.Equals {
			if _, ok := properties[key]; !ok {
				return errors.New("TASK_INVALID: authority names nonexistent input property")
			}
		}
	}
	return nil
}

// Authorized performs exact top-level scalar matching. It is not JSON Schema
// validation, business idempotency, or a replacement for server-side isolation.
func (t *Task) Authorized(tool string, input map[string]any) bool {
	for _, rule := range t.Authority {
		if rule.Tool != tool {
			continue
		}
		matches := true
		for key, want := range rule.Equals {
			got, ok := input[key]
			if !ok || !ScalarEqual(got, want) {
				matches = false
				break
			}
		}
		if matches {
			return true
		}
	}
	return false
}

func scalar(v any) bool {
	if v == nil {
		return false
	}
	switch reflect.TypeOf(v).Kind() {
	case reflect.String, reflect.Bool, reflect.Int, reflect.Int64, reflect.Float64:
		return true
	}
	return false
}

// ScalarEqual preserves numeric identity across YAML integers and decoded JSON.
func ScalarEqual(a, b any) bool {
	if reflect.DeepEqual(a, b) {
		return true
	}
	// CEL/JSON numeric equivalence is handled by the bounded oracle; permission
	// checks deliberately only accept exactly representable integral values.
	toInt := func(v any) (int64, bool) {
		switch n := v.(type) {
		case int:
			return int64(n), true
		case int64:
			return n, true
		case float64:
			if n >= -9007199254740991 && n <= 9007199254740991 && n == float64(int64(n)) {
				return int64(n), true
			}
		}
		return 0, false
	}
	x, xok := toInt(a)
	y, yok := toInt(b)
	return xok && yok && x == y
}

func PortablePath(name string) error {
	if name == "" || len(name) > 240 || path.IsAbs(name) || path.Clean(name) != name || strings.ContainsAny(name, "\\:\x00") {
		return errors.New("TASK_INVALID: unsafe bundle path")
	}
	for _, s := range strings.Split(name, "/") {
		if s == "." || s == ".." || strings.TrimRight(s, " .") != s {
			return errors.New("TASK_INVALID: unsafe bundle path")
		}
	}
	return nil
}

// Visible returns only explicitly agent-facing task data, never the oracle or
// artifact locations. A new map prevents a caller mutating private task fields.
func (t *Task) Visible() map[string]string {
	return map[string]string{"objective": t.Objective, "context": t.Context}
}
