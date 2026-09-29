# SPEC-0055: Task-level Offline Comparison Summaries

Status: accepted by [ADR-0055](ADR-0055-TASK-COMPARISON-SUMMARIES.md).
Bounded B09-offline report extension; no new execution lane.

## Contract

`eval compare` retains every planned pair and the SPEC-0048–0050 decision,
verification, IO and exit rules. Add `summaryPolicy: planned-task-counts-v1`,
`baselineOutcomes`, `candidateOutcomes` and ordered `taskSummaries` to the existing
experimental report. Strict readers of the old field set must opt into this
extension. Plan, evidence and decision-policy identifiers remain unchanged.

Each outcome object contains `success`, `expectedAbstention`, `taskFailed`,
`policyViolation`, `unscored`. Only trials with `validation: valid` contribute to
the first four buckets; missing, partial, invalid, identity-mismatched and
not-evaluated trials count as unscored even if an artifact claims success.

- The first four buckets sum to `validlyEvaluated`.
- All five buckets sum to `planned`.
- Expected abstention remains separate from success; no rates or statistical
  inference are introduced.

Each task summary has `taskId`, `plannedPairs`, `decision`, `decisionCounts`,
`baseline` and `candidate`. Each side contains existing lifecycle `counts` and
the outcome buckets above. Decision counts are `regression`, `incomparable`,
`inconclusive`, `noRegressionObserved`; they sum to plannedPairs. Task decision
uses the existing priority. Summaries group by the exact planned Task ID, in
first-appearance order, including interleaved repeats and missing-only tasks.
Each pair contributes exactly once. Per-task side counts/outcomes sum to their
global equivalents. Existing combined counts still equal the two global sides.

Task ID grouping is descriptive, not proof of a uniform Task revision, oracle,
budget or runtime across repeats. Pairwise definition comparison stays
authoritative; no average can erase an incomparable or regressed pair. Do not
claim these groups are homogeneous statistical cohorts. The original rows and
reasons remain visible. Unknown internal scored outcomes or pair decisions fail
with `COMPARE_SUMMARY_INVALID` rather than producing an invented summary.

## Rendering and bounds

JSON and Markdown expose the same counts, including zero-valued buckets and
unscored samples. Markdown shows task decisions, lifecycle denominators and
outcome counts, followed by original pair details. Only already-admitted Task
IDs and enums are displayed. No paths, raw errors, expressions or new evidence
content are exposed. Summarization uses existing verified rows without reopening
artifacts, rerunning trials or generating file digests. At most 32 task summaries
can exist under the unchanged 32-pair limit; ordering is independent of map order.
Both outputs remain deterministic and `upgradeAllowed` remains false.

## Plan repeat admission

Serialized pair `repeat` MUST have an original integer scalar tag, be explicitly
present and remain within 1–16. Reject floats (including integral floats and
exponents), strings, null, booleans and omissions before interpreting a plan.
YAML integer spellings retain the existing parser semantics. This closes float
truncation into a different planned repeat; typed Go plans already have integers.
Invalid plans fail before evidence reads and emit no comparison. No source is
rewritten. All existing unknown/duplicate/depth/size rules continue to apply.

## Acceptance

Use real synthetic evidence and replay for multiple tasks and interleaved
repeats. Cover success, expected abstention, complete failure, policy violations,
missing/partial/invalid/identity mismatch and evaluator errors. Verify task/global
denominator equalities, decision priority, all-missing and maximum-size plans,
deterministic JSON/Markdown, unchanged evidence, no path disclosure and CLI
output plus nonzero gates. Regression tests must reproduce fractional repeat
admission before the fix. Full vet/tests/race and PR CI are required; no paid
provider calls or release promotion are part of this increment.
