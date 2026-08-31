# ADR-0019: Durable Local Episode Journal

- **Status:** Accepted
- **Date:** 2026-08-26
- **Accepts:** SPEC-0017 and the local persistence/idempotency subset of ST-EPISODE-004
- **Amends:** RFC-0003 sections 5, 7, 8 and ADR-0013 resource profile
- **Stable v0.1 impact:** none; v0.2 development preview only

## Context

ADR-0018 made local scripted Episodes reproducible but ephemeral. Process exit
discarded lifecycle progress, a repeated Episode ID could execute again, and
operators had no durable inspection point. Reusing the world-state SQLite
schema would couple a v0.2 orchestration concern to the accepted v0.1 storage
profile and complicate rollback.

Remote host cancellation and commit ambiguity are not yet specified well
enough to support automatic recovery. A journal may preserve uncertainty; it
must not invent a safe retry.

## Decision

Add an optional, independent SQLite Episode Journal:

1. the journal has its own application ID (`EPJL`) and schema version `1`;
2. one Episode ID is permanently bound to a request digest over Bundle,
   Scenario and runtime identity;
3. creation and every lifecycle transition are transactional;
4. updates use status/sequence compare-and-swap;
5. terminal Scenario evidence and its terminal lifecycle event commit in one
   transaction;
6. repeating an identical completed request returns the persisted Evidence
   without executing the Scenario again;
7. repeating an ID with a different request fails `episode identity conflict`;
8. a non-terminal existing record is returned as `incomplete` and is never
   silently retried; and
9. runtime failures persist only a typed `RUNTIME_ERROR`, not raw error text.

The CLI opt-in is:

```text
statetwin episode run ... --journal episodes.db
statetwin episode inspect --journal episodes.db --id episode-001
```

Omitting `--journal` preserves the ADR-0018 one-shot behavior.

## Journal record contract

`statetwin.dev/episode-journal/v1alpha1` records:

- immutable request and Bundle digests;
- selected Scenario and runtime version/revision;
- current status and monotonic sequence;
- the complete ordered lifecycle;
- `incomplete` derived from terminality;
- terminal outcome and Evidence, when available;
- canonical Evidence digest; and
- operational creation/update timestamps.

Journal timestamps are observational metadata. They are not added to canonical
EpisodeEvidence and do not alter deterministic evaluation identity.

## Safety and recovery semantics

- A foreign/unidentified non-empty SQLite database and a future schema are
  refused.
- At most 10,000 Episode records are admitted by `local-preview-v3`.
- Persisted Evidence is bounded by the existing report limit and its digest is
  revalidated on read.
- A terminal status without valid Evidence is not reusable.
- An interrupted non-terminal record remains inspectable. Operators must use a
  new Episode ID for a new attempt until a future recovery ADR defines attempt
  lineage and commit-state semantics.
- Journal persistence performs no network or provider call.

## Explicit exclusions

This ADR does not accept:

- automatic retry or resume;
- cancellation semantics;
- process lease/heartbeat or abandoned-run timeout;
- remote workers or `/control/v1/episodes`;
- provider thread identity;
- cross-process work claiming/distributed locks;
- Episode deletion/retention automation;
- encryption, multi-tenancy or remote authentication; or
- a claim of exactly-once model/tool execution.

Those require separate failure and kill-point matrices. In particular,
`ST-EPISODE-005` remains blocked.

## Consequences

- The resource profile advances to `local-preview-v3`.
- Existing world databases and storage schema v4 are unchanged.
- Existing one-shot commands remain compatible.
- Durable local evidence can be inspected after process restart without
  overstating crash recovery.

## Amendment: ADR-0020 (2026-08-26)

ADR-0020 accepts a separate schema-v2 coordinator profile and supersedes the
retry, cancellation, lease, remote-worker and `ST-EPISODE-005` exclusions above
only for that bounded profile. The one-shot local Journal behavior remains
compatible. Arbitrary external exactly-once execution, retention, multi-tenant
HA and provider-thread recovery remain excluded.
