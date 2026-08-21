# Documentation Governance

This repository contains implementation records, accepted design decisions,
and a large forward-looking SPEC pack. They are intentionally different kinds
of documents.

## Authority order

When documents disagree, use this order:

1. `AGENTS.md` for repository process and safety rules;
2. accepted ADRs for binding architectural decisions;
3. RFC-0002 for the current v0.1 release profile and gates;
4. `SPEC-0001` through accepted SPEC documents for normative semantics;
5. `IMPLEMENTATION-STATUS.md` for what executable evidence exists today;
6. `ROADMAP.md` and the vNext pack for planned work only.

The implementation status is the authority for public claims about the current
binary. A proposal cannot override a test, an ADR, or a documented limitation.

## Document statuses

| Status | Meaning | Allowed wording |
|---|---|---|
| Accepted | Binding design decision | “MUST”/“current contract” |
| Implemented | Code and tests exist for the stated subset | “implemented and tested” |
| Partial | A bounded subset is accepted; remainder is open | Name the subset and limits |
| Proposal | Design exploration or future requirement | “proposed”/“planned” only |
| Blocked | Requires missing dependency or evidence | Explain the gate |
| Retired | Replaced by a later decision | Link the successor |

## Required change sequence

For a semantic or security change:

```text
issue / design question
  -> ADR or RFC decision
  -> implementation
  -> positive + negative tests
  -> implementation-status update
  -> README / changelog claim review
  -> release evidence
```

Changing only README prose never changes the product contract.

## Naming and ownership

- `ADR-NNNN-*.md`: one accepted architectural decision; immutable after
  acceptance except for a clearly marked amendment.
- `SPEC-NNNN-*.md`: normative semantic contract; each MUST/SHOULD needs an
  evidence path.
- `RFC-NNNN-*.md`: release or cross-cutting proposal; accepted RFCs define a
  release boundary, not automatic implementation.
- `IMPLEMENTATION-STATUS.md`: current evidence ledger; update in the same PR as
  a newly accepted capability.
- `docs/00-...` through `docs/33-...`: vendored vNext proposal material; not
  implementation authority.

## Claim review checklist

Before merging a public claim, verify:

- the claim names an exact subset and version;
- a test, CI run, or reproducible command supports it;
- unsupported hosts/providers are not implied by protocol support;
- performance, adoption and security numbers have a source and date;
- roadmap items are labeled planned or blocked; and
- the claim is consistent across all README language variants.

## Evidence retention

Release evidence belongs in the release notes and the maintainer ledger. Do not
commit credentials, private traces, production data, or copied provider logs.
Use synthetic fixtures and links to public CI/issue/PR records instead.
