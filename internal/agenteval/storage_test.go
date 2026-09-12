package agenteval

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/augety121/mcp-state-twin/internal/bundle"
	"github.com/augety121/mcp-state-twin/internal/limits"
	"github.com/augety121/mcp-state-twin/internal/task"
)

type failingEvidenceFS struct {
	evidenceFS
	op, leaf string
	after    bool
	observed []string
	hook     func(string)
}

func (f *failingEvidenceFS) operation(op, name string) error {
	key := op + ":" + path.Base(name)
	f.observed = append(f.observed, key)
	if f.hook != nil {
		f.hook(key)
	}
	if op == f.op && path.Base(name) == f.leaf {
		return syscall.ENOSPC
	}
	return nil
}
func (f *failingEvidenceFS) Mkdir(name string, mode os.FileMode) error {
	if err := f.operation("mkdir", name); err != nil {
		return err
	}
	return f.evidenceFS.Mkdir(name, mode)
}
func (f *failingEvidenceFS) CreateExclusive(name string) (evidenceFile, error) {
	if err := f.operation("open", name); err != nil {
		return nil, err
	}
	w, err := f.evidenceFS.CreateExclusive(name)
	if err != nil {
		return nil, err
	}
	return &failingEvidenceFile{evidenceFile: w, fs: f, name: name}, nil
}
func (f *failingEvidenceFS) Link(old, new string) error {
	if !f.after {
		if err := f.operation("link", new); err != nil {
			return err
		}
	}
	if err := f.evidenceFS.Link(old, new); err != nil {
		return err
	}
	if f.after {
		return f.operation("link", new)
	}
	return nil
}
func (f *failingEvidenceFS) Remove(name string) error {
	if !f.after {
		if err := f.operation("remove", name); err != nil {
			return err
		}
	}
	if err := f.evidenceFS.Remove(name); err != nil {
		return err
	}
	if f.after {
		return f.operation("remove", name)
	}
	return nil
}

type failingEvidenceFile struct {
	evidenceFile
	fs   *failingEvidenceFS
	name string
}

func (f *failingEvidenceFile) Write(p []byte) (int, error) {
	if f.fs.op == "short" && path.Base(f.name) == f.fs.leaf {
		return f.evidenceFile.Write(p[:len(p)/2])
	}
	if err := f.fs.operation("write", f.name); err != nil {
		return 0, err
	}
	return f.evidenceFile.Write(p)
}
func (f *failingEvidenceFile) Sync() error {
	if !f.fs.after {
		if err := f.fs.operation("sync", f.name); err != nil {
			return err
		}
	}
	if err := f.evidenceFile.Sync(); err != nil {
		return err
	}
	if f.fs.after {
		return f.fs.operation("sync", f.name)
	}
	return nil
}
func (f *failingEvidenceFile) Close() error {
	err := f.evidenceFile.Close()
	if failure := f.fs.operation("close", f.name); failure != nil {
		return failure
	}
	return err
}
func recordStorageCase(ctx context.Context, root string, ta *task.Task, raw []byte, w *Witness, open openEvidenceFS, started *int) (*AgentEpisode, error) {
	c := mockConfig("trial")
	e := &AgentEvidence{Format: EvidenceFormat, Bundle: base64.StdEncoding.EncodeToString(raw)}
	return recordWithStorage(ctx, root, "trial", ta, raw, c, func(b *bundle.Artifact, stage stageEpisode) (*AgentEpisode, error) {
		*started++
		return runMock(ctx, ta, b, c, mockWitness(w), stage)
	}, func(r *AgentEpisode) any { e.Episode = r; return e }, func(ctx context.Context, r *AgentEpisode) error { e.Episode = r; return replay(ctx, e, false) }, open)
}

func TestEvidenceStorageFailureMatrix(t *testing.T) {
	_, load := kit(t)
	ta, w := load("close-issue")
	raw := rawBundle(t)
	type point struct {
		op, leaf string
		after    bool
	}
	points := []point{{"mkdir", "trial", false}, {"link", "terminal.json", false}, {"link", "terminal.json", true}, {"remove", "terminal.pending.json", false}, {"remove", "closure.json", false}, {"remove", "terminal.pending.json", true}, {"remove", "closure.json", true}}
	for _, leaf := range []string{"claim.json", "closure.json", "terminal.pending.json"} {
		for _, op := range []string{"open", "write", "short", "sync", "close"} {
			points = append(points, point{op, leaf, false})
		}
	}
	for _, p := range points {
		t.Run(p.op+"/"+p.leaf+map[bool]string{true: "/after", false: "/before"}[p.after], func(t *testing.T) {
			root := t.TempDir()
			started := 0
			_, err := recordStorageCase(context.Background(), root, ta, raw, w, func(root string) (evidenceFS, error) {
				fs, err := openRootedEvidenceFS(root)
				if err != nil {
					return nil, err
				}
				return &failingEvidenceFS{evidenceFS: fs, op: p.op, leaf: p.leaf, after: p.after}, nil
			}, &started)
			if err == nil {
				t.Fatal("storage failure hidden")
			}
			if p.leaf == "claim.json" || p.op == "mkdir" {
				if started != 0 {
					t.Fatal("execution before durable claim")
				}
			} else if started != 1 {
				t.Fatal("execution repeated")
			}
			terminal, readErr := os.ReadFile(filepath.Join(root, "trial", "terminal.json"))
			published := p.op == "remove" || (p.op == "link" && p.after)
			if published {
				if readErr != nil {
					t.Fatal("published artifact lost", readErr)
				}
				e, err := DecodeEvidence(terminal)
				if err != nil || VerifyEvidence(context.Background(), e) != nil {
					t.Fatal("published artifact invalid")
				}
			} else if !os.IsNotExist(readErr) {
				t.Fatal("failure promoted incomplete publication")
			}
			if p.op != "mkdir" {
				if _, err := recordStorageCase(context.Background(), root, ta, raw, w, openRootedEvidenceFS, &started); err == nil {
					t.Fatal("failed directory reused")
				}
			}
			if strings.Contains(err.Error(), root) {
				t.Fatal("filesystem details leaked")
			}
		})
	}
}

func TestEvidenceWriterPrivateBoundsAndNoClobber(t *testing.T) {
	for _, value := range []any{map[string]any{"note": "api_key=synthetic-private-sentinel"}, make(chan int)} {
		opened := false
		_, err := claimEvidence(t.TempDir(), "trial", value, func(string) (evidenceFS, error) { opened = true; return nil, nil })
		if err == nil || opened {
			t.Fatal("unsafe claim reached filesystem")
		}
	}
	root := t.TempDir()
	w, err := claimEvidence(root, "trial", mockConfig("trial"), openRootedEvidenceFS)
	if err != nil {
		t.Fatal(err)
	}
	defer w.fs.Close()
	if err = w.write("../outside.json", map[string]any{}); err == nil {
		t.Fatal("writer path escaped")
	}
	if err = w.write("closure.json", map[string]any{"api_key": "synthetic-private-sentinel"}); err == nil {
		t.Fatal("private staging written")
	}
	if _, err = os.Stat(filepath.Join(root, "trial", "closure.json")); !os.IsNotExist(err) {
		t.Fatal("unsafe content file created")
	}
	if err = w.write("closure.json", map[string]any{"note": strings.Repeat("x", limits.MaxReportBytes)}); err == nil || err.Error() != "EVIDENCE_WRITE_FAILED" {
		t.Fatal("oversized envelope accepted", err)
	}
	if _, err = os.Stat(filepath.Join(root, "trial", "closure.json")); !os.IsNotExist(err) {
		t.Fatal("oversized envelope created a file")
	}
	if err = w.write("closure.json", map[string]any{"ok": true}); err != nil {
		t.Fatal(err)
	}
	if err = w.write("closure.json", map[string]any{"ok": false}); err == nil {
		t.Fatal("staging overwritten")
	}
	if err = w.write("terminal.pending.json", map[string]any{"ok": true}); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(root, "trial", "terminal.json"), []byte("existing operator artifact"), 0600); err != nil {
		t.Fatal(err)
	}
	if w.publish() == nil {
		t.Fatal("terminal overwritten")
	}
	got, _ := os.ReadFile(filepath.Join(root, "trial", "terminal.json"))
	if string(got) != "existing operator artifact" {
		t.Fatal("operator artifact changed")
	}
}

// The child exits only at a synthetic, parent-owned test directory. This is
// real process-exit evidence, not hardware power loss or filesystem fsync proof.
func TestEvidenceCrashHelper(t *testing.T) {
	point := os.Getenv("STATETWIN_TEST_EVIDENCE_CUT")
	if point == "" {
		return
	}
	root := os.Getenv("STATETWIN_TEST_EVIDENCE_ROOT")
	data, err := task.ReadFile(root, "task.json", task.MaxBytes)
	if err != nil {
		t.Fatal(err)
	}
	ta, err := task.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	data, err = task.ReadFile(root, "witness.json", task.MaxBytes)
	if err != nil {
		t.Fatal(err)
	}
	w, err := DecodeWitness(data)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := task.ReadFile(root, "bundle.stb", 32<<20)
	if err != nil {
		t.Fatal(err)
	}
	started := 0
	_, err = recordStorageCase(context.Background(), root, ta, raw, w, func(root string) (evidenceFS, error) {
		fs, err := openRootedEvidenceFS(root)
		if err != nil {
			return nil, err
		}
		return &failingEvidenceFS{evidenceFS: fs, after: true, hook: func(key string) {
			if key == point {
				os.Exit(67)
			}
		}}, nil
	}, &started)
	t.Fatalf("cut point was not reached: %v", err)
}

func TestEvidenceProcessExitCutPoints(t *testing.T) {
	_, load := kit(t)
	ta, w := load("close-issue")
	raw := rawBundle(t)
	for _, point := range []string{"sync:claim.json", "sync:closure.json", "sync:terminal.pending.json", "link:terminal.json", "remove:terminal.pending.json"} {
		t.Run(point, func(t *testing.T) {
			root := t.TempDir()
			taBytes, _ := json.Marshal(ta)
			wBytes, _ := json.Marshal(w)
			for name, data := range map[string][]byte{"task.json": taBytes, "witness.json": wBytes, "bundle.stb": raw} {
				if err := os.WriteFile(filepath.Join(root, name), data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestEvidenceCrashHelper$")
			cmd.Env = append(os.Environ(), "GOMAXPROCS=1", "STATETWIN_TEST_EVIDENCE_CUT="+point, "STATETWIN_TEST_EVIDENCE_ROOT="+root)
			output, err := cmd.CombinedOutput()
			var exit *exec.ExitError
			if !errors.As(err, &exit) || exit.ExitCode() != 67 {
				t.Fatalf("unexpected child completion: %v %s", err, output)
			}
			r, err := InspectDirectory(context.Background(), root, "trial")
			if err != nil {
				t.Fatal(err)
			}
			published := strings.HasPrefix(point, "link:") || strings.HasPrefix(point, "remove:")
			if r.EvidenceComplete != published || r.ResumeAllowed || r.State == "invalid" {
				t.Fatalf("cut %s: %+v", point, r)
			}
			started := 0
			if _, err = recordStorageCase(context.Background(), root, ta, raw, w, openRootedEvidenceFS, &started); err == nil || started != 0 {
				t.Fatal("crashed attempt resumed")
			}
		})
	}
}
