# SPEC-0054: Host Report Scope Assessment

Status: accepted by [ADR-0054](ADR-0054-SCOPED-HOST-REPORT-ASSESSMENT.md).
Comparison policy `host-report-scope-v1`. TTL remains `host-report-time-v1`.

## 1. Command and result

```text
statetwin compatibility assess --report report.yaml --target target.yaml --at <UTC-time>
statetwin compatibility assess --report report.yaml --target target.yaml --at <UTC-time> --require-current
```

Target is optional only for plain/time-only assessment. `--require-current`
MUST require a target and explicit assessment time before loading input.
The operation validates both inputs, retains SPEC-0052 temporal classification,
and compares every identity field below. Invalid report/target/time returns no
successful JSON. Valid mismatches produce a diagnostic report. Output write
failures remain errors, never swallowed by an eligibility decision.

Target mode sets `scopeStatus` to `matched` or `mismatched` and adds `scope`:
`policy`, ordered `mismatches` (empty array for match), `claimCurrentEligible`.
Eligibility is `claimTimeEligible && scopeStatus == matched`, not a verification
of the declared run. There is no implicit clock, default target, network call,
credential discovery, source rewrite or file hashing.

## 2. Comparison contract

Exact comparisons, in fixed order:

1. runtime: version, revision, specDigest, surfaceDigest, snapshotDigest;
2. host: profile, name, version, provider, requestedModel, model;
3. mcp: configuredVersion, negotiatedVersion, transport, endpointTrust,
   deploymentProfileDigest, observedSurfaceDigest, surfaceStatus;
4. claim.procedureDigest;
5. trial: scenarioDigest, promptDigest, toolPolicyDigest, then each of the seven
   limits in SPEC-0053 order;
6. redaction.policy.

Each mismatch is only the report field path from this finite list. Preserve all
differences; no first-error-only shortcut, map-order nondeterminism, value echo,
version-range compatibility, wildcard, case folding or alias resolution. Profile
changes cannot inherit time eligibility from another product. Existing TTL is
evaluated on the report profile; mismatch independently blocks current eligibility.

Observation date, expiry and declared claim level are separately handled by
SPEC-0052; trial index, execution outcome/assertions and observed state/trace IDs
are not target identity. They remain subject to report validation, not silently
equalized or replaced. Unknown new target fields are refused.

## 3. Exit and authority boundaries

Without a gate, valid diagnostic results exit zero even if stale/mismatched.
`--require-current` prints diagnostics then fails `HOST_REPORT_NOT_CURRENT`
unless claimCurrentEligible is true. If both gates are supplied, current gate
takes precedence; it includes the old freshness condition. `--require-fresh`
alone remains time-only even with a target and may succeed with scope mismatch;
the output still exposes that mismatch. Documentation MUST make this explicit.

All outputs retain `publicationAllowed: false`, `provenance: not_verified`.
Matched means declarations match, not actual currently running configuration.
The target lacks full adapter/version/capability identity; this cannot close
HostProfile v1alpha2, artifact/provenance checks, revocation or live gates.

## 4. Acceptance

Every compared dimension has a negative test. Test same-profile/six-profile
matches, API/product and generic/custom mismatches, optional missing-to-present
changes, budget changes including zero, several simultaneous changes with stable
ordering, unchanged source bytes, deterministic JSON, in-window mismatch,
expired match, experimental/regressed match, invalid input and output failure.
CLI tests MUST prove require-current without target fails before IO, and both
gate precedence and time-only backward compatibility. Full suite/vet/race and
exact-revision CI are required; no provider calls are part of acceptance.
