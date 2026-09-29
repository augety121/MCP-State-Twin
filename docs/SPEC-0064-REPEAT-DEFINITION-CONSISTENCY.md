# SPEC-0064: Replay-verified Repeat Definition Consistency

Status: **Proposed — not implemented**, under
[ADR-0063](ADR-0063-OFFLINE-ASSESSMENT-PROPOSAL.md).

## 1. Existing behavior

`compare.go` compares full RunDefinition within each pair after clearing only
Config.Model and Config.TrialID. `compare_summary.go` groups the resulting rows
by Task ID without proving equal definitions across repeats. Two internally
comparable pairs can therefore share an ID but use different oracles or budgets.
Existing summaries are explicitly descriptive and must remain so.

Add a separate consistency result to the new assessment envelope. Never rewrite
old PairResult decisions, successful task counts, or persisted Comparison data.

## 2. Identity rule

Process all planned trials in plan order, grouped by Task ID in first-appearance
order. Admit a definition only after full offline evidence replay has succeeded.
Validly evaluated and replay-verified are distinct: a reproducibly unscorable
trial may contribute definition identity but cannot pass SPEC-0065 outcome gates.
Partial, invalid or identity-mismatched evidence contributes no identity.

Normalize a copy of the full RunDefinition by clearing **only** Config.Model and
Config.TrialID, matching existing pair comparison semantics. Compare canonical
semantic JSON values. Keep Task revision, objective, authority, budgets, oracle,
bundle reference, existing BundleDigest, projection, runtime version/revision,
isolation and model snapshot fields. Offline source admission remains a separate
existing evidence check; Source is not currently a RunDefinition field. Future fields are included by
default; do not introduce a handpicked field list that silently omits them.

Model is the sole declared configuration difference; trial ID is sample identity.
Do not remove oracle text, source paths or runtime fields merely to make repeats
match. No new file digests, hash lists or renamed-copy equivalence is introduced.

## 3. Result and precedence

Each Task ID result has plannedPairs, plannedTrials, verifiedDefinitions,
unavailableDefinitions, identityStatus, coverage and an ordered mismatching-trial
ID list. Trial IDs are safe admitted labels; never output normalized definitions,
oracle expressions or raw evidence. Counts must conserve the planned denominator.

| Condition | identityStatus | coverage |
|---|---|---|
| any two admitted definitions differ | heterogeneous | complete or incomplete |
| no proven difference, at least one unavailable definition | unverifiable | incomplete |
| all agree, exactly one planned pair | single_pair | complete |
| all agree, at least two planned pairs | homogeneous | complete |

The first admitted definition is a deterministic reference for equality only,
not a trusted correct oracle. Report later trial IDs that differ from it, once
each in plan order. If none are admitted there is no reference and the state is
unverifiable. Do not infer that missing evidence agrees with the observed prefix.

`homogeneous` means equal observed definitions, not independent random samples,
statistical confidence, representative tasks or model capability. `single_pair`
may meet a consistency gate but must never be labelled repeated validation.

## 4. Resource and integration contract

Reuse evidence already verified by the audit's Compare path through an internal
observer/collector. Keep only one normalized reference per Task ID and bounded
diagnostic counters/IDs; do not retain all raw traces or repeat filesystem reads
just for cohort classification. Default old calls have no collector and their
serialized comparison results must remain semantically unchanged.

Bound retained canonical reference bytes to 4 MiB cumulative across all groups,
in addition to existing file, plan, size and cooperative deadline limits. Refuse
before retaining a reference that would exceed that allowance. Existing per-Task
bounds apply to transient encoding; this is not a hard process RSS guarantee.
On exhaustion return a finite assessment resource error, not a partial success.

## 5. Required future acceptance

Use real replay-valid fixtures with equal definitions across multiple repeats.
Then vary oracle, Task revision, authority, task budget, output-token budget,
runtime identity and bundle identity, while keeping each baseline/candidate pair
internally equal. Existing pair comparisons should retain their old verdicts;
the new cross-repeat result must detect heterogeneity.

Cover allowed model/trial-ID differences, interleaved Task IDs, one pair,
missing first/middle/last evidence, all missing, unscorable replay-valid evidence,
known mismatch plus missing evidence, maximum admitted pairs, byte budget,
cancellation, deterministic ordering and unchanged old comparison serialization.
Assert no retained or rendered raw task/trace content and no filesystem writes.
