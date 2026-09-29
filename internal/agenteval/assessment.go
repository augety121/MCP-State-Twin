package agenteval

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/augety121/mcp-state-twin/internal/canonical"
	"github.com/augety121/mcp-state-twin/internal/task"
)

const ExpectationFormat = "statetwin.dev/agent-suite-expectation/v1alpha1"
const AssessmentFormat = "statetwin.dev/agent-suite-assessment/v1alpha1"
const maxDefinitionReferences = 4 << 20

var errAssessmentResourceLimit = errors.New("ASSESSMENT_RESOURCE_LIMIT")

type SuiteExpectation struct {
	Format          string      `json:"format"`
	Profile         string      `json:"profile"`
	MaxOutputTokens int         `json:"maxOutputTokens"`
	Plan            ComparePlan `json:"plan"`
}

func DecodeExpectation(raw []byte) (*SuiteExpectation, error) {
	var e SuiteExpectation
	if decodeSuiteMetadata(raw, MaxSuitePlanBytes, &e) != nil || e.Format != ExpectationFormat || e.Profile != SuiteProfile || e.MaxOutputTokens < 1 || e.MaxOutputTokens > 8192 || validateSuiteComparisonPlan(&e.Plan) != nil {
		return nil, errors.New("ASSESSMENT_EXPECTATION_INVALID")
	}
	return &e, nil
}

type ExpectationCoverage struct {
	PlannedTrials  int `json:"plannedTrials"`
	VerifiedTrials int `json:"verifiedTrials"`
}
type ExpectationResult struct {
	Status   string              `json:"status"`
	Reasons  []string            `json:"reasons"`
	Coverage ExpectationCoverage `json:"coverage"`
}
type DefinitionConsistency struct {
	TaskID                 string   `json:"taskId"`
	PlannedPairs           int      `json:"plannedPairs"`
	PlannedTrials          int      `json:"plannedTrials"`
	VerifiedDefinitions    int      `json:"verifiedDefinitions"`
	UnavailableDefinitions int      `json:"unavailableDefinitions"`
	IdentityStatus         string   `json:"identityStatus"`
	Coverage               string   `json:"coverage"`
	MismatchingTrialIDs    []string `json:"mismatchingTrialIds"`
}
type AssessmentReason struct {
	Code    string `json:"code"`
	TaskID  string `json:"taskId,omitempty"`
	TrialID string `json:"trialId,omitempty"`
}
type SuiteAssessment struct {
	Format            string                  `json:"format"`
	AssessmentProfile string                  `json:"assessmentProfile"`
	Policy            string                  `json:"policy"`
	Decision          string                  `json:"decision"`
	Reasons           []AssessmentReason      `json:"reasons"`
	Expectation       ExpectationResult       `json:"expectation"`
	Consistency       []DefinitionConsistency `json:"consistency"`
	Audit             *SuiteInspection        `json:"audit"`
	UpgradeAllowed    bool                    `json:"upgradeAllowed"`
	Provenance        string                  `json:"provenance"`
}

type assessmentCollector struct {
	expected                              *SuiteExpectation
	planSeen, planMatches, budgetMismatch bool
	verified                              int
	groups                                []DefinitionConsistency
	indices                               map[string]int
	references                            map[string][]byte
	remaining                             int
}

func newAssessmentCollector(e *SuiteExpectation, limit int) *assessmentCollector {
	return &assessmentCollector{expected: e, groups: []DefinitionConsistency{}, indices: map[string]int{}, references: map[string][]byte{}, remaining: limit}
}
func (c *assessmentCollector) plan(p ComparePlan) {
	c.planSeen, c.planMatches = true, same(p, c.expected.Plan)
	for _, pair := range p.Pairs {
		i, ok := c.indices[pair.TaskID]
		if !ok {
			i = len(c.groups)
			c.indices[pair.TaskID] = i
			c.groups = append(c.groups, DefinitionConsistency{TaskID: pair.TaskID, MismatchingTrialIDs: []string{}})
		}
		c.groups[i].PlannedPairs++
		c.groups[i].PlannedTrials += 2
	}
}
func (c *assessmentCollector) observe(taskID, trialID string, d RunDefinition) error {
	if c.planMatches {
		c.verified++
	}
	if d.Config.MaxOutputTokens != c.expected.MaxOutputTokens {
		c.budgetMismatch = true
	}
	d.Config.Model, d.Config.TrialID = "", ""
	raw, err := canonical.JSON(d)
	if err != nil {
		return errors.New("ASSESSMENT_DEFINITION_INVALID")
	}
	g := &c.groups[c.indices[taskID]]
	if ref, ok := c.references[taskID]; ok {
		if !bytes.Equal(ref, raw) {
			g.MismatchingTrialIDs = append(g.MismatchingTrialIDs, trialID)
		}
	} else {
		if len(raw) > c.remaining {
			return errAssessmentResourceLimit
		}
		c.remaining -= len(raw)
		c.references[taskID] = raw
	}
	g.VerifiedDefinitions++
	return nil
}
func (c *assessmentCollector) results() (ExpectationResult, []DefinitionConsistency) {
	e := ExpectationResult{Status: "matched", Reasons: []string{}, Coverage: ExpectationCoverage{PlannedTrials: 2 * len(c.expected.Plan.Pairs), VerifiedTrials: c.verified}}
	if c.planSeen && !c.planMatches {
		e.Reasons = append(e.Reasons, "plan_mismatch")
	}
	if c.budgetMismatch {
		e.Reasons = append(e.Reasons, "output_budget_mismatch")
	}
	provenMismatch := len(e.Reasons) > 0
	if !c.planSeen || e.Coverage.VerifiedTrials != e.Coverage.PlannedTrials {
		e.Reasons = append(e.Reasons, "evidence_unavailable")
		e.Status = "unverifiable"
	}
	if provenMismatch {
		e.Status = "mismatched"
	}
	for i := range c.groups {
		g := &c.groups[i]
		g.UnavailableDefinitions = g.PlannedTrials - g.VerifiedDefinitions
		g.Coverage = "complete"
		g.IdentityStatus = "homogeneous"
		if g.PlannedPairs == 1 {
			g.IdentityStatus = "single_pair"
		}
		if g.UnavailableDefinitions > 0 {
			g.Coverage = "incomplete"
			g.IdentityStatus = "unverifiable"
		}
		if len(g.MismatchingTrialIDs) > 0 {
			g.IdentityStatus = "heterogeneous"
		}
	}
	return e, c.groups
}

// AssessSuite uses an independently provided expectation; never creates one
// from the result directory. All collected definitions have passed replay.
func AssessSuite(parent context.Context, root, out, expectation, policy string) (*SuiteAssessment, error) {
	if !oneOf(policy, "candidate-pass-v1", "both-pass-v1") || task.PortablePath(out) != nil || task.PortablePath(expectation) != nil {
		return nil, errors.New("ASSESSMENT_ARGUMENTS_INVALID")
	}
	foldOut, foldExpect := strings.ToLower(out), strings.ToLower(expectation)
	if foldExpect == foldOut || strings.HasPrefix(foldExpect, foldOut+"/") {
		return nil, errors.New("ASSESSMENT_EXPECTATION_PATH_INVALID")
	}
	ctx, cancel := context.WithTimeout(parent, 120*time.Second)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	raw, err := task.ReadFile(root, expectation, MaxSuitePlanBytes)
	if err != nil {
		return nil, errors.New("ASSESSMENT_EXPECTATION_INVALID")
	}
	e, err := DecodeExpectation(raw)
	if err != nil {
		return nil, err
	}
	return assessPrepared(ctx, root, out, e, policy, maxDefinitionReferences)
}

func assessPrepared(ctx context.Context, root, out string, e *SuiteExpectation, policy string, limit int) (*SuiteAssessment, error) {
	c := newAssessmentCollector(e, limit)
	audit, err := inspectSuiteObserved(ctx, root, out, maxSuiteWriteBytes, c.plan, c.observe)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	expected, groups := c.results()
	r := &SuiteAssessment{Format: AssessmentFormat, AssessmentProfile: "offline-acceptance-v1", Policy: policy, Decision: "passed", Reasons: []AssessmentReason{}, Expectation: expected, Consistency: groups, Audit: audit, Provenance: "not-proven"}
	r.evaluate(e)
	return r, nil
}

func (r *SuiteAssessment) evaluate(e *SuiteExpectation) {
	add := func(code, taskID, trialID string) {
		r.Reasons = append(r.Reasons, AssessmentReason{Code: code, TaskID: taskID, TrialID: trialID})
	}
	if r.Audit.State != "published" || r.Audit.ReportVerification != "matched" || r.Audit.StagingResidue {
		add("audit_not_clean", "", "")
	}
	if r.Expectation.Status == "mismatched" {
		add("expectation_mismatch", "", "")
	} else if r.Expectation.Status != "matched" {
		add("expectation_unverifiable", "", "")
	}
	for _, g := range r.Consistency {
		if g.IdentityStatus == "heterogeneous" {
			add("definitions_heterogeneous", g.TaskID, "")
		}
		if g.Coverage != "complete" || g.IdentityStatus == "unverifiable" {
			add("definitions_unverifiable", g.TaskID, "")
		}
	}
	if r.Audit.Comparison == nil || r.Audit.Comparison.Decision != "no_regression_observed" {
		add("comparison_not_eligible", "", "")
	}
	trials := map[string]TrialResult{}
	key := func(taskID string, repeat int, model, trialID string) string {
		return fmt.Sprintf("%s/%d/%s/%s", taskID, repeat, model, trialID)
	}
	if r.Audit.Comparison != nil {
		for _, p := range r.Audit.Comparison.Pairs {
			for j, row := range []TrialResult{p.Baseline, p.Candidate} {
				model := r.Audit.Comparison.BaselineModel
				if j == 1 {
					model = r.Audit.Comparison.CandidateModel
				}
				trials[key(p.TaskID, p.Repeat, model, row.TrialID)] = row
			}
		}
	}
	for _, p := range e.Plan.Pairs {
		for j, t := range []PlannedTrial{p.Baseline, p.Candidate} {
			if j == 0 && r.Policy != "both-pass-v1" {
				continue
			}
			model := e.Plan.BaselineModel
			if j == 1 {
				model = e.Plan.CandidateModel
			}
			row := trials[key(p.TaskID, p.Repeat, model, t.TrialID)]
			if row.Validation != "valid" || !passing(row.Outcome) {
				code := "baseline_not_passed"
				if j == 1 {
					code = "candidate_not_passed"
				}
				add(code, p.TaskID, t.TrialID)
			}
		}
	}
	if len(r.Reasons) > 0 {
		r.Decision = "failed"
	}
}

func (r *SuiteAssessment) Markdown() string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Offline candidate assessment\n\nPolicy: `%s`. Decision: `%s`.\n\nExpectation: `%s`; verified %d / planned %d. Reasons: %s.\n\nNo automatic upgrade. Provenance not proven; equal definitions do not establish independent samples or approved oracles.\n\n| Task | Planned pairs | Planned trials | Verified definitions | Unavailable | Identity | Coverage | Mismatching trials |\n|---|---|---|---|---|---|---|---|\n", r.Policy, r.Decision, r.Expectation.Status, r.Expectation.Coverage.VerifiedTrials, r.Expectation.Coverage.PlannedTrials, strings.Join(r.Expectation.Reasons, ", "))
	for _, g := range r.Consistency {
		fmt.Fprintf(&b, "| %s | %d | %d | %d | %d | %s | %s | %s |\n", g.TaskID, g.PlannedPairs, g.PlannedTrials, g.VerifiedDefinitions, g.UnavailableDefinitions, g.IdentityStatus, g.Coverage, strings.Join(g.MismatchingTrialIDs, ", "))
	}
	b.WriteString("\n## Acceptance failures\n\n")
	for _, reason := range r.Reasons {
		fmt.Fprintf(&b, "- `%s` %s %s\n", reason.Code, reason.TaskID, reason.TrialID)
	}
	b.WriteString("\n" + r.Audit.Markdown())
	return b.String()
}
