package agenteval

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/augety121/mcp-state-twin/internal/agenthost"
	"github.com/augety121/mcp-state-twin/internal/bundle"
	"github.com/augety121/mcp-state-twin/internal/evaluator"
	"github.com/augety121/mcp-state-twin/internal/limits"
	"github.com/augety121/mcp-state-twin/internal/server"
	"github.com/augety121/mcp-state-twin/internal/strictyaml"
	"github.com/augety121/mcp-state-twin/internal/task"
)

const EpisodeFormat = "statetwin.dev/agent-episode-offline/v1alpha1"
const ConfigFormat = "statetwin.dev/agent-run-offline/v1alpha1"

var runID = regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}$`)

type RunConfig struct {
	Format          string `json:"format" yaml:"format"`
	TrialID         string `json:"trialId" yaml:"trialId"`
	Profile         string `json:"profile" yaml:"profile"`
	Model           string `json:"model" yaml:"model"`
	MaxOutputTokens int    `json:"maxOutputTokens" yaml:"maxOutputTokens"`
	SyntheticOnly   bool   `json:"syntheticOnly" yaml:"syntheticOnly"`
}

func DecodeRun(data []byte) (*RunConfig, error) {
	var c RunConfig
	if strictyaml.DecodeOneWithDepth(data, 16<<10, 16, "AgentRunConfig", &c) != nil {
		return nil, errors.New("HOST_PROFILE_UNSUPPORTED")
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

func (c *RunConfig) Validate() error {
	if c == nil || c.Format != ConfigFormat || !runID.MatchString(c.TrialID) || c.Profile != agenthost.Profile || !c.SyntheticOnly || !strings.HasPrefix(c.Model, "mock-") || !runID.MatchString(c.Model) || c.MaxOutputTokens < 1 || c.MaxOutputTokens > 8192 {
		return errors.New("HOST_PROFILE_UNSUPPORTED")
	}
	return nil
}

type RunDefinition struct {
	Config          RunConfig  `json:"config"`
	Task            *task.Task `json:"task"`
	BundleDigest    string     `json:"bundleDigest"`
	RuntimeVersion  string     `json:"runtimeVersion"`
	RuntimeRevision string     `json:"runtimeRevision"`
	Isolation       string     `json:"isolation"`
	Projection      string     `json:"projection"`
	ModelSnapshot   string     `json:"modelSnapshot"`
}

type AgentEpisode struct {
	Format           string            `json:"format"`
	Source           string            `json:"source"`
	Definition       RunDefinition     `json:"definition"`
	ExecutionStatus  string            `json:"executionStatus"`
	FailureCode      string            `json:"failureCode,omitempty"`
	EvidenceStatus   string            `json:"evidenceStatus"`
	CleanupStatus    string            `json:"cleanupStatus"`
	Usage            agenthost.Usage   `json:"usage"`
	RequestFrontiers []int             `json:"requestFrontiers"`
	Evaluation       *evaluator.Result `json:"evaluation,omitempty"`
	View             evaluator.View    `json:"view"`
	// The separate bounded evidence artifact is added only after the world
	// replay closure has been checked; this in-memory object is not a seal.
	WorldReplayable bool `json:"worldReplayable"`
}

// RunMock executes the turn/dispatch lifecycle against synthetic Responses
// items. It proves host behavior, not model intelligence or live compatibility.
// There is no remote I/O; all in-flight work is joined synchronously.
func RunMock(parent context.Context, t *task.Task, b *bundle.Artifact, c *RunConfig, m *agenthost.MockScript) (r *AgentEpisode, err error) {
	return runMock(parent, t, b, c, m, nil)
}

func runMock(parent context.Context, t *task.Task, b *bundle.Artifact, c *RunConfig, m *agenthost.MockScript, stage func(*AgentEpisode) error) (r *AgentEpisode, err error) {
	if err = c.Validate(); err != nil {
		return nil, err
	}
	if err = Admit(t, b); err != nil {
		return nil, err
	}
	// Freeze the private definition; later caller edits cannot rewrite this run.
	taskBytes, copyErr := json.Marshal(t)
	if copyErr != nil {
		return nil, errors.New("TASK_INVALID")
	}
	t, err = task.Decode(taskBytes)
	if err != nil {
		return nil, err
	}
	configCopy := *c
	c = &configCopy
	if m == nil || m.Kind != "MockResponses" || !m.SyntheticOnly || len(m.Responses) == 0 || len(m.Responses) > 16 {
		return nil, errors.New("HOST_PROFILE_UNSUPPORTED")
	}
	ctx, cancel := context.WithTimeout(parent, time.Duration(t.Budgets.EpisodeSeconds)*time.Second)
	defer cancel()
	env, err := provision(ctx, t, b)
	if err != nil {
		return nil, err
	}
	defer func() {
		if env != nil {
			_ = env.close()
		}
	}()
	host, err := agenthost.New(c.Model, c.MaxOutputTokens, t, env.tools)
	if err != nil {
		return nil, err
	}
	defer host.Stop()
	r = &AgentEpisode{Format: EpisodeFormat, Source: "mock-responses", Definition: RunDefinition{Config: *c, Task: t, BundleDigest: b.Digest, RuntimeVersion: server.Version, RuntimeRevision: server.Revision, Isolation: "in-process-offline-trusted", Projection: agenthost.Profile, ModelSnapshot: "not-applicable-mock"}, ExecutionStatus: "running", EvidenceStatus: "partial", CleanupStatus: "pending", View: evaluator.View{Before: env.before, Events: []evaluator.Event{}}}
	failure := error(nil)
	for _, raw := range m.Responses {
		if _, failure = host.Request(ctx); failure != nil {
			break
		}
		r.RequestFrontiers = append(r.RequestFrontiers, len(r.View.Events))
		for i := range r.View.Events {
			r.View.Events[i].Delivered = true
		}
		var turn agenthost.Turn
		turn, failure = host.Accept(ctx, raw)
		if failure != nil {
			break
		}
		if turn.Final {
			r.View.Answer = turn.Answer
			r.ExecutionStatus = "completed"
			break
		}
		for _, call := range turn.Calls {
			if failure = host.AdmitTool(ctx, call.Reference); failure != nil {
				break
			}
			var e evaluator.Event
			e, failure = env.step(ctx, Call{Tool: call.Tool, Input: call.Input})
			if failure != nil {
				break
			}
			// Delivery refers to inclusion in the next model request, not merely
			// the local MCP client receiving the result. It starts false here.
			e.Delivered = false
			r.View.Events = append(r.View.Events, e)
			if failure = host.Deliver(ctx, call.Reference, e.Result); failure != nil {
				break
			}
		}
		if failure != nil {
			break
		}
		// A later request is the delivery acknowledgement in this offline lane.
		if len(r.View.Events) > t.Budgets.ToolAttempts {
			failure = errors.New("BUDGET_EXHAUSTED")
			break
		}
	}
	host.Stop()
	r.Usage = host.Usage()
	if failure != nil {
		r.ExecutionStatus = status(failure)
		r.FailureCode = code(failure)
	} else if r.ExecutionStatus != "completed" {
		r.ExecutionStatus = "host_error"
		r.FailureCode = "MOCK_EXHAUSTED"
	}
	// A canceled execution context never prevents bounded terminal inspection.
	finish, finishCancel := context.WithTimeout(context.Background(), time.Duration(t.Budgets.CleanupSeconds)*time.Second)
	defer finishCancel()
	r.View.After, err = env.state(finish)
	if err == nil {
		var grade *evaluator.Evaluator
		grade, err = evaluator.Compile(t)
		if err == nil {
			r.Evaluation, err = grade.Evaluate(finish, r.View)
		}
	}
	if err != nil {
		r.EvidenceStatus = "partial"
		r.FailureCode = "TERMINAL_INSPECTION_FAILED"
		err = nil
	}
	if stage != nil {
		if stageErr := stage(r); stageErr != nil {
			return nil, stageErr
		}
	}
	if closeErr := env.close(); closeErr != nil {
		r.CleanupStatus = "failed"
		r.FailureCode = "CLEANUP_FAILED"
	} else {
		r.CleanupStatus = "complete"
	}
	env = nil
	if err = safe(r, limits.MaxReportBytes); err != nil {
		return nil, err
	}
	return r, nil
}

func code(err error) string {
	if errors.Is(err, context.Canceled) {
		return "CANCELED"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "EPISODE_TIMEOUT"
	}
	for _, c := range []string{"BUDGET_EXHAUSTED", "HOST_PROTOCOL_ERROR", "HOST_PROFILE_UNSUPPORTED", "DATA_POLICY_REJECTED", "INFRASTRUCTURE_ERROR", "EPISODE_TIMEOUT", "EPISODE_STOPPED"} {
		if err.Error() == c {
			return c
		}
	}
	return "INFRASTRUCTURE_ERROR"
}
func status(err error) string {
	switch code(err) {
	case "CANCELED":
		return "canceled"
	case "EPISODE_TIMEOUT":
		return "timed_out"
	case "BUDGET_EXHAUSTED":
		return "budget_exhausted"
	default:
		return "host_error"
	}
}
