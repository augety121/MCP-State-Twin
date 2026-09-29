# SPEC-0061: Replay-backed Suite Report Verification

Status: accepted by [ADR-0059](ADR-0059-SUITE-EVIDENCE-AUDIT.md).

## Contract

Recompute comparison from the admitted saved plan and terminal evidence using
the existing Compare replay path. Output the recomputed comparison, never trust
the saved aggregate for gating. All planned samples remain in the denominator.
Preserve the distinction between evidence validity and successful task grading.

If a saved report claims a complete comparison it must exactly match recomputed
semantic values. If both pending and published reports exist, their admitted
semantic values must match. Any discrepancy => invalid. Pending alone never
counts as published. Trial directory claims must match the expected offline
trial/model identity; Compare additionally binds Task ID.

A completed saved execution must agree with every independently verified terminal
status and outcome. A complete report with missing/corrupt evidence is invalid.
Mark reportVerification matched only for a published, completed report whose
comparison, trial rows, claims and complete evidence agree. Partial/canceled/stopped
reports can be structurally valid but remain unverified; observed artifacts do
not prove the historical reason execution stopped.

State is published or published_with_residue only for a matched report; otherwise
published_unverified, incomplete_or_running or invalid. regressionGatePassed is
true only for state published, matched report and no_regression_observed. Residue
blocks this new clean-publication gate. upgradeAllowed is always false. A valid
regression report is matched but does not pass the regression gate.

## Acceptance

Detect changed summary counts/verdicts, deleted terminal files, changed trial
outcomes/identity, conflicting staging and forged success reports. Match normal
and regressed completed runs; never promote partial or missing reports. Rereading
must not alter files or call providers; cancellation/resource failure returns error.
