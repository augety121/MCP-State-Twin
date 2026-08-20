# SPEC-0008 — Deterministic Fault Injection (Accepted Preview Subset)

- **Status:** Accepted subset via ADR-0012
- **Verification status:** two transaction phases implemented and tested
- **Release profile:** development preview; outside stable v0.1 claims
- **Depends on:** SPEC-0002, SPEC-0007, SPEC-0012

## 1. Supported contract

A private control client may install a branch-local plan:

```json
{
  "id": "lose-close-response",
  "branch": "run-a",
  "tool": "close_issue",
  "phase": "after-commit-before-response",
  "errorClass": "TIMEOUT_AFTER_EFFECT",
  "message": "synthetic response loss",
  "repeatCount": 1,
  "expectedHeadVersion": 0
}
```

Supported phase/outcome combinations are normative:

| Phase | Allowed error classes | World transition |
|---|---|---|
| `before-validation` | `RATE_LIMITED`, `TIMEOUT_BEFORE_EFFECT` | transition callback is not invoked; state is unchanged |
| `after-commit-before-response` | `TIMEOUT_AFTER_EFFECT` | successful business transition commits; its normal result is not delivered |

The current MCP adapter reports the timeout as a structured tool error with
`isError=true`; it does not tear down the HTTP connection or fabricate socket
timing. Transport-level connection loss remains a separate future profile.

All other phase/outcome combinations MUST fail with `FAULT_INVALID`.

## 2. Selection and counters

For one branch/tool/phase, eligible plans are selected by ascending plan ID.
This ordering is deterministic and is part of the current semantic version.
Each call consumes at most one plan. A consumed plan decrements
`remaining_count` and increments `fired_count` atomically.

Plans are not inherited by snapshot/fork operations. This prevents a hidden
fault from leaking into a sibling evaluation. A caller that wants the same plan
on multiple branches MUST install it explicitly on every branch.

## 3. Identity

`FaultPlanDigest` binds:

- format version;
- ordered plan IDs;
- tool selectors;
- phases;
- canonical error classes;
- messages; and
- original repeat counts.

It excludes host timestamps and mutable fired/remaining counters. Therefore
consuming a plan does not change the plan identity, while installing, removing,
or changing configuration does.

## 4. Audit and observability

Every fired fault persists:

- branch ID;
- fault ID;
- call index;
- phase;
- canonical error class;
- before and after state digests; and
- informational creation time.

The event is inserted in the affected call transaction. In particular,
`TIMEOUT_AFTER_EFFECT` cannot be returned unless both the world mutation and
fault event committed.

Configured plans and fired events are separately queryable through the private
control API.

## 5. Resource limits

| Limit | Value |
|---|---:|
| plans per branch | 128 |
| repeat count per plan | 1,000 |
| message bytes | 4,096 |
| faults consumed per call | 1 |

Requests use the existing 1 MiB control-body limit and strict unknown-field
rejection.

## 6. Private control API

| Method | Path | Purpose |
|---|---|---|
| `POST` | `/v1/faults` | install one validated plan |
| `GET` | `/v1/faults?branch=...` | list plans and stable plan digest |
| `POST` | `/v1/faults/remove` | remove a plan with optional head CAS |
| `GET` | `/v1/fault-events?branch=...` | list fired events |

These endpoints require the independent control bearer token. They MUST NOT be
registered as MCP tools.

## 7. Executable acceptance evidence

Tests cover:

- callback suppression and unchanged digest before validation;
- commit followed by caller-visible timeout;
- one-shot exhaustion;
- stable plan digest across counter consumption;
- branch-local plan isolation;
- invalid phase/class/count rejection;
- expected-head installation;
- private HTTP installation and event inspection; and
- continued absence of fault controls from MCP `tools/list`.

## 8. Explicit non-claims

This subset does not implement the other SPEC-0008 phases, realistic network
latency or connection teardown, OS crash simulation, retry idempotency, scheduled visibility,
eventual consistency, or cancellation races. Those remain proposal work and
must not be inferred from the phrase “deterministic fault injection.”
