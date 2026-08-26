package bundle

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/augety121/mcp-state-twin/internal/canonical"
	"github.com/augety121/mcp-state-twin/internal/limits"
	"github.com/augety121/mcp-state-twin/internal/scenario"
	"github.com/augety121/mcp-state-twin/internal/spec"
	"github.com/augety121/mcp-state-twin/internal/strictyaml"
	"github.com/augety121/mcp-state-twin/internal/world"
)

const (
	APIVersion       = "statetwin.dev/v1alpha1"
	Kind             = "TwinBundle"
	Format           = "statetwin.dev/twin-bundle/v1alpha1"
	ManifestName     = "bundle.yaml"
	MaxManifestBytes = 256 << 10
	MaxMemberBytes   = limits.MaxBundleMember
)

var (
	namePattern    = regexp.MustCompile(`^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$`)
	versionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?$`)
)

type Manifest struct {
	APIVersion string            `json:"apiVersion" yaml:"apiVersion"`
	Kind       string            `json:"kind" yaml:"kind"`
	Format     string            `json:"format" yaml:"format"`
	Metadata   Metadata          `json:"metadata" yaml:"metadata"`
	Spec       string            `json:"spec" yaml:"spec"`
	Fixture    string            `json:"fixture" yaml:"fixture"`
	Scenarios  []string          `json:"scenarios" yaml:"scenarios"`
	Digests    map[string]string `json:"digests,omitempty" yaml:"digests,omitempty"`
}

type Metadata struct {
	Name    string `json:"name" yaml:"name"`
	Version string `json:"version" yaml:"version"`
}

type Artifact struct {
	Manifest Manifest          `json:"manifest"`
	Files    map[string][]byte `json:"-"`
	Digest   string            `json:"digest"`
	Size     int64             `json:"sizeBytes"`
}

type Result struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Format     string `json:"format"`
	Name       string `json:"name"`
	Version    string `json:"version"`
	Digest     string `json:"digest"`
	Files      int    `json:"files"`
	Size       int64  `json:"sizeBytes"`
}

func DecodeManifest(data []byte) (*Manifest, error) {
	var manifest Manifest
	if err := strictyaml.DecodeOne(data, MaxManifestBytes, "TwinBundle manifest", &manifest); err != nil {
		return nil, err
	}
	if err := manifest.Validate(false); err != nil {
		return nil, err
	}
	return &manifest, nil
}

func (m *Manifest) Validate(requireDigests bool) error {
	if m == nil {
		return errors.New("TwinBundle manifest is required")
	}
	var problems []string
	if m.APIVersion != APIVersion {
		problems = append(problems, fmt.Sprintf("apiVersion must be %q", APIVersion))
	}
	if m.Kind != Kind {
		problems = append(problems, fmt.Sprintf("kind must be %q", Kind))
	}
	if m.Format != Format {
		problems = append(problems, fmt.Sprintf("format must be %q", Format))
	}
	if !namePattern.MatchString(m.Metadata.Name) {
		problems = append(problems, "metadata.name is invalid")
	}
	if !versionPattern.MatchString(m.Metadata.Version) {
		problems = append(problems, "metadata.version must be a concrete SemVer without build metadata")
	}
	paths := append([]string{m.Spec, m.Fixture}, m.Scenarios...)
	if len(m.Scenarios) == 0 {
		problems = append(problems, "scenarios must not be empty")
	}
	if len(paths) > limits.MaxBundleFiles-1 {
		problems = append(problems, fmt.Sprintf("bundle members exceed limit %d", limits.MaxBundleFiles-1))
	}
	seen := make(map[string]struct{}, len(paths))
	for _, member := range paths {
		if err := validatePath(member); err != nil {
			problems = append(problems, err.Error())
			continue
		}
		if member == ManifestName {
			problems = append(problems, "bundle.yaml is reserved for the generated manifest")
		}
		if _, exists := seen[member]; exists {
			problems = append(problems, fmt.Sprintf("bundle member %q is duplicated", member))
		}
		seen[member] = struct{}{}
	}
	if requireDigests {
		if len(m.Digests) != len(seen) {
			problems = append(problems, "digests must cover every declared member exactly once")
		}
		for member := range seen {
			if !validDigest(m.Digests[member]) {
				problems = append(problems, fmt.Sprintf("digest for %q must be lowercase sha256", member))
			}
		}
		for member := range m.Digests {
			if _, exists := seen[member]; !exists {
				problems = append(problems, fmt.Sprintf("digest references undeclared member %q", member))
			}
		}
	} else if len(m.Digests) != 0 {
		problems = append(problems, "source manifest digests must be omitted; bundle build calculates them")
	}
	if len(problems) > 0 {
		sort.Strings(problems)
		return fmt.Errorf("invalid TwinBundle manifest: %s", strings.Join(problems, "; "))
	}
	return nil
}

func Build(manifestPath, outputPath string) (*Result, error) {
	if outputPath == "" {
		return nil, errors.New("bundle output path is required")
	}
	if _, err := os.Lstat(outputPath); err == nil {
		return nil, fmt.Errorf("bundle output %q already exists", outputPath)
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("inspect bundle output: %w", err)
	}
	manifestInfo, err := os.Lstat(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("stat TwinBundle manifest: %w", err)
	}
	if !manifestInfo.Mode().IsRegular() || manifestInfo.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("TwinBundle manifest must be a regular non-symlink file")
	}
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("read TwinBundle manifest: %w", err)
	}
	manifest, err := DecodeManifest(data)
	if err != nil {
		return nil, err
	}
	root, err := filepath.Abs(filepath.Dir(manifestPath))
	if err != nil {
		return nil, fmt.Errorf("resolve bundle root: %w", err)
	}
	files := make(map[string][]byte)
	for _, member := range declaredMembers(*manifest) {
		memberData, err := readSourceMember(root, member)
		if err != nil {
			return nil, err
		}
		files[member] = memberData
	}
	manifest.Digests = make(map[string]string, len(files))
	for name, memberData := range files {
		manifest.Digests[name] = digestBytes(memberData)
	}
	if err := manifest.Validate(true); err != nil {
		return nil, err
	}
	if err := validatePayloads(*manifest, files); err != nil {
		return nil, err
	}
	manifestData, err := canonical.JSON(manifest)
	if err != nil {
		return nil, fmt.Errorf("encode generated bundle manifest: %w", err)
	}
	files[ManifestName] = manifestData

	outputDir := filepath.Dir(outputPath)
	temporary, err := os.CreateTemp(outputDir, ".statetwin-bundle-*.tmp")
	if err != nil {
		return nil, fmt.Errorf("create temporary bundle: %w", err)
	}
	temporaryName := temporary.Name()
	committed := false
	defer func() {
		_ = temporary.Close()
		if !committed {
			_ = os.Remove(temporaryName)
		}
	}()
	if err := writeArchive(temporary, files); err != nil {
		return nil, err
	}
	if err := temporary.Sync(); err != nil {
		return nil, fmt.Errorf("sync temporary bundle: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return nil, fmt.Errorf("close temporary bundle: %w", err)
	}
	if err := os.Rename(temporaryName, outputPath); err != nil {
		return nil, fmt.Errorf("publish bundle: %w", err)
	}
	committed = true
	return Verify(outputPath)
}

func Verify(bundlePath string) (*Result, error) {
	artifact, err := Open(bundlePath)
	if err != nil {
		return nil, err
	}
	return &Result{
		APIVersion: APIVersion, Kind: "TwinBundleVerification", Format: Format,
		Name: artifact.Manifest.Metadata.Name, Version: artifact.Manifest.Metadata.Version,
		Digest: artifact.Digest, Files: len(artifact.Files), Size: artifact.Size,
	}, nil
}

func Open(bundlePath string) (*Artifact, error) {
	info, err := os.Lstat(bundlePath)
	if err != nil {
		return nil, fmt.Errorf("stat TwinBundle: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("TwinBundle must be a regular non-symlink file")
	}
	if info.Size() > int64(limits.MaxBundleCompressed) {
		return nil, fmt.Errorf("TwinBundle compressed bytes %d exceed limit %d", info.Size(), limits.MaxBundleCompressed)
	}
	reader, err := zip.OpenReader(bundlePath)
	if err != nil {
		return nil, fmt.Errorf("open TwinBundle archive: %w", err)
	}
	defer reader.Close()
	if len(reader.File) == 0 || len(reader.File) > limits.MaxBundleFiles {
		return nil, fmt.Errorf("TwinBundle file count %d is outside 1..%d", len(reader.File), limits.MaxBundleFiles)
	}
	files := make(map[string][]byte, len(reader.File))
	var extracted int64
	for _, entry := range reader.File {
		if err := validatePath(entry.Name); err != nil {
			return nil, err
		}
		if entry.FileInfo().Mode()&os.ModeSymlink != 0 || !entry.Mode().IsRegular() {
			return nil, fmt.Errorf("TwinBundle member %q must be a regular non-symlink file", entry.Name)
		}
		if _, exists := files[entry.Name]; exists {
			return nil, fmt.Errorf("TwinBundle member %q is duplicated", entry.Name)
		}
		if entry.UncompressedSize64 > MaxMemberBytes {
			return nil, fmt.Errorf("TwinBundle member %q exceeds %d bytes", entry.Name, MaxMemberBytes)
		}
		memberData, err := readArchiveMember(entry)
		if err != nil {
			return nil, err
		}
		extracted += int64(len(memberData))
		if extracted > int64(limits.MaxBundleExtracted) {
			return nil, fmt.Errorf("TwinBundle extracted bytes exceed limit %d", limits.MaxBundleExtracted)
		}
		files[entry.Name] = memberData
	}
	manifestData, exists := files[ManifestName]
	if !exists {
		return nil, fmt.Errorf("TwinBundle must contain %s", ManifestName)
	}
	var manifest Manifest
	if err := strictyaml.DecodeOne(manifestData, MaxManifestBytes, "TwinBundle manifest", &manifest); err != nil {
		return nil, err
	}
	if err := manifest.Validate(true); err != nil {
		return nil, err
	}
	if len(files) != len(manifest.Digests)+1 {
		return nil, errors.New("TwinBundle contains undeclared members")
	}
	for name, expected := range manifest.Digests {
		memberData, exists := files[name]
		if !exists {
			return nil, fmt.Errorf("TwinBundle declared member %q is missing", name)
		}
		if actual := digestBytes(memberData); actual != expected {
			return nil, fmt.Errorf("TwinBundle member %q digest mismatch", name)
		}
	}
	if err := validatePayloads(manifest, files); err != nil {
		return nil, err
	}
	bundleData, err := os.ReadFile(bundlePath)
	if err != nil {
		return nil, fmt.Errorf("hash TwinBundle: %w", err)
	}
	return &Artifact{Manifest: manifest, Files: files, Digest: digestBytes(bundleData), Size: info.Size()}, nil
}

func validatePayloads(manifest Manifest, files map[string][]byte) error {
	if _, err := spec.Decode(files[manifest.Spec]); err != nil {
		return fmt.Errorf("decode bundled TwinSpec: %w", err)
	}
	if _, err := world.DecodeStrict(files[manifest.Fixture]); err != nil {
		return fmt.Errorf("decode bundled fixture: %w", err)
	}
	for _, name := range manifest.Scenarios {
		if _, err := scenario.Decode(files[name]); err != nil {
			return fmt.Errorf("decode bundled Scenario %q: %w", name, err)
		}
	}
	return nil
}

func declaredMembers(manifest Manifest) []string {
	members := append([]string{manifest.Spec, manifest.Fixture}, manifest.Scenarios...)
	sort.Strings(members)
	return members
}

func validatePath(name string) error {
	if name == "" || len(name) > 256 {
		return fmt.Errorf("bundle member path %q is empty or too long", name)
	}
	if strings.Contains(name, "\\") || strings.ContainsRune(name, '\x00') || strings.HasPrefix(name, "/") {
		return fmt.Errorf("bundle member path %q is not portable", name)
	}
	cleaned := path.Clean(name)
	if cleaned != name || cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") || path.IsAbs(cleaned) {
		return fmt.Errorf("bundle member path %q escapes or is not canonical", name)
	}
	for _, segment := range strings.Split(name, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return fmt.Errorf("bundle member path %q contains an invalid segment", name)
		}
	}
	return nil
}

func readSourceMember(root, name string) ([]byte, error) {
	target := filepath.Join(root, filepath.FromSlash(name))
	resolved, err := filepath.Abs(target)
	if err != nil {
		return nil, fmt.Errorf("resolve bundle member %q: %w", name, err)
	}
	relative, err := filepath.Rel(root, resolved)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("bundle member %q escapes source root", name)
	}
	current := root
	segments := strings.Split(filepath.FromSlash(name), string(filepath.Separator))
	var info os.FileInfo
	for index, segment := range segments {
		current = filepath.Join(current, segment)
		info, err = os.Lstat(current)
		if err != nil {
			return nil, fmt.Errorf("stat bundle member %q: %w", name, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("bundle member %q traverses a symlink", name)
		}
		if index < len(segments)-1 && !info.IsDir() {
			return nil, fmt.Errorf("bundle member %q has a non-directory path component", name)
		}
	}
	if info == nil || !info.Mode().IsRegular() {
		return nil, fmt.Errorf("bundle member %q must be a regular non-symlink file", name)
	}
	if info.Size() > MaxMemberBytes {
		return nil, fmt.Errorf("bundle member %q exceeds %d bytes", name, MaxMemberBytes)
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		return nil, fmt.Errorf("read bundle member %q: %w", name, err)
	}
	return data, nil
}

func writeArchive(output io.Writer, files map[string][]byte) error {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	archive := zip.NewWriter(output)
	fixedTime := time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, name := range names {
		header := &zip.FileHeader{Name: name, Method: zip.Store, Modified: fixedTime}
		header.SetMode(0o644)
		writer, err := archive.CreateHeader(header)
		if err != nil {
			_ = archive.Close()
			return fmt.Errorf("create TwinBundle member %q: %w", name, err)
		}
		if _, err := writer.Write(files[name]); err != nil {
			_ = archive.Close()
			return fmt.Errorf("write TwinBundle member %q: %w", name, err)
		}
	}
	if err := archive.Close(); err != nil {
		return fmt.Errorf("close TwinBundle archive: %w", err)
	}
	return nil
}

func readArchiveMember(entry *zip.File) ([]byte, error) {
	reader, err := entry.Open()
	if err != nil {
		return nil, fmt.Errorf("open TwinBundle member %q: %w", entry.Name, err)
	}
	defer reader.Close()
	limited := io.LimitReader(reader, MaxMemberBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("read TwinBundle member %q: %w", entry.Name, err)
	}
	if len(data) > MaxMemberBytes {
		return nil, fmt.Errorf("TwinBundle member %q exceeds %d bytes", entry.Name, MaxMemberBytes)
	}
	return data, nil
}

func digestBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func validDigest(value string) bool {
	if len(value) != len("sha256:")+sha256.Size*2 || !strings.HasPrefix(value, "sha256:") {
		return false
	}
	decoded, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil && len(decoded) == sha256.Size && value == strings.ToLower(value)
}
