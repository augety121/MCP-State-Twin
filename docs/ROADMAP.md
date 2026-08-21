# Engineering Roadmap

**Status:** implementation in progress; see `IMPLEMENTATION-STATUS.md` for evidence.
**Rule:** 先完成最小可信闭环，再扩功能。每一阶段都必须有可运行 demo、自动测试和退出标准。

## Phase 0 — Design freeze

Progress: protocol, control-plane isolation, expression engine, storage,
canonicalization, and operational logging decisions have ADRs. RFC-0001 remains
the umbrella Draft; SPEC-0001 through SPEC-0006 define the proposed normative
layers and still require maintainer review before a tagged v0.1. ADR-0011
accepts only bounded preview slices of SPEC-0007 and SPEC-0012; the remaining
vNext pack is not accepted or implemented.

Deliverables:

- RFC-0001 umbrella reviewed and reconciled with RFC-0002.
- SPEC-0001 through SPEC-0006 prepared as the proposed v0.1 normative set;
  maintainer acceptance remains a governance gate.
- Failure matrix P0/P1 reviewed.
- ADRs for expression engine, storage snapshot strategy, MCP protocol support, and artifact format.
- Two reference domains selected.
- Naming/license check.

Exit criteria:

- 团队能用一句话解释产品，不出现 “agent framework / universal simulator / perfect twin” 等 scope creep。
- 每个 hard invariant 有对应测试设计。

## Phase 1 — Deterministic kernel

Progress: implemented in the development preview. A 1,000-call two-branch
deterministic replay test currently passes. Bounded TwinSpec and CEL fuzz
targets pass in Linux CI run #6; crash kill-point coverage remains incomplete.

Build:

- Go module and CLI skeleton.
- TwinSpec v1alpha1 parser and schema validator.
- Canonical JSON + digest.
- VirtualClock.
- Deterministic ID/random provider.
- SQLite state backend.
- Bounded expression evaluator.
- Atomic transition engine.
- State assertions and canonical diff.

Phase 1 remains network-independent; the current development preview also
contains the Phase 3 MCP server described below.

Exit criteria:

- 1,000 repeated executions of the same transition corpus produce identical hashes.
- crash/failure tests prove no half-commit in normal atomic mode.
- fuzz parser/evaluator finds no panic for bounded test budget.

## Phase 2 — Snapshot/fork world model

Progress: logical snapshots, isolated forks, reset, and canonical diff are
implemented. The 100-way concurrent isolation gate passes. A bounded Scenario
v1alpha1 format and scripted evidence report are implemented. Export/import and
retention/GC remain incomplete.

Build:

- immutable snapshots.
- isolated branches.
- fork/reset/export/import.
- deterministic branch state digest.
- retention/GC.
- scenario format.

Exit criteria:

- parent snapshot immutability property test.
- 100 parallel scenario branches cannot observe each other's state.
- canonical export/import round-trip preserves digest.

## Phase 3 — MCP data plane

Progress: the official Go SDK serves the TwinSpec tool surface over stateless
Streamable HTTP, and an official-SDK client integration test passes. Official
conformance `v0.1.16` initialize, ping, tools-list, and JSON Schema 2020-12
scenarios pass on Linux CI. Broader negotiated-version coverage remains open.

Build:

- official Go MCP SDK integration.
- `tools/list` copied from TwinSpec surface.
- `tools/call` -> transition engine.
- Streamable HTTP current protocol path.
- backwards-compatible transport only where official SDK makes it safe.
- cancellation and error mapping.

Exit criteria:

- official relevant MCP conformance tests pass.
- local generic MCP client smoke test.
- tool descriptions/schemas exactly match canonical captured surface unless an explicit override exists.

## Phase 4 — Private control plane

Progress: a separate bearer-authenticated HTTP plane implements state,
snapshot, fork, reset, and diff, and tests verify that control functions are
absent from MCP `tools/list`. Snapshot/fork/reset control audit is committed in
the same transaction as each mutation.

Build:

- separate local/private API.
- snapshots/forks/reset/clock/fault APIs.
- auth boundary for remote control plane.
- audit log.
- ensure no control function appears in agent data plane.

Exit criteria:

- security test attempts to discover control tools through MCP and fails.
- control-plane mutation audit completeness test.

## Phase 5 — Recorder and surface drift

Progress: the canonical model-facing tool-surface envelope, digest, and
fail-closed startup binding are implemented. The upstream inspector, recorder,
redaction pipeline, and automatic refresh remain unimplemented.

Build:

- upstream inspector.
- canonical surface fingerprint.
- safe recorder for explicitly selected fixture interactions.
- header/body redaction.
- drift state: CURRENT / DRIFTED / UNKNOWN / UNBOUND.

Exit criteria:

- injected secrets never persist in golden trace tests.
- description-only tool change triggers surface drift.
- L2 twin in DRIFTED state fails CI by default.

## Phase 6 — Reference twins

The repository now contains two independent synthetic reference domains:

- **GitHub-like issue/repository workflow** for maintainer and coding-agent
  scenarios;
- **package-registry workflow** for publish, yank, install and advisory-aware
  dependency scenarios.

These are synthetic state models. They do not claim fidelity to GitHub,
registries, package managers or any upstream production API.

Minimum modeled workflow:

- list/get repository.
- list/search/create/update/close issue.
- comments.
- labels.
- synthetic permissions.
- pagination.
- rate-limit scenario.
- timeout-before/after-effect scenarios.

Why this domain:

- OSS maintainers immediately understand it.
- state transitions are concrete and testable.
- useful for Codex/Claude coding-agent demos.
- avoids financial/medical correctness risk for the first reference twin.

Exit criteria:

- 20+ multi-step scenarios.
- state-based scoring.
- differential test against a disposable fixture implementation/account for declared observable fields.
- explicit fidelity report.

## Phase 7 — Cross-provider smoke matrix

Run the **same twin endpoint and same initial snapshot** with:

- ChatGPT Developer Mode.
- OpenAI API/Codex-compatible harness where appropriate.
- Claude MCP connector.
- Claude Code/local MCP path.
- one generic MCP client.

Important: success criterion is protocol/tool usability, not identical trajectories.

Exit criteria:

- each supported host can list and call the twin's tools.
- all runs produce environment digest and terminal state diff.
- host-specific limitations documented.

## Phase 8 — Trace-assisted TwinSpec bootstrap

Only after the deterministic core is trusted.

Build:

- trace normalizer.
- candidate entity/key extraction.
- candidate read/write dependency extraction.
- candidate output templates.
- optional LLM assist.
- provenance labels: observed / inferred / declared / verified.

Exit criteria:

- generated spec always starts unverified.
- malicious/prompt-injected trace cannot become compiler instruction.
- benchmark reports precision/recall of extracted relations on reference twins; no marketing claim before measurement.

## Phase 9 — Fidelity and differential validation

Build:

- observable projection definitions.
- real-vs-twin differential runner.
- transition coverage report.
- unmodeled state/effect report.
- L0/L1/L2 promotion checks.

Exit criteria:

- L2 promotion is machine-checkable + human-reviewable.
- fidelity badge contains date and upstream surface digest.

## Phase 10 — Fault model

Build deterministic faults:

Current evidence: ADR-0012 implements and tests `before-validation` rate-limit/
timeout and `after-commit-before-response` timeout plans with branch-local
persistence. The remainder of this phase is open.

- rate limit.
- timeout before effect.
- timeout after effect.
- partial effect.
- stale visibility/eventual consistency.
- transient 5xx.
- auth denied.
- output truncation/corruption as explicitly configured.

Exit criteria:

- same seed and call sequence replays the exact fired faults.
- run report distinguishes configured vs fired faults.

## v0.1 release gate

v0.1 should be released only when:

- deterministic kernel passes all golden replay tests.
- snapshot/fork is stable.
- MCP conformance passes for supported subset.
- two model-provider families successfully use the same twin.
- one real reference twin has useful scenarios.
- P0 failure modes have tests or are architecturally impossible.
- docs state limitations prominently.
- CI can run with network egress denied.

## After v0.1 — only based on user evidence

Potential expansions:

- second domain (commerce/order or ticketing).
- OpenAPI importer.
- multi-agent deterministic scheduler.
- MCP Tasks/async operation simulation.
- native high-fidelity adapters.
- RL environment API.
- cloud-hosted remote twins.
- registry and signed TwinSpec bundles.

Do not implement these because they sound impressive. Require issues, users, benchmarks or integration pull requests that prove demand.

## OSS adoption plan

The fastest path to ecosystem relevance is not a giant AGI claim. It is:

1. make the GitHub-like reference twin genuinely useful for coding-agent CI;
2. ship reproducible demos for OpenAI and Claude against the same snapshot;
3. invite maintainers of MCP servers to contribute TwinSpecs/scenarios;
4. publish a simple fidelity report format;
5. integrate with existing eval frameworks rather than competing with them;
6. collect real issues/PRs/releases before applying to OSS support programs.

## Proposed first public demo

**“Same issue tracker. Same initial state. Three agents. Zero real writes.”**

- Start snapshot contains a repository with one bug report and fixture code metadata.
- Model A, B, C receive the same task.
- Each gets its own fork.
- They may choose different tool trajectories.
- Final state assertions verify whether the issue was correctly updated/closed and whether forbidden side effects occurred.
- Report shows terminal state diff, tool-call count, fault handling and environment digest.

This demonstrates the project in one minute without claiming AGI.
