# SPEC-0021: Claim Registry and Evidence Freshness

- **Status:** Accepted via ADR-0021
- **Registry:** `docs/CLAIM-REGISTRY.md`
- **Target:** all releases
- **Depends on:** SPEC-0004, SPEC-0019

## 1. Rule

Every externally visible implementation, compatibility, fidelity, security or
performance claim MUST have a stable claim ID and map to one of:

- executable tests;
- a public CI job/artifact;
- an accepted architectural proof;
- an explicit `unsupported`, `unverified`, `experimental`, `regressed` or
  `stale` state.

Proposal prose is not evidence.

## 2. Claim record

Each record contains:

```text
claim ID
exact statement
scope/profile/version
state
requirement IDs
evidence command or artifact
last verified revision/date
freshness rule
known exclusions
owner
```

## 3. Admission

A claim becomes `verified` only when its evidence passes on the exact candidate
revision and every security/identity prerequisite is satisfied. A mock test may
verify adapter contract shape but cannot verify a provider or product profile.

## 4. Invalidation

A verified claim becomes `stale` or `regressed` when:

- its TTL expires;
- a bound protocol/format/schema/profile/adapter changes;
- the supporting test is removed or skipped;
- the exact CI gate fails;
- a security incident invalidates the evidence;
- upstream drift changes a relevant observable contract.

## 5. Public surfaces

README translations, release notes, GitHub descriptions and application text
MUST agree with the registry. Short marketing prose MAY omit implementation
detail but MUST NOT widen scope.
