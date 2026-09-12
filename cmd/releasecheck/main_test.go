package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/augety121/mcp-state-twin/internal/releasepolicy"
)

type brokenOutput struct{}

func (brokenOutput) Write([]byte) (int, error) { return 0, errors.New("synthetic output unavailable") }

func TestReleasecheckErrorIsSanitized(t *testing.T) {
	if diagnostic(errors.New("private output/path details")) != "RELEASE_OUTPUT_FAILED" || diagnostic(errors.New("RELEASE_REVIEW_REQUIRED")) != "RELEASE_REVIEW_REQUIRED" {
		t.Fatal("error diagnostic lost its closed code boundary")
	}
}

func TestReleasecheckCLI(t *testing.T) {
	root := t.TempDir()
	tag := "v0.1.0-alpha.2"
	if err := os.Mkdir(filepath.Join(root, "releases"), 0700); err != nil {
		t.Fatal(err)
	}
	p := releasepolicy.Plan{Format: releasepolicy.Format, Tag: tag, Profile: "local-core-v0.1", Channel: "prerelease", ClaimsReviewed: true, CompatibilityReviewed: true}
	raw, _ := json.Marshal(p)
	if err := os.WriteFile(filepath.Join(root, "releases", tag+".json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	notes := ""
	for _, title := range []string{"Scope", "Verified changes", "Compatibility and migration", "Security and hermeticity", "Known limitations / deferred proposals", "Evidence", "Contributors"} {
		notes += "## " + title + "\n\nSynthetic test.\n\n"
	}
	if err := os.WriteFile(filepath.Join(root, "releases", tag+".md"), []byte(notes), 0600); err != nil {
		t.Fatal(err)
	}
	for _, format := range []string{"json", "github"} {
		var out bytes.Buffer
		args := []string{"--root", root, "--tag", tag, "--format", format}
		if err := run(args, &out); err != nil {
			t.Fatal(err)
		}
		if format == "json" {
			var a releasepolicy.Admission
			if json.Unmarshal(out.Bytes(), &a) != nil || !a.Prerelease || !a.Draft || a.Latest {
				t.Fatal("invalid JSON policy output")
			}
		} else if out.String() != "tag="+tag+"\nversion="+tag[1:]+"\nprerelease=true\nnotes=releases/"+tag+".md\n" {
			t.Fatal("unsafe workflow output")
		}
		if strings.Contains(out.String(), "Synthetic test.") || strings.Contains(out.String(), root) {
			t.Fatal("private content leaked")
		}
		if run(args, brokenOutput{}) == nil {
			t.Fatal("output failure hidden")
		}
	}
	for _, args := range [][]string{{}, {"--tag", "bad"}, {"--tag", tag, "extra"}, {"--token", "synthetic"}, {"--format", "sh"}, {"--root", root, "--tag", "v0.1.1-alpha.1"}} {
		var out bytes.Buffer
		if run(args, &out) == nil || out.Len() != 0 {
			t.Fatal("invalid input produced admission")
		}
	}
}
