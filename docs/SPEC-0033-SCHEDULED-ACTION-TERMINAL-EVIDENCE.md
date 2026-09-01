# SPEC-0033: Scheduled-action Terminal Lifecycle and Evidence

- **Status:** Accepted via ADR-0033
- **Implementation status:** implemented one-attempt local subset
- **Verification status:** success, domain failure, after-effect fault, crash rollback, call-count and evidence tests

## 1. Lifecycle

An action has exactly these current transitions:

```text
pending ──cancel──> canceled
pending ──execute─> completed
pending ──execute─> failed
```

`completed`, `failed` and `canceled` are terminal. A terminal action is never
selected again. Current `maxScheduledAttempts` is 1; there is no automatic
retry transition.

## 2. Terminal record

An executed action records:

```text
attemptCount      = 1
finishedAt        = dueAt
outcome.callIndex = allocated branch call index
outcome.result
outcome.errorClass
outcome.effectCommitted
outcome.faultId / faultPhase, when applicable
```

`completed` requires an empty error class and `effectCommitted = true`.
`failed` requires a typed non-empty error class and may have either effect
state:

| Failure | effectCommitted |
|---|---:|
| schema/precondition/not-found/conflict/unmodeled/before-effect fault | false |
| deterministic `TIMEOUT_AFTER_EFFECT` fault | true |

This distinction prevents a response-loss simulation from being mistaken for
an effect that never occurred.

## 3. Atomic transaction

One scheduler step transaction includes all selected event lifecycles and:

- optional virtual-clock movement;
- TwinSpec effects and invariants;
- branch-local entropy/sequence changes;
- fault-plan selection and counter consumption;
- one tool-audit row per action;
- one fault-event row when fired;
- action outcome records;
- branch state/digest, `call_count`, one aggregate `head_version` increment;
- aggregate `scheduler.advance.next` control audit.

An infrastructure error, callback error, corrupt state, invalid result, budget
failure, state write failure, audit write failure or commit failure rolls back
the entire selected prefix. Pending actions, clock, call count, fault counters,
audit and modeled entities then remain at the prior committed head.

## 4. Domain failure versus infrastructure failure

A modeled/domain failure is a deterministic tool result. It consumes one call
index and commits `failed` evidence without committing candidate tool state.
An infrastructure failure returns an operation error and leaves the action
pending because no new head exists.

This rule allows a caller to retry an uncommitted scheduler step using the same
expected head. Retrying a committed step with the old expected head conflicts
and cannot execute the action twice.

## 5. Audit chain

Each action's `outcome.callIndex` identifies its append-only tool audit row.
The row records canonical input/result, error class and before/after complete
world-state digests. Multiple actions in one scheduler transaction form a
digest chain in total order; the final action digest and subsequent signal
lifecycle lead to the aggregate branch digest.

Host `createdAt` remains operational metadata and is not deterministic world
identity.

## 6. Exactly-once vocabulary

Within one local hermetic branch, transaction coupling provides one terminal
modeled execution per scheduled event. The project MUST NOT generalize this to
exactly-once provider requests, MCP calls, HTTP effects, remote workers or
production writes. External effects remain unsupported here.
