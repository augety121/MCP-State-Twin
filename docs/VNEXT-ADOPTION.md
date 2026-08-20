# vNext SPEC Pack Adoption Record

The uploaded Lifecycle SPEC Pack is a proposal bundle, not an instruction to
mark every future capability as accepted or implemented. This repository adopts
the following bounded slice in this change:

1. governance and claim vocabulary remain evidence-first;
2. MCP 2026-07-28 tools-first wire evidence is recorded in
   `PHASE-0-MCP-2026-GAP-MATRIX.md`;
3. the private virtual-clock advancement subset is implemented;
4. schema-v3 monotonic branch heads and snapshot source-head identity are
   implemented;
5. scheduler, entropy, deterministic faults, recorder/replay, differential
   fidelity, remote security, bundles, and host adapters remain proposals with
   explicit gates.

The authoritative implementation status is
[`IMPLEMENTATION-STATUS.md`](IMPLEMENTATION-STATUS.md). A proposal becomes an
accepted implementation only after an ADR, executable tests, evidence, and a
status update land together. “Compatible with ChatGPT/Claude/all agents” is
never inferred from protocol support alone.

The complete file-by-file disposition of the uploaded archive is maintained in
[`VNEXT-TRACEABILITY.md`](VNEXT-TRACEABILITY.md). That matrix is the review
index for the pack: it records every proposal item, its current boundary, and
the evidence required before promotion.
