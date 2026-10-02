package agenteval

import (
	"context"
	"errors"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"github.com/augety121/mcp-state-twin/internal/limits"
	"github.com/augety121/mcp-state-twin/internal/task"
)

type RegistryEntry struct {
	ID  string `json:"id"`
	Out string `json:"out"`
}
type SuiteRegistry struct {
	Format  string          `json:"format"`
	Entries []RegistryEntry `json:"entries"`
}
type InventoryEntry struct {
	ID                 string `json:"id"`
	ObservationState   string `json:"observationState"`
	AuditState         string `json:"auditState"`
	ReportVerification string `json:"reportVerification"`
	ComparisonDecision string `json:"comparisonDecision"`
	ObservedFiles      int    `json:"observedFiles"`
	ObservedBytes      int64  `json:"observedBytes"`
	SizeComplete       bool   `json:"sizeComplete"`
	Problem            string `json:"problem"`
}
type SuiteInventory struct {
	Format          string           `json:"format"`
	Mode            string           `json:"mode"`
	Completion      string           `json:"completion"`
	PlannedEntries  int              `json:"plannedEntries"`
	ObservedEntries int              `json:"observedEntries"`
	Entries         []InventoryEntry `json:"entries"`
	Provenance      string           `json:"provenance"`
}

func loadRegistry(root, name string) (*SuiteRegistry, error) {
	raw, err := task.ReadFile(root, name, 64<<10)
	if err != nil {
		return nil, errors.New("INV_INPUT_INVALID")
	}
	var r SuiteRegistry
	if decodeSuiteMetadata(raw, 64<<10, &r) != nil || r.Format != "statetwin.dev/suite-registry/v1alpha1" || len(r.Entries) < 1 || len(r.Entries) > 16 {
		return nil, errors.New("INV_INPUT_INVALID")
	}
	for i, e := range r.Entries {
		if !validLabel(e.ID) || task.PortablePath(e.Out) != nil {
			return nil, errors.New("INV_INPUT_INVALID")
		}
		for _, prev := range r.Entries[:i] {
			if prev.ID == e.ID || beneath(e.Out, prev.Out) || beneath(prev.Out, e.Out) {
				return nil, errors.New("INV_INPUT_INVALID")
			}
		}
	}
	return &r, nil
}

type inventoryBudget struct {
	bytes     int64
	entries   int
	exhausted bool
}
type budgetReadRoot struct {
	evidenceReadRoot
	budget *inventoryBudget
}
type budgetReadFile struct {
	evidenceReadFile
	budget *inventoryBudget
}

func (r budgetReadRoot) Open(n string) (evidenceReadFile, error) {
	f, err := r.evidenceReadRoot.Open(n)
	if err != nil {
		return nil, err
	}
	return &budgetReadFile{f, r.budget}, nil
}
func (f *budgetReadFile) Read(p []byte) (int, error) {
	if f.budget.exhausted {
		return 0, errors.New("INV_RESOURCE_LIMIT")
	}
	if int64(len(p)) > f.budget.bytes+1 {
		p = p[:f.budget.bytes+1]
	}
	n, err := f.evidenceReadFile.Read(p)
	f.budget.bytes -= int64(n)
	if f.budget.bytes < 0 {
		f.budget.exhausted = true
		return n, errors.New("INV_RESOURCE_LIMIT")
	}
	return n, err
}
func (f *budgetReadFile) ReadDir(n int) ([]os.DirEntry, error) {
	if f.budget.exhausted {
		return nil, errors.New("INV_RESOURCE_LIMIT")
	}
	if n <= 0 || n > f.budget.entries+1 {
		n = f.budget.entries + 1
	}
	e, err := f.evidenceReadFile.ReadDir(n)
	f.budget.entries -= len(e)
	if f.budget.entries < 0 {
		f.budget.exhausted = true
		return nil, errors.New("INV_RESOURCE_LIMIT")
	}
	return e, err
}
func InventorySuites(parent context.Context, root, registry, mode string) (*SuiteInventory, error) {
	if !oneOf(mode, "metadata", "replay") {
		return nil, errors.New("INV_ARGUMENTS_INVALID")
	}
	ctx, cancel := context.WithTimeout(parent, 120*time.Second)
	defer cancel()
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	reg, err := loadRegistry(root, registry)
	if err != nil {
		return nil, err
	}
	return inventoryRegistry(ctx, root, reg, mode)
}
func inventoryRegistry(ctx context.Context, root string, reg *SuiteRegistry, mode string) (*SuiteInventory, error) {
	r := &SuiteInventory{Format: "statetwin.dev/suite-inventory/v1alpha1", Mode: mode, Completion: "complete", PlannedEntries: len(reg.Entries), Entries: []InventoryEntry{}, Provenance: "not-proven"}
	for _, e := range reg.Entries {
		r.Entries = append(r.Entries, InventoryEntry{ID: e.ID, ObservationState: "not_checked", ReportVerification: "not_checked"})
	}
	fs, err := os.OpenRoot(root)
	if err != nil {
		return nil, errors.New("INV_INPUT_INVALID")
	}
	defer fs.Close()
	budget := &inventoryBudget{bytes: 256 << 20, entries: 4096}
	view := budgetReadRoot{diskReadRoot{fs}, budget}
	for i, e := range reg.Entries {
		if ctx.Err() != nil {
			r.Completion = "incomplete"
			return r, ctx.Err()
		}
		row := &r.Entries[i]
		*row = inventoryMetadata(ctx, view, e)
		if mode == "replay" && row.ObservationState == "observed" {
			audit, err := inspectSuiteView(ctx, view, e.Out, maxSuiteWriteBytes, nil, nil)
			if err != nil {
				row.ObservationState = "unreadable"
				row.Problem = "audit_unavailable"
			} else {
				row.AuditState = audit.State
				row.ReportVerification = audit.ReportVerification
				if audit.Comparison != nil {
					row.ComparisonDecision = audit.Comparison.Decision
				}
				if audit.State == "invalid" {
					row.ObservationState = "invalid"
					row.Problem = "audit_invalid"
				}
			}
		}
		if budget.exhausted || ctx.Err() != nil {
			r.Completion = "incomplete"
			row.Problem = "observation_incomplete"
			row.SizeComplete = false
			if ctx.Err() != nil {
				return r, ctx.Err()
			}
			return r, errors.New("INV_RESOURCE_LIMIT")
		}
		r.ObservedEntries++
	}
	return r, nil
}
func inventoryMetadata(ctx context.Context, fs evidenceReadRoot, e RegistryEntry) InventoryEntry {
	r := InventoryEntry{ID: e.ID, ObservationState: "observed", ReportVerification: "not_checked"}
	bad := func(code string) InventoryEntry { r.ObservationState = "invalid"; r.Problem = code; return r }
	parts := strings.Split(e.Out, "/")
	for i := range parts {
		info, err := fs.Lstat(strings.Join(parts[:i+1], "/"))
		if os.IsNotExist(err) {
			r.ObservationState = "missing"
			r.Problem = "directory_missing"
			return r
		}
		if err != nil {
			r.ObservationState = "unreadable"
			r.Problem = "directory_unreadable"
			return r
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return bad("unsafe_directory")
		}
	}
	entries, err := suiteEntries(fs, e.Out, 37)
	if err != nil {
		r.ObservationState = "unreadable"
		r.Problem = "directory_unreadable"
		return r
	}
	if len(entries) > 36 {
		return bad("unexpected_members")
	}
	meta := map[string]int{"claim.json": MaxSuitePlanBytes, "plan.json": MaxSuitePlanBytes, "report.json": maxSuiteReportBytes, "report.pending.json": maxSuiteReportBytes}
	present := map[string]bool{}
	trialDirs := []string{}
	for _, entry := range entries {
		if ctx.Err() != nil {
			return r
		}
		name := entry.Name()
		info, err := fs.Lstat(path.Join(e.Out, name))
		if err != nil {
			r.ObservationState = "unreadable"
			r.Problem = "member_unreadable"
			return r
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return bad("unsafe_member")
		}
		if limit, ok := meta[name]; ok {
			if !info.Mode().IsRegular() || info.Size() > int64(limit) {
				return bad("unsafe_member")
			}
			r.ObservedFiles++
			r.ObservedBytes += info.Size()
			present[name] = true
		} else if info.IsDir() {
			trialDirs = append(trialDirs, name)
		} else {
			return bad("unexpected_member")
		}
	}
	if !present["claim.json"] {
		if len(entries) > 0 {
			return bad("claim_missing")
		}
		r.SizeComplete = true
		r.Problem = "metadata_partial"
		return r
	}
	raw, err := readArtifact(fs, path.Join(e.Out, "claim.json"), MaxSuitePlanBytes)
	if err != nil {
		return bad("claim_unreadable")
	}
	claim, err := decodeSuiteClaim(raw)
	if err != nil {
		return bad("claim_invalid")
	}
	allowed := map[string]bool{}
	for _, p := range claim.Plan.Pairs {
		allowed[p.Baseline.TrialID] = true
		allowed[p.Candidate.TrialID] = true
	}
	complete := present["plan.json"] && present["report.json"] && !present["report.pending.json"] && len(trialDirs) == len(allowed)
	for _, dir := range trialDirs {
		if !allowed[dir] {
			return bad("unexpected_trial")
		}
		members, err := suiteEntries(fs, path.Join(e.Out, dir), 5)
		if err != nil {
			r.ObservationState = "unreadable"
			r.Problem = "trial_unreadable"
			return r
		}
		found := map[string]bool{}
		for _, member := range members {
			name := member.Name()
			if !oneOf(name, artifactNames...) {
				return bad("unexpected_member")
			}
			info, err := fs.Lstat(path.Join(e.Out, dir, name))
			if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() > limits.MaxReportBytes {
				return bad("unsafe_member")
			}
			r.ObservedFiles++
			r.ObservedBytes += info.Size()
			found[name] = true
		}
		complete = complete && found["claim.json"] && found["terminal.json"] && !found["closure.json"] && !found["terminal.pending.json"]
	}
	if present["plan.json"] {
		raw, err := readArtifact(fs, path.Join(e.Out, "plan.json"), MaxSuitePlanBytes)
		var p ComparePlan
		if err != nil || decodeSuiteMetadata(raw, MaxSuitePlanBytes, &p) != nil || !same(p, claim.Plan) {
			return bad("plan_invalid")
		}
	}
	for _, name := range []string{"report.json", "report.pending.json"} {
		if present[name] {
			raw, err := readArtifact(fs, path.Join(e.Out, name), maxSuiteReportBytes)
			if err != nil {
				return bad("report_unreadable")
			}
			if _, err := decodeSuiteReport(raw, &claim.Plan); err != nil {
				return bad("report_invalid")
			}
		}
	}
	r.SizeComplete = true
	r.Problem = "metadata_present"
	if !complete {
		r.Problem = "metadata_partial"
	}
	return r
}

var _ io.Reader = (*budgetReadFile)(nil)
