package doccheck

import (
	"bufio"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

var markdownLink = regexp.MustCompile(`\[[^\]]*\]\(([^)]+)\)`)

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate doccheck source")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(source), "..", ".."))
}

func TestLocalMarkdownLinksResolve(t *testing.T) {
	root := repositoryRoot(t)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == ".git" || strings.HasPrefix(entry.Name(), ".codex-go-cache") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.EqualFold(filepath.Ext(path), ".md") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, match := range markdownLink.FindAllStringSubmatch(string(data), -1) {
			target := strings.Trim(strings.TrimSpace(match[1]), "<>")
			if target == "" || strings.HasPrefix(target, "#") || strings.Contains(target, "://") || strings.HasPrefix(target, "mailto:") {
				continue
			}
			if index := strings.IndexByte(target, '#'); index >= 0 {
				target = target[:index]
			}
			decoded, err := url.PathUnescape(target)
			if err != nil {
				t.Errorf("%s has invalid escaped link %q: %v", path, target, err)
				continue
			}
			resolved := filepath.Clean(filepath.Join(filepath.Dir(path), filepath.FromSlash(decoded)))
			relative, err := filepath.Rel(root, resolved)
			if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
				t.Errorf("%s link escapes repository: %q", path, target)
				continue
			}
			if _, err := os.Stat(resolved); err != nil {
				t.Errorf("%s has unresolved local link %q", path, target)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestUnifiedLifecycleDocumentsExist(t *testing.T) {
	root := repositoryRoot(t)
	required := []string{
		"docs/ADR-0021-UNIFIED-LIFECYCLE-AND-RELEASE-BOUNDARIES.md",
		"docs/CLAIM-REGISTRY.md",
		"docs/COMPATIBILITY-MATRIX.md",
		"docs/DECISION-REGISTER.md",
		"docs/REQUIREMENT-TRACEABILITY.md",
		"docs/SPEC-0019-HOST-PROFILE-AND-LIVE-EVIDENCE.md",
		"docs/SPEC-0020-REMOTE-SECURITY-PROFILE.md",
		"docs/SPEC-0021-CLAIM-REGISTRY-AND-FRESHNESS.md",
	}
	for phase := 0; phase <= 7; phase++ {
		matches, err := filepath.Glob(filepath.Join(root, "docs", "PHASE-0"+string(rune('0'+phase))+"-*.md"))
		if err != nil || len(matches) != 1 {
			t.Errorf("Phase %d contract count = %d, want 1", phase, len(matches))
		}
	}
	for _, name := range required {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(name))); err != nil {
			t.Errorf("required lifecycle document missing: %s", name)
		}
	}
}

func TestHardInvariantTraceabilityIsComplete(t *testing.T) {
	root := repositoryRoot(t)
	file, err := os.Open(filepath.Join(root, "docs", "REQUIREMENT-TRACEABILITY.md"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	seen := map[string]bool{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		for i := 1; i <= 16; i++ {
			id := "I-" + strconv.Itoa(i)
			if strings.Contains(line, "| "+id+" ") {
				if seen[id] {
					t.Errorf("duplicate traceability row for %s", id)
				}
				seen[id] = true
			}
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 16; i++ {
		id := "I-" + strconv.Itoa(i)
		if !seen[id] {
			t.Errorf("missing traceability row for %s", id)
		}
	}
}
