# ADR-0007: SQLite Identity, Schema Version, and Control Audit

- **Status:** Accepted
- **Date:** 2026-08-17

## Context

SQLite files previously had no application identity or schema version. A newer
runtime could misinterpret an older file, and an accidentally supplied foreign
SQLite file could receive State Twin tables. Privileged snapshot/fork/reset
mutations also lacked durable audit evidence.

## Decision

- Set SQLite `application_id` to `0x5354574E` (`STWN`).
- Set and validate `user_version`; the current schema version is `2`.
- Persist `storage_schema_version` on every snapshot and bind it into new
  snapshot IDs.
- Reject non-zero foreign application IDs.
- Reject database versions newer than the runtime.
- Execute schema upgrades transactionally.
- Preserve old tool-call audit rows during migration.
- Write snapshot, fork, and reset control-audit rows in the same transaction as
  their state mutation.
- Never persist bearer tokens or request headers in audit rows.

The control-audit minimum is operation, branch, snapshot, before digest, after
digest, and operational timestamp.

## Consequences

- Database files fail closed on identity/version mismatch.
- Reset can reuse a logical call index while append-only audit history remains.
- Operational audit timestamps are intentionally outside simulated deterministic
  state.
- Every future tagged schema version needs forward migration and refusal tests.
