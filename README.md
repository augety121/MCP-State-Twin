# MCP State Twin

> **Fork the tool world, not production.**

MCP State Twin is an open-source, deterministic, forkable **stateful service-virtualization runtime for AI agent tools**.

It exposes an agent-facing MCP tool surface compatible with the real service, but executes calls against an isolated simulated world that can be snapshotted, forked, reset, diffed, time-travelled, and fault-injected.

**Status:** specification-first / pre-implementation. The current repository contains the engineering RFC and failure analysis. It does **not** yet claim production readiness.

## The problem

A serious agent test needs more than fake JSON.

An agent may:

1. search an issue,
2. create a branch,
3. edit state,
4. retry after a timeout,
5. observe the new state,
6. choose a different next tool depending on what happened.

Running that against real GitHub/Jira/CRM/payment infrastructure is slow, destructive, expensive, rate-limited, hard to reset, and hard to reproduce.

Record/replay solves a different problem: it is excellent when the future call sequence resembles what was recorded. A changed model, prompt, or planner can take a new valid trajectory that was never captured.

MCP State Twin gives the agent a **real state machine instead of a pile of canned responses**.

```text
                         production
                             X
                             |
                     no live writes
                             |
Agent / ChatGPT / Claude / Codex
              |
              | MCP tools
              v
+---------------------------------------+
|           MCP State Twin              |
|                                       |
| Tool-compatible data plane            |
| Deterministic transition engine       |
| Virtual clock + seeded faults         |
| Snapshot / fork / reset / state diff  |
+-------------------+-------------------+
                    |
                    v
              isolated state
```

## What makes it different

MCP State Twin is **not** another agent framework, memory system, MCP gateway, mock-data generator, or generic eval dashboard.

The core abstraction is **TwinSpec**: a reviewable contract describing the state and behavior behind an agent-visible tool surface.

```yaml
apiVersion: statetwin.dev/v1alpha1
kind: Twin
metadata:
  name: issue-tracker

state:
  entities:
    issue:
      key: [repo, number]
      schema: {...}

tools:
  - name: create_issue
    reads:
      - repository($input.repo)
    preconditions:
      - repository.exists == true
    effects:
      - insert issue(...)
    postconditions:
      - created_issue.number > 0
    result:
      template: {...}
```

The runtime executes the contract deterministically.

## The hard guarantees we are designing for

In hermetic mode:

- no hidden upstream writes;
- simulation-control tools are not visible to the tested agent;
- same runtime + spec + snapshot + seed + ordered tool calls produce the same simulated outputs and final state;
- unknown behavior fails explicitly instead of inventing a plausible success;
- every L2 contract is bound to an upstream tool-surface fingerprint;
- a single normal tool transition is atomic;
- every scenario runs in its own forked world;
- correctness is checked with deterministic state assertions, not an LLM judge.

See [`docs/RFC-0001.md`](docs/RFC-0001.md) for the normative design.

## Fidelity levels

We do not use the word “twin” to imply perfect equivalence.

| Level | Meaning | Intended use |
|---|---|---|
| L0 | cassette replay | smoke tests, exact regressions |
| L1 | stateful template twin | development and broad workflow tests |
| L2 | contract-backed, human-reviewed, differentially tested | CI/eval environment |
| L3 | native/reference domain logic | high-fidelity training/simulation |

Every twin declares its level and uncovered behavior. Automated inference never silently promotes a twin to L2/L3.

## Planned workflow

```bash
# inspect a real MCP tool surface
statetwin inspect --upstream https://example.com/mcp

# optionally record safe fixture-account interactions
statetwin record --upstream https://example.com/mcp

# generate a DRAFT TwinSpec from schemas + traces
statetwin init --surface surface.json --traces .statetwin/traces

# human review + deterministic validation
statetwin validate twin.yaml

# serve the isolated twin
statetwin serve --spec twin.yaml --state fixtures/base.json

# branch the world for an eval run
statetwin snapshot create --name base
statetwin fork base --branch model-a-run-1
statetwin fork base --branch model-b-run-1

# compare terminal world states
statetwin diff model-a-run-1 model-b-run-1
```

CLI details are still proposed and may change before v0.1.

## Why MCP

MCP gives us a model/provider-neutral tool boundary. The latest MCP specification defines a standardized host/client/server protocol for exposing tools and context. The project will target the current protocol line through an official Tier-1 SDK instead of hand-writing the transport.

This means the same twin can be exposed to:

- ChatGPT Developer Mode / OpenAI integrations that consume remote MCP tools;
- Claude MCP connectors and Claude Code;
- Codex and other MCP-capable coding agents;
- custom agent harnesses.

Protocol compatibility does **not** mean different models will choose the same tools or produce the same trajectories.

## Why this matters for agent engineering

Stateful tool sandboxes are increasingly used in agent benchmarks and training because real tool use is interdependent: one call changes what future calls should return. Today, however, teams often hand-build those environments per benchmark or rely on static mocks/recordings.

The project goal is to make **forkable agent worlds** a reusable infrastructure primitive.

Possible engineering uses:

- deterministic CI for tool-using agents;
- model/prompt upgrade regression tests;
- zero-side-effect shadow evaluation;
- seeded fault and recovery testing;
- RL / trajectory generation on isolated stateful tools;
- cross-model comparison from identical world snapshots;
- security tests using synthetic production-shaped state;
- counterfactual plan evaluation without touching the real system.

## What we explicitly do not promise

We will not claim:

- one-click perfect simulation of any SaaS;
- automatic discovery of undocumented business rules;
- identical behavior across ChatGPT, Claude, Gemini, or other models;
- perfect prediction of production;
- that a low-fidelity twin is equivalent to its upstream;
- that an LLM-generated TwinSpec is trustworthy without validation.

If a behavior is unknown, the correct result is **unknown**, not a hallucinated success.

## Architecture

The project is split into two trust domains.

### Agent data plane

Exposes only the simulated business tools.

### Private control plane

Owns snapshots, forks, resets, virtual time, fault schedules, exports, and state inspection. It is never advertised to the tested agent by default.

This separation is a security and benchmark-integrity invariant, not a UI choice.

## Implementation direction

The RFC currently recommends:

- Go for the runtime and single-binary CLI;
- the official Tier-1 MCP Go SDK;
- SQLite as the first transactional state backend;
- YAML authoring + canonical JSON hashing for TwinSpec;
- a bounded declarative expression language, not arbitrary embedded scripts;
- OpenTelemetry as optional telemetry rather than a core dependency.

No implementation dependency is frozen until its ADR is merged.

## Engineering docs

- [`docs/RFC-0001.md`](docs/RFC-0001.md) — normative architecture/specification
- [`docs/FAILURE-MODE-MATRIX.md`](docs/FAILURE-MODE-MATRIX.md) — 100 concrete failure modes and required behavior
- [`docs/COMPETITIVE-LANDSCAPE.md`](docs/COMPETITIVE-LANDSCAPE.md) — prior art and rejected directions
- [`docs/ROADMAP.md`](docs/ROADMAP.md) — implementation sequence and launch gates

## Research basis

This design deliberately builds on prior work instead of hiding it:

- MCP Specification: https://modelcontextprotocol.io/specification/2026-07-28
- Official MCP Go SDK: https://github.com/modelcontextprotocol/go-sdk
- OpenAI ChatGPT Developer Mode: https://developers.openai.com/api/docs/guides/developer-mode
- Anthropic MCP Connector: https://platform.claude.com/docs/en/agents-and-tools/mcp-connector
- Agent VCR: https://pypi.org/project/agent-vcr/
- Cisco MCP mock toolkit: https://github.com/cisco-open/mcptoolkit-mock
- ComplexMCP: https://arxiv.org/abs/2605.10787
- PROVE stateful MCP environments: https://arxiv.org/abs/2606.03892
- State Twins: https://arxiv.org/abs/2605.11522
- Stateful service emulation research: https://arxiv.org/abs/2510.18519

As of the research date, we found strong adjacent work but did not find a mature open-source project owning this exact combined scope: **general MCP-compatible stateful service virtualization + explicit TwinSpec + forkable snapshots + deterministic execution + trace-assisted bootstrap + fidelity/differential validation**.

That is not proof that no similar project exists. If stronger prior art appears, the positioning will be updated rather than ignored.

## License

Planned: Apache-2.0 or MIT. Final choice will be made before the first code release.