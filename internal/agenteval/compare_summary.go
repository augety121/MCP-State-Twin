package agenteval

import (
	"errors"
	"fmt"
	"strings"
)

const CompareSummaryPolicy = "planned-task-counts-v1"

type OutcomeCounts struct {
	Success            int `json:"success"`
	ExpectedAbstention int `json:"expectedAbstention"`
	TaskFailed         int `json:"taskFailed"`
	PolicyViolation    int `json:"policyViolation"`
	Unscored           int `json:"unscored"`
}

type DecisionCounts struct {
	Regression           int `json:"regression"`
	Incomparable         int `json:"incomparable"`
	Inconclusive         int `json:"inconclusive"`
	NoRegressionObserved int `json:"noRegressionObserved"`
}

type TaskCohort struct {
	Counts   Denominators  `json:"counts"`
	Outcomes OutcomeCounts `json:"outcomes"`
}

type TaskSummary struct {
	TaskID         string         `json:"taskId"`
	PlannedPairs   int            `json:"plannedPairs"`
	Decision       string         `json:"decision"`
	DecisionCounts DecisionCounts `json:"decisionCounts"`
	Baseline       TaskCohort     `json:"baseline"`
	Candidate      TaskCohort     `json:"candidate"`
}

func (o *OutcomeCounts) observe(t TrialResult) error {
	if t.Validation != "valid" {
		o.Unscored++
		return nil
	}
	switch t.Outcome {
	case "success":
		o.Success++
	case "expected_abstention":
		o.ExpectedAbstention++
	case "task_failed":
		o.TaskFailed++
	case "policy_violation":
		o.PolicyViolation++
	default:
		return errors.New("COMPARE_SUMMARY_INVALID")
	}
	return nil
}

// Summaries are projections of final verified rows, never additional reads or
// selection of best repeats. A Task ID is a grouping label, not a claim that
// its definition is identical across all pairs.
func (r *Comparison) summarizeTasks() error {
	r.SummaryPolicy = CompareSummaryPolicy
	r.BaselineOutcomes, r.CandidateOutcomes = OutcomeCounts{}, OutcomeCounts{}
	r.TaskSummaries = []TaskSummary{}
	indices := make(map[string]int)
	for _, pair := range r.Pairs {
		index, exists := indices[pair.TaskID]
		if !exists {
			index = len(r.TaskSummaries)
			indices[pair.TaskID] = index
			r.TaskSummaries = append(r.TaskSummaries, TaskSummary{TaskID: pair.TaskID, Decision: "no_regression_observed"})
		}
		s := &r.TaskSummaries[index]
		s.PlannedPairs++
		switch pair.Decision {
		case "regression":
			s.DecisionCounts.Regression++
		case "incomparable":
			s.DecisionCounts.Incomparable++
		case "inconclusive":
			s.DecisionCounts.Inconclusive++
		case "no_regression_observed":
			s.DecisionCounts.NoRegressionObserved++
		default:
			return errors.New("COMPARE_SUMMARY_INVALID")
		}
		if decisionPriority(pair.Decision) > decisionPriority(s.Decision) {
			s.Decision = pair.Decision
		}
		for _, side := range []struct {
			trial  TrialResult
			cohort *TaskCohort
			total  *OutcomeCounts
		}{{pair.Baseline, &s.Baseline, &r.BaselineOutcomes}, {pair.Candidate, &s.Candidate, &r.CandidateOutcomes}} {
			side.cohort.Counts.observe(side.trial)
			if err := side.cohort.Outcomes.observe(side.trial); err != nil {
				return err
			}
			if err := side.total.observe(side.trial); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *Comparison) writeTaskSummaries(out *strings.Builder) {
	fmt.Fprintf(out, "\n## Task summaries\n\nSummary policy: `%s`. Grouped by planned Task ID; uniform definitions across repeats are not established. Pair decisions remain authoritative.\n\n", r.SummaryPolicy)
	out.WriteString("| Task | Planned pairs | Decision | Regression | Incomparable | Inconclusive | No regression observed |\n|---|---|---|---|---|---|---|\n")
	for _, s := range r.TaskSummaries {
		d := s.DecisionCounts
		fmt.Fprintf(out, "| %s | %d | %s | %d | %d | %d | %d |\n", s.TaskID, s.PlannedPairs, s.Decision, d.Regression, d.Incomparable, d.Inconclusive, d.NoRegressionObserved)
	}
	out.WriteString("\n| Task | Cohort | Planned | Not started | Started | Incomplete | Terminal | Validly evaluated |\n|---|---|---|---|---|---|---|---|\n")
	for _, s := range r.TaskSummaries {
		for _, side := range []struct {
			name string
			c    Denominators
		}{{"Baseline", s.Baseline.Counts}, {"Candidate", s.Candidate.Counts}} {
			c := side.c
			fmt.Fprintf(out, "| %s | %s | %d | %d | %d | %d | %d | %d |\n", s.TaskID, side.name, c.Planned, c.NotStarted, c.Started, c.Incomplete, c.Terminal, c.ValidlyEvaluated)
		}
	}
	out.WriteString("\nUnscored includes every planned trial without verified scoring, including missing evidence. Counts are descriptive, not success rates.\n\n| Task | Cohort | Success | Expected abstention | Task failed | Policy violation | Unscored |\n|---|---|---|---|---|---|---|\n")
	write := func(taskID, side string, o OutcomeCounts) {
		fmt.Fprintf(out, "| %s | %s | %d | %d | %d | %d | %d |\n", taskID, side, o.Success, o.ExpectedAbstention, o.TaskFailed, o.PolicyViolation, o.Unscored)
	}
	write("All planned tasks", "Baseline", r.BaselineOutcomes)
	write("All planned tasks", "Candidate", r.CandidateOutcomes)
	for _, s := range r.TaskSummaries {
		write(s.TaskID, "Baseline", s.Baseline.Outcomes)
		write(s.TaskID, "Candidate", s.Candidate.Outcomes)
	}
	out.WriteString("\n## Pair details\n")
}
