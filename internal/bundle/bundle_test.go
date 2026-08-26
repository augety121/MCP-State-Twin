package bundle

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/augety121/mcp-state-twin/internal/canonical"
)

func TestBuildIsDeterministicAndVerifyChecksEveryMember(t *testing.T) {
	root := t.TempDir()
	manifestPath := filepath.Join("..", "..", "examples", "issue-tracker", "bundle.yaml")
	first := filepath.Join(root, "first.stb")
	second := filepath.Join(root, "second.stb")
	result, err := Build(manifestPath, first)
	if err != nil {
		t.Fatal(err)
	}
	if result.Kind != "TwinBundleVerification" || result.Files != 4 {
		t.Fatalf("unexpected build result: %+v", result)
	}
	if _, err := Build(manifestPath, second); err != nil {
		t.Fatal(err)
	}
	firstBytes, _ := os.ReadFile(first)
	secondBytes, _ := os.ReadFile(second)
	if !bytes.Equal(firstBytes, secondBytes) {
		t.Fatal("same inputs produced different TwinBundle bytes")
	}
	artifact, err := Open(first)
	if err != nil {
		t.Fatal(err)
	}
	if artifact.Digest != result.Digest || artifact.Manifest.Digests["state.json"] == "" {
		t.Fatalf("unexpected artifact identity: %+v", artifact)
	}

	tampered := make(map[string][]byte, len(artifact.Files))
	for name, data := range artifact.Files {
		tampered[name] = append([]byte(nil), data...)
	}
	tampered["state.json"] = []byte(`{"entities":{"tampered":{}},"sequences":{}}`)
	tamperedPath := filepath.Join(root, "tampered.stb")
	writeArchiveFile(t, tamperedPath, tampered)
	if _, err := Verify(tamperedPath); err == nil || !strings.Contains(err.Error(), "digest mismatch") {
		t.Fatalf("expected digest mismatch, got %v", err)
	}
}

func TestBuildRefusesExistingOutput(t *testing.T) {
	root := t.TempDir()
	output := writeTestFile(t, root, "existing.stb", "keep")
	manifestPath := filepath.Join("..", "..", "examples", "issue-tracker", "bundle.yaml")
	if _, err := Build(manifestPath, output); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected existing output refusal, got %v", err)
	}
	data, err := os.ReadFile(output)
	if err != nil || string(data) != "keep" {
		t.Fatalf("existing output changed: data=%q err=%v", data, err)
	}
}

func TestBuildRejectsOversizedMemberBeforeReadingIt(t *testing.T) {
	root := t.TempDir()
	exampleRoot := filepath.Join("..", "..", "examples", "issue-tracker")
	for _, name := range []string{"twin.yaml", "scenario-close-issue.yaml"} {
		data, err := os.ReadFile(filepath.Join(exampleRoot, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, name), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	fixture := filepath.Join(root, "state.json")
	file, err := os.Create(fixture)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(MaxMemberBytes + 1); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	manifest := `apiVersion: statetwin.dev/v1alpha1
kind: TwinBundle
format: statetwin.dev/twin-bundle/v1alpha1
metadata: {name: oversized-source, version: 0.2.0-alpha.1}
spec: twin.yaml
fixture: state.json
scenarios: [scenario-close-issue.yaml]
`
	manifestPath := writeTestFile(t, root, "bundle-source.yaml", manifest)
	if _, err := Build(manifestPath, filepath.Join(root, "oversized.stb")); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("expected oversized member refusal, got %v", err)
	}
}

func TestOpenRejectsUndeclaredAndSymlinkMembers(t *testing.T) {
	root := t.TempDir()
	manifestPath := filepath.Join("..", "..", "examples", "issue-tracker", "bundle.yaml")
	valid := filepath.Join(root, "valid.stb")
	if _, err := Build(manifestPath, valid); err != nil {
		t.Fatal(err)
	}
	artifact, err := Open(valid)
	if err != nil {
		t.Fatal(err)
	}
	artifact.Files["undeclared.txt"] = []byte("not declared")
	undeclared := filepath.Join(root, "undeclared.stb")
	writeArchiveFile(t, undeclared, artifact.Files)
	if _, err := Open(undeclared); err == nil || !strings.Contains(err.Error(), "undeclared") {
		t.Fatalf("expected undeclared member refusal, got %v", err)
	}

	symlink := filepath.Join(root, "symlink.stb")
	writeSymlinkArchive(t, symlink)
	if _, err := Open(symlink); err == nil || !strings.Contains(err.Error(), "non-symlink") {
		t.Fatalf("expected archive symlink refusal, got %v", err)
	}
}

func TestSourceAndBundleSymlinksFailClosedWhenSupported(t *testing.T) {
	root := t.TempDir()
	exampleRoot := filepath.Join("..", "..", "examples", "issue-tracker")
	for _, name := range []string{"state.json", "scenario-close-issue.yaml"} {
		data, err := os.ReadFile(filepath.Join(exampleRoot, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, name), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	twinTarget, err := filepath.Abs(filepath.Join(exampleRoot, "twin.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(twinTarget, filepath.Join(root, "twin.yaml")); err != nil {
		t.Skipf("symlink creation is unavailable: %v", err)
	}
	manifest := `apiVersion: statetwin.dev/v1alpha1
kind: TwinBundle
format: statetwin.dev/twin-bundle/v1alpha1
metadata: {name: symlink-source, version: 0.2.0-alpha.1}
spec: twin.yaml
fixture: state.json
scenarios: [scenario-close-issue.yaml]
`
	manifestPath := writeTestFile(t, root, "bundle-source.yaml", manifest)
	if _, err := Build(manifestPath, filepath.Join(root, "source.stb")); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected source symlink refusal, got %v", err)
	}

	valid := filepath.Join(root, "valid.stb")
	if _, err := Build(filepath.Join(exampleRoot, "bundle.yaml"), valid); err != nil {
		t.Fatal(err)
	}
	linked := filepath.Join(root, "linked.stb")
	if err := os.Symlink(valid, linked); err != nil {
		t.Skipf("bundle symlink creation is unavailable: %v", err)
	}
	if _, err := Open(linked); err == nil || !strings.Contains(err.Error(), "non-symlink") {
		t.Fatalf("expected bundle symlink refusal, got %v", err)
	}
}

func TestOpenRejectsSemanticallyInvalidPayload(t *testing.T) {
	root := t.TempDir()
	manifestPath := filepath.Join("..", "..", "examples", "issue-tracker", "bundle.yaml")
	valid := filepath.Join(root, "valid.stb")
	if _, err := Build(manifestPath, valid); err != nil {
		t.Fatal(err)
	}
	artifact, err := Open(valid)
	if err != nil {
		t.Fatal(err)
	}
	artifact.Files[artifact.Manifest.Spec] = []byte("apiVersion: invalid\n")
	artifact.Manifest.Digests[artifact.Manifest.Spec] = digestBytes(artifact.Files[artifact.Manifest.Spec])
	manifestData, err := canonical.JSON(artifact.Manifest)
	if err != nil {
		t.Fatal(err)
	}
	artifact.Files[ManifestName] = manifestData
	invalid := filepath.Join(root, "invalid.stb")
	writeArchiveFile(t, invalid, artifact.Files)
	if _, err := Open(invalid); err == nil || !strings.Contains(err.Error(), "decode bundled TwinSpec") {
		t.Fatalf("expected semantic TwinSpec refusal, got %v", err)
	}
}

func TestManifestAndArchivePathsFailClosed(t *testing.T) {
	manifest := []byte(`apiVersion: statetwin.dev/v1alpha1
kind: TwinBundle
format: statetwin.dev/twin-bundle/v1alpha1
metadata: {name: unsafe, version: 0.2.0}
spec: ../twin.yaml
fixture: state.json
scenarios: [scenario.yaml]
`)
	if _, err := DecodeManifest(manifest); err == nil || !strings.Contains(err.Error(), "escapes") {
		t.Fatalf("expected traversal refusal, got %v", err)
	}

	archivePath := filepath.Join(t.TempDir(), "unsafe.stb")
	writeArchiveFile(t, archivePath, map[string][]byte{"../escape": []byte("x")})
	if _, err := Open(archivePath); err == nil || !strings.Contains(err.Error(), "escapes") {
		t.Fatalf("expected archive traversal refusal, got %v", err)
	}
}

func TestManifestRejectsUnknownFieldsAndCallerSuppliedDigests(t *testing.T) {
	base := `apiVersion: statetwin.dev/v1alpha1
kind: TwinBundle
format: statetwin.dev/twin-bundle/v1alpha1
metadata: {name: strict, version: 0.2.0}
spec: twin.yaml
fixture: state.json
scenarios: [scenario.yaml]
`
	if _, err := DecodeManifest([]byte(base + "unknown: true\n")); err == nil {
		t.Fatal("unknown field was accepted")
	}
	if _, err := DecodeManifest([]byte(base + "digests: {twin.yaml: 'sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa'}\n")); err == nil || !strings.Contains(err.Error(), "must be omitted") {
		t.Fatalf("expected source digest refusal, got %v", err)
	}
}

func writeTestFile(t *testing.T, root, name, contents string) string {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeArchiveFile(t *testing.T, target string, files map[string][]byte) {
	t.Helper()
	output, err := os.Create(target)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(output)
	for name, data := range files {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := output.Close(); err != nil {
		t.Fatal(err)
	}
}

func writeSymlinkArchive(t *testing.T, target string) {
	t.Helper()
	output, err := os.Create(target)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(output)
	header := &zip.FileHeader{Name: ManifestName, Method: zip.Store}
	header.SetMode(os.ModeSymlink | 0o777)
	entry, err := writer.CreateHeader(header)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte("target")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := output.Close(); err != nil {
		t.Fatal(err)
	}
}
