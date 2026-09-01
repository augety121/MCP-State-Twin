# SPEC-0027: Branch-local Deterministic Signal Scheduler

- **Status:** Accepted via ADR-0027
- **Implementation status:** implemented bounded signal queue
- **Verification status:** ordering, lifecycle, CAS, snapshot/fork isolation and HTTP tests
- **Policy introduced here:** `signal-queue-v1`; current combined queue policy is `deterministic-queue-v2` via ADR-0032

## 1. Scope

The scheduler models that a bounded JSON signal becomes due at virtual world
time. It is hidden harness infrastructure, not an MCP business tool and not an
Agent/workflow scheduler.

## 2. Event record

Each signal event contains:

```text
id
dueAt                    RFC3339Nano UTC
priority                 integer -1000..1000
creationSequence         branch-local monotonic integer
kind                     exactly "signal" under this event subtype
payload                  bounded canonical JSON
status                   pending | delivered | canceled
deliveredAt              set only for delivered
canceledAt               set only for canceled
```

Event IDs use the repository control identifier grammar and are never reusable
within retained branch scheduler state. Payload has no executable semantics.
Unknown `kind` values fail closed. ADR-0032 later admits the separate
`tool-call` subtype under `deterministic-queue-v2`; it does not change signal
payload or lifecycle semantics.

## 3. Ordering

The total order is:

```text
(dueAt ASC, priority DESC, creationSequence ASC, id ASC)
```

`creationSequence` is assigned atomically when the event is accepted. Host
wall time, map iteration, goroutine completion and SQLite row order MUST NOT
affect ordering.

## 4. Lifecycle

Allowed transitions are:

```text
create -> pending
pending -> delivered
pending -> canceled
```

Delivered and canceled are terminal. Repeating cancellation, canceling a
delivered event or reusing an ID returns `SCHEDULE_CONFLICT`; it is not silently
treated as success. A missing ID returns `SCHEDULE_NOT_FOUND`.

`dueAt` MUST be strictly after the branch's current virtual clock and within the
10-year maximum horizon. Creation and cancellation may carry
`expectedHeadVersion`; stale callers receive `BRANCH_CONFLICT` without state
change.

## 5. Canonical state, snapshot and fork

The scheduler queue and next creation sequence are part of canonical branch
state and state digest, but excluded from TwinSpec CEL state. Consequently:

- snapshot captures pending and terminal events plus creation sequence;
- fork starts from the exact captured scheduler digest;
- sibling lifecycle changes remain isolated;
- reset restores queue and sequence;
- ordinary Agent tool calls cannot inspect or mutate scheduler internals.

Scheduler identity is the canonical digest of format, policy and complete
queue state. Host timestamps are not inputs.

## 6. Private control API

All routes require the independent control bearer token:

| Method | Path | Result |
|---|---|---|
| `POST` | `/v1/scheduler/events` | create one pending event |
| `GET` | `/v1/scheduler/events?branch=...` | ordered events and scheduler digest |
| `POST` | `/v1/scheduler/events/cancel` | cancel one pending event |

These routes MUST NOT be registered in MCP `tools/list`. Their bodies use the
existing 1 MiB strict single-JSON-value decoder.

Payload is persisted in canonical branch state and returned to authenticated
control clients without a recorder-redaction pass. The current profile is
therefore synthetic/test data only; credentials, authorization headers,
production traces and personal data MUST NOT be submitted.

## 7. Limits

The current `local-preview-v7` retains the 1,024-event branch bound introduced
by `local-preview-v5`. Pending,
delivered and canceled events all count because automatic retention/GC is not
implemented. Payload is bounded by the normal 1 MiB input limit and the final
branch state remains bounded by the 16 MiB state limit.

## 8. Audit and errors

Create and cancel update state digest and head version in one SQLite transaction
with `scheduler.event.create` or `scheduler.event.cancel` control audit. Errors
are typed as `SCHEDULE_INVALID`, `SCHEDULE_NOT_FOUND`, `SCHEDULE_CONFLICT`,
`BRANCH_CONFLICT` or `RESOURCE_LIMIT`. An error MUST NOT partially mutate the
queue, head or audit ledger.

## 9. Explicit non-claims

This signal subtype has no executable payload semantics. The repository has no
recurring schedule, cron parser, background goroutine, wall-clock wakeup,
Agent resume, provider call, retry queue, distributed lease,
multi-coordinator ordering, retention/GC or external effect. Runtime-bound
local TwinSpec actions are governed only by SPEC-0032 through SPEC-0034.
