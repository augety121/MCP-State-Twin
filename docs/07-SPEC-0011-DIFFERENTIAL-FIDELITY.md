# SPEC-0011 — Differential Validation & Fidelity Promotion

> **Status:** Proposal  
> **Implementation baseline:** Not verified/treated as not implemented.

## Core rule

Fidelity is attached to a **coverage profile**, not automatically to an entire service.

Good:

```text
close_issue / success+missing+already-closed /
authorized sandbox profile X / L2
```

Too broad:

```text
GitHub Twin = L2
```

unless the project can actually justify that scope.

## Differential architecture

```text
same semantic case
     /      \
  Twin    Authorized sandbox/upstream
     \      /
     comparator
```

## Comparison dimensions

- surface;
- accepted input;
- error/success class;
- output;
- observable state;
- idempotency;
- retry;
- timing class;
- auth scope;
- visibility.

## Result taxonomy

```text
MATCH
ACCEPTED_DIVERGENCE
UNMODELED
UPSTREAM_NONDETERMINISTIC
UPSTREAM_UNAVAILABLE
TEST_INVALID
SECURITY_BLOCKED
FAIL
```

## Requirements

### ST-DIFF-R001
Accepted divergence has owner, rationale, scope and review/expiry condition.

### ST-DIFF-R002
Unknown upstream nondeterminism stays unknown; do not force it into a deterministic oracle.

### ST-DIFF-R003
Production differential tests require explicit authorization; sandbox/test service preferred.

### ST-DIFF-R004
Credentials never appear in evidence.

### ST-DIFF-R005
Evidence identifies upstream profile/surface/date.

### ST-DIFF-R006
L2 requires human-reviewed rules + differential evidence in declared coverage.

### ST-DIFF-R007
AI-generated/inferred behavior cannot self-promote.

### ST-DIFF-R008
A material upstream drift can downgrade verification state until revalidated.

## Coverage dimensions

```text
tool
x operation
x input partition
x pre-state
x success/error
x transition
x retry/idempotency
x relevant fault
x authorization profile
```

Full Cartesian coverage is usually infeasible. Use risk-based partitions and document exclusions.

## Feasibility

**Medium-high.** Hardest issues: upstream oracle quality, nondeterminism, legal/credential constraints.
