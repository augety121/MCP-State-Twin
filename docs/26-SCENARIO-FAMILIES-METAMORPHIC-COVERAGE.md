# Scenario Families & Metamorphic Coverage

> **Status:** Proposal / Unverified  
> **Purpose:** Expand evaluation coverage without pretending exhaustive enumeration is possible.

## 1. Problem

A fixed set of public scenarios can be:
- memorized;
- overfit;
- too narrow;
- brittle to irrelevant details.

The answer is not uncontrolled random generation.

The answer is **deterministic scenario families**.

## 2. Scenario family

A family defines:

```text
generator version
parameter schema
seed
pre-state partitions
objective template
fault profile choices
metamorphic transforms
assertion generator
limits
```

## 3. Deterministic generation

### ST-FAM-R001
Given the same generator version, family, seed and parameters, the generated episode artifacts MUST be identical.

### ST-FAM-R002
Generator version MUST be included in environment identity.

### ST-FAM-R003
A change that alters generated semantics requires a generator-version change.

## 4. Parameter partitions

Examples:
- entity exists/missing;
- open/closed;
- zero/one/many related records;
- permission allowed/denied;
- first attempt/retry;
- rate-limit active/inactive;
- visibility delay;
- conflicting writer.

Partitions should be risk-driven.

## 5. Metamorphic transforms

A metamorphic test changes irrelevant or predictably related aspects while preserving a property.

Examples:

### Identifier renaming
Rename issue IDs/repository names while expected semantic success remains.

### Irrelevant ordering
Reorder unrelated collections where order is not semantically meaningful.

### Non-semantic text variation
Change description/body text that should not alter the required transition.

### Time shift
Shift the entire virtual-time origin while preserving relative timing behavior.

### Extra distractor entity
Add unrelated state; task should still target correct entity.

### Equivalent state encoding
Where canonical semantics permit multiple source encodings, final behavior should agree.

### Fault insertion
Insert an allowed transient failure; recovery invariant should still hold.

## 6. Metamorphic properties

Examples:

```text
renaming invariant
irrelevant-state invariance
retry safety
idempotency
monotonic state transition
no cross-entity contamination
fork isolation
time-shift invariance
```

## 7. Negative metamorphic tests

Some transforms should change the correct answer.

Example:
- close target already closed;
- permission revoked;
- visibility delay beyond deadline.

The family must explicitly classify expected semantic impact.

## 8. Held-out seeds

Benchmark policy MAY use:
- public training seeds;
- public validation generator;
- private/held-out evaluation seeds.

Seed secrecy is useful only if the generator does not trivially encode the answer.

## 9. No generative semantic invention

AI may help propose scenario-family candidates.

But expected invariants must come from accepted Twin semantics.

The generator cannot invent new upstream behavior.

## 10. Differential integration

For L2 coverage:
- family instances may run against Twin and authorized upstream;
- upstream nondeterminism is separately classified;
- only proven coverage partitions are promoted.

## 11. Shrinking/minimization

When a generated scenario fails, the harness MAY attempt deterministic minimization:
- reduce unrelated entities;
- shorten trace;
- simplify parameters.

The minimized reproduction must preserve:
- failure class;
- relevant invariant failure;
- generator/provenance link.

## 12. Coverage reporting

Report coverage by dimensions, not one percentage alone.

Example:

```yaml
coverage:
  tools: 6/6
  inputPartitions: ...
  errorClasses: ...
  faultClasses: ...
  timeProfiles: ...
  concurrencyProfiles: ...
  hostProfiles: ...
```

## 13. Feasibility

**High.**

It is a scalable way to broaden coverage while preserving reproducibility and anti-overfitting properties.
