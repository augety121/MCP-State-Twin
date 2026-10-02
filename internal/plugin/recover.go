package plugin

import (
	"context"
	"errors"
	"os"
	"path"
	"strings"

	"github.com/augety121/mcp-state-twin/internal/baselinepack"
	"github.com/augety121/mcp-state-twin/internal/limits"
)

type Recovery struct {
	Format           string `json:"format"`
	State            string `json:"state"`
	OriginalDecision string `json:"originalDecision"`
	Resumable        bool   `json:"resumable"`
	Files            int    `json:"files"`
}

// Recover is copy-to-new only. It preserves partial bytes and never repairs a
// grade, retries an interaction or changes the original output directory.
func Recover(ctx context.Context, root, name, destination string) (*Recovery, error) {
	p, err := Prepare(ctx, root, name)
	if err != nil {
		return nil, err
	}
	if p.Pack.Reader.RejectOutput(destination) != nil || strings.EqualFold(destination, p.Plan.Output) || strings.HasPrefix(strings.ToLower(destination), strings.ToLower(p.Plan.Output)+"/") {
		return nil, errors.New("PLUGIN_RECOVERY_INVALID")
	}
	r := baselinepack.NewReader(ctx, root)
	frozen := map[string][]byte{}
	for n, in := range p.Pack.Reader.Inputs {
		frozen[n] = in.Raw
	}
	fs, err := os.OpenRoot(root)
	if err != nil {
		return nil, err
	}
	defer fs.Close()
	info, err := fs.Lstat(p.Plan.Output)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("PLUGIN_RECOVERY_INVALID")
	}
	d, err := fs.Open(p.Plan.Output)
	if err != nil {
		return nil, err
	}
	entries, err := d.ReadDir(7)
	d.Close()
	if err != nil && len(entries) == 0 || len(entries) > 5 {
		return nil, errors.New("PLUGIN_RECOVERY_INVALID")
	}
	allowed := map[string]bool{"claim.json": true, "terminal.json": true, "session-report.json": true, "terminal.json.pending": true, "session-report.json.pending": true}
	for _, entry := range entries {
		if !allowed[entry.Name()] || !entry.Type().IsRegular() {
			return nil, errors.New("PLUGIN_RECOVERY_INVALID")
		}
		n := path.Join(p.Plan.Output, entry.Name())
		raw, e := r.Read(n, limits.MaxReportBytes)
		if e != nil {
			return nil, e
		}
		frozen[n] = raw
	}
	report, _ := Inspect(ctx, root, name)
	decision := "invalid"
	if report != nil {
		decision = report.Decision
	}
	parts := strings.Split(destination, "/")
	for i := 1; i < len(parts); i++ {
		info, e := fs.Lstat(strings.Join(parts[:i], "/"))
		if e != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil, errors.New("PLUGIN_RECOVERY_INVALID")
		}
	}
	if fs.Mkdir(destination, 0700) != nil {
		return nil, errors.New("PLUGIN_OUTPUT_EXISTS")
	}
	dest, err := fs.OpenRoot(destination)
	if err != nil {
		return nil, err
	}
	defer dest.Close()
	s := &store{root: dest, out: "."}
	for n, raw := range frozen {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if dest.MkdirAll(path.Dir(n), 0700) != nil {
			return nil, errors.New("PLUGIN_OUTPUT_FAILED")
		}
		f, e := dest.OpenFile(n, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if e != nil {
			return nil, errors.New("PLUGIN_OUTPUT_FAILED")
		}
		count, e := f.Write(raw)
		if e == nil && count != len(raw) {
			e = errors.New("PLUGIN_OUTPUT_FAILED")
		}
		if e == nil {
			e = f.Sync()
		}
		closed := f.Close()
		if e != nil || closed != nil {
			return nil, errors.New("PLUGIN_OUTPUT_FAILED")
		}
	}
	result := &Recovery{"statetwin.dev/plugin-recovery/v1alpha1", "copied-to-new", decision, false, len(frozen)}
	if err = s.write("recovery.json", result, 64<<10); err != nil {
		return nil, err
	}
	return result, nil
}
