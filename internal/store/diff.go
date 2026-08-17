package store

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
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
	var changes []Change
	diffValue("", left, right, &changes)
	sort.Slice(changes, func(i, j int) bool { return changes[i].Path < changes[j].Path })
	return changes, nil
}

func diffValue(path string, before, after any, changes *[]Change) {
	if reflect.DeepEqual(before, after) {
		return
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
				*changes = append(*changes, Change{Path: child, After: rv})
			case !rok:
				*changes = append(*changes, Change{Path: child, Before: lv})
			default:
				diffValue(child, lv, rv, changes)
			}
		}
		return
	}
	if path == "" {
		path = "/"
	}
	*changes = append(*changes, Change{Path: path, Before: before, After: after})
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
