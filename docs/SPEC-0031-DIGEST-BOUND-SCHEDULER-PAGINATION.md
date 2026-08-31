# SPEC-0031: Digest-bound Scheduler Inspection Pagination

- **Status:** Accepted via ADR-0031
- **Implementation status:** implemented for the authenticated control plane
- **Verification status:** ordering, filtering, continuation, malformed cursor and stale cursor tests

## 1. Problem statement

Returning every retained event in one response makes inspection memory and
response size depend on the complete queue. It also gives no way to detect a
queue mutation between client pages. Scheduler inspection therefore requires a
bounded page and an opaque continuation bound to the inspected scheduler state.

## 2. Request contract

```text
GET /v1/scheduler/events
  ?branch=<branch-id>
  &status=<pending|delivered|canceled>
  &limit=<1..256>
  &cursor=<opaque-base64url>
```

- `branch` is required;
- `status` is optional and must be one of the three lifecycle states;
- default `limit` is 100 and maximum is 256;
- cursor length is limited to 2,048 bytes;
- unknown status, malformed integer, malformed cursor or unsupported cursor
  format fails explicitly.

The first page omits `cursor`. Every continuation MUST repeat the same branch,
status and limit policy. Limit may be reduced or increased within the valid
range because it does not alter the cursor boundary.

## 3. Cursor identity

Cursor format is `statetwin.dev/scheduler-cursor/v1`, encoded as unpadded
base64url canonical JSON. It binds:

- branch ID;
- branch head version;
- scheduler digest;
- status filter;
- the final event's full ordering tuple: parsed-time representation,
  priority, creation sequence and ID.

The cursor is opaque transport state, not an authorization credential and not
a stability promise across scheduler mutations. The authenticated control
route remains the security boundary.

## 4. Consistency behavior

Each page reads one validated branch snapshot. If the branch head or scheduler
digest changes before a continuation, the request MUST fail with
`SCHEDULE_CONFLICT`; it MUST NOT mix events from two branch snapshots or
silently restart. The client may restart from page one. Binding both fields
also rejects an ABA reset that restores old queue bytes under a newer head.

The response contains format, policy, branch, head version, scheduler digest,
the legacy `digest` alias during v1alpha1 compatibility, applied status, a JSON
array of events and an optional next cursor. Event order is SPEC-0029 order.

Head version and scheduler digest jointly bind the continuation. This
conservative profile intentionally invalidates a page after unrelated branch
mutation rather than claiming a snapshot that the server no longer holds.

## 5. Failure matrix

| Condition | Result |
|---|---|
| limit omitted | 100 |
| limit 0, negative or above 256 | `SCHEDULE_INVALID` |
| cursor is not base64url canonical shape | `SCHEDULER_CURSOR_INVALID` |
| cursor branch/filter differs | `SCHEDULE_CONFLICT` |
| scheduler mutates between pages | `SCHEDULE_CONFLICT` |
| cursor boundary absent under same claimed digest | fail closed |
| final page | empty `nextCursor` omission, events remains an array |

## 6. Security and non-claims

Cursor payloads contain identifiers and a state digest; operators must treat
them as evaluation metadata. They contain no token or secret by design. This
contract does not provide signed cursors, retention, server-side sessions,
cross-branch pagination or Agent-facing queue inspection.
