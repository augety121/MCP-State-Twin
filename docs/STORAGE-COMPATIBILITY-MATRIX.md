# Storage Compatibility Matrix

**Status:** executable evidence ledger
**Last reviewed:** 2026-09-12 (artifact appendix; SQLite profiles unchanged)
**Authority:** ADR-0016 for the world store; ADR-0019 and ADR-0020 for the
Episode Journal

MCP State Twin owns two independent SQLite formats. Compatibility for one must
never be inferred for the other.

| Database | Application ID | Current schema | Accepted input | Migration evidence |
|---|---:|---:|---|---|
| World store | `0x5354574e` (`STWN`) | 4 | empty/new, schema 1, 2, 3, or 4 | generated v1/v2 migration tests, `schema-v3-pre-fault.sql`, tagged `v0.1.0-alpha.1-schema-v4.sql`, two process-exit kill-points |
| Episode Journal | `0x45504a4c` (`EPJL`) | 2 | empty/new, schema 1 or 2 | published-commit fixture `b8e836b-journal-v1.sql`, preservation test, two process-exit kill-points and `PRAGMA integrity_check` |

## Admission invariants

1. Identity and `user_version` are read before persistent pragmas or migration.
2. A foreign application ID, unidentified non-empty SQLite file, inconsistent
   zero-ID/non-zero-version file, or future schema is refused.
3. Refusal must not set application identity, schema version, or WAL mode.
4. Forward migration is transactional. A process exit before commit must leave
   either the old valid schema or the complete new schema, never a partial
   accepted schema.
5. World-store migrations preserve branches, snapshots, digests, virtual time,
   audit records and the defined compatibility defaults.
6. Journal v1 → v2 preserves parent Episodes/events byte-semantically while
   adding empty task/attempt tables. Existing local one-shot Episode behavior
   remains valid.
7. Every task read cross-validates parent, attempt count/order, timestamps,
   fencing lineage, active/terminal identity and final Evidence digest.

## Explicit non-claims

This matrix does not promise downgrade, arbitrary future-schema import,
cross-database conversion, online multi-writer migration, backup/restore,
replication, disk-full recovery, encryption, retention/GC or HA. Those remain
separate requirements and cannot be inferred from local forward-migration
tests.

## Reproduce

```bash
go test ./internal/store -run 'Test.*(Migration|Schema|Foreign|Reopen)' -count=1
go test ./internal/episode -run 'Test.*(Journal|Task|Migration)' -count=1
go test -race ./internal/store ./internal/episode
```

Fixtures are synthetic and contain no credentials, private traces or personal
data. Adding or changing a schema requires a provenance-labelled fixture,
positive preservation assertions, refusal tests, interrupted-migration tests,
this matrix, the changelog and release evidence in the same change.

## Independent Agent artifact files (not a SQLite schema)

ADR/SPEC-0042–0043 cover the separate claim/closure/pending/terminal JSON
directory used by offline and opt-in local-API Agent runs. Neither world schema
4 nor Journal schema 2 changes in this increment.

| Boundary | Executable evidence | Does not establish |
|---|---|---|
| Exclusive claim and stage creation | existing-directory/no-clobber refusals, 22 injected failures including short writes and `ENOSPC` | actual full-volume behavior, disk/controller failures or concurrent hostile writers |
| Publication and residue | link/cleanup failures before and after modeled effect; terminal preservation; read-only diagnosis | overwrite/copy fallback, automatic repair, retention/GC |
| Process interruption | five test child processes exit after claim sync, closure sync, pending sync, publication link and pending removal | hardware power-loss durability or a filesystem-wide atomic snapshot |
| Artifact compatibility | existing readers accept historical optional-field absence; new failure reports add `terminalFailureCode` / `cleanupFailureCode` | stable wire compatibility with older strict readers, artifact migration or schema downgrade |

Run `go test -p 1 ./internal/agenteval` for this bounded evidence. Replaying a
published artifact checks its modeled world, not that all writes survive power
loss or that the file originated from a particular Provider. Existing directories
remain inspect-only, including after failed publication; no automatic execution
retry is introduced. Full failure details are in
[SPEC-0042](SPEC-0042-EVIDENCE-STORAGE-AND-TERMINAL-FAILURES.md).
