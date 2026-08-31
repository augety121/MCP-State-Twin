# ADR-0031: Use Digest-bound Opaque Cursors for Scheduler Inspection

- **Status:** Accepted
- **Date:** 2026-09-01
- **Related:** ADR-0002, ADR-0027, ADR-0029, SPEC-0031

## Decision

Scheduler list responses are bounded to 100 events by default and 256 at most.
Continuation uses a versioned base64url cursor binding branch, branch head,
status filter, scheduler digest and the last total-order tuple. Any branch
mutation makes the cursor stale and returns a conflict, preventing scheduler
ABA through reset.

## Consequences

- control responses no longer scale to the entire retained queue by default;
- clients detect rather than unknowingly mix mutated pages;
- cursors remain implementation-readable but client-opaque and are not auth;
- `digest` is temporarily retained beside `schedulerDigest` for v1alpha1
  response compatibility;
- pagination and preview remain outside Agent MCP discovery.
