# ADR-0037: Preserve CEL Null at the JSON Boundary

- **Status:** Accepted
- **Date:** 2026-09-02
- **Related:** ADR-0003, ADR-0005, ADR-0036, SPEC-0037

## Decision

Normalize CEL's native protobuf null enum to Go nil recursively, and refuse
unknown null enum values. Preserve normal numeric zero and missing-key
distinctions. Gate with literal/nested/null-schema/persisted-state tests.

## Compatibility

This fixes a defect reproduced on the previous dependency, independently of
the CEL module migration. Result/state digests may differ for affected
trajectories. Existing snapshots and Evidence are immutable, and stored zero
is never guessed to be historical null. No canonical-format or storage-layout
change is introduced; runtime revision distinguishes the corrected semantics.
