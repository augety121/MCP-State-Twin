# SPEC-0030: Bounded Next-due Preview and Advancement

- **Status:** Accepted via ADR-0030
- **Implementation status:** implemented for private opaque signals
- **Verification status:** preview, bounded legacy drain, CAS, empty-queue and HTTP tests

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
clock, branch head, scheduler digest, number pending at that instant, maximum
batch size and whether more than one step is required. An empty queue returns
typed `SCHEDULER_EMPTY` and performs no write.

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

selects the earliest pending instant and delivers a prefix of at most 256
signals using the complete SPEC-0029 total order. For a newly admitted queue,
the entire equal-time set fits one step. For a legacy overfull set, the selected
prefix is deterministic and `moreAtInstant` is true.

On the first step for a future instant, the branch clock advances to that exact
instant. A continuation drain at the same instant keeps the clock unchanged but
still commits a lifecycle/head transition. Clock regression is never allowed.

## 4. Transaction boundary

Selection, delivery lifecycle, optional clock movement, canonical state and
digest, head CAS, scheduler digest and `scheduler.advance.next` audit MUST
commit in one SQLite transaction. An error commits none of them.

Each delivered event records `deliveredAt = dueAt`. The response contains the
new head, delivered events in total order, `clockAdvanced`, `moreAtInstant`,
the next pending due time when present, and the resulting scheduler digest.

## 5. Interaction with ordinary advance

`/v1/clock/advance` continues to require the whole selected due set to fit its
budget and never commits a prefix. When a large jump would exceed that budget,
the caller may repeatedly use preview plus `advance-next`. This is an explicit
operation choice; the runtime MUST NOT silently change an ordinary advance
into stepwise delivery.

## 6. Failure and recovery matrix

| Condition | Required result |
|---|---|
| empty pending queue | `404 SCHEDULER_EMPTY`, no head change |
| stale expected head | `409 BRANCH_CONFLICT`, no delivery |
| earliest time beyond maximum horizon | `CLOCK_INVALID`, no delivery |
| corrupt earliest event | explicit failure, no delivery |
| database/commit failure | old clock, lifecycle and head remain visible |
| crash after first legacy batch commits | replay from new head drains next deterministic prefix |
| retry with old expected head | conflict; no double delivery |

## 7. Non-claims

An `advance-next` success proves only logical delivery into the modeled private
signal queue. It does not prove a tool call, Agent turn, remote worker, provider
request or external side effect. It is neither a wall-clock timer service nor a
distributed scheduler.
