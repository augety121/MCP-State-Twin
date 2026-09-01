package world

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"time"

	"github.com/augety121/mcp-state-twin/internal/limits"
)

const (
	SchedulerKindSignal = "signal"
	SchedulerKindAction = "tool-call"
	SchedulerPending    = "pending"
	SchedulerDelivered  = "delivered"
	SchedulerCanceled   = "canceled"
	SchedulerCompleted  = "completed"
	SchedulerFailed     = "failed"
)

var (
	entropyStreamPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$`)
	controlIDPattern     = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)
	toolNamePattern      = regexp.MustCompile(`^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$`)
	digestPattern        = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
)

type State struct {
	Entities  map[string]map[string]map[string]any `json:"entities" yaml:"entities"`
	Sequences map[string]int64                     `json:"sequences" yaml:"sequences"`
	Entropy   map[string]uint64                    `json:"entropy,omitempty" yaml:"entropy,omitempty"`
	Scheduler *SchedulerState                      `json:"scheduler,omitempty" yaml:"scheduler,omitempty"`
}

// SchedulerState is hidden modeled-world infrastructure. It is persisted in
// the canonical branch state so snapshot, fork, reset, and digest equality all
// bind the same timer queue, but it is deliberately excluded from CELValue.
type SchedulerState struct {
	NextCreationSequence int64                     `json:"nextCreationSequence" yaml:"nextCreationSequence"`
	Events               map[string]ScheduledEvent `json:"events" yaml:"events"`
}

type ScheduledEvent struct {
	ID               string                  `json:"id" yaml:"id"`
	DueAt            string                  `json:"dueAt" yaml:"dueAt"`
	Priority         int                     `json:"priority" yaml:"priority"`
	CreationSequence int64                   `json:"creationSequence" yaml:"creationSequence"`
	Kind             string                  `json:"kind" yaml:"kind"`
	Payload          any                     `json:"payload" yaml:"payload"`
	Action           *ScheduledAction        `json:"action,omitempty" yaml:"action,omitempty"`
	Outcome          *ScheduledActionOutcome `json:"outcome,omitempty" yaml:"outcome,omitempty"`
	AttemptCount     int                     `json:"attemptCount,omitempty" yaml:"attemptCount,omitempty"`
	Status           string                  `json:"status" yaml:"status"`
	DeliveredAt      string                  `json:"deliveredAt,omitempty" yaml:"deliveredAt,omitempty"`
	FinishedAt       string                  `json:"finishedAt,omitempty" yaml:"finishedAt,omitempty"`
	CanceledAt       string                  `json:"canceledAt,omitempty" yaml:"canceledAt,omitempty"`
}

type ScheduledAction struct {
	Tool       string         `json:"tool" yaml:"tool"`
	Input      map[string]any `json:"input" yaml:"input"`
	SpecDigest string         `json:"specDigest" yaml:"specDigest"`
}

type ScheduledActionOutcome struct {
	CallIndex       int64  `json:"callIndex" yaml:"callIndex"`
	Result          any    `json:"result" yaml:"result"`
	ErrorClass      string `json:"errorClass,omitempty" yaml:"errorClass,omitempty"`
	EffectCommitted bool   `json:"effectCommitted" yaml:"effectCommitted"`
	FaultID         string `json:"faultId,omitempty" yaml:"faultId,omitempty"`
	FaultPhase      string `json:"faultPhase,omitempty" yaml:"faultPhase,omitempty"`
}

// ValidateBudget applies the default resource profile to a complete world
// state. It is called both by the engine and the storage boundary so alternate
// callers cannot bypass state limits.
func (s *State) ValidateBudget() error {
	if s == nil {
		return fmt.Errorf("world state is required")
	}
	s.Normalize()
	records := 0
	for _, entities := range s.Entities {
		records += len(entities)
		if records > limits.MaxEntitiesPerBranch {
			return fmt.Errorf("entity records %d exceed limit %d", records, limits.MaxEntitiesPerBranch)
		}
	}
	if len(s.Entropy) > limits.MaxEntropyStreams {
		return fmt.Errorf("entropy streams %d exceed limit %d", len(s.Entropy), limits.MaxEntropyStreams)
	}
	if s.Scheduler != nil && len(s.Scheduler.Events) > limits.MaxScheduledEvents {
		return fmt.Errorf("scheduled events %d exceed limit %d", len(s.Scheduler.Events), limits.MaxScheduledEvents)
	}
	if err := s.validateInternalState(); err != nil {
		return err
	}
	if err := limits.ValidateJSON(s, limits.MaxStateBytes); err != nil {
		return fmt.Errorf("world state: %w", err)
	}
	return nil
}

func (s *State) validateInternalState() error {
	streams := make([]string, 0, len(s.Entropy))
	for stream := range s.Entropy {
		streams = append(streams, stream)
	}
	sort.Strings(streams)
	for _, stream := range streams {
		if !entropyStreamPattern.MatchString(stream) {
			return fmt.Errorf("entropy stream %q is invalid", stream)
		}
	}
	if s.Scheduler == nil {
		return nil
	}
	if s.Scheduler.NextCreationSequence < 0 {
		return fmt.Errorf("scheduler creation sequence must not be negative")
	}
	sequences := make(map[int64]string, len(s.Scheduler.Events))
	eventKeys := make([]string, 0, len(s.Scheduler.Events))
	for key := range s.Scheduler.Events {
		eventKeys = append(eventKeys, key)
	}
	sort.Strings(eventKeys)
	for _, key := range eventKeys {
		event := s.Scheduler.Events[key]
		if key != event.ID || !controlIDPattern.MatchString(event.ID) {
			return fmt.Errorf("scheduled event key/id %q/%q is invalid", key, event.ID)
		}
		if event.CreationSequence < 1 || event.CreationSequence > s.Scheduler.NextCreationSequence {
			return fmt.Errorf("scheduled event %s has invalid creation sequence", event.ID)
		}
		if previous, exists := sequences[event.CreationSequence]; exists {
			return fmt.Errorf("scheduled events %s and %s share creation sequence %d", previous, event.ID, event.CreationSequence)
		}
		sequences[event.CreationSequence] = event.ID
		if event.Priority < -1000 || event.Priority > 1000 {
			return fmt.Errorf("scheduled event %s has invalid priority", event.ID)
		}
		dueAt, err := time.Parse(time.RFC3339Nano, event.DueAt)
		if err != nil || dueAt.Location() != time.UTC || dueAt.Format(time.RFC3339Nano) != event.DueAt {
			return fmt.Errorf("scheduled event %s has non-canonical dueAt", event.ID)
		}
		switch event.Kind {
		case SchedulerKindSignal:
			if err := validateSignalLifecycle(event); err != nil {
				return err
			}
		case SchedulerKindAction:
			if err := validateActionLifecycle(event); err != nil {
				return err
			}
		default:
			return fmt.Errorf("scheduled event %s has invalid kind", event.ID)
		}
	}
	return nil
}

func validateSignalLifecycle(event ScheduledEvent) error {
	if event.Action != nil || event.Outcome != nil || event.AttemptCount != 0 || event.FinishedAt != "" {
		return fmt.Errorf("signal event %s contains action lifecycle fields", event.ID)
	}
	switch event.Status {
	case SchedulerPending:
		if event.DeliveredAt != "" || event.CanceledAt != "" {
			return fmt.Errorf("pending scheduled event %s has terminal timestamps", event.ID)
		}
	case SchedulerDelivered:
		if event.DeliveredAt != event.DueAt || event.CanceledAt != "" {
			return fmt.Errorf("delivered scheduled event %s has invalid lifecycle timestamps", event.ID)
		}
	case SchedulerCanceled:
		if err := validateCanceledAt(event); err != nil || event.DeliveredAt != "" {
			return fmt.Errorf("canceled scheduled event %s has invalid lifecycle timestamps", event.ID)
		}
	default:
		return fmt.Errorf("signal event %s has invalid status", event.ID)
	}
	return nil
}

func validateActionLifecycle(event ScheduledEvent) error {
	if event.Payload != nil || event.Action == nil || !toolNamePattern.MatchString(event.Action.Tool) ||
		event.Action.Input == nil || !digestPattern.MatchString(event.Action.SpecDigest) || event.DeliveredAt != "" {
		return fmt.Errorf("scheduled action %s has invalid action envelope", event.ID)
	}
	switch event.Status {
	case SchedulerPending:
		if event.Outcome != nil || event.AttemptCount != 0 || event.FinishedAt != "" || event.CanceledAt != "" {
			return fmt.Errorf("pending scheduled action %s has terminal fields", event.ID)
		}
	case SchedulerCanceled:
		if event.Outcome != nil || event.AttemptCount != 0 || event.FinishedAt != "" || validateCanceledAt(event) != nil {
			return fmt.Errorf("canceled scheduled action %s has invalid lifecycle", event.ID)
		}
	case SchedulerCompleted:
		if event.Outcome == nil || event.AttemptCount != limits.MaxScheduledAttempts || event.FinishedAt != event.DueAt || event.CanceledAt != "" ||
			event.Outcome.CallIndex < 1 || event.Outcome.ErrorClass != "" || !event.Outcome.EffectCommitted || event.Outcome.FaultID != "" || event.Outcome.FaultPhase != "" {
			return fmt.Errorf("completed scheduled action %s has invalid outcome", event.ID)
		}
	case SchedulerFailed:
		if event.Outcome == nil || event.AttemptCount != limits.MaxScheduledAttempts || event.FinishedAt != event.DueAt || event.CanceledAt != "" ||
			event.Outcome.CallIndex < 1 || !controlIDPattern.MatchString(event.Outcome.ErrorClass) ||
			(event.Outcome.EffectCommitted && event.Outcome.FaultPhase != "after-commit-before-response") {
			return fmt.Errorf("failed scheduled action %s has invalid outcome", event.ID)
		}
	default:
		return fmt.Errorf("scheduled action %s has invalid status", event.ID)
	}
	if event.Outcome != nil {
		if (event.Outcome.FaultID == "") != (event.Outcome.FaultPhase == "") {
			return fmt.Errorf("scheduled action %s has incomplete fault identity", event.ID)
		}
		if event.Outcome.FaultID != "" && (!controlIDPattern.MatchString(event.Outcome.FaultID) ||
			(event.Outcome.FaultPhase != "before-validation" && event.Outcome.FaultPhase != "after-commit-before-response")) {
			return fmt.Errorf("scheduled action %s has invalid fault identity", event.ID)
		}
		if event.Outcome.FaultPhase == "after-commit-before-response" && !event.Outcome.EffectCommitted {
			return fmt.Errorf("scheduled action %s has inconsistent after-effect evidence", event.ID)
		}
	}
	return nil
}

func validateCanceledAt(event ScheduledEvent) error {
	canceledAt, err := time.Parse(time.RFC3339Nano, event.CanceledAt)
	if err != nil || canceledAt.Location() != time.UTC || canceledAt.Format(time.RFC3339Nano) != event.CanceledAt {
		return fmt.Errorf("invalid canceledAt")
	}
	return nil
}

func New() *State {
	return &State{
		Entities:  make(map[string]map[string]map[string]any),
		Sequences: make(map[string]int64),
		Entropy:   make(map[string]uint64),
	}
}

func (s *State) Normalize() {
	if s.Entities == nil {
		s.Entities = make(map[string]map[string]map[string]any)
	}
	if s.Sequences == nil {
		s.Sequences = make(map[string]int64)
	}
	if s.Entropy == nil {
		s.Entropy = make(map[string]uint64)
	}
	if s.Scheduler != nil && s.Scheduler.Events == nil {
		s.Scheduler.Events = make(map[string]ScheduledEvent)
	}
	for name, entities := range s.Entities {
		if entities == nil {
			s.Entities[name] = make(map[string]map[string]any)
		}
	}
}

func (s *State) EnsureScheduler() *SchedulerState {
	if s.Scheduler == nil {
		s.Scheduler = &SchedulerState{Events: make(map[string]ScheduledEvent)}
	}
	if s.Scheduler.Events == nil {
		s.Scheduler.Events = make(map[string]ScheduledEvent)
	}
	return s.Scheduler
}

func (event ScheduledEvent) DueTime() (time.Time, error) {
	return time.Parse(time.RFC3339Nano, event.DueAt)
}

func (s *State) Clone() (*State, error) {
	data, err := json.Marshal(s)
	if err != nil {
		return nil, fmt.Errorf("marshal world state: %w", err)
	}
	var result State
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("unmarshal world state: %w", err)
	}
	result.Normalize()
	return &result, nil
}

// CELValue returns a dynamic map deliberately detached from the State struct.
// Expressions can observe this value but cannot mutate persisted state.
func (s *State) CELValue() map[string]any {
	entities := make(map[string]any, len(s.Entities))
	for entity, records := range s.Entities {
		copyRecords := make(map[string]any, len(records))
		for key, value := range records {
			copyRecords[key] = value
		}
		entities[entity] = copyRecords
	}
	sequences := make(map[string]any, len(s.Sequences))
	for key, value := range s.Sequences {
		sequences[key] = value
	}
	return map[string]any{"entities": entities, "sequences": sequences}
}
