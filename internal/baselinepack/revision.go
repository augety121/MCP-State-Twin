package baselinepack

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path"
	"strings"
	"time"

	"github.com/augety121/mcp-state-twin/internal/task"
)

type Revision struct {
	Format             string            `json:"format"`
	PackID             string            `json:"packId"`
	Revision           string            `json:"revision"`
	Profile            Profile           `json:"profile"`
	Inputs             map[string][]byte `json:"inputs"`
	ReviewIndependence string            `json:"reviewIndependence"`
}
type RevisionEvent struct {
	Format          string   `json:"format"`
	State           string   `json:"state"`
	At              string   `json:"at"`
	Reason          string   `json:"reason"`
	Replacement     string   `json:"replacement"`
	PreviewVersions []string `json:"previewVersions"`
}

func revision(p *Prepared) Revision {
	r := Revision{"statetwin.dev/baseline-pack-revision/v1alpha1", p.Pack.ID, p.Pack.Revision, p.Profile, map[string][]byte{}, p.Pack.ReviewIndependence}
	for n, in := range p.Reader.Inputs {
		r.Inputs[n] = append([]byte(nil), in.Raw...)
	}
	return r
}

// Register freezes full raw inputs in a user-selected local registry. Exact
// duplicate registration is idempotent; a changed definition under the same
// id/revision fails. No hash catalog or signature/provenance claim is created.
func Register(ctx context.Context, root, pack, profile, registry string) (*Revision, error) {
	p, err := Prepare(ctx, root, pack, profile)
	if err != nil {
		return nil, err
	}
	if p.Reader.RejectOutput(registry) != nil {
		return nil, errors.New("BASELINE_REGISTRY_INVALID")
	}
	r := revision(p)
	raw, err := json.MarshalIndent(r, "", "  ")
	if err != nil || len(raw) > InputLimit {
		return nil, errors.New("BASELINE_RESOURCE_LIMIT")
	}
	fs, err := os.OpenRoot(root)
	if err != nil {
		return nil, err
	}
	defer fs.Close()
	dir := path.Join(registry, r.PackID, r.Revision)
	for i, part := range strings.Split(dir, "/") {
		_ = part
		prefix := strings.Join(strings.Split(dir, "/")[:i+1], "/")
		info, e := fs.Lstat(prefix)
		if os.IsNotExist(e) {
			if e = fs.Mkdir(prefix, 0700); e != nil {
				return nil, e
			}
		} else if e != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil, errors.New("BASELINE_REGISTRY_INVALID")
		}
	}
	name := path.Join(dir, "revision.json")
	old, e := task.ReadFile(root, name, InputLimit)
	if e == nil {
		var existing Revision
		if Decode(old, InputLimit, &existing) != nil || !Equal(existing, r) {
			return nil, errors.New("BASELINE_REVISION_CONFLICT")
		}
		return &r, nil
	}
	if !os.IsNotExist(e) {
		if _, e = fs.Stat(name); !os.IsNotExist(e) {
			return nil, errors.New("BASELINE_REVISION_CONFLICT")
		}
	}
	file, err := fs.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return nil, errors.New("BASELINE_REVISION_CONFLICT")
	}
	n, err := file.Write(raw)
	if err == nil && n != len(raw) {
		err = errors.New("BASELINE_OUTPUT_FAILED")
	}
	if err == nil {
		err = file.Sync()
	}
	closed := file.Close()
	if err != nil {
		return nil, err
	}
	if closed != nil {
		return nil, closed
	}
	return &r, nil
}

func ValidateRevisionEvent(e RevisionEvent) error {
	if e.Format != "statetwin.dev/baseline-pack-event/v1alpha1" || (e.State != "deprecated" && e.State != "revoked") || e.Reason == "" || len(e.Reason) > 1024 {
		return errors.New("BASELINE_REVISION_EVENT_INVALID")
	}
	if _, err := time.Parse(time.RFC3339Nano, e.At); err != nil {
		return errors.New("BASELINE_REVISION_EVENT_INVALID")
	}
	if e.State == "deprecated" {
		if !Label(e.Replacement) || len(e.PreviewVersions) < 2 || len(e.PreviewVersions) > 16 {
			return errors.New("BASELINE_REVISION_EVENT_INVALID")
		}
		seen := map[string]bool{}
		for _, v := range e.PreviewVersions {
			if !Label(v) || seen[v] {
				return errors.New("BASELINE_REVISION_EVENT_INVALID")
			}
			seen[v] = true
		}
	}
	return nil
}

// Transition appends a non-overwriting marker. Revocation always dominates
// deprecation; historical raw evidence remains available for read-only replay.
func Transition(root, registry, packID, revisionID string, event RevisionEvent) error {
	if task.PortablePath(registry) != nil || !Label(packID) || !Label(revisionID) || ValidateRevisionEvent(event) != nil {
		return errors.New("BASELINE_REVISION_EVENT_INVALID")
	}
	dir := path.Join(registry, packID, revisionID)
	raw, err := task.ReadFile(root, path.Join(dir, "revision.json"), InputLimit)
	if err != nil {
		return err
	}
	var r Revision
	if Decode(raw, InputLimit, &r) != nil || r.PackID != packID || r.Revision != revisionID {
		return errors.New("BASELINE_REVISION_CONFLICT")
	}
	fs, err := os.OpenRoot(root)
	if err != nil {
		return err
	}
	defer fs.Close()
	if event.State == "deprecated" {
		if _, e := fs.Stat(path.Join(dir, "revoked.json")); e == nil {
			return errors.New("BASELINE_REVISION_REVOKED")
		}
	}
	raw, err = json.MarshalIndent(event, "", "  ")
	if err != nil {
		return err
	}
	f, err := fs.OpenFile(path.Join(dir, event.State+".json"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return errors.New("BASELINE_REVISION_CONFLICT")
	}
	n, e := f.Write(raw)
	if e == nil && n != len(raw) {
		e = errors.New("BASELINE_OUTPUT_FAILED")
	}
	if e == nil {
		e = f.Sync()
	}
	closed := f.Close()
	if e != nil {
		return e
	}
	return closed
}

func RevisionState(root, registry, packID, revisionID string) (string, error) {
	if task.PortablePath(registry) != nil || !Label(packID) || !Label(revisionID) {
		return "", errors.New("BASELINE_REGISTRY_INVALID")
	}
	dir := path.Join(registry, packID, revisionID)
	if _, e := task.ReadFile(root, path.Join(dir, "revision.json"), InputLimit); e != nil {
		return "", e
	}
	fs, err := os.OpenRoot(root)
	if err != nil {
		return "", err
	}
	defer fs.Close()
	for _, state := range []string{"revoked", "deprecated"} {
		if _, e := fs.Lstat(path.Join(dir, state+".json")); e != nil {
			if os.IsNotExist(e) {
				continue
			}
			return "", errors.New("BASELINE_REVISION_EVENT_INVALID")
		}
		raw, e := task.ReadFile(root, path.Join(dir, state+".json"), 64<<10)
		if e != nil {
			return "", errors.New("BASELINE_REVISION_EVENT_INVALID")
		}
		if e == nil {
			var event RevisionEvent
			if Decode(raw, 64<<10, &event) != nil || ValidateRevisionEvent(event) != nil || event.State != state {
				return "", errors.New("BASELINE_REVISION_EVENT_INVALID")
			}
			return state, nil
		}
	}
	return "frozen", nil
}
