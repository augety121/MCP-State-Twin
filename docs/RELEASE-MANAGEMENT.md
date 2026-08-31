# Release Management

This document defines the maintainer operating model. It is designed for a
small but serious infrastructure project: every release is a reproducible
claim about a reviewed commit, not a marketing snapshot.

## Release channels

| Channel | Tag pattern | Meaning | Compatibility promise |
|---|---|---|---|
| Development | no tag / `main` | Unreleased work | APIs and storage may change |
| Alpha | `v0.1.0-alpha.N` | Integrator preview with known open gates | Only documented preview subset |
| Stable | `v0.1.0` | RFC-0002 gates are closed | v0.1 profile only |
| Patch | `v0.1.1` | Backward-compatible bug/security fix | No semantic expansion |
| Minor | `v0.2.0` | Backward-compatible feature/profile addition | New capability is explicitly scoped |

The plain `v0.1.0` tag must not be created while a required RFC-0002 gate is
open. A release can be delayed without treating the delay as a failure.

## Branch and merge policy

- `main` is the default integration branch and must remain buildable.
- Feature work lands through pull requests; direct pushes are reserved for
  emergency maintainer recovery and must be documented afterward.
- A PR that changes semantics must include the ADR/SPEC/status updates in the
  same reviewable change.
- Every merge should leave a reproducible test command or CI run.

## Required release evidence

Before tagging:

1. `git status --short` is empty and the commit is on `main`.
2. `go test ./...` and `go vet ./...` pass locally.
3. Linux CI passes `go test -race ./...`.
4. Scenario, TwinBundle/Episode, MCP wire, limits, fuzz, secret-policy and hermetic-egress jobs
   pass, or the release notes record a precise exception.
5. README language variants, `CHANGELOG.md`, RFC-0002 and
   `IMPLEMENTATION-STATUS.md` agree.
6. The release profile, Go version, MCP SDK version, schema/storage version and
   migration notes are recorded.
7. Fixtures are synthetic and repository scans show no credentials or private
   traces.
8. Every tagged storage schema has a provenance-labelled reopen/migration
   fixture, and interrupted-migration recovery passes on the candidate commit.
9. For stable v0.1, provider/product compatibility remains explicitly outside
   the release profile. A v0.3 release that claims a verified provider profile
   requires an admitted report, reviewed remote-staging profile, and proof that
   no raw provider request ID, transcript, or credential is committed.
10. For a release containing TwinBundle/Episode preview changes, build and
    verify both reference bundles, run one Episode per reference domain, and
    record that artifacts are unsigned and local-only.
11. For a release containing Journal changes, verify reopen, identical-request
    terminal replay, identity conflict, incomplete refusal, foreign/future
    schema refusal, and Evidence tamper detection.
12. For a release containing remote Episode changes, verify one-owner claiming,
    lease/heartbeat bounds, fencing, stale-worker refusal, hermetic expiry
    recovery, attempt-budget exhaustion, queued/leased cancellation, external
    `COMMIT_UNKNOWN`, duplicate completion, coordinator authentication and
    non-loopback TLS admission.
13. A verified provider-profile gate requires dated artifacts from the manual
    `provider-smoke` workflow for both OpenAI and Anthropic plus the admitted
    HostCompatibilityReports required by SPEC-0019. Mock provider tests
    establish API contract shape only and cannot satisfy this gate.

The release notes must use the phrase **exactly-once terminal Evidence
acceptance** for ADR-0020. They must not shorten it to “exactly-once execution”.
External `COMMIT_UNKNOWN` is an unresolved outcome, never success or a retry
signal.

## Tag and publish procedure

The reviewed maintainer sequence is:

```text
merge green PR
  -> update CHANGELOG and release evidence
  -> tag vX.Y.Z from the reviewed main commit
  -> GitHub release workflow runs tests and builds platform binaries
  -> checksums are attached
  -> maintainer reviews generated notes and publishes
  -> announce only verified scope and known limitations
```

The repository workflow is intentionally fail-closed: a malformed tag, failed
test, failed build, or missing artifact stops publication.

## Release notes format

Each release note should contain:

```text
## Scope
## Verified changes
## Compatibility and migration
## Security and hermeticity
## Known limitations / deferred proposals
## Evidence
## Contributors
```

Do not add fabricated stars, download counts, latency numbers, provider
compatibility, or AGI capability claims. Link public CI, PR, issue and release
records instead.

## Maintainer operations between releases

At least one maintenance record should be visible per cycle:

- triage an issue to a reproducible fixture;
- review a PR against invariants and tests;
- merge a dependency or CI update;
- publish a release or changelog correction; or
- run a security/hermeticity regression check.

Use `docs/MAINTAINER-EVIDENCE.md` to collect real URLs and snapshot dates. It is
a template, not evidence by itself.
