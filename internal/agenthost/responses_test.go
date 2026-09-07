package agenthost

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/augety121/mcp-state-twin/internal/spec"
	"github.com/augety121/mcp-state-twin/internal/task"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func fixture(t *testing.T) (*task.Task, []*mcp.Tool) {
	t.Helper()
	raw, err := os.ReadFile("../../examples/issue-tracker/agent-tasks/close-issue.json")
	if err != nil {
		t.Fatal(err)
	}
	ta, err := task.Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	raw, err = os.ReadFile("../../examples/issue-tracker/twin.yaml")
	if err != nil {
		t.Fatal(err)
	}
	tw, err := spec.Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	var tools []*mcp.Tool
	for _, x := range tw.Tools {
		tools = append(tools, &mcp.Tool{Name: x.Name, Description: x.Description, InputSchema: x.InputSchema})
	}
	return ta, tools
}

func session(t *testing.T) *Session {
	t.Helper()
	ta, tools := fixture(t)
	s, err := New("mock-baseline", 1024, ta, tools)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func response(items ...any) []byte {
	b, _ := json.Marshal(map[string]any{"status": "completed", "output": items})
	return b
}
func call(id, tool, args string) any {
	return map[string]any{"type": "function_call", "call_id": id, "name": tool, "arguments": args}
}
func final(text string) any {
	return map[string]any{"type": "message", "role": "assistant", "content": []any{map[string]any{"type": "output_text", "text": text}}}
}

func TestProjectionFreshSessionAndPrivateContinuation(t *testing.T) {
	ctx := context.Background()
	s := session(t)
	defer s.Stop()
	raw, err := s.Request(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, private := range []string{"oracle", "faultTool", "expectedOutcome", "bundle", "agent-tasks", "effectCommitted"} {
		if strings.Contains(string(raw), private) {
			t.Fatalf("private field %s", private)
		}
	}
	var request map[string]any
	if err = json.Unmarshal(raw, &request); err != nil {
		t.Fatal(err)
	}
	if request["store"] != false || request["stream"] != false || request["parallel_tool_calls"] != false {
		t.Fatal("unsafe request settings")
	}
	for _, tool := range request["tools"].([]any) {
		if tool.(map[string]any)["strict"] != false {
			t.Fatal("schema silently changed")
		}
	}
	item := map[string]any{"type": "reasoning", "id": "private_reasoning", "encrypted_content": "opaque_test_continuation", "summary": []any{}}
	turn, err := s.Accept(ctx, response(item, call("private_a", "get_issue", `{"owner":"octo","repository":"demo","number":1}`)))
	if err != nil {
		t.Fatal(err)
	}
	if turn.Final || len(turn.Calls) != 1 || turn.Calls[0].Input["number"] != int64(1) {
		t.Fatal("bad turn")
	}
	if err = s.Deliver(ctx, "private_a", map[string]any{}); err == nil {
		t.Fatal("unadmitted delivery")
	}
	if err = s.AdmitTool(ctx, "private_a"); err != nil {
		t.Fatal(err)
	}
	if err = s.AdmitTool(ctx, "private_a"); err == nil {
		t.Fatal("double admission")
	}
	if err = s.Deliver(ctx, "private_a", map[string]any{"state": "open"}); err != nil {
		t.Fatal(err)
	}
	raw, err = s.Request(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "opaque_test_continuation") || !strings.Contains(string(raw), "function_call_output") {
		t.Fatal("continuation dropped")
	}
	public, _ := json.Marshal(turn.Calls)
	if strings.Contains(string(public), "private_a") {
		t.Fatal("private reference persisted")
	}
	other := session(t)
	defer other.Stop()
	fresh, _ := other.Request(ctx)
	if strings.Contains(string(fresh), "opaque_test_continuation") || strings.Contains(string(fresh), "private_a") {
		t.Fatal("cross-trial leak")
	}
	turn, err = s.Accept(ctx, response(final("null")))
	if err != nil || !turn.Final || turn.Answer != nil {
		t.Fatalf("%+v %v", turn, err)
	}
	if _, err = s.Request(ctx); err == nil {
		t.Fatal("request after final")
	}
	if s.Usage().ModelRequests != 2 || s.Usage().ToolAttempts != 1 {
		t.Fatal(s.Usage())
	}
}

func TestResponseAdmissionFailsWholeBatch(t *testing.T) {
	for name, raw := range map[string][]byte{
		"duplicate ids":          response(call("c", "get_issue", `{}`), call("c", "close_issue", `{}`)),
		"invalid json":           response(call("a", "get_issue", `{}`), call("b", "close_issue", `{"x":1,"x":2}`)),
		"non-object":             response(call("c", "get_issue", `null`)),
		"overflow":               response(call("c", "get_issue", `{"x":1e999}`)),
		"unknown type":           response(map[string]any{"type": "shell_call"}),
		"empty":                  response(),
		"incomplete":             []byte(`{"status":"incomplete","output":[]}`),
		"duplicate response key": []byte(`{"status":"failed","status":"completed","output":[]}`),
		"yaml":                   []byte("status: completed\noutput: []"),
		"trailing":               []byte(`{} {}`),
		"secret":                 response(final("api_key=synthetic-secret-sentinel")),
		"async extension":        response(map[string]any{"type": "function_call", "call_id": "c", "name": "get_issue", "arguments": "{}", "async": true}),
	} {
		t.Run(name, func(t *testing.T) {
			s := session(t)
			_, _ = s.Request(context.Background())
			turn, err := s.Accept(context.Background(), raw)
			if err == nil || len(turn.Calls) > 0 {
				t.Fatalf("partial dispatch allowed: %+v %v", turn, err)
			}
			if _, err = s.Request(context.Background()); err == nil {
				t.Fatal("failed session reusable")
			}
		})
	}
}

func TestCallOrderReplayAndBudget(t *testing.T) {
	ctx := context.Background()
	s := session(t)
	_, _ = s.Request(ctx)
	_, err := s.Accept(ctx, response(call("a", "get_issue", `{}`), call("b", "get_issue", `{}`)))
	if err != nil {
		t.Fatal(err)
	}
	if err = s.AdmitTool(ctx, "b"); err == nil {
		t.Fatal("out of order")
	}
	for _, ref := range []string{"a", "b"} {
		if err = s.AdmitTool(ctx, ref); err != nil {
			t.Fatal(err)
		}
		if err = s.Deliver(ctx, ref, map[string]any{"error": "INVALID_INPUT"}); err != nil {
			t.Fatal(err)
		}
	}
	_, _ = s.Request(ctx)
	if _, err = s.Accept(ctx, response(call("a", "get_issue", `{}`))); err == nil {
		t.Fatal("reused call ID")
	}
	ta, tools := fixture(t)
	ta.Budgets.ModelRequests = 1
	s, err = New("mock-test", 1, ta, tools)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = s.Request(ctx)
	_, err = s.Accept(ctx, response(call("a", "get_issue", `{}`)))
	if err != nil {
		t.Fatal(err)
	}
	_ = s.AdmitTool(ctx, "a")
	_ = s.Deliver(ctx, "a", map[string]any{})
	if _, err = s.Request(ctx); err == nil || s.Usage().ModelRequests != 1 {
		t.Fatal("request budget exceeded")
	}
}

func TestProfilesAndJSONLimits(t *testing.T) {
	ta, tools := fixture(t)
	if _, err := New("real-model", 1024, ta, tools); err == nil {
		t.Fatal("unapproved live profile")
	}
	if _, err := New("mock-x", 1024, ta, tools[:1]); err == nil {
		t.Fatal("missing tools")
	}
	var v any
	for _, raw := range []string{strings.Repeat("[", 34) + "0" + strings.Repeat("]", 34), `{"a":{"x":1,"x":2}}`} {
		if decodeJSON([]byte(raw), 1<<20, &v) == nil {
			t.Fatal("invalid document admitted")
		}
	}
	s := session(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.Request(ctx); err == nil {
		t.Fatal("canceled request")
	}
}

func TestCommentaryIsNotFinalAnswer(t *testing.T) {
	ctx := context.Background()
	s := session(t)
	_, _ = s.Request(ctx)
	comment := final("working").(map[string]any)
	comment["phase"] = "commentary"
	if _, err := s.Accept(ctx, response(comment)); err == nil {
		t.Fatal("commentary promoted to final")
	}
	s = session(t)
	_, _ = s.Request(ctx)
	answer := final("null").(map[string]any)
	answer["phase"] = "final_answer"
	turn, err := s.Accept(ctx, response(comment, answer))
	if err != nil || !turn.Final || turn.Answer != nil {
		t.Fatalf("%+v %v", turn, err)
	}
}

func TestGovernorConcurrentStopAndCountBound(t *testing.T) {
	ta, _ := fixture(t)
	ta.Budgets.ToolAttempts = 3
	g, err := NewGovernor(ta.Budgets)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _ = g.Admit(context.Background(), "tool", 0) }()
	}
	wg.Wait()
	if g.Usage().ToolAttempts != 3 {
		t.Fatal(g.Usage())
	}
	g.Stop()
	if err = g.Admit(context.Background(), "content", 0); err == nil {
		t.Fatal("late admission")
	}
	g, _ = NewGovernor(ta.Budgets)
	g.deadline = time.Now().Add(-time.Second)
	if err = g.Admit(context.Background(), "model", 0); err == nil {
		t.Fatal("expired deadline")
	}
	g, _ = NewGovernor(ta.Budgets)
	if err = g.Admit(context.Background(), "content", ta.Budgets.TraceBytes-(64<<10)+1); err == nil {
		t.Fatal("control reserve consumed")
	}
}
