# SPEC-0017: Durable Local Episode Journal

- **Status:** Accepted local subset via ADR-0019
- **Format:** `statetwin.dev/episode-journal/v1alpha1`
- **Storage schema:** SQLite application ID `EPJL`, version `1`
- **Profile:** local, single-process, synthetic scripted Episodes
- **Stable v0.1 impact:** none

## 1. Purpose

The Episode Journal is an optional durable control record for local scripted
evaluation. It answers:

- which immutable request owns an Episode ID;
- which lifecycle transition was last committed;
- whether the record is terminal or incomplete; and
- which canonical Evidence belongs to a completed Episode.

It is not a workflow engine, distributed queue, scheduler, provider thread
store or exactly-once execution system.

## 2. Normative terms

`MUST`, `MUST NOT`, `SHOULD` and `MAY` use RFC 2119/RFC 8174 meanings.

An **Episode request** is:

```text
EpisodeRequest = (
  episode_id,
  TwinBundle_digest,
  selected_Scenario_path,
  runtime_version,
  runtime_revision
)
```

An **incomplete record** has a non-terminal lifecycle status. Incomplete does
not mean failed, retryable, abandoned or safe to resume.

## 3. Identity

### ST-JOURNAL-R001

The request digest MUST be canonical SHA-256 over all EpisodeRequest fields and
the Journal format identifier.

### ST-JOURNAL-R002

One Episode ID MUST bind to one request digest for the lifetime of a Journal.
The same ID with a different digest MUST fail before execution.

### ST-JOURNAL-R003

Runtime revision is part of identity. Reusing an Episode ID after rebuilding or
upgrading the runtime therefore conflicts unless the immutable revision is
unchanged.

### ST-JOURNAL-R004

An identical request MAY reuse Evidence only when the stored status is
`SUCCEEDED` or `ASSERTION_FAILED` and the Evidence passes admission.

## 4. Storage identity and migration

### ST-JOURNAL-R010

The Journal MUST use an application identity distinct from the world-state
database. It MUST NOT add tables or migrations to the v0.1 world schema.

### ST-JOURNAL-R011

A non-zero foreign application ID, an unidentified non-empty SQLite database,
or a schema version newer than supported MUST be rejected before Journal
writes or persistent journal-mode changes.

### ST-JOURNAL-R012

Schema creation and application/version identity MUST commit atomically.

### ST-JOURNAL-R013

Opening an older recognized schema requires an explicit forward migration.
There is no best-effort reinterpretation. v1alpha1 currently has no older
recognized Journal schema.

## 5. Record schema

The public inspection record contains:

```text
apiVersion
kind = EpisodeRecord
format
storageSchemaVersion
episodeId
requestDigest
bundleDigest
scenarioPath
runtime.version
runtime.revision
status
sequence
incomplete
outcome?
evidenceDigest?
evidence?
errorClass?
createdAt
updatedAt
events[]
```

`createdAt` and `updatedAt` are operational wall-clock metadata. They MUST NOT
be added to canonical EpisodeEvidence or its digest.

Raw runtime error text MUST NOT be stored. The current accepted failure record
stores only `RUNTIME_ERROR`.

## 6. Lifecycle persistence

Accepted lifecycle:

```text
CREATED -> PROVISIONING -> READY -> RUNNING -> EVALUATING
                                              |-> SUCCEEDED
                                              |-> ASSERTION_FAILED

PROVISIONING/RUNNING/EVALUATING -> RUNTIME_ERROR
```

`CANCELLED` exists in the general lifecycle model but has no accepted durable
CLI transition in ADR-0019.

### ST-JOURNAL-R020

Creation MUST atomically insert the Episode record and sequence `0` `CREATED`
event.

### ST-JOURNAL-R021

Every later transition MUST compare-and-swap the expected prior status and
sequence. Sequence MUST begin at zero and increase by exactly one.

### ST-JOURNAL-R022

The event chain MUST be contiguous: each event's `from` equals the preceding
event's `to`, and every edge is allowed by the lifecycle state machine.

### ST-JOURNAL-R023

The terminal event, terminal record status, outcome, encoded Evidence and
Evidence digest MUST commit in one transaction.

### ST-JOURNAL-R024

`incomplete` is derived as `!terminal(status)` and MUST NOT be an independently
mutable storage field.

## 7. Idempotency and retry matrix

| Existing record | Incoming request | Required result |
|---|---|---|
| none | valid | reserve and execute |
| terminal success/assertion failure with valid Evidence | identical digest | return stored Evidence, do not execute |
| terminal runtime error/cancelled | identical digest | fail as terminal without reusable Evidence |
| non-terminal | identical digest | fail `episode is incomplete`, do not execute |
| any | different digest | fail `episode identity conflict`, do not execute |

This is idempotent **completed-result lookup**, not proof of exactly-once
Scenario, model or tool execution.

## 8. Crash and ambiguity matrix

| Crash point | Durable observation after reopen | Automatic action |
|---|---|---|
| before reservation commit | no record | none |
| after reservation, before first transition | `CREATED`, incomplete | none |
| between non-terminal transitions | last committed status/sequence | none |
| while Scenario executes | normally `RUNNING`, incomplete | none |
| after evaluation transition, before terminal commit | `EVALUATING`, incomplete | none |
| during terminal transaction | either prior incomplete state or complete terminal Evidence | none |
| after terminal commit | terminal status and valid Evidence | identical request may read Evidence |

The Journal MUST NOT infer whether an external provider action committed. A
future retry/resume design requires attempt lineage, leases, heartbeat expiry,
world/provider commit classification and kill-point evidence.

## 9. Read admission

Every `episode inspect` and completed-result reuse MUST validate:

1. request digest recomputed from stored identity fields;
2. RFC3339Nano operational timestamps with `updatedAt >= createdAt`;
3. event count equals `sequence + 1`;
4. exact contiguous event numbering and legal transition chain;
5. last event status equals the record status;
6. Evidence envelope presence only for successful/assertion-failed terminal
   statuses;
7. envelope digest field equals the record digest;
8. canonical Evidence digest recomputes exactly;
9. Evidence ID, Bundle, Scenario, Runtime, outcome and format fields match the
   Journal record exactly; and
10. Evidence lifecycle equals the Journal lifecycle.

Unknown, missing, malformed or mismatched state MUST fail explicitly. The
reader MUST NOT repair or normalize corrupted evidence.

## 10. Resource and concurrency profile

- Maximum Episode records: 10,000.
- Maximum terminal Evidence bytes: existing `MaxReportBytes`.
- SQLite connections: one active connection for the local serial profile.
- Writer conflicts: status/sequence compare-and-swap.
- Distributed claiming and fairness: unsupported.
- Retention/GC and disk quota: unsupported and operator-managed.

The record-count bound limits rows, not total disk usage. Operators MUST apply
filesystem quotas externally where required.

## 11. Security and privacy

- Journal files are local and unencrypted.
- They may contain complete synthetic tool inputs/results inside Evidence.
- Raw runtime error messages are not persisted.
- Journal files MUST NOT contain production traces, credentials or personal
  data.
- A Journal digest is not a signature or tamper-proof audit guarantee.
- Opening a Journal performs no network access.

## 12. CLI

```text
statetwin episode run \
  --bundle twin.stb \
  --id episode-001 \
  --journal episodes.db \
  [--scenario path] \
  [--out evidence.json]

statetwin episode inspect \
  --journal episodes.db \
  --id episode-001
```

`--journal` remains optional. `episode inspect` is read-only at the logical
record level; opening SQLite may perform normal local SQLite housekeeping.

## 13. Required executable evidence

- complete run, close/reopen and inspect;
- identical completed request returns the same Evidence without record change;
- same ID/different request conflict;
- incomplete request refusal without lifecycle mutation;
- runtime error stores typed class without raw error details;
- Evidence, request and lifecycle tamper refusal;
- foreign/unidentified and future-schema refusal;
- record resource bound configuration;
- CLI smoke in normal, hermetic and release workflows; and
- race testing on supported Linux CI.

## 14. Deferred promotion gates

Before automatic retry, cancellation or remote workers can be accepted, a new
ADR MUST define:

- attempt and parent-Episode identity;
- ownership lease and fencing token;
- heartbeat/abandonment semantics;
- cancel-before-effect, cancel-after-effect and unknown-commit outcomes;
- provider thread and world branch correlation;
- credential lifecycle and cleanup;
- retry budgets and backoff under virtual versus wall time;
- multi-process crash/kill-point tests; and
- retention/deletion/audit policy.

## 15. Accepted coordinator amendment

ADR-0020 and SPEC-0018 satisfy the attempt identity, lease, fencing,
cancellation, bounded retry, credential and commit-ambiguity gates for a
single-coordinator synthetic TwinBundle profile. Journal schema v2 adds task
and attempt tables through a forward migration that preserves schema-v1
Episode data. Retention/deletion, multi-coordinator HA, provider-thread
recovery and exactly-once external effects remain unaccepted.
