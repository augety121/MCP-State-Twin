# SPEC-0034: Scheduled-action Budget and Zero-cascade Profile

- **Status:** Accepted via ADR-0034
- **Implementation status:** implemented `local-preview-v7` subset
- **Verification status:** 33-action split, mixed action/signal prefix, scheduler-mutation rollback, result-bound rollback and resource-profile tests

## 1. Current resource profile

`statetwin.dev/resource-profile/v1alpha1` `local-preview-v7` binds:

| Field | Value |
|---|---:|
| `maxScheduledDeliveriesPerAdvance` | 256 total events |
| `maxScheduledActionsPerStep` | 32 actions |
| `maxScheduledEventsPerInstant` | 256 pending events |
| `maxScheduledAttempts` | 1 |
| `maxScheduledCascadeDepth` | 0 |
| action input / result | 1 MiB / 1 MiB |
| action tool-audit input + result | 2 MiB |
| complete branch state | 16 MiB |

The 32-action bound limits one SQLite transaction's CEL/schema/effect work and
is independent of the host ExecutionProfile. The default local command still
uses one Go scheduler slot, but that is a soft operational policy rather than a
modeled outcome.

## 2. Prefix selection

Starting at the earliest pending event, a step includes consecutive events in
SPEC-0029 total order until the first of these bounds would be exceeded:

- 256 total events;
- 32 actions;
- end of the earliest equal-time set.

The step never skips an event. If another event remains at the same instant,
`moreAtInstant = true` and the next step retains the same virtual clock.
Preview reports total pending/actions at the instant plus selected batch
events/actions.

The action limit counts actions, not signals. Therefore 32 actions followed by
a signal may all be selected when the 256-total-event bound is not exceeded;
the same prefix followed by a 33rd action stops before that action.

## 3. Zero-cascade enforcement

The scheduled executor may mutate only modeled TwinSpec state through existing
declarative effects. Scheduler state is hidden from CEL and is not a TwinSpec
effect target. The store additionally compares scheduler digest immediately
before and after every executor callback. Any attempted creation, cancellation,
reordering or lifecycle mutation by the callback fails the whole transaction.

Only the scheduler itself may terminalize the current event after the callback
returns. Therefore `maxScheduledCascadeDepth = 0` is executable behavior, not a
documentation convention.

## 4. Retry and recurrence

There is no retry queue, delay/backoff, recurrence expansion, child-event
creation, dead-letter store or compensation in this profile. Domain failure is
terminal. Infrastructure rollback leaves the same pending action at the same
head; a control client may explicitly retry that scheduler transaction.

Any future automatic retry/cascade design requires a new policy identifier,
cycle detection, per-root budgets, idempotency/ambiguity rules, retention and
new executable evidence. It cannot silently reinterpret v2 events.

## 5. Failure rules

| Condition | Result |
|---|---|
| 33 equal-time actions | first 32, then one on the next explicit step |
| 32 actions followed by a signal | all 33 events may commit in one step; action count remains 32 |
| callback changes scheduler | `SCHEDULE_INVALID`, whole step rollback |
| result exceeds 1 MiB | `RESOURCE_LIMIT`, whole step rollback |
| complete action evidence exceeds state limit | `RESOURCE_LIMIT`, rollback |
| automatic child schedule attempt | rejected by zero-cascade check |

## 6. Non-claims

The budget is not a throughput, latency, thermal, CPU-percentage or production
capacity guarantee. It does not make local SQLite a distributed workflow
system and does not authorize executing arbitrary or external work.
