# SPEC-0051: Host Report Consistency

Status: accepted by [ADR-0051](ADR-0051-CONSISTENT-HOST-REPORT-ADMISSION.md).
Amends only the executable report-admission subset of SPEC-0006 / ADR-0017.
This is part of B15 prerequisites, not full HostProfile v1alpha2 implementation.

## 1. Admission invariants

1. An `exact` observed tool surface MUST have the same declared digest as the
   runtime surface, for every claim level. `modified` cannot be `verified`.
2. A `verified` declaration MUST have at least one passed assertion, zero failed
   assertions and a completed outcome. Counts do not prove the assertions ran.
3. `cancellation` and `cancellation-unsupported` MUST NOT coexist. Duplicate
   checks remain invalid. Check names are identifiers of at most 128 bytes;
   unknown additional identifiers remain permitted but satisfy no required check.
4. Configured/negotiated protocol dates MUST be calendar-valid `YYYY-MM-DD`.
   They MAY differ; this validator does not establish supported negotiation.
5. Runtime version and host name/version/provider/model are nonempty, at most
   256 UTF-8 bytes, without control characters or leading/trailing whitespace.
   Optional requested-model identity follows the same rule when present.
   `verified` runtime/host versions and provider-model identities reject the
   case-insensitive placeholders `latest`, `auto`, `unknown`, `none`.
   Generic MCP still requires provider/model `none`. This does not recognize
   every mutable provider alias or attest to an observed version.
   Verified custom reports naming a provider other than `none` also reject
   placeholder provider/model identities; custom profile is not a bypass.
6. Any declared expiry MUST be later than creation, including non-verified
   reports. Verified reports still require an expiry. Admission is independent
   of today's clock so historical structurally consistent artifacts remain readable.

## 2. Decode and privacy boundary

Retain the one-document, known-field, no-anchor/alias/tag YAML contract. Reject
input above 1 MiB before scanning. After decoding, serialize the typed report
in memory and apply the existing sensitive-pattern scanner to decoded values;
the serialized form is also limited to 1 MiB. No serialization is written.
This catches credential-like quoted YAML escapes and applies to direct Validate
callers too. Error messages MUST NOT echo sensitive decoded values. It is not
general-purpose DLP, a raw transcript parser or hostile-filesystem isolation.

## 3. Compatibility and verification

Report format remains v1alpha1; new admission is intentionally stricter.
Do not edit historical failed artifacts into passing ones. `compatibility
validate` retains its fields and adds `validationScope: structure-only`,
`provenance: not_verified`, `publicationAllowed: false`. Old strict output readers
may need updating. Existing report business digest generation is unchanged.

Executable acceptance covers each contradiction above, encoded synthetic-secret
refusal, control/size boundaries, all six profile identities, modified experimental
reports, calendar leap days and preservation of positive legacy fixtures.
Full vet/test, platform CI and race checks are required for the candidate.
