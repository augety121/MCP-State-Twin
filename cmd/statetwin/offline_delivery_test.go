package main

import (
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/augety121/mcp-state-twin/internal/agenteval"
	"github.com/augety121/mcp-state-twin/internal/bundle"
)

func preparePackageCLI(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	source := filepath.Join("..", "..", "examples", "package-registry")
	err := filepath.WalkDir(source, func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(source, name)
		if err != nil {
			return err
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return os.MkdirAll(filepath.Join(root, rel), 0700)
		}
		raw, err := os.ReadFile(name)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(root, rel), raw, 0600)
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, ".statetwin"), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := bundle.Build(filepath.Join(root, "bundle-agent.yaml"), filepath.Join(root, ".statetwin", "agent-world.stb")); err != nil {
		t.Fatal(err)
	}
	return root
}
func captureDelivery(t *testing.T, command string, args ...string) ([]byte, error) {
	t.Helper()
	name := filepath.Join(t.TempDir(), "output")
	f, err := os.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = f
	defer func() { os.Stdout = old }()
	if command == "cases" || command == "qualify" {
		err = runTask(context.Background(), append([]string{command}, args...))
	} else {
		err = runAgentEval(context.Background(), append([]string{command}, args...))
	}
	if closeErr := f.Close(); closeErr != nil {
		t.Fatal(closeErr)
	}
	raw, readErr := os.ReadFile(name)
	if readErr != nil {
		t.Fatal(readErr)
	}
	return raw, err
}
func TestOfflineDeliveryPackageRegistryCLI(t *testing.T) {
	root := preparePackageCLI(t)
	raw, err := captureDelivery(t, "qualify", "--root", root, "--cases", "agent-cases.json")
	var q agenteval.TaskQualification
	if err != nil || json.Unmarshal(raw, &q) != nil || q.Decision != "qualified" || q.CaseReport.Matched != 18 || len(q.Tasks) != 6 {
		t.Fatal(string(raw), err)
	}
	raw, err = captureDelivery(t, "cases", "--root", root, "--cases", "agent-cases.json", "--format", "markdown")
	if err != nil || !strings.Contains(string(raw), `"matched": 18`) {
		t.Fatal(string(raw), err)
	}
	reviewArgs := []string{"--root", root, "--suite", "agent-suite.json", "--out", ".statetwin/suite", "--expect", "agent-expectation.json", "--tasks", "reviewed/catalog.json"}
	raw, err = captureDelivery(t, "suite-review", reviewArgs...)
	var review agenteval.SuiteReview
	if err != nil || json.Unmarshal(raw, &review) != nil || review.Decision != "matched" || review.CheckedPairs != 6 {
		t.Fatal(string(raw), err)
	}
	if _, err := os.Stat(filepath.Join(root, ".statetwin", "suite")); !os.IsNotExist(err) {
		t.Fatal("review created output")
	}
	if raw, err = captureDelivery(t, "suite", "--root", root, "--suite", "agent-suite.json", "--out", ".statetwin/suite"); err != nil {
		t.Fatal(string(raw), err)
	}
	for i, policy := range []string{"candidate-pass-v1", "both-pass-v1"} {
		format := []string{"json", "markdown"}[i]
		raw, err = captureDelivery(t, "suite-assess-reviewed", "--root", root, "--out", ".statetwin/suite", "--expect", "agent-expectation.json", "--tasks", "reviewed/catalog.json", "--policy", policy, "--format", format)
		if err != nil || !strings.Contains(string(raw), `"matchedTrials": 12`) {
			t.Fatal(string(raw), err)
		}
	}
	raw, err = captureDelivery(t, "suite-export", "--root", root, "--out", ".statetwin/suite", "--archive", "copy.tar")
	if err != nil || !strings.Contains(string(raw), `"state": "published"`) {
		t.Fatal(string(raw), err)
	}
	raw, err = captureDelivery(t, "suite-import", "--root", root, "--out", ".statetwin/restored", "--archive", "copy.tar")
	if err != nil || !strings.Contains(string(raw), `"state": "published"`) {
		t.Fatal(string(raw), err)
	}
	registry := `{"format":"statetwin.dev/suite-registry/v1alpha1","entries":[{"id":"original","out":".statetwin/suite"},{"id":"restored","out":".statetwin/restored"}]}`
	os.WriteFile(filepath.Join(root, "registry.json"), []byte(registry), 0600)
	raw, err = captureDelivery(t, "inventory", "--root", root, "--registry", "registry.json", "--mode", "metadata")
	var inventory agenteval.SuiteInventory
	if err != nil || json.Unmarshal(raw, &inventory) != nil || inventory.ObservedEntries != 2 || inventory.Entries[0].ReportVerification != "not_checked" {
		t.Fatal(string(raw), err)
	}
	retention := `{"format":"statetwin.dev/retention-intent/v1alpha1","entries":[{"id":"original","intent":"keep","active":false,"references":["restored"]},{"id":"restored","intent":"review","active":false,"references":[]}]}`
	os.WriteFile(filepath.Join(root, "retention.json"), []byte(retention), 0600)
	raw, err = captureDelivery(t, "retention-preview", "--root", root, "--registry", "registry.json", "--policy", "retention.json", "--format", "markdown")
	if err != nil || strings.Count(string(raw), `"status": "protected"`) != 2 {
		t.Fatal(string(raw), err)
	}
	// A coherent source oracle substitution is rejected before any new run.
	taskName := filepath.Join(root, "agent-tasks", "pkg-read-release.json")
	taRaw, _ := os.ReadFile(taskName)
	var ta map[string]any
	json.Unmarshal(taRaw, &ta)
	ta["oracle"].([]any)[0].(map[string]any)["expr"] = "true"
	taRaw, _ = json.Marshal(ta)
	os.WriteFile(taskName, taRaw, 0600)
	raw, err = captureDelivery(t, "suite-review", append(reviewArgs, "--format", "markdown")...)
	if err == nil || !strings.Contains(string(raw), "task_mismatch") {
		t.Fatal(string(raw), err)
	}
}
func TestOfflineDeliveryArgumentsCancellationAndStdout(t *testing.T) {
	for _, command := range []string{"suite-review", "suite-assess-reviewed", "cases", "qualify", "inventory", "retention-preview", "suite-export", "suite-import"} {
		for _, args := range [][]string{nil, {"--unknown"}, {"--format", "html"}, {"position"}} {
			raw, err := captureDelivery(t, command, args...)
			if err == nil || len(raw) != 0 {
				t.Fatal(command, args, string(raw), err)
			}
		}
	}
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "registry.json"), []byte(`{"format":"statetwin.dev/suite-registry/v1alpha1","entries":[{"id":"missing","out":"missing"}]}`), 0600)
	for _, format := range []string{"json", "markdown"} {
		out, err := os.Create(filepath.Join(t.TempDir(), "closed"))
		if err != nil {
			t.Fatal(err)
		}
		out.Close()
		old := os.Stdout
		os.Stdout = out
		err = runOfflineDelivery(context.Background(), "inventory", []string{"--root", root, "--registry", "registry.json", "--mode", "metadata", "--format", format})
		os.Stdout = old
		if err == nil {
			t.Fatal("stdout failure swallowed")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := runOfflineDelivery(ctx, "inventory", []string{"--root", root, "--registry", "registry.json", "--mode", "metadata"}); err == nil {
		t.Fatal("canceled command succeeded")
	}
}
