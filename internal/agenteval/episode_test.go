package agenteval

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/augety121/mcp-state-twin/internal/agenthost"
)

func mockConfig(id string) *RunConfig {
	return &RunConfig{Format: ConfigFormat, TrialID: id, Profile: agenthost.Profile, Model: "mock-baseline", MaxOutputTokens: 1024, SyntheticOnly: true}
}
func mockResponse(items ...any) json.RawMessage {
	b, _ := json.Marshal(map[string]any{"status": "completed", "output": items})
	return b
}
func mockCall(id string, c Call) any {
	b, _ := json.Marshal(c.Input)
	return map[string]any{"type": "function_call", "call_id": id, "name": c.Tool, "arguments": string(b)}
}
func mockFinal(answer any) any {
	b, _ := json.Marshal(answer)
	return map[string]any{"type": "message", "role": "assistant", "content": []any{map[string]any{"type": "output_text", "text": string(b)}}}
}
func mockWitness(w *Witness) *agenthost.MockScript {
	m := &agenthost.MockScript{Kind: "MockResponses", SyntheticOnly: true}
	for i, c := range w.Calls {
		m.Responses = append(m.Responses, mockResponse(mockCall(string(rune('a'+i)), c)))
	}
	m.Responses = append(m.Responses, mockResponse(mockFinal(w.Answer)))
	return m
}

func TestMockLoopSixTasks(t *testing.T) {
	b, load := kit(t)
	for _, id := range []string{"read-issue", "create-issue", "close-issue", "already-closed", "scope-protection", "after-commit-confirm"} {
		t.Run(id, func(t *testing.T) {
			ta, w := load(id)
			r, err := RunMock(context.Background(), ta, b, mockConfig(id), mockWitness(w))
			if err != nil {
				t.Fatal(err)
			}
			if r.ExecutionStatus != "completed" || r.CleanupStatus != "complete" || r.Source != "mock-responses" || r.Evaluation.Outcome != ta.ExpectedOutcome {
				t.Fatalf("%+v %+v", r, r.Evaluation)
			}
			if r.EvidenceStatus != "partial" || r.WorldReplayable {
				t.Fatal("unsealed report claims replay")
			}
			if r.Usage.ModelRequests != len(w.Calls)+1 || r.Usage.ToolAttempts != len(w.Calls) {
				t.Fatal(r.Usage)
			}
			for _, e := range r.View.Events {
				if !e.Delivered {
					t.Fatal("missing request delivery")
				}
			}
		})
	}
}

func TestMockProtocolStopPreservesCommittedPrefix(t *testing.T) {
	b, load := kit(t)
	ta, w := load("close-issue")
	m := mockWitness(w)
	m.Responses[1] = mockResponse(mockCall("a", w.Calls[0]))
	r, err := RunMock(context.Background(), ta, b, mockConfig("duplicate"), m)
	if err != nil {
		t.Fatal(err)
	}
	if r.ExecutionStatus != "host_error" || r.FailureCode != "HOST_PROTOCOL_ERROR" || len(r.View.Events) != 1 || !r.View.Events[0].EffectCommitted {
		t.Fatalf("lost committed prefix: %+v", r)
	}
	if r.CleanupStatus != "complete" || r.EvidenceStatus != "partial" {
		t.Fatal("incorrect evidence status")
	}
}

func TestMockBatchAtomicAdmissionAndPerToolBudget(t *testing.T) {
	b, load := kit(t)
	ta, w := load("close-issue")
	m := &agenthost.MockScript{Kind: "MockResponses", SyntheticOnly: true, Responses: []json.RawMessage{mockResponse(mockCall("a", w.Calls[0]), mockCall("a", w.Calls[0]))}}
	r, err := RunMock(context.Background(), ta, b, mockConfig("batch"), m)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.View.Events) != 0 || r.FailureCode != "HOST_PROTOCOL_ERROR" {
		t.Fatal("invalid batch dispatched")
	}
	ta.Budgets.ModelRequests = 1
	m = mockWitness(w)
	r, err = RunMock(context.Background(), ta, b, mockConfig("budget"), m)
	if err != nil {
		t.Fatal(err)
	}
	if r.ExecutionStatus != "budget_exhausted" || len(r.View.Events) != 1 || r.View.Events[0].Delivered || !r.View.Events[0].EffectCommitted {
		t.Fatalf("cutoff wrong: %+v", r)
	}
}

func TestMockUnknownToolAttemptAndRefusal(t *testing.T) {
	b, load := kit(t)
	ta, w := load("close-issue")
	w.Calls = append(w.Calls, Call{Tool: "reset_world", Input: map[string]any{}})
	r, err := RunMock(context.Background(), ta, b, mockConfig("unknown"), mockWitness(w))
	if err != nil {
		t.Fatal(err)
	}
	if r.Evaluation.Outcome != "policy_violation" || r.Evaluation.BlockedAttempts != 1 || r.View.Events[1].Dispatched {
		t.Fatalf("%+v", r.Evaluation)
	}
	if r.Usage.ToolAttempts != 2 {
		t.Fatal("refused attempt not counted")
	}
}

func TestMockSensitiveResponseIsNotRetained(t *testing.T) {
	b, load := kit(t)
	ta, w := load("close-issue")
	m := mockWitness(w)
	m.Responses[1] = mockResponse(mockFinal("api_key=synthetic-secret-sentinel"))
	r, err := RunMock(context.Background(), ta, b, mockConfig("privacy"), m)
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(r)
	if r.FailureCode != "DATA_POLICY_REJECTED" || strings.Contains(string(encoded), "synthetic-secret-sentinel") {
		t.Fatal("unsafe failure report")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err = RunMock(ctx, ta, b, mockConfig("canceled"), m); err == nil {
		t.Fatal("canceled preflight admitted")
	}
}

func TestRunConfigClosedSurface(t *testing.T) {
	c := mockConfig("x")
	b, _ := json.Marshal(c)
	if _, err := DecodeRun(b); err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{strings.TrimSuffix(string(b), "}") + `,"endpoint":"https://example.invalid"}`, strings.Replace(string(b), "mock-baseline", "arbitrary-live-model", 1)} {
		if _, err := DecodeRun([]byte(raw)); err == nil {
			t.Fatal("unsafe profile admitted")
		}
	}
}
