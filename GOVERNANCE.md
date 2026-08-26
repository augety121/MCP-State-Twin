# Governance

MCP State Twin is currently maintained by the repository owner and accepts
changes through evidence-backed pull requests.

## Decision authority

- Accepted ADRs control architectural decisions.
- RFC-0002 controls the stable v0.1 release profile.
- RFC-0003 controls only the v0.2 subsets explicitly accepted by ADR-0018 and ADR-0019.
- `docs/IMPLEMENTATION-STATUS.md` controls current implementation claims.
- Roadmap and archived vNext documents do not grant implementation status.

Semantic, protocol, storage, artifact-format or security-boundary changes need
an ADR or RFC before they can be described as supported.

## Merge policy

`main` must remain buildable. Pull requests should include the implementation,
positive and negative tests, compatibility notes and public-claim updates in
one reviewable change. Required CI must pass before merge.

While the project has one active maintainer, that maintainer is the final
decision owner. If another active maintainer joins, storage, protocol, release
and security changes should require a code-owner review from someone other than
the author.

## Becoming a maintainer

A contributor may be added after sustained, technically sound participation in
issues, reviews, releases or security work. Maintainer changes are recorded in
`MAINTAINERS.md` through a pull request.

## Conduct and conflicts

Participants follow `CODE_OF_CONDUCT.md`. Security reports use the private
process in `SECURITY.md`. Project decisions should be based on reproducible
evidence and disclosed technical trade-offs, not adoption or AGI marketing
claims.
