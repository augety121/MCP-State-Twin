# Reviewed release declarations

This directory currently contains instructions only. It does **not** approve a
new tag, stable release, provider profile or draft publication.

Before requesting a real release, a maintainer adds and reviews both files in
the same candidate commit:

1. `releases/<exact-tag>.json`, following
   [SPEC-0045](../docs/SPEC-0045-RELEASE-PLAN-ADMISSION.md);
2. `releases/<exact-tag>.md`, with the seven ordered, nonempty sections below.

Start with an **unapproved** plan. Replace `REVIEW_REQUIRED` with the chosen tag
only after deciding the release scope. Do not silently set review flags to true.

```json
{
  "format": "statetwin.dev/release-plan/v1alpha1",
  "tag": "REVIEW_REQUIRED",
  "profile": "local-core-v0.1",
  "channel": "prerelease",
  "claimsReviewed": false,
  "compatibilityReviewed": false,
  "stableGatesReviewed": false
}
```

For a reviewed prerelease, claims/compatibility declarations must be true and
stableGatesReviewed must remain false. A stable tag requires channel `stable`
and all three explicit review declarations, plus all actual RFC-0002 gates.
This validator currently refuses other release trains and `+build` metadata.
The JSON file is a repository declaration, not a signed reviewer attestation.

Notes require these exact second-level headings, in order, with real evidence
and meaningful nonempty content under each (not template markers):

```text
## Scope
## Verified changes
## Compatibility and migration
## Security and hermeticity
## Known limitations / deferred proposals
## Evidence
## Contributors
```

State the local-core scope and all bundled experimental exclusions explicitly.
Use actual CI/PR links; unknown or unavailable evidence stays unknown. Do not
fabricate provider compatibility, contributor identity, adoption or benchmarks.

Read-only preflight, after files exist (`<exact-tag>` is a placeholder):

```text
go run -p 1 ./cmd/releasecheck --root . --tag <exact-tag>
```

Preflight does not check remote tag existence, execute CI, build artifacts or
publish. After a separately authorized tag push, the workflow validates main
ancestry, runs the full same-commit CI, then creates a **draft** with reviewed
notes. Prereleases are explicitly marked; nothing is automatically marked Latest.
Human publication is still required.

Build refuses any existing `dist`, including partial previous output. Do not
delete or reuse it automatically. A failed upload may leave a remote draft or
some assets; inspect before any operator-authorized reconciliation. See
[release management](../docs/RELEASE-MANAGEMENT.md),
[CI gate contract](../docs/SPEC-0046-CANDIDATE-CI-AND-DRAFT-GATES.md) and
[build contract](../docs/SPEC-0047-SAFE-RELEASE-ARTIFACT-BUILD.md).
