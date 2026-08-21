# Release and Maintainer Operations

The canonical release policy is [`docs/RELEASE-MANAGEMENT.md`](docs/RELEASE-MANAGEMENT.md).
This file is the short pre-flight checklist for the current repository.

MCP State Twin uses small, evidence-backed releases. A release must not turn
proposal text into an implementation claim.

## Before a release

1. Confirm the working tree is clean and the target commit is on main.
2. Run the commands in docs/IMPLEMENTATION-STATUS.md.
3. Run go test -race ./... on Linux CI; Windows without cgo is not race evidence.
4. Run scenario, MCP wire, fuzz, secret-policy, and hermetic-egress jobs.
5. Review changed public claims in all README language variants.
6. Update CHANGELOG.md with verified behavior and explicit limitations.
7. Update the relevant ADR/SPEC and implementation status together.
8. Create a GitHub release only from the reviewed commit.

For a tagged release, use a SemVer tag such as `v0.1.0-alpha.1` or `v0.1.1`.
The tag workflow in `.github/workflows/release.yml` reruns the release gates,
builds Linux/macOS/Windows binaries, writes `SHA256SUMS`, and creates a draft
GitHub release only if every step succeeds. A maintainer reviews and publishes
the draft. Do not create the stable `v0.1.0` tag while an RFC-0002 required gate
is open.

## Release notes must contain

- commit/tag and verification date;
- supported Go and MCP SDK versions;
- commands and CI run links used as evidence;
- changed protocol, storage, or report compatibility;
- migration or rollback notes;
- known limitations and unimplemented proposal items; and
- security notes and fixture provenance.

Do not include fabricated adoption metrics, provider compatibility, performance
numbers, or “AGI” capability claims in release notes.

## Maintainer cadence

For each maintenance cycle, record at least one of:

- triaged issue with a reproducible fixture;
- reviewed pull request with tests and boundary analysis;
- dependency or CI update;
- release or changelog update; or
- security or hermeticity regression check.

This is a workflow checklist, not a claim that every item has already occurred.
