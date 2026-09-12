package agenteval

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	"github.com/augety121/mcp-state-twin/internal/agentapi"
	"github.com/augety121/mcp-state-twin/internal/agenthost"
	"github.com/augety121/mcp-state-twin/internal/bundle"
	"github.com/augety121/mcp-state-twin/internal/limits"
	"github.com/augety121/mcp-state-twin/internal/server"
	"github.com/augety121/mcp-state-twin/internal/spec"
	"github.com/augety121/mcp-state-twin/internal/task"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const LiveProfile = agentapi.Profile
const LiveConfigFormat = "statetwin.dev/agent-run-live/v1alpha1"
const LiveEpisodeFormat = "statetwin.dev/agent-episode-live/v1alpha1"
const LiveEvidenceFormat = "statetwin.dev/agent-evidence-live/v1alpha1"

type LiveEvidence struct {
	Format    string             `json:"format"`
	Plan      *agentapi.Plan     `json:"plan"`
	StartedAt string             `json:"startedAt"`
	Receipts  []agentapi.Receipt `json:"receipts"`
	Bundle    string             `json:"bundle"`
	Episode   *AgentEpisode      `json:"episode"`
}

func liveConfig(p *agentapi.Plan) *RunConfig {
	return &RunConfig{Format: LiveConfigFormat, TrialID: p.ID, Profile: LiveProfile, Model: p.Model, MaxOutputTokens: p.MaxOutputTokens, SyntheticOnly: true}
}

// PreflightLive performs only local structural checks, never credentials,
// network access, provider availability, billing or approval inference.
func PreflightLive(p *agentapi.Plan, b *bundle.Artifact) error {
	if err := p.Validate(); err != nil {
		return err
	}
	if b == nil || p.BundleDigest != b.Digest {
		return errors.New("LIVE_PLAN_BINDING_MISMATCH")
	}
	if err := Admit(p.Task, b); err != nil {
		return err
	}
	twin, err := spec.Decode(b.Files[b.Manifest.Spec])
	if err != nil {
		return err
	}
	var tools []*mcp.Tool
	for _, s := range twin.Tools {
		tools = append(tools, &mcp.Tool{Name: s.Name, Description: s.Description, InputSchema: s.InputSchema})
	}
	host, err := agenthost.NewResponses(p.Model, p.MaxOutputTokens, p.Task, tools)
	if err != nil {
		return err
	}
	host.Stop()
	return safe(p, limits.MaxReportBytes)
}

// RecordLive is the sole production entry point for the new API bridge.
// It uses the fixed transport, claims a plan-specific directory before the
// first POST, and never treats an existing/partial directory as resumable.
func RecordLive(ctx context.Context, root string, p *agentapi.Plan, bundleBytes []byte, key string, allowLive bool) (*AgentEpisode, error) {
	return recordLive(ctx, root, p, bundleBytes, allowLive, "provider-live", func(p *agentapi.Plan) (liveExchange, func(), error) {
		c, err := agentapi.New(p, key, allowLive)
		if err != nil {
			return nil, nil, err
		}
		return c.Exchange, c.Close, nil
	})
}

type liveExchange func(context.Context, []byte) ([]byte, agentapi.Receipt, error)

// Injection is package-private for contract tests, which must label their
// artifacts contract-test. It is not a public custom-endpoint escape hatch.
func recordLive(ctx context.Context, root string, p *agentapi.Plan, bundleBytes []byte, allow bool, source string, makeClient func(*agentapi.Plan) (liveExchange, func(), error)) (*AgentEpisode, error) {
	if err := p.Authorize(time.Now(), allow); err != nil {
		return nil, err
	}
	data, err := json.Marshal(p)
	if err != nil {
		return nil, errors.New("LIVE_PLAN_INVALID")
	}
	p, err = agentapi.DecodePlan(data)
	if err != nil {
		return nil, err
	}
	b, err := bundle.OpenBytes(bundleBytes)
	if err != nil {
		return nil, errors.New("TASK_INVALID")
	}
	if err = PreflightLive(p, b); err != nil {
		return nil, err
	}
	if source != "provider-live" && source != "contract-test" {
		return nil, errors.New("HOST_PROFILE_UNSUPPORTED")
	}
	e := &LiveEvidence{Format: LiveEvidenceFormat, Plan: p, StartedAt: time.Now().UTC().Format(time.RFC3339Nano), Receipts: []agentapi.Receipt{}, Bundle: base64.StdEncoding.EncodeToString(bundleBytes)}
	return recordEpisode(ctx, root, p.OutputDirectory(), p.Task, bundleBytes, p, func(b *bundle.Artifact, stage stageEpisode) (*AgentEpisode, error) {
		exchange, close, err := makeClient(p)
		if err != nil {
			return nil, err
		}
		defer close()
		driver := loopDriver{live: true, steps: p.MaxRequests, source: source, exhausted: "BUDGET_EXHAUSTED", next: func(ctx context.Context, request []byte, _ int) ([]byte, error) {
			raw, receipt, err := exchange(ctx, request)
			if receipt.Sequence > 0 {
				e.Receipts = append(e.Receipts, receipt)
			}
			return raw, err
		}}
		return runLoop(ctx, p.Task, b, liveConfig(p), driver, stage)
	}, func(r *AgentEpisode) any { e.Episode = r; return e }, func(ctx context.Context, r *AgentEpisode) error { e.Episode = r; return verifyLive(ctx, e, false) })
}

func DecodeLiveEvidence(data []byte) (*LiveEvidence, error) {
	var e LiveEvidence
	if agenthost.DecodeDocument(data, limits.MaxReportBytes, &e) != nil || e.Format != LiveEvidenceFormat || e.Plan == nil || e.Episode == nil || e.Episode.Definition.Task == nil {
		return nil, errors.New("EVIDENCE_INVALID")
	}
	for _, ta := range []*task.Task{e.Plan.Task, e.Episode.Definition.Task} {
		if ta == nil {
			return nil, errors.New("EVIDENCE_INVALID")
		}
		for _, rule := range ta.Authority {
			if _, err := agenthost.NormalizeNumbers(rule.Equals); err != nil {
				return nil, errors.New("EVIDENCE_INVALID")
			}
		}
	}
	for _, ev := range e.Episode.View.Events {
		if _, err := agenthost.NormalizeNumbers(ev.Input); err != nil {
			return nil, errors.New("EVIDENCE_INVALID")
		}
	}
	return &e, nil
}

// Verification replays local world transitions. Unsigned receipts are harness
// assertions, not independent proof of provider origin or actual charges.
func VerifyLiveEvidence(ctx context.Context, e *LiveEvidence) error { return verifyLive(ctx, e, true) }
func verifyLive(ctx context.Context, e *LiveEvidence, terminal bool) error {
	if e == nil || e.Format != LiveEvidenceFormat || e.Plan == nil || e.Episode == nil {
		return errors.New("EVIDENCE_INVALID")
	}
	started, err := time.Parse(time.RFC3339Nano, e.StartedAt)
	if err != nil || e.Plan.Authorize(started, true) != nil {
		return errors.New("EVIDENCE_INVALID")
	}
	r := e.Episode
	d := r.Definition
	if r.Format != LiveEpisodeFormat || (r.Source != "provider-live" && r.Source != "contract-test") || !same(d.Task, e.Plan.Task) || !same(d.Config, liveConfig(e.Plan)) || d.BundleDigest != e.Plan.BundleDigest || d.Isolation != "in-process-tools-fixed-provider-egress" || d.Projection != LiveProfile || d.ModelSnapshot != "unknown" || d.RuntimeVersion != server.Version || d.RuntimeRevision != server.Revision {
		return errors.New("EVIDENCE_INCOMPATIBLE")
	}
	if r.ExecutionStatus != "completed" || r.FailureCode != "" {
		return errors.New("EVIDENCE_PARTIAL")
	}
	if len(e.Receipts) != r.Usage.ModelRequests || len(e.Receipts) < 1 || len(e.Receipts) > e.Plan.MaxRequests {
		return errors.New("EVIDENCE_INVALID")
	}
	last := started
	for i, receipt := range e.Receipts {
		at, err := time.Parse(time.RFC3339Nano, receipt.AdmittedAt)
		if err != nil || at.Before(last) || e.Plan.Authorize(at, true) != nil || receipt.Sequence != i+1 || receipt.Outcome != "response_received" || receipt.HTTPStatus != 200 || !agentapi.ValidModel(receipt.ReportedModel) || !agentapi.ValidTokens(receipt.Tokens) || receipt.Cost != "unknown" {
			return errors.New("EVIDENCE_INVALID")
		}
		last = at
	}
	return replayWorld(ctx, e.Bundle, r, terminal)
}
