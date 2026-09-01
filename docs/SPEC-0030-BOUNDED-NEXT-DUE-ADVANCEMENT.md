# SPEC-0030: Bounded Next-due Preview and Advancement

- **Status:** Accepted via ADR-0030
- **Implementation status:** implemented for private signals and runtime-bound local TwinSpec actions
- **Verification status:** preview, bounded legacy drain/action execution, CAS, empty-queue and HTTP tests

## 1. Purpose

Arbitrary clock jumps intentionally remain all-or-nothing under SPEC-0028. A
second operation is required for deterministic orchestration and for recovery
of development-state queues that contain more than 256 pending events at one
instant. This specification adds a read-only preview and an atomic
`advance-next` step. It does not weaken `/v1/clock/advance`.

## 2. Preview contract

Authenticated private route:

```text
GET /v1/scheduler/next?branch=<branch-id>
```

returns the earliest pending instant according to SPEC-0029, current virtual
clock, branch head, scheduler digest, number pending/actions at that instant,
selected batch event/action counts and whether more than one step is required.
An empty queue returns typed `SCHEDULER_EMPTY` and performs no write.

Preview is advisory. A writer MUST use `expectedHeadVersion` on the subsequent
step when it requires read-modify-write consistency.

## 3. Advance-next contract

Authenticated private route:

```text
POST /v1/clock/advance-next
{
  "branch": "run-a",
  "expectedHeadVersion": 12
}
```

selects the earliest pending instant and processes a prefix of at most 256
events and 32 runtime-bound actions using the complete SPEC-0029 total order.
Signal lifecycle follows SPEC-0028; action execution and evidence follow
SPEC-0032 through SPEC-0034. For a newly admitted signal-only queue, the entire
equal-time set fits one step. An action-heavy or legacy overfull set may require
multiple deterministic prefixes with `moreAtInstant = true`.

On the first step for a future instant, the branch clock advances to that exact
instant. A continuation drain at the same instant keeps the clock unchanged but
still commits a lifecycle/head transition. Clock regression is never allowed.

## 4. Transaction boundary

Selection, signal/action lifecycle, modeled action effects/fault/audit,
optional clock movement, canonical state and digest, call count, head CAS,
scheduler digest and `scheduler.advance.next` audit MUST commit in one SQLite
transaction. An infrastructure error commits none of them.

Each delivered signal records `deliveredAt = dueAt`; each action records the
SPEC-0033 terminal outcome. The response contains the new head, delivered
signals, all processed events in total order, signal/action counts,
`clockAdvanced`, `moreAtInstant`, the next pending due time when present, and
the resulting scheduler digest.

## 5. Interaction with ordinary advance

`/v1/clock/advance` continues to require the whole signal due set to fit its
budget and never commits a prefix. If any pending action is due at or before
the target, ordinary advance fails with `SCHEDULE_ACTION_REQUIRED` and commits
nothing. When a large signal jump would exceed the budget, the caller may
repeatedly use preview plus `advance-next`. The runtime MUST NOT silently
change an ordinary advance into stepwise delivery/action execution.

## 6. Failure and recovery matrix

| Condition | Required result |
|---|---|
| empty pending queue | `404 SCHEDULER_EMPTY`, no head change |
| stale expected head | `409 BRANCH_CONFLICT`, no delivery |
| earliest time beyond maximum horizon | `CLOCK_INVALID`, no delivery |
| corrupt earliest event | explicit failure, no delivery |
| database/commit failure | old clock, lifecycle and head remain visible |
| crash after first legacy batch commits | replay from new head drains next deterministic prefix |
| retry with old expected head | conflict; no double delivery/action execution |

## 7. Non-claims

For a signal, `advance-next` proves only logical delivery. For a `tool-call`, it
proves only the recorded local hermetic TwinSpec transition/evidence. It does
not prove an Agent turn, remote worker, provider request, process or external
side effect. It is neither a wall-clock timer service nor a distributed
scheduler.
