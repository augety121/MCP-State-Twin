# ADR-0053: Explicit Host Compatibility Target

- Status: Accepted; bounded B15 declaration matching, not full HostProfile v1alpha2
- Date: 2026-09-26
- Contract: [SPEC-0053](SPEC-0053-HOST-COMPATIBILITY-TARGET.md)

SPEC-0052 assesses report age only. A still-in-window report must not be reused
after an operator changes the runtime, host, model, protocol, surface, deployment
or trial policy. Introduce an independent, strict HostCompatibilityTarget input
that declares the expected legacy report identity without inventing execution
results, assertions, timestamps, request IDs or provenance.

The target must be operator supplied, not silently extracted from the report
being checked. Identity matching is equality of supplied declarations, not live
discovery, file integrity verification or an authenticated configuration source.
The target has no digest operation and no provider or filesystem traversal.

No report migration or rewrite is required. Target admission reuses relevant
report identity rules and requires explicit limits, including zero budgets.
Unknown fields and unsupported identity dimensions fail rather than disappear.
Full adapter/security capability identity, revocation and referenced-artifact
verification remain outside this legacy report subset.

Implementation review reproduced YAML float-to-integer truncation in report
failed/passed counters, trial index and budgets, and in the new target decoder.
For example `failed: 0.5` became zero. Both file decoders must require original
integer scalar tokens and explicit presence of budgets/counters. This deliberate
preview admission tightening does not alter valid integer report serialization
or rewrite historical evidence. Direct Go callers already supply typed integers.
