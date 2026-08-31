# SPEC-0028: Atomic Due-signal Delivery on Virtual-clock Advance

- **Status:** Accepted via ADR-0028
- **Implementation status:** implemented bounded delivery subset
- **Verification status:** atomic success, deterministic order, cancellation exclusion and budget rollback tests

## 1. Delivery rule

For target virtual time `T`, every event satisfying both conditions is due:

```text
status == pending
dueAt <= T
```

The runtime sorts due events by the SPEC-0027 total order, changes them to
`delivered`, sets `deliveredAt = dueAt`, advances the branch clock to `T`,
updates canonical state/digest and increments `headVersion` exactly once.

## 2. Atomicity

Clock, scheduler lifecycle, state digest, head version and the aggregate
`clock.advance` control-audit record MUST commit in one SQLite transaction. On
parse error, stale head, clock regression, horizon violation, corrupt event,
resource violation, database error or commit failure, none may become visible.

Canceled or previously delivered events are never delivered again. A clock
advance with zero due signals remains a valid audited head change and does not
change state digest.

## 3. Bounded processing

At most 256 events may be delivered by one advance under
`local-preview-v5`. If the due set is larger, the request returns
`RESOURCE_LIMIT`; the client must use smaller deterministic time steps or
reduce the retained queue. The runtime MUST NOT deliver a prefix, silently drop
events or defer an arbitrary suffix.

This is a delivery batch bound, not a throughput promise. The operational
ExecutionProfile independently limits local CPU/heap/HTTP concurrency.

## 4. Response and evidence

Successful `/v1/clock/advance` returns:

```json
{
  "branch": "run-a",
  "clock": "2026-08-01T02:00:00Z",
  "headVersion": 4,
  "delivered": [],
  "schedulerDigest": "sha256:..."
}
```

`delivered` uses deterministic order and is always a JSON array. The digest
binds terminal lifecycle state. Operational audit `createdAt` remains host-time
metadata and is excluded from deterministic equality.

## 5. Interaction table

| Interaction | Required result |
|---|---|
| event due exactly at target | delivered |
| event after target | remains pending |
| canceled event before target | remains canceled |
| sibling fork advanced | original/sibling queue unchanged |
| stale expected head | no clock or queue change |
| 257 due events | fail whole advance |
| malformed persisted dueAt | explicit failure; no invented delivery |

## 6. Explicit non-claims

Delivery means deterministic signal availability in the private harness. It
does not mean a TwinSpec effect executed, an MCP notification was pushed, an
Agent woke up, a provider accepted a request, or an external side effect
occurred. Cascading/scheduled effects require a later contract with a separate
bounded execution budget and rollback model.
