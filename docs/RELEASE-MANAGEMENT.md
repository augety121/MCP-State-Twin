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
4. Scenario, MCP wire, limits, fuzz, secret-policy and hermetic-egress jobs
   pass, or the release notes record a precise exception.
5. README language variants, `CHANGELOG.md`, RFC-0002 and
   `IMPLEMENTATION-STATUS.md` agree.
6. The release profile, Go version, MCP SDK version, schema/storage version and
   migration notes are recorded.
7. Fixtures are synthetic and repository scans show no credentials or private
   traces.

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
