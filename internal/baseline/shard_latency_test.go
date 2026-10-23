package baseline

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/augety121/mcp-state-twin/internal/baselinepack"
)

func TestBaselineLatencyJSONBoundaries(t *testing.T) {
	latency := 1.0
	row := Observation{TrialID: "trial-001", State: "completed", Verified: true, Cleanup: "complete", Success: true, ObservedModel: "unknown", LatencySeconds: &latency, Usage: Usage{Status: "unavailable"}}
	raw, err := json.Marshal(row)
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"NaN", "Infinity", "-Infinity", "1e309", "-1e309", "null"} {
		t.Run(value, func(t *testing.T) {
			mutated := bytes.Replace(raw, []byte(`"latencySeconds":1`), []byte(`"latencySeconds":`+value), 1)
			var decoded Observation
			if baselinepack.Decode(mutated, 64<<10, &decoded) == nil {
				t.Fatal("non-finite or null latency crossed the JSON evidence boundary")
			}
		})
	}
	// Absence represents unavailable timing; a measured zero is also a valid
	// value. Neither needs fabricated timing or a replacement negative sentinel.
	for _, value := range []*float64{nil, new(float64)} {
		row.LatencySeconds = value
		raw, err := json.Marshal(row)
		if err != nil {
			t.Fatal(err)
		}
		var decoded Observation
		if err = baselinepack.Decode(raw, 64<<10, &decoded); err != nil {
			t.Fatal("valid timing availability rejected", err)
		}
		if (decoded.LatencySeconds == nil) != (value == nil) || (value != nil && *decoded.LatencySeconds != 0) {
			t.Fatal("timing availability changed")
		}
	}
}
