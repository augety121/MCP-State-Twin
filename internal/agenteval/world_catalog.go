package agenteval

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"

	"github.com/augety121/mcp-state-twin/internal/bundle"
	"github.com/augety121/mcp-state-twin/internal/limits"
)

type worldReference struct {
	TaskID string `json:"taskId"`
	Bundle string `json:"bundle"`
}
type worldCatalog struct {
	Format  string           `json:"format"`
	Profile string           `json:"profile"`
	Worlds  []worldReference `json:"worlds"`
}
type frozenWorlds struct {
	bundles    map[string]*bundle.Artifact
	references []worldReference
}

func loadWorlds(s *projectSource, prefix, name string, p *preparedProject) (*frozenWorlds, error) {
	raw, err := s.read(prefix, name, 64<<10)
	if err != nil {
		return nil, err
	}
	var c worldCatalog
	if decodeSuiteMetadata(raw, 64<<10, &c) != nil || c.Format != "statetwin.dev/reviewed-world-catalog/v1alpha1" || c.Profile != "offline-world-content-v1" || len(c.Worlds) != len(p.catalog.tasks) {
		return nil, errors.New("PROJECT_INPUT_INVALID")
	}
	f := &frozenWorlds{bundles: map[string]*bundle.Artifact{}, references: c.Worlds}
	for _, ref := range c.Worlds {
		t := p.cases.tasks[ref.TaskID]
		if t == nil || f.bundles[ref.TaskID] != nil {
			return nil, errors.New("PROJECT_REFERENCE_MISMATCH")
		}
		raw, err := s.read(prefix, ref.Bundle, limits.MaxBundleCompressed)
		if err != nil {
			return nil, err
		}
		for _, actual := range p.cases.tasks {
			if !s.separate(prefix, ref.Bundle, actual.Bundle) {
				return nil, errors.New("PROJECT_REFERENCE_MISMATCH")
			}
		}
		b, err := s.openBundle(raw)
		if err != nil {
			return nil, err
		}
		f.bundles[ref.TaskID] = b
		for _, in := range p.cases.inputs {
			if in.task.ID == ref.TaskID && len(worldDifferences(b, in.bundle)) > 0 {
				return nil, errors.New("PROJECT_REFERENCE_MISMATCH")
			}
		}
	}
	return f, nil
}
func worldDifferences(a, b *bundle.Artifact) []string {
	r := []string{}
	if a == nil || b == nil {
		return []string{"content"}
	}
	if !same(a.Manifest, b.Manifest) {
		r = append(r, "manifest")
	}
	members, content := len(a.Files) != len(b.Files), false
	for n, v := range a.Files {
		w, ok := b.Files[n]
		if !ok {
			members = true
		} else if !bytes.Equal(v, w) {
			content = true
		}
	}
	if members {
		r = append(r, "members")
	}
	if content {
		r = append(r, "content")
	}
	return r
}

type WorldBinding struct {
	Status            string      `json:"status"`
	PlannedTrials     int         `json:"plannedTrials"`
	MatchedTrials     int         `json:"matchedTrials"`
	MismatchedTrials  int         `json:"mismatchedTrials"`
	UnavailableTrials int         `json:"unavailableTrials"`
	Trials            []TaskMatch `json:"trials"`
}

func worldBindingEmpty(p *preparedProject) WorldBinding {
	b := WorldBinding{Status: "unverifiable", PlannedTrials: len(p.suite.trials), UnavailableTrials: len(p.suite.trials), Trials: []TaskMatch{}}
	for _, pair := range p.suite.plan.Pairs {
		for _, t := range []PlannedTrial{pair.Baseline, pair.Candidate} {
			b.Trials = append(b.Trials, TaskMatch{TaskID: pair.TaskID, Repeat: pair.Repeat, TrialID: t.TrialID, Status: "unverifiable", Differences: []string{}})
		}
	}
	return b
}
func assessProject(ctx context.Context, fs evidenceReadRoot, out string, p *preparedProject) (*ReviewedAssessment, WorldBinding, []ProjectCheckFailure, error) {
	b := worldBindingEmpty(p)
	observations := map[string]TaskMatch{}
	failures := map[string][]ProjectCheckFailure{}
	verify := func(ctx context.Context, e *AgentEvidence, terminal bool) error {
		if err := replay(ctx, e, terminal); err != nil {
			return err
		}
		if !terminal {
			return nil
		}
		d := e.Episode.Definition
		failures[d.Config.TrialID] = []ProjectCheckFailure{}
		if e.Episode.Evaluation != nil {
			for _, c := range e.Episode.Evaluation.Checks {
				if !c.Passed {
					failures[d.Config.TrialID] = append(failures[d.Config.TrialID], ProjectCheckFailure{d.Task.ID, d.Config.TrialID, c.ID, c.Category})
				}
			}
		}
		raw, err := base64.StdEncoding.DecodeString(e.Bundle)
		if err != nil {
			return errors.New("EVIDENCE_INVALID")
		}
		actual, err := bundle.OpenBytes(raw)
		if err != nil {
			return errors.New("EVIDENCE_INVALID")
		}
		diff := worldDifferences(p.worlds.bundles[d.Task.ID], actual)
		status := "matched"
		if len(diff) > 0 {
			status = "mismatched"
		}
		observations[d.Config.TrialID] = TaskMatch{TaskID: d.Task.ID, TrialID: d.Config.TrialID, Status: status, Differences: diff}
		return nil
	}
	a, err := assessReviewedView(ctx, fs, out, p.expect, p.catalog, p.manifest.Policy, verify)
	if err != nil {
		return nil, b, nil, err
	}
	if a.Assessment.Expectation.Status == "matched" {
		for i, row := range b.Trials {
			if v, ok := observations[row.TrialID]; ok && v.TaskID == row.TaskID && a.TaskBinding.Trials[i].Status != "unverifiable" {
				v.Repeat = row.Repeat
				b.Trials[i] = v
				b.UnavailableTrials--
				if v.Status == "matched" {
					b.MatchedTrials++
				} else {
					b.MismatchedTrials++
				}
			}
		}
	}
	if b.MismatchedTrials > 0 {
		b.Status = "mismatched"
	} else if b.UnavailableTrials == 0 {
		b.Status = "matched"
	}
	checks := []ProjectCheckFailure{}
	for _, row := range b.Trials {
		if row.Status != "unverifiable" {
			checks = append(checks, failures[row.TrialID]...)
		}
	}
	return a, b, checks, nil
}
