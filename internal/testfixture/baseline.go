// Package testfixture copies only synthetic source fixtures into owned test roots.
package testfixture

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/augety121/mcp-state-twin/internal/bundle"
)

func Baseline(t testing.TB) string {
	t.Helper()
	_, source, _, _ := runtime.Caller(0)
	src := filepath.Join(filepath.Dir(source), "..", "..", "examples")
	root := t.TempDir()
	err := filepath.WalkDir(src, func(n string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, n)
		if err != nil {
			return err
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return os.MkdirAll(filepath.Join(root, rel), 0700)
		}
		if !strings.HasSuffix(n, ".json") && !strings.HasSuffix(n, ".yaml") {
			return nil
		}
		raw, err := os.ReadFile(n)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(root, rel), raw, 0600)
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, domain := range []string{"issue-tracker", "package-registry"} {
		dir := filepath.Join(root, domain)
		if err := os.Mkdir(filepath.Join(dir, ".statetwin"), 0700); err != nil {
			t.Fatal(err)
		}
		worlds := []string{"agent"}
		if domain == "package-registry" {
			worlds = append(worlds, "project")
		}
		for _, world := range worlds {
			for _, reviewed := range []bool{false, true} {
				manifest := filepath.Join(dir, "bundle-"+world+".yaml")
				out := world + "-world.stb"
				if reviewed {
					manifest = filepath.Join(dir, "reviewed", "world", world, "bundle-"+world+".yaml")
					out = "reviewed-" + out
				}
				if _, err := bundle.Build(manifest, filepath.Join(dir, ".statetwin", out)); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	return root
}
