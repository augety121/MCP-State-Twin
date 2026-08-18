package scenario

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/augety121/mcp-state-twin/internal/canonical"
	"github.com/augety121/mcp-state-twin/internal/engine"
	"github.com/augety121/mcp-state-twin/internal/spec"
	"github.com/augety121/mcp-state-twin/internal/store"
	"github.com/augety121/mcp-state-twin/internal/strictyaml"
	"github.com/augety121/mcp-state-twin/internal/world"
)

const (
	APIVersion       = "statetwin.dev/v1alpha1"
	Kind             = "Scenario"
	ReportFormat     = "statetwin.dev/scenario-report/v1alpha1"
	MaxScenarioBytes = 256 << 10
	MaxSteps         = 256
	MaxAssertions    = 1024
	MaxPointerDepth  = 64
	MaxValueDepth    = 64
	MaxTraceBytes    = 16 << 20
)

var identifierPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$`)

type Scenario struct {
	APIVersion      string      `json:"apiVersion" yaml:"apiVersion"`
	Kind            string      `json:"kind" yaml:"kind"`
	Metadata        Metadata    `json:"metadata" yaml:"metadata"`
	Steps           []Step      `json:"steps" yaml:"steps"`
	FinalAssertions []Assertion `json:"finalAssertions,omitempty" yaml:"finalAssertions,omitempty"`
}

type Metadata struct {
	Name string `json:"name" yaml:"name"`
}

type Step struct {
	ID     string         `json:"id" yaml:"id"`
	Tool   string         `json:"tool" yaml:"tool"`
	Input  map[string]any `json:"input,omitempty" yaml:"input,omitempty"`
	Expect Expectation    `json:"expect" yaml:"expect"`
}

type Expectation struct {
	ErrorClass string      `json:"errorClass" yaml:"errorClass"`
	Assertions []Assertion `json:"assertions,omitempty" yaml:"assertions,omitempty"`
}

type Assertion struct {
	Path     string `json:"path" yaml:"path"`
	Operator string `json:"operator" yaml:"operator"`
	Value    any    `json:"value,omitempty" yaml:"value,omitempty"`
}

type EnvironmentIdentity struct {
	Format                 string `json:"format"`
	RuntimeSemanticVersion string `json:"runtimeSemanticVersion"`
	SpecDigest             string `json:"specDigest"`
	SurfaceDigest          string `json:"surfaceDigest"`
	SnapshotDigest         string `json:"snapshotDigest"`
	ScenarioDigest         string `json:"scenarioDigest"`
	Seed                   int64  `json:"seed"`
	ClockInitial           string `json:"clockInitial"`
	SchedulerPolicy        string `json:"schedulerPolicy"`
	FaultProfile           string `json:"faultProfile"`
}

type TraceEntry struct {
	StepID       string `json:"stepId"`
	Tool         string `json:"tool"`
	Input        any    `json:"input"`
	Result       any    `json:"result"`
	ErrorClass   string `json:"errorClass,omitempty"`
	BeforeDigest string `json:"beforeDigest"`
	AfterDigest  string `json:"afterDigest"`
	CallIndex    int64  `json:"callIndex"`
}

type AssertionResult struct {
	StepID   string `json:"stepId,omitempty"`
	Path     string `json:"path"`
	Operator string `json:"operator"`
	Expected any    `json:"expected"`
	Actual   any    `json:"actual"`
	Passed   bool   `json:"passed"`
	Message  string `json:"message,omitempty"`
}

type Report struct {
	Format               string              `json:"format"`
	ScenarioName         string              `json:"scenarioName"`
	AgentIdentity        string              `json:"agentIdentity"`
	Passed               bool                `json:"passed"`
	Environment          EnvironmentIdentity `json:"environment"`
	EnvironmentDigest    string              `json:"environmentDigest"`
	InitialStateDigest   string              `json:"initialStateDigest"`
	TerminalStateDigest  string              `json:"terminalStateDigest"`
	OrderedToolTrace     []TraceEntry        `json:"orderedToolTrace"`
	StateDiff            []store.Change      `json:"stateDiff"`
	Assertions           []AssertionResult   `json:"assertions"`
	DeclaredInvariantIDs []string            `json:"declaredInvariantIds"`
	ConfiguredFaults     []string            `json:"configuredFaults"`
	FiredFaults          []string            `json:"firedFaults"`
	UnsupportedBehaviors int                 `json:"unsupportedBehaviors"`
	Failures             []string            `json:"failures"`
}

func Load(path string) (*Scenario, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read Scenario: %w", err)
	}
	return Decode(data)
}

func Decode(data []byte) (*Scenario, error) {
	var result Scenario
	if err := strictyaml.DecodeOne(data, MaxScenarioBytes, "Scenario", &result); err != nil {
		return nil, err
	}
	if err := result.Validate(); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *Scenario) Digest() (string, error) {
	return canonical.Digest(s)
}

func (s *Scenario) Validate() error {
	var problems []string
	if s.APIVersion != APIVersion {
		problems = append(problems, fmt.Sprintf("apiVersion must be %q", APIVersion))
	}
	if s.Kind != Kind {
		problems = append(problems, fmt.Sprintf("kind must be %q", Kind))
	}
	if !identifierPattern.MatchString(s.Metadata.Name) {
		problems = append(problems, "metadata.name must use lowercase letters, digits, '-' or '_'")
	}
	if len(s.Steps) == 0 {
		problems = append(problems, "steps must not be empty")
	}
	if len(s.Steps) > MaxSteps {
		problems = append(problems, fmt.Sprintf("steps exceeds limit %d", MaxSteps))
	}

	ids := make(map[string]struct{}, len(s.Steps))
	assertionCount := len(s.FinalAssertions)
	for i, step := range s.Steps {
		prefix := fmt.Sprintf("steps[%d]", i)
		if !identifierPattern.MatchString(step.ID) {
			problems = append(problems, prefix+".id is invalid")
		} else if _, exists := ids[step.ID]; exists {
			problems = append(problems, prefix+".id must be unique")
		}
		ids[step.ID] = struct{}{}
		if strings.TrimSpace(step.Tool) == "" {
			problems = append(problems, prefix+".tool is required")
		}
		if step.Expect.ErrorClass != "" && !spec.IsCanonicalErrorClass(step.Expect.ErrorClass) {
			problems = append(problems, prefix+".expect.errorClass is not a canonical error class")
		}
		if step.Input == nil {
			step.Input = make(map[string]any)
			s.Steps[i] = step
		}
		if err := validateJSONValue(step.Input, 0); err != nil {
			problems = append(problems, prefix+".input "+err.Error())
		}
		assertionCount += len(step.Expect.Assertions)
		for j, assertion := range step.Expect.Assertions {
			if err := validateAssertion(assertion); err != nil {
				problems = append(problems, fmt.Sprintf("%s.expect.assertions[%d]: %v", prefix, j, err))
			}
		}
	}
	for i, assertion := range s.FinalAssertions {
		if err := validateAssertion(assertion); err != nil {
			problems = append(problems, fmt.Sprintf("finalAssertions[%d]: %v", i, err))
		}
	}
	if assertionCount > MaxAssertions {
		problems = append(problems, fmt.Sprintf("assertions exceeds limit %d", MaxAssertions))
	}
	if len(problems) > 0 {
		return errors.New("invalid Scenario: " + strings.Join(problems, "; "))
	}
	return nil
}

func validateAssertion(assertion Assertion) error {
	if assertion.Path == "" || !strings.HasPrefix(assertion.Path, "/") {
		return errors.New("path must be a non-empty JSON Pointer")
	}
	if len(strings.Split(strings.TrimPrefix(assertion.Path, "/"), "/")) > MaxPointerDepth {
		return fmt.Errorf("path exceeds depth limit %d", MaxPointerDepth)
	}
	if _, err := pointerSegments(assertion.Path); err != nil {
		return err
	}
	switch assertion.Operator {
	case "equals":
		if err := validateJSONValue(assertion.Value, 0); err != nil {
			return errors.New("value " + err.Error())
		}
	case "exists", "absent":
		if assertion.Value != nil {
			return errors.New("exists/absent assertions must not declare value")
		}
	default:
		return errors.New("operator must be equals, exists, or absent")
	}
	return nil
}

func validateJSONValue(value any, depth int) error {
	if depth > MaxValueDepth {
		return fmt.Errorf("exceeds JSON value depth limit %d", MaxValueDepth)
	}
	if value == nil {
		return nil
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Bool, reflect.String,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return nil
	case reflect.Float32, reflect.Float64:
		if math.IsNaN(reflected.Float()) || math.IsInf(reflected.Float(), 0) {
			return errors.New("contains a non-finite number")
		}
		return nil
	case reflect.Map:
		if reflected.Type().Key().Kind() != reflect.String {
			return errors.New("contains a map with non-string keys")
		}
		iterator := reflected.MapRange()
		for iterator.Next() {
			if err := validateJSONValue(iterator.Value().Interface(), depth+1); err != nil {
				return err
			}
		}
		return nil
	case reflect.Slice, reflect.Array:
		for i := 0; i < reflected.Len(); i++ {
			if err := validateJSONValue(reflected.Index(i).Interface(), depth+1); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("contains unsupported %s outside the JSON value domain", reflected.Kind())
	}
}

func Run(ctx context.Context, twin *spec.TwinSpec, initial *world.State, scenario *Scenario, runtimeVersion string) (*Report, error) {
	if twin == nil {
		return nil, errors.New("TwinSpec is required")
	}
	if scenario == nil {
		return nil, errors.New("Scenario is required")
	}
	if err := scenario.Validate(); err != nil {
		return nil, err
	}
	stateStore, err := store.Open(":memory:")
	if err != nil {
		return nil, err
	}
	defer stateStore.Close()
	runtime, err := engine.New(twin, stateStore)
	if err != nil {
		return nil, err
	}
	if initial == nil {
		initial = world.New()
	} else {
		initial, err = initial.Clone()
		if err != nil {
			return nil, err
		}
	}
	const branchID = "scenario"
	if err := runtime.Initialize(ctx, branchID, initial); err != nil {
		return nil, err
	}
	base, err := stateStore.CreateSnapshot(ctx, "scenario-base", branchID)
	if err != nil {
		return nil, err
	}
	if err := stateStore.Fork(ctx, "scenario-base", "scenario-baseline"); err != nil {
		return nil, err
	}
	scenarioDigest, err := scenario.Digest()
	if err != nil {
		return nil, err
	}
	environment := EnvironmentIdentity{
		Format:                 "statetwin.dev/environment/v1alpha1",
		RuntimeSemanticVersion: runtimeVersion,
		SpecDigest:             runtime.Digest(),
		SurfaceDigest:          runtime.SurfaceDigest(),
		SnapshotDigest:         base.ID,
		ScenarioDigest:         scenarioDigest,
		Seed:                   0,
		ClockInitial:           twin.Clock.Initial,
		SchedulerPolicy:        "serial-v0.1",
		FaultProfile:           "none",
	}
	environmentDigest, err := canonical.Digest(environment)
	if err != nil {
		return nil, err
	}
	report := &Report{
		Format:               ReportFormat,
		ScenarioName:         scenario.Metadata.Name,
		AgentIdentity:        "scripted-scenario",
		Passed:               true,
		Environment:          environment,
		EnvironmentDigest:    environmentDigest,
		InitialStateDigest:   base.StateDigest,
		OrderedToolTrace:     make([]TraceEntry, 0, len(scenario.Steps)),
		Assertions:           make([]AssertionResult, 0),
		DeclaredInvariantIDs: make([]string, 0, len(twin.Invariants)),
		ConfiguredFaults:     []string{},
		FiredFaults:          []string{},
		Failures:             []string{},
	}
	for _, invariant := range twin.Invariants {
		report.DeclaredInvariantIDs = append(report.DeclaredInvariantIDs, invariant.ID)
	}

	traceBytes := 0
	for _, step := range scenario.Steps {
		result, callErr := runtime.Call(ctx, branchID, step.Tool, step.Input)
		if callErr != nil {
			return nil, fmt.Errorf("execute step %s: %w", step.ID, callErr)
		}
		entry := TraceEntry{
			StepID: step.ID, Tool: step.Tool, Input: step.Input, Result: result.Result,
			ErrorClass: result.ErrorClass, BeforeDigest: result.BeforeDigest,
			AfterDigest: result.AfterDigest, CallIndex: result.CallIndex,
		}
		entryBytes, err := canonical.JSON(entry)
		if err != nil {
			return nil, err
		}
		traceBytes += len(entryBytes)
		if traceBytes > MaxTraceBytes {
			return nil, fmt.Errorf("scenario trace exceeds %d bytes", MaxTraceBytes)
		}
		report.OrderedToolTrace = append(report.OrderedToolTrace, entry)
		if result.ErrorClass == "UNMODELED_BEHAVIOR" {
			report.UnsupportedBehaviors++
		}
		if result.ErrorClass != step.Expect.ErrorClass {
			report.Passed = false
			report.Failures = append(report.Failures, fmt.Sprintf("step %s: errorClass %q, expected %q", step.ID, result.ErrorClass, step.Expect.ErrorClass))
			break
		}
		branch, err := stateStore.Branch(ctx, branchID)
		if err != nil {
			return nil, err
		}
		if !appendAssertions(report, step.ID, branch.State.CELValue(), step.Expect.Assertions) {
			report.Passed = false
			break
		}
	}
	terminal, err := stateStore.Branch(ctx, branchID)
	if err != nil {
		return nil, err
	}
	if report.Passed && !appendAssertions(report, "", terminal.State.CELValue(), scenario.FinalAssertions) {
		report.Passed = false
	}
	report.TerminalStateDigest = terminal.StateDigest
	report.StateDiff, err = stateStore.DiffBranches(ctx, "scenario-baseline", branchID)
	if err != nil {
		return nil, err
	}
	if report.StateDiff == nil {
		report.StateDiff = []store.Change{}
	}
	return report, nil
}

func appendAssertions(report *Report, stepID string, state map[string]any, assertions []Assertion) bool {
	allPassed := true
	for _, assertion := range assertions {
		actual, found, err := resolvePointer(state, assertion.Path)
		result := AssertionResult{StepID: stepID, Path: assertion.Path, Operator: assertion.Operator, Expected: assertion.Value, Actual: actual}
		switch {
		case err != nil:
			result.Message = err.Error()
		case assertion.Operator == "exists":
			result.Passed = found
		case assertion.Operator == "absent":
			result.Passed = !found
		case assertion.Operator == "equals" && found:
			left, leftErr := canonical.JSON(actual)
			right, rightErr := canonical.JSON(assertion.Value)
			if leftErr != nil || rightErr != nil {
				result.Message = "assertion value is outside canonical JSON domain"
			} else {
				result.Passed = string(left) == string(right)
			}
		default:
			result.Message = "path does not exist"
		}
		if !result.Passed {
			allPassed = false
			if result.Message == "" {
				result.Message = "assertion did not match"
			}
			report.Failures = append(report.Failures, fmt.Sprintf("assertion %s %s failed", assertion.Path, assertion.Operator))
		}
		report.Assertions = append(report.Assertions, result)
	}
	return allPassed
}

func resolvePointer(root any, pointer string) (any, bool, error) {
	segments, err := pointerSegments(pointer)
	if err != nil {
		return nil, false, err
	}
	current := root
	for _, segment := range segments {
		switch typed := current.(type) {
		case map[string]any:
			var found bool
			current, found = typed[segment]
			if !found {
				return nil, false, nil
			}
		case []any:
			index, err := strconv.Atoi(segment)
			if err != nil || index < 0 || index >= len(typed) {
				return nil, false, nil
			}
			current = typed[index]
		default:
			return nil, false, nil
		}
	}
	return current, true, nil
}

func pointerSegments(pointer string) ([]string, error) {
	if !strings.HasPrefix(pointer, "/") {
		return nil, errors.New("path must start with '/'")
	}
	raw := strings.Split(strings.TrimPrefix(pointer, "/"), "/")
	segments := make([]string, len(raw))
	for i, segment := range raw {
		var builder strings.Builder
		for j := 0; j < len(segment); j++ {
			if segment[j] != '~' {
				builder.WriteByte(segment[j])
				continue
			}
			if j+1 >= len(segment) || (segment[j+1] != '0' && segment[j+1] != '1') {
				return nil, errors.New("path contains invalid JSON Pointer escape")
			}
			j++
			if segment[j] == '0' {
				builder.WriteByte('~')
			} else {
				builder.WriteByte('/')
			}
		}
		segments[i] = builder.String()
	}
	return segments, nil
}
