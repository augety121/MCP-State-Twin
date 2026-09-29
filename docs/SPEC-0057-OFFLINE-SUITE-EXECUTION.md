# SPEC-0057: Bounded Sequential Offline Suite Execution

Status: accepted by [ADR-0056](ADR-0056-BOUNDED-OFFLINE-SUITES.md).
Depends on [SPEC-0056](SPEC-0056-OFFLINE-SUITE-PREFLIGHT.md).

## 1. Execution contract

`eval suite --root ROOT --suite PLAN --out NEW_RELATIVE_DIRECTORY` runs only
successfully prepared private inputs. It claims a new directory exclusively,
writes/syncs a minimal claim and the generated `plan.json`, then executes pairs
in declared order: baseline first, candidate second. Output parents must exist
and pass the existing trusted-root checks. Existing files/directories always
fail; no overwrite or resume, even after an interrupted prior invocation.

Each trial reuses RecordMock, its own fresh in-memory world/session, existing
Task budgets, trace policy, replay closure and no-clobber terminal publication.
It gets a private copy of the frozen Task/config/script. No trial inherits world
state, opaque continuation or grading feedback. Execution is strictly sequential.
Completed `task_failed`, `policy_violation` and evaluator-error results do not
stop subsequent trials if execution, evidence and cleanup are complete. They
remain visible to comparison and the final CLI gate.

Stop starting new trials on context cancellation/deadline, RecordMock error,
non-completed execution, incomplete evidence or cleanup. Preserve the completed
prefix and all unstarted entries; never retry, skip a failed trial then pretend
the suite completed, or remove evidence. A failed run may retain staging.

## 2. Resource and failure boundary

The suite supplies a 120-second cooperative execution/comparison deadline;
Task deadlines may be shorter. Existing bounded terminal cleanup can outlive a
canceled execution context; this is not a hard OS/filesystem time guarantee.

A shared 128 MiB cumulative artifact-write budget applies to all suite and trial
files, including staging writes later removed. Account actual written bytes;
reject a write before it could exceed the remaining allowance. This conservative
IO budget is not final directory size. No file hashes or new hash manifests.
Use the existing exclusive writer, short-write/sync/close checks and hard-link
publication requirements. A storage/budget error must not produce success.

All rows begin `not_started`; admitted attempts become `completed` only when
execution/evidence/cleanup are complete, otherwise `failed`. Known status and
outcome fields may be copied from the returned Episode. Raw internal errors,
input paths and scripts never enter the suite report. Suite stop codes are finite.

## 3. Comparison and terminal result

After the loop, if the parent/deadline permits, compare all planned artifacts
through the existing replay-based Compare path (SPEC-0048–0055). Missing entries
stay in denominators. A stopped suite may have a complete diagnostic comparison;
that does not make its execution successful. Canceled/timed-out suites do not
start comparison. Comparison error records failure, not an invented verdict.

Report format `statetwin.dev/agent-suite-report/v1alpha1` contains executionStatus
(`completed`, `stopped`, `canceled`, `timed_out`), plannedTrials, ordered trials,
finite failureCode, comparisonStatus (`complete`, `not_attempted`, `failed`),
optional existing Comparison, profile and `upgradeAllowed:false`. A complete
suite can still have a regressed/inconclusive comparison. No live/model claim.

## 4. Acceptance

Run six existing tasks × two mock labels end-to-end; verify all terminal artifacts
and count conservation. Cover ordinary goal failure followed by later successful
trials, partial/host failure stop, cancellation between trials, output conflicts,
write-budget exhaustion, short writes/sync/link failures, immutable source inputs,
no new attempts after stop and unchanged prior evidence. Full vet/test/race and
exact PR candidate CI are required.
