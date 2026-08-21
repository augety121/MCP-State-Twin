# Cross-Protocol Boundary — MCP, ACP, A2A & Future Bridges

> **Status:** Forward Architecture Proposal / Unverified  
> **Purpose:** Prevent MCP State Twin from becoming an unfocused agent protocol stack while still allowing integration with editor-agent and agent-agent ecosystems.

## 1. Protocol roles

The architecture should distinguish:

```text
MCP
  host/agent <-> tools/context/world capabilities

ACP
  editor/client <-> coding agent

A2A
  agent <-> remote/peer agent
```

These roles are complementary.

## 2. State Twin ownership

MCP State Twin owns:

```text
World Runtime
MCP Agent Data Plane
Simulation Control Plane
Evidence
```

It does NOT become:
- an ACP coding agent;
- an A2A autonomous agent;
- a generic agent router;

by default.

## 3. ACP composition

An editor may use ACP to run an external agent, while that agent receives MCP servers from the editor or its own config.

Composition:

```text
Editor
  |
  | ACP
  v
Agent
  |
  | MCP
  v
MCP State Twin
```

The ACP layer is part of Host Profile evidence if it affects configuration or tool visibility.

## 4. A2A composition

Future multi-agent systems may delegate between agents over A2A while each agent uses MCP for tools.

Example:

```text
Agent A
  |
  | A2A delegation
  v
Agent B
  |
  | MCP
  v
State Twin
```

State Twin remains the tool/world endpoint.

## 5. No credential inheritance

### ST-XPROTO-R001
Credentials accepted on one protocol boundary MUST NOT automatically authorize another boundary.

### ST-XPROTO-R002
MCP control-plane credentials MUST never be propagated through ACP/A2A.

### ST-XPROTO-R003
Bridge identity/authorization must be explicit.

## 6. Evidence propagation

Cross-protocol evidence should preserve:

```text
origin host
agent identity if exposed
delegation/parent identity
MCP request identity
world commit identity
```

Do not require private chain-of-thought or unavailable internal IDs.

## 7. Protocol-neutral internal semantics

Long term, it is beneficial for the world engine to have an internal semantic interface independent from MCP wire types.

However:
- do not overengineer a second public protocol today;
- introduce an adapter only when real non-MCP demand exists.

## 8. Function-tool bridge

A future non-MCP host MAY map canonical State Twin tools to its native function-call system.

Requirements:
- same canonical tool identity;
- explicit projection;
- no MCP conformance claim;
- host/profile-specific evidence;
- semantic differences classified.

## 9. ACP versioning

ACP itself evolves.

State Twin should record ACP version only in profiles where ACP is actually part of the path.

Do not add ACP as a core dependency merely because some editors support it.

## 10. A2A versioning

Likewise, A2A may evolve.

Use it only if a concrete agent-to-agent evaluation needs it.

## 11. Why this matters for future agents

Separating protocol roles allows MCP State Twin to remain a stable external-world layer while:
- editors change;
- coding-agent protocols change;
- agent delegation protocols change.

The world contract does not need to absorb every orchestration protocol.

## 12. Feasibility

**High as an architectural boundary.**

No new runtime implementation is required until an actual ACP/A2A integration scenario is chosen.
