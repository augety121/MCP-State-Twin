# vNext Lifecycle SPEC Pack — Adoption and Traceability Matrix

**Status:** repository decision record (not a claim that the uploaded pack is implemented)
**Decision date:** 2026-08-20
**Input artifact:** `MCP-State-Twin-Lifecycle-SPEC-Pack-vNext.zip`
**Input SHA-256:** `a781d217e19b681c510e12f785de56dea95066cbaf3eff35c3afc9535a819cf1`
**Authority:** this matrix is subordinate to accepted ADRs and the executable
implementation status. It does not promote proposal text to normative runtime
behaviour.

> The uploaded pack is explicitly marked **Proposal / Unverified**. “Complete”
> therefore means that every item has an explicit disposition, owner-facing
> acceptance gate, and evidence path. It does **not** mean that every future
> capability is present in the binary.

## 1. Decision vocabulary

| Disposition | Meaning | What may be claimed publicly |
|---|---|---|
| **Adopted / implemented** | An accepted ADR or RFC requirement has executable code, tests, and recorded evidence. | The bounded behaviour only, with a link to the evidence. |
| **Adopted / partial** | A deliberately bounded subset is accepted; the remainder is not part of the current release profile. | The subset and its limitations. Never the full proposal. |
| **Governance / planning** | The document defines process, taxonomy, or future gates. | That the process exists; not that the gated capability exists. |
| **Proposal / blocked** | The idea is useful, but implementation requires external systems, new evidence, or an unresolved decision. | “Planned” or “not implemented”. |
| **Rejected / superseded** | The item conflicts with an accepted boundary or has been replaced by a later decision. | Do not advertise it as a feature. |

An item can move forward only through: (1) a written decision, (2) an
implementation change, (3) executable tests including negative paths, (4) an
evidence artifact, and (5) an update to
[`IMPLEMENTATION-STATUS.md`](IMPLEMENTATION-STATUS.md). A roadmap entry alone
never changes the disposition.

## 2. Complete pack inventory

The table below covers every numbered document in the uploaded archive. The
proposal pack is vendored in `docs/00-...` through `docs/33-...` for public
review, while the separately named `SPEC-*`, `ADR-*`, and status documents are
the repository's maintained implementation records. Vendoring proposal text
does not accept or implement it.

| Pack file | Disposition in this repository | Current evidence / next gate |
|---|---|---|
| `00-MASTER-LIFECYCLE-SPEC.md` | Governance / planning | Umbrella design only. Use the matrices below and RFC-0001 for current authority. |
| `01-SPEC-GOVERNANCE-CLAIMS-VERSIONING.md` | Governance / planning | Claim vocabulary is enforced by this matrix and IMPLEMENTATION-STATUS. A future release must add a machine-readable claim manifest before calling the process automated. |
| `02-PHASE-0-MCP-2026-REBASELINE.md` | Adopted / implemented (bounded) | Raw modern `server/discover`, direct `tools/list`, result discriminator, version mismatch, and legacy initialize tests are recorded in [PHASE-0-MCP-2026-GAP-MATRIX](PHASE-0-MCP-2026-GAP-MATRIX.md). Official modern conformance and optional profiles remain open. |
| `03-SPEC-0007-VIRTUAL-TIME-ENTROPY-SCHEDULER.md` | Adopted / partial | Forward-only private clock advancement, bounds, CAS head checks, and audit are accepted by [ADR-0011](ADR-0011-HEAD-VERSION-AND-VIRTUAL-CLOCK.md). Scheduler, due-event delivery, seeded entropy, and scheduled effects require a new ADR and deterministic tests. |
| `04-SPEC-0008-DETERMINISTIC-FAULTS.md` | Adopted / partial | ADR-0012 accepts branch-local `before-validation` and `after-commit-before-response` plans with atomic counters/events and bounded private controls. Remaining phases, probability/seed syntax, idempotency, crash/cancellation, latency, and consistency models remain proposals. |
| `05-SPEC-0009-RECORD-REPLAY-REDACTION.md` | Proposal / blocked | Recorder, cassette format, secret redaction, and replay are not implemented. Gate: redaction-before-persist tests, bounded artifact sizes, and a no-secrets fixture scan. |
| `06-SPEC-0010-UPSTREAM-SURFACE-DRIFT.md` | Proposal / blocked | Local surface digest/admission is implemented. Upstream discovery, refresh, and authenticated drift evidence are not. Gate: pinned upstream fixture, digest mismatch tests, and explicit fail-closed behaviour. |
| `07-SPEC-0011-DIFFERENTIAL-FIDELITY.md` | Proposal / blocked | No upstream differential runner or L2 promotion exists. Gate: deterministic paired execution, comparator policy, mismatch taxonomy, and human-reviewed promotion evidence. |
| `08-SPEC-0012-STORAGE-CONCURRENCY-RECOVERY.md` | Adopted / partial | SQLite schema v4, monotonic `head_version`, CAS calls/reset/clock/fault configuration, snapshot source-head binding, and legacy migrations are tested. Crash recovery, tagged migration fixtures, import/export, and multi-writer/remote guarantees are not accepted. |
| `09-SPEC-0013-SECURITY-NETWORK-BOUNDARY.md` | Proposal / blocked | Local control bearer authentication and loopback-only hermetic CI evidence exist. TLS, mTLS/OAuth, remote tenancy, network policy, and production deployment are not implemented. |
| `10-SPEC-0014-EVIDENCE-AUDIT-OBSERVABILITY.md` | Proposal / blocked | Mutation audit and operational redaction are implemented in the current boundary. Full evidence manifests, trace redaction/retention, OpenTelemetry, and signed attestations require a separate acceptance decision. |
| `11-SPEC-0015-RESOURCE-GOVERNANCE.md` | Adopted / partial | ADR-0013 adds a versioned profile, `statetwin limits`, environment digest binding, typed `RESOURCE_LIMIT`, and fail-closed local state/input/output/diff/storage bounds. OS quotas, distributed fairness, scheduler/bundle/cassette quotas, and empirical performance budgets remain open. |
| `12-SPEC-0016-REPRODUCIBLE-EVALUATION-BUNDLE.md` | Proposal / blocked | No portable bundle import/export or signature verification is implemented. Gate: path-safe archive handling, canonical manifest, schema/version checks, digest verification, size limits, and negative tests. |
| `13-HOST-EVALUATION-CODEX-CLAUDE-OTHER-AGENTS.md` | Proposal / blocked | The project has a tools-first MCP surface, not verified provider adapters. Gate: versioned host profile, isolated smoke run, model-visible surface capture, and reproducible evidence for each host. |
| `14-MULTI-AGENT-LONG-RUNNING-FUTURE-AGI.md` | Governance / planning | Research direction only. It cannot change the single-agent data/control-plane boundary or release claims. |
| `15-FAILURE-MODE-EDGE-CASE-CATALOG.md` | Governance / planning | Failure taxonomy informs tests and the existing matrix. A catalog row becomes “covered” only when a regression test exists. |
| `16-RELEASE-LIFECYCLE-AND-GATES.md` | Governance / planning | Release gates are maintained in [ROADMAP](ROADMAP.md), RFC-0002, and CI. No vNext release is implied by this document. |
| `17-SOURCE-REGISTRY.md` | Governance / planning | Source registry is research provenance, not runtime evidence. URLs and claims must be rechecked at the time of a compatibility claim. |
| `18-FEASIBILITY-DEPENDENCY-MATRIX.md` | Governance / planning | Dependency risks are captured here; a row is not authorization to add a dependency or network path. |
| `19-OPEN-QUESTIONS-DECISION-GATES.md` | Governance / planning | Unresolved questions remain explicit. No implementation may silently choose a value that changes semantics or security. |
| `20-UNIVERSAL-AGENT-COMPATIBILITY-ARCHITECTURE.md` | Proposal / blocked | Architecture model only. It does not establish compatibility with any listed host. |
| `21-PORTABLE-MCP-TOOLS-PROFILE.md` | Proposal / blocked | Tools-first is the current interoperability baseline. Resources, prompts, roots, elicitation, Tasks, and MRTR are not claimed. |
| `22-HOST-ADAPTER-SPI-AND-REGISTRY.md` | Proposal / blocked | No host adapter SPI or registry is shipped. Gate: versioned interface, capability negotiation, isolation, and fixture tests. |
| `23-CURRENT-AGENT-HOST-MATRIX-2026-08.md` | Governance / planning | Documentation-only research matrix. Each row remains unverified until evidence is attached; provider behaviour may change. |
| `24-HOST-ISOLATION-BENCHMARK-INTEGRITY.md` | Proposal / blocked | Control-plane separation exists. Hidden-state protection, workspace identity, memory reset, and held-out evaluation tiers need an episode runner and tests. |
| `25-EVALUATION-EPISODES-CURRICULUM-CAPABILITY-UPLIFT.md` | Proposal / blocked | No capability-uplift or curriculum mode is shipped. Gate: blind/diagnostic/coaching separation and contamination-resistant reports. |
| `26-SCENARIO-FAMILIES-METAMORPHIC-COVERAGE.md` | Proposal / blocked | Current scenario runner is explicit and deterministic but has no generator/metamorphic engine. Gate: seed-stable generation and held-out collision tests. |
| `27-CROSS-PROTOCOL-ACP-A2A-BOUNDARY.md` | Governance / planning | MCP remains the external-world/tool boundary. ACP/A2A adapters are not implemented and must not be inferred from MCP support. |
| `28-COMPATIBILITY-CI-EVIDENCE-FRESHNESS.md` | Proposal / blocked | CI runs local protocol and safety checks. Host smoke, evidence freshness, and stale-claim invalidation are not implemented. |
| `29-MULTIMODAL-ARTIFACT-OUTPUT-PROFILE.md` | Proposal / blocked | No multimodal/artifact profile is claimed. Gate: typed artifact schema, size/content policy, digesting, and host projection tests. |
| `30-PORTABLE-SURFACE-PROJECTION-AND-COMPAT-LINT.md` | Proposal / blocked | Canonical surface digest exists; projection/lint does not. Gate: deterministic authorization and host projection fixtures with negative cases. |
| `31-REFERENCE-FRAMEWORK-ARCHITECTURE.md` | Governance / planning | Seven-plane reference architecture. It is a map for future work, not a statement that all planes are present. |
| `32-UNIVERSAL-COMPATIBILITY-REQUIREMENT-CATALOG.md` | Governance / planning | Requirement IDs are proposal identifiers until linked to an accepted ADR, code, tests, and evidence. |
| `33-IMPLEMENTATION-WORKSTREAMS-AND-EXIT-CRITERIA.md` | Governance / planning | Workstream sequencing is planning guidance. Exit criteria are gates, not completed work. |
| `PACK-QA.md` | Governance / planning | Archive QA notes are retained as provenance; they do not replace repository tests. |
| `VNEXT-DELTA.md` | Governance / planning | Compatibility extension summary; all host claims remain unverified. |

### Templates in the archive

The archive also contains eleven templates. They are design aids, not shipped
CLI schemas or accepted wire formats. They remain **proposal artifacts** until
each template has a versioned repository schema, parser/serializer, negative
tests, and a migration policy:

`AGENT-COMPATIBILITY-PROFILE.yaml`, `COMPATIBILITY-CLAIM.yaml`,
`EVALUATION-EPISODE.yaml`, `EVIDENCE-MANIFEST.yaml`, `HOST-PROFILE.yaml`,
`HOST-SURFACE-PROJECTION.yaml`, `MCP-2026-07-28-GAP-MATRIX.md`,
`RELEASE-EVIDENCE-INVENTORY.yaml`, `REQUIREMENT-TRACEABILITY.md`,
`SCENARIO-FAMILY.yaml`, and `TWIN-BUNDLE-MANIFEST.yaml`.

The existing repository documents with similar names (for example the Phase 0
gap matrix and scenario artifacts) are not silently treated as implementations
of these templates; their accepted fields and tests remain authoritative.

## 3. Release boundary after this review

The current branch may truthfully be described as a **local development preview
of a deterministic, hermetic, tools-first MCP State Twin** with:

- bounded TwinSpec validation and declarative state transitions;
- isolated snapshots/forks and canonical state digests;
- stateless MCP 2026 wire evidence plus legacy initialize compatibility;
- a private, forward-only virtual clock with CAS protection and audit; and
- control-plane isolation from the agent-visible MCP tool surface.

It may **not** be described as a universal AGI base, a verified ChatGPT/Claude
integration, a durable scheduler, a fault-injection system, an upstream-fidelity
service, a remote multi-tenant service, or a complete evaluation platform. Those
are valid directions in the pack, but they require the gates above.

## 4. Required evidence for the next promotion

Before accepting any additional vNext capability, the change must include all
of the following in one reviewable commit or pull request:

1. an ADR naming the exact subset and compatibility/security boundary;
2. executable unit, integration, and negative tests;
3. deterministic fixtures with no credentials, production data, or private
   traces;
4. updated implementation status and README claim text;
5. CI commands that reproduce the evidence on the supported operating systems;
6. a failure-mode entry and recovery/rollback note; and
7. an explicit statement of external evidence still missing (if any).

This is the completion criterion for the uploaded pack. It prevents a large
architecture document from becoming an accidental promise of capabilities the
repository cannot yet prove.
