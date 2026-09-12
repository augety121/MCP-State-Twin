package agenteval

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path"
	"strings"

	"github.com/augety121/mcp-state-twin/internal/limits"
	"github.com/augety121/mcp-state-twin/internal/logging"
	"github.com/augety121/mcp-state-twin/internal/task"
)

// Narrow, instance-local seams allow storage failures to be tested without
// process-global hooks, replacing the production filesystem, or filling a disk.
type evidenceFile interface {
	Write([]byte) (int, error)
	Sync() error
	Close() error
}
type evidenceFS interface {
	Lstat(string) (os.FileInfo, error)
	Mkdir(string, os.FileMode) error
	CreateExclusive(string) (evidenceFile, error)
	Link(string, string) error
	Remove(string) error
	Close() error
}
type rootedEvidenceFS struct{ *os.Root }

func (f rootedEvidenceFS) CreateExclusive(name string) (evidenceFile, error) {
	return f.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
}

type openEvidenceFS func(string) (evidenceFS, error)

func openRootedEvidenceFS(root string) (evidenceFS, error) {
	f, err := os.OpenRoot(root)
	if err != nil {
		return nil, err
	}
	return rootedEvidenceFS{f}, nil
}

type evidenceWriter struct {
	fs  evidenceFS
	out string
}

func prepareEvidence(value any) ([]byte, error) {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil || len(raw) > limits.MaxReportBytes {
		return nil, errors.New("EVIDENCE_WRITE_FAILED")
	}
	if logging.ContainsSensitive(string(raw)) {
		return nil, errors.New("DATA_POLICY_REJECTED")
	}
	return raw, nil
}

func claimEvidence(root, out string, claim any, open openEvidenceFS) (*evidenceWriter, error) {
	if err := task.PortablePath(out); err != nil {
		return nil, err
	}
	raw, err := prepareEvidence(claim)
	if err != nil {
		return nil, err
	}
	fs, err := open(root)
	if err != nil {
		return nil, errors.New("EVIDENCE_WRITE_FAILED")
	}
	w := &evidenceWriter{fs: fs, out: out}
	accepted := false
	defer func() {
		if !accepted {
			_ = fs.Close()
		}
	}()
	parent := path.Dir(out)
	if parent != "." {
		prefix := ""
		for _, component := range strings.Split(parent, "/") {
			prefix = path.Join(prefix, component)
			info, err := fs.Lstat(prefix)
			if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
				return nil, errors.New("EVIDENCE_WRITE_FAILED")
			}
		}
	}
	if err = fs.Mkdir(out, 0700); err != nil {
		return nil, errors.New("EVIDENCE_ALREADY_EXISTS_OR_UNWRITABLE")
	}
	if err = w.writeBytes("claim.json", raw); err != nil {
		return nil, err
	}
	accepted = true
	return w, nil
}

func (w *evidenceWriter) write(name string, value any) error {
	raw, err := prepareEvidence(value)
	if err != nil {
		return err
	}
	return w.writeBytes(name, raw)
}
func (w *evidenceWriter) writeBytes(name string, raw []byte) error {
	if name != "claim.json" && name != "closure.json" && name != "terminal.pending.json" {
		return errors.New("EVIDENCE_WRITE_FAILED")
	}
	f, err := w.fs.CreateExclusive(path.Join(w.out, name))
	if err != nil {
		return errors.New("EVIDENCE_WRITE_FAILED")
	}
	n, err := f.Write(raw)
	if err == nil && n != len(raw) {
		err = io.ErrShortWrite
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil || closeErr != nil {
		return errors.New("EVIDENCE_WRITE_FAILED")
	}
	return nil
}

func (w *evidenceWriter) publish() error {
	// No rename/overwrite or copy fallback. Failure preserves owned staging.
	if err := w.fs.Link(path.Join(w.out, "terminal.pending.json"), path.Join(w.out, "terminal.json")); err != nil {
		return errors.New("EVIDENCE_PUBLISH_UNSUPPORTED_OR_FAILED")
	}
	for _, name := range []string{"terminal.pending.json", "closure.json"} {
		if err := w.fs.Remove(path.Join(w.out, name)); err != nil {
			return errors.New("EVIDENCE_STAGING_CLEANUP_FAILED")
		}
	}
	return nil
}
