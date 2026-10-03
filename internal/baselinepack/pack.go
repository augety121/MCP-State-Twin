// Package baselinepack admits declaration-only, bounded evaluation packs.
package baselinepack

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/augety121/mcp-state-twin/internal/agenteval"
	"github.com/augety121/mcp-state-twin/internal/agenthost"
	"github.com/augety121/mcp-state-twin/internal/bundle"
	"github.com/augety121/mcp-state-twin/internal/canonical"
	"github.com/augety121/mcp-state-twin/internal/limits"
	"github.com/augety121/mcp-state-twin/internal/logging"
	"github.com/augety121/mcp-state-twin/internal/server"
	"github.com/augety121/mcp-state-twin/internal/task"
)

const Format = "statetwin.dev/baseline-pack/v1alpha1"
const ProfileFormat = "statetwin.dev/plugin-profile/v1alpha1"
const ProfileID = "local-stdio-tools-v1"
const InputLimit = 256 << 20

var label = regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}$`)

func Label(s string) bool { return label.MatchString(s) }

type Entry struct {
	ID            string `json:"id"`
	Root          string `json:"root"`
	Task          string `json:"task"`
	ReviewedTask  string `json:"reviewedTask"`
	ReviewedWorld string `json:"reviewedWorld"`
	Cases         string `json:"cases"`
	Family        string `json:"familyId"`
	Variant       string `json:"variantId"`
	Split         string `json:"split"`
	Disclosure    string `json:"disclosure"`
}
type Pack struct {
	Format               string  `json:"format"`
	ID                   string  `json:"id"`
	Revision             string  `json:"revision"`
	License              string  `json:"license"`
	RuntimeCompatibility string  `json:"runtimeCompatibility"`
	Profile              string  `json:"profile"`
	ResourceProfile      string  `json:"resourceProfile"`
	State                string  `json:"state"`
	ReviewIndependence   string  `json:"reviewIndependence"`
	Entries              []Entry `json:"entries"`
}
type Profile struct {
	Format           string `json:"format"`
	ID               string `json:"id"`
	Revision         string `json:"revision"`
	ProtocolProfile  string `json:"protocolProfile"`
	Transport        string `json:"transport"`
	Isolation        string `json:"isolation"`
	TaskProjection   string `json:"taskProjection"`
	LifecycleVersion string `json:"lifecycleVersion"`
	MaxInflight      int    `json:"maxInflight"`
	StartupSeconds   int    `json:"startupSeconds"`
}
type Input struct {
	Raw  []byte
	Info os.FileInfo
}

// Reader owns one operation's raw inputs; no process-wide cache or hashes.
type Reader struct {
	Root   string
	Left   int
	Inputs map[string]Input
	ctx    context.Context
}

func NewReader(ctx context.Context, root string) *Reader {
	return &Reader{root, InputLimit, map[string]Input{}, ctx}
}
func (r *Reader) Read(name string, limit int) ([]byte, error) {
	if r.ctx.Err() != nil {
		return nil, r.ctx.Err()
	}
	if task.PortablePath(name) != nil {
		return nil, errors.New("PLUGIN_PLAN_INVALID")
	}
	if in, ok := r.Inputs[name]; ok {
		if len(in.Raw) > limit {
			return nil, errors.New("PLUGIN_RESOURCE_LIMIT")
		}
		return in.Raw, nil
	}
	for other := range r.Inputs {
		if strings.EqualFold(other, name) {
			return nil, errors.New("PLUGIN_PLAN_INVALID")
		}
	}
	fs, err := os.OpenRoot(r.Root)
	if err != nil {
		return nil, errors.New("PLUGIN_INPUT_UNAVAILABLE")
	}
	defer fs.Close()
	info, err := fs.Lstat(name)
	if err != nil {
		return nil, errors.New("PLUGIN_INPUT_UNAVAILABLE")
	}
	if !info.Mode().IsRegular() || info.Size() > int64(min(limit, r.Left)) {
		return nil, errors.New("PLUGIN_RESOURCE_LIMIT")
	}
	for _, previous := range r.Inputs {
		if os.SameFile(previous.Info, info) {
			return nil, errors.New("PLUGIN_REFERENCE_MISMATCH")
		}
	}
	raw, err := task.ReadFile(r.Root, name, min(limit, r.Left))
	if err != nil {
		return nil, errors.New("PLUGIN_PLAN_INVALID")
	}
	if logging.ContainsSensitive(string(raw)) {
		return nil, errors.New("PLUGIN_PLAN_INVALID")
	}
	r.Left -= len(raw)
	r.Inputs[name] = Input{raw, info}
	return raw, nil
}
func (r *Reader) Separate(a, b string) bool {
	x, xok := r.Inputs[a]
	y, yok := r.Inputs[b]
	return xok && yok && a != b && !os.SameFile(x.Info, y.Info)
}
func (r *Reader) RejectOutput(out string) error {
	if task.PortablePath(out) != nil {
		return errors.New("PLUGIN_PLAN_INVALID")
	}
	for name := range r.Inputs {
		if strings.EqualFold(out, name) || strings.HasPrefix(strings.ToLower(name), strings.ToLower(out)+"/") {
			return errors.New("PLUGIN_PLAN_INVALID")
		}
	}
	return nil
}
func Decode(raw []byte, limit int, dst any) error {
	if logging.ContainsSensitive(string(raw)) || agenthost.DecodeDocument(raw, limit, dst) != nil || requiredFields(raw, dst) != nil {
		return errors.New("PLUGIN_PLAN_INVALID")
	}
	return nil
}
func Equal(a, b any) bool {
	x, e := canonical.JSON(a)
	y, f := canonical.JSON(b)
	return e == nil && f == nil && bytes.Equal(x, y)
}
func SameWorld(a, b *bundle.Artifact) bool {
	if a == nil || b == nil || !Equal(a.Manifest, b.Manifest) || len(a.Files) != len(b.Files) {
		return false
	}
	for n, v := range a.Files {
		if w, ok := b.Files[n]; !ok || !bytes.Equal(v, w) {
			return false
		}
	}
	return true
}

type PreparedEntry struct {
	Entry  Entry
	Task   *task.Task
	Bundle []byte
}
type Prepared struct {
	Pack    Pack
	Profile Profile
	Entries []PreparedEntry
	Reader  *Reader
}
type Check struct {
	Format             string `json:"format"`
	Status             string `json:"status"`
	PackID             string `json:"packId"`
	Revision           string `json:"revision"`
	Entries            int    `json:"entries"`
	Families           int    `json:"families"`
	ExecutionPerformed bool   `json:"executionPerformed"`
	SourceTrust        string `json:"sourceTrust"`
}

func (p *Prepared) Summary() Check {
	f := map[string]bool{}
	for _, e := range p.Entries {
		f[e.Entry.Family] = true
	}
	return Check{"statetwin.dev/plugin-check/v1alpha1", "statically-valid", p.Pack.ID, p.Pack.Revision, len(p.Entries), len(f), false, "local-operator-asserted"}
}
func (p *Prepared) Find(id string) *PreparedEntry {
	for i := range p.Entries {
		if p.Entries[i].Entry.ID == id {
			return &p.Entries[i]
		}
	}
	return nil
}

func Prepare(ctx context.Context, root, packName, profileName string) (*Prepared, error) {
	r := NewReader(ctx, root)
	return PrepareWith(r, packName, profileName)
}
func PrepareWith(r *Reader, packName, profileName string) (*Prepared, error) {
	p := &Prepared{Reader: r}
	raw, err := r.Read(packName, 256<<10)
	if err != nil {
		return nil, err
	}
	if Decode(raw, 256<<10, &p.Pack) != nil {
		return nil, errors.New("PLUGIN_PLAN_INVALID")
	}
	m := p.Pack
	if m.Format != Format || !Label(m.ID) || !Label(m.Revision) || m.License != "MIT" || m.RuntimeCompatibility != "go1.26-sdk1.8.0" || m.Profile != ProfileID || m.ResourceProfile != "baseline-plugin-v1" || (m.State != "draft" && m.State != "qualified" && m.State != "frozen") || (m.ReviewIndependence != "self-reviewed" && m.ReviewIndependence != "independent") || len(m.Entries) < 1 || len(m.Entries) > 72 {
		return nil, errors.New("PLUGIN_PLAN_INVALID")
	}
	raw, err = r.Read(profileName, 64<<10)
	if err != nil {
		return nil, err
	}
	if Decode(raw, 64<<10, &p.Profile) != nil {
		return nil, errors.New("PLUGIN_PLAN_INVALID")
	}
	f := p.Profile
	if f.Format != ProfileFormat || f.ID != ProfileID || f.Revision != "v1" || f.Transport != "stdio" || f.Isolation != "trusted-tools-only-v1" || f.TaskProjection != "blind-objective-v1" || f.LifecycleVersion != "v1" || f.MaxInflight != 4 || f.StartupSeconds != 10 || (f.ProtocolProfile != server.LegacyProtocolVersion && f.ProtocolProfile != server.ModernProtocolVersion) {
		return nil, errors.New("PLUGIN_HOST_UNSUPPORTED")
	}
	ids := map[string]bool{}
	taskPaths := map[string]bool{}
	worldPaths := map[string]bool{}
	refs := [][2]string{}
	// A case manifest's relative references are interpreted under its root.
	// Identical manifest paths reached through different roots are not the
	// same admission, even when the selected Task JSON happens to be equal.
	caseTasks := map[[2]string][]*task.Task{}
	bundles := map[string]*bundle.Artifact{}
	extracted := InputLimit
	open := func(name string) ([]byte, *bundle.Artifact, error) {
		data, e := r.Read(name, limits.MaxBundleCompressed)
		if e != nil {
			return nil, nil, e
		}
		if b := bundles[name]; b != nil {
			return data, b, nil
		}
		var b *bundle.Artifact
		for previous, admitted := range bundles {
			if bytes.Equal(data, r.Inputs[previous].Raw) {
				b = admitted
				break
			}
		}
		reused := b != nil
		if !reused {
			b, e = bundle.OpenBytes(data)
			if e != nil {
				return nil, nil, errors.New("PLUGIN_PLAN_INVALID")
			}
		}
		for _, v := range b.Files {
			// Debit each physical input even when decoding an exact byte copy
			// is reused. Paths, hard links and original bytes were checked above.
			if len(v) > extracted || (!reused && logging.ContainsSensitive(string(v))) {
				return nil, nil, errors.New("PLUGIN_RESOURCE_LIMIT")
			}
			extracted -= len(v)
		}
		bundles[name] = b
		return data, b, nil
	}
	for _, e := range m.Entries {
		if !Label(e.ID) || !Label(e.Family) || !Label(e.Variant) || ids[e.ID] || (e.Root != "." && task.PortablePath(e.Root) != nil) || task.PortablePath(e.Task) != nil || task.PortablePath(e.ReviewedTask) != nil || task.PortablePath(e.ReviewedWorld) != nil || task.PortablePath(e.Cases) != nil || (e.Split != "dev" && e.Split != "regression" && e.Split != "evaluation") || (e.Disclosure != "public" && e.Disclosure != "public-evaluation-split" && e.Disclosure != "private-holdout") || (e.Disclosure == "private-holdout" && e.Split != "evaluation") || (e.Disclosure == "public-evaluation-split" && e.Split != "evaluation") {
			return nil, errors.New("PLUGIN_PLAN_INVALID")
		}
		ids[e.ID] = true
		tn, rn := path.Join(e.Root, e.Task), path.Join(e.Root, e.ReviewedTask)
		data, err := r.Read(tn, task.MaxBytes)
		if err != nil {
			return nil, err
		}
		t, err := task.Decode(data)
		if err != nil {
			return nil, errors.New("PLUGIN_PLAN_INVALID")
		}
		data, err = r.Read(rn, task.MaxBytes)
		if err != nil {
			return nil, err
		}
		review, err := task.Decode(data)
		if err != nil || !Equal(t, review) || !r.Separate(tn, rn) {
			return nil, errors.New("PLUGIN_REFERENCE_MISMATCH")
		}
		if taskPaths[tn] {
			return nil, errors.New("PLUGIN_PLAN_INVALID")
		}
		taskPaths[tn] = true
		refs = append(refs, [2]string{tn, rn})
		bn, wn := path.Join(e.Root, t.Bundle), path.Join(e.Root, e.ReviewedWorld)
		data, b, err := open(bn)
		if err != nil {
			return nil, err
		}
		_, reviewed, err := open(wn)
		if err != nil || !r.Separate(bn, wn) || !SameWorld(b, reviewed) {
			return nil, errors.New("PLUGIN_REFERENCE_MISMATCH")
		}
		worldPaths[bn] = true
		refs = append(refs, [2]string{bn, wn})
		read := func(name string, max int) ([]byte, error) { return r.Read(path.Join(e.Root, name), max) }
		caseName := [2]string{e.Root, e.Cases}
		if caseTasks[caseName] == nil {
			caseTasks[caseName], err = agenteval.CheckPluginCaseTasks(r.ctx, e.Cases, read)
			if err != nil {
				return nil, err
			}
		}
		matched := false
		for _, candidate := range caseTasks[caseName] {
			if Equal(candidate, t) {
				matched = true
				break
			}
		}
		if !matched {
			return nil, errors.New("PLUGIN_QUALITY_REFERENCE_INVALID")
		}
		// CheckPluginCaseTasks already admitted this exact Task against the
		// same frozen root-relative Bundle, including oracle compilation and
		// data-policy checks. Do not compile and scan it a second time here.
		p.Entries = append(p.Entries, PreparedEntry{e, t, bytes.Clone(data)})
	}
	for _, ref := range refs {
		for actual := range taskPaths {
			if !r.Separate(actual, ref[1]) {
				return nil, errors.New("PLUGIN_REFERENCE_MISMATCH")
			}
		}
		for actual := range worldPaths {
			if !r.Separate(actual, ref[1]) {
				return nil, errors.New("PLUGIN_REFERENCE_MISMATCH")
			}
		}
	}
	return p, nil
}

// PublicProjection deliberately has no oracle, expected outcome or fault plan.
type PublicProjection struct {
	TaskID    string   `json:"taskId"`
	Objective string   `json:"objective"`
	Context   string   `json:"context"`
	Tools     []string `json:"tools"`
}

func (p *PreparedEntry) Projection() PublicProjection {
	return PublicProjection{p.Task.ID, p.Task.Objective, p.Task.Context, append([]string{}, p.Task.Tools...)}
}

func Clone[T any](v T) T { raw, _ := json.Marshal(v); var x T; _ = json.Unmarshal(raw, &x); return x }
