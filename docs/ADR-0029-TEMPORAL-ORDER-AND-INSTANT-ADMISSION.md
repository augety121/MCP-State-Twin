# ADR-0029: Parse Modeled Time and Bound Pending Events per Instant

- **Status:** Accepted
- **Date:** 2026-09-01
- **Related:** ADR-0011, ADR-0027, ADR-0028, SPEC-0029

## Decision

Scheduler order compares parsed UTC instants rather than RFC3339Nano text. New
event admission limits a branch to 256 pending signals at one canonical world
instant under `local-preview-v6`.

The limit is enforced in the event-creation transaction. It is not imposed as
a read-time invariant on older preview state; legacy overfull instants are
drained through ADR-0030.

## Consequences

- sub-second timestamps have correct temporal order;
- newly admitted equal-time sets fit the ordinary atomic delivery budget;
- cancellation releases pending capacity without deleting lifecycle evidence;
- the resource-profile digest changes from `local-preview-v5` to
  `local-preview-v6`;
- existing overfull preview queues remain readable rather than being silently
  repaired or rendered unrecoverable.
