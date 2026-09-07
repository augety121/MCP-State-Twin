package agenteval

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/augety121/mcp-state-twin/internal/agentapi"
	"github.com/augety121/mcp-state-twin/internal/bundle"
)

func livePlan(t *testing.T, id string) (*agentapi.Plan, []byte, *Witness) {
	t.Helper()
	_, load := kit(t)
	ta, w := load(id)
	raw := rawBundle(t)
	b, err := bundle.OpenBytes(raw)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	return &agentapi.Plan{Format: agentapi.PlanFormat, ID: id, Provider: "openai", Profile: LiveProfile, Model: "test-model-not-a-product", Task: ta, BundleDigest: b.Digest, IssuedAt: now.Add(-time.Minute).Format(time.RFC3339Nano), ExpiresAt: now.Add(time.Hour).Format(time.RFC3339Nano), MaxRequests: ta.Budgets.ModelRequests, MaxOutputTokens: 1024, CostPolicy: agentapi.CostPolicy, Approved: true, SyntheticDataApproved: true, UnknownCostApproved: true}, raw, w
}
func liveRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".statetwin", "live"), 0700); err != nil {
		t.Fatal(err)
	}
	return root
}
func fakeClient(m [][]byte, fail int, posts *int, closed *bool) func(*agentapi.Plan) (liveExchange, func(), error) {
	return func(p *agentapi.Plan) (liveExchange, func(), error) {
		return func(_ context.Context, request []byte) ([]byte, agentapi.Receipt, error) {
			*posts++
			r := agentapi.Receipt{Sequence: *posts, AdmittedAt: time.Now().UTC().Format(time.RFC3339Nano), Outcome: "response_received", HTTPStatus: 200, ReportedModel: p.Model, Cost: "unknown"}
			if *posts == fail {
				r.Outcome = "acceptance_unknown"
				r.HTTPStatus = 0
				return nil, r, errors.New("PROVIDER_ACCEPTANCE_UNKNOWN")
			}
			if *posts > len(m) {
				return nil, r, errors.New("HOST_PROTOCOL_ERROR")
			}
			return m[*posts-1], r, nil
		}, func() { *closed = true }, nil
	}
}
func liveScript(w *Witness) [][]byte {
	var raw [][]byte
	for _, item := range mockWitness(w).Responses {
		raw = append(raw, item)
	}
	return raw
}
func readLive(t *testing.T, root string, p *agentapi.Plan) *LiveEvidence {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(p.OutputDirectory()), "terminal.json"))
	if err != nil {
		t.Fatal(err)
	}
	e, err := DecodeLiveEvidence(data)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func TestLiveContractSixTasksReplayAndSingleUse(t *testing.T) {
	for _, id := range []string{"read-issue", "create-issue", "close-issue", "already-closed", "scope-protection", "after-commit-confirm"} {
		t.Run(id, func(t *testing.T) {
			p, raw, w := livePlan(t, id)
			root := liveRoot(t)
			posts := 0
			closed := false
			factory := fakeClient(liveScript(w), 0, &posts, &closed)
			r, err := recordLive(context.Background(), root, p, raw, true, "contract-test", factory)
			if err != nil {
				t.Fatal(err)
			}
			if !closed || r.Source != "contract-test" || r.ExecutionStatus != "completed" || r.EvidenceStatus != "complete" || r.Evaluation.Outcome != p.Task.ExpectedOutcome || posts != len(w.Calls)+1 {
				t.Fatalf("%+v posts=%d", r, posts)
			}
			e := readLive(t, root, p)
			if err = VerifyLiveEvidence(context.Background(), e); err != nil {
				t.Fatal(err)
			}
			encoded, _ := json.Marshal(e)
			if _, err = DecodeEvidence(encoded); err == nil {
				t.Fatal("live evidence admitted as mock")
			}
			if _, err = recordLive(context.Background(), root, p, raw, true, "contract-test", factory); err == nil || posts != len(w.Calls)+1 {
				t.Fatal("plan reused")
			}
		})
	}
}

func TestLiveUnknownAcceptanceAndBudgetPreserveEffects(t *testing.T) {
	for _, fail := range []bool{true, false} {
		t.Run(map[bool]string{true: "unknown", false: "budget"}[fail], func(t *testing.T) {
			p, raw, w := livePlan(t, "close-issue")
			root := liveRoot(t)
			posts := 0
			closed := false
			failAt := 2
			if !fail {
				p.MaxRequests = 1
				failAt = 0
			}
			r, err := recordLive(context.Background(), root, p, raw, true, "contract-test", fakeClient(liveScript(w), failAt, &posts, &closed))
			if err != nil {
				t.Fatal(err)
			}
			want := "PROVIDER_ACCEPTANCE_UNKNOWN"
			if !fail {
				want = "BUDGET_EXHAUSTED"
			}
			if !closed || r.FailureCode != want || len(r.View.Events) != 1 || r.View.Events[0].Delivered || !r.View.Events[0].EffectCommitted || r.EvidenceStatus != "partial" || r.WorldReplayable {
				t.Fatalf("%+v", r)
			}
			e := readLive(t, root, p)
			if VerifyLiveEvidence(context.Background(), e) == nil {
				t.Fatal("uncertain evidence sealed")
			}
			if _, err = recordLive(context.Background(), root, p, raw, true, "contract-test", fakeClient(liveScript(w), 0, &posts, &closed)); err == nil {
				t.Fatal("partial run automatically resumed")
			}
		})
	}
}

func TestLiveApprovalAndBindingBeforeClaimOrClient(t *testing.T) {
	for _, kind := range []string{"flag", "approval", "binding", "oracle invalid", "model", "expired"} {
		t.Run(kind, func(t *testing.T) {
			p, raw, _ := livePlan(t, "close-issue")
			root := liveRoot(t)
			allow := true
			switch kind {
			case "flag":
				allow = false
			case "approval":
				p.Approved = false
			case "binding":
				p.BundleDigest = "sha256:" + strings.Repeat("0", 64)
			case "oracle invalid":
				p.Task.Oracle[0].Expr = "unknown_private_variable"
			case "model":
				p.Model = "mock-invalid"
			case "expired":
				p.ExpiresAt = time.Now().Add(-time.Second).Format(time.RFC3339Nano)
			}
			_, err := recordLive(context.Background(), root, p, raw, allow, "contract-test", func(*agentapi.Plan) (liveExchange, func(), error) {
				t.Fatal("client constructed before admission")
				return nil, nil, nil
			})
			if err == nil {
				t.Fatal("unsafe plan accepted")
			}
			if _, err = os.Stat(filepath.Join(root, filepath.FromSlash(p.OutputDirectory()))); !os.IsNotExist(err) {
				t.Fatal("invalid plan claimed output")
			}
		})
	}
}

func TestLiveEvidenceCrossBindingsAndReceiptTampering(t *testing.T) {
	p, raw, w := livePlan(t, "close-issue")
	root := liveRoot(t)
	posts := 0
	closed := false
	if _, err := recordLive(context.Background(), root, p, raw, true, "contract-test", fakeClient(liveScript(w), 0, &posts, &closed)); err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(readLive(t, root, p))
	for name, change := range map[string]func(*LiveEvidence){
		"approval": func(e *LiveEvidence) { e.Plan.Approved = false }, "model": func(e *LiveEvidence) { e.Plan.Model = "other-model" }, "task": func(e *LiveEvidence) { e.Plan.Task.Objective = "different task" }, "source": func(e *LiveEvidence) { e.Episode.Source = "mock-responses" }, "receipt missing": func(e *LiveEvidence) { e.Receipts = e.Receipts[1:] }, "timestamp": func(e *LiveEvidence) { e.Receipts[0].AdmittedAt = e.Plan.ExpiresAt }, "unknown": func(e *LiveEvidence) { e.Receipts[0].Outcome = "acceptance_unknown" }, "tokens": func(e *LiveEvidence) { e.Receipts[0].Tokens = &agentapi.Tokens{Input: -1} }, "state": func(e *LiveEvidence) { e.Episode.View.After.Sequences["comment_id"]++ }, "failure": func(e *LiveEvidence) { e.Episode.FailureCode = "PROVIDER_ACCEPTANCE_UNKNOWN" },
	} {
		t.Run(name, func(t *testing.T) {
			e, err := DecodeLiveEvidence(data)
			if err != nil {
				t.Fatal(err)
			}
			change(e)
			if VerifyLiveEvidence(context.Background(), e) == nil {
				t.Fatal("corrupt evidence accepted")
			}
		})
	}
}
