# SPEC-0012: Storage, Concurrency, Migration, and Recovery

- **Status:** Proposed; local storage subset implemented
- **Implementation status:** partial
- **Verification status:** schema migration, head monotonicity, and branch isolation are tested
- **Source:** `MCP-State-Twin-Lifecycle-SPEC-Pack-vNext/08-SPEC-0012...`

## 1. Supported profile

The current profile is one host, one SQLite database, multiple logical
branches, serialized SQLite writes, and branch-local semantic ordering. It does
not claim distributed consensus, multi-region replication, network-filesystem
SQLite, or remote multi-tenancy.

## 2. Branch head contract

Each branch stores:

```text
branch_id
head_version        monotonic committed branch version
call_count          user-visible call index; reset may rewind it
state_digest
world_time
```

Every committed tool call and privileged clock/reset mutation updates
`head_version` with a compare-and-swap predicate. A stale update returns the
typed `BRANCH_CONFLICT` condition and cannot silently merge or recompute work.
Business-domain `CONFLICT` remains a separate error class.

## 3. Schema v3 migration

SQLite `user_version` is now `3`. Opening an older supported database adds
`branches.head_version`, `snapshots.source_head_version`, and any missing
legacy columns inside the migration transaction. A database with a newer
schema or foreign application ID is rejected. Historical snapshots without a
source head version are retained with `0` and are not retroactively promoted to
strong concurrency evidence.

New snapshot identity includes the source branch head version. A fork starts a
new branch at head `0`; reset preserves monotonic head history while rewinding
the call index and world state to the selected snapshot.

## 4. Recovery boundary

SQLite transaction atomicity protects normal transitions and control audits.
Crash kill-point fixtures, interrupted migration recovery, disk-full behavior,
WAL checkpoint policy, and backup/restore verification remain open gates. The
project MUST NOT describe the local preview as HA or crash-proof until those
fixtures exist.
