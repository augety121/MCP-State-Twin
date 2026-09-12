package releasepolicy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func approved(tag string) Plan {
	_, pre, _ := ParseTag(tag)
	channel := "stable"
	if pre {
		channel = "prerelease"
	}
	return Plan{Format: Format, Tag: tag, Profile: "local-core-v0.1", Channel: channel, ClaimsReviewed: true, CompatibilityReviewed: true, StableGatesReviewed: !pre}
}
func notesFixture() string {
	var s strings.Builder
	for _, name := range sections {
		s.WriteString("## " + name + "\n\nSynthetic test-only statement.\n\n")
	}
	return s.String()
}
func put(t *testing.T, name string, data []byte) {
	t.Helper()
	if err := os.WriteFile(name, data, 0600); err != nil {
		t.Fatal(err)
	}
}
func fixture(t *testing.T, tag string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "releases"), 0700); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(approved(tag))
	put(t, filepath.Join(root, "releases", tag+".json"), raw)
	put(t, filepath.Join(root, "releases", tag+".md"), []byte(notesFixture()))
	return root
}

func TestTagGrammarAndNumericBounds(t *testing.T) {
	for _, tag := range []string{"v0.1.0", "v0.1.12-alpha.2", "v1.2.3-0", "v1.2.3-x-y-z.--", "v1.2.3-001a", "v" + strings.Repeat("9", 80) + ".0.0"} {
		version, pre, err := ParseTag(tag)
		if err != nil || version != tag[1:] || pre != strings.Contains(tag, "-") {
			t.Fatalf("%s: %s %v %v", tag, version, pre, err)
		}
	}
	for _, tag := range []string{"", "1.2.3", "v01.2.3", "v1.02.3", "v1.2.03", "v1.2.3.4", "v1.2.3-01", "v1.2.3-alpha.01", "v1.2.3-", "v1.2.3-alpha..1", "v1.2.3+build", "v1.2.3-alpha+build", "v1.2.3-α", "v1.2.3\nother=1", "../v1.2.3", "v1.2.3-x/../x", "v1.2.3-x y", "v1.2.3;cmd", strings.Repeat("v", 129)} {
		if _, _, err := ParseTag(tag); err == nil {
			t.Fatal("invalid tag admitted", tag)
		}
	}
}

func TestPlanReviewAndChannelMatrix(t *testing.T) {
	for _, tag := range []string{"v0.1.0", "v0.1.1-alpha.2"} {
		for mask := 0; mask < 8; mask++ {
			p := approved(tag)
			p.ClaimsReviewed = mask&1 != 0
			p.CompatibilityReviewed = mask&2 != 0
			p.StableGatesReviewed = mask&4 != 0
			raw, _ := json.Marshal(p)
			a, err := decodePlan(raw, tag)
			want := mask == 7
			if strings.Contains(tag, "-") {
				want = mask == 3
			}
			if (err == nil) != want {
				t.Fatalf("tag=%s mask=%d err=%v", tag, mask, err)
			}
			if a != nil && (!a.Draft || a.Latest || a.ApprovalEvidence != "repository-declaration-not-attestation") {
				t.Fatal("approval overstated")
			}
		}
	}
	for _, change := range []func(*Plan){func(p *Plan) { p.Channel = "stable" }, func(p *Plan) { p.Tag = "v0.1.0-alpha.8" }, func(p *Plan) { p.Profile = "remote-production" }, func(p *Plan) { p.Format = "unknown" }} {
		p := approved("v0.1.0-alpha.2")
		change(&p)
		raw, _ := json.Marshal(p)
		if _, err := decodePlan(raw, "v0.1.0-alpha.2"); err == nil {
			t.Fatal("plan mismatch admitted")
		}
	}
	p := approved("v0.2.0")
	raw, _ := json.Marshal(p)
	if _, err := decodePlan(raw, p.Tag); err == nil {
		t.Fatal("unaccepted release train admitted")
	}
}

func TestPlanStrictJSON(t *testing.T) {
	raw, _ := json.Marshal(approved("v0.1.0-alpha.2"))
	good := string(raw)
	for _, bad := range []string{"null", "[]", "{}", "format: value", good + good, good[:len(good)-1] + `,"unknown":1}`, good[:len(good)-1] + `,"tag":"v0.1.0-alpha.2"}`, strings.Replace(good, `"claimsReviewed":true`, `"claimsReviewed":"true"`, 1), strings.Replace(good, `"stableGatesReviewed":false`, `"stableGatesReviewed":null`, 1), strings.Replace(good, `,"stableGatesReviewed":false`, "", 1), strings.Replace(good, "local-core-v0.1", "api_key=synthetic-sentinel", 1), strings.Repeat(" ", maxPlan) + good} {
		if _, err := decodePlan([]byte(bad), "v0.1.0-alpha.2"); err == nil {
			t.Fatal("invalid JSON admitted")
		}
	}
}

func TestReviewedNotesStructureAndPrivacy(t *testing.T) {
	good := notesFixture()
	for _, valid := range []string{good, strings.ReplaceAll(good, "\n", "\r\n"), strings.Replace(good, "## Scope\n", "## Scope\n\n```text\n## not a section\n```\n", 1)} {
		if err := checkNotes([]byte(valid)); err != nil {
			t.Fatal(err)
		}
	}
	for _, bad := range []string{"", "```\n" + good + "```\n", strings.Replace(good, "## Scope", "## Other", 1), good + "## Scope\nagain\n", strings.Replace(good, "Synthetic test-only statement.", "", 1), good + "REVIEW_REQUIRED", good + "\napi_key=synthetic-sentinel", good + "\x00", good + "\xff", good + "\n~~~\n", strings.Repeat("x", maxNotes+1)} {
		if checkNotes([]byte(bad)) == nil {
			t.Fatal("invalid notes admitted")
		}
	}
}

func TestReleaseFilesystemIsReadOnlyBoundedAndConfined(t *testing.T) {
	tag := "v0.1.0-alpha.2"
	root := fixture(t, tag)
	file := filepath.Join(root, "releases", tag+".json")
	before, _ := os.ReadFile(file)
	a, err := Check(root, tag)
	if err != nil || !a.Prerelease || a.Notes != "releases/"+tag+".md" {
		t.Fatalf("%+v %v", a, err)
	}
	after, _ := os.ReadFile(file)
	entries, _ := os.ReadDir(root)
	if string(before) != string(after) || len(entries) != 1 {
		t.Fatal("validation wrote files")
	}
	put(t, file, []byte(strings.Repeat("x", maxPlan+1)))
	if _, err = Check(root, tag); err == nil {
		t.Fatal("oversized plan admitted")
	}
	put(t, file, before)
	put(t, filepath.Join(root, "releases", tag+".md"), []byte(strings.Repeat("x", maxNotes+1)))
	if _, err = Check(root, tag); err == nil {
		t.Fatal("oversized notes admitted")
	}
	if _, err = Check(root, "../escape"); err == nil {
		t.Fatal("path escape")
	}
	if _, err = Check(t.TempDir(), tag); err == nil {
		t.Fatal("missing plan admitted")
	}
}

func TestReleaseRejectsSymlinksAndNonRegularFiles(t *testing.T) {
	tag := "v0.1.0-alpha.2"
	root := fixture(t, tag)
	target := t.TempDir()
	if err := os.Symlink(filepath.Join(root, "releases"), filepath.Join(target, "releases")); err != nil {
		t.Skip("symlink privilege unavailable")
	}
	if _, err := Check(target, tag); err == nil {
		t.Fatal("symlink directory followed")
	}
	notes := filepath.Join(root, "releases", tag+".md")
	if err := os.Remove(notes); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "releases", tag+".json"), notes); err != nil {
		t.Fatal(err)
	}
	if _, err := Check(root, tag); err == nil {
		t.Fatal("symlink notes followed")
	}
	if err := os.Remove(notes); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(notes, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := Check(root, tag); err == nil {
		t.Fatal("directory notes admitted")
	}
}
