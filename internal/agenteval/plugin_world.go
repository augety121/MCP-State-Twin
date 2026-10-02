package agenteval

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/augety121/mcp-state-twin/internal/agenthost"
	"github.com/augety121/mcp-state-twin/internal/bundle"
	"github.com/augety121/mcp-state-twin/internal/evaluator"
	"github.com/augety121/mcp-state-twin/internal/limits"
	"github.com/augety121/mcp-state-twin/internal/server"
	"github.com/augety121/mcp-state-twin/internal/task"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const PluginEvidenceFormat = "statetwin.dev/plugin-world-evidence/v1alpha1"

// CheckPluginCases uses the existing strict case admission without running any
// witness, and binds the selected Task to its quality inputs.
func CheckPluginCases(ctx context.Context, name string, read func(string, int) ([]byte, error), expected *task.Task) error {
	p, err := prepareCasesWith(ctx, name, read, bundle.OpenBytes)
	if err != nil || !same(pTask(p, expected.ID), expected) {
		return errors.New("PLUGIN_QUALITY_REFERENCE_INVALID")
	}
	return nil
}

// CheckPluginCaseTasks returns detached admitted definitions for operation-local
// catalog reuse. No witness is executed by this static admission.
func CheckPluginCaseTasks(ctx context.Context, name string, read func(string, int) ([]byte, error)) ([]*task.Task, error) {
	p, err := prepareCasesWith(ctx, name, read, bundle.OpenBytes)
	if err != nil {
		return nil, errors.New("PLUGIN_QUALITY_REFERENCE_INVALID")
	}
	var result []*task.Task
	for _, ref := range p.manifest.Tasks {
		result = append(result, p.tasks[ref.TaskID])
	}
	return result, nil
}

// RunPluginCases qualifies the admitted frozen inputs, not paths reread after
// preflight. Results remain scripted quality evidence, never model trajectories.
func RunPluginCases(ctx context.Context, name string, read func(string, int) ([]byte, error)) (*CaseReport, error) {
	p, err := prepareCasesWith(ctx, name, read, bundle.OpenBytes)
	if err != nil {
		return nil, err
	}
	return runCases(ctx, p, RunWitness)
}
func pTask(p *preparedCases, id string) *task.Task {
	if p == nil {
		return nil
	}
	return p.tasks[id]
}

// PluginTerminal belongs to the trusted harness. It must never be returned as
// an MCP tool result or inserted in the model's context.
type PluginTerminal struct {
	Format          string            `json:"format"`
	RuntimeVersion  string            `json:"runtimeVersion"`
	RuntimeRevision string            `json:"runtimeRevision"`
	Task            *task.Task        `json:"task"`
	Bundle          string            `json:"bundle"`
	Source          string            `json:"source"`
	Execution       string            `json:"execution"`
	Cleanup         string            `json:"cleanup"`
	FailureCode     string            `json:"failureCode"`
	ToolAttempts    int               `json:"toolAttempts"`
	View            evaluator.View    `json:"view"`
	Evaluation      *evaluator.Result `json:"evaluation"`
}

// PluginWorld shares the existing Task authority dispatcher (environment.step).
// It grants no state/control capability to an external MCP client.
type PluginWorld struct {
	mu        sync.Mutex
	accepting atomic.Bool
	slots     chan struct{}
	env       *environment
	terminal  *PluginTerminal
	ctx       context.Context
	cancel    context.CancelFunc
	finished  bool
}

func NewPluginWorld(parent context.Context, t *task.Task, raw []byte) (*PluginWorld, error) {
	b, err := bundle.OpenBytes(raw)
	if err != nil {
		return nil, errors.New("PLUGIN_PLAN_INVALID")
	}
	encoded, err := json.Marshal(t)
	if err != nil {
		return nil, errors.New("PLUGIN_PLAN_INVALID")
	}
	t, err = task.Decode(encoded)
	if err != nil || Admit(t, b) != nil {
		return nil, errors.New("PLUGIN_PLAN_INVALID")
	}
	for _, member := range b.Files {
		if err := safe(string(member), limits.MaxBundleExtracted); err != nil {
			return nil, errors.New("PLUGIN_PLAN_INVALID")
		}
	}
	ctx, cancel := context.WithTimeout(parent, time.Duration(t.Budgets.EpisodeSeconds)*time.Second)
	env, err := provision(ctx, t, b)
	if err != nil {
		cancel()
		return nil, errors.New("PLUGIN_STARTUP_FAILED")
	}
	w := &PluginWorld{env: env, ctx: ctx, cancel: cancel, slots: make(chan struct{}, 4), terminal: &PluginTerminal{
		Format: PluginEvidenceFormat, RuntimeVersion: server.Version, RuntimeRevision: server.Revision,
		Task: t, Bundle: base64.StdEncoding.EncodeToString(raw), Source: "external-harness",
		Execution: "running", Cleanup: "pending", View: evaluator.View{Before: env.before, Events: []evaluator.Event{}},
	}}
	w.accepting.Store(true)
	return w, nil
}

func (w *PluginWorld) Done() <-chan struct{} { return w.ctx.Done() }

// Tools is the public frozen projection; copy nested schemas before returning.
func (w *PluginWorld) Tools() []*mcp.Tool {
	w.mu.Lock()
	defer w.mu.Unlock()
	allowed := map[string]bool{}
	for _, name := range w.terminal.Task.Tools {
		allowed[name] = true
	}
	var result []*mcp.Tool
	for _, tool := range w.env.tools {
		if allowed[tool.Name] {
			raw, _ := json.Marshal(tool)
			var detached mcp.Tool
			_ = json.Unmarshal(raw, &detached)
			result = append(result, &detached)
		}
	}
	return result
}

func (w *PluginWorld) Call(ctx context.Context, call Call) (evaluator.Event, error) {
	if !w.accepting.Load() {
		return evaluator.Event{}, errors.New("PLUGIN_STATE_CONFLICT")
	}
	select {
	case w.slots <- struct{}{}:
		defer func() { <-w.slots }()
	default:
		return evaluator.Event{}, errors.New("PLUGIN_RESOURCE_LIMIT")
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.accepting.Load() || w.ctx.Err() != nil || ctx.Err() != nil {
		return evaluator.Event{}, errors.New("PLUGIN_INTERRUPTED")
	}
	t := w.terminal
	if t.ToolAttempts >= t.Task.Budgets.ToolAttempts {
		t.FailureCode = "PLUGIN_RESOURCE_LIMIT"
		w.accepting.Store(false)
		return evaluator.Event{}, errors.New(t.FailureCode)
	}
	raw, err := json.Marshal(call)
	if err != nil || len(raw) > limits.MaxInputBytes || safe(call, limits.MaxInputBytes) != nil {
		t.FailureCode = "PLUGIN_INPUT_INVALID"
		w.accepting.Store(false)
		return evaluator.Event{}, errors.New(t.FailureCode)
	}
	var detached Call
	if agenthost.DecodeDocument(raw, limits.MaxInputBytes, &detached) != nil {
		return evaluator.Event{}, errors.New("PLUGIN_INPUT_INVALID")
	}
	if _, err := agenthost.NormalizeNumbers(detached.Input); err != nil {
		return evaluator.Event{}, errors.New("PLUGIN_INPUT_INVALID")
	}
	callCtx, cancel := context.WithTimeout(w.ctx, time.Duration(t.Task.Budgets.RequestSeconds)*time.Second)
	stop := context.AfterFunc(ctx, cancel)
	defer func() { stop(); cancel() }()
	event, err := w.env.step(callCtx, detached)
	t.ToolAttempts++
	t.View.Events = append(t.View.Events, event)
	if err != nil || safe(t.View.Events, t.Task.Budgets.TraceBytes-(64<<10)) != nil {
		t.FailureCode = "PLUGIN_EXECUTION_FAILED"
		w.accepting.Store(false)
		return evaluator.Event{}, errors.New(t.FailureCode)
	}
	return event, nil
}

// Finish erects the barrier before waiting for the in-flight dispatcher.
// An answer is bounded task input, never a host-supplied grade or state.
func (w *PluginWorld) Finish(outcome string, answer any) (*PluginTerminal, error) {
	w.accepting.Store(false)
	w.mu.Lock()
	defer w.mu.Unlock()
	defer w.cancel()
	if w.finished {
		return clonePluginTerminal(w.terminal), nil
	}
	w.finished = true
	t := w.terminal
	t.Execution = "interrupted"
	if outcome == "completed" && w.ctx.Err() == nil && t.FailureCode == "" {
		t.Execution = "completed"
	} else if t.FailureCode == "" {
		t.FailureCode = "PLUGIN_INTERRUPTED"
	}
	finish, cancel := context.WithTimeout(context.Background(), time.Duration(t.Task.Budgets.CleanupSeconds)*time.Second)
	defer cancel()
	if safe(answer, 32<<10) != nil {
		t.Execution, t.FailureCode = "failed", "PLUGIN_INPUT_INVALID"
	} else {
		raw, _ := json.Marshal(answer)
		_ = json.Unmarshal(raw, &t.View.Answer)
	}
	var err error
	t.View.After, err = w.env.state(finish)
	if err == nil {
		var grade *evaluator.Evaluator
		grade, err = evaluator.Compile(t.Task)
		if err == nil {
			t.Evaluation, err = grade.Evaluate(finish, t.View)
		}
	}
	if err != nil {
		t.Execution, t.FailureCode = "failed", "PLUGIN_EVIDENCE_INVALID"
	}
	if w.env.close() != nil {
		t.Cleanup, t.FailureCode = "failed", "PLUGIN_CLEANUP_FAILED"
	} else {
		t.Cleanup = "complete"
	}
	if safe(t, limits.MaxReportBytes) != nil {
		return nil, errors.New("PLUGIN_EVIDENCE_INVALID")
	}
	return clonePluginTerminal(t), nil
}

func clonePluginTerminal(t *PluginTerminal) *PluginTerminal {
	raw, _ := json.Marshal(t)
	var result PluginTerminal
	_ = json.Unmarshal(raw, &result)
	return &result
}

func DecodePluginTerminal(raw []byte) (*PluginTerminal, error) {
	var t PluginTerminal
	if agenthost.DecodeDocument(raw, limits.MaxReportBytes, &t) != nil || t.Task == nil || safe(&t, limits.MaxReportBytes) != nil {
		return nil, errors.New("PLUGIN_EVIDENCE_INVALID")
	}
	for _, rule := range t.Task.Authority {
		if _, err := agenthost.NormalizeNumbers(rule.Equals); err != nil {
			return nil, errors.New("PLUGIN_EVIDENCE_INVALID")
		}
	}
	for _, e := range t.View.Events {
		if _, err := agenthost.NormalizeNumbers(e.Input); err != nil {
			return nil, errors.New("PLUGIN_EVIDENCE_INVALID")
		}
	}
	return &t, nil
}

// VerifyPluginTerminal replays world calls and original grading against the
// independently frozen expected Task/world; it does not authenticate a host.
func VerifyPluginTerminal(ctx context.Context, t *PluginTerminal, expected *task.Task, raw []byte) error {
	if t == nil || t.Task == nil || expected == nil || t.Format != PluginEvidenceFormat || t.Source != "external-harness" || t.RuntimeVersion != server.Version || t.RuntimeRevision != server.Revision || !same(t.Task, expected) || t.Execution != "completed" || t.FailureCode != "" || t.Cleanup != "complete" || t.Evaluation == nil || t.ToolAttempts != len(t.View.Events) || t.ToolAttempts > expected.Budgets.ToolAttempts || safe(t, limits.MaxReportBytes) != nil {
		return errors.New("PLUGIN_EVIDENCE_INVALID")
	}
	actual, err := base64.StdEncoding.DecodeString(t.Bundle)
	if err != nil {
		return errors.New("PLUGIN_EVIDENCE_INVALID")
	}
	a, err := bundle.OpenBytes(actual)
	if err != nil {
		return errors.New("PLUGIN_EVIDENCE_INVALID")
	}
	b, err := bundle.OpenBytes(raw)
	if err != nil || len(worldDifferences(a, b)) != 0 {
		return errors.New("PLUGIN_EVIDENCE_INVALID")
	}
	w, err := NewPluginWorld(ctx, expected, raw)
	if err != nil {
		return err
	}
	defer w.Finish("cancelled", nil)
	if !same(w.terminal.View.Before, t.View.Before) {
		return errors.New("PLUGIN_EVIDENCE_INVALID")
	}
	for _, want := range t.View.Events {
		got, err := w.Call(ctx, Call{Tool: want.Tool, Input: want.Input})
		if err != nil || !same(got, want) {
			return errors.New("PLUGIN_EVIDENCE_INVALID")
		}
	}
	got, err := w.Finish("completed", t.View.Answer)
	if err != nil || got.Execution != "completed" || got.Cleanup != "complete" || !same(got.View, t.View) || !same(got.Evaluation, t.Evaluation) {
		return errors.New("PLUGIN_EVIDENCE_INVALID")
	}
	return nil
}
