# SPEC-0018: Remote Episode Coordinator

- **Status:** Accepted via ADR-0020
- **Format:** `statetwin.dev/episode-coordinator/v1alpha1`
- **Journal storage:** SQLite application `EPJL`, schema v2
- **Profile:** single coordinator, multiple workers, synthetic TwinBundles

## 1. Safety invariants

### ST-REMOTE-R001 — Immutable parent

An Episode ID binds to the ADR-0019 request digest. Submitting the same ID with
different Bundle, Scenario or runtime identity MUST conflict.

### ST-REMOTE-R002 — One active attempt

At most one attempt may own a non-expired lease for an Episode. Claim, attempt
creation, task state, lease and fencing-token advancement MUST commit in one
transaction.

### ST-REMOTE-R003 — Fencing

Every mutating worker request MUST carry Episode ID, attempt ID and fencing
token. A stale tuple MUST fail without changing task, Episode or Evidence.

### ST-REMOTE-R004 — Bounded lease

Lease duration MUST be positive and no greater than the resource profile. A
heartbeat may extend only the currently fenced lease and MUST return the
durable cancellation flag.

### ST-REMOTE-R005 — No unsafe automatic retry

The coordinator may requeue only:

- a failed attempt that reports `NO_EFFECT`; or
- an expired attempt whose task policy is `hermetic`.

An expired `external` attempt and every reported `UNKNOWN` commit state MUST
become `COMMIT_UNKNOWN`. They MUST NOT be automatically retried.

### ST-REMOTE-R006 — Terminal acceptance

Completion MUST validate the full Evidence envelope, parent request identity,
runtime identity, canonical digest and lifecycle. Attempt completion, task
completion, parent lifecycle/events and Evidence MUST commit atomically.

### ST-REMOTE-R007 — Duplicate completion

An exact repeat of an accepted completion MAY return the stored Evidence.
Different Evidence, attempt or fence after completion MUST conflict.

### ST-REMOTE-R008 — Cancellation

Cancellation is durable and monotonic. Queued cancellation is terminal.
Leased cancellation is cooperative. A worker acknowledgement with `NO_EFFECT`
is terminal `CANCELLED`; `UNKNOWN` is `COMMIT_UNKNOWN`. Cancellation MUST NOT
erase an already committed completion.

### ST-REMOTE-R009 — Secret boundary

Coordinator storage and public records MUST NOT persist authorization headers,
API keys, OAuth tokens or raw provider error bodies. Errors are typed and
bounded.

### ST-REMOTE-R010 — Control-plane isolation

Submit, claim, heartbeat, cancel, fail, reconcile and complete operations are
never MCP tools and never appear on the agent data plane.

## 2. Task and attempt states

```text
Task:    QUEUED -> LEASED -> COMPLETED
           |         |-> QUEUED          (proven safe retry)
           |         |-> CANCELLED       (NO_EFFECT)
           |         |-> COMMIT_UNKNOWN  (external ambiguity)
           |         |-> FAILED          (budget exhausted)
           |-> CANCELLED

Attempt: LEASED -> COMPLETED | FAILED | CANCELLED | EXPIRED | COMMIT_UNKNOWN
```

`COMMIT_UNKNOWN` is not success and not failure. It requires operator/provider
reconciliation. The coordinator MUST NOT silently collapse it to another state.

## 3. Effect classification

```text
NOT_STARTED  no provider/tool request was issued
NO_EFFECT    the worker has authoritative evidence that no effect committed
COMMITTED    the worker has terminal admissible Evidence
UNKNOWN      a request may have committed but the response is unavailable
```

Only `NO_EFFECT` and expired `hermetic` work permit automatic retry.

## 4. Exactly-once claim vocabulary

The following claim is allowed:

> For one parent Episode, the coordinator atomically accepts at most one
> terminal Evidence envelope; stale or duplicate workers cannot replace it.

The following claims are forbidden:

- exactly-once model inference;
- exactly-once HTTP delivery;
- exactly-once MCP tool execution;
- exactly-once external side effects; or
- transparent recovery from unknown provider commit state.

Provider-native idempotency may strengthen a specific adapter only after its
exact request key, retention window and replay behavior have first-party
documentation plus live evidence.

## 5. Coordinator API

The JSON control API is bearer-authenticated and versioned under `/v1`:

```text
POST /v1/episodes
POST /v1/claims
POST /v1/episodes/{id}/heartbeat
POST /v1/episodes/{id}/complete
POST /v1/episodes/{id}/fail
POST /v1/episodes/{id}/cancel
GET  /v1/episodes/{id}
```

Bodies reject unknown fields and are size bounded. Bundle bytes are accepted
only after normal TwinBundle verification.

## 6. Provider capability contract

Every adapter records independently:

```text
backgroundExecution: supported | unsupported | unknown
retrieve: supported | unsupported | unknown
remoteCancel: supported | disconnect-only | unsupported | unknown
cancelIdempotent: documented | observed | unknown
requestId: body | header | both | unavailable
remoteMCPTools: supported | beta | unsupported | unknown
providerIdempotency: documented | unsupported | unknown
```

OpenAI Responses background polling/cancellation and Anthropic Messages MCP
connector behavior are separate adapter profiles. One MUST NOT inherit the
other's capabilities.

## 7. Required evidence

- schema-v1 to schema-v2 Journal migration and future-version refusal;
- interrupted migration reopen plus `PRAGMA integrity_check`;
- concurrent claim gives one lease owner;
- heartbeat extension and lease bound;
- stale-fence refusal after recovery;
- hermetic expiry requeue and max-attempt exhaustion;
- external expiry to `COMMIT_UNKNOWN` without retry;
- queued and leased cancellation races;
- exact duplicate completion lookup and conflicting duplicate refusal;
- credentials and raw provider errors absent from storage;
- authenticated HTTP negative tests and non-loopback TLS admission;
- mock-server contract tests for every provider adapter; and
- opt-in, dated live evidence for OpenAI-family and Anthropic-family profiles.

Mock tests establish request/response contract shape only. They do not satisfy
a verified provider-profile gate under SPEC-0019.
