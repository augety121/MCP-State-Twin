package world

import (
	"encoding/json"
	"fmt"

	"github.com/augety121/mcp-state-twin/internal/limits"
)

type State struct {
	Entities  map[string]map[string]map[string]any `json:"entities" yaml:"entities"`
	Sequences map[string]int64                     `json:"sequences" yaml:"sequences"`
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
	if err := limits.ValidateJSON(s, limits.MaxStateBytes); err != nil {
		return fmt.Errorf("world state: %w", err)
	}
	return nil
}

func New() *State {
	return &State{
		Entities:  make(map[string]map[string]map[string]any),
		Sequences: make(map[string]int64),
	}
}

func (s *State) Normalize() {
	if s.Entities == nil {
		s.Entities = make(map[string]map[string]map[string]any)
	}
	if s.Sequences == nil {
		s.Sequences = make(map[string]int64)
	}
	for name, entities := range s.Entities {
		if entities == nil {
			s.Entities[name] = make(map[string]map[string]any)
		}
	}
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
