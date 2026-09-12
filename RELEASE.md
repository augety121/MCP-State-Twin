# Release and Maintainer Operations

The canonical release policy is [`docs/RELEASE-MANAGEMENT.md`](docs/RELEASE-MANAGEMENT.md).
This file is the short pre-flight checklist for the current repository.

MCP State Twin uses small, evidence-backed releases. A release must not turn
proposal text into an implementation claim.

## Before a release

1. Confirm the working tree is clean and the target commit is on main.
2. Run the commands in docs/IMPLEMENTATION-STATUS.md.
3. Run go test -race ./... on Linux CI; Windows without cgo is not race evidence.
4. Run Scenario, TwinBundle/Episode, MCP wire, fuzz, secret-policy, and
   hermetic-egress jobs.
5. Review changed public claims in all README language variants.
6. Update CHANGELOG.md with verified behavior and explicit limitations.
7. Update the relevant ADR/SPEC and implementation status together.
   Add a reviewed version-bound plan and notes under [releases/](releases/README.md);
   validate with `go run -p 1 ./cmd/releasecheck --tag <exact-tag>`.
8. Create a GitHub release only from the reviewed commit.
9. For stable v0.1, verify that no provider/product compatibility is implied by
   local MCP evidence. Live provider reports are a v0.3 profile gate, not a
   local-core v0.1 gate.
10. Treat unsigned TwinBundle verification as integrity/semantic evidence only;
    it is not publisher identity or supply-chain provenance.
11. For Journal changes, run a completed Episode twice with the same request,
    inspect after reopen, and verify incomplete/conflict/tamper negative tests.
12. For ADR-0020 changes, run the remote Episode lease, fencing, cancellation,
    ambiguity, duplicate-completion and Journal-v1-to-v2 migration suites.
    Release notes may claim exactly-once terminal Evidence acceptance only,
    never exactly-once provider/tool/external execution.
13. Provider mock tests are not live evidence. A release that claims a verified
    provider profile requires
    dated OpenAI and Anthropic workflow artifacts and admitted compatibility
    reports from the same reviewed revision.

For a tagged release, use a SemVer tag such as `v0.1.0-alpha.1` or `v0.1.1`.
The tag workflow in `.github/workflows/release.yml` admits the checked-in plan,
exact tag and main ancestry, then reuses the full CI at that commit. Only the
final staging job has contents:write. It rechecks the remote tag, builds into a
new `dist`, retains the existing release checksum mechanism and creates a draft
with reviewed notes. Existing output is never deleted or overwritten;
prereleases use the actual prerelease flag, and Latest is not assigned automatically.
A maintainer reviews and publishes the draft. Failed uploads may leave partial
remote draft state; do not assume they had no effect. Do not create stable
`v0.1.0` while an RFC-0002 required gate is open.

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
