package baselinepack

import (
	"context"
	"errors"
	"path"
	"time"

	"github.com/augety121/mcp-state-twin/internal/agenteval"
)

type QualityGroup struct {
	Root   string                `json:"root"`
	Cases  string                `json:"cases"`
	Report *agenteval.CaseReport `json:"report,omitempty"`
	State  string                `json:"state"`
}
type Quality struct {
	Format             string         `json:"format"`
	PackID             string         `json:"packId"`
	Revision           string         `json:"revision"`
	Decision           string         `json:"decision"`
	Families           int            `json:"families"`
	Instances          int            `json:"instances"`
	Groups             []QualityGroup `json:"groups"`
	Source             string         `json:"source"`
	ReviewIndependence string         `json:"reviewIndependence"`
}

func Qualify(parent context.Context, root, pack, profile string) (*Quality, error) {
	ctx, cancel := context.WithTimeout(parent, 120*time.Second)
	defer cancel()
	p, err := Prepare(ctx, root, pack, profile)
	if err != nil {
		return nil, err
	}
	summary := p.Summary()
	result := &Quality{Format: "statetwin.dev/baseline-quality/v1alpha1", PackID: p.Pack.ID, Revision: p.Pack.Revision, Decision: "partial", Families: summary.Families, Instances: summary.Entries, Source: "scripted-witness-and-synthetic-mutation", ReviewIndependence: p.Pack.ReviewIndependence, Groups: []QualityGroup{}}
	seen := map[[2]string]bool{}
	for _, e := range p.Entries {
		// Relative references bind to the entry root, not the manifest's
		// canonical path. Match static admission's identity exactly.
		key := [2]string{e.Entry.Root, e.Entry.Cases}
		if !seen[key] {
			seen[key] = true
			result.Groups = append(result.Groups, QualityGroup{Root: e.Entry.Root, Cases: e.Entry.Cases, State: "not_started"})
		}
	}
	for i := range result.Groups {
		g := &result.Groups[i]
		read := func(name string, max int) ([]byte, error) { return p.Reader.Read(path.Join(g.Root, name), max) }
		report, err := agenteval.RunPluginCases(ctx, g.Cases, read)
		g.Report = report
		g.State = "matched"
		if err != nil || report == nil || report.Decision != "matched" {
			g.State = "failed"
			return result, errors.New("BASELINE_QUALITY_FAILED")
		}
	}
	result.Decision = "matched"
	return result, nil
}
