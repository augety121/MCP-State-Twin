# SPEC-0037: CEL Null and the JSON Value Boundary

- **Status:** Accepted via ADR-0037
- **Date:** 2026-09-02
- **Scope:** native CEL result conversion before JSON/schema/state processing

## 1. Defect and consequence

Before the fix, `has(input.missing) ? input.missing : null` produced canonical
JSON `0`, not `null`, under the existing CEL v0.31.0 dependency. The new
regression test reproduced this before dependency migration. CEL's conversion
to `interface{}` exposes null as `structpb.NullValue_NULL_VALUE`, whose numeric
representation is zero. Generic Go JSON encoding therefore lost its meaning.

This is a semantic error, not an acceptable representation alias: JSON null
and numeric zero have different schema, equality and digest meanings.

## 2. Required conversion

| Native value | Admitted normalized value |
|---|---|
| protobuf `NullValue_NULL_VALUE` | Go `nil`, JSON `null` |
| unknown numeric value of protobuf `NullValue` | explicit error |
| ordinary integer or float zero | unchanged numeric zero |
| missing map key | remains absent; no automatic null insertion |
| explicit null-valued map key | key retained with null value |
| map/list containing null | normalize recursively at all visited levels |
| non-string map key | explicit conversion error |

Normalization occurs after CEL evaluation and before consumers encode JSON,
validate output schemas, apply object effects or compute canonical digests.
The change MUST NOT convert every zero-valued enum or integer into null and
MUST NOT rewrite persisted data heuristically. Existing NaN/infinity and
non-JSON refusals remain governed by ADR-0005 and downstream validation.

No `recover` handler may turn an unknown value into a successful null result.
An unsupported/invalid value follows existing fail-closed error handling.

## 3. Executable acceptance

- Top-level null serializes as literal `null`.
- Nested map/list nulls retain null, while a neighboring integer zero remains
  `0` in the same canonical result.
- A nullable tool output passes a JSON Schema `type: null` check.
- A modeled insert stores a present null property; re-reading branch state
  preserves key presence and nil value.
- Unknown protobuf null enum values are refused.
- Ordinary arithmetic, map/list operations and explicit error cases continue
  to pass the dependency-compatibility vectors.

The implementation is covered by `TestExpressionCompatibilityVectors`,
`TestExpressionCompatibilityKeepsBoundsAndNoExternalFunctions` and
`TestExpressionNullSurvivesSchemaAndStoredState` in the engine package.

## 4. Compatibility and recovery

This intentionally changes outputs and derived state digests for trajectories
that previously materialized CEL null as zero. It is an unreleased-preview
semantic bug fix and must appear in the changelog. Do not claim old/new runtime
replay equivalence for those trajectories, even if TwinSpec/surface digests
are equal. Use exact runtime revision when evaluating or comparing evidence.

The canonical JSON algorithm, TwinSpec syntax, schema storage layout and
ResourceProfile limits do not change. Existing stored zero remains zero:
there is no information proving whether it came from null conversion or was
legitimate domain data. Existing snapshots/audit/Episode evidence MUST NOT be
edited or rehashed in place. Re-evaluate from reviewed synthetic fixtures
under a new runtime revision when corrected semantics are required.

If a preview consumer relied on the erroneous zero, it must express an
explicit numeric default in its TwinSpec and revalidate the resulting spec.
Rolling back runtime code restores old behavior for new calls, not semantic
equivalence between evidence generated before and after this correction.

## 5. Non-claims

This is not complete arbitrary protobuf conversion, full RFC 8785 support,
a storage migration or proof that all possible CEL/JSON boundary defects have
been found. Future native type additions require separate reviewed admission
rules and tests; no implicit expansion of the supported value domain occurs.
