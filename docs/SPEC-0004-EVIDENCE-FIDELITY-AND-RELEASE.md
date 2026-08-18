# SPEC-0004: Evidence, Fidelity, and Release

- **Status:** Proposed normative specification

## 1. Evidence model

Every claim in README, release notes, or CLI output MUST map to one of:

1. executable test;
2. architectural absence with a negative test or reviewable proof;
3. reproducible CI artifact;
4. explicit unsupported/partial status.

Roadmap text MUST NOT be used as implementation evidence.

## 2. Provenance

Twin behavior is classified as `observed`, `inferred`, `declared`, or
`verified`. Generated or trace-assisted content starts as `inferred` and
`unverified`. Human review alone is insufficient for L2; it must be paired
with deterministic contract and differential evidence.

## 3. Fidelity levels

- **L0:** exact or rule-based cassette replay;
- **L1:** stateful template with declared limitations;
- **L2:** reviewed contract-backed behavior with coverage and differential data;
- **L3:** native/reference implementation with a documented authority.

The bundled issue-tracker world is currently an L1 synthetic fixture. The
project MUST NOT call it L2/L3 or production-equivalent without a promotion
record.

## 4. Scenario report

Each run SHOULD emit, and a stable release MUST emit:

```text
environmentDigest
agentIdentity
initialSnapshotDigest
orderedToolTrace
terminalStateDigest
stateDiff
invariants
configuredFaults
firedFaults
unsupportedBehaviors
```

Secrets, authorization headers, raw production traces, and personal data MUST
not be included in committed evidence.

## 5. Stable release gates

`v0.1.0` MUST NOT be tagged until all required gates pass or are explicitly
removed from the profile:

- deterministic replay and race tests;
- schema, MCP conformance, fuzz, and hermetic egress tests;
- storage identity, migration, and crash kill-point fixtures;
- control-plane isolation and audit tests;
- pinned dependency/action and secret scanning;
- at least two independent synthetic stateful domains;
- documented OpenAI-family and Anthropic-family protocol smoke runs;
- P0 failure rows mapped to evidence or an accepted exclusion;
- README/status/RFC consistency review.

An alpha release MAY leave gates open only when the exact open gates are
published. Performance targets are measurements with hardware and command
context, never universal guarantees.

## 6. Change control

Changes to canonical JSON, TwinSpec semantics, error classes, MCP boundary,
storage identity, deterministic scheduling, or trust zones require an ADR and
compatibility note. Every incident MUST add a regression test or a documented
reason why testing is impossible.
