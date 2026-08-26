package world

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/augety121/mcp-state-twin/internal/limits"
)

// DecodeStrict decodes one bounded fixture document. It is shared by CLI,
// bundle admission, and Episode execution so all entry points enforce the same
// unknown-field, trailing-data, and world-budget rules.
func DecodeStrict(data []byte) (*State, error) {
	if len(data) > limits.MaxStateBytes {
		return nil, fmt.Errorf("document exceeds %d bytes", limits.MaxStateBytes)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	decoder.UseNumber()
	var state State
	if err := decoder.Decode(&state); err != nil {
		return nil, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return nil, errors.New("document must contain one JSON value")
	}
	state.Normalize()
	if err := state.ValidateBudget(); err != nil {
		return nil, err
	}
	return &state, nil
}
