package agenteval

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestSuiteArchiveRoundtripFrozenNegativeAndNoClobber(t *testing.T) {
	ctx := context.Background()
	root, p := suiteFixture(t, "close-issue")
	runAssessmentFixture(t, root, p)
	before := suiteSnapshot(t, root)
	files, err := freezeSuite(ctx, root, "suite")
	if err != nil {
		t.Fatal(err)
	}
	var a, b bytes.Buffer
	if err := encodeArchive(ctx, &a, files); err != nil {
		t.Fatal(err)
	}
	encodeArchive(ctx, &b, files)
	if !bytes.Equal(a.Bytes(), b.Bytes()) {
		t.Fatal("nondeterministic archive")
	}
	decoded, err := decodeArchive(ctx, a.Bytes())
	if err != nil || !reflect.DeepEqual(files, decoded) {
		t.Fatal("roundtrip", err)
	}
	exported, err := ExportSuite(ctx, root, "suite", "copy.tar")
	if err != nil || exported.State != "published" {
		t.Fatal(exported, err)
	}
	imported, err := ImportSuite(ctx, root, "copy.tar", "restored")
	if err != nil || imported.State != "published" || imported.WrittenFiles != exported.Files || imported.WrittenBytes != exported.Bytes {
		t.Fatal(imported, err)
	}
	original, err := InspectSuite(ctx, root, "suite")
	if err != nil {
		t.Fatal(err)
	}
	restored, err := InspectSuite(ctx, root, "restored")
	if err != nil || !same(original, restored) {
		t.Fatal("audit changed", err)
	}
	after := suiteSnapshot(t, root)
	for n, v := range before {
		if v != after[n] {
			t.Fatal("source changed", n)
		}
	}
	if _, err := ExportSuite(ctx, root, "suite", "copy.tar"); err == nil {
		t.Fatal("overwrite archive")
	}
	if _, err := ImportSuite(ctx, root, "copy.tar", "restored"); err == nil {
		t.Fatal("overwrite suite")
	}
	os.Mkdir(filepath.Join(root, "empty"), 0700)
	if _, err := ImportSuite(ctx, root, "copy.tar", "empty"); err == nil {
		t.Fatal("overwrite empty directory")
	}
	// The frozen view remains auditable after its source is changed.
	os.WriteFile(filepath.Join(root, "suite", "report.json"), []byte("changed source"), 0600)
	if err := archiveEligible(ctx, files); err != nil {
		t.Fatal(err)
	}
	if _, err := ExportSuite(ctx, root, "suite", "bad.tar"); err == nil {
		t.Fatal("bad source exported")
	}
	if _, err := os.Stat(filepath.Join(root, "bad.tar.pending")); !os.IsNotExist(err) {
		t.Fatal("wrote before admission")
	}
	ctx, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := ImportSuite(ctx, root, "copy.tar", "canceled"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
func TestSuiteArchiveMalformedContainers(t *testing.T) {
	ctx := context.Background()
	build := func(headers []*tar.Header, bodies [][]byte) []byte {
		var b bytes.Buffer
		tw := tar.NewWriter(&b)
		for i, h := range headers {
			if err := tw.WriteHeader(h); err != nil {
				t.Fatal(err)
			}
			if len(bodies[i]) > 0 {
				tw.Write(bodies[i])
			}
		}
		tw.Close()
		return b.Bytes()
	}
	manifest, _ := json.Marshal(archiveManifest{Format: archiveFormat, Profile: "offline-suite-copy-v1", Members: []archiveMember{{Path: "payload/claim.json", Bytes: 2}}})
	header := func(name string, size int) *tar.Header {
		return &tar.Header{Name: name, Size: int64(size), Mode: 0600, Typeflag: tar.TypeReg, ModTime: time.Unix(0, 0), Format: tar.FormatUSTAR}
	}
	for _, name := range []string{"../escape", "/absolute", "payload/../escape", "payload\\escape", "payload/CLAIM.json", "payload/unknown.json"} {
		raw := build([]*tar.Header{header("manifest.json", len(manifest)), header(name, 2)}, [][]byte{manifest, []byte("{}")})
		if _, err := decodeArchive(ctx, raw); err == nil {
			t.Fatal("unsafe archive", name)
		}
	}
	link := header("payload/claim.json", 0)
	link.Typeflag = tar.TypeSymlink
	link.Linkname = "outside"
	if _, err := decodeArchive(ctx, build([]*tar.Header{header("manifest.json", len(manifest)), link}, [][]byte{manifest, nil})); err == nil {
		t.Fatal("link accepted")
	}
	dup := build([]*tar.Header{header("manifest.json", len(manifest)), header("payload/claim.json", 2), header("payload/claim.json", 2)}, [][]byte{manifest, []byte("{}"), []byte("{}")})
	if _, err := decodeArchive(ctx, dup); err == nil {
		t.Fatal("duplicate accepted")
	}
	if _, err := decodeArchive(ctx, append(make([]byte, 1024), make([]byte, 512)...)); err == nil {
		t.Fatal("empty archive accepted")
	}
}

func TestArchivePaddingOrderLimitsAndRuntime(t *testing.T) {
	ctx := context.Background()
	root, p := suiteFixture(t, "read-issue")
	runAssessmentFixture(t, root, p)
	files, err := freezeSuite(ctx, root, "suite")
	if err != nil {
		t.Fatal(err)
	}
	var encoded bytes.Buffer
	if err := encodeArchive(ctx, &encoded, files); err != nil {
		t.Fatal(err)
	}
	raw := encoded.Bytes()
	for _, kind := range []string{"padding", "trailing", "truncated", "manifest-length", "runtime"} {
		t.Run(kind, func(t *testing.T) {
			bad := bytes.Clone(raw)
			switch kind {
			case "padding":
				tr := tar.NewReader(bytes.NewReader(bad))
				h, e := tr.Next()
				if e != nil {
					t.Fatal(e)
				}
				bad[512+int(h.Size)] = 1
			case "trailing":
				bad = append(bad, make([]byte, 512)...)
			case "truncated":
				bad = bad[:len(bad)-512]
			case "manifest-length":
				bad = bytes.Replace(bad, []byte(`"bytes":`), []byte(`"bytex":`), 1)
			case "runtime":
				changed := frozenFiles{}
				for name, b := range files {
					changed[name] = b
				}
				name := "suite/baseline-01/terminal.json"
				e, decodeErr := DecodeEvidence(changed[name])
				if decodeErr != nil {
					t.Fatal(decodeErr)
				}
				e.Episode.Definition.RuntimeRevision = "incompatible-runtime"
				changed[name], _ = json.Marshal(e)
				var b bytes.Buffer
				if e := encodeArchive(ctx, &b, changed); e != nil {
					t.Fatal(e)
				}
				bad = b.Bytes()
			}
			if err := os.WriteFile(filepath.Join(root, "bad-"+kind+".tar"), bad, 0600); err != nil {
				t.Fatal(err)
			}
			r, err := ImportSuite(ctx, root, "bad-"+kind+".tar", "out-"+kind)
			if err == nil || r.State != "refused" {
				t.Fatal(r, err)
			}
			if kind == "runtime" && r.FailureCode != "IMPORT_INCOMPATIBLE" {
				t.Fatal(r)
			}
			if _, err := os.Stat(filepath.Join(root, "out-"+kind)); !os.IsNotExist(err) {
				t.Fatal("created output before validation", err)
			}
		})
	}
	// Reject an excessive declared member size without allocating its payload.
	var b bytes.Buffer
	tw := tar.NewWriter(&b)
	if err := tw.WriteHeader(&tar.Header{Name: "manifest.json", Mode: 0600, Size: 65537, Typeflag: tar.TypeReg, ModTime: time.Unix(0, 0), Format: tar.FormatUSTAR}); err != nil {
		t.Fatal(err)
	}
	if _, err := decodeArchive(ctx, b.Bytes()); err == nil || err.Error() != "IMPORT_RESOURCE_LIMIT" {
		t.Fatal(err)
	}
}
func TestArchivePublicationFaults(t *testing.T) {
	ctx := context.Background()
	root, p := suiteFixture(t, "read-issue")
	runAssessmentFixture(t, root, p)
	for _, op := range []string{"open", "write", "short", "sync", "close", "link", "remove"} {
		t.Run("export-"+op, func(t *testing.T) {
			dest := "export-" + op + ".tar"
			leaf := dest + ".pending"
			if op == "link" {
				leaf = dest
			}
			open := func(root string) (evidenceFS, error) {
				fs, err := openRootedEvidenceFS(root)
				if err != nil {
					return nil, err
				}
				return &failingEvidenceFS{evidenceFS: fs, op: op, leaf: leaf}, nil
			}
			r, err := exportSuite(ctx, root, "suite", dest, open)
			if err == nil || r.State == "published" {
				t.Fatal(r, err)
			}
			if op == "remove" && r.State != "published_with_residue" {
				t.Fatal(r)
			}
		})
	}
	if _, err := ExportSuite(ctx, root, "suite", "good.tar"); err != nil {
		t.Fatal(err)
	}
	for _, point := range []struct{ op, leaf string }{{"mkdir", "target"}, {"open", "claim.json"}, {"write", "terminal.json"}, {"short", "terminal.json"}, {"sync", "terminal.json"}, {"close", "terminal.json"}, {"link", "report.json"}, {"remove", "report.pending.json"}} {
		t.Run("import-"+point.op, func(t *testing.T) {
			out := "restore-" + point.op
			leaf := point.leaf
			if leaf == "target" {
				leaf = out
			}
			open := func(root string) (evidenceFS, error) {
				fs, err := openRootedEvidenceFS(root)
				if err != nil {
					return nil, err
				}
				return &failingEvidenceFS{evidenceFS: fs, op: point.op, leaf: leaf}, nil
			}
			r, err := importSuite(ctx, root, "good.tar", out, open)
			if err == nil || r.State == "published" {
				t.Fatal(r, err)
			}
			if point.op == "remove" && r.State != "published_with_residue" {
				t.Fatal(r)
			}
		})
	}
}
func TestArchivePreservesNegativeOutcomes(t *testing.T) {
	ctx := context.Background()
	root, p := suiteFixture(t, "close-issue")
	raw, _ := os.ReadFile(filepath.Join(root, p.Pairs[0].Task))
	var ta map[string]any
	json.Unmarshal(raw, &ta)
	ta["oracle"].([]any)[0].(map[string]any)["expr"] = "false"
	writeTestJSON(t, root, p.Pairs[0].Task, ta)
	runAssessmentFixture(t, root, p)
	if r, err := ExportSuite(ctx, root, "suite", "negative.tar"); err != nil || r.State != "published" {
		t.Fatal(r, err)
	}
	if r, err := ImportSuite(ctx, root, "negative.tar", "negative-copy"); err != nil || r.State != "published" {
		t.Fatal(r, err)
	}
	r, err := InspectSuite(ctx, root, "negative-copy")
	if err != nil || r.Comparison.Pairs[0].Candidate.Outcome != "task_failed" {
		t.Fatal(r, err)
	}
}

func TestArchiveProcessExit(t *testing.T) {
	if mode := os.Getenv("STATETWIN_ARCHIVE_CRASH_HELPER"); mode != "" {
		root := os.Getenv("STATETWIN_ARCHIVE_CRASH_ROOT")
		open := func(root string) (evidenceFS, error) {
			fs, err := openRootedEvidenceFS(root)
			if err != nil {
				return nil, err
			}
			return &failingEvidenceFS{evidenceFS: fs, hook: func(key string) {
				if (mode == "export" && key == "close:crash.tar.pending") || (mode == "import" && key == "close:report.pending.json") {
					os.Exit(87)
				}
			}}, nil
		}
		if mode == "export" {
			exportSuite(context.Background(), root, "suite", "crash.tar", open)
		} else {
			importSuite(context.Background(), root, "good.tar", "crash-import", open)
		}
		t.Fatal("crash point not reached")
	}
	ctx := context.Background()
	root, p := suiteFixture(t, "read-issue")
	runAssessmentFixture(t, root, p)
	if _, err := ExportSuite(ctx, root, "suite", "good.tar"); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"export", "import"} {
		cmd := exec.Command(os.Args[0], "-test.run=^TestArchiveProcessExit$")
		cmd.Env = append(os.Environ(), "STATETWIN_ARCHIVE_CRASH_HELPER="+mode, "STATETWIN_ARCHIVE_CRASH_ROOT="+root)
		err := cmd.Run()
		var exit *exec.ExitError
		if !errors.As(err, &exit) || exit.ExitCode() != 87 {
			t.Fatal(mode, err)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "crash.tar")); !os.IsNotExist(err) {
		t.Fatal("archive published before crash")
	}
	r, err := InspectSuite(ctx, root, "crash-import")
	if err != nil || r.State == "published" || !r.StagingResidue {
		t.Fatal(r, err)
	}
	if _, err := ImportSuite(ctx, root, "good.tar", "crash-import"); err == nil {
		t.Fatal("resumed partial target")
	}
}
