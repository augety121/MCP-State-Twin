package agenteval

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/augety121/mcp-state-twin/internal/bundle"
)

func rawBundle(t *testing.T) []byte {
	t.Helper()
	out := filepath.Join(t.TempDir(), "world.stb")
	if _, err := bundle.Build("../../examples/issue-tracker/bundle-agent.yaml", out); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func record(t *testing.T) (string, *AgentEvidence) {
	t.Helper()
	_, load := kit(t)
	ta, w := load("after-commit-confirm")
	root := t.TempDir()
	r, err := RecordMock(context.Background(), root, "trial", ta, rawBundle(t), mockConfig("trial"), mockWitness(w))
	if err != nil {
		t.Fatal(err)
	}
	if !r.WorldReplayable || r.EvidenceStatus != "complete" || r.CleanupStatus != "complete" {
		t.Fatalf("%+v", r)
	}
	if _, err = os.Stat(filepath.Join(root, "trial", "closure.json")); !os.IsNotExist(err) {
		t.Fatal("staging not cleaned")
	}
	data, err := os.ReadFile(filepath.Join(root, "trial", "terminal.json"))
	if err != nil {
		t.Fatal(err)
	}
	e, err := DecodeEvidence(data)
	if err != nil {
		t.Fatal(err)
	}
	return root, e
}

func TestRecordVerifyReplayAndNoClobber(t *testing.T) {
	root, e := record(t)
	if err := VerifyEvidence(context.Background(), e); err != nil {
		t.Fatal(err)
	}
	_, load := kit(t)
	ta, w := load("after-commit-confirm")
	before, _ := os.ReadFile(filepath.Join(root, "trial", "terminal.json"))
	if _, err := RecordMock(context.Background(), root, "trial", ta, rawBundle(t), mockConfig("trial"), mockWitness(w)); err == nil {
		t.Fatal("overwrite allowed")
	}
	after, _ := os.ReadFile(filepath.Join(root, "trial", "terminal.json"))
	if string(before) != string(after) {
		t.Fatal("evidence overwritten")
	}
}

func TestEvidenceTamperingAndClosedFields(t *testing.T) {
	_, original := record(t)
	raw, _ := json.Marshal(original)
	for name, change := range map[string]func(*AgentEvidence){
		"missing event":     func(e *AgentEvidence) { e.Episode.View.Events = e.Episode.View.Events[1:] },
		"wrong result":      func(e *AgentEvidence) { e.Episode.View.Events[0].Result = map[string]any{"success": true} },
		"wrong delivery":    func(e *AgentEvidence) { e.Episode.View.Events[0].Delivered = false },
		"wrong state":       func(e *AgentEvidence) { e.Episode.View.After.Sequences["comment_id"]++ },
		"wrong counter":     func(e *AgentEvidence) { e.Episode.Usage.ToolAttempts++ },
		"wrong grade":       func(e *AgentEvidence) { e.Episode.Evaluation.Outcome = "task_failed" },
		"wrong frontier":    func(e *AgentEvidence) { e.Episode.RequestFrontiers[0] = 1 },
		"different runtime": func(e *AgentEvidence) { e.Episode.Definition.RuntimeVersion = "different" },
	} {
		t.Run(name, func(t *testing.T) {
			e, err := DecodeEvidence(raw)
			if err != nil {
				t.Fatal(err)
			}
			change(e)
			if VerifyEvidence(context.Background(), e) == nil {
				t.Fatal("tampering accepted")
			}
		})
	}
	for _, bad := range []string{`{"format":"x","episode":null}`, strings.TrimSuffix(string(raw), "}") + `,"unexpected":true}`, strings.Replace(string(raw), `"source":"mock-responses"`, `"source":"live","source":"mock-responses"`, 1)} {
		if _, err := DecodeEvidence([]byte(bad)); err == nil {
			t.Fatal("malformed evidence admitted")
		}
	}
}

func TestPartialEvidenceDoesNotClaimCompletion(t *testing.T) {
	_, load := kit(t)
	ta, w := load("close-issue")
	ta.Budgets.ModelRequests = 1
	root := t.TempDir()
	r, err := RecordMock(context.Background(), root, "partial", ta, rawBundle(t), mockConfig("partial"), mockWitness(w))
	if err != nil {
		t.Fatal(err)
	}
	if r.WorldReplayable || r.EvidenceStatus != "partial" || r.ExecutionStatus != "budget_exhausted" {
		t.Fatal("partial promoted")
	}
	data, _ := os.ReadFile(filepath.Join(root, "partial", "terminal.json"))
	e, err := DecodeEvidence(data)
	if err != nil {
		t.Fatal(err)
	}
	if VerifyEvidence(context.Background(), e) == nil {
		t.Fatal("partial verified complete")
	}
}
