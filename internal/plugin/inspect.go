package plugin

import (
	"context"
	"errors"
	"os"
	"path"
	"strings"

	"github.com/augety121/mcp-state-twin/internal/agenteval"
	"github.com/augety121/mcp-state-twin/internal/baselinepack"
	"github.com/augety121/mcp-state-twin/internal/limits"
)

func Inspect(ctx context.Context, root, planName string) (*Report, error) {
	p, err := Prepare(ctx, root, planName)
	if err != nil {
		return nil, err
	}
	r := &Report{Format: ReportFormat, Plan: p.Plan, PackID: p.Pack.Pack.ID, PackRevision: p.Pack.Pack.Revision, TaskID: p.Entry.Task.ID, Lifecycle: "partial", Execution: "unknown", Evidence: "partial", Cleanup: "unknown", Decision: "invalid", FailureCode: "PLUGIN_EVIDENCE_INVALID", SourceTrust: "local-operator-asserted", HostObservation: "contract-test"}
	fs, err := os.OpenRoot(root)
	if err != nil {
		return r, errors.New("PLUGIN_EVIDENCE_INVALID")
	}
	defer fs.Close()
	parts := strings.Split(p.Plan.Output, "/")
	for i := range parts {
		info, e := fs.Lstat(strings.Join(parts[:i+1], "/"))
		if e != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return r, errors.New("PLUGIN_EVIDENCE_INVALID")
		}
	}
	dir, err := fs.Open(p.Plan.Output)
	if err != nil {
		return r, errors.New("PLUGIN_EVIDENCE_INVALID")
	}
	entries, err := dir.ReadDir(7)
	dir.Close()
	if err != nil && len(entries) == 0 {
		return r, errors.New("PLUGIN_EVIDENCE_INVALID")
	}
	allowed := map[string]bool{"claim.json": true, "terminal.json": true, "session-report.json": true, "terminal.json.pending": true, "session-report.json.pending": true}
	residue := false
	for _, e := range entries {
		if !allowed[e.Name()] || !e.Type().IsRegular() {
			return r, errors.New("PLUGIN_EVIDENCE_INVALID")
		}
		if strings.HasSuffix(e.Name(), ".pending") {
			residue = true
		}
	}
	reader := baselinepack.NewReader(ctx, root)
	raw, err := reader.Read(path.Join(p.Plan.Output, "claim.json"), 64<<10)
	if err != nil {
		return r, errors.New("PLUGIN_EVIDENCE_INVALID")
	}
	var claim Plan
	if baselinepack.Decode(raw, 64<<10, &claim) != nil || !baselinepack.Equal(claim, p.Plan) {
		return r, errors.New("PLUGIN_EVIDENCE_INVALID")
	}
	raw, err = reader.Read(path.Join(p.Plan.Output, "terminal.json"), limits.MaxReportBytes)
	if err != nil {
		return r, errors.New("PLUGIN_EVIDENCE_INVALID")
	}
	terminal, err := agenteval.DecodePluginTerminal(raw)
	if err != nil {
		return r, err
	}
	verified := agenteval.VerifyPluginTerminal(ctx, terminal, p.Entry.Task, p.Entry.Bundle) == nil
	raw, err = reader.Read(path.Join(p.Plan.Output, "session-report.json"), 1<<20)
	if err != nil {
		return r, errors.New("PLUGIN_EVIDENCE_INVALID")
	}
	var stored Report
	if baselinepack.Decode(raw, 1<<20, &stored) != nil || !baselinepack.Equal(&stored, reportFor(p, terminal, verified)) {
		return r, errors.New("PLUGIN_EVIDENCE_INVALID")
	}
	for a := range reader.Inputs {
		for b := range reader.Inputs {
			if a != b && !reader.Separate(a, b) {
				return r, errors.New("PLUGIN_EVIDENCE_INVALID")
			}
		}
	}
	if residue {
		stored.Lifecycle = "published_with_residue"
		stored.Decision = "invalid"
		stored.FailureCode = "PLUGIN_OUTPUT_RESIDUE"
		return &stored, errors.New("PLUGIN_OUTPUT_RESIDUE")
	}
	if !verified {
		return &stored, errors.New("PLUGIN_EVIDENCE_INVALID")
	}
	return &stored, nil
}
