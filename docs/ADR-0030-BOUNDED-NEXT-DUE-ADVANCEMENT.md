# ADR-0030: Add an Explicit Bounded Advance-next Operation

- **Status:** Accepted
- **Date:** 2026-09-01
- **Related:** ADR-0028, ADR-0029, SPEC-0030

## Decision

Keep ordinary clock advance all-or-nothing and add a separate private
`advance-next` operation. It delivers at most 256 events from the earliest
pending instant in total order. It may retain the current clock while draining
an overfull legacy instant, with each prefix committed as an independent head
transition.

## Consequences

- newly admitted queues drain one equal-time instant atomically;
- older overfull preview queues have a deterministic liveness path;
- callers choose explicitly between arbitrary atomic jumps and bounded steps;
- retries can use branch-head CAS to prevent double delivery;
- no scheduled business effect, Agent wakeup or distributed lease is implied.
