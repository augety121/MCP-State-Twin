# SPEC-0059: Stored Suite Metadata Admission

Status: accepted by [ADR-0059](ADR-0059-SUITE-EVIDENCE-AUDIT.md).

## Contract

Read existing suite claim, plan and report as bounded strict JSON. Claim/plan
limits are 64 KiB each; each report is 512 KiB. Reject duplicate/unknown fields,
noninteger counters, missing required fields, null substitutions, extra documents,
known sensitive patterns and unsupported formats. JSON key order and whitespace
are immaterial. These are derived artifacts, not signed provenance.

The claim must equal the fixed offline-suite-v1 preflight contract with the
suite-claim format, including exact limits, counts and generated comparison plan.
The plan must match the claim, contain 1–16 pairs and use the positional trial IDs
and direct terminal paths emitted by SPEC-0056. No arbitrary artifact traversal.

Report format/profile, ordered unique trial IDs, planned count, finite lifecycle,
outcome/failure values, explicit upgradeAllowed:false and comparison presence
must be admitted before any saved value enters diagnostic output. Completed
execution requires every row completed with complete evidence/cleanup. A stopped
prefix cannot restart after a failed or unstarted row. Comparison complete requires
a comparison value; other comparison states require none. Semantic comparison
correctness is established only by SPEC-0061, never by decoding.

## Acceptance

Read writer-generated metadata; reject malformed/duplicate/unknown data, missing
false flags, floats/nulls, altered limits, mismatched plan/IDs and sensitive text.
Errors expose finite codes, never parser messages, paths or input content. Keep
the existing suite artifact schema and writer behavior compatible.
