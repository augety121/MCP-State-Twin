# ADR-0028: Atomic Due-signal Delivery During Virtual-clock Advance

- **Status:** Accepted
- **Date:** 2026-08-31
- **Related:** ADR-0011, ADR-0027, SPEC-0007, SPEC-0028

## Decision

Virtual-clock advancement and delivery of all bounded due signals are one
branch transaction. Due signals are selected using `signal-queue-v1` ordering.
If more than 256 signals would become due, the operation fails with
`RESOURCE_LIMIT` and commits neither clock nor scheduler lifecycle changes.

A delivered signal records its own `dueAt` as `deliveredAt`. This represents
logical delivery at the scheduled world instant even when the control client
jumps over several due instants in one request.

## Consequences

- observers cannot see a new clock with an old due queue;
- replay and fork comparison bind the same delivery order;
- bounded failure is explicit instead of processing an unbounded cascade;
- this decision does not authorize scheduled business effects or Agent wakeup.
