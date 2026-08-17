# ADR-0005: Canonical JSON Digest Boundary

- **Status:** Accepted and implemented for the development preview
- **Date:** 2026-08-17

## Context

Deterministic replay, snapshot identity, drift reporting, and branch comparison
need stable digests that do not depend on Go map insertion order or YAML source
formatting.

## Decision

The alpha implementation normalizes values through Go's JSON encoder/decoder,
preserves JSON numbers with `json.Number`, relies on sorted string map keys,
and hashes compact normalized JSON with SHA-256.

The supported canonical value domain is JSON plus Go structs with explicit JSON
tags. Functions, channels, non-string object keys, NaN, and infinities are not
valid artifacts.

## Consequences

- semantically equivalent map insertion orders produce the same digest;
- source YAML comments and formatting do not affect the digest;
- this is an alpha project canonicalization contract, not a claim of full
  RFC 8785 conformance;
- golden digest tests must gate any serializer or Go-version change;
- a future incompatible canonicalization rule requires an artifact-format
  version bump and migration design.
