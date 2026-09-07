package task

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func sample() Task {
	return Task{APIVersion: APIVersion, Kind: Kind, Format: Format, ID: "close-issue", Revision: "v1", Domain: "issue-tracker", Bundle: "world.stb", Objective: "Close issue 1 only", Mode: "blind", Tools: []string{"close_issue"}, Authority: []Rule{{Tool: "close_issue", Equals: map[string]any{"number": 1}}}, Budgets: Budgets{16, 32, 30, 180, 10, 1 << 20, 8 << 20}, ExpectedOutcome: "success", Oracle: []Assertion{{"closed", "goal", "true"}, {"scope", "policy", "true"}}}
}

func TestTaskStrictAdmission(t *testing.T) {
	base := sample()
	data, _ := json.Marshal(base)
	if _, err := Decode(data); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*Task){
		"old kind": func(v *Task) { v.Kind = "Scenario" }, "missing budget": func(v *Task) { v.Budgets.ToolAttempts = 0 },
		"traversal": func(v *Task) { v.Bundle = "../secret" }, "empty policy": func(v *Task) { v.Oracle = v.Oracle[:1] },
		"duplicate tool": func(v *Task) { v.Tools = append(v.Tools, v.Tools[0]) }, "hidden authority": func(v *Task) { v.Authority[0].Tool = "reset" },
	} {
		t.Run(name, func(t *testing.T) {
			v := sample()
			mutate(&v)
			b, _ := json.Marshal(v)
			if _, err := Decode(b); err == nil {
				t.Fatal("accepted invalid task")
			}
		})
	}
	for _, bad := range [][]byte{append(data, []byte("\n---\n{}")...), []byte(strings.Repeat("x", MaxBytes+1)), []byte(strings.Replace(string(data), `"id":`, `"unknown":0,"id":`, 1))} {
		if _, err := Decode(bad); err == nil {
			t.Fatal("accepted malformed task")
		}
	}
}

func TestTaskVisibilityAndAuthority(t *testing.T) {
	v := sample()
	v.Oracle[0].Expr = "private-oracle-sentinel"
	b, _ := json.Marshal(v.Visible())
	if strings.Contains(string(b), "sentinel") || strings.Contains(string(b), "world.stb") {
		t.Fatal("private field leaked")
	}
	if !v.Authorized("close_issue", map[string]any{"number": float64(1)}) || v.Authorized("close_issue", map[string]any{"number": 2}) || v.Authorized("reset", map[string]any{"number": 1}) {
		t.Fatal("bad authority")
	}
	if ScalarEqual(float64(9007199254740992), int64(9007199254740993)) {
		t.Fatal("rounded authorization")
	}
}

func TestBoundedRootRead(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "task.json"), []byte("abcd"), 0600); err != nil {
		t.Fatal(err)
	}
	if b, err := ReadFile(root, "task.json", 4); err != nil || string(b) != "abcd" {
		t.Fatal(err)
	}
	for _, name := range []string{"../task.json", "/task.json", "C:/task.json", "a\\b", ".", "a/../task.json"} {
		if _, err := ReadFile(root, name, 4); err == nil {
			t.Fatalf("accepted %q", name)
		}
	}
	if _, err := ReadFile(root, "task.json", 3); err == nil {
		t.Fatal("oversize read")
	}
	if err := os.Symlink(filepath.Join(root, "task.json"), filepath.Join(root, "link")); err == nil {
		if _, err := ReadFile(root, "link", 4); err == nil {
			t.Fatal("symlink accepted")
		}
	}
}
