# SPEC-0012: Storage, Concurrency, Migration, and Recovery

- **Status:** Local v0.1 storage subset accepted by ADR-0016
- **Implementation status:** accepted single-process SQLite profile implemented
- **Verification status:** tagged-schema reopen, v1/v2/v3 migration, process-exit recovery, head monotonicity, and branch isolation are tested
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

Every committed tool call and privileged clock/reset/fault-configuration mutation updates
`head_version` with a compare-and-swap predicate. A stale update returns the
typed `BRANCH_CONFLICT` condition and cannot silently merge or recompute work.
Business-domain `CONFLICT` remains a separate error class.

## 3. Schema v4 migration

SQLite `user_version` is now `4`. Opening an older supported database adds
`branches.head_version`, `snapshots.source_head_version`, and any missing
legacy columns inside the migration transaction. A database with a newer
schema or foreign application ID is rejected. Historical snapshots without a
source head version are retained with `0` and are not retroactively promoted to
strong concurrency evidence.

New snapshot identity includes the source branch head version. A fork starts a
new branch at head `0`; reset preserves monotonic head history while rewinding
the call index and world state to the selected snapshot.

Schema v4 adds branch-local `fault_plans` and append-only `fault_events`.
Consumption of a fault counter and insertion of its event occur in the same
transaction as the affected call. Fault configuration is not silently copied
through snapshot/fork operations.

## 4. Recovery boundary

SQLite transaction atomicity protects normal transitions and control audits.
Subprocess kill-points after schema mutation and metadata mutation verify that
an interrupted migration can be reopened, migrated, and pass SQLite integrity
checking. The public alpha schema-v4 fixture and historical v1/v2/v3 migrations
are covered by executable tests.

ADR-0016 explicitly excludes disk-full behavior, network-filesystem SQLite,
multi-process writers, WAL checkpoint tuning guarantees, online backup/restore,
replication, and HA from the stable v0.1 storage profile. The project MUST NOT
describe the local runtime as HA, disaster-recovery capable, or generally
crash-proof.
