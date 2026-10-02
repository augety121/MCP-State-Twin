package agenteval

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/augety121/mcp-state-twin/internal/agenthost"
	"github.com/augety121/mcp-state-twin/internal/bundle"
	"github.com/augety121/mcp-state-twin/internal/limits"
	"github.com/augety121/mcp-state-twin/internal/logging"
	"github.com/augety121/mcp-state-twin/internal/strictyaml"
	"github.com/augety121/mcp-state-twin/internal/task"
	"gopkg.in/yaml.v3"
)

const (
	SuiteFormat        = "statetwin.dev/agent-suite-offline/v1alpha1"
	SuiteProfile       = "offline-suite-v1"
	MaxSuitePlanBytes  = 64 << 10
	maxSuitePairs      = 16
	maxSuiteInputBytes = 64 << 20
	maxSuiteWriteBytes = 128 << 20
)

type SuitePair struct {
	TaskID             string `json:"taskId" yaml:"taskId"`
	Task               string `json:"task" yaml:"task"`
	Repeat             int    `json:"repeat" yaml:"repeat"`
	BaselineResponses  string `json:"baselineResponses" yaml:"baselineResponses"`
	CandidateResponses string `json:"candidateResponses" yaml:"candidateResponses"`
}

type SuitePlan struct {
	Format          string      `json:"format" yaml:"format"`
	BaselineModel   string      `json:"baselineModel" yaml:"baselineModel"`
	CandidateModel  string      `json:"candidateModel" yaml:"candidateModel"`
	MaxOutputTokens int         `json:"maxOutputTokens" yaml:"maxOutputTokens"`
	Pairs           []SuitePair `json:"pairs" yaml:"pairs"`
}

type SuitePreflight struct {
	Format             string      `json:"format"`
	Profile            string      `json:"profile"`
	Status             string      `json:"status"`
	PlannedPairs       int         `json:"plannedPairs"`
	PlannedTrials      int         `json:"plannedTrials"`
	InputByteLimit     int         `json:"inputByteLimit"`
	ExtractedByteLimit int         `json:"extractedByteLimit"`
	WriteByteLimit     int         `json:"writeByteLimit"`
	DeadlineSeconds    int         `json:"deadlineSeconds"`
	Plan               ComparePlan `json:"plan"`
}

type preparedSuiteTrial struct {
	config                            RunConfig
	taskBytes, bundleBytes, responses []byte
}

// Private frozen bytes cannot be supplied or edited through the public summary.
type PreparedSuite struct {
	plan   ComparePlan
	trials []preparedSuiteTrial
}

func (p *PreparedSuite) Summary() SuitePreflight {
	plan := p.plan
	plan.Pairs = append([]PlannedPair(nil), plan.Pairs...)
	plan.AllowedDifferences = append([]string(nil), plan.AllowedDifferences...)
	return SuitePreflight{Format: SuiteFormat, Profile: SuiteProfile, Status: "offline-statically-valid", PlannedPairs: len(plan.Pairs), PlannedTrials: len(p.trials), InputByteLimit: maxSuiteInputBytes, ExtractedByteLimit: maxSuiteInputBytes, WriteByteLimit: maxSuiteWriteBytes, DeadlineSeconds: 120, Plan: plan}
}

func DecodeSuite(data []byte) (*SuitePlan, error) {
	var p SuitePlan
	if strictyaml.DecodeOneWithDepth(data, MaxSuitePlanBytes, 16, "SuitePlan", &p) != nil {
		return nil, errors.New("SUITE_PLAN_INVALID")
	}
	var tokens struct {
		MaxOutputTokens yaml.Node `yaml:"maxOutputTokens"`
		Pairs           []struct {
			Repeat yaml.Node `yaml:"repeat"`
		} `yaml:"pairs"`
	}
	if yaml.Unmarshal(data, &tokens) != nil || tokens.MaxOutputTokens.Tag != "!!int" || len(tokens.Pairs) != len(p.Pairs) {
		return nil, errors.New("SUITE_PLAN_INVALID")
	}
	for _, pair := range tokens.Pairs {
		if pair.Repeat.Tag != "!!int" {
			return nil, errors.New("SUITE_PLAN_INVALID")
		}
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return &p, nil
}

func (p *SuitePlan) comparisonPlan() ComparePlan {
	c := ComparePlan{Format: CompareFormat, BaselineModel: p.BaselineModel, CandidateModel: p.CandidateModel, AllowedDifferences: []string{"model"}}
	for i, pair := range p.Pairs {
		base, cand := fmt.Sprintf("baseline-%02d", i+1), fmt.Sprintf("candidate-%02d", i+1)
		c.Pairs = append(c.Pairs, PlannedPair{TaskID: pair.TaskID, Repeat: pair.Repeat, Baseline: PlannedTrial{TrialID: base, Artifact: base + "/terminal.json"}, Candidate: PlannedTrial{TrialID: cand, Artifact: cand + "/terminal.json"}})
	}
	return c
}

func (p *SuitePlan) Validate() error {
	if p == nil || p.Format != SuiteFormat || len(p.Pairs) == 0 || len(p.Pairs) > maxSuitePairs || p.MaxOutputTokens < 1 || p.MaxOutputTokens > 8192 {
		return errors.New("SUITE_PLAN_INVALID")
	}
	c := p.comparisonPlan()
	if c.Validate() != nil {
		return errors.New("SUITE_PLAN_INVALID")
	}
	for _, pair := range p.Pairs {
		for _, file := range []string{pair.Task, pair.BaselineResponses, pair.CandidateResponses} {
			if task.PortablePath(file) != nil {
				return errors.New("SUITE_PLAN_INVALID")
			}
		}
	}
	return nil
}

func PrepareSuite(ctx context.Context, root string, data []byte) (*PreparedSuite, error) {
	return prepareSuite(ctx, root, data, maxSuiteInputBytes, maxSuiteInputBytes)
}

func prepareSuite(ctx context.Context, root string, data []byte, inputBudget, extractedBudget int) (*PreparedSuite, error) {
	return prepareSuiteWith(ctx, data, inputBudget, extractedBudget, func(n string, limit int) ([]byte, error) { return task.ReadFile(root, n, limit) }, bundle.OpenBytes)
}

type inputReader func(string, int) ([]byte, error)
type bundleReader func([]byte) (*bundle.Artifact, error)

func prepareSuiteWith(ctx context.Context, data []byte, inputBudget, extractedBudget int, readFile inputReader, openBundle bundleReader) (*PreparedSuite, error) {
	p, err := DecodeSuite(data)
	if err != nil {
		return nil, err
	}
	read := func(name string, limit int) ([]byte, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if inputBudget <= 0 {
			return nil, errors.New("SUITE_RESOURCE_LIMIT")
		}
		readLimit := min(limit, inputBudget)
		raw, err := readFile(name, readLimit)
		if err != nil {
			if readLimit < limit {
				return nil, errors.New("SUITE_RESOURCE_LIMIT")
			}
			return nil, errors.New("SUITE_INPUT_INVALID")
		}
		if len(raw) > inputBudget {
			return nil, errors.New("SUITE_RESOURCE_LIMIT")
		}
		inputBudget -= len(raw)
		return raw, nil
	}
	prepared := &PreparedSuite{plan: p.comparisonPlan()}
	for i, pair := range p.Pairs {
		taskBytes, err := read(pair.Task, task.MaxBytes)
		if err != nil {
			return nil, err
		}
		t, err := task.Decode(taskBytes)
		if err != nil || t.ID != pair.TaskID {
			return nil, errors.New("SUITE_INPUT_INVALID")
		}
		bundleBytes, err := read(t.Bundle, limits.MaxBundleCompressed)
		if err != nil {
			return nil, err
		}
		b, err := openBundle(bundleBytes)
		if err != nil {
			return nil, errors.New("SUITE_INPUT_INVALID")
		}
		names := make([]string, 0, len(b.Files))
		for name := range b.Files {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			member := b.Files[name]
			if len(member) > extractedBudget {
				return nil, errors.New("SUITE_RESOURCE_LIMIT")
			}
			extractedBudget -= len(member)
			if logging.ContainsSensitive(string(member)) {
				return nil, errors.New("DATA_POLICY_REJECTED")
			}
		}
		for _, side := range []struct {
			trial         PlannedTrial
			model, script string
		}{
			{prepared.plan.Pairs[i].Baseline, p.BaselineModel, pair.BaselineResponses},
			{prepared.plan.Pairs[i].Candidate, p.CandidateModel, pair.CandidateResponses},
		} {
			cfg := RunConfig{Format: ConfigFormat, TrialID: side.trial.TrialID, Profile: agenthost.Profile, Model: side.model, MaxOutputTokens: p.MaxOutputTokens, SyntheticOnly: true}
			if err := Preflight(t, b, &cfg); err != nil {
				return nil, errors.New("SUITE_INPUT_INVALID")
			}
			raw, err := read(side.script, 8<<20)
			if err != nil {
				return nil, err
			}
			m, err := agenthost.DecodeMock(raw)
			if err != nil {
				return nil, errors.New("SUITE_INPUT_INVALID")
			}
			if logging.ContainsSensitive(string(raw)) || safe(m, 8<<20) != nil {
				return nil, errors.New("DATA_POLICY_REJECTED")
			}
			prepared.trials = append(prepared.trials, preparedSuiteTrial{config: cfg, taskBytes: taskBytes, bundleBytes: bundleBytes, responses: raw})
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return prepared, nil
}
