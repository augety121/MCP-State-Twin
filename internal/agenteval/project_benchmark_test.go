package agenteval

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/augety121/mcp-state-twin/internal/bundle"
)

// BenchmarkProjectPrepare includes real reads, strict admission and bundle
// decoding. The uncached baseline uses the same validation functions.
func BenchmarkProjectPrepare(b *testing.B) {
	for _, pairs := range []int{6, 12, 16} {
		for _, size := range []struct {
			name string
			pad  int
		}{{"small", 0}, {"medium", 64 << 10}, {"near-suite-limit", 3 << 20}} {
			for _, distinct := range []bool{false, true} {
				b.Run(fmt.Sprintf("pairs%d/%s/distinct%t", pairs, size.name, distinct), func(b *testing.B) {
					root := campaignAssets(b)
					dir := filepath.Join(root, "examples", "issue-tracker")
					// Whitespace increases raw member size without changing the world state.
					if size.pad > 0 {
						file := filepath.Join(dir, "agent-state.json")
						raw, err := os.ReadFile(file)
						if err != nil {
							b.Fatal(err)
						}
						raw = append(raw, make([]byte, size.pad)...)
						for i := len(raw) - size.pad; i < len(raw); i++ {
							raw[i] = ' '
						}
						if err := os.WriteFile(file, raw, 0600); err != nil {
							b.Fatal(err)
						}
						out := filepath.Join(dir, ".statetwin", "padded.stb")
						if _, err := bundle.Build(filepath.Join(dir, "bundle-agent.yaml"), out); err != nil {
							b.Fatal(err)
						}
						raw, err = os.ReadFile(out)
						if err != nil {
							b.Fatal(err)
						}
						for _, n := range []string{"agent-world.stb", "reviewed-agent-world.stb"} {
							if err := os.WriteFile(filepath.Join(dir, ".statetwin", n), raw, 0600); err != nil {
								b.Fatal(err)
							}
						}
					}
					var suite SuitePlan
					raw, _ := os.ReadFile(filepath.Join(dir, "project-suite-extended.json"))
					json.Unmarshal(raw, &suite)
					original := append([]SuitePair{}, suite.Pairs...)
					suite.Pairs = nil
					for i := 0; i < pairs; i++ {
						p := original[i%len(original)]
						p.Repeat = i/len(original) + 1
						if distinct {
							for _, field := range []*string{&p.Task, &p.BaselineResponses, &p.CandidateResponses} {
								raw, err := os.ReadFile(filepath.Join(dir, *field))
								if err != nil {
									b.Fatal(err)
								}
								name := fmt.Sprintf("bench-%02d-%s", i, filepath.Base(*field))
								if err := os.WriteFile(filepath.Join(dir, name), raw, 0600); err != nil {
									b.Fatal(err)
								}
								*field = name
							}
						}
						suite.Pairs = append(suite.Pairs, p)
					}
					write := func(n string, v any) {
						raw, err := json.Marshal(v)
						if err != nil {
							b.Fatal(err)
						}
						if err := os.WriteFile(filepath.Join(dir, n), raw, 0600); err != nil {
							b.Fatal(err)
						}
					}
					write("project-suite-extended.json", suite)
					write("project-expectation-extended.json", SuiteExpectation{Format: ExpectationFormat, Profile: SuiteProfile, MaxOutputTokens: suite.MaxOutputTokens, Plan: suite.comparisonPlan()})
					for _, off := range []bool{true, false} {
						b.Run(fmt.Sprintf("cache%t", !off), func(b *testing.B) {
							b.ReportAllocs()
							b.ResetTimer()
							reads, decodes := 0, 0
							for i := 0; i < b.N; i++ {
								s := newProjectSource(context.Background(), dir, "", projectInputLimit)
								s.noCache = off
								if _, _, err := prepareProject(s, ".", "project-extended.json"); err != nil {
									b.Fatal(err)
								}
								reads += s.reads
								decodes += s.decodes
							}
							b.ReportMetric(float64(reads)/float64(b.N), "reads/op")
							b.ReportMetric(float64(decodes)/float64(b.N), "decodes/op")
						})
					}
				})
			}
		}
	}
}

func BenchmarkProjectAssess(b *testing.B) {
	for _, pairs := range []int{6, 12, 16} {
		b.Run(fmt.Sprintf("pairs%d", pairs), func(b *testing.B) {
			root := campaignAssets(b)
			dir := filepath.Join(root, "examples", "issue-tracker")
			s := newProjectSource(context.Background(), dir, "", projectInputLimit)
			p, _, err := prepareProject(s, ".", "project-extended.json")
			if err != nil {
				b.Fatal(err)
			}
			// Replicate the admitted six-task plan with fresh trial identities.
			var plan SuitePlan
			raw, _ := os.ReadFile(filepath.Join(dir, "project-suite-extended.json"))
			json.Unmarshal(raw, &plan)
			orig := append([]SuitePair{}, plan.Pairs...)
			plan.Pairs = nil
			for i := 0; i < pairs; i++ {
				pair := orig[i%6]
				pair.Repeat = i/6 + 1
				plan.Pairs = append(plan.Pairs, pair)
			}
			raw, _ = json.Marshal(plan)
			p.suite, err = PrepareSuite(context.Background(), dir, raw)
			if err != nil {
				b.Fatal(err)
			}
			p.expect.Plan = plan.comparisonPlan()
			if _, err := RunSuite(context.Background(), dir, "benchmark-suite", p.suite); err != nil {
				b.Fatal(err)
			}
			fs, err := os.OpenRoot(dir)
			if err != nil {
				b.Fatal(err)
			}
			defer fs.Close()
			view := diskReadRoot{fs}
			terminal := filepath.Join(dir, "benchmark-suite", "candidate-01", "terminal.json")
			saved, _ := os.ReadFile(terminal)
			for _, state := range []string{"valid", "missing", "corrupt"} {
				b.Run(state, func(b *testing.B) {
					if state == "missing" {
						os.Remove(terminal)
					} else if state == "corrupt" {
						os.WriteFile(terminal, []byte("{}"), 0600)
					} else {
						os.WriteFile(terminal, saved, 0600)
					}
					b.ReportAllocs()
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						if _, _, _, err := assessProject(context.Background(), view, "benchmark-suite", p); err != nil {
							b.Fatal(err)
						}
					}
				})
			}
		})
	}
}
