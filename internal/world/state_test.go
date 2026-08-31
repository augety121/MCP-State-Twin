package world

import (
	"strings"
	"testing"
)

func TestValidateBudgetRejectsMalformedInternalDeterministicState(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*State)
		want   string
	}{
		{name: "bad entropy stream", mutate: func(state *State) { state.Entropy["Bad Stream"] = 1 }, want: "entropy stream"},
		{name: "event id mismatch", mutate: func(state *State) {
			scheduler := state.EnsureScheduler()
			scheduler.NextCreationSequence = 1
			scheduler.Events["other"] = validScheduledEvent()
		}, want: "key/id"},
		{name: "duplicate sequence", mutate: func(state *State) {
			scheduler := state.EnsureScheduler()
			scheduler.NextCreationSequence = 1
			first := validScheduledEvent()
			second := validScheduledEvent()
			second.ID = "second"
			scheduler.Events[first.ID] = first
			scheduler.Events[second.ID] = second
		}, want: "share creation sequence"},
		{name: "delivered without timestamp", mutate: func(state *State) {
			scheduler := state.EnsureScheduler()
			scheduler.NextCreationSequence = 1
			event := validScheduledEvent()
			event.Status = SchedulerDelivered
			scheduler.Events[event.ID] = event
		}, want: "lifecycle"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := New()
			test.mutate(state)
			if err := state.ValidateBudget(); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("validation error = %v, want %q", err, test.want)
			}
		})
	}
}

func validScheduledEvent() ScheduledEvent {
	return ScheduledEvent{
		ID: "first", DueAt: "2026-08-01T01:00:00Z", Priority: 0,
		CreationSequence: 1, Kind: SchedulerKindSignal,
		Payload: map[string]any{}, Status: SchedulerPending,
	}
}
