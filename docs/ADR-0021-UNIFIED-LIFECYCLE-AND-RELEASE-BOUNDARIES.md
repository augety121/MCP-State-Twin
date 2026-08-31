# ADR-0021: Unified Lifecycle and Release Boundaries

- **Status:** Accepted
- **Date:** 2026-08-31
- **Amends:** RFC-0001, RFC-0002, RFC-0003, DOCS-GOVERNANCE
- **Scope:** documentation authority, claim states and product release phases

## Context

The repository accumulated an umbrella RFC, accepted release RFCs, accepted
SPECs, ADR-bounded preview work and a large `00-*` through `33-*` lifecycle
proposal pack. Several documents described the same future capability at
different maturity levels. RFC-0002 also required live provider evidence for a
local-only v0.1 profile, even though safe provider validation depends on a
remote security profile that v0.1 explicitly does not provide.

Without a single authority and phase model, a proposal could be mistaken for
implementation and a mock API test could be mistaken for product
compatibility.

## Decision

1. The product remains MCP State Twin. Its bounded definition is a
   deterministic, forkable, evidence-backed stateful environment for testing
   tool-using agents.
2. The authority order is `AGENTS.md`, accepted ADR, accepted RFC, accepted
   SPEC, implementation status, then roadmap/proposal material.
3. The lifecycle proposal pack (`00-*` through `33-*`) is non-normative unless
   an accepted ADR incorporates a requirement.
4. Product, TwinSpec, protocol, storage, Journal, Evidence, Bundle,
   ResourceProfile, HostProfile and adapter versions are independent.
5. Public claims use the states `unsupported`, `unverified`, `experimental`,
   `verified`, `regressed` or `stale`.
6. v0.1 is the local hermetic deterministic-core release. Provider live smoke
   is not a v0.1 gate.
7. Remote Episode execution targets v0.2. Secure provider/host validation
   targets v0.3. Fidelity/recorder work targets v0.4. Stable public formats and
   compatibility policy target v1.0.
8. A provider API report never implies ChatGPT, Codex, Claude, Claude Code or
   another product profile.
9. Remote Episode execution may claim exactly-once terminal Evidence
   acceptance only. It may not claim exactly-once inference, delivery, tool
   execution or external side effects.

## Consequences

- RFC-0002 no longer has a circular provider gate.
- SPEC-0019 through SPEC-0021 define host evidence, remote security and public
  claim admission.
- Phase documents define entry conditions and executable exit evidence.
- A release remains blocked by failures on its exact candidate commit even
  when older CI evidence exists.
