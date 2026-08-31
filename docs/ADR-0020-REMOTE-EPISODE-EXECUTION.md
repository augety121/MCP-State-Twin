# ADR-0020: Fenced Remote Episode Execution

- **Status:** Accepted
- **Date:** 2026-08-26
- **Scope:** v0.2 development preview; independent Episode coordinator
- **Amends:** ADR-0019
- **Does not amend:** the hermetic MCP data plane or v0.1 world-store profile

## Context

ADR-0019 persists local Episode results but deliberately refuses to infer that
an incomplete record is safe to retry. Remote workers require a stronger
contract: attempts, bounded leases, fencing, cancellation races, and explicit
classification of effects whose commit state cannot be observed after a
disconnect.

No coordinator can guarantee exactly-once execution across arbitrary model
providers or external tools. A process may lose its response after the remote
system committed. Retrying in that state can duplicate effects; declaring
success can invent an effect that never occurred.

## Decision

The Episode Journal gains an independent schema-v2 coordinator profile:

1. an immutable parent Episode request;
2. a bounded task policy (`hermetic` or `external`);
3. numbered attempts with one active lease;
4. a monotonically increasing fencing token per claim;
5. heartbeat-based lease extension;
6. durable cancellation requests and fenced worker acknowledgement;
7. automatic retry only after a worker proves `NO_EFFECT`, or after an expired
   `hermetic` attempt;
8. `COMMIT_UNKNOWN` as a terminal administrative state for ambiguous external
   effects; and
9. atomic, idempotent acceptance of one terminal EpisodeEvidence envelope.

The coordinator protocol is control plane. It MUST NOT be registered in the
agent-facing MCP `tools/list` surface.

## Cancellation precedence

- A queued task can be cancelled atomically without an attempt.
- A leased task records `cancelRequested=true`; the worker learns this through
  heartbeat and must stop cooperatively.
- Completion and cancellation are serialized by the Journal transaction. A
  completion committed first wins. A prior cancellation request prevents a
  later completion from being admitted.
- `NO_EFFECT` cancellation becomes `CANCELLED`.
- `UNKNOWN` or unproven external cancellation becomes `COMMIT_UNKNOWN`.

## Recovery and delivery semantics

- Claim delivery is **at-least-once** for `hermetic` tasks within `maxAttempts`.
- External tasks are not automatically redelivered after lease expiry.
- A stale worker cannot heartbeat, fail, cancel or complete after its fencing
  token has been superseded.
- Repeating an already accepted completion is an idempotent lookup only when
  the attempt, fencing token and Evidence digest all match.
- The system claims **exactly-once terminal acceptance**, not exactly-once
  provider calls, model inference, tool execution or external side effects.

## Network and credential boundary

The coordinator requires bearer authentication. Non-loopback listeners require
TLS. Tokens and provider credentials are supplied at runtime and MUST NOT be
stored in the Journal, reports, traces or repository.

## Consequences

- Journal schema v1 must migrate forward to v2 with preservation evidence.
- Worker crash, lease expiry, stale fencing, duplicate completion, cancellation
  race and ambiguous commit tests are release gates for this preview.
- Provider adapters must publish capability differences instead of pretending
  all providers support remote cancellation or idempotency.
- `ST-EPISODE-005` may be marked implemented only for this bounded coordinator
  profile. Arbitrary exactly-once execution remains impossible and unclaimed.
