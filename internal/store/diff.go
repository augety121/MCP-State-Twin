package store

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"

	"github.com/augety121/mcp-state-twin/internal/canonical"
	"github.com/augety121/mcp-state-twin/internal/limits"
)

type Change struct {
	Path   string `json:"path"`
	Before any    `json:"before,omitempty"`
	After  any    `json:"after,omitempty"`
}

func (s *Store) DiffBranches(ctx context.Context, beforeID, afterID string) ([]Change, error) {
	before, err := s.Branch(ctx, beforeID)
	if err != nil {
		return nil, fmt.Errorf("read before branch: %w", err)
	}
	after, err := s.Branch(ctx, afterID)
	if err != nil {
		return nil, fmt.Errorf("read after branch: %w", err)
	}
	var left, right any
	lb, _ := json.Marshal(before.State)
	rb, _ := json.Marshal(after.State)
	if err := json.Unmarshal(lb, &left); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(rb, &right); err != nil {
		return nil, err
	}
	changes := make([]Change, 0)
	budget := diffBudget{}
	if err := diffValue("", left, right, &changes, &budget); err != nil {
		return nil, err
	}
	sort.Slice(changes, func(i, j int) bool { return changes[i].Path < changes[j].Path })
	return changes, nil
}

type diffBudget struct {
	bytes int
}

func diffValue(path string, before, after any, changes *[]Change, budget *diffBudget) error {
	if reflect.DeepEqual(before, after) {
		return nil
	}
	left, leftMap := before.(map[string]any)
	right, rightMap := after.(map[string]any)
	if leftMap && rightMap {
		keys := make(map[string]struct{}, len(left)+len(right))
		for key := range left {
			keys[key] = struct{}{}
		}
		for key := range right {
			keys[key] = struct{}{}
		}
		ordered := make([]string, 0, len(keys))
		for key := range keys {
			ordered = append(ordered, key)
		}
		sort.Strings(ordered)
		for _, key := range ordered {
			child := path + "/" + escapePointer(key)
			lv, lok := left[key]
			rv, rok := right[key]
			switch {
			case !lok:
				if err := appendChange(Change{Path: child, After: rv}, changes, budget); err != nil {
					return err
				}
			case !rok:
				if err := appendChange(Change{Path: child, Before: lv}, changes, budget); err != nil {
					return err
				}
			default:
				if err := diffValue(child, lv, rv, changes, budget); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if path == "" {
		path = "/"
	}
	return appendChange(Change{Path: path, Before: before, After: after}, changes, budget)
}

func appendChange(change Change, changes *[]Change, budget *diffBudget) error {
	encoded, err := canonical.JSON(change)
	if err != nil {
		return fmt.Errorf("canonicalize diff entry: %w", err)
	}
	if len(*changes)+1 > limits.MaxDiffEntries {
		return fmt.Errorf("%w: diff entries exceed limit %d", ErrResourceLimit, limits.MaxDiffEntries)
	}
	if budget.bytes+len(encoded) > limits.MaxDiffBytes {
		return fmt.Errorf("%w: diff bytes exceed limit %d", ErrResourceLimit, limits.MaxDiffBytes)
	}
	budget.bytes += len(encoded)
	*changes = append(*changes, change)
	return nil
}

func escapePointer(value string) string {
	result := ""
	for _, r := range value {
		switch r {
		case '~':
			result += "~0"
		case '/':
			result += "~1"
		default:
			result += string(r)
		}
	}
	return result
}
