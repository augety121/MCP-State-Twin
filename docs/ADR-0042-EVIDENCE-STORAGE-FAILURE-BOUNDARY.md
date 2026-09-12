# ADR-0042: Evidence Storage Failure Boundary

- Status: Accepted for the experimental Agent evidence lanes
- Date: 2026-09-12
- Basis: maintainer request to continue implementation; AE-014/019/020
- Scope: SPEC-0039/0041 artifacts; no world/Journal schema migration

## Decision

Accept [SPEC-0042](SPEC-0042-EVIDENCE-STORAGE-AND-TERMINAL-FAILURES.md).
Separate filesystem operations behind instance-local, package-private interfaces
so each write/sync/close/publication failure can be tested without global hooks,
real disk exhaustion, or production configuration switches.

Retain exclusive claim, replay closure before world disposal and no-overwrite
hard-link publication. A failed/ambiguous operation does not permit resume.
Validate the complete serializable artifact before opening its file, including
known sensitive-pattern refusal and short-write detection.

Use the independent terminal context for both terminal grading and replay
closure. Preserve the original execution failure; add optional
`terminalFailureCode` and `cleanupFailureCode` for subsequent failures.
The two fields are additive experimental-format extensions: old artifacts stay
readable; older strict readers may reject newly enriched failure reports. There
is no stable format or cross-version source-authenticity claim.

## Evidence and exclusions

Unit fault injection covers 22 filesystem outcomes; five subprocess exits
exercise actual files after synchronization/publication steps. These do not
prove hardware power-loss durability, all-filesystem behavior, interruptible
filesystem syscalls, remote cancellation, automatic recovery or exactly-once
external effects. Existing SQLite world schema 4 and Journal schema 2 do not change.
