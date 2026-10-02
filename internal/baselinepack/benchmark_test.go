package baselinepack

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/augety121/mcp-state-twin/internal/testfixture"
)

func BenchmarkPackPreparation(b *testing.B) {
	for _, size := range []int{1, 24} {
		name := "single"
		if size == 24 {
			name = "catalog"
		}
		b.Run(name, func(b *testing.B) {
			root := testfixture.Baseline(b)
			file := filepath.Join(root, "baseline-pack.json")
			raw, err := os.ReadFile(file)
			if err != nil {
				b.Fatal(err)
			}
			var pack Pack
			if err = json.Unmarshal(raw, &pack); err != nil {
				b.Fatal(err)
			}
			pack.Entries = pack.Entries[:size]
			raw, err = json.Marshal(pack)
			if err != nil {
				b.Fatal(err)
			}
			if err = os.WriteFile(file, raw, 0600); err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := Prepare(context.Background(), root, "baseline-pack.json", "plugin-profile.json"); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
