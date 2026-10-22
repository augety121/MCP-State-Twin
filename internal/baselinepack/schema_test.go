package baselinepack_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/augety121/mcp-state-twin/internal/baseline"
	"github.com/augety121/mcp-state-twin/internal/baselinepack"
	"github.com/augety121/mcp-state-twin/internal/plugin"
)

func TestStrictPluginPlanJSON(t *testing.T) {
	p := plugin.Plan{Format: plugin.PlanFormat, ID: "session", Pack: "pack.json", Profile: "profile.json", EntryID: "entry", Output: "output", Mode: "offline-contract", DeadlineSeconds: 1,
		Host: plugin.Host{Name: "mcp-client", Version: "go-sdk-1.8.0", Model: "mock/statetwin", FreshSession: true, ToolsOnly: true}}
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var decoded plugin.Plan
	if err = baselinepack.Decode(raw, 64<<10, &decoded); err != nil || decoded.Validate() != nil {
		t.Fatalf("valid plan: %v", err)
	}
	for name, replacement := range map[string]string{
		"null-integer": `"additionalTools":null`,
		"case-alias":   `"additionalTools":1,"AdditionalTools":0`,
		"upper-alias":  `"additionalTools":1,"ADDITIONALTOOLS":0`,
		"missing-name": `"AdditionalTools":0`,
	} {
		t.Run(name, func(t *testing.T) {
			mutated := strings.Replace(string(raw), `"additionalTools":0`, replacement, 1)
			var invalid plugin.Plan
			if baselinepack.Decode([]byte(mutated), 64<<10, &invalid) == nil {
				t.Fatal("accepted a null or alternate-spelling host authority declaration")
			}
		})
	}
}

func TestStrictBaselinePolicyJSON(t *testing.T) {
	delta := 0.05
	p := baseline.Policy{Delta: &delta, Floors: map[string]float64{"issues": 0.9, "packages": 0.9}, Required: []string{}, Algorithm: baseline.Algorithm, Seed: 0, BootstrapSamples: 10000}
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	for name, mutation := range map[string][2]string{
		"null-floor": {`"issues":0.9`, `"issues":null`},
		"null-seed":  {`"seed":0`, `"seed":null`},
		"null-delta": {`"delta":0.05`, `"delta":null`},
		"alias-seed": {`"seed":0`, `"seed":1,"Seed":0`},
	} {
		t.Run(name, func(t *testing.T) {
			mutated := strings.Replace(string(raw), mutation[0], mutation[1], 1)
			var invalid baseline.Policy
			if baselinepack.Decode([]byte(mutated), 64<<10, &invalid) == nil {
				t.Fatal("accepted an ambiguous or null statistical policy")
			}
		})
	}
	var decoded baseline.Policy
	if err := baselinepack.Decode(raw, 64<<10, &decoded); err != nil {
		t.Fatal("valid policy rejected", err)
	}
}

func TestStrictClaimAndFrozenBytesJSON(t *testing.T) {
	// A false boolean and an empty string are valid JSON values; null must not
	// silently become either value. Snapshot members also require byte strings.
	for name, value := range map[string]any{
		"claim": struct {
			Revoked bool   `json:"revoked"`
			Reason  string `json:"reason"`
		}{},
		"bytes": struct {
			Inputs map[string][]byte `json:"inputs"`
		}{Inputs: map[string][]byte{"member": []byte("fixture")}},
	} {
		t.Run(name, func(t *testing.T) {
			raw, err := json.Marshal(value)
			if err != nil {
				t.Fatal(err)
			}
			if name == "claim" {
				for _, raw := range []string{`{"revoked":null,"reason":""}`, `{"revoked":false,"reason":null}`} {
					var claim struct {
						Revoked bool   `json:"revoked"`
						Reason  string `json:"reason"`
					}
					if baselinepack.Decode([]byte(raw), 1024, &claim) == nil {
						t.Fatal("accepted null claim scalar")
					}
				}
			} else {
				var inputs struct {
					Inputs map[string][]byte `json:"inputs"`
				}
				if baselinepack.Decode(raw, 1024, &inputs) != nil {
					t.Fatal("valid frozen bytes rejected")
				}
				if baselinepack.Decode([]byte(`{"inputs":{"member":null}}`), 1024, &inputs) == nil {
					t.Fatal("accepted null frozen member")
				}
			}
		})
	}
}

func TestUnconstrainedJSONPayloadKeepsNull(t *testing.T) {
	var document struct {
		Answer any            `json:"answer"`
		Values map[string]any `json:"values"`
	}
	if err := baselinepack.Decode([]byte(`{"answer":null,"values":{"result":null}}`), 1024, &document); err != nil {
		t.Fatal("schema {} must accept JSON null", err)
	}
	if document.Answer != nil || document.Values["result"] != nil {
		t.Fatal("null payload changed")
	}
	if err := baselinepack.Decode([]byte(`{"values":{}}`), 1024, &document); err == nil {
		t.Fatal("required payload may be null but cannot be omitted")
	}
}
