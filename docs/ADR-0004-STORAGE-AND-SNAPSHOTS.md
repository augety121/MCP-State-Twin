# ADR-0004: SQLite State Store and Logical Snapshots

- **Status:** Accepted and implemented for the development preview
- **Date:** 2026-08-17

## Context

The first implementation needs atomic tool transitions, crash-safe persisted
state, portable local development, and deterministic branches. Copy-on-write
page stores and event sourcing would add complexity before reference workload
measurements exist.

## Decision

- Use pure-Go SQLite (`modernc.org/sqlite`).
- Serialize normal state transitions through one database connection.
- Store canonical JSON branch heads and immutable logical snapshots.
- Derive snapshot IDs from snapshot name, spec digest, state digest, and virtual
  clock.
- Fork by copying the immutable snapshot root into a new branch.
- Treat the internal SQL schema as private; canonical JSON is the future
  portability boundary.

## Consequences

- transition atomicity is delegated to SQLite transactions;
- the CLI remains a single Go binary without a C toolchain requirement;
- large snapshots are not storage-efficient yet;
- copy-on-write, retention, and GC require measurements and a later ADR;
- changing internal tables does not imply a public snapshot-format guarantee.
