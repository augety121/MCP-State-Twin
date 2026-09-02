package doccheck

import (
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestCELModuleMigrationIsComplete(t *testing.T) {
	root := repositoryRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "cel.dev/cel-go v0.32.0") || strings.Contains(string(data), "github.com/google/cel-go") {
		t.Fatal("CEL module migration pin is missing or the old module is still required")
	}
	err = filepath.WalkDir(filepath.Join(root, "internal"), func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imported := range file.Imports {
			if strings.HasPrefix(imported.Path.Value, `"github.com/google/cel-go/`) {
				t.Errorf("stale CEL import in %s", path)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestFuzzWorkflowPreservesFailureAndEvidence(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(repositoryRoot(t), ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var workflow struct {
		Jobs map[string]struct {
			Timeout int               `yaml:"timeout-minutes"`
			Env     map[string]string `yaml:"env"`
			Steps   []struct {
				Name     string `yaml:"name"`
				Run      string `yaml:"run"`
				If       string `yaml:"if"`
				Continue bool   `yaml:"continue-on-error"`
				With     struct {
					Path string `yaml:"path"`
				} `yaml:"with"`
			} `yaml:"steps"`
		} `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(data, &workflow); err != nil {
		t.Fatal(err)
	}
	job := workflow.Jobs["fuzz-smoke"]
	if job.Timeout != 15 || job.Env["GOMAXPROCS"] != "1" {
		t.Fatal("missing bounded single-worker fuzz job")
	}
	seen := map[string]bool{}
	for _, step := range job.Steps {
		if step.Continue {
			t.Fatal("fuzz job must not suppress a failed step")
		}
		switch step.Name {
		case "TwinSpec parser fuzz":
			seen["spec"] = step.Run == "bash scripts/run-bounded-fuzz.sh spec"
		case "CEL compiler fuzz":
			seen["engine"] = step.Run == "bash scripts/run-bounded-fuzz.sh engine" && step.If == "${{ !cancelled() }}"
		case "Preserve synthetic fuzz counterexamples":
			seen["artifacts"] = step.If == "${{ failure() }}" &&
				strings.TrimSpace(step.With.Path) == "internal/spec/testdata/fuzz/\ninternal/engine/testdata/fuzz/"
		}
	}
	for _, required := range []string{"spec", "engine", "artifacts"} {
		if !seen[required] {
			t.Errorf("missing fuzz evidence contract: %s", required)
		}
	}
}

func TestBoundedFuzzScriptExitStatus(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX wrapper is executed in Linux CI; Windows runs the native Go tests")
	}
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	// No fuzz work here: verify argument admission, exact profile and failure
	// propagation using a synthetic Go-command fixture.
	mock := "#!/usr/bin/env bash\nif [[ $1 == version ]]; then exit 0; fi\nprintf 'maxprocs=%s args=%s\\n' \"$GOMAXPROCS\" \"$*\"\nexit \"${FUZZ_TEST_EXIT:-0}\"\n"
	if err := os.WriteFile(filepath.Join(dir, "go"), []byte(mock), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("GOMAXPROCS", "8")
	for _, test := range []struct {
		target, budget        string
		commandExit, wantExit int
	}{
		{"spec", "200000x", 0, 0}, {"engine", "10000x", 0, 0},
		{"spec", "200000x", 23, 23}, {"unknown", "", 0, 2},
	} {
		t.Setenv("FUZZ_TEST_EXIT", strconv.Itoa(test.commandExit))
		cmd := exec.Command(bash, "scripts/run-bounded-fuzz.sh", test.target)
		cmd.Dir = repositoryRoot(t)
		output, err := cmd.CombinedOutput()
		gotExit := 0
		if err != nil {
			exit, ok := err.(*exec.ExitError)
			if !ok {
				t.Fatal(err)
			}
			gotExit = exit.ExitCode()
		}
		if gotExit != test.wantExit {
			t.Fatalf("%s exit=%d want=%d output=%s", test.target, gotExit, test.wantExit, output)
		}
		if test.budget != "" {
			for _, want := range []string{"maxprocs=1", "-p 1", "-fuzztime=" + test.budget, "-parallel=1", "-timeout=3m", "-fuzzminimizetime=1000x"} {
				if !strings.Contains(string(output), want) {
					t.Errorf("missing %s in %s", want, output)
				}
			}
		}
	}
}
