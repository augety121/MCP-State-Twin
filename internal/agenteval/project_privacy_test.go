package agenteval

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerificationCacheInvalidation(t *testing.T) {
	root, _ := auditFixture(t)
	ctx := context.Background()
	fs, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer fs.Close()
	view := diskReadRoot{fs}
	name := filepath.Join(root, "suite", "baseline-01", "terminal.json")
	original, _ := os.ReadFile(name)
	for _, change := range []bool{false, true} {
		counts := map[string]int{}
		verify := func(ctx context.Context, e *AgentEvidence, terminal bool) error {
			id := e.Episode.Definition.Config.TrialID
			counts[id]++
			err := replay(ctx, e, terminal)
			if change && id == "baseline-01" && counts[id] == 1 {
				var altered AgentEvidence
				json.Unmarshal(original, &altered)
				altered.Episode.Definition.RuntimeVersion = "unsupported"
				writeTestJSON(t, root, "suite/baseline-01/terminal.json", altered)
			}
			return err
		}
		r, err := inspectSuiteViewVerified(ctx, view, "suite", maxSuiteWriteBytes, nil, nil, verify)
		if err != nil {
			t.Fatal(err)
		}
		if !change {
			if r.State != "published" || counts["baseline-01"] != 1 || counts["candidate-01"] != 1 {
				t.Fatal(r, counts)
			}
		} else {
			if r.State != "invalid" || counts["baseline-01"] != 2 {
				t.Fatal("stale verification reused", r, counts)
			}
		}
		os.WriteFile(name, original, 0600)
	}
}

func TestProjectPrivacyAndMetadataLimits(t *testing.T) {
	root, p := preparedProjectFixture(t)
	raw, _ := os.ReadFile(filepath.Join(root, "project.json"))
	sentinel := "sk-" + strings.Repeat("Z", 36)
	escaped := strings.ReplaceAll(sentinel, "Z", `\u005a`)
	altered := strings.Replace(string(raw), `"issue-core"`, `"`+escaped+`"`, 1)
	os.WriteFile(filepath.Join(root, "project.json"), []byte(altered), 0600)
	r, err := CheckProject(context.Background(), root, "project.json")
	if err == nil {
		t.Fatal("privacy marker admitted")
	}
	encoded, _ := json.Marshal(r)
	if strings.Contains(string(encoded)+err.Error(), sentinel) || strings.Contains(string(encoded), escaped) {
		t.Fatal("privacy marker leaked")
	}
	os.WriteFile(filepath.Join(root, "project.json"), raw, 0600)
	fs, err := openRootedEvidenceFS(root)
	if err != nil {
		t.Fatal(err)
	}
	defer fs.Close()
	left := projectReportLimit
	// JSON string quotes count toward the inclusive per-report limit.
	if err := writeProjectJSON(context.Background(), fs, "at-limit.json", strings.Repeat("x", projectReportLimit-2), &left); err != nil || left != 0 {
		t.Fatal(err, left)
	}
	left = 4 << 20
	if err := writeProjectJSON(context.Background(), fs, "over-limit.json", strings.Repeat("x", projectReportLimit-1), &left); err == nil || err.Error() != "PROJECT_RESOURCE_LIMIT" {
		t.Fatal(err)
	}
	left = 1
	if err := writeProjectJSON(context.Background(), fs, "cumulative.json", newProjectReport(p), &left); err == nil {
		t.Fatal("cumulative limit ignored")
	}
	s := newProjectSource(context.Background(), root, "", projectInputLimit)
	_, err = s.read(".", "project.json", 64<<10)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.read(".", "PROJECT.json", 64<<10); err == nil {
		t.Fatal("case alias admitted")
	}
}
