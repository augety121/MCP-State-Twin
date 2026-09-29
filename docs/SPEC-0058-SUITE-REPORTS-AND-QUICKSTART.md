# SPEC-0058: Suite Reports, CLI Gates and Six-task Quickstart

Status: accepted by [ADR-0056](ADR-0056-BOUNDED-OFFLINE-SUITES.md).
Builds on [SPEC-0056](SPEC-0056-OFFLINE-SUITE-PREFLIGHT.md) and
[SPEC-0057](SPEC-0057-OFFLINE-SUITE-EXECUTION.md).

## 1. Artifacts and durable boundaries

A claimed suite directory contains `claim.json`, generated `plan.json`, zero or
more trial directories, and on successful report publication `report.json`.
The report is written/synced as `report.pending.json`, then hard-linked without
replacement to `report.json`; the owned pending file is removed only afterwards.
No rename/copy fallback or deletion of trial evidence. Publication/cleanup errors
return failure even if the final link became visible. Power-loss durability on
every filesystem and atomic snapshots are not claimed.

The plan and report are derived metadata, not standalone execution provenance.
Never accept report.json alone as replay proof: use each trial's `eval verify`
or rerun `eval compare --root SUITE_DIR --plan plan.json`. These only read existing
evidence. Interrupted directories remain inspect-only; per-trial `eval inspect`
is available. Whole-suite resume and a new suite-inspection command are not part
of this batch. Do not manually repair a failed directory to reuse it.

## 2. CLI and output

`eval suite-preflight` prints its safe static summary and exits zero only on
successful admission. `eval suite` prints the available SuiteReport as JSON even
for diagnostic failure, then returns nonzero on stopped/canceled/timed-out runs,
storage failures, missing comparison or any comparison gate other than
`no_regression_observed`. With no report available, print no successful JSON.
Output-write failures must not be swallowed by a gate. Existing single-trial
commands and compare flags/behavior stay available.

Suite report publication can fail after execution; do not report that the
artifact exists merely because a report object exists in memory. Preserve the
error and directory. Reports and plans use finite safe labels; no host endpoints,
Task/oracle expressions, original paths or raw script frames in suite metadata.
The ordinary trial evidence retains the established synthetic-only whitelist.

## 3. Reproducible example and documentation

Ship one six-pair plan referencing the existing issue-tracker task/mock fixtures:
read-issue, close-issue, create-issue, already-closed, scope-protection,
after-commit-confirm. Two distinct mock labels share output-token settings.
The user builds the existing agent TwinBundle once, runs suite-preflight, then
suite. Comparison may be rerendered as JSON or Markdown using the saved plan.
The example validates infrastructure, not autonomous model quality.

The guide must explain output conflicts, keeping failed runs, a candidate
omit-action regression, fixed repeats/trial IDs, cumulative budgets and safe
fresh-directory reruns. Wire the example through automated CLI integration
tests in a temporary clean root. All generated test artifacts are owned and
removed by test cleanup; user examples/data are never overwritten.

## 4. Acceptance and release boundary

CLI tests cover dry-run zero writes, all six tasks, twelve independently verified
trial artifacts, saved plan/report, expected nonzero regression, malformed late
inputs and no-overwrite. Runner fault-injection tests cover cancellation and
report publication failure. Documentation
links and existing test suites must pass. This batch closes only the explicitly
named offline usability/runner subset, not all B09/B10, real provider evaluation,
external-user feedback, model ranking or stable release qualification.
