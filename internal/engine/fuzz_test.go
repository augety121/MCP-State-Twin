package engine

import (
	"strings"
	"testing"

	"github.com/augety121/mcp-state-twin/internal/spec"
)

func FuzzExpressionCompilation(f *testing.F) {
	f.Add("input.name == 'example'")
	f.Add("state.entities.item.all(k, state.entities.item[k].id != '')")
	f.Add("[1, 2, 3].map(x, x * x)")
	f.Add("input[")

	f.Fuzz(func(t *testing.T, expression string) {
		if len(expression) > MaxExpressionBytes*2 {
			t.Skip()
		}
		twin := testSpec()
		twin.Invariants = []spec.InvariantSpec{{ID: "fuzz", Assert: expression}}
		_, _ = New(twin, nil)
	})
}

func TestExpressionSizeLimit(t *testing.T) {
	twin := testSpec()
	twin.Invariants = []spec.InvariantSpec{{ID: "too-large", Assert: strings.Repeat("x", MaxExpressionBytes+1)}}
	if _, err := New(twin, nil); err == nil || !strings.Contains(err.Error(), "expression exceeds") {
		t.Fatalf("expected expression size rejection, got %v", err)
	}
}
