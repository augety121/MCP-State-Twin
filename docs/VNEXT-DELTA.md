# vNext SPEC Delta — Universal Agent Compatibility

> **Status:** Proposal delta  
> **No implementation is implied.**

This refinement does not replace the deterministic/fidelity lifecycle created previously. It adds the missing compatibility/evaluation framework required to make that runtime usable across heterogeneous current and future agent hosts.

## Main additions

### 1. Host Surface Projection

The spec now explicitly models:

```text
canonical MCP surface
 -> authorization-scoped surface
 -> host projection
 -> model-visible surface
```

This prevents protocol conformance from being confused with model-visible equivalence.

### 2. Portable MCP Tools Profile

The cross-host baseline is intentionally tools-first.

Resources/prompts/roots/elicitation/Tasks/MRTR remain optional profiles.

### 3. Host Adapter SPI

Host-specific config, permissions, schema/name transforms, readiness and output behavior live outside TwinSpec core.

### 4. Current Agent Host Matrix

Research profiles now include:
- Codex / OpenAI API
- Claude Code / Anthropic Messages / Managed Agents
- Gemini CLI
- GitHub Copilot IDE/CLI / cloud agent
- Cursor
- Windsurf
- Cline
- Amazon Q
- JetBrains/Junie
- Zed
- OpenCode
- custom MCP clients

All rows are documented-only until State Twin evidence exists.

### 5. Host Isolation & Benchmark Integrity

Adds:
- control-token isolation;
- hidden expected-state protection;
- workspace identity;
- built-in tool inventory;
- agent-memory reset state;
- public/validation/held-out tiers.

### 6. Episode Orchestrator

Defines a single run as:

```text
Bundle
+ Scenario
+ Host Profile
+ Projection
+ Isolation
+ Mode
+ Snapshot
+ Fault/Time profile
+ Budgets
+ Scoring
```

### 7. Capability Uplift

Separates:
- blind evaluation;
- diagnostic evaluation;
- coaching;
- curriculum;
- stress.

This allows State Twin to improve agent-system behavior without contaminating benchmark claims.

### 8. Scenario Families

Adds deterministic generation and metamorphic coverage to reduce overfitting to a handful of public scenarios.

### 9. MCP / ACP / A2A Boundary

Keeps:
- MCP for world/tool interaction;
- ACP for editor-agent integration;
- A2A for agent-agent collaboration.

State Twin remains the external-world layer.

### 10. Compatibility CI

Adds:
- static host/source lint;
- surface lint;
- adapter fixtures;
- protocol conformance;
- real-host smoke;
- repeated evaluation;
- evidence freshness/staleness.

### 11. Reference Framework

The project is now organized conceptually as seven planes:

```text
Storage
World
Protocol
Evidence
Host Compatibility
Evaluation
Governance
```

Fast-changing host logic stays above the deterministic world.

## New requirement namespaces

- `ST-HCOMP-*`
- `ST-PORT-*`
- `ST-ADAPT-*`
- `ST-ISO-*`
- `ST-EPISODE-*`
- `ST-UPLIFT-*`
- `ST-FAM-*`
- `ST-XPROTO-*`
- `ST-COMPATCI-*`
- `ST-ART-*`

## Important non-claim

This refinement still does NOT claim:
- current State Twin compatibility with any listed host;
- literal support for every Agent;
- complete coverage of future AGI behavior;
- that any proposed adapter or compatibility-lint command exists.

It defines the framework needed to make future compatibility claims rigorous.
