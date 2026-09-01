# ADR-0033: Couple Scheduled Tool Effects and Terminal Evidence

- **Status:** Accepted
- **Date:** 2026-09-01
- **Related:** ADR-0012, ADR-0028, ADR-0030, ADR-0032, SPEC-0033

## Decision

Execute each selected scheduled TwinSpec action, consume deterministic faults,
append tool/fault audit and terminalize the action in the same SQLite branch
transaction. Domain failure commits a `failed` terminal record; infrastructure
failure rolls back the entire scheduler step.

Record `effectCommitted` so before-effect/domain failures remain distinct from
`TIMEOUT_AFTER_EFFECT`. Allocate one branch call index per executed action and
one branch head per aggregate scheduler step.

## Consequences

- crash recovery observes either the old pending head or the complete new head;
- stale expected-head retries cannot execute the same committed event twice;
- action outcomes link to ordinary tool-audit/fault evidence;
- failed-after-effect is explicit rather than silently converted to success;
- this decision does not claim exactly-once external effects.
