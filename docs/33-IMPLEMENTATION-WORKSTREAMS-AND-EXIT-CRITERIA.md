# Implementation Workstreams & Exit Criteria

> **Status:** Proposal planning artifact  
> **Purpose:** Convert the architecture into parallelizable engineering work without violating dependency order.

## WS0 — Truth Inventory

Scope:
- current code/docs/tests;
- current SPEC/RFC/ADR conflicts;
- current dependency versions;
- current CI evidence.

Exit:
- no current capability is classified from memory;
- Implementation Status reflects actual repository.

## WS1 — MCP Protocol Rebaseline

Scope:
- 2026-07-28;
- modern vs legacy decision;
- SDK pin;
- conformance pin;
- wire tests.

Depends on: WS0.

Exit:
- protocol claim mechanically backed by evidence.

## WS2 — Deterministic World Core

Scope:
- virtual time;
- scheduler;
- entropy;
- event ordering;
- deterministic faults.

Depends on: WS0 + semantic governance.

Exit:
- reproducibility corpus passes.

## WS3 — Storage / Recovery / Limits

Scope:
- branch head concurrency;
- crash killpoints;
- migrations;
- WAL operations;
- resource governance.

Can run in parallel with WS2 after interfaces are agreed.

Exit:
- conflict/recovery/limit evidence.

## WS4 — Evidence & Bundle

Scope:
- evidence schema;
- environment identity;
- raw/server/host/evaluation layers;
- bundle manifest;
- redaction.

Start design early; stabilize after WS2/WS3 semantics.

Exit:
- one deterministic scripted episode produces self-contained evidence.

## WS5 — Portable Agent Surface

Scope:
- Portable MCP Tools Profile;
- surface layers;
- projection model;
- compatibility linter;
- target host profiles.

Depends on: stable canonical tool surface.

Exit:
- current reference twin statically linted against initial target set with explicit PASS/WARN/UNKNOWN.

## WS6 — Host Adapter Framework

Scope:
- Adapter SPI;
- registry;
- fixtures;
- isolation;
- surface readiness;
- workspace identity.

Depends on: WS5 + WS4.

Exit:
- at least two different host adapters can run the same scripted fixture/profile harness without world-core changes.

## WS7 — Episode Orchestrator

Scope:
- episode manifest;
- world fork;
- host prepare;
- budget;
- scoring;
- cleanup.

Depends on: WS4 + WS6.

Exit:
- one local host episode is reproducible at environment/evidence level.

## WS8 — Current Agent Matrix

Initial targets:
- Codex;
- Claude Code;
- Gemini CLI;
- GitHub Copilot;
- Cursor;
- Windsurf;
- Cline;
- Amazon Q;
- JetBrains/Junie;
- Zed;
- OpenCode.

Each host gets a separate small PR/profile.

Exit per host:
- primary-source profile;
- adapter fixture;
- connection smoke;
- stateful scenario;
- limitations;
- evidence.

## WS9 — Remote Provider Profiles

Targets:
- OpenAI remote MCP API;
- Anthropic Messages connector;
- Managed Agents if justified.

Depends on: remote security-qualified staging.

Exit:
- no production data;
- remote auth profile;
- stateful evidence.

## WS10 — Fidelity

Scope:
- recorder;
- redaction;
- upstream surface inspector;
- differential harness;
- L0/L2.

Exit:
- bounded evidence-qualified fidelity profile.

Can overlap WS6–WS9, but fidelity claims require stable evidence.

## WS11 — Curriculum / Capability Uplift

Scope:
- scenario families;
- metamorphic tests;
- held-out seeds;
- coaching;
- stress.

Depends on: deterministic runtime + episodes.

Exit:
- blind vs coached runs separated;
- generalization to new family instances demonstrated without claiming base-model training.

## WS12 — Multi-Agent / Cross-Protocol

Scope:
- isolated multi-agent;
- shared branch;
- ACP path characterization;
- optional A2A delegation experiments.

Depends on: concurrency/evidence/host adapters.

Exit:
- explicit interleaving evidence;
- no control secret propagation.

## WS13 — Stable Release

Scope:
- migration/deprecation;
- platform support;
- SBOM/provenance;
- claim registry;
- release docs.

Depends on prior stable contracts.

Exit:
- v1.0 gates.

## Parallelization map

```text
WS0
 |
 +--> WS1
 |
 +--> WS2 -----+
 |             |
 +--> WS3 -----+--> WS4 --> WS5 --> WS6 --> WS7 --> WS8
                                      |              |
                                      |              +--> WS9
                                      |
                                      +--> WS10
                                              |
                                WS7 ----------+--> WS11
                                WS3 + WS6 --------> WS12
                                                  |
                                                  v
                                                WS13
```

## PR-size rule

Do not land an entire workstream in one PR.

Preferred unit:

```text
one accepted requirement cluster
+ implementation
+ tests
+ evidence fixture
+ traceability update
```

## Exit criterion rule

A workstream is not complete because its code exists.

It completes only when:
- its required evidence exists;
- known limitations are explicit;
- public claims are updated;
- unresolved blockers are classified.
