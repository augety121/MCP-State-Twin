# ADR-0024 — Bounded HTTP admission and backpressure

- **Status:** Accepted
- **Date:** 2026-08-31
- **Scope:** MCP data plane, simulation control plane and Episode coordinator

## Context

HTTP body and timeout limits do not prevent an unbounded number of concurrent
requests from creating goroutines and memory pressure. Waiting in an unbounded
queue would make overload latency and cancellation behavior unpredictable.

## Decision

Each listener receives an independent, non-queueing admission handler. A
request acquires one in-flight permit or immediately receives:

```text
HTTP 503
Retry-After: 1
{"error":{"code":"SERVER_BUSY","message":"server is at operational capacity"}}
```

Mode defaults are 4 (`quiet`), 16 (`balanced`) and 64 (`throughput`). The
validated override is `--max-inflight` or `STATETWIN_MAX_INFLIGHT`, range
`1..1024`.

Data plane, control plane and coordinator do not share permit pools. This
prevents Agent traffic from consuming control capacity and preserves the
control/data trust boundary.

## Non-claims

This is process-local admission, not distributed rate limiting, tenant
fairness, priority scheduling, durable queueing, global concurrency control or
DDoS protection. It does not promise that an accepted request will finish.

## Consequences

Rejected requests never enter the application handler and cannot mutate world
state. Operators may retry after backoff. Clients must not interpret
`SERVER_BUSY` as a modeled domain error or successful MCP tool result.
