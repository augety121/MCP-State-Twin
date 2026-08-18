# ADR-0009: Operational logging boundary

- **Status:** Accepted
- **Date:** 2026-08-18
- **Related:** RFC-0001 I-9/I-10, F-006, F-087

## Decision

The development CLI emits only operational status, endpoint addresses, and
sanitized error summaries. It MUST NOT log MCP arguments, tool results, world
state, authorization headers, bearer tokens, API keys, private keys, or email
addresses. Every top-level CLI error is passed through `internal/logging` before
it reaches the standard logger. The sanitizer is deliberately not applied to
MCP responses or audit records, which retain their typed contract and existing
storage semantics.

The data plane has no request-body logging middleware. The control plane never
logs the bearer token. Future structured observability or recorder work requires
a new ADR with field-level redaction tests before it enters a release scope.

## Evidence

- unit tests cover authorization, generic secret, known token, private-key,
  and email redaction;
- CLI top-level error logging calls `logging.SafeError`;
- no data-plane request logging middleware exists.
