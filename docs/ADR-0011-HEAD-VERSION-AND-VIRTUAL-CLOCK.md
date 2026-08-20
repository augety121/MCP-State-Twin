# ADR-0011: Monotonic Branch Head and Private Virtual Clock Control

- **Status:** Accepted for the development preview
- **Date:** 2026-08-19
- **Related:** SPEC-0002, SPEC-0003, SPEC-0007, SPEC-0012

## Decision

Use a separate monotonic `head_version` instead of treating the rewinding
`call_count` as the branch version. Normal tool calls, reset, and virtual-clock
advancement update the head in the same SQLite transaction as their state or
control audit mutation. Updates include a head-version predicate and fail with
`BRANCH_CONFLICT` when the predicate is stale.

Virtual clock advancement is a private control-plane operation. It is
forward-only, bounded, and auditable. Scheduled events, deterministic entropy,
and fault injection remain separate proposals and are not implied by this ADR.

## Consequences

- reset can rewind call numbering without destroying concurrency evidence;
- snapshots can bind the exact source head they captured;
- stale harness control requests fail closed;
- local SQLite remains a serialized single-host profile, not a distributed
  concurrency system.

## Evidence

- store schema-v3 migration tests;
- monotonic head tests across call/reset/call;
- private clock advance and stale-head HTTP tests;
- snapshot source-head identity assertion.
