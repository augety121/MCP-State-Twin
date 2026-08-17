package world

import (
	"encoding/json"
	"fmt"
)

type State struct {
	Entities  map[string]map[string]map[string]any `json:"entities" yaml:"entities"`
	Sequences map[string]int64                     `json:"sequences" yaml:"sequences"`
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
