# ADR-0027: Branch-local Deterministic Signal Scheduler

- **Status:** Accepted
- **Date:** 2026-08-31
- **Related:** ADR-0002, ADR-0011, SPEC-0007, SPEC-0012, SPEC-0027

## Context

Virtual time without a queue cannot represent a future signal, compare equal
initial worlds after a fork or prove equal-time ordering. A general workflow
engine or background Agent dispatcher would materially expand the project and
would not be justified by the current deterministic kernel.

## Decision

Accept a bounded `signal-queue-v1` scheduler. Events are branch-local canonical
world infrastructure with `pending`, `delivered` and `canceled` states. Create,
list and cancel operations exist only on the authenticated control plane.

Events carry an opaque JSON payload. This version records deterministic signal
availability; it does not execute a TwinSpec tool, call a provider, wake an
Agent or create an external side effect.

## Consequences

- snapshots and forks bind queue state without a storage-schema migration;
- equal-time order is explicit and testable;
- lifecycle changes advance branch head and are audited;
- the Agent data plane remains unchanged;
- recurring timers, scheduled tool effects and distributed scheduling remain
  deferred.

## Amendment (2026-09-01)

ADR-0032 advances the combined scheduler identity to
`deterministic-queue-v2` and admits a distinct runtime-bound local TwinSpec
`tool-call` subtype. The opaque signal subtype defined here remains
non-executable. Agent/provider/external wakeup and distributed scheduling stay
deferred.
