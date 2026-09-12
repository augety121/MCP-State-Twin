# ADR-0046: Full candidate gates before draft creation

Status: Accepted, 2026-09-12. The former tag workflow repeated only part of CI.

Make the existing `ci.yml` reusable through `workflow_call`. Release admission
checks the tag, matching checked-in plan, current commit and main ancestry, then
calls `./.github/workflows/ci.yml` at the same workflow commit. Draft staging
depends on both admission and the entire reusable gate job. No older green badge,
manual exception, `always()` or continue-on-error may bypass it.

Read-only token permissions are the default. Only the final draft staging job
gets contents:write. It rechecks the remote tag before attachment. All releases
remain draft and not-latest; prerelease tags set the actual prerelease flag.
The workflow never creates missing tags or automatically publishes drafts.

Main CI tests the source contract. A real tag-triggered publication remains
unexecuted until separately authorized; do not confuse those evidence levels.
See SPEC-0046.
