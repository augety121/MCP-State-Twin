# SPEC-0010 — Upstream Surface Inspection, Binding & Drift

> **Status:** Proposal  
> **Implementation baseline:** Partial source-reported foundation: canonical tool-surface digest/binding exist; automatic inspection/refresh is not verified.

## Inspector scope

Can collect facts:
- protocol version;
- capabilities;
- tools;
- input/output schema;
- descriptions;
- annotations;
- execution metadata;
- extensions;
- authorization profile.

Cannot prove semantic equivalence.

## States

```text
unbound
current
drifted
unknown
incompatible
error
```

## Requirements

### ST-SURFACE-R001
Digest records canonicalization ID + hash.

### ST-SURFACE-R002
A binding that requires `current` fails closed if current status cannot be established.

### ST-SURFACE-R003
Auto-refresh may generate candidate changes but cannot silently modify accepted Twin behavior.

### ST-SURFACE-R004
Classify metadata, annotation, additive schema, compatible contract and breaking contract drift separately.

### ST-SURFACE-R005
Authorization-scoped surfaces are separate profiles.

### ST-SURFACE-R006
Cache expiry/TTL cannot preserve stale `current` status beyond policy.

### ST-SURFACE-R007
Host-transformed surface is not the same as server-native surface.

## Drift severity

- D0 editorial
- D1 additive
- D2 evaluation-significant compatible change
- D3 breaking
- D4 unknown/unsafe

D3/D4 block strict binding.

## Feasibility

**High structurally.** Semantic drift requires SPEC-0011 evidence.
