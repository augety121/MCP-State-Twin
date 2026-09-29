# SPEC-0065: Explicit Offline Candidate Acceptance

Status: **Accepted**, under
[ADR-0063](ADR-0063-OFFLINE-ASSESSMENT-PROPOSAL.md).
Depends on [expectations](SPEC-0063-INDEPENDENT-SUITE-EXPECTATIONS.md) and
[repeat consistency](SPEC-0064-REPEAT-DEFINITION-CONSISTENCY.md).

## 1. Problem and non-goals

The current no-regression policy correctly permits equal baseline/candidate
task failures. `suite-verify` checks clean publication and report agreement, then
that relative policy; it does not claim an absolutely passing candidate.
Keep that behavior. Add an explicitly named acceptance policy rather than
silently changing historical verdicts or presenting success rates as approval.

No provider/live traffic, ranking, automatic upgrade, statistical significance,
custom script policy, threshold expression, retry, subset selection or best-repeat
selection. All planned trials count. No writes to audited directories.

## 2. CLI

```text
statetwin eval suite-assess --root ROOT --out RESULT_DIR --expect EXPECTATION.json --policy POLICY --format json|markdown
```

`--out`, `--expect` and `--policy` are required; no inferred expectation or default
policy. JSON is default. Reject unknown options/formats/positional arguments and
invalid expectation before evidence work. This command is part of
the implemented CLI. Existing suite-inspect/suite-verify commands are unchanged.

One shared cooperative 120-second context covers expectation, audit, collector
and assessment. Retain the existing audit's bounded inventory, file limits and
replay limits, plus expectation and retained-definition bounds from the two
preceding Specs. Cancellation/resource/read errors produce no passing assessment.

## 3. Named policies and exact decisions

Policy IDs: `candidate-pass-v1` and `both-pass-v1`. Both require:

1. existing audit state `published`, reportVerification `matched`, no residue;
2. independent expectation `matched` with complete coverage;
3. every consistency group `homogeneous` or `single_pair`, complete coverage;
4. existing recomputed comparison decision `no_regression_observed`;
5. every required-side planned trial has comparison Validation `valid` and
   Outcome `success` or `expected_abstention`.

`candidate-pass-v1` applies step 5 to candidates; `both-pass-v1` to both sides.
`expected_abstention` qualifies only because existing replay/grading verified
the task's declared expected outcome; arbitrary refusal text never qualifies.
Policy violations, task failure, missing samples and evaluator errors cannot
count as passes. Baseline failure with passing candidate can satisfy candidate
policy when comparison and other checks pass; it fails both-side policy.

Output `decision: passed|failed`, all blocked reasons and named policy. Preserve
reason ordering by check group 1–5, then Task/trial plan order; deduplicate exact
reason/ID combinations. At minimum distinguish `audit_not_clean`,
`expectation_mismatch`, `expectation_unverifiable`, `definitions_heterogeneous`,
`definitions_unverifiable`, `comparison_not_eligible`, `candidate_not_passed`,
`baseline_not_passed`. Never collapse an invalid audit into "no regression".

## 4. New assessment envelope and compatibility

Format `statetwin.dev/agent-suite-assessment/v1alpha1`, assessmentProfile
`offline-acceptance-v1`. Required fields: format, assessmentProfile, policy,
decision, reasons (objects containing code and optional taskId/trialId),
expectation (status, reasons and coverage), consistency (ordered group results),
audit (existing SuiteInspection), upgradeAllowed:false, provenance:not-proven.
For an invalid/unavailable saved plan, expectation is unverifiable, consistency
is an empty list and the audit explains absent observed coverage. Expectation
coverage still retains the independently known expected denominator with zero
verified matches; do not fabricate observed samples.

This output is derived and is not a new replay artifact. It does not modify
SuiteReport, Comparison or SuiteInspection schemas. In particular, do not add
fields to saved Comparison: strict old report admission and equality would break.
No automatic JSON file output; shell redirection is an explicit caller action.

Print available assessment before returning nonzero for failed policy; malformed
input/read/cancellation/resource errors may return without an assessment. Stdout
failure takes precedence over policy success/failure. Markdown shows chosen
policy, expected/observed coverage, consistency, ordered failures and underlying
audit verdict. Do not render Task/oracle text or arbitrary saved diagnostics.

## 5. Required test matrix

| Baseline / candidate | Other checks | candidate-pass-v1 | both-pass-v1 |
|---|---|---|---|
| pass / pass | all satisfied | passed | passed |
| task failure / pass | all satisfied | passed | failed |
| same task failure / same task failure | comparison has no regression | failed | failed |
| pass / task failure | regression | failed | failed |
| expected abstention / expected abstention | verified expected outcomes | passed | passed |
| any / any | missing, invalid, partial or residue | failed | failed |
| pass / pass | expected plan mismatch | failed | failed |
| pass / pass | heterogeneous repeat definitions | failed | failed |
| unscorable / unscorable | replay valid, grading unavailable | failed | failed |

Implement unit and real CLI integration tests for both policies/formats, unknown
policy, missing required flags, canceled execution, failed stdout, deterministic
reasons, unchanged evidence and old CLI/serialization compatibility. Include a
bounded six-task example and a repeat-definition negative example. Run repository
gofmt, vet, tests, race and exact-candidate CI for implementation changes.
