package agenteval

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/augety121/mcp-state-twin/internal/bundle"
)

func preparedProjectFixture(t *testing.T) (string, *preparedProject) {
	t.Helper()
	root, _ := projectFixture(t)
	p, _, err := prepareProject(newProjectSource(context.Background(), root, "delivery", projectInputLimit), ".", "project.json")
	if err != nil {
		t.Fatal(err)
	}
	return root, p
}

func TestProjectClosedAdmission(t *testing.T) {
	root, _ := projectFixture(t)
	original, _ := os.ReadFile(filepath.Join(root, "project.json"))
	for _, raw := range []string{
		strings.Replace(string(original), `"id":`, `"unknown":1,"id":`, 1),
		strings.Replace(string(original), `"id":`, `"id":"duplicate","id":`, 1),
		strings.Replace(string(original), `"suite.json"`, `null`, 1),
		strings.Replace(string(original), `"suite.json"`, `32`, 1),
		strings.Replace(string(original), `"suite.json"`, `"../suite.json"`, 1),
		strings.Replace(string(original), `"suite.json"`, `"C:/suite.json"`, 1),
		strings.Replace(string(original), `"cases.json"`, `"missing-last-reference.json"`, 1),
		strings.Repeat(" ", 64<<10) + string(original),
		strings.Replace(string(original), `"id":`, `"extra":`+strings.Repeat("[", 17)+"0"+strings.Repeat("]", 17)+`,"id":`, 1),
	} {
		if err := os.WriteFile(filepath.Join(root, "project.json"), []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
		before := suiteSnapshot(t, root)
		r, err := RunProject(context.Background(), root, "project.json", "delivery")
		if err == nil || r != nil {
			t.Fatal("admitted", r, err)
		}
		if !reflect.DeepEqual(before, suiteSnapshot(t, root)) {
			t.Fatal("preflight wrote output")
		}
	}
}
func TestProjectCheckReadOnly(t *testing.T) {
	root, _ := projectFixture(t)
	before := suiteSnapshot(t, root)
	r, err := CheckProject(context.Background(), root, "project.json")
	if err != nil || r.ExecutionPerformed || r.Provenance != "not-proven" {
		t.Fatal(r, err)
	}
	if !reflect.DeepEqual(before, suiteSnapshot(t, root)) {
		t.Fatal("write during check")
	}
}

func TestProjectInputCoverage(t *testing.T) {
	for _, kind := range []string{"catalog-task", "cases-task", "world-missing", "expectation", "case-id"} {
		t.Run(kind, func(t *testing.T) {
			root, _ := projectFixture(t)
			name := "cases.json"
			var v map[string]any
			if kind == "catalog-task" {
				name = "catalog.json"
			}
			if kind == "world-missing" {
				name = "worlds.json"
			}
			if kind == "expectation" {
				name = "expected.json"
			}
			b, _ := os.ReadFile(filepath.Join(root, name))
			json.Unmarshal(b, &v)
			switch kind {
			case "catalog-task":
				v["tasks"].([]any)[0].(map[string]any)["taskId"] = "different"
			case "cases-task":
				v["tasks"].([]any)[0].(map[string]any)["taskId"] = "different"
			case "world-missing":
				v["worlds"] = []any{}
			case "expectation":
				v["maxOutputTokens"] = 2048
			case "case-id":
				v["cases"].([]any)[3].(map[string]any)["taskId"] = "different"
			}
			writeTestJSON(t, root, name, v)
			if _, err := CheckProject(context.Background(), root, "project.json"); err == nil {
				t.Fatal("accepted mismatch")
			}
		})
	}
}

func TestProjectReferenceSeparation(t *testing.T) {
	for _, kind := range []string{"same-world", "hardlink-world", "same-task", "hardlink-task", "symlink-input", "output-input"} {
		t.Run(kind, func(t *testing.T) {
			root, _ := projectFixture(t)
			switch kind {
			case "same-world":
				var c worldCatalog
				b, _ := os.ReadFile(filepath.Join(root, "worlds.json"))
				json.Unmarshal(b, &c)
				c.Worlds[0].Bundle = "world.stb"
				writeTestJSON(t, root, "worlds.json", c)
			case "hardlink-world":
				os.Remove(filepath.Join(root, "reviewed-world.stb"))
				if err := os.Link(filepath.Join(root, "world.stb"), filepath.Join(root, "reviewed-world.stb")); err != nil {
					t.Skip(err)
				}
			case "same-task", "hardlink-task":
				var c taskCatalog
				b, _ := os.ReadFile(filepath.Join(root, "catalog.json"))
				json.Unmarshal(b, &c)
				if kind == "same-task" {
					c.Tasks[0].Task = "task-close-issue.json"
				} else {
					os.Remove(filepath.Join(root, c.Tasks[0].Task))
					if err := os.Link(filepath.Join(root, "task-close-issue.json"), filepath.Join(root, c.Tasks[0].Task)); err != nil {
						t.Skip(err)
					}
				}
				writeTestJSON(t, root, "catalog.json", c)
			case "symlink-input":
				os.Remove(filepath.Join(root, "positive.json"))
				if err := os.Symlink(filepath.Join(root, "negative.json"), filepath.Join(root, "positive.json")); err != nil {
					t.Skip(err)
				}
			case "output-input":
				if _, err := RunProject(context.Background(), root, "project.json", "task-close-issue.json"); err == nil {
					t.Fatal("input overwritten")
				}
				return
			}
			if _, err := CheckProject(context.Background(), root, "project.json"); err == nil {
				t.Fatal("reference alias accepted")
			}
		})
	}
}

func TestProjectFrozenInputs(t *testing.T) {
	root, p := preparedProjectFixture(t)
	for _, n := range []string{"world.stb", "reviewed-world.stb", "task-close-issue.json", "mock-close-issue.json", "positive.json", "negative.json", "policy.json", "catalog.json", "worlds.json"} {
		if err := os.WriteFile(filepath.Join(root, n), []byte("changed after admission"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	r, err := runPreparedProject(context.Background(), root, "delivery", p, defaultProjectOperations())
	if err != nil || r.Decision != "passed" {
		t.Fatal(r, err)
	}
	if _, err := CheckProject(context.Background(), root, "project.json"); err == nil {
		t.Fatal("cache escaped operation")
	}
}

func TestReviewedWorldEquality(t *testing.T) {
	raw := rawBundle(t)
	a, err := bundle.OpenBytes(raw)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		t.Fatal(err)
	}
	var repacked bytes.Buffer
	w := zip.NewWriter(&repacked)
	w.SetComment("synthetic different container comment")
	for i := len(reader.File) - 1; i >= 0; i-- {
		f := reader.File[i]
		r, e := f.Open()
		if e != nil {
			t.Fatal(e)
		}
		data, e := io.ReadAll(r)
		r.Close()
		if e != nil {
			t.Fatal(e)
		}
		out, e := w.Create(f.Name)
		if e != nil {
			t.Fatal(e)
		}
		out.Write(data)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	b, err := bundle.OpenBytes(repacked.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if len(worldDifferences(a, b)) != 0 {
		t.Fatal("container bytes became world identity")
	}
	b.Files[b.Manifest.Fixture] = append(append([]byte{}, b.Files[b.Manifest.Fixture]...), byte(' '))
	if !containsID(worldDifferences(a, b), "content") {
		t.Fatal("member bytes ignored")
	}
	b.Manifest.Metadata.Version = "9.0.0"
	if !containsID(worldDifferences(a, b), "manifest") {
		t.Fatal("manifest ignored")
	}
	delete(b.Files, b.Manifest.Spec)
	if !containsID(worldDifferences(a, b), "members") {
		t.Fatal("membership ignored")
	}
}

func TestPreparedProjectReadAccounting(t *testing.T) {
	root, _ := projectFixture(t)
	var counts [2][2]int
	for i, off := range []bool{false, true} {
		s := newProjectSource(context.Background(), root, "delivery", projectInputLimit)
		s.noCache = off
		p, _, err := prepareProject(s, ".", "project.json")
		if err != nil || p == nil {
			t.Fatal(err)
		}
		counts[i] = [2]int{s.reads, s.decodes}
		if !off && (s.reads != len(s.files) || s.decodes != 2) {
			t.Fatal(counts)
		}
	}
	if counts[0][0] >= counts[1][0] || counts[0][1] >= counts[1][1] {
		t.Fatal("no actual reuse", counts)
	}
	t.Logf("cached reads/decodes=%v uncached=%v", counts[0], counts[1])
}

func TestProjectResourceBoundaries(t *testing.T) {
	root, _ := projectFixture(t)
	raw, _ := os.ReadFile(filepath.Join(root, "project.json"))
	for _, extra := range []int{0, 1} {
		s := newProjectSource(context.Background(), root, "", len(raw)-extra)
		_, err := s.read(".", "project.json", len(raw))
		if (err != nil) != (extra == 1) {
			t.Fatal(extra, err)
		}
	}
	s := newProjectSource(context.Background(), root, "", projectInputLimit)
	s.projectFiles = map[string]bool{}
	s.projectRaw = len(raw) - 1
	if _, err := s.read(".", "project.json", len(raw)); err == nil || err.Error() != "PROJECT_RESOURCE_LIMIT" {
		t.Fatal(err)
	}
	s = newProjectSource(context.Background(), root, "", projectInputLimit)
	raw = rawBundle(t)
	b, _ := bundle.OpenBytes(raw)
	total := 0
	for _, v := range b.Files {
		total += len(v)
	}
	for _, extra := range []int{0, 1} {
		s = newProjectSource(context.Background(), root, "", projectInputLimit)
		s.extractedLeft = total - extra
		if _, err := s.openBundle(raw); (err != nil) != (extra == 1) {
			t.Fatal(extra, err)
		}
	}
	for _, extra := range []int{0, 1} {
		s = newProjectSource(context.Background(), root, "", projectInputLimit)
		s.projectBundles = map[*byte]bool{}
		s.projectExtracted = total - extra
		if _, err := s.openBundle(raw); (err != nil) != (extra == 1) {
			t.Fatal(extra, err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := CheckProject(ctx, "missing-root", "project.json"); err == nil || err.Error() != "PROJECT_CANCELED_OR_TIMED_OUT" {
		t.Fatal(err)
	}
}
