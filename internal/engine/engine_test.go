package engine

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/augety121/mcp-state-twin/internal/spec"
	"github.com/augety121/mcp-state-twin/internal/store"
	"github.com/augety121/mcp-state-twin/internal/world"
)

func TestScheduledActionExecutesAtomicallyThroughBoundRuntime(t *testing.T) {
	ctx := context.Background()
	twin := testSpec()
	stateStore, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stateStore.Close() })
	runtime, err := New(twin, stateStore)
	if err != nil {
		t.Fatal(err)
	}
	if err := runtime.Initialize(ctx, "main", world.New()); err != nil {
		t.Fatal(err)
	}
	if err := runtime.ValidateScheduledAction("create_item", map[string]any{"name": "scheduled"}); err != nil {
		t.Fatal(err)
	}
	dueAt := "2026-08-01T01:00:00Z"
	if _, err := stateStore.ScheduleEvent(ctx, store.ScheduleRequest{
		ID: "create-later", BranchID: "main", DueAt: dueAt, Kind: store.SchedulerKindAction,
		Action: &world.ScheduledAction{Tool: "create_item", Input: map[string]any{"name": "scheduled"}, SpecDigest: runtime.Digest()},
	}, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := stateStore.AdvanceClock(ctx, "main", time.Date(2026, 8, 1, 2, 0, 0, 0, time.UTC), nil); !errors.Is(err, store.ErrSchedulerAction) {
		t.Fatalf("ordinary advance action error = %v", err)
	}
	result, err := runtime.AdvanceClockToNext(ctx, "main", int64Pointer(1))
	if err != nil {
		t.Fatal(err)
	}
	if result.ExecutedActions != 1 || result.DeliveredSignals != 0 || len(result.Processed) != 1 || result.Processed[0].Status != store.SchedulerCompleted {
		t.Fatalf("scheduler result = %#v", result)
	}
	branch, err := stateStore.Branch(ctx, "main")
	if err != nil {
		t.Fatal(err)
	}
	if branch.CallCount != 1 || branch.HeadVersion != 2 || branch.Clock.Format(time.RFC3339Nano) != dueAt {
		t.Fatalf("branch = %#v", branch)
	}
	if got := branch.State.Entities["item"]["1"]["name"]; got != "scheduled" {
		t.Fatalf("scheduled entity name = %v", got)
	}
	event := branch.State.Scheduler.Events["create-later"]
	if event.Outcome == nil || event.Outcome.CallIndex != 1 || !event.Outcome.EffectCommitted || event.AttemptCount != 1 {
		t.Fatalf("scheduled action evidence = %#v", event)
	}
}

func TestScheduledActionFailureAndAfterCommitFaultAreExplicit(t *testing.T) {
	ctx := context.Background()
	twin := testSpec()
	stateStore, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stateStore.Close() })
	runtime, err := New(twin, stateStore)
	if err != nil {
		t.Fatal(err)
	}
	if err := runtime.Initialize(ctx, "main", world.New()); err != nil {
		t.Fatal(err)
	}
	if _, err := stateStore.InstallFault(ctx, store.FaultPlan{
		ID: "lost-response", BranchID: "main", ToolName: "create_item",
		Phase: store.FaultPhaseAfterCommitBeforeResponse, ErrorClass: "TIMEOUT_AFTER_EFFECT",
		Message: "synthetic response loss", RemainingCount: 1,
	}, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := stateStore.ScheduleEvent(ctx, store.ScheduleRequest{
		ID: "faulted-create", BranchID: "main", DueAt: "2026-08-01T01:00:00Z", Kind: store.SchedulerKindAction,
		Action: &world.ScheduledAction{Tool: "create_item", Input: map[string]any{"name": "committed"}, SpecDigest: runtime.Digest()},
	}, nil); err != nil {
		t.Fatal(err)
	}
	result, err := runtime.AdvanceClockToNext(ctx, "main", int64Pointer(2))
	if err != nil {
		t.Fatal(err)
	}
	event := result.Processed[0]
	if event.Status != store.SchedulerFailed || event.Outcome == nil || event.Outcome.ErrorClass != "TIMEOUT_AFTER_EFFECT" ||
		!event.Outcome.EffectCommitted || event.Outcome.FaultID != "lost-response" {
		t.Fatalf("faulted action evidence = %#v", event)
	}
	branch, err := stateStore.Branch(ctx, "main")
	if err != nil {
		t.Fatal(err)
	}
	if branch.State.Entities["item"]["1"]["name"] != "committed" {
		t.Fatal("after-commit fault discarded modeled effect")
	}
	events, err := stateStore.FaultEvents(ctx, "main")
	if err != nil || len(events) != 1 || events[0].CallIndex != 1 {
		t.Fatalf("fault events = %#v err=%v", events, err)
	}
}

func int64Pointer(value int64) *int64 { return &value }

func boolPtr(v bool) *bool { return &v }

func testSpec() *spec.TwinSpec {
	return &spec.TwinSpec{
		APIVersion: spec.APIVersion,
		Kind:       spec.Kind,
		Metadata: spec.Metadata{
			Name:     "items",
			Upstream: spec.UpstreamMetadata{Protocol: "mcp", Status: "unbound"},
			Fidelity: spec.FidelityMetadata{Level: "L1", Status: "unverified"},
		},
		Clock:      spec.ClockSpec{Mode: "virtual", Initial: "2026-08-01T00:00:00Z"},
		State:      spec.StateSpec{Entities: map[string]spec.EntitySpec{"item": {Key: []string{"id"}}}},
		Invariants: []spec.InvariantSpec{{ID: "non-empty", Assert: "state.entities.item.all(k, state.entities.item[k].id != '')"}},
		Tools: []spec.ToolSpec{
			{
				Name: "create_item", Description: "Create an item.", Modeled: boolPtr(true),
				InputSchema: map[string]any{
					"type": "object", "required": []any{"name"}, "additionalProperties": false,
					"properties": map[string]any{"name": map[string]any{"type": "string"}},
				},
				Effects: []spec.Effect{
					{Op: "allocate", Sequence: "item", As: "id"},
					{Op: "insert", Entity: "item", Key: "string(vars.id)", Value: "{'id': string(vars.id), 'name': input.name}"},
				},
				Query:  &spec.Query{Entity: "item", Key: "string(vars.id)", As: "created"},
				Result: "{'item': vars.created}",
			},
		},
	}
}

func TestSurfaceBindingFailsClosedOnDrift(t *testing.T) {
	twin := testSpec()
	digest, err := twin.SurfaceDigest()
	if err != nil {
		t.Fatal(err)
	}
	twin.Metadata.Upstream = spec.UpstreamMetadata{Protocol: "mcp", Status: "current", SurfaceDigest: digest}
	if _, err := New(twin, nil); err != nil {
		t.Fatalf("matching current surface was rejected: %v", err)
	}

	twin.Tools[0].Description = "Upstream changed this description."
	if _, err := New(twin, nil); err == nil || !strings.Contains(err.Error(), "SPEC_DRIFT") {
		t.Fatalf("expected SPEC_DRIFT after descriptor change, got %v", err)
	}
}

func TestUnknownAndDriftedSurfaceCannotStart(t *testing.T) {
	for _, status := range []string{"unknown", "drifted"} {
		twin := testSpec()
		digest, err := twin.SurfaceDigest()
		if err != nil {
			t.Fatal(err)
		}
		twin.Metadata.Upstream = spec.UpstreamMetadata{Protocol: "mcp", Status: status, SurfaceDigest: digest}
		if _, err := New(twin, nil); err == nil || !strings.Contains(err.Error(), "SPEC_DRIFT") {
			t.Fatalf("status %s should fail closed, got %v", status, err)
		}
	}
}

func TestNewRejectsNilTwinSpec(t *testing.T) {
	if _, err := New(nil, nil); err == nil || !strings.Contains(err.Error(), "TwinSpec is required") {
		t.Fatalf("expected nil TwinSpec rejection, got %v", err)
	}
}

func TestToolResultSizeLimitFailsClosed(t *testing.T) {
	twin := testSpec()
	twin.Tools[0].Result = "{'payload': input.payload}"
	twin.Tools[0].Effects = nil
	twin.Tools[0].Query = nil
	twin.Invariants = nil
	twin.Tools[0].InputSchema = map[string]any{
		"type": "object", "required": []any{"payload"}, "additionalProperties": false,
		"properties": map[string]any{"payload": map[string]any{"type": "string"}},
	}
	runtime, err := New(twin, nil)
	if err != nil {
		t.Fatal(err)
	}
	result := runtime.apply(twin.Tools[0], world.New(), time.Unix(0, 0).UTC(), 1, map[string]any{"payload": strings.Repeat("x", MaxToolResultBytes)})
	if result.ErrorClass != "RESOURCE_LIMIT" || !strings.Contains(result.Result.(map[string]any)["error"].(map[string]any)["message"].(string), "exceeds") {
		t.Fatalf("expected bounded result failure, got %#v", result)
	}
}

func TestDeterministicCreateAndInvalidInput(t *testing.T) {
	ctx := context.Background()
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	twin := testSpec()
	if err := twin.Validate(); err != nil {
		t.Fatal(err)
	}
	runtime, err := New(twin, s)
	if err != nil {
		t.Fatal(err)
	}
	if err := runtime.Initialize(ctx, "main", world.New()); err != nil {
		t.Fatal(err)
	}
	created, err := runtime.Call(ctx, "main", "create_item", map[string]any{"name": "first"})
	if err != nil {
		t.Fatal(err)
	}
	if created.ErrorClass != "" {
		t.Fatalf("unexpected domain error: %#v", created.Result)
	}
	failed, err := runtime.Call(ctx, "main", "create_item", map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if failed.ErrorClass != "INVALID_INPUT" {
		t.Fatalf("error class = %q", failed.ErrorClass)
	}
	if failed.BeforeDigest != failed.AfterDigest {
		t.Fatal("invalid input changed state")
	}
}

func TestThousandCallCorpusReplaysToSameDigest(t *testing.T) {
	ctx := context.Background()
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	runtime, err := New(testSpec(), s)
	if err != nil {
		t.Fatal(err)
	}
	if err := runtime.Initialize(ctx, "main", world.New()); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateSnapshot(ctx, "base", "main"); err != nil {
		t.Fatal(err)
	}
	if err := s.Fork(ctx, "base", "replay-a"); err != nil {
		t.Fatal(err)
	}
	if err := s.Fork(ctx, "base", "replay-b"); err != nil {
		t.Fatal(err)
	}
	for i := range 1000 {
		input := map[string]any{"name": fmt.Sprintf("item-%04d", i)}
		a, err := runtime.Call(ctx, "replay-a", "create_item", input)
		if err != nil {
			t.Fatal(err)
		}
		b, err := runtime.Call(ctx, "replay-b", "create_item", input)
		if err != nil {
			t.Fatal(err)
		}
		if a.AfterDigest != b.AfterDigest {
			t.Fatalf("digest diverged at call %d: %s != %s", i+1, a.AfterDigest, b.AfterDigest)
		}
	}
	a, err := s.Branch(ctx, "replay-a")
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.Branch(ctx, "replay-b")
	if err != nil {
		t.Fatal(err)
	}
	if a.StateDigest != b.StateDigest {
		t.Fatalf("final state digest differs: %s != %s", a.StateDigest, b.StateDigest)
	}
}

func TestExplicitlyUnmodeledToolFailsWithoutStateChange(t *testing.T) {
	ctx := context.Background()
	twin := testSpec()
	twin.Tools[0].Modeled = boolPtr(false)
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	runtime, err := New(twin, s)
	if err != nil {
		t.Fatal(err)
	}
	if err := runtime.Initialize(ctx, "main", world.New()); err != nil {
		t.Fatal(err)
	}
	result, err := runtime.Call(ctx, "main", "create_item", map[string]any{"name": "first"})
	if err != nil {
		t.Fatal(err)
	}
	if result.ErrorClass != "UNMODELED_BEHAVIOR" {
		t.Fatalf("error class = %q, want UNMODELED_BEHAVIOR", result.ErrorClass)
	}
	if result.BeforeDigest != result.AfterDigest {
		t.Fatal("unmodeled behavior changed state")
	}
}
