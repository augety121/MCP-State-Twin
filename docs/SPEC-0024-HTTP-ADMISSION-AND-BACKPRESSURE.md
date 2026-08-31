# SPEC-0024 — HTTP Admission and Backpressure

- **Status:** Accepted by ADR-0024
- **Profile:** local process, independent listener pools

## 1. Admission algorithm

Every hardened HTTP server MUST wrap its application handler in a bounded
permit pool. Acquisition MUST be non-blocking:

```text
permit available -> increment active -> call handler -> release permit
no permit        -> increment rejected -> HTTP 503 SERVER_BUSY
```

No goroutine, timer, queue entry or body read may be created for rejected
application work beyond what the Go HTTP server already created to parse the
request.

## 2. Configuration

| Mode | Maximum in-flight requests per listener |
|---|---:|
| `quiet` | 4 |
| `balanced` | 16 |
| `throughput` | 64 |

`--max-inflight` overrides `STATETWIN_MAX_INFLIGHT`; accepted values are
`1..1024`. Configuration errors MUST stop startup.

## 3. Error contract

An overload response MUST contain status 503, `Retry-After: 1`,
`Cache-Control: no-store`, `X-Content-Type-Options: nosniff`, JSON content type,
code `SERVER_BUSY`, and a static message without request, branch, token, tool,
state or storage details.

The rejected request MUST NOT call the wrapped handler or mutate state. A
released permit MUST be reusable after success, failure, cancellation or panic
unwinding.

## 4. Isolation

Every data-plane, control-plane and Episode-coordinator listener MUST own a
separate handler and pool. Admission controls are transport infrastructure and
MUST NOT appear in MCP `tools/list`.

## 5. Observability and privacy

The internal counter set is limited to configured capacity, current active
requests and cumulative rejects. This version does not publish a metrics
endpoint. Future export requires a cardinality and privacy review and MUST NOT
label metrics by branch, token, prompt, request body or hidden state.

## 6. Evidence

Tests MUST occupy every permit, prove the next request receives the exact
overload contract without invoking the application, release capacity, prove a
later request succeeds, and validate counters and configuration boundaries.
