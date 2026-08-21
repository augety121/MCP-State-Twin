# Portable Surface Projection & Compatibility Lint

> **Status:** Proposal / Unverified  
> **Purpose:** Make host incompatibilities visible before a real agent run.

## 1. Surface layers

```text
S0 Canonical TwinSpec surface
S1 MCP-native server surface
S2 authorization-scoped surface
S3 host-projected surface
S4 model-visible surface
```

Each layer has its own digest where observable.

## 2. Projection record

Use `templates/HOST-SURFACE-PROJECTION.yaml`.

## 3. Transformation classes

### Name
- prefix
- suffix
- sanitize
- truncate
- collision remap

### Schema
- remove keyword
- rewrite type
- flatten union
- remove default
- reject unsupported construct

### Output
- text serialization
- file indirection
- truncation
- structured-content removal

### Surface
- tool filtering
- progressive loading
- context-driven deferred exposure

## 4. Loss analysis

### `none`
No observable semantic loss.

### `syntactic`
Representation changes but accepted input/output semantics are expected to be preserved.

Still requires fixture verification.

### `semantic-risk`
Some valid canonical inputs/results may no longer be representable or host-visible.

Blocks strict compatibility/fidelity until tested.

### `unsupported`
Cannot safely project.

### `unknown`
Insufficient evidence.

## 5. Linter checks

A compatibility linter SHOULD check:

```text
tool name after transform
name collisions
tool count
feature requirements
schema keyword support
input/output representability
required transport
auth profile
approval requirements
result size
multimodal support
dynamic surface policy
```

## 6. Target sets

Lint against one or many hosts:

```yaml
targets:
  - claude-code
  - gemini-cli
  - cursor
```

A multi-target lint computes:
- common portable subset;
- per-host warnings;
- impossible intersections.

Do not mutate canonical surface automatically merely to make every host pass.

## 7. Suggested remediation

The linter MAY suggest:
- shorten canonical tool name;
- expose smaller scenario-specific tool set;
- avoid unsupported optional schema construct;
- switch transport;
- select different host profile.

Suggestions are not automatic semantic changes.

## 8. Exact-fidelity protection

If a bound upstream contract requires a construct one host cannot represent:
- keep canonical/upstream contract;
- mark that host/profile unsupported or semantic-risk;
- do not weaken the Twin and call it equivalent.

## 9. Projection fixture

Every nontrivial host projection SHOULD have golden fixtures:

```text
canonical tool/schema
 -> projected tool/schema
 -> transform log
 -> expected loss class
```

## 10. Dynamic surface readiness

The linter cannot guarantee runtime host visibility.

Real run must additionally prove:
- server connected;
- required projected tools visible;
- surface settled under the host's loading model.

## 11. Feasibility

**High.**

This should become one of the strongest practical differentiators of MCP State Twin as the host ecosystem fragments.
