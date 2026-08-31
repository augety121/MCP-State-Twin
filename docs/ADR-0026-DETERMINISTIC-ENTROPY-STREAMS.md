# ADR-0026: Deterministic Modeled-World Entropy Streams

- **Status:** Accepted
- **Date:** 2026-08-31
- **Related:** SPEC-0002, SPEC-0007, SPEC-0012, SPEC-0015, SPEC-0026

## Context

Modeled identifiers sometimes need values that look non-sequential while still
remaining reproducible across replay, snapshot and fork. Reading the operating
system RNG, process-global randomness, database-generated random IDs or model
output would make the world transition depend on an undeclared input.

## Decision

Accept one bounded modeled-world entropy profile, `sha256-ctr-v1`. A TwinSpec
opts in with a public synthetic 256-bit seed. An `entropy` effect selects a
named stream and emits 1 through 32 bytes encoded as lowercase hexadecimal.
The stream counter is part of canonical branch state and advances only when the
containing business transition commits.

The exact derivation and input encoding are normative in SPEC-0026. Snapshot,
fork, reset and canonical state digest therefore include stream position
without adding a new storage table or schema migration.

## Consequences

- identical accepted inputs produce identical modeled values;
- sibling forks begin from the same stream position and then diverge locally;
- failed transitions cannot consume entropy invisibly;
- a changed algorithm or seed changes TwinSpec/environment identity;
- the feature is not suitable for credentials, signing keys, session tokens or
  any security decision.

Host randomness, model sampling and cryptographic RNG APIs remain outside this
decision.
