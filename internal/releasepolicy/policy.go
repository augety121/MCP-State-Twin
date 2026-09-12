// Package releasepolicy validates repository declarations without Git, network,
// credential, hashing, build or publication operations.
package releasepolicy

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/augety121/mcp-state-twin/internal/logging"
	"github.com/augety121/mcp-state-twin/internal/strictyaml"
)

const Format = "statetwin.dev/release-plan/v1alpha1"
const maxPlan = 16 << 10
const maxNotes = 64 << 10

var tagPattern = regexp.MustCompile(`^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$`)
var numeric = regexp.MustCompile(`^[0-9]+$`)

type Plan struct {
	Format                string `json:"format" yaml:"format"`
	Tag                   string `json:"tag" yaml:"tag"`
	Profile               string `json:"profile" yaml:"profile"`
	Channel               string `json:"channel" yaml:"channel"`
	ClaimsReviewed        bool   `json:"claimsReviewed" yaml:"claimsReviewed"`
	CompatibilityReviewed bool   `json:"compatibilityReviewed" yaml:"compatibilityReviewed"`
	StableGatesReviewed   bool   `json:"stableGatesReviewed" yaml:"stableGatesReviewed"`
}

type Admission struct {
	Tag              string `json:"tag"`
	Version          string `json:"version"`
	Profile          string `json:"profile"`
	Prerelease       bool   `json:"prerelease"`
	Notes            string `json:"notes"`
	Draft            bool   `json:"draft"`
	Latest           bool   `json:"latest"`
	ApprovalEvidence string `json:"approvalEvidence"`
}

// ParseTag accepts the release tag subset, not all SemVer metadata syntax.
func ParseTag(tag string) (version string, prerelease bool, err error) {
	if len(tag) > 128 {
		return "", false, errors.New("RELEASE_TAG_INVALID")
	}
	m := tagPattern.FindStringSubmatch(tag)
	if m == nil {
		return "", false, errors.New("RELEASE_TAG_INVALID")
	}
	for _, component := range strings.Split(m[4], ".") {
		if numeric.MatchString(component) && len(component) > 1 && component[0] == '0' {
			return "", false, errors.New("RELEASE_TAG_INVALID")
		}
	}
	return tag[1:], m[4] != "", nil
}

func decodePlan(raw []byte, tag string) (*Admission, error) {
	version, pre, err := ParseTag(tag)
	if err != nil {
		return nil, err
	}
	var p Plan
	if !json.Valid(raw) || strictyaml.DecodeOneWithDepth(raw, maxPlan, 8, "ReleasePlan", &p) != nil || json.Unmarshal(raw, &p) != nil || logging.ContainsSensitive(string(raw)) {
		return nil, errors.New("RELEASE_PLAN_INVALID")
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil {
		return nil, errors.New("RELEASE_PLAN_INVALID")
	}
	for _, name := range []string{"claimsReviewed", "compatibilityReviewed", "stableGatesReviewed"} {
		value := strings.TrimSpace(string(fields[name]))
		if value != "true" && value != "false" {
			return nil, errors.New("RELEASE_PLAN_INVALID")
		}
	}
	channel := "stable"
	if pre {
		channel = "prerelease"
	}
	if p.Format != Format || p.Tag != tag || p.Channel != channel || p.Profile != "local-core-v0.1" || !strings.HasPrefix(version, "0.1.") {
		return nil, errors.New("RELEASE_PLAN_MISMATCH_OR_UNSUPPORTED")
	}
	if !p.ClaimsReviewed || !p.CompatibilityReviewed || p.StableGatesReviewed == pre {
		return nil, errors.New("RELEASE_REVIEW_REQUIRED")
	}
	return &Admission{Tag: tag, Version: version, Profile: p.Profile, Prerelease: pre, Notes: "releases/" + tag + ".md", Draft: true, ApprovalEvidence: "repository-declaration-not-attestation"}, nil
}

// Check reads only fixed regular files under a trusted, quiescent root.
func Check(root, tag string) (*Admission, error) {
	if _, _, err := ParseTag(tag); err != nil {
		return nil, err
	}
	fs, err := os.OpenRoot(root)
	if err != nil {
		return nil, errors.New("RELEASE_ROOT_UNAVAILABLE")
	}
	defer fs.Close()
	info, err := fs.Lstat("releases")
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("RELEASE_DIRECTORY_INVALID")
	}
	raw, err := readFile(fs, "releases/"+tag+".json", maxPlan)
	if err != nil {
		return nil, err
	}
	a, err := decodePlan(raw, tag)
	if err != nil {
		return nil, err
	}
	notes, err := readFile(fs, a.Notes, maxNotes)
	if err != nil {
		return nil, err
	}
	if err = checkNotes(notes); err != nil {
		return nil, err
	}
	return a, nil
}

func readFile(fs *os.Root, name string, limit int) ([]byte, error) {
	info, err := fs.Lstat(name)
	if err != nil || !info.Mode().IsRegular() || info.Size() > int64(limit) {
		return nil, errors.New("RELEASE_FILE_INVALID")
	}
	f, err := fs.Open(name)
	if err != nil {
		return nil, errors.New("RELEASE_FILE_INVALID")
	}
	defer f.Close()
	info, err = f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > int64(limit) {
		return nil, errors.New("RELEASE_FILE_INVALID")
	}
	raw, err := io.ReadAll(io.LimitReader(f, int64(limit)+1))
	if err != nil || len(raw) > limit {
		return nil, errors.New("RELEASE_FILE_INVALID")
	}
	return raw, nil
}

var sections = []string{"Scope", "Verified changes", "Compatibility and migration", "Security and hermeticity", "Known limitations / deferred proposals", "Evidence", "Contributors"}

func checkNotes(raw []byte) error {
	invalid := errors.New("RELEASE_NOTES_INVALID")
	if len(raw) > maxNotes || !utf8.Valid(raw) || logging.ContainsSensitive(string(raw)) || strings.Contains(string(raw), "REVIEW_REQUIRED") {
		return invalid
	}
	for _, b := range raw {
		if (b < 32 && b != '\n' && b != '\r' && b != '\t') || b == 127 {
			return invalid
		}
	}
	index, fenceSize := 0, 0
	var fence byte
	body := false
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSuffix(line, "\r")
		trimmed := strings.TrimLeft(line, " ")
		if len(line)-len(trimmed) <= 3 {
			if len(trimmed) >= 3 && (trimmed[0] == '`' || trimmed[0] == '~') {
				run := len(trimmed) - len(strings.TrimLeft(trimmed, string(trimmed[0])))
				if fenceSize == 0 && run >= 3 {
					fence, fenceSize = trimmed[0], run
					continue
				}
				if fenceSize > 0 && trimmed[0] == fence && run >= fenceSize && strings.TrimSpace(trimmed[run:]) == "" {
					fenceSize = 0
					continue
				}
			}
			if fenceSize == 0 && strings.HasPrefix(trimmed, "## ") {
				if index >= len(sections) || trimmed != "## "+sections[index] || (index > 0 && !body) {
					return invalid
				}
				index++
				body = false
				continue
			}
		}
		if index > 0 && strings.TrimSpace(line) != "" {
			body = true
		}
	}
	if index != len(sections) || !body || fenceSize != 0 {
		return invalid
	}
	return nil
}
