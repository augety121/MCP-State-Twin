# SPEC Governance — Claims, Versioning & Change Control

> **Status:** Proposal  
> **Implementation:** N/A  
> **Verification:** Unverified  
> **Adoption order:** First

## 1. Why this spec exists

MCP State Twin will eventually have several independent moving parts:
- runtime;
- TwinSpec;
- MCP protocol;
- storage;
- evidence;
- host adapters;
- provider/model versions;
- bundle formats.

Without strict governance, a README can easily claim more than the executable evidence proves.

## 2. Four independent status axes

Every externally meaningful SPEC/RFC SHOULD declare:

```yaml
spec_status: draft | accepted | superseded | retired
implementation_status: none | partial | complete
verification_status: unverified | partially_verified | verified
release_scope: phase-0 | v0.1 | v0.2 | v0.3 | v1.0 | future
```

No status implies another.

## 3. Public claim vocabulary

Allowed:

### `specified`
Normative text exists.

### `implemented`
Code exists and code-level tests cover the declared implementation.

### `verified`
Pinned repeatable evidence satisfies a verification profile.

### `compatible`
A named host/product/version/profile passed a named capability/scenario set.

### `fidelity-qualified`
A named behavior coverage profile meets a declared fidelity level.

### `production-ready`
A separately defined production profile has passed security, recovery, upgrade and operations gates.

Forbidden without explicit bounded scope:
- “fully compatible”
- “production equivalent”
- “GitHub equivalent”
- “Claude compatible”
- “OpenAI compatible”
- “AGI-ready”
- “perfect twin”

## 4. Stable requirement IDs

Prefixes:

| Domain | Prefix |
|---|---|
| governance | `ST-GOV-` |
| MCP | `ST-MCP-` |
| time/entropy | `ST-VTIME-` |
| faults | `ST-FAULT-` |
| replay | `ST-REPLAY-` |
| surface | `ST-SURFACE-` |
| differential | `ST-DIFF-` |
| storage | `ST-STORE-` |
| security | `ST-SEC-` |
| evidence | `ST-EVID-` |
| limits | `ST-LIMIT-` |
| bundle | `ST-BUNDLE-` |
| host | `ST-HOST-` |
| future | `ST-FUTURE-` |
| release | `ST-REL-` |

IDs MUST NOT be reused.

## 5. Normative hierarchy

Recommended order:

1. Accepted SPEC requirements
2. Accepted ADR resolving implementation choice
3. Accepted RFC defining product/release boundary
4. executable conformance/evidence
5. Implementation Status
6. README

README cannot redefine semantics.

## 6. Change classes

### Editorial
No observable behavior changes.

### Compatible semantic
New opt-in or additive behavior that preserves existing accepted inputs.

### Breaking
Any change to:
- canonical bytes/digest;
- error class;
- input/output contract;
- state transition;
- snapshot identity;
- evidence schema;
- protocol behavior;
- auth behavior;
- storage interpretation.

Breaking changes MUST carry migration/version impact.

## 7. Canonicalization policy

Every digest MUST identify both:

```yaml
hash: sha256
canonicalization_id: statetwin-canonical-v1alpha1
```

A switch to RFC 8785/JCS or another scheme MUST:
- use a new canonicalization ID;
- run a compatibility corpus;
- define historical evidence behavior;
- never silently reinterpret old digests.

## 8. Evidence-first lifecycle

```text
Problem
 -> primary-source research
 -> RFC/SPEC
 -> ADR
 -> stable requirement IDs
 -> implementation
 -> tests
 -> evidence
 -> Implementation Status
 -> release claim
```

## 9. AI-generated artifacts

An AI MAY:
- propose SPECs;
- infer candidate behavior;
- generate tests;
- summarize drift;
- suggest patches.

It MUST NOT self-certify:
- compatibility;
- fidelity;
- security;
- production readiness.

Generated semantics remain `candidate` until reviewed and evidenced.

## 10. Closed-world anti-hallucination rule

If behavior is not represented by an accepted contract, the runtime returns a typed `UNMODELED` / unsupported outcome.

It MUST NOT generate “plausible” behavior as fallback.

## 11. Source freshness

Time-sensitive claims record:
- source ID;
- observed date;
- software/protocol version if available.

Host/provider claims SHOULD have a revalidation date/policy.

## 12. SPEC acceptance checklist

A SPEC cannot become Accepted until:
- goals/non-goals clear;
- terminology defined;
- state machine/data model defined when relevant;
- determinism boundary defined;
- failure/cancellation semantics defined;
- limits defined;
- security/privacy reviewed;
- migration/version impact defined;
- evidence plan defined;
- release gate defined;
- blocking open questions resolved.

## 13. Definition of done

`Done` means:

```text
accepted requirement
+ implementation
+ executable tests
+ evidence
+ documentation
```

Not merely merged code.
