package agenteval

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/augety121/mcp-state-twin/internal/bundle"
	"github.com/augety121/mcp-state-twin/internal/engine"
	"github.com/augety121/mcp-state-twin/internal/evaluator"
	"github.com/augety121/mcp-state-twin/internal/server"
	"github.com/augety121/mcp-state-twin/internal/spec"
	"github.com/augety121/mcp-state-twin/internal/store"
	"github.com/augety121/mcp-state-twin/internal/task"
	"github.com/augety121/mcp-state-twin/internal/world"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// environment owns a single fresh in-memory world and an SDK MCP session.
// It is private to the trusted sequential harness, never a model capability.
type environment struct {
	task      *task.Task
	db        *store.Store
	runtime   *engine.Runtime
	session   *mcp.ClientSession
	tools     []*mcp.Tool
	before    *world.State
	effectful map[string]bool
	sequence  int
}

func provision(ctx context.Context, t *task.Task, b *bundle.Artifact) (env *environment, err error) {
	if err = Admit(t, b); err != nil {
		return nil, err
	}
	twin, _ := spec.Decode(b.Files[b.Manifest.Spec])
	initial, err := world.DecodeStrict(b.Files[b.Manifest.Fixture])
	if err != nil {
		return nil, err
	}
	db, err := store.Open(":memory:")
	if err != nil {
		return nil, err
	}
	e := &environment{task: t, db: db, effectful: map[string]bool{}}
	defer func() {
		if err != nil {
			if closeErr := e.close(); closeErr != nil {
				err = closeErr
			}
		}
	}()
	e.runtime, err = engine.New(twin, db)
	if err != nil {
		return nil, err
	}
	if err = e.runtime.Initialize(ctx, "witness", initial); err != nil {
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
	e.before = base.State
	client := mcp.NewClient(&mcp.Implementation{Name: "agent-task-offline", Version: "experimental"}, nil)
	e.session, err = client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: "https://statetwin.invalid/mcp/witness", HTTPClient: &http.Client{Transport: localTransport{server.NewDataPlane(e.runtime)}, Timeout: time.Duration(t.Budgets.RequestSeconds) * time.Second}}, nil)
	if err != nil {
		return nil, err
	}
	listed, err := e.session.ListTools(ctx, nil)
	if err != nil {
		return nil, err
	}
	e.tools = listed.Tools
	available := map[string]bool{}
	for _, tool := range e.tools {
		available[tool.Name] = true
	}
	for _, name := range t.Tools {
		if !available[name] {
			return nil, errors.New("HOST_NOT_READY")
		}
	}
	for _, tool := range twin.Tools {
		e.effectful[tool.Name] = len(tool.Effects) > 0
	}
	return e, nil
}

func (e *environment) close() error {
	var err error
	if e.session != nil {
		if e.session.Close() != nil {
			err = errors.New("CLEANUP_FAILED")
		}
		e.session = nil
	}
	if e.db != nil {
		if e.db.Close() != nil {
			err = errors.New("CLEANUP_FAILED")
		}
		e.db = nil
	}
	return err
}

func (e *environment) state(ctx context.Context) (*world.State, error) {
	b, err := e.db.Branch(ctx, "witness")
	if err != nil {
		return nil, err
	}
	return b.State, nil
}

func (env *environment) step(ctx context.Context, call Call) (e evaluator.Event, err error) {
	e = evaluator.Event{Sequence: env.sequence, Tool: call.Tool, Input: call.Input, Authorized: env.task.Authorized(call.Tool, call.Input)}
	env.sequence++
	if err = ctx.Err(); err != nil {
		return e, err
	}
	if !e.Authorized {
		e.ErrorClass = "AUTHORITY_DENIED"
	} else if env.runtime.ValidateScheduledAction(call.Tool, call.Input) != nil {
		e.ErrorClass = "INVALID_INPUT"
	}
	if e.ErrorClass != "" {
		e.Result = map[string]any{"error": map[string]any{"code": e.ErrorClass}}
		e.Delivered = true
		return e, safe(e, env.task.Budgets.ResponseBytes)
	}
	e.Dispatched = true
	r, err := env.session.CallTool(ctx, &mcp.CallToolParams{Name: call.Tool, Arguments: call.Input})
	if err != nil {
		return e, errors.New("INFRASTRUCTURE_ERROR")
	}
	e.Delivered = true
	if r.StructuredContent != nil {
		e.Result = r.StructuredContent
	} else {
		for _, c := range r.Content {
			if text, ok := c.(*mcp.TextContent); ok {
				if json.Unmarshal([]byte(text.Text), &e.Result) != nil {
					return e, errors.New("INFRASTRUCTURE_ERROR")
				}
				break
			}
		}
	}
	encoded, encodeErr := json.Marshal(e.Result)
	var object map[string]any
	if encodeErr != nil || json.Unmarshal(encoded, &object) != nil || object == nil {
		return e, errors.New("INFRASTRUCTURE_ERROR")
	}
	if r.IsError {
		if detail, ok := object["error"].(map[string]any); ok {
			e.ErrorClass, _ = detail["code"].(string)
		}
		if e.ErrorClass == "" {
			return e, errors.New("INFRASTRUCTURE_ERROR")
		}
	}
	current, err := env.db.Branch(ctx, "witness")
	if err != nil {
		return e, err
	}
	faults, err := env.db.FaultEvents(ctx, "witness")
	if err != nil {
		return e, err
	}
	for _, fault := range faults {
		if fault.CallIndex == current.CallCount {
			e.FaultPhase = fault.Phase
		}
	}
	e.EffectCommitted = env.effectful[call.Tool] && (e.ErrorClass == "" || e.FaultPhase == store.FaultPhaseAfterCommitBeforeResponse)
	return e, safe(e, env.task.Budgets.ResponseBytes)
}
