# SPEC-0025 — Operational Health and Readiness

- **Status:** Accepted by ADR-0025
- **Format:** `statetwin.dev/health/v1alpha1`

## 1. Endpoint contract

Successful liveness and readiness responses use HTTP 200:

```json
{
  "format": "statetwin.dev/health/v1alpha1",
  "check": "live",
  "status": "ok",
  "version": "0.1.0-dev"
}
```

Readiness uses `check: ready`. A storage ping failure uses HTTP 503 and the
existing error envelope with code `NOT_READY` and message
`storage is unavailable`.

## 2. Authorization and surface isolation

Both routes MUST pass through constant-time control-plane bearer validation.
Missing/invalid authorization receives the existing generic `AUTH_DENIED`
response. Neither route may be registered on the MCP data plane or represented
as a tool, resource or prompt.

## 3. Data minimization

Responses MUST NOT contain:

- branch, snapshot, Episode, attempt or worker identifiers;
- tool names, schemas, prompts, state or counts;
- database paths, SQL/driver errors or schema details;
- tokens, headers, client identities, hostnames or network addresses;
- provider or upstream availability.

The store check MUST use request context and perform no modeled-state read or
write.

## 4. Semantics

`live` means only that the authenticated handler is executing. `ready` means
only that the local store accepted a ping at that instant. Neither result is a
durable guarantee, release-fidelity claim, or proof that an Agent scenario will
succeed.

## 5. Evidence

Tests MUST cover authentication, successful live/ready responses, exact format,
absence of sensitive identifiers, closed-store failure, static redacted error,
and continued absence from MCP discovery.
