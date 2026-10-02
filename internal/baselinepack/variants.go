package baselinepack

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/augety121/mcp-state-twin/internal/bundle"
	"github.com/augety121/mcp-state-twin/internal/task"
)

// Variant parameters are an enumeration, not a templating or scripting language.
// Namespace substitution is applied to every admitted Task/oracle/witness and
// every member of both actual and reviewed worlds before rebuilding bundles.
func variantText(raw []byte, variant string) ([]byte, error) {
	var owner, publisher string
	switch variant {
	case "dev":
		return append([]byte(nil), raw...), nil
	case "regression":
		owner, publisher = "team-regression", "vendor-regression"
	case "evaluation":
		owner, publisher = "team-evaluation", "vendor-evaluation"
	default:
		return nil, errors.New("BASELINE_VARIANT_INVALID")
	}
	mapped := []byte(strings.NewReplacer("octo", owner, "acme", publisher).Replace(string(raw)))
	var object map[string]any
	if json.Unmarshal(mapped, &object) == nil && object != nil {
		suffix := "-" + variant
		appendID := func(m map[string]any, key string) {
			if id, ok := m[key].(string); ok {
				m[key] = id + suffix
			}
		}
		switch object["kind"] {
		case "AgentTask":
			appendID(object, "id")
			object["revision"] = "v2"
		case "TaskWitness":
			appendID(object, "taskId")
		}
		if format, ok := object["format"].(string); ok && strings.HasPrefix(format, "statetwin.dev/task-cases/") {
			for _, field := range []string{"tasks", "cases"} {
				if rows, ok := object[field].([]any); ok {
					for _, row := range rows {
						if m, ok := row.(map[string]any); ok {
							appendID(m, "taskId")
						}
					}
				}
			}
		}
		var err error
		mapped, err = json.MarshalIndent(object, "", "  ")
		if err != nil {
			return nil, err
		}
	}
	return mapped, nil
}

// GenerateVariants publishes a new, exclusively owned directory. Original
// assets remain untouched; a failure leaves partial output visibly incomplete.
func GenerateVariants(ctx context.Context, root, packName, profileName, output string) (*Check, error) {
	p, err := Prepare(ctx, root, packName, profileName)
	if err != nil {
		return nil, err
	}
	if len(p.Entries) != 24 || p.Summary().Families != 24 || p.Reader.RejectOutput(output) != nil {
		return nil, errors.New("BASELINE_VARIANT_INVALID")
	}
	for _, e := range p.Entries {
		if e.Entry.Variant != "original" || e.Entry.Split != "dev" || e.Entry.Disclosure != "public" {
			return nil, errors.New("BASELINE_VARIANT_INVALID")
		}
	}
	fs, err := os.OpenRoot(root)
	if err != nil {
		return nil, err
	}
	defer fs.Close()
	if task.PortablePath(output) != nil {
		return nil, errors.New("BASELINE_VARIANT_INVALID")
	}
	// Parent must exist and be free of symlinks. Do not create arbitrary ancestors.
	parts := strings.Split(output, "/")
	for i := 1; i < len(parts); i++ {
		info, e := fs.Lstat(strings.Join(parts[:i], "/"))
		if e != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil, errors.New("BASELINE_OUTPUT_INVALID")
		}
	}
	if fs.Mkdir(output, 0700) != nil {
		return nil, errors.New("BASELINE_OUTPUT_EXISTS")
	}
	dest, err := fs.OpenRoot(output)
	if err != nil {
		return nil, err
	}
	defer dest.Close()
	write := func(name string, raw []byte) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if task.PortablePath(name) != nil {
			return errors.New("BASELINE_OUTPUT_INVALID")
		}
		if err := dest.MkdirAll(path.Dir(name), 0700); err != nil {
			return err
		}
		f, e := dest.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if e != nil {
			return e
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
	names := make([]string, 0, len(p.Reader.Inputs))
	for n := range p.Reader.Inputs {
		if n != packName && n != profileName {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	result := p.Pack
	result.Revision = "v2"
	result.State = "draft"
	result.ReviewIndependence = "self-reviewed"
	result.Entries = nil
	for _, variant := range []string{"dev", "regression", "evaluation"} {
		for index, name := range names {
			input := p.Reader.Inputs[name].Raw
			if strings.HasSuffix(name, ".stb") {
				b, e := bundle.OpenBytes(input)
				if e != nil {
					return nil, e
				}
				b.Manifest.Digests = nil
				stage := path.Join(variant, "bundle-sources", "world-"+integer(index))
				manifest, e := json.Marshal(b.Manifest)
				if e != nil {
					return nil, e
				}
				if e = write(path.Join(stage, "source.json"), manifest); e != nil {
					return nil, e
				}
				for member, raw := range b.Files {
					if member == bundle.ManifestName {
						continue
					}
					mapped, e := variantText(raw, variant)
					if e != nil {
						return nil, e
					}
					if e = write(path.Join(stage, member), mapped); e != nil {
						return nil, e
					}
				}
				if e = dest.MkdirAll(path.Dir(path.Join(variant, name)), 0700); e != nil {
					return nil, e
				}
				// Both paths are below a directory exclusively claimed by this operation.
				if _, e = bundle.Build(filepath.Join(root, output, stage, "source.json"), filepath.Join(root, output, variant, name)); e != nil {
					return nil, e
				}
			} else {
				raw, e := variantText(input, variant)
				if e != nil {
					return nil, e
				}
				if e = write(path.Join(variant, name), raw); e != nil {
					return nil, e
				}
			}
		}
		for _, old := range p.Pack.Entries {
			e := old
			e.Root = path.Join(variant, old.Root)
			e.Split = variant
			if variant != "dev" {
				e.ID += "-" + variant
				e.Variant = "namespace-" + variant
			}
			if variant == "evaluation" {
				e.Disclosure = "public-evaluation-split"
			}
			result.Entries = append(result.Entries, e)
		}
	}
	raw, _ := json.MarshalIndent(p.Profile, "", "  ")
	if err = write("plugin-profile.json", raw); err != nil {
		return nil, err
	}
	raw, _ = json.MarshalIndent(result, "", "  ")
	if err = write("baseline-pack.json", raw); err != nil {
		return nil, err
	}
	checked, err := Prepare(ctx, filepath.Join(root, output), "baseline-pack.json", "plugin-profile.json")
	if err != nil {
		return nil, err
	}
	summary := checked.Summary()
	return &summary, nil
}

func integer(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}
