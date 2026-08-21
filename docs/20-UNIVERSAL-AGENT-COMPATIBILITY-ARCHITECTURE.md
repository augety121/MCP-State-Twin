# Universal Agent Compatibility Architecture

> **Status:** Proposal / Unverified  
> **Purpose:** Define how MCP State Twin can be systematically integrated with many current and future agent hosts without pretending that protocol compliance automatically equals host/model compatibility.  
> **Research cut:** 2026-08-18

## 1. Problem statement

The project should not aim for a literal claim such as:

> "compatible with all agents"

That statement is not technically defensible because agent hosts expose different:
- MCP feature subsets;
- transports;
- approval models;
- authentication models;
- tool-name transforms;
- schema transforms;
- tool-count/context limits;
- built-in tools;
- retry/cancellation behavior;
- output transformations;
- persistence/memory behavior.

The correct target is:

> **A universal compatibility architecture in which every host can be represented by a versioned capability profile, an optional projection/adapter, and executable evidence.**

## 2. Compatibility chain

Every live-agent integration should be modeled as:

```text
Canonical State Twin World
        |
        v
Canonical MCP Server Surface
        |
        v
Authorization-Scoped Surface
        |
        v
Host Projection / Adapter
        |
        v
Host-Visible Surface
        |
        v
Model-Visible/Selected Surface
        |
        v
Observed Calls
        |
        v
World Commit Trace
        |
        v
Terminal State + Evidence
```

The key rule:

> **The canonical server surface and the model-visible surface are not assumed to be identical.**

## 3. Compatibility object

Compatibility is a tuple:

```text
Compatibility =
    world_runtime_profile
  + protocol_profile
  + host_product_profile
  + host_projection_profile
  + model_profile
  + isolation_profile
  + scenario_profile
  + evidence
```

No single boolean may replace this tuple.

## 4. Compatibility statuses

A host entry may use lifecycle states:

```text
documented
adapter_ready
connected
scenario_verified
stale
blocked
unsupported
unknown
```

These are operational lifecycle states only. The primary truth remains the detailed capability vector.

## 5. Native compatibility classes

### Class N — Native MCP

The host directly consumes MCP.

Preferred where available.

### Class A — MCP through an editor/agent bridge

Example architecture:

```text
Editor/IDE
   |
   | ACP or host-native agent transport
   v
Coding Agent
   |
   | forwarded/configured MCP
   v
State Twin
```

The bridge must be part of the Host Profile and evidence.

### Class B — Compatibility bridge

A non-MCP host MAY use a future adapter mapping canonical State Twin tool contracts to a provider-specific function/tool API.

This is not MCP conformance.

### Class U — Unsupported/unknown

No safe verified adapter exists.

Do not fake compatibility.

## 6. Portable common denominator

The baseline portable host profile SHOULD depend only on:

```text
MCP business tools
+ bounded JSON tool input/output
+ explicit tool errors
+ terminal-state scoring
```

The baseline MUST NOT require:
- resources;
- prompts;
- roots;
- elicitation;
- Tasks;
- MRTR;
- host-specific UI;
- hidden reasoning;
- provider-specific memory.

Those features MAY exist in optional capability profiles.

## 7. Transport strategy

Recommended target hierarchy:

```text
Streamable HTTP   preferred universal network profile
stdio             optional local-host profile
legacy SSE        compatibility-only where still required
```

Exact transport support remains host-specific and evidence-backed.

## 8. Host-surface projection

A host projection describes every transformation between canonical server tools and the surface the model actually receives.

Example:

```yaml
canonicalTool: close_issue
hostToolName: mcp_statetwin_close_issue
sourceSchemaDigest: sha256:...
projectedSchemaDigest: sha256:...
transforms:
  - tool_name_prefix
  - strip_schema_keyword
lossClass: syntactic
```

Allowed loss classes:

```text
none
syntactic
semantic-risk
unsupported
unknown
```

### ST-HCOMP-R001
Every known name/schema/output transformation MUST be recorded.

### ST-HCOMP-R002
A `semantic-risk` projection MUST block strict upstream-fidelity claims until differential evidence shows the transformed interface preserves the declared behavior.

### ST-HCOMP-R003
A projection MUST preserve a mapping back to the canonical tool identity.

## 9. Host-specific limits

Host limits belong in the Host Profile.

Examples of limit categories:
- maximum simultaneously available tools;
- tool-description/context budget;
- tool-name limits after host prefixing;
- schema feature support;
- result/output size;
- approval policy;
- concurrency;
- number of configured servers.

State Twin must not choose a universal numeric limit merely because one host imposes it.

Instead:
- canonical world limits remain project-defined;
- host adapters/lint detect incompatibility with selected target hosts;
- scenario bundles expose only the smallest required tool set where possible.

## 10. Tool-name portability

Canonical MCP tool names should remain conservative and stable.

The compatibility layer MUST account for hosts/proxies that:
- prefix names;
- sanitize names;
- truncate names;
- resolve collisions.

A host-generated name is evidence, not canonical identity.

## 11. Schema portability

The compatibility layer must distinguish:

```text
canonical schema
host-projected schema
model-visible schema
```

If a host strips or rewrites schema keywords, State Twin preserves:
- original digest;
- projected digest;
- transformation log;
- compatibility loss classification.

Never silently pretend the model saw the canonical contract.

## 12. Structured result portability

For State-Twin-native tools, the portable tools profile SHOULD provide:
- `structuredContent` when useful;
- a canonical serialized JSON text representation when required for broad backwards compatibility.

Exception:
- strict fidelity mode may not add output fields/content that the bound upstream does not expose.

This is resolved by explicit execution mode.

## 13. Execution modes

### `blind_eval`

- no answer hints;
- no expected-state leakage;
- no training feedback during run;
- suitable for benchmark evidence.

### `diagnostic_eval`

- same run isolation as blind eval;
- detailed evidence released after completion to evaluator/human;
- not fed back into the same episode.

### `coaching`

- post-step or post-run feedback MAY be supplied to the agent;
- intended to improve agent behavior;
- results are not comparable to blind evaluation.

### `curriculum`

- progressive scenario families;
- feedback policy explicitly declared.

### `stress`

- faults, concurrency, adversarial state, permissions and ambiguous outcomes.

The mode MUST be part of episode identity.

## 14. Fidelity mode vs augmentation mode

### Fidelity mode

Goal:
- represent the bound upstream contract as closely as evidence supports.

Rules:
- no State Twin-only hints visible to the model;
- no extra semantics merely to make agents perform better;
- host projection differences recorded;
- fidelity claims coverage-scoped.

### Augmentation mode

Goal:
- use State Twin as an agent capability-improvement environment.

May provide:
- richer diagnostic output;
- explicit recovery feedback;
- curriculum;
- safer synthetic tasks.

Augmentation results MUST NOT be used as evidence of upstream equivalence.

## 15. Universal compatibility gate

A host can enter the compatibility matrix only if:

1. official/primary capability sources exist or behavior is directly probed;
2. Host Profile is created;
3. host projection is characterized;
4. isolation mode is declared;
5. adapter has fixture tests;
6. connection smoke succeeds;
7. at least one stateful scenario runs;
8. evidence is archived;
9. known limitations are listed.

## 16. Compatibility is not transitive

Examples:

```text
Claude Code verified
!= Anthropic Messages MCP verified

GitHub Copilot VS Code verified
!= GitHub coding agent verified

Codex local MCP verified
!= OpenAI Responses remote MCP verified
```

Each product/profile is separate.

## 17. Universal-agent design invariant

The long-term design target is:

> New agents should usually require a new Host Profile and adapter/projection, not changes to TwinSpec world semantics.

If a host forces a world-semantic change only because of its UI/API quirks, the adapter boundary has failed.

## 18. Feasibility

**High** for a broad set of MCP-capable coding/agent hosts.

**Not provable** as literal all-agent compatibility.

The architecture instead makes unsupported hosts explicit and provides a controlled bridge path for future non-MCP environments.
