package engine

import (
	"context"
	"testing"

	"github.com/augety121/mcp-state-twin/internal/spec"
	"github.com/augety121/mcp-state-twin/internal/store"
	"github.com/augety121/mcp-state-twin/internal/world"
)

func entropyTwin() *spec.TwinSpec {
	twin := testSpec()
	twin.Entropy = &spec.EntropySpec{
		Algorithm: "sha256-ctr-v1",
		Seed:      "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	}
	twin.Tools[0].Effects = []spec.Effect{{Op: "entropy", Stream: "token", Bytes: 16, As: "token"}}
	twin.Tools[0].Query = nil
	twin.Tools[0].Result = "{'token': vars.token}"
	twin.Tools[0].InputSchema = map[string]any{"type": "object", "additionalProperties": false}
	twin.Invariants = nil
	return twin
}

func TestEntropyDrawIsDeterministicBranchLocalAndPersistent(t *testing.T) {
	ctx := context.Background()
	stateStore, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stateStore.Close() })
	twin := entropyTwin()
	runtime, err := New(twin, stateStore)
	if err != nil {
		t.Fatal(err)
	}
	for _, branch := range []string{"run-a", "run-b"} {
		if err := runtime.Initialize(ctx, branch, world.New()); err != nil {
			t.Fatal(err)
		}
	}
	firstA, err := runtime.Call(ctx, "run-a", "create_item", map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	firstB, err := runtime.Call(ctx, "run-b", "create_item", map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	tokenA := firstA.Result.(map[string]any)["token"].(string)
	tokenB := firstB.Result.(map[string]any)["token"].(string)
	if tokenA != tokenB || len(tokenA) != 32 {
		t.Fatalf("first deterministic draws = %q and %q", tokenA, tokenB)
	}
	if tokenA != "7c276bb3f61ddd5f0713f151e1918618" {
		t.Fatalf("sha256-ctr-v1 golden vector changed: %q", tokenA)
	}
	secondA, err := runtime.Call(ctx, "run-a", "create_item", map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if secondA.Result.(map[string]any)["token"] == tokenA {
		t.Fatal("counter did not advance the entropy stream")
	}
	branch, err := stateStore.Branch(ctx, "run-a")
	if err != nil {
		t.Fatal(err)
	}
	if branch.State.Entropy["token"] != 2 {
		t.Fatalf("entropy counter = %d, want 2", branch.State.Entropy["token"])
	}
}

func TestEntropyCounterRollsBackWithFailedTransition(t *testing.T) {
	ctx := context.Background()
	twin := entropyTwin()
	twin.Tools[0].Effects = append(twin.Tools[0].Effects,
		spec.Effect{Op: "insert", Entity: "item", Key: "'existing'", Value: "{'id': 'existing'}"})
	stateStore, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stateStore.Close() })
	runtime, err := New(twin, stateStore)
	if err != nil {
		t.Fatal(err)
	}
	state := world.New()
	state.Entities["item"] = map[string]map[string]any{"existing": {"id": "existing"}}
	if err := runtime.Initialize(ctx, "main", state); err != nil {
		t.Fatal(err)
	}
	result, err := runtime.Call(ctx, "main", "create_item", map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if result.ErrorClass != "CONFLICT" {
		t.Fatalf("failed entropy result = %#v", result)
	}
	branch, err := stateStore.Branch(ctx, "main")
	if err != nil {
		t.Fatal(err)
	}
	if len(branch.State.Entropy) != 0 {
		t.Fatalf("failed transition committed entropy counters: %#v", branch.State.Entropy)
	}
}

func TestEntropyCounterExhaustionFailsWithoutWraparound(t *testing.T) {
	ctx := context.Background()
	stateStore, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stateStore.Close() })
	twin := entropyTwin()
	runtime, err := New(twin, stateStore)
	if err != nil {
		t.Fatal(err)
	}
	state := world.New()
	state.Entropy["token"] = ^uint64(0)
	if err := runtime.Initialize(ctx, "main", state); err != nil {
		t.Fatal(err)
	}
	result, err := runtime.Call(ctx, "main", "create_item", map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if result.ErrorClass != "INTERNAL_TWIN_ERROR" {
		t.Fatalf("counter exhaustion result = %#v", result)
	}
	branch, err := stateStore.Branch(ctx, "main")
	if err != nil {
		t.Fatal(err)
	}
	if branch.State.Entropy["token"] != ^uint64(0) {
		t.Fatal("exhausted entropy counter wrapped")
	}
}
