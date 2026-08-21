# SPEC-0012 — Storage, Concurrency, Migration & Crash Recovery

> **Status:** Proposal  
> **Specification status:** draft  
> **Implementation baseline:** source-reported partial foundation  
> **Verification status:** unverified for this proposal  
> **Primary profile:** local / single-host SQLite  
> **Target:** deterministic-runtime completion before stable v1.0

## 1. Abstract

This specification defines the storage and concurrency semantics that allow MCP State Twin to remain reproducible when branches are read, mutated, forked, reset, diffed, migrated, interrupted, or accessed concurrently.

The primary pre-1.0 storage profile remains a **single-host SQLite database**. This is a deliberate product boundary, not an accidental implementation detail.

## 2. Goals

- Atomic world transitions.
- Explicit and replayable branch concurrency semantics.
- Immutable snapshot semantics.
- Defined crash/restart behavior.
- Versioned storage identity.
- Safe migrations.
- No dependence on goroutine scheduling or database lock race order.
- Clear separation between domain failures and storage/infrastructure failures.

## 3. Non-goals

This SPEC does not define:
- multi-region replication;
- distributed consensus;
- active-active writers;
- network-filesystem SQLite;
- remote multi-tenant database architecture;
- HA guarantees.

Those require a separate RFC.

## 4. Storage profile

The initial supported profile is:

```text
single host
single logical State Twin database
multiple branches
multiple concurrent readers
serialized SQLite writer
explicit per-branch optimistic head version
```

SQLite's serialization does not by itself define State Twin's semantic ordering. State Twin must make conflicts explicit.

## 5. Database identity

### ST-STORE-R001
Every database MUST carry a State Twin application identity and explicit storage schema version.

### ST-STORE-R002
A runtime MUST reject a database whose application identity is not State Twin's.

### ST-STORE-R003
A runtime MUST reject a storage schema newer than the maximum schema it can safely interpret.

### ST-STORE-R004
Opening an older supported schema MAY trigger an explicit migration workflow; it MUST NOT silently reinterpret rows under new semantics.

## 6. Branch head model

Each mutable branch MUST have:

```yaml
branch_id: "<stable logical id>"
head_version: <monotonic integer>
state_digest: "sha256:<...>"
world_time: "<virtual time>"
```

### ST-STORE-R010
`head_version` MUST increase monotonically for every committed semantic branch mutation.

### ST-STORE-R011
A state-changing operation MUST read the current branch head version before computing the commit.

### ST-STORE-R012
Unless a different serialization profile is explicitly specified, the operation MUST commit only if the branch head still equals the expected version.

### ST-STORE-R013
If the head changed, the runtime MUST return a typed conflict such as `BRANCH_CONFLICT` and MUST NOT silently recompute/merge the operation.

### ST-STORE-R014
Conflict results MUST be distinguishable from modeled business-domain `CONFLICT`.

## 7. Concurrent reads and writes

### Same branch

Two writes racing on the same branch must resolve through the branch-head contract, not whichever goroutine reaches SQLite first.

### Different branches

Different branches MAY execute concurrently, subject to SQLite's physical writer serialization.

The semantic contract is branch-local; storage scheduling must not cause cross-branch state contamination.

### ST-STORE-R020
A read MUST observe one well-defined branch head/state snapshot.

### ST-STORE-R021
A read MUST NOT observe partially applied effects.

### ST-STORE-R022
Cross-branch writes MUST preserve branch isolation even if physical commits are serialized.

### ST-STORE-R023
Audit evidence SHOULD include both a branch-local semantic sequence and a database/global audit sequence where needed for forensics.

## 8. Snapshot semantics

A snapshot is a semantic immutable artifact, not merely "a copied SQLite file".

A snapshot SHOULD bind:

```yaml
snapshot:
  id: "<stable id>"
  source_branch: "<id>"
  source_head_version: <n>
  state_digest: "sha256:<...>"
  storage_schema_version: "<...>"
  canonicalization_id: "<...>"
  world_time: "<...>"
  scheduler_state_digest: null
  entropy_state_digest: null
```

Scheduler/entropy fields become required once SPEC-0007 is implemented.

### ST-STORE-R030
Snapshot contents MUST be immutable after successful creation.

### ST-STORE-R031
Forking a snapshot MUST create a logically equivalent initial branch world.

### ST-STORE-R032
Snapshot identity MUST change if a component included in its semantic identity changes.

### ST-STORE-R033
A snapshot MUST NOT reference mutable external state required to reproduce the world unless that reference is explicitly part of a non-hermetic profile.

## 9. Snapshot/write race

The implementation MUST define exactly which branch head a concurrent snapshot captures.

Recommended semantics:

1. snapshot transaction resolves a committed branch head;
2. snapshot binds exactly that head;
3. a concurrent later write may produce a newer head but does not alter the snapshot.

## 10. Reset semantics

### ST-STORE-R040
Reset is a privileged control-plane mutation.

### ST-STORE-R041
Reset MUST atomically install the selected snapshot state and update branch head metadata.

### ST-STORE-R042
Reset racing with an ordinary branch write MUST use the same explicit head/version conflict model.

### ST-STORE-R043
Reset MUST produce control-audit evidence.

## 11. Fork semantics

### ST-STORE-R050
Fork MUST use an immutable snapshot as its source under the default profile.

### ST-STORE-R051
A fork MUST start with the snapshot's world state, time, scheduler state and deterministic entropy state.

### ST-STORE-R052
Future mutations on the fork MUST not affect the snapshot or sibling branches.

### ST-STORE-R053
A fork ID/branch ID MUST not be treated as an authentication credential.

## 12. Diff semantics

### ST-STORE-R060
Diff compares two explicit committed semantic states.

### ST-STORE-R061
Diff MUST state the canonicalization/diff schema version used.

### ST-STORE-R062
Diff output MUST be bounded by SPEC-0015 limits.

### ST-STORE-R063
If two states use incompatible canonicalization/storage semantics, the runtime MUST reject or explicitly migrate them; it MUST NOT silently compare incompatible representations.

## 13. SQLite/WAL policy

The SQLite primary source material establishes:
- writes are serialized;
- WAL allows readers to coexist with a writer;
- there is still one writer;
- WAL is a same-host design and is not a network-filesystem distributed database;
- checkpoint starvation/WAL growth are operational concerns.

Therefore:

### ST-STORE-R070
WAL mode MAY be used only under a supported same-host filesystem profile.

### ST-STORE-R071
A shared network filesystem MUST NOT be advertised as a supported distributed SQLite/WAL deployment.

### ST-STORE-R072
The runtime MUST define checkpoint policy and operational warning thresholds.

### ST-STORE-R073
Long-lived readers that prevent checkpoint progress MUST be detectable.

### ST-STORE-R074
The live `-wal`/`-shm` relationship MUST be respected by backup/copy operations.

## 14. Crash model

Crash testing is defined through **explicit killpoints** rather than pretending to model every OS/hardware failure.

Required killpoints:

```text
K0 before transaction
K1 after branch read
K2 after validation
K3 after effects in working state
K4 immediately before commit
K5 immediately after commit
K6 after audit commit but before response
K7 during snapshot creation
K8 during reset
K9 during migration
```

Each killpoint must specify:
- whether world state is committed;
- whether audit is committed;
- what retry sees;
- what recovery action is required.

## 15. Crash/restart invariants

### ST-STORE-R080
A crash before a successful commit MUST NOT expose partial world effects.

### ST-STORE-R081
A crash immediately after commit MAY yield an ambiguous client outcome, but restart MUST expose the committed world state.

### ST-STORE-R082
State and mandatory audit records that are contractually atomic MUST recover together.

### ST-STORE-R083
Crash recovery MUST NOT fabricate a successful tool response that was never durably recorded.

## 16. Operational failures

Explicitly classify:

- disk full;
- read-only filesystem;
- permission denied;
- corrupt database;
- unsupported schema;
- failed checkpoint;
- I/O error;
- lock timeout/busy;
- migration failure.

### ST-STORE-R090
These failures MUST be classified as storage/operational failures, not normal domain tool errors.

### ST-STORE-R091
The prior committed world state MUST remain the recovery reference unless the storage engine reports corruption that prevents trustworthy interpretation.

## 17. Migration lifecycle

Every storage migration requires:

```text
from_schema
to_schema
preconditions
migration procedure
post-migration validation
failure behavior
backup/recovery policy
compatibility impact
```

### ST-STORE-R100
A destructive migration MUST NOT be automatic without an explicit release policy.

### ST-STORE-R101
Migration MUST validate the final database identity/version.

### ST-STORE-R102
Interrupted migration behavior MUST be tested.

### ST-STORE-R103
Downgrade is unsupported by default unless explicitly implemented and tested.

### ST-STORE-R104
Snapshots must declare their storage schema so the runtime can determine compatibility.

## 18. Backup and restore

Before v1.0 define:
- supported backup method;
- WAL-safe backup process;
- restore verification;
- digest verification;
- rollback procedure for failed upgrades.

## 19. Required tests

- two writers same branch;
- writers different branches;
- read during write;
- snapshot during write;
- fork while sibling branch writes;
- reset/write race;
- diff/write race;
- crash killpoints K0–K9;
- disk full;
- permission failure;
- corrupt DB;
- foreign DB;
- future schema;
- interrupted migration;
- WAL checkpoint stress;
- 100+ branches isolation;
- large-state boundaries.

## 20. Feasibility

**High for the intended local/single-host profile.**

Remote multi-node semantics should not be added incrementally to this SPEC. They change the architecture enough to require a separate product RFC.
