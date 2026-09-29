package agenteval

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"sort"
	"strings"

	"github.com/augety121/mcp-state-twin/internal/agenthost"
	"github.com/augety121/mcp-state-twin/internal/canonical"
	"github.com/augety121/mcp-state-twin/internal/logging"
	"github.com/augety121/mcp-state-twin/internal/task"
)

const catalogFormat = "statetwin.dev/agent-task-catalog/v1alpha1"

var safeLabel = regexp.MustCompile(`^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$`)

func validLabel(s string) bool { return len(s) <= 128 && safeLabel.MatchString(s) }
func beneath(name, parent string) bool {
	n, p := strings.ToLower(name), strings.ToLower(parent)
	return n == p || strings.HasPrefix(n, p+"/")
}

type taskReference struct {
	TaskID string `json:"taskId"`
	Task   string `json:"task"`
}
type taskCatalog struct {
	Format  string          `json:"format"`
	Profile string          `json:"profile"`
	Tasks   []taskReference `json:"tasks"`
}
type frozenCatalog struct {
	entries []taskReference
	tasks   map[string]*task.Task
}

func loadCatalog(ctx context.Context, root, name, out string) (*frozenCatalog, error) {
	if task.PortablePath(name) != nil || task.PortablePath(out) != nil || beneath(name, out) {
		return nil, errors.New("TASK_CATALOG_PATH_INVALID")
	}
	raw, err := task.ReadFile(root, name, MaxSuitePlanBytes)
	if err != nil {
		return nil, errors.New("TASK_CATALOG_INVALID")
	}
	var c taskCatalog
	if decodeSuiteMetadata(raw, MaxSuitePlanBytes, &c) != nil || c.Format != catalogFormat || c.Profile != "offline-task-binding-v1" || len(c.Tasks) < 1 || len(c.Tasks) > 16 {
		return nil, errors.New("TASK_CATALOG_INVALID")
	}
	f := &frozenCatalog{entries: c.Tasks, tasks: map[string]*task.Task{}}
	paths := map[string]bool{}
	bytesLeft, canonicalLeft := 4<<20, 4<<20
	for _, ref := range c.Tasks {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if !validLabel(ref.TaskID) || f.tasks[ref.TaskID] != nil || paths[strings.ToLower(ref.Task)] {
			return nil, errors.New("TASK_CATALOG_INVALID")
		}
		if task.PortablePath(ref.Task) != nil || beneath(ref.Task, out) {
			return nil, errors.New("TASK_CATALOG_PATH_INVALID")
		}
		paths[strings.ToLower(ref.Task)] = true
		raw, err := task.ReadFile(root, ref.Task, task.MaxBytes)
		if err != nil {
			return nil, errors.New("TASK_CATALOG_INVALID")
		}
		bytesLeft -= len(raw)
		t, err := decodeCatalogTask(raw)
		if err != nil {
			return nil, err
		}
		if t.ID != ref.TaskID {
			return nil, errors.New("TASK_CATALOG_INVALID")
		}
		encoded, err := canonical.JSON(t)
		if err != nil {
			return nil, errors.New("TASK_CATALOG_INVALID")
		}
		canonicalLeft -= len(encoded)
		if bytesLeft < 0 || canonicalLeft < 0 || len(encoded) > task.MaxBytes {
			return nil, errors.New("TASK_CATALOG_RESOURCE_LIMIT")
		}
		f.tasks[t.ID] = t
	}
	return f, ctx.Err()
}
func decodeCatalogTask(raw []byte) (*task.Task, error) {
	var t task.Task
	if agenthost.DecodeDocument(raw, task.MaxBytes, &t) != nil {
		return nil, errors.New("TASK_CATALOG_INVALID")
	}
	// Normalize dynamic authority scalars before Task.Validate. Keep typed budgets
	// under encoding/json integer admission; do not coerce fractional spelling.
	for i := range t.Authority {
		if _, err := agenthost.NormalizeNumbers(t.Authority[i].Equals); err != nil {
			return nil, errors.New("TASK_CATALOG_INVALID")
		}
	}
	if t.Validate() != nil || safe(&t, task.MaxBytes) != nil {
		return nil, errors.New("TASK_CATALOG_INVALID")
	}
	var original map[string]any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if decoder.Decode(&original) != nil {
		return nil, errors.New("TASK_CATALOG_INVALID")
	}
	for _, key := range []string{"context", "faultTool"} {
		if v, ok := original[key]; ok && v == "" {
			delete(original, key)
		}
	}
	if !same(original, &t) {
		return nil, errors.New("TASK_CATALOG_INVALID")
	}
	// Also scan raw text, including credentials hidden in discarded syntax.
	if logging.ContainsSensitive(string(raw)) {
		return nil, errors.New("DATA_POLICY_REJECTED")
	}
	return &t, nil
}
func catalogCoverage(c *frozenCatalog, p ComparePlan) (missing, extra []string) {
	missing, extra = []string{}, []string{}
	seen := map[string]bool{}
	for _, pair := range p.Pairs {
		if !seen[pair.TaskID] {
			seen[pair.TaskID] = true
			if c.tasks[pair.TaskID] == nil {
				missing = append(missing, pair.TaskID)
			}
		}
	}
	for id := range c.tasks {
		if !seen[id] {
			extra = append(extra, id)
		}
	}
	sort.Strings(extra)
	return
}
func taskDifferences(a, b *task.Task) []string {
	result := []string{}
	if same(a, b) {
		return result
	}
	if a == nil || b == nil {
		return []string{"other"}
	}
	x, y := *a, *b
	if !same([]any{x.APIVersion, x.Kind, x.Format, x.ID, x.Revision, x.Domain, x.Bundle, x.Mode}, []any{y.APIVersion, y.Kind, y.Format, y.ID, y.Revision, y.Domain, y.Bundle, y.Mode}) {
		result = append(result, "identity")
	}
	if !same([]any{x.Objective, x.Context, x.ExpectedOutcome}, []any{y.Objective, y.Context, y.ExpectedOutcome}) {
		result = append(result, "goal")
	}
	if !same([]any{x.Tools, x.FaultTool}, []any{y.Tools, y.FaultTool}) {
		result = append(result, "tools")
	}
	if !same(x.Authority, y.Authority) {
		result = append(result, "authority")
	}
	if !same(x.Budgets, y.Budgets) {
		result = append(result, "budgets")
	}
	if !same(x.Oracle, y.Oracle) {
		result = append(result, "oracle")
	}
	// Zero known fields on copies so future Task fields participate by default.
	x.APIVersion = ""
	x.Kind = ""
	x.Format = ""
	x.ID = ""
	x.Revision = ""
	x.Domain = ""
	x.Bundle = ""
	x.Mode = ""
	x.Objective = ""
	x.Context = ""
	x.ExpectedOutcome = ""
	x.Tools = nil
	x.FaultTool = ""
	x.Authority = nil
	x.Budgets = task.Budgets{}
	x.Oracle = nil
	y.APIVersion = ""
	y.Kind = ""
	y.Format = ""
	y.ID = ""
	y.Revision = ""
	y.Domain = ""
	y.Bundle = ""
	y.Mode = ""
	y.Objective = ""
	y.Context = ""
	y.ExpectedOutcome = ""
	y.Tools = nil
	y.FaultTool = ""
	y.Authority = nil
	y.Budgets = task.Budgets{}
	y.Oracle = nil
	if !same(x, y) || len(result) == 0 {
		result = append(result, "other")
	}
	return result
}
