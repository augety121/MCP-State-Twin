package agenthost

import (
	"encoding/json"
	"errors"
)

// MockScript supplies synthetic transport responses, not recorded provider
// traffic. No transcript field can configure an endpoint or credentials.
type MockScript struct {
	Kind          string            `json:"kind"`
	SyntheticOnly bool              `json:"syntheticOnly"`
	Responses     []json.RawMessage `json:"responses"`
}

func DecodeMock(data []byte) (*MockScript, error) {
	var fields map[string]any
	if err := decodeJSON(data, 8<<20, &fields); err != nil {
		return nil, err
	}
	for k := range fields {
		if k != "kind" && k != "syntheticOnly" && k != "responses" {
			return nil, errors.New("HOST_PROTOCOL_ERROR")
		}
	}
	var m MockScript
	if err := decodeJSON(data, 8<<20, &m); err != nil {
		return nil, err
	}
	if m.Kind != "MockResponses" || !m.SyntheticOnly || len(m.Responses) == 0 || len(m.Responses) > 16 {
		return nil, errors.New("HOST_PROFILE_UNSUPPORTED")
	}
	return &m, nil
}
