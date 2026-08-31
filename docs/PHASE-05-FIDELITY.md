# Phase 5: Recorder, Replay and L2 Fidelity

- **Target:** `v0.4.x`
- **Entry:** stable deterministic/evidence formats and reviewed privacy policy

## Required scope

- opt-in recorder with consent and source identity;
- secret and personal-data redaction before persistence;
- bounded cassette format and deterministic replay;
- incomplete/failed trace semantics;
- upstream tool-surface inspector and drift states;
- differential runner against disposable upstream fixtures;
- contract, invariant, error and field coverage reports;
- human-reviewed L2 admission and revocation workflow.

## Safety rules

Recording is evidence, not truth. Inferred behavior remains draft/L1 candidate.
An LLM, compiler or trace corpus cannot promote a Twin to L2 automatically.
Production credentials and private traces are never repository fixtures.

## Exit evidence

- adversarial redaction and secret-absence suite;
- exact cassette replay digest;
- incomplete trace cannot produce invented success;
- upstream drift changes admission state;
- differential mismatch is explicit and reproducible;
- at least one reference twin passes reviewed L2 criteria;
- coverage exclusions are published with the fidelity claim.
