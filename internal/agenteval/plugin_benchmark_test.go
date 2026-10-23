package agenteval

import (
	"context"
	"testing"

	"github.com/augety121/mcp-state-twin/internal/bundle"
	"github.com/augety121/mcp-state-twin/internal/testfixture"
)

// Fixtures, witness decoding and the replay benchmark's original terminal are
// excluded. Every timed lifecycle still creates and tears down its own world.
func BenchmarkPluginPreparation(b *testing.B) {
	root := testfixture.Baseline(b)
	ctx := context.Background()
	for _, id := range []string{"read-issue", "close-issue", "after-commit-confirm"} {
		ta, raw, witness := pluginPreparationInputs(b, root, id)
		b.Run(id, func(b *testing.B) {
			world, err := bundle.OpenBytes(raw)
			if err != nil {
				b.Fatal(err)
			}
			run := func(runB *testing.B) *PluginTerminal {
				w, err := NewPluginWorld(ctx, ta, raw)
				if err != nil {
					runB.Fatal(err)
				}
				defer w.Finish("cancelled", nil)
				for _, call := range witness.Calls {
					if _, err = w.Call(ctx, call); err != nil {
						runB.Fatal(err)
					}
				}
				terminal, err := w.Finish("completed", witness.Answer)
				if err != nil || terminal.Evaluation.Outcome != ta.ExpectedOutcome {
					runB.Fatal(terminal, err)
				}
				return terminal
			}
			terminal := run(b)
			b.Run("lifecycle", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					run(b)
				}
			})
			b.Run("replay", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					if err := VerifyPluginTerminal(ctx, terminal, ta, raw); err != nil {
						b.Fatal(err)
					}
				}
			})
			b.Run("inprocess-witness", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					r, err := RunWitness(ctx, ta, world, witness)
					if err != nil || r.Evaluation.Outcome != ta.ExpectedOutcome {
						b.Fatal(r, err)
					}
				}
			})
		})
	}
}
