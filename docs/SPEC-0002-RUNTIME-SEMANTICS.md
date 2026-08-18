# SPEC-0002: Runtime Semantics

- **Status:** Proposed normative specification
- **Applies to:** TwinSpec v1alpha1 development and v0.1 release profile

## 1. Environment identity

The reproducible environment identity is:

```text
H(runtimeSemanticVersion,
  TwinSpecDigest,
  SurfaceDigest,
  SnapshotDigest,
  ScenarioDigest,
  Seed,
  ClockInitial,
  SchedulerPolicy,
  FaultProfile)
```

Provider/model identity is recorded separately and MUST NOT be confused with
environment determinism.

## 2. Atomic call

Normal calls are single SQLite transactions. A failed input validation,
precondition, effect, postcondition, invariant, or output validation MUST NOT
commit a state mutation. The default runtime has no partial effects.

`TIMEOUT_BEFORE_EFFECT`, `TIMEOUT_AFTER_EFFECT`, and `PARTIAL_EFFECT` are
different semantics and MUST NOT be collapsed into one error. They are deferred
fault-profile extensions in v0.1, not implemented claims.

## 3. Deterministic sources

All values that become simulated state or tool output MUST derive from the
virtual clock, deterministic allocation rules, the scenario seed, or declared
input. Host wall clock, random UUIDs, process scheduling, and memory addresses
MUST NOT enter the simulated state.

The current v0.1 profile uses deterministic serial execution. Deterministic
multi-agent interleaving is a later scheduler profile and MUST have an explicit
policy rather than inheriting OS scheduling.

## 4. Snapshot and branch

Snapshots are immutable logical roots. A fork creates an isolated branch and
MUST NOT mutate its parent or sibling. Every evaluation trial MUST record the
initial snapshot digest and final state digest. A branch bound to a different
TwinSpec digest MUST fail with `SPEC_DRIFT`.

Snapshot identity is bound to the runtime storage schema and export format.
Unknown newer storage versions MUST be refused. Supported older versions MAY
be migrated only with a tested migration path.

## 5. Explicit unknown behavior

When a tool behavior is not modeled, the runtime MUST return
`UNMODELED_BEHAVIOR` or another declared typed error. It MUST NOT call an LLM,
upstream service, or heuristic fallback to manufacture success.

## 6. Resource limits

The v0.1 profile bounds TwinSpec size to 1 MiB, entities to 256, tools to 256,
invariants to 512, schema canonical bytes to 256 KiB, schema depth to 32,
expressions to 4,096 bytes, tool results to 1 MiB, HTTP body/header size, and
server timeouts. Future limits MUST also cover collection traversal, call
count, branch count, and disk quota before accepting untrusted bundles.

## 7. Crash and recovery contract

The normal transaction contract is: a committed state is fully visible, and an
uncommitted state is absent. Kill-point tests, database corruption tests, and
disk-full tests are required before a stable release. Operational timestamps
and logs are not part of deterministic state.

## 8. Error preservation

The internal canonical error class MUST remain available in audit/evidence. MCP
presentation MAY use an upstream-like envelope, but it MUST NOT turn unknown,
timeout, validation, or authorization failures into success.
