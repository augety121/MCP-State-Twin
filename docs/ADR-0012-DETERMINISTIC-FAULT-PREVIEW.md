# ADR-0012 — Branch-local deterministic fault preview

- **Status:** Accepted
- **Date:** 2026-08-21
- **Scope:** preview extension; not part of the stable v0.1 release claim

## Context

SPEC-0008 proposes ten fault phases and several service, consistency, crash, and
ambiguous-outcome classes. Implementing those names without exact transaction
semantics would produce misleading evaluation results. The current SQLite
runtime can prove two useful boundaries:

1. reject a call before runtime validation/effects execute; and
2. atomically commit a successful business transition, then replace the
   model-visible result with a response-loss failure.

## Decision

The runtime accepts a bounded branch-local fault-plan subset:

- phases: `before-validation` and `after-commit-before-response`;
- canonical outcomes:
  - `RATE_LIMITED` or `TIMEOUT_BEFORE_EFFECT` before validation;
  - `TIMEOUT_AFTER_EFFECT` after commit;
- exact tool-name selection;
- deterministic lexicographic plan ordering;
- explicit integer repeat counts from 1 through 1,000;
- at most 128 stored plans per branch;
- plan installation/removal through the authenticated private control plane;
- plan consumption and fault-event audit in the same SQLite transaction as the
  affected call;
- a plan identity digest that excludes wall time and mutable counters.

Installing or removing a plan advances `head_version`. Firing a plan advances
the normal call/head counters exactly once. Plans are branch-local and are not
silently copied to sibling forks.

## Invariants

1. Fault controls never appear in agent-facing MCP `tools/list`.
2. No probability, wall-clock timing, or uncontrolled randomness selects a
   fault.
3. A before-validation fault does not invoke the transition callback and does
   not change the world-state digest.
4. An after-commit fault preserves the committed world-state digest while the
   caller receives `TIMEOUT_AFTER_EFFECT`.
5. Fault counter update, call audit, world mutation, and fault event either
   commit together or roll back together.
6. Exhausted plans remain inspectable but cannot fire.
7. Unknown phases and error classes fail closed.

## Deferred

The following remain unimplemented proposals:

- validation/precondition/effect sub-phases;
- response-delivery faults;
- virtual-time rate-limit expiry;
- latency and scheduled visibility;
- partial business effects;
- idempotency-key collapse;
- process/SQLite crash killpoints;
- cancellation/commit races;
- seeded probability syntax;
- eventual-consistency queues.

Each deferred class requires its own deterministic state model and acceptance
tests before it can be advertised.
