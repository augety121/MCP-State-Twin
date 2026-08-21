# SPEC-0008 — Deterministic Fault & Failure Injection

> **Status:** Proposal  
> **Implementation baseline:** Not verified/treated as not implemented.  
> **Depends on:** SPEC-0007 for time-dependent faults.

## Fault definition

```yaml
id: ambiguous-close
selector: ...
trigger: ...
phase: after-commit-before-response
outcome:
  type: timeout
repeat:
  count: 1
```

Fault outcome must be derived only from declared environment/fault/call state.

## Required phases

- before-validation
- after-validation
- before-preconditions
- after-preconditions
- before-effect
- after-effect-before-commit
- after-commit-before-response
- response-delivery
- scheduled-visibility
- future task completion

## Classes

- domain;
- transport;
- service/rate limit;
- consistency;
- crash;
- ambiguous outcome.

## Requirements

### ST-FAULT-R001
No uncontrolled random fault selection.

### ST-FAULT-R002
Probability syntax, if later supported, MUST compile into a seeded deterministic plan whose identity is recorded.

### ST-FAULT-R003
`after-commit-before-response` preserves state commit while caller observes failure.

### ST-FAULT-R004
Partial **business** effect must be represented as a valid atomic world state. Do not intentionally corrupt SQLite atomicity.

### ST-FAULT-R005
Fault counters default branch-local.

### ST-FAULT-R006
Fault controls stay private control-plane operations.

### ST-FAULT-R007
Fault plan digest is part of environment identity.

### ST-FAULT-R008
Time-dependent fault uses virtual time.

### ST-FAULT-R009
Fault event is auditable even when response delivery fails.

### ST-FAULT-R010
Fault plan cardinality/state is bounded.

### ST-FAULT-R011
Retries MUST be observable as distinct calls unless an explicit idempotency semantic collapses business effect.

## Mandatory acceptance scenarios

1. fail before effect
2. commit then response lost
3. safe retry with idempotency
4. unsafe duplicate without idempotency
5. rate-limit -> virtual time advance -> success
6. stale read -> delayed visibility -> fresh read
7. crash before commit
8. crash after commit
9. cancellation before commit
10. cancellation racing commit

## Feasibility

**High** for controlled semantic faults.  
**Medium** for claims about realistic OS/storage crash behavior; use explicit killpoints and bounded profiles.
