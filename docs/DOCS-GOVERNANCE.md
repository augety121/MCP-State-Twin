# Documentation Governance

This repository contains implementation records, accepted design decisions,
and a large forward-looking SPEC pack. They are intentionally different kinds
of documents.

## Authority order

When documents disagree, use this order:

1. `AGENTS.md` for repository process and safety rules;
2. accepted ADRs for binding architectural decisions, including ADR-0021 for
   lifecycle and release boundaries;
3. accepted RFCs: RFC-0001 revision 3 for the product contract, RFC-0002 for
   v0.1, and RFC-0003 only for subsets accepted by ADR-0018 through ADR-0020
   and ADR-0026 through ADR-0028;
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

## Independent version dimensions

The following versions MUST be recorded independently when they apply:

- product release;
- TwinSpec API;
- MCP protocol profile;
- world-store schema;
- Episode-Journal schema;
- Evidence format;
- TwinBundle format;
- ResourceProfile;
- HostProfile; and
- provider/host adapter.

Changing one dimension does not silently change or verify another.

## Normative indexes

- `REQUIREMENT-TRACEABILITY.md` maps accepted requirements to evidence.
- `CLAIM-REGISTRY.md` controls public implementation claims.
- `COMPATIBILITY-MATRIX.md` records exact profiles without transitive claims.
- `DECISION-REGISTER.md` separates resolved decisions from open roadmap work.
- `PHASE-00` through `PHASE-07` define entry, scope, exclusions and exit
  evidence for each lifecycle phase.

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
