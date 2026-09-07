// Package agenteval provides the bounded offline Task witness lane. Witness
// playback validates solvability and grading; it is never live-agent evidence.
package agenteval

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/augety121/mcp-state-twin/internal/bundle"
	"github.com/augety121/mcp-state-twin/internal/engine"
	"github.com/augety121/mcp-state-twin/internal/evaluator"
	"github.com/augety121/mcp-state-twin/internal/limits"
	"github.com/augety121/mcp-state-twin/internal/logging"
	"github.com/augety121/mcp-state-twin/internal/server"
	"github.com/augety121/mcp-state-twin/internal/spec"
	"github.com/augety121/mcp-state-twin/internal/store"
	"github.com/augety121/mcp-state-twin/internal/strictyaml"
	"github.com/augety121/mcp-state-twin/internal/task"
	"github.com/augety121/mcp-state-twin/internal/world"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Call struct {
	Tool  string         `json:"tool" yaml:"tool"`
	Input map[string]any `json:"input" yaml:"input"`
}
type Witness struct {
	Kind          string `json:"kind" yaml:"kind"`
	TaskID        string `json:"taskId" yaml:"taskId"`
	SyntheticOnly bool   `json:"syntheticOnly" yaml:"syntheticOnly"`
	Calls         []Call `json:"calls" yaml:"calls"`
	Answer        any    `json:"answer" yaml:"answer"`
}
type Report struct {
	Format          string            `json:"format"`
	Source          string            `json:"source"`
	TaskID          string            `json:"taskId"`
	ExecutionStatus string            `json:"executionStatus"`
	CleanupStatus   string            `json:"cleanupStatus"`
	Evaluation      *evaluator.Result `json:"evaluation"`
	View            evaluator.View    `json:"view"`
}

func DecodeWitness(data []byte) (*Witness, error) {
	var w Witness
	if err := strictyaml.DecodeOneWithDepth(data, task.MaxBytes, 32, "TaskWitness", &w); err != nil {
		return nil, errors.New("TASK_INVALID: malformed witness")
	}
	if w.Kind != "TaskWitness" || !w.SyntheticOnly || len(w.Calls) > 32 {
		return nil, errors.New("TASK_INVALID: unsupported witness")
	}
	if err := safe(w, task.MaxBytes); err != nil {
		return nil, err
	}
	return &w, nil
}

func Load(root, name string) (*task.Task, *bundle.Artifact, error) {
	data, err := task.ReadFile(root, name, task.MaxBytes)
	if err != nil {
		return nil, nil, err
	}
	t, err := task.Decode(data)
	if err != nil {
		return nil, nil, err
	}
	data, err = task.ReadFile(root, t.Bundle, limits.MaxBundleCompressed)
	if err != nil {
		return nil, nil, err
	}
	b, err := bundle.OpenBytes(data)
	if err != nil {
		return nil, nil, errors.New("TASK_INVALID: invalid bundle")
	}
	if err = Admit(t, b); err != nil {
		return nil, nil, err
	}
	return t, b, nil
}

func Admit(t *task.Task, b *bundle.Artifact) error {
	if b == nil {
		return errors.New("TASK_INVALID: bundle required")
	}
	twin, err := spec.Decode(b.Files[b.Manifest.Spec])
	if err != nil {
		return err
	}
	if err = t.AdmitSurface(twin); err != nil {
		return err
	}
	if _, err = evaluator.Compile(t); err != nil {
		return err
	}
	if err = safe(t, task.MaxBytes); err != nil {
		return err
	}
	// Entire fixture/tool text must be synthetic and pass the same bounded
	// sensitive-pattern refusal used for the offline report.
	for _, member := range []string{b.Manifest.Spec, b.Manifest.Fixture} {
		if logging.ContainsSensitive(string(b.Files[member])) {
			return errors.New("DATA_POLICY_REJECTED")
		}
	}
	return nil
}

// localTransport runs actual MCP HTTP/JSON-RPC through the official SDK and
// data-plane handler without a listening port, DNS, or network fallback.
type localTransport struct{ handler http.Handler }

func (t localTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.URL.Host != "statetwin.invalid" || r.URL.Path != "/mcp/witness" {
		return nil, errors.New("local MCP route refused")
	}
	if err := r.Context().Err(); err != nil {
		return nil, err
	}
	w := httptest.NewRecorder()
	t.handler.ServeHTTP(w, r)
	return w.Result(), nil
}

func RunWitness(ctx context.Context, t *task.Task, b *bundle.Artifact, w *Witness) (report *Report, err error) {
	if err = Admit(t, b); err != nil {
		return nil, err
	}
	if w == nil || w.Kind != "TaskWitness" || !w.SyntheticOnly || w.TaskID != t.ID || len(w.Calls) > t.Budgets.ToolAttempts {
		return nil, errors.New("TASK_INVALID: witness identity or budget")
	}
	if err = safe(w, task.MaxBytes); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(t.Budgets.EpisodeSeconds)*time.Second)
	defer cancel()
	twin, _ := spec.Decode(b.Files[b.Manifest.Spec])
	initial, err := world.DecodeStrict(b.Files[b.Manifest.Fixture])
	if err != nil {
		return nil, err
	}
	db, err := store.Open(":memory:")
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			err = errors.New("CLEANUP_FAILED: world store")
			if report != nil {
				report.CleanupStatus = "failed"
			}
		}
	}()
	runtime, err := engine.New(twin, db)
	if err != nil {
		return nil, err
	}
	if err = runtime.Initialize(ctx, "witness", initial); err != nil {
		return nil, err
	}
	if t.FaultTool != "" {
		_, err = db.InstallFault(ctx, store.FaultPlan{ID: "task-after-commit", BranchID: "witness", ToolName: t.FaultTool, Phase: store.FaultPhaseAfterCommitBeforeResponse, ErrorClass: "TIMEOUT_AFTER_EFFECT", Message: "synthetic response failure", RemainingCount: 1}, nil)
		if err != nil {
			return nil, err
		}
	}
	base, err := db.Branch(ctx, "witness")
	if err != nil {
		return nil, err
	}
	transport := localTransport{server.NewDataPlane(runtime)}
	client := mcp.NewClient(&mcp.Implementation{Name: "task-witness", Version: "experimental"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: "https://statetwin.invalid/mcp/witness", HTTPClient: &http.Client{Transport: transport, Timeout: time.Duration(t.Budgets.RequestSeconds) * time.Second}}, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := session.Close(); closeErr != nil {
			err = errors.New("CLEANUP_FAILED: MCP session")
			if report != nil {
				report.CleanupStatus = "failed"
			}
		}
	}()
	listed, err := session.ListTools(ctx, nil)
	if err != nil {
		return nil, err
	}
	available := map[string]bool{}
	for _, tool := range listed.Tools {
		available[tool.Name] = true
	}
	for _, name := range t.Tools {
		if !available[name] {
			return nil, errors.New("HOST_NOT_READY: required tool missing")
		}
	}
	report = &Report{Format: "statetwin.dev/task-witness-report/v1alpha1", Source: "scripted-witness", TaskID: t.ID, ExecutionStatus: "completed", CleanupStatus: "complete", View: evaluator.View{Before: base.State, Events: []evaluator.Event{}, Answer: w.Answer}}
	effectful := map[string]bool{}
	for _, tool := range twin.Tools {
		effectful[tool.Name] = len(tool.Effects) > 0
	}
	for i, call := range w.Calls {
		if err = ctx.Err(); err != nil {
			return nil, err
		}
		e := evaluator.Event{Sequence: i, Tool: call.Tool, Input: call.Input, Authorized: t.Authorized(call.Tool, call.Input)}
		if !e.Authorized {
			e.ErrorClass = "AUTHORITY_DENIED"
			e.Result = map[string]any{"error": map[string]any{"code": e.ErrorClass}}
			e.Delivered = true
		} else if validationErr := runtime.ValidateScheduledAction(call.Tool, call.Input); validationErr != nil {
			e.ErrorClass = "INVALID_INPUT"
			e.Result = map[string]any{"error": map[string]any{"code": e.ErrorClass}}
			e.Delivered = true
		} else {
			result, callErr := session.CallTool(ctx, &mcp.CallToolParams{Name: call.Tool, Arguments: call.Input})
			if callErr != nil {
				return nil, errors.New("INFRASTRUCTURE_ERROR: local MCP call")
			}
			e.Dispatched = true
			e.Delivered = true
			if result.StructuredContent != nil {
				e.Result = result.StructuredContent
			} else {
				for _, c := range result.Content {
					if text, ok := c.(*mcp.TextContent); ok {
						if decodeErr := json.Unmarshal([]byte(text.Text), &e.Result); decodeErr != nil {
							return nil, errors.New("INFRASTRUCTURE_ERROR: malformed MCP result")
						}
						break
					}
				}
			}
			encoded, _ := json.Marshal(e.Result)
			var object map[string]any
			if json.Unmarshal(encoded, &object) != nil {
				return nil, errors.New("INFRASTRUCTURE_ERROR: MCP result is not an object")
			}
			if result.IsError {
				if detail, ok := object["error"].(map[string]any); ok {
					e.ErrorClass, _ = detail["code"].(string)
				}
				if e.ErrorClass == "" {
					return nil, errors.New("INFRASTRUCTURE_ERROR: missing tool error code")
				}
			}
			current, readErr := db.Branch(ctx, "witness")
			if readErr != nil {
				return nil, readErr
			}
			faults, readErr := db.FaultEvents(ctx, "witness")
			if readErr != nil {
				return nil, readErr
			}
			for _, fault := range faults {
				if fault.CallIndex == current.CallCount {
					e.FaultPhase = fault.Phase
				}
			}
			e.EffectCommitted = effectful[call.Tool] && (e.ErrorClass == "" || e.FaultPhase == store.FaultPhaseAfterCommitBeforeResponse)
		}
		if err = safe(e, t.Budgets.ResponseBytes); err != nil {
			return nil, err
		}
		report.View.Events = append(report.View.Events, e)
		if err = safe(report.View.Events, t.Budgets.TraceBytes-(64<<10)); err != nil {
			return nil, err
		}
	}
	terminal, err := db.Branch(ctx, "witness")
	if err != nil {
		return nil, err
	}
	report.View.After = terminal.State
	grade, err := evaluator.Compile(t)
	if err != nil {
		return nil, err
	}
	report.Evaluation, err = grade.Evaluate(ctx, report.View)
	if err != nil {
		return nil, err
	}
	if err = safe(report, limits.MaxReportBytes); err != nil {
		return nil, err
	}
	return report, nil
}

func safe(v any, max int) error {
	if err := limits.ValidateJSON(v, max); err != nil {
		return errors.New("RESOURCE_LIMIT: task witness content")
	}
	b, err := json.Marshal(v)
	if err != nil {
		return errors.New("DATA_POLICY_REJECTED")
	}
	if logging.ContainsSensitive(string(b)) {
		return errors.New("DATA_POLICY_REJECTED")
	}
	return nil
}
