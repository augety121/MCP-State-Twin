# SPEC-0053: Host Compatibility Target Admission

Status: accepted by [ADR-0053](ADR-0053-EXPLICIT-HOST-COMPATIBILITY-TARGET.md).
Format `statetwin.dev/host-compatibility-target/v1alpha1`;
apiVersion `statetwin.dev/v1alpha1`, kind `HostCompatibilityTarget`.

## 1. Purpose and schema

The target declares an expected identity, never an observed result. Required
top-level fields are `apiVersion`, `kind`, `format`, `runtime`, `host`, `mcp`,
`procedureDigest`, `trial`, `redactionPolicy`. No metadata, claim level, timestamps,
assertions, outcomes, endpoint URLs, credentials or evidence path are accepted.

`runtime`, `host` and `mcp` use the corresponding SPEC-0006/0051 report shapes.
All identity rules for a verified declaration apply, including exact internally
consistent surface, resolved non-placeholder versions/models, immutable revision,
calendar-valid protocol dates, allowed transport and reviewed-deployment digest
for non-loopback profiles. These checks do not establish real observations.

`trial` contains only `scenarioDigest`, `promptDigest`, `toolPolicyDigest`,
`limits`. All seven limits MUST be explicitly present integers (not null):
providerRequests, toolCalls, wallTimeMs, maxTraceBytes, retriesPerProviderRequest,
retriesPerToolCall, repeatedIdenticalCalls. The same positive/nonnegative rules
as report admission apply. Zero is an exact value, never unlimited or a wildcard.
No trial outcome/index is part of the target.

`procedureDigest` and the trial digests use the existing business-identity syntax.
`redactionPolicy` MUST be `synthetic-only-v1`. Optional requestedModel and loopback
deploymentProfileDigest may be absent; absence means an empty value that must
match an empty report value, NOT skip comparison. No target is generated from
the report by this command. Synthetic test fixtures are not observed evidence.

## 2. Resource and privacy admission

One strict YAML/JSON document, at most 1 MiB; no unknown/duplicate fields,
anchors, aliases, explicit tags or extra documents. Nested depth uses the existing
strict YAML bound. Required objects, required scalar identities and every limit
must be present. Raw and decoded typed content pass the existing finite secret
pattern checks; parser diagnostics MUST NOT echo decoded input.

After strict decode, inspect the original YAML scalar types: every target limit
MUST have integer tag, not a float, numeric string, bool or null. The same rule
also applies to report limits, trial index and passed/failed assertion counts;
all those report fields MUST now be present explicitly. This prevents YAML's
float-to-Go-int truncation (including failed: 0.5 becoming zero). Accepted YAML
integer spelling follows yaml.v3; all values must fit the existing Go int range.
No decimal/exponent float token is accepted merely because its numeric value
is integral. Direct typed Go values remain integers without lexical inspection.

File loading rejects observed non-regular/symlink targets and oversize content,
rechecks file type after open and reads at most limit+1 bytes. Use a trusted,
quiescent local directory; parent symlinks/concurrent path replacement and OS
blocking are not a hardened sandbox or atomic snapshot. Errors contain finite
codes without source values/paths. No files, digests or manifests are written.

## 3. Acceptance and compatibility

Tests cover six admitted profiles, all missing limit fields including zero-valued
ones, null/unknown/duplicate fields, privacy escapes, size/type refusal, identity
inconsistency, optional emptiness and valid YAML/JSON. Existing valid integer
reports and time-only CLI retain their semantics; omitted or coerced report
numbers are now rejected without rewriting files. Target identity is deliberately narrower
than full HostProfile v1alpha2; unknown adapter/capability fields MUST fail.
