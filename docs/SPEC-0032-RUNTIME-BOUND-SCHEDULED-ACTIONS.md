# SPEC-0032: Runtime-bound Scheduled TwinSpec Actions

- **Status:** Accepted via ADR-0032
- **Implementation status:** implemented bounded hermetic subset
- **Verification status:** admission, spec binding, ordinary-advance refusal, due-time execution and private HTTP tests
- **Scheduler policy:** `deterministic-queue-v2`

## 1. Scope

This specification adds one scheduled event kind:

```json
{
  "id": "close-later",
  "branch": "run-a",
  "dueAt": "2026-09-01T00:00:00Z",
  "priority": 10,
  "kind": "tool-call",
  "action": {
    "tool": "close_issue",
    "input": {
      "owner": "octo",
      "repository": "demo",
      "number": 1
    }
  }
}
```

It executes a tool already modeled by the exact TwinSpec bound to the branch.
It is not a generic command, prompt, provider request, shell task, arbitrary
HTTP callback, MCP client call or Agent wakeup.

## 2. Admission boundary

Only the authenticated private control plane may create an action. The running
server MUST have the branch's loaded runtime and MUST verify before insertion:

1. event/branch/time/priority and global queue bounds;
2. `kind == tool-call` and no signal payload;
3. tool exists in the loaded TwinSpec and is modeled;
4. input is a JSON object, within the 1 MiB input bound and valid against the
   tool's compiled JSON Schema 2020-12 input schema;
5. the action records the runtime's canonical TwinSpec digest;
6. that digest equals the branch's immutable spec binding in the insertion
   transaction.

A store-only control plane MUST reject action admission as
`SCHEDULE_ACTION_UNAVAILABLE`. The client cannot choose or override the stored
spec digest. Invalid admission changes neither queue, head nor audit.

## 3. Execution boundary

Scheduled actions execute only through the explicit bounded
`POST /v1/clock/advance-next` operation with a runtime-backed control plane.
`POST /v1/clock/advance` remains an arbitrary all-or-nothing clock operation;
if its due set includes a pending action it MUST fail with
`SCHEDULE_ACTION_REQUIRED` and commit no due signal prefix.

At execution the runtime MUST recheck:

- branch digest equals loaded runtime digest;
- stored action digest equals branch digest;
- stored tool still exists in that runtime;
- persisted world/scheduler lifecycle is valid.

The action observes virtual `clock = dueAt` and the next monotonic branch
`call_index`. Equal-time signals and actions share the SPEC-0029 total order.

## 4. Isolation

Creation, preview, cancellation and advancement remain control-plane routes and
MUST NOT appear in MCP `tools/list`. The scheduled tool itself remains an
ordinary TwinSpec business tool, but the Agent has no scheduling/fork/reset/
fault/state-inspection capability unless a different product explicitly adds
one under a future ADR.

The callback receives only in-memory modeled state inside the current SQLite
transaction. Hermetic mode has no upstream passthrough or production-write
path. An unmodeled tool or unknown behavior fails explicitly.

## 5. Ordering and mixed batches

A step selects a contiguous prefix from the earliest instant. It MUST NOT skip
an action to execute a later signal or skip a high-priority event to satisfy a
budget. Signal events become `delivered`; action events follow SPEC-0033.

## 6. Non-claims

This subset does not schedule OpenAI/Anthropic requests, Codex/Claude turns,
MCP hosts, remote Episode workers, operating-system processes or external side
effects. It does not provide recurrence, retry, dead letters, compensation,
distributed leases or wall-clock service-level guarantees.
