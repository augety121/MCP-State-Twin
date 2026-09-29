# SPEC-0063: Independent Suite Expectations

Status: **Proposed — not implemented**. Decision proposal:
[ADR-0063](ADR-0063-OFFLINE-ASSESSMENT-PROPOSAL.md).

## 1. Current gap and intended result

`internal/agenteval/suite_inspect.go` admits `claim.json` and `plan.json` from the
same directory before replaying its evidence. A coherent replacement of that
directory's plan/report/evidence can still describe a different intended suite.
The audit does not and should not claim an independent source of intent.

Add a separately supplied expectation for the new assessment in SPEC-0065.
The operator writes/reviews it outside the result directory. Matching means the
observed suite satisfies these specific declarations, not that it ran at a
particular time, was preregistered, or came from an authenticated model.

## 2. Closed input schema

Strict JSON only, one object, at most 64 KiB, depth at most 32. Reuse bounded
duplicate/unknown-field rejection. All fields are required, including numeric
values; reject nulls, string/float coercion and case-aliased field names.

| Field | Rule |
|---|---|
| format | `statetwin.dev/agent-suite-expectation/v1alpha1` |
| profile | `offline-suite-v1` |
| maxOutputTokens | original integer 1–8192, required for every planned trial |
| plan | full current ComparePlan, restricted to suite-generated paths/IDs |

The embedded plan has 1–16 ordered pairs, distinct admitted mock model labels,
unique Task ID/repeat combinations and only `model` in allowedDifferences. Trial
IDs are positional `baseline-01`/`candidate-01`, with direct `<id>/terminal.json`
paths. Order is part of intent. Repeats retain current 1–16 original-integer
semantics; they are identifiers, not automatic expansion counts.

Minimal example (future authoring format, not currently accepted by a command):

```json
{
  "format": "statetwin.dev/agent-suite-expectation/v1alpha1",
  "profile": "offline-suite-v1",
  "maxOutputTokens": 1024,
  "plan": {
    "format": "statetwin.dev/agent-compare-offline/v1alpha1",
    "baselineModel": "mock-baseline",
    "candidateModel": "mock-candidate",
    "allowedDifferences": ["model"],
    "pairs": [{
      "taskId": "close-issue", "repeat": 1,
      "baseline": {"trialId": "baseline-01", "artifact": "baseline-01/terminal.json"},
      "candidate": {"trialId": "candidate-01", "artifact": "candidate-01/terminal.json"}
    }]
  }
}
```

## 3. Admission and matching

1. Validate CLI arguments and expectation before reading suite evidence.
2. Read it through trusted-root portable-path and regular-file checks. Refuse
   an expectation lexically inside the selected output subtree (case-insensitive
   comparison on all platforms) and observed symlinks. This prevents accidental
   use of the suite's own metadata; it does not authenticate file independence.
3. Freeze decoded expectation for the operation. Perform synthetic/privacy
   checks on raw and decoded values before output. Never echo input paths.
4. Run the existing suite audit and compare admitted saved plan to the expected
   plan using semantic equality. Reject reorder, omission, extra pair, changed
   labels, Task IDs, repeat numbers, trial IDs and artifact paths as mismatches.
5. For every replay-verified terminal, compare Config.MaxOutputTokens to the
   expected value. A claim or report counter is insufficient budget evidence.
   Missing/partial/invalid evidence cannot establish a match.

Result: `matched`, `mismatched` or `unverifiable`. A proven difference takes
precedence over unavailable evidence; emit ordered, deduplicated reason codes
`plan_mismatch`, `output_budget_mismatch`, `evidence_unavailable`. Coverage records
planned and verified trial counts separately. A complete match requires every
planned trial verified. Cancellation or resource errors abort assessment.

This binds listed identity labels, order and output budget only. It does not bind
Task/oracle contents to an independent expected definition. SPEC-0064 checks
observed cross-repeat consistency, which is a different and narrower assertion
than authorizing the oracle. No claim of independently approved task semantics.

## 4. Compatibility and implementation map

Add expectation admission/matching under `internal/agenteval`; use the existing
suite plan validator and rooted reads, without weakening older formats. Extend
internal audit plumbing to observe verified definitions in memory rather than
trusting report fields. Public Compare/InspectSuite and their output stay unchanged.
The assessment envelope and CLI are specified by SPEC-0065; do not write a new
expectation into old result directories or silently generate one from their plan.

## 5. Required future acceptance

Positive: independent matching plan, multiple tasks/repeats, equivalent JSON
whitespace/key order, exact integer boundaries and unchanged input bytes.
Negative: coherent saved-plan substitution, dropped/reordered pair, alternate
model, changed output budget, missing/partial trial, null/float/missing fields,
duplicate/unknown fields, symlink/nonregular/oversized input, expectation inside
output, sensitive escaped text and cancellation before evidence reads.
Assert finite errors without paths/content and full planned denominators.
