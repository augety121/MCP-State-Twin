package agenteval

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/augety121/mcp-state-twin/internal/limits"
	"github.com/augety121/mcp-state-twin/internal/task"
)

const archiveFormat = "statetwin.dev/offline-suite-archive/v1alpha1"
const maxArchiveBytes = 129 << 20

type archiveMember struct {
	Path  string `json:"path"`
	Bytes int    `json:"bytes"`
}
type archiveManifest struct {
	Format  string          `json:"format"`
	Profile string          `json:"profile"`
	Members []archiveMember `json:"members"`
}
type SuiteExportResult struct {
	Format      string `json:"format"`
	State       string `json:"state"`
	Files       int    `json:"files"`
	Bytes       int64  `json:"bytes"`
	FailureCode string `json:"failureCode"`
	Provenance  string `json:"provenance"`
}
type SuiteImportResult struct {
	Format       string `json:"format"`
	State        string `json:"state"`
	WrittenFiles int    `json:"writtenFiles"`
	WrittenBytes int64  `json:"writtenBytes"`
	FailureCode  string `json:"failureCode"`
	Provenance   string `json:"provenance"`
}

func archiveEligible(ctx context.Context, files frozenFiles) error {
	incompatible := false
	r, err := inspectSuiteViewVerified(ctx, files, "suite", maxSuiteWriteBytes, nil, nil, func(ctx context.Context, e *AgentEvidence, terminal bool) error {
		err := replay(ctx, e, terminal)
		if err != nil && err.Error() == "EVIDENCE_INCOMPATIBLE" {
			incompatible = true
		}
		return err
	})
	if incompatible {
		return errors.New("EVIDENCE_INCOMPATIBLE")
	}
	if err != nil {
		return err
	}
	if r.State != "published" || r.ReportVerification != "matched" || r.StagingResidue || r.CompleteTrials != r.PlannedTrials {
		return errors.New("EXPORT_NOT_ELIGIBLE")
	}
	return ctx.Err()
}
func freezeSuite(ctx context.Context, root, out string) (frozenFiles, error) {
	if task.PortablePath(out) != nil {
		return nil, errors.New("EXPORT_INPUT_INVALID")
	}
	disk, err := os.OpenRoot(root)
	if err != nil {
		return nil, errors.New("EXPORT_INPUT_INVALID")
	}
	defer disk.Close()
	fs := diskReadRoot{disk}
	parts := strings.Split(out, "/")
	for i := range parts {
		info, err := fs.Lstat(strings.Join(parts[:i+1], "/"))
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil, errors.New("EXPORT_INPUT_INVALID")
		}
	}
	files := frozenFiles{}
	remaining := maxSuiteWriteBytes
	read := func(name string, limit int) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		info, err := fs.Lstat(path.Join(out, name))
		if err == nil && info.Mode().IsRegular() && info.Size() > int64(min(limit, remaining)) {
			return errors.New("EXPORT_RESOURCE_LIMIT")
		}
		raw, err := readArtifact(fs, path.Join(out, name), min(limit, remaining))
		if err != nil {
			return errors.New("EXPORT_INPUT_INVALID")
		}
		remaining -= len(raw)
		files[path.Join("suite", name)] = raw
		return nil
	}
	entries, err := suiteEntries(fs, out, 36)
	if err != nil {
		return nil, err
	}
	if err := read("claim.json", MaxSuitePlanBytes); err != nil {
		return nil, err
	}
	claim, err := decodeSuiteClaim(files["suite/claim.json"])
	if err != nil {
		return nil, errors.New("EXPORT_INPUT_INVALID")
	}
	dirs := map[string]bool{}
	for _, p := range claim.Plan.Pairs {
		dirs[p.Baseline.TrialID] = true
		dirs[p.Candidate.TrialID] = true
	}
	if len(entries) != 3+len(dirs) {
		return nil, errors.New("EXPORT_NOT_ELIGIBLE")
	}
	for _, e := range entries {
		n := e.Name()
		switch n {
		case "claim.json":
			continue
		case "plan.json", "report.json":
			limit := MaxSuitePlanBytes
			if n == "report.json" {
				limit = maxSuiteReportBytes
			}
			if err := read(n, limit); err != nil {
				return nil, err
			}
		default:
			if !dirs[n] {
				return nil, errors.New("EXPORT_NOT_ELIGIBLE")
			}
			info, err := fs.Lstat(path.Join(out, n))
			if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
				return nil, errors.New("EXPORT_INPUT_INVALID")
			}
			children, err := suiteEntries(fs, path.Join(out, n), 3)
			if err != nil || len(children) != 2 {
				return nil, errors.New("EXPORT_NOT_ELIGIBLE")
			}
			for _, child := range children {
				if !oneOf(child.Name(), "claim.json", "terminal.json") {
					return nil, errors.New("EXPORT_NOT_ELIGIBLE")
				}
				limit := limits.MaxReportBytes
				if child.Name() == "claim.json" {
					limit = 16 << 10
				}
				if err := read(path.Join(n, child.Name()), limit); err != nil {
					return nil, err
				}
			}
		}
	}
	if err := archiveEligible(ctx, files); err != nil {
		return nil, err
	}
	return files, nil
}
func archiveNames(files frozenFiles) []string {
	names := make([]string, 0, len(files))
	for n := range files {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
func encodeArchive(ctx context.Context, w io.Writer, files frozenFiles) error {
	names := archiveNames(files)
	m := archiveManifest{Format: archiveFormat, Profile: "offline-suite-copy-v1", Members: []archiveMember{}}
	for _, n := range names {
		m.Members = append(m.Members, archiveMember{Path: "payload/" + strings.TrimPrefix(n, "suite/"), Bytes: len(files[n])})
	}
	raw, err := json.Marshal(m)
	if err != nil || len(raw) > 64<<10 {
		return errors.New("EXPORT_RESOURCE_LIMIT")
	}
	tw := tar.NewWriter(w)
	write := func(name string, b []byte) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		h := &tar.Header{Name: name, Mode: 0600, Size: int64(len(b)), Typeflag: tar.TypeReg, ModTime: time.Unix(0, 0), Format: tar.FormatUSTAR}
		if err := tw.WriteHeader(h); err != nil {
			return err
		}
		n, err := tw.Write(b)
		if err == nil && n != len(b) {
			err = io.ErrShortWrite
		}
		return err
	}
	if err := write("manifest.json", raw); err != nil {
		return err
	}
	for _, n := range names {
		if err := write("payload/"+strings.TrimPrefix(n, "suite/"), files[n]); err != nil {
			return err
		}
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return tw.Close()
}
func decodeArchive(ctx context.Context, raw []byte) (frozenFiles, error) {
	invalid := errors.New("IMPORT_INPUT_INVALID")
	if len(raw) > maxArchiveBytes {
		return nil, errors.New("IMPORT_RESOURCE_LIMIT")
	}
	if len(raw)%512 != 0 {
		return nil, invalid
	}
	br := bytes.NewReader(raw)
	tr := tar.NewReader(br)
	files := frozenFiles{}
	var m archiveManifest
	names := map[string]bool{}
	remaining := maxSuiteWriteBytes
	count := 0
	lastEnd := 0
	actualNames := []string{}
	for {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, invalid
		}
		count++
		if count > 68 || h.Format != tar.FormatUSTAR || h.Typeflag != tar.TypeReg || h.Mode != 0600 || h.Uid != 0 || h.Gid != 0 || h.Uname != "" || h.Gname != "" || h.Linkname != "" || !h.ModTime.Equal(time.Unix(0, 0)) || len(h.PAXRecords) > 0 || task.PortablePath(h.Name) != nil || names[strings.ToLower(h.Name)] {
			return nil, invalid
		}
		names[strings.ToLower(h.Name)] = true
		limit := limits.MaxReportBytes
		if count == 1 {
			if h.Name != "manifest.json" {
				return nil, invalid
			}
			limit = 64 << 10
		} else if !strings.HasPrefix(h.Name, "payload/") {
			return nil, invalid
		}
		if h.Size > int64(limit) || h.Size > int64(remaining) {
			return nil, errors.New("IMPORT_RESOURCE_LIMIT")
		}
		if h.Size < 0 {
			return nil, invalid
		}
		start := len(raw) - br.Len()
		b, err := io.ReadAll(io.LimitReader(tr, h.Size+1))
		if err != nil || int64(len(b)) != h.Size {
			return nil, invalid
		}
		end := start + len(b)
		lastEnd = (end + 511) / 512 * 512
		if lastEnd > len(raw) {
			return nil, invalid
		}
		for _, v := range raw[end:lastEnd] {
			if v != 0 {
				return nil, invalid
			}
		}
		if count == 1 {
			if decodeSuiteMetadata(b, 64<<10, &m) != nil || m.Format != archiveFormat || m.Profile != "offline-suite-copy-v1" || len(m.Members) < 1 || len(m.Members) > 67 {
				return nil, invalid
			}
		} else {
			remaining -= len(b)
			files["suite/"+strings.TrimPrefix(h.Name, "payload/")] = b
			actualNames = append(actualNames, h.Name)
		}
	}
	if count != len(m.Members)+1 || len(raw) != lastEnd+1024 {
		return nil, invalid
	}
	for _, v := range raw[lastEnd:] {
		if v != 0 {
			return nil, invalid
		}
	}
	expected := archiveNames(files)
	if len(expected) != len(m.Members) {
		return nil, invalid
	}
	for i, n := range expected {
		member := m.Members[i]
		if member.Path != "payload/"+strings.TrimPrefix(n, "suite/") || member.Bytes != len(files[n]) || actualNames[i] != member.Path {
			return nil, invalid
		}
	}
	// The immutable view must contain exactly a clean suite; no metadata from the
	// container is placed inside the restored suite.
	if err := archiveEligible(ctx, files); err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if err.Error() == "EVIDENCE_INCOMPATIBLE" {
			return nil, errors.New("IMPORT_INCOMPATIBLE")
		}
		return nil, errors.New("IMPORT_NOT_ELIGIBLE")
	}
	return files, nil
}
func outputParent(fs evidenceFS, name string) error {
	if task.PortablePath(name) != nil {
		return errors.New("unsafe destination")
	}
	parts := strings.Split(path.Dir(name), "/")
	if path.Dir(name) == "." {
		return nil
	}
	for i := range parts {
		info, err := fs.Lstat(strings.Join(parts[:i+1], "/"))
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("unsafe destination")
		}
	}
	return nil
}
func absent(fs evidenceFS, name string) bool { _, err := fs.Lstat(name); return os.IsNotExist(err) }
func ExportSuite(parent context.Context, root, out, dest string) (*SuiteExportResult, error) {
	return exportSuite(parent, root, out, dest, openRootedEvidenceFS)
}
func exportSuite(parent context.Context, root, out, dest string, open openEvidenceFS) (*SuiteExportResult, error) {
	ctx, cancel := context.WithTimeout(parent, 120*time.Second)
	defer cancel()
	r := &SuiteExportResult{Format: "statetwin.dev/suite-export-result/v1alpha1", State: "refused", Provenance: "not-proven"}
	fail := func(code string) (*SuiteExportResult, error) {
		r.FailureCode = code
		if ctx.Err() != nil {
			return r, ctx.Err()
		}
		return r, errors.New(code)
	}
	if task.PortablePath(dest) != nil || beneath(dest, out) || beneath(dest+".pending", out) {
		return fail("EXPORT_INPUT_INVALID")
	}
	files, err := freezeSuite(ctx, root, out)
	if err != nil {
		if oneOf(err.Error(), "EXPORT_INPUT_INVALID", "EXPORT_RESOURCE_LIMIT") {
			return fail(err.Error())
		}
		return fail("EXPORT_NOT_ELIGIBLE")
	}
	fs, err := open(root)
	if err != nil {
		return fail("EXPORT_INPUT_INVALID")
	}
	defer fs.Close()
	if outputParent(fs, dest) != nil {
		return fail("EXPORT_INPUT_INVALID")
	}
	if !absent(fs, dest) || !absent(fs, dest+".pending") {
		return fail("EXPORT_DEST_EXISTS")
	}
	if ctx.Err() != nil {
		return fail("EXPORT_CANCELED")
	}
	f, err := fs.CreateExclusive(dest + ".pending")
	if err != nil {
		return fail("EXPORT_WRITE_FAILED")
	}
	r.State = "failed"
	err = encodeArchive(ctx, f, files)
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil || closeErr != nil {
		return fail("EXPORT_WRITE_FAILED")
	}
	if ctx.Err() != nil {
		return fail("EXPORT_CANCELED")
	}
	if fs.Link(dest+".pending", dest) != nil {
		return fail("EXPORT_PUBLISH_FAILED")
	}
	r.State = "published_with_residue"
	r.Files = len(files)
	for _, b := range files {
		r.Bytes += int64(len(b))
	}
	if fs.Remove(dest+".pending") != nil {
		return fail("EXPORT_CLEANUP_FAILED")
	}
	r.State = "published"
	return r, nil
}
func ImportSuite(parent context.Context, root, archive, out string) (*SuiteImportResult, error) {
	return importSuite(parent, root, archive, out, openRootedEvidenceFS)
}
func importSuite(parent context.Context, root, archive, out string, open openEvidenceFS) (*SuiteImportResult, error) {
	ctx, cancel := context.WithTimeout(parent, 120*time.Second)
	defer cancel()
	r := &SuiteImportResult{Format: "statetwin.dev/suite-import-result/v1alpha1", State: "refused", Provenance: "not-proven"}
	fail := func(code string) (*SuiteImportResult, error) {
		r.FailureCode = code
		if ctx.Err() != nil {
			return r, ctx.Err()
		}
		return r, errors.New(code)
	}
	if task.PortablePath(out) != nil || task.PortablePath(archive) != nil || beneath(archive, out) {
		return fail("IMPORT_INPUT_INVALID")
	}
	if ctx.Err() != nil {
		return fail("IMPORT_CANCELED")
	}
	raw, err := task.ReadFile(root, archive, maxArchiveBytes)
	if err != nil {
		return fail("IMPORT_INPUT_INVALID")
	}
	files, err := decodeArchive(ctx, raw)
	if err != nil {
		if oneOf(err.Error(), "IMPORT_RESOURCE_LIMIT", "IMPORT_INCOMPATIBLE", "IMPORT_NOT_ELIGIBLE") {
			return fail(err.Error())
		}
		return fail("IMPORT_INPUT_INVALID")
	}
	fs, err := open(root)
	if err != nil {
		return fail("IMPORT_INPUT_INVALID")
	}
	defer fs.Close()
	if outputParent(fs, out) != nil {
		return fail("IMPORT_INPUT_INVALID")
	}
	if !absent(fs, out) {
		return fail("IMPORT_DEST_EXISTS")
	}
	if ctx.Err() != nil {
		return fail("IMPORT_CANCELED")
	}
	if fs.Mkdir(out, 0700) != nil {
		return fail("IMPORT_DEST_EXISTS")
	}
	r.State = "failed"
	dirs := map[string]bool{}
	write := func(name string, b []byte) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		dir := path.Dir(name)
		if dir != "." && !dirs[dir] {
			if err := fs.Mkdir(path.Join(out, dir), 0700); err != nil {
				return err
			}
			dirs[dir] = true
		}
		if err := writeExclusiveBytes(fs, path.Join(out, name), b); err != nil {
			return err
		}
		r.WrittenFiles++
		r.WrittenBytes += int64(len(b))
		return nil
	}
	// Claim and plan are always present before any trial is copied.
	for _, name := range []string{"claim.json", "plan.json"} {
		if err := write(name, files["suite/"+name]); err != nil {
			return fail("IMPORT_WRITE_FAILED")
		}
	}
	for _, name := range archiveNames(files) {
		rel := strings.TrimPrefix(name, "suite/")
		if !strings.Contains(rel, "/") {
			continue
		}
		if err := write(rel, files[name]); err != nil {
			return fail("IMPORT_WRITE_FAILED")
		}
	}
	if ctx.Err() != nil {
		return fail("IMPORT_CANCELED")
	}
	report := files["suite/report.json"]
	if writeExclusiveBytes(fs, path.Join(out, "report.pending.json"), report) != nil {
		return fail("IMPORT_WRITE_FAILED")
	}
	if ctx.Err() != nil {
		return fail("IMPORT_CANCELED")
	}
	if fs.Link(path.Join(out, "report.pending.json"), path.Join(out, "report.json")) != nil {
		return fail("IMPORT_PUBLISH_FAILED")
	}
	r.WrittenFiles++
	r.WrittenBytes += int64(len(report))
	r.State = "published_with_residue"
	if fs.Remove(path.Join(out, "report.pending.json")) != nil {
		return fail("IMPORT_CLEANUP_FAILED")
	}
	r.State = "published"
	return r, nil
}
