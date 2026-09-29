package agenteval

import (
	"context"
	"errors"
	"time"

	"github.com/augety121/mcp-state-twin/internal/task"
)

type RetentionIntent struct {
	ID         string   `json:"id"`
	Intent     string   `json:"intent"`
	Active     bool     `json:"active"`
	References []string `json:"references"`
}
type RetentionPolicy struct {
	Format  string            `json:"format"`
	Entries []RetentionIntent `json:"entries"`
}
type RetentionRow struct {
	ID           string   `json:"id"`
	Status       string   `json:"status"`
	Reasons      []string `json:"reasons"`
	ReferencedBy []string `json:"referencedBy"`
}
type RetentionPreview struct {
	Format                  string         `json:"format"`
	Completion              string         `json:"completion"`
	Entries                 []RetentionRow `json:"entries"`
	DependencyGroups        [][]string     `json:"dependencyGroups"`
	DeletionPerformed       bool           `json:"deletionPerformed"`
	GlobalReferenceCoverage string         `json:"globalReferenceCoverage"`
	Provenance              string         `json:"provenance"`
}

func PreviewRetention(parent context.Context, root, registry, policy string) (*RetentionPreview, error) {
	ctx, cancel := context.WithTimeout(parent, 120*time.Second)
	defer cancel()
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	reg, err := loadRegistry(root, registry)
	if err != nil {
		return nil, err
	}
	raw, err := task.ReadFile(root, policy, 64<<10)
	if err != nil {
		return nil, errors.New("RETENTION_POLICY_INVALID")
	}
	var p RetentionPolicy
	if decodeSuiteMetadata(raw, 64<<10, &p) != nil || p.Format != "statetwin.dev/retention-intent/v1alpha1" || len(p.Entries) != len(reg.Entries) {
		return nil, errors.New("RETENTION_POLICY_INVALID")
	}
	ids := map[string]bool{}
	for _, e := range reg.Entries {
		ids[e.ID] = true
	}
	seen := map[string]bool{}
	for _, e := range p.Entries {
		if !ids[e.ID] || seen[e.ID] || !oneOf(e.Intent, "keep", "review") || len(e.References) > 16 {
			return nil, errors.New("RETENTION_POLICY_INVALID")
		}
		seen[e.ID] = true
		refs := map[string]bool{}
		for _, id := range e.References {
			if !ids[id] || refs[id] {
				return nil, errors.New("RETENTION_POLICY_INVALID")
			}
			refs[id] = true
		}
	}
	inv, err := inventoryRegistry(ctx, root, reg, "metadata")
	if inv == nil {
		return nil, err
	}
	return retentionResult(reg, &p, inv), err
}
func retentionResult(reg *SuiteRegistry, p *RetentionPolicy, inv *SuiteInventory) *RetentionPreview {
	r := &RetentionPreview{Format: "statetwin.dev/retention-preview/v1alpha1", Completion: inv.Completion, Entries: []RetentionRow{}, DependencyGroups: [][]string{}, GlobalReferenceCoverage: "unknown", Provenance: "not-proven"}
	intents := map[string]RetentionIntent{}
	indices := map[string]int{}
	for _, e := range p.Entries {
		intents[e.ID] = e
	}
	n := len(reg.Entries)
	protected, unknown := make([]bool, n), make([]bool, n)
	reach := make([][]bool, n)
	for i, e := range reg.Entries {
		indices[e.ID] = i
		reach[i] = make([]bool, n)
		intent := intents[e.ID]
		observation := inv.Entries[i]
		unknown[i] = observation.ObservationState != "observed" || observation.Problem != "metadata_present" || !observation.SizeComplete
		protected[i] = unknown[i] || intent.Active || intent.Intent == "keep"
	}
	for i, e := range reg.Entries {
		for _, id := range intents[e.ID].References {
			reach[i][indices[id]] = true
		}
	}
	for k := 0; k < n; k++ {
		for i := 0; i < n; i++ {
			for j := 0; j < n; j++ {
				reach[i][j] = reach[i][j] || (reach[i][k] && reach[k][j])
			}
		}
	}
	for pass := 0; pass < n; pass++ {
		for i := 0; i < n; i++ {
			if protected[i] {
				for j := 0; j < n; j++ {
					if reach[i][j] {
						protected[j] = true
					}
				}
			}
		}
	}
	grouped := map[int]bool{}
	for i, e := range reg.Entries {
		row := RetentionRow{ID: e.ID, Status: "review_candidate", Reasons: []string{}, ReferencedBy: []string{}}
		intent := intents[e.ID]
		if intent.Intent == "keep" {
			row.Reasons = append(row.Reasons, "keep_requested")
		}
		if intent.Active {
			row.Reasons = append(row.Reasons, "active_declared")
		}
		if unknown[i] {
			row.Reasons = append(row.Reasons, "observation_unknown")
		}
		for j, other := range reg.Entries {
			if j != i && protected[j] && reach[j][i] {
				row.ReferencedBy = append(row.ReferencedBy, other.ID)
			}
		}
		if len(row.ReferencedBy) > 0 {
			row.Reasons = append(row.Reasons, "referenced_by_protected")
		}
		if protected[i] {
			row.Status = "protected"
		} else {
			row.Reasons = append(row.Reasons, "not_protected_in_registry")
		}
		if unknown[i] {
			row.Status = "unknown"
		}
		r.Entries = append(r.Entries, row)
		if !grouped[i] {
			group := []string{e.ID}
			grouped[i] = true
			for j := i + 1; j < n; j++ {
				if reach[i][j] && reach[j][i] {
					group = append(group, reg.Entries[j].ID)
					grouped[j] = true
				}
			}
			r.DependencyGroups = append(r.DependencyGroups, group)
		}
	}
	return r
}
