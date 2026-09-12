package agenteval

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	"github.com/augety121/mcp-state-twin/internal/agenthost"
	"github.com/augety121/mcp-state-twin/internal/bundle"
	"github.com/augety121/mcp-state-twin/internal/canonical"
	"github.com/augety121/mcp-state-twin/internal/evaluator"
	"github.com/augety121/mcp-state-twin/internal/limits"
	"github.com/augety121/mcp-state-twin/internal/logging"
	"github.com/augety121/mcp-state-twin/internal/server"
	"github.com/augety121/mcp-state-twin/internal/task"
)

const EvidenceFormat = "statetwin.dev/agent-evidence-offline/v1alpha1"

type AgentEvidence struct {
	Format  string        `json:"format" yaml:"format"`
	Bundle  string        `json:"bundle" yaml:"bundle"`
	Episode *AgentEpisode `json:"episode" yaml:"episode"`
}

// DecodeEvidence is strict JSON, with closed fields and the same numeric
// representation as Task/TwinSpec authoring. It does not trust claimed flags.
func DecodeEvidence(data []byte) (*AgentEvidence, error) {
	if len(data) > limits.MaxReportBytes || !json.Valid(data) {
		return nil, errors.New("EVIDENCE_INVALID")
	}
	var e AgentEvidence
	if err := agenthost.DecodeDocument(data, limits.MaxReportBytes, &e); err != nil {
		return nil, errors.New("EVIDENCE_INVALID")
	}
	if e.Format != EvidenceFormat || e.Episode == nil || e.Episode.Definition.Task == nil {
		return nil, errors.New("EVIDENCE_INVALID")
	}
	for _, rule := range e.Episode.Definition.Task.Authority {
		if _, err := agenthost.NormalizeNumbers(rule.Equals); err != nil {
			return nil, errors.New("EVIDENCE_INVALID")
		}
	}
	for _, event := range e.Episode.View.Events {
		if _, err := agenthost.NormalizeNumbers(event.Input); err != nil {
			return nil, errors.New("EVIDENCE_INVALID")
		}
	}
	return &e, nil
}

func same(a, b any) bool {
	x, e := canonical.JSON(a)
	if e != nil {
		return false
	}
	y, e := canonical.JSON(b)
	return e == nil && string(x) == string(y)
}

// replay checks a closed world's inputs, responses, effects and final grading.
// It does not call the model or authenticate unsigned provider provenance.
func replay(ctx context.Context, e *AgentEvidence, terminal bool) error {
	if e == nil || e.Format != EvidenceFormat || e.Episode == nil {
		return errors.New("EVIDENCE_INVALID")
	}
	r := e.Episode
	d := r.Definition
	if err := d.Config.Validate(); err != nil {
		return err
	}
	if r.Format != EpisodeFormat || r.Source != "mock-responses" || d.Projection != agenthost.Profile || d.Isolation != "in-process-offline-trusted" || d.ModelSnapshot != "not-applicable-mock" || d.RuntimeVersion != server.Version || d.RuntimeRevision != server.Revision {
		return errors.New("EVIDENCE_INCOMPATIBLE")
	}
	return replayWorld(ctx, e.Bundle, r, terminal)
}

func replayWorld(ctx context.Context, encoded string, r *AgentEpisode, terminal bool) error {
	d := r.Definition
	if r.ExecutionStatus != "completed" || r.FailureCode != "" || r.TerminalFailureCode != "" || r.CleanupFailureCode != "" || r.Evaluation == nil {
		return errors.New("EVIDENCE_PARTIAL")
	}
	if terminal && (r.CleanupStatus != "complete" || r.EvidenceStatus != "complete" || !r.WorldReplayable) {
		return errors.New("EVIDENCE_PARTIAL")
	}
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return errors.New("EVIDENCE_INVALID")
	}
	b, err := bundle.OpenBytes(raw)
	if err != nil || b.Digest != d.BundleDigest {
		return errors.New("EVIDENCE_INVALID")
	}
	if err = Admit(d.Task, b); err != nil {
		return err
	}
	if err = safe(r, limits.MaxReportBytes); err != nil {
		return err
	}
	for _, contents := range b.Files {
		if logging.ContainsSensitive(string(contents)) {
			return errors.New("DATA_POLICY_REJECTED")
		}
	}
	budgets := d.Task.Budgets
	if len(r.View.Events) > budgets.ToolAttempts || r.Usage.ToolAttempts != len(r.View.Events) || r.Usage.ModelRequests != len(r.RequestFrontiers) || r.Usage.ModelRequests < 1 || r.Usage.ModelRequests > budgets.ModelRequests || r.Usage.ContentBytes < 0 || r.Usage.ContentBytes > budgets.TraceBytes-(64<<10) {
		return errors.New("EVIDENCE_INVALID")
	}
	last := 0
	for i, frontier := range r.RequestFrontiers {
		if frontier < last || frontier > len(r.View.Events) || (i == 0 && frontier != 0) {
			return errors.New("EVIDENCE_INVALID")
		}
		last = frontier
	}
	if last != len(r.View.Events) {
		return errors.New("EVIDENCE_INVALID")
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(budgets.EpisodeSeconds)*time.Second)
	defer cancel()
	env, err := provision(ctx, d.Task, b)
	if err != nil {
		return err
	}
	defer env.close()
	if !same(env.before, r.View.Before) {
		return errors.New("EVIDENCE_REPLAY_MISMATCH")
	}
	for _, want := range r.View.Events {
		got, err := env.step(ctx, Call{Tool: want.Tool, Input: want.Input})
		if err != nil {
			return errors.New("EVIDENCE_REPLAY_MISMATCH")
		}
		if !same(got, want) {
			return errors.New("EVIDENCE_REPLAY_MISMATCH")
		}
	}
	after, err := env.state(ctx)
	if err != nil || !same(after, r.View.After) {
		return errors.New("EVIDENCE_REPLAY_MISMATCH")
	}
	grade, err := evaluator.Compile(d.Task)
	if err != nil {
		return err
	}
	result, err := grade.Evaluate(ctx, r.View)
	if err != nil || !same(result, r.Evaluation) {
		return errors.New("EVIDENCE_REPLAY_MISMATCH")
	}
	if err = env.close(); err != nil {
		return err
	}
	return nil
}

func VerifyEvidence(ctx context.Context, e *AgentEvidence) error { return replay(ctx, e, true) }

// RecordMock claims a new directory before execution. Only whitelisted
// synthetic world evidence is staged; transport requests/responses stay out.
// A verified replay closure is synced before the execution world is closed.
// Interrupted directories are inspect-only: no automatic resume or overwrite.
func RecordMock(ctx context.Context, root, out string, t *task.Task, bundleBytes []byte, c *RunConfig, m *agenthost.MockScript) (*AgentEpisode, error) {
	if err := task.PortablePath(out); err != nil {
		return nil, err
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	e := &AgentEvidence{Format: EvidenceFormat, Bundle: base64.StdEncoding.EncodeToString(bundleBytes)}
	return recordEpisode(ctx, root, out, t, bundleBytes, c, func(b *bundle.Artifact, stage stageEpisode) (*AgentEpisode, error) {
		return runMock(ctx, t, b, c, m, stage)
	}, func(r *AgentEpisode) any { e.Episode = r; return e }, func(ctx context.Context, r *AgentEpisode) error { e.Episode = r; return replay(ctx, e, false) })
}

type episodeRunner func(*bundle.Artifact, stageEpisode) (*AgentEpisode, error)

func recordEpisode(ctx context.Context, root, out string, t *task.Task, bundleBytes []byte, claim any, run episodeRunner, wrap func(*AgentEpisode) any, verify func(context.Context, *AgentEpisode) error) (*AgentEpisode, error) {
	return recordWithStorage(ctx, root, out, t, bundleBytes, claim, run, wrap, verify, openRootedEvidenceFS)
}

func recordWithStorage(ctx context.Context, root, out string, t *task.Task, bundleBytes []byte, claim any, run episodeRunner, wrap func(*AgentEpisode) any, verify func(context.Context, *AgentEpisode) error, open openEvidenceFS) (*AgentEpisode, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := task.PortablePath(out); err != nil {
		return nil, err
	}
	b, err := bundle.OpenBytes(bundleBytes)
	if err != nil {
		return nil, errors.New("TASK_INVALID")
	}
	if err = Admit(t, b); err != nil {
		return nil, err
	}
	// All archive members, including otherwise unused scripted scenarios, are
	// part of retained data and must pass policy before any filesystem write.
	for _, contents := range b.Files {
		if logging.ContainsSensitive(string(contents)) {
			return nil, errors.New("DATA_POLICY_REJECTED")
		}
	}
	w, err := claimEvidence(root, out, claim, open)
	if err != nil {
		return nil, err
	}
	defer w.fs.Close()
	closed := false
	staged := false
	r, err := run(b, func(finish context.Context, r *AgentEpisode) error {
		if err := safe(r, limits.MaxReportBytes); err != nil {
			return err
		}
		if r.ExecutionStatus == "completed" && r.Evaluation != nil && r.FailureCode == "" {
			if err := verify(finish, r); err != nil {
				return err
			}
			closed = true
		}
		if err := w.write("closure.json", wrap(r)); err != nil {
			return err
		}
		staged = true
		return nil
	})
	if err != nil {
		return nil, err
	}
	if !staged || r == nil {
		return nil, errors.New("EVIDENCE_STAGE_MISSING")
	}
	if closed && r.CleanupStatus == "complete" {
		r.WorldReplayable = true
		r.EvidenceStatus = "complete"
	}
	if err = w.write("terminal.pending.json", wrap(r)); err != nil {
		return nil, err
	}
	if err = w.publish(); err != nil {
		return r, err
	}
	return r, nil
}
