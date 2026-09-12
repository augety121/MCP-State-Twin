package agenteval

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"

	"github.com/augety121/mcp-state-twin/internal/bundle"
	"github.com/augety121/mcp-state-twin/internal/world"
)

func TestTerminalFailuresPreserveExecutionCause(t *testing.T) {
	_, load := kit(t)
	ta, _ := load("close-issue")
	for _, cause := range []string{"PROVIDER_ACCEPTANCE_UNKNOWN", "CANCELED", ""} {
		r := &AgentEpisode{ExecutionStatus: "host_error", FailureCode: cause, EvidenceStatus: "partial", CleanupStatus: "pending"}
		closes := 0
		err := finishEpisode(ta, r, func(ctx context.Context) (*world.State, error) {
			if ctx.Err() != nil {
				t.Fatal("cleanup context canceled")
			}
			return nil, errors.New("private state failure")
		}, func() error { closes++; return errors.New("private close failure") }, nil)
		want := cause
		if want == "" {
			want = "TERMINAL_INSPECTION_FAILED"
		}
		if err != nil || closes != 1 || r.FailureCode != want || r.TerminalFailureCode != "TERMINAL_INSPECTION_FAILED" || r.CleanupFailureCode != "CLEANUP_FAILED" || r.CleanupStatus != "failed" || r.EvidenceStatus != "partial" {
			t.Fatalf("%+v %v", r, err)
		}
	}
}

func TestTerminalStageFailureStillClosesWorld(t *testing.T) {
	_, load := kit(t)
	ta, _ := load("close-issue")
	closes := 0
	err := finishEpisode(ta, &AgentEpisode{EvidenceStatus: "partial"}, func(context.Context) (*world.State, error) { return nil, errors.New("state unavailable") }, func() error { closes++; return nil }, func(context.Context, *AgentEpisode) error { return errors.New("EVIDENCE_WRITE_FAILED") })
	if err == nil || err.Error() != "EVIDENCE_WRITE_FAILED" || closes != 1 {
		t.Fatal("staging failure leaked world", err, closes)
	}
}

func TestLateCancellationDoesNotEraseCompletedReplayClosure(t *testing.T) {
	_, load := kit(t)
	ta, w := load("close-issue")
	raw := rawBundle(t)
	c := mockConfig("trial")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	e := &AgentEvidence{Format: EvidenceFormat, Bundle: base64.StdEncoding.EncodeToString(raw)}
	verified := false
	r, err := recordEpisode(ctx, t.TempDir(), "trial", ta, raw, c, func(b *bundle.Artifact, stage stageEpisode) (*AgentEpisode, error) {
		return runMock(ctx, ta, b, c, mockWitness(w), func(finish context.Context, r *AgentEpisode) error {
			cancel() // execution has completed; only operator-side sealing remains
			return stage(finish, r)
		})
	}, func(r *AgentEpisode) any { e.Episode = r; return e }, func(finish context.Context, r *AgentEpisode) error {
		verified = true
		e.Episode = r
		if finish.Err() != nil {
			t.Fatal("execution cancellation leaked into closure context")
		}
		return replay(finish, e, false)
	})
	if err != nil || !verified || r.EvidenceStatus != "complete" || r.CleanupStatus != "complete" {
		t.Fatalf("%+v %v", r, err)
	}
}

func TestSecondaryTerminalFailureCannotVerifyComplete(t *testing.T) {
	_, e := record(t)
	e.Episode.TerminalFailureCode = "TERMINAL_INSPECTION_FAILED"
	if VerifyEvidence(context.Background(), e) == nil {
		t.Fatal("secondary failure ignored")
	}
}
