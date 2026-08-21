# SPEC-0016 — Reproducible Evaluation Bundle

> **Status:** Proposal  
> **Implementation baseline:** not implemented/verified  
> **Verification status:** unverified

## 1. Goal

Make an evaluation environment portable without depending on an unpinned Git checkout.

The bundle is the unit a developer can hand to:
- CI;
- Codex harness;
- Claude harness;
- another team;
- benchmark runner.

## 2. Logical layout

```text
TwinBundle/
  manifest.yaml
  spec/
    twin.yaml
  fixture/
    state.json
  scenarios/
  schemas/
  policy/
  expected/
  provenance/
```

## 3. Independent versions

The bundle manifest MUST identify:

```text
bundle_schema_version
TwinSpec_api_version
runtime_compatibility
MCP_protocol_profile
canonicalization_id
scenario_schema_version
evidence_schema_version
required_extensions
```

These are independent.

## 4. Component identity

### ST-BUNDLE-R001
Every required component MUST have an expected content digest.

### ST-BUNDLE-R002
Import MUST verify digests before execution.

### ST-BUNDLE-R003
An unsupported version MUST fail explicitly.

### ST-BUNDLE-R004
A missing required component MUST fail before starting the evaluation.

## 5. Hermeticity

### ST-BUNDLE-R010
Hermetic bundles MUST NOT require undeclared network downloads.

### ST-BUNDLE-R011
External schema references are rejected under hermetic profile.

### ST-BUNDLE-R012
If a non-hermetic dependency is allowed, its identity/version/digest and retrieval policy MUST be declared.

## 6. Archive security

### ST-BUNDLE-R020
Reject `../` traversal and absolute paths.

### ST-BUNDLE-R021
Reject symlink escape.

### ST-BUNDLE-R022
Enforce compressed size, extracted size and file-count limits.

### ST-BUNDLE-R023
Do not execute code merely because it exists in the bundle.

## 7. Provenance

Optional later fields:

```yaml
provenance:
  created_by:
    tool: statetwin
    version: "..."
  build_attestation: null
  sbom: null
  signature_bundle: null
```

## 8. Signatures

Future releases MAY sign bundle artifacts using Sigstore/Cosign or another governed scheme.

A valid signature proves artifact origin/integrity under the selected trust policy. It does **not** prove:
- fidelity;
- security;
- correctness;
- compatibility.

## 9. Expected evidence

A bundle MAY include expected deterministic digests for scripted scenarios.

Live-agent scenarios SHOULD usually include terminal invariants rather than one exact trajectory.

## 10. Import result

Import SHOULD produce a resolved environment manifest:

```text
bundle id
verified component digests
runtime version
resolved spec version
protocol profile
canonicalization
limit profile
extensions
```

This resolved manifest becomes part of run evidence.

## 11. Feasibility

**High.**

Core work is packaging/versioning/security. SBOM/provenance/signing can be layered later without changing world semantics.
