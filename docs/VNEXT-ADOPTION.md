# vNext SPEC Pack Adoption Record

The uploaded Lifecycle SPEC Pack is a proposal bundle, not an instruction to
mark every future capability as accepted or implemented. This repository adopts
the following bounded slice in this change:

1. governance and claim vocabulary remain evidence-first;
2. MCP 2026-07-28 tools-first wire evidence is recorded in
   `PHASE-0-MCP-2026-GAP-MATRIX.md`;
3. the private virtual-clock advancement subset is implemented;
4. schema-v4 monotonic branch heads, snapshot source-head identity, and
   branch-local fault tables are implemented;
5. the bounded deterministic-fault preview in ADR-0012 and local resource
   governance profile in ADR-0013 are implemented; all
   other fault phases, scheduler, entropy, recorder/replay, differential
   fidelity, remote security, and host adapters remain proposals with explicit
   gates.
6. ADR-0014 accepts the synthetic package-registry reference domain as the
   second stateful domain required by the v0.1 release profile; it does not
   claim compatibility with any real package registry.
7. ADR-0015 accepts RFC-0002's L1-only v0.1 profile and explicitly defers
   recorder/L0 and L2/L3 fidelity work; exclusion is not an implementation
   claim.
8. ADR-0016 accepts the single-process local SQLite compatibility profile with
   tagged fixtures and interrupted-migration recovery evidence.
9. ADR-0017 accepts strict HostCompatibilityReport admission without claiming
   that a live OpenAI or Anthropic run has occurred.
10. ADR-0018 accepts deterministic unsigned TwinBundle admission and one-shot
    local scripted EvaluationEpisode evidence. Remote execution, HostProfile
    adapters, signatures, registries, and provider claims remain proposals.
11. ADR-0019 accepts an independent durable local Episode Journal with
    immutable request identity, lifecycle CAS, idempotent completed-evidence
    reads and explicit incomplete records. Automatic recovery, cancellation,
    leases, remote workers and exactly-once execution remain proposals.

The authoritative implementation status is
[`IMPLEMENTATION-STATUS.md`](IMPLEMENTATION-STATUS.md). A proposal becomes an
accepted implementation only after an ADR, executable tests, evidence, and a
status update land together. “Compatible with ChatGPT/Claude/all agents” is
never inferred from protocol support alone.

The complete file-by-file disposition of the uploaded archive is maintained in
[`VNEXT-TRACEABILITY.md`](VNEXT-TRACEABILITY.md). That matrix is the review
index for the pack: it records every proposal item, its current boundary, and
the evidence required before promotion.
