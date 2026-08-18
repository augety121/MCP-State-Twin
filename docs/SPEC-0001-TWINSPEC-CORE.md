# SPEC-0001: TwinSpec Core

- **Status:** Proposed normative specification
- **Version:** `statetwin.dev/v1alpha1`
- **Scope:** declarative stateful MCP tool worlds
- **Normative language:** RFC 2119 / RFC 8174

## 1. Purpose

TwinSpec is the versioned, reviewable intermediate representation for a
stateful MCP tool world. It describes the model-facing tool contract and the
deterministic state transition rules. It is not a model prompt, agent plan,
workflow definition, or executable plugin.

The v1alpha1 profile is intentionally smaller than the long-term design. A
field or component mentioned in another document is not part of this profile
unless it is defined here and has executable evidence.

## 2. Top-level document

Every document MUST contain exactly one YAML document with:

```yaml
apiVersion: statetwin.dev/v1alpha1
kind: Twin
metadata: {}
clock: {}
state: {}
tools: []
```

Unknown typed fields MUST be rejected. The document MUST be at most 1 MiB.
YAML aliases, multiple documents, remote references, and native extensions are
not supported by v1alpha1.

## 3. Metadata and binding

`metadata.name` is a stable local identifier. `metadata.upstream` contains:

- `protocol`, which MUST be `mcp` in this profile;
- `status`: `current`, `drifted`, `unknown`, or `unbound`;
- `surfaceDigest` when status is not `unbound`.

`metadata.fidelity.level` is `L0`, `L1`, `L2`, or `L3`; `L0` and `L1` MUST NOT
claim `verified`. A manually declared digest is a binding, not proof that an
upstream service was inspected. Automatic inspection is a separate extension.

## 4. State model

State is a finite JSON-shaped value partitioned into named entities. Each
entity has a stable key definition. Entity names and keys MUST be deterministic
and MUST NOT depend on host process memory addresses or wall-clock time.

The v1alpha1 engine supports declarative `allocate`, `insert`, `update`, and
`delete` effects. Arbitrary code, filesystem access, network access, and
reflection are forbidden in an effect.

## 5. Tool contract

Each tool MUST declare a unique name, human-readable description, input JSON
Schema, and optional output JSON Schema. The runtime MUST expose these fields
through MCP `tools/list` exactly as declared, subject only to the documented
annotation derivation.

Optional transition sections are evaluated in this order:

```text
input validation
-> preconditions
-> effects
-> query/result construction
-> postconditions
-> global invariants
-> output validation
-> commit
```

An omitted rule means “not modeled”; it does not authorize the runtime to
infer a hidden side effect.

## 6. Expression boundary

Expressions use the bounded CEL environment defined by SPEC-0002. Tool input
is data only and MUST never be parsed as expression source. Source is limited
to 4,096 bytes and evaluation uses a cost limit of 10,000 in the v0.1 profile.

Expressions may access only `input`, `state`, `vars`, `item`, `clock`, and
`call_index`. They MUST NOT access the filesystem, network, environment
variables, reflection, or unbounded host functions.

## 7. Canonicalization

TwinSpec digest uses the repository canonical JSON contract. MCP surface digest
uses the envelope from ADR-0008: sorted tool names, descriptions, input/output
schemas, and effective read-only/destructive/open-world annotations.

YAML formatting and tool list order MUST NOT change a surface digest. A change
to a description, schema, name, or effective annotation MUST change it.

## 8. Admission rules

The loader MUST reject malformed documents before runtime construction. Runtime
startup MUST:

- accept `unbound` without claiming upstream equivalence;
- accept `current` only when the declared digest equals the computed surface;
- reject `drifted` and `unknown` with `SPEC_DRIFT`;
- reject unsupported schema or expression features explicitly.

## 9. Future extensions

Reads/writes dependency declarations, idempotency ledgers, virtual-time
advancement, fault schedules, provenance per transition, signatures, and
native/reference adapters require a new compatible profile or an accepted ADR.
They MUST NOT be silently added to v1alpha1 semantics.
