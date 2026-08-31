# ADR-0025 — Authenticated control-plane health and readiness

- **Status:** Accepted
- **Date:** 2026-08-31
- **Scope:** simulation control plane only

## Context

Operators need to distinguish a live process from one whose SQLite store is no
longer reachable. Reusing an Agent-facing MCP tool would leak operational
controls into the evaluation surface and could let an Agent infer hidden state.

## Decision

The existing authenticated control plane adds:

- `GET /v1/health/live` — process/router liveness;
- `GET /v1/health/ready` — liveness plus a context-bound SQLite ping.

Both endpoints require the same control-plane bearer authorization as every
other private control operation. Responses contain only format, check, status
and runtime version. A readiness failure returns a static `NOT_READY` message.

## Non-claims

These endpoints do not inspect modeled branches, invariants, upstream systems,
provider availability, disk capacity, replication, migrations in progress or
remote-worker health. They are not a production orchestration contract.

## Consequences

Health endpoints never appear in MCP discovery. Storage errors are not returned
to callers. A future unauthenticated Kubernetes probe requires a separate
network and information-disclosure decision.
