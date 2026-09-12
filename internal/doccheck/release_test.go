package doccheck

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

type releaseStep struct {
	Name     string            `yaml:"name"`
	ID       string            `yaml:"id"`
	Run      string            `yaml:"run"`
	Uses     string            `yaml:"uses"`
	If       string            `yaml:"if"`
	Continue any               `yaml:"continue-on-error"`
	With     map[string]any    `yaml:"with"`
	Env      map[string]string `yaml:"env"`
}
type releaseJob struct {
	Needs       any               `yaml:"needs"`
	Uses        string            `yaml:"uses"`
	If          string            `yaml:"if"`
	Continue    any               `yaml:"continue-on-error"`
	Permissions map[string]string `yaml:"permissions"`
	Outputs     map[string]string `yaml:"outputs"`
	Steps       []releaseStep     `yaml:"steps"`
	Strategy    struct {
		Matrix map[string][]string `yaml:"matrix"`
	} `yaml:"strategy"`
}
type releaseWorkflow struct {
	On          map[string]any        `yaml:"on"`
	Permissions map[string]string     `yaml:"permissions"`
	Jobs        map[string]releaseJob `yaml:"jobs"`
	Concurrency struct {
		Group  string `yaml:"group"`
		Cancel bool   `yaml:"cancel-in-progress"`
	} `yaml:"concurrency"`
}

func loadWorkflow(t *testing.T, name string) releaseWorkflow {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(repositoryRoot(t), ".github", "workflows", name))
	if err != nil {
		t.Fatal(err)
	}
	var w releaseWorkflow
	if err = yaml.Unmarshal(raw, &w); err != nil {
		t.Fatal(err)
	}
	return w
}
func jobScript(j releaseJob) string {
	var b strings.Builder
	for _, s := range j.Steps {
		b.WriteString(s.Run)
		b.WriteByte('\n')
	}
	return b.String()
}
func requireTokens(t *testing.T, script string, tokens ...string) {
	t.Helper()
	for _, token := range tokens {
		if !strings.Contains(script, token) {
			t.Errorf("missing release contract %q", token)
		}
	}
}

func TestReleaseWorkflowRequiresFullSameCommitGates(t *testing.T) {
	w := loadWorkflow(t, "release.yml")
	ci := loadWorkflow(t, "ci.yml")
	if len(w.Permissions) != 1 || w.Permissions["contents"] != "read" {
		t.Fatal("release token is not read-only by default")
	}
	if w.Concurrency.Group != "release-${{ github.ref }}" || w.Concurrency.Cancel {
		t.Fatal("release ref serialization changed")
	}
	if len(w.Jobs) != 3 || fmt.Sprint(w.Jobs["gates"].Needs) != "admit" || fmt.Sprint(w.Jobs["stage"].Needs) != "[admit gates]" {
		t.Fatal("release admission/gates dependency missing")
	}
	if w.Jobs["gates"].Uses != "./.github/workflows/ci.yml" {
		t.Fatal("gates are not pinned by same-repository commit")
	}
	if w.Jobs["gates"].Permissions["contents"] != "read" || w.Jobs["stage"].Permissions["contents"] != "write" {
		t.Fatal("release privilege boundary changed")
	}
	if _, ok := ci.On["workflow_call"]; !ok {
		t.Fatal("CI not reusable")
	}
	for _, event := range []string{"push", "pull_request"} {
		if _, ok := ci.On[event]; !ok {
			t.Fatal("ordinary CI trigger lost")
		}
	}
	for _, name := range []string{"test", "platform-smoke", "fuzz-smoke", "secret-policy", "hermetic-egress", "mcp-conformance"} {
		j, ok := ci.Jobs[name]
		if !ok || j.If != "" || j.Continue != nil {
			t.Fatalf("required CI gate bypassed: %s", name)
		}
	}
	if fmt.Sprint(ci.Jobs["platform-smoke"].Strategy.Matrix["os"]) != "[windows-latest macos-latest]" {
		t.Fatal("platform matrix incomplete")
	}
	requireTokens(t, jobScript(ci.Jobs["test"]), "go vet ./...", "go test -race", "./internal/doccheck", "scenario-close-issue.yaml", "scenario-release-lifecycle.yaml", "episode inspect", "protocols", "go build")
	requireTokens(t, jobScript(ci.Jobs["fuzz-smoke"]), "run-bounded-fuzz.sh spec", "run-bounded-fuzz.sh engine")
	requireTokens(t, jobScript(ci.Jobs["hermetic-egress"]), "unshare --net", "run-hermetic-tests.sh")
	requireTokens(t, jobScript(ci.Jobs["mcp-conformance"]), "run-conformance.sh")
	requireTokens(t, jobScript(ci.Jobs["secret-policy"]), "check-sensitive-fixtures.sh")
	pin := regexp.MustCompile(`@[0-9a-f]{40}$`)
	checkout := ci.Jobs["test"].Steps[0].Uses
	for name, j := range w.Jobs {
		if j.If != "" || j.Continue != nil {
			t.Fatalf("release job bypass: %s", name)
		}
		for _, s := range j.Steps {
			if s.If != "" || s.Continue != nil {
				t.Fatal("release step bypass")
			}
			if s.Uses != "" && !pin.MatchString(s.Uses) {
				t.Fatal("unpinned action", s.Uses)
			}
			if strings.HasPrefix(s.Uses, "actions/checkout@") && s.With["ref"] != "${{ github.sha }}" {
				t.Fatal("checkout not exact candidate")
			}
			if strings.HasPrefix(s.Uses, "actions/checkout@") && s.Uses != checkout {
				t.Fatal("release checkout pin drifted from CI")
			}
		}
	}
	a := jobScript(w.Jobs["admit"])
	requireTokens(t, a, "./cmd/releasecheck", `--tag "${GITHUB_REF_NAME}"`, `--format github`, "${GITHUB_OUTPUT}", "refs/tags/${GITHUB_REF_NAME}^{commit}", "${GITHUB_SHA}", "merge-base --is-ancestor")
	s := jobScript(w.Jobs["stage"])
	requireTokens(t, s, `git fetch origin "refs/tags/${GITHUB_REF_NAME}" --no-tags`, "FETCH_HEAD^{commit}", "${GITHUB_SHA}", "bash scripts/build-release.sh", "--verify-tag", "--draft", "--latest=false", "--notes-file", "--prerelease", "*) exit 2", `gh release create "${GITHUB_REF_NAME}" dist/* "${flags[@]}"`)
	if strings.Index(s, "FETCH_HEAD^{commit}") > strings.Index(s, "bash scripts/build-release.sh") {
		t.Fatal("tag recheck happens too late")
	}
	for _, unsafe := range []string{"--generate-notes", "gh release edit", "gh release delete", "--clobber", "git tag ", "|| true", "${{ github.ref_name }}"} {
		if strings.Contains(a+s, unsafe) {
			t.Fatal("unsafe release fallback", unsafe)
		}
	}
	stage := w.Jobs["stage"].Steps[len(w.Jobs["stage"].Steps)-1]
	if stage.Env["RELEASE_PRERELEASE"] != "${{ needs.admit.outputs.prerelease }}" || stage.Env["RELEASE_NOTES"] != "${{ needs.admit.outputs.notes }}" {
		t.Fatal("unvalidated release metadata source")
	}
}

func TestReleaseBuildHasNoDestructiveFallback(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repositoryRoot(t), "scripts", "build-release.sh"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	for _, bad := range []string{"rm -", "mkdir -p", "eval ", "source ", "|| true"} {
		if strings.Contains(s, bad) {
			t.Fatal("unsafe packaging operation", bad)
		}
	}
	requireTokens(t, s, "./cmd/releasecheck", `source_status="$(git status --porcelain --untracked-files=normal)"`, `test -z "$source_status"`, `tag_revision="$(git rev-parse --verify "refs/tags/${tag}^{commit}")"`, `test "$tag_revision" = "$revision"`, "umask 077", "mkdir dist", "go build -p 1", `GOMAXPROCS="${GOMAXPROCS:-1}"`)
	if strings.Index(s, "refs/tags/${tag}^{commit}") > strings.Index(s, "mkdir dist") {
		t.Fatal("output created before tag admission")
	}
}

func TestReleaseBuildRefusalAndFailurePropagation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX build wrapper executes in Linux/macOS CI; native release policy is tested separately")
	}
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(repositoryRoot(t), "scripts", "build-release.sh")
	for _, kind := range []string{"no-argument", "extra-argument", "policy-refused", "dirty", "status-fails", "wrong-root", "missing-tag", "wrong-tag", "existing-directory", "existing-file", "existing-symlink", "last-build-fails"} {
		t.Run(kind, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), "workspace with spaces")
			bin := filepath.Join(t.TempDir(), "bin")
			for _, dir := range []string{root, bin} {
				if err := os.Mkdir(dir, 0700); err != nil {
					t.Fatal(err)
				}
			}
			git := `#!/usr/bin/env bash
set -eu
case "$*" in
  "rev-parse HEAD") printf 'synthetic-revision\n' ;;
  "rev-parse --show-toplevel") if [[ $RELEASE_TEST_CASE == wrong-root ]]; then printf '/not-the-root\n'; else pwd -P; fi ;;
  "status --porcelain --untracked-files=normal")
    if [[ $RELEASE_TEST_CASE == status-fails ]]; then exit 40; fi
    if [[ $RELEASE_TEST_CASE == dirty ]]; then printf ' M source.go\n'; fi ;;
  "rev-parse --verify "*)
    case "$RELEASE_TEST_CASE" in
      missing-tag) exit 39 ;;
      wrong-tag) printf 'other-revision\n' ;;
      *) printf 'synthetic-revision\n' ;;
    esac ;;
  *) exit 90 ;;
esac
`
			goStub := `#!/usr/bin/env bash
set -eu
if [[ $1 == run ]]; then
  if [[ $RELEASE_TEST_CASE == policy-refused ]]; then exit 38; fi
  exit 0
fi
[[ $1 == build ]] || exit 91
printf '%s/%s procs=%s cgo=%s args=%s\n' "$GOOS" "$GOARCH" "$GOMAXPROCS" "$CGO_ENABLED" "$*" >> calls.log
# The last build deliberately fails: no real compiler, hashes or release manifest.
[[ $GOOS != windows ]] || exit 41
output=''
while [[ $# -gt 0 ]]; do
  if [[ $1 == -o ]]; then shift; output=$1; fi
  shift
done
printf 'synthetic build fixture\n' > "$output"
`
			for name, data := range map[string]string{"git": git, "go": goStub} {
				if err := os.WriteFile(filepath.Join(bin, name), []byte(data), 0700); err != nil {
					t.Fatal(err)
				}
			}
			t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
			t.Setenv("RELEASE_TEST_CASE", kind)
			t.Setenv("GOMAXPROCS", "")
			dist := filepath.Join(root, "dist")
			owner := ""
			switch kind {
			case "existing-directory":
				if err := os.Mkdir(dist, 0700); err != nil {
					t.Fatal(err)
				}
				owner = filepath.Join(dist, "owner.txt")
			case "existing-file":
				owner = dist
			case "existing-symlink":
				target := t.TempDir()
				owner = filepath.Join(target, "owner.txt")
				if err := os.Symlink(target, dist); err != nil {
					t.Fatal(err)
				}
			}
			if owner != "" {
				if err := os.WriteFile(owner, []byte("owned-before-build"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			args := []string{script, "v0.1.0-alpha.2"}
			if kind == "no-argument" {
				args = args[:1]
			}
			if kind == "extra-argument" {
				args = append(args, "unexpected")
			}
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, bash, args...)
			cmd.Dir = root
			out, err := cmd.CombinedOutput()
			exit, ok := err.(*exec.ExitError)
			if !ok {
				t.Fatalf("expected refusal: %v %s", err, out)
			}
			want := 1
			switch kind {
			case "no-argument", "extra-argument":
				want = 2
			case "policy-refused":
				want = 38
			case "missing-tag":
				want = 39
			case "status-fails":
				want = 40
			case "last-build-fails":
				want = 41
			}
			if exit.ExitCode() != want {
				t.Fatalf("exit %d want %d: %s", exit.ExitCode(), want, out)
			}
			calls, callErr := os.ReadFile(filepath.Join(root, "calls.log"))
			if kind == "last-build-fails" {
				if callErr != nil {
					t.Fatal(callErr)
				}
				lines := strings.Split(strings.TrimSpace(string(calls)), "\n")
				if len(lines) != 5 {
					t.Fatal("not all five targets attempted")
				}
				for i, target := range []string{"linux/amd64", "linux/arm64", "darwin/amd64", "darwin/arm64", "windows/amd64"} {
					requireTokens(t, lines[i], target, "procs=1", "cgo=0", "build -p 1", "-trimpath", "Version=0.1.0-alpha.2", "Revision=synthetic-revision")
				}
				entries, err := os.ReadDir(dist)
				if err != nil || len(entries) != 4 {
					t.Fatal("partial owned output lost", err)
				}
			} else if !os.IsNotExist(callErr) {
				t.Fatal("build started before admission")
			}
			if owner != "" {
				got, err := os.ReadFile(owner)
				if err != nil || string(got) != "owned-before-build" {
					t.Fatal("existing data changed")
				}
			}
			if owner == "" && kind != "last-build-fails" {
				if _, err := os.Lstat(dist); !os.IsNotExist(err) {
					t.Fatal("output created before admission")
				}
			}
		})
	}
}
