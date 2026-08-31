# SPEC-0029: Temporal Ordering and Per-instant Scheduler Admission

- **Status:** Accepted via ADR-0029
- **Implementation status:** implemented bounded signal subset
- **Verification status:** sub-second ordering, admission, cancellation-capacity and legacy-state tests

## 1. Problem statement

`time.RFC3339Nano` is the canonical wire representation for modeled time, but
its variable fractional-second component is not lexicographically sortable.
For example, `01:00:00.1Z` sorts before `01:00:00Z` as text even though it is
later in time. The scheduler therefore MUST compare parsed UTC instants, never
raw timestamp strings, before applying its remaining tie-break fields.

The existing retained-event limit also does not by itself guarantee progress.
A legal queue could previously contain 257 pending signals at one instant while
one ordinary clock advance may deliver only 256. No smaller target could split
that equal-time set. New admission must prevent creation of such an undrainable
instant, while previously persisted preview data remains readable and can be
recovered through SPEC-0030.

## 2. Normative total order

For validated events `a` and `b`, the total order is:

1. parsed UTC `dueAt`, ascending;
2. `priority`, descending;
3. `creationSequence`, ascending;
4. `id`, ascending.

Canonical timestamp formatting remains required for identity and hashing, but
formatting is not the temporal comparison algorithm. A malformed timestamp is
invalid persisted state and MUST fail explicitly before business execution.

## 3. Per-instant admission

Under resource profile `local-preview-v6`, at most 256 **pending** events may
share one canonical `dueAt` in a branch. Event creation MUST count the existing
pending set in the same branch transaction and fail with `RESOURCE_LIMIT`
before state, digest, head version or audit mutation when the limit is reached.

Canceled and delivered events are retained lifecycle evidence but do not
consume pending capacity at that instant. Canceling a pending event therefore
frees one admission slot. The global retained-event limit of 1,024 still
applies independently.

This rule is intentionally an admission invariant rather than a new read-time
rejection. States produced by an earlier development build may contain a larger
equal-time set; making them unreadable would remove the only deterministic
recovery path. SPEC-0030 defines that recovery.

## 4. Atomicity and concurrency

The due-time count, duplicate-ID check, sequence allocation, event insertion,
state validation, digest update, head CAS and control audit MUST occur in one
SQLite transaction. Concurrent creators may yield at most one successful
admission at the final slot; a stale head or resource failure commits nothing.

## 5. Resource identity

The following values are part of `statetwin.dev/resource-profile/v1alpha1`
`local-preview-v6`:

| Field | Value |
|---|---:|
| `maxScheduledEvents` | 1,024 |
| `maxScheduledEventsPerInstant` | 256 |
| `maxScheduledDeliveriesPerAdvance` | 256 |

`maxScheduledEventsPerInstant` MUST NOT exceed the ordinary delivery budget in
a profile that claims single-operation drainability for newly admitted events.

## 6. Required failure behavior

| Condition | Result |
|---|---|
| 256 pending signals already at `dueAt` | reject the 257th with `RESOURCE_LIMIT` |
| one of 256 signals is canceled | one replacement may be admitted |
| same instant represented non-canonically | reject as invalid input/state |
| timestamp parse fails | explicit invalid-state/input failure |
| stale `expectedHeadVersion` | `BRANCH_CONFLICT`, no capacity consumed |
| 257-event legacy state read | admit the state, expose recovery via SPEC-0030 |

## 7. Non-claims

This specification does not add recurrence, scheduled TwinSpec effects,
provider calls, Agent wakeup, retry, dead-letter queues, retention or external
time synchronization. It only makes the private signal queue ordered and
progress-safe at admission.
