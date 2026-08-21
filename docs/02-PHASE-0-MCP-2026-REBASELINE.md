# Phase 0 — MCP 2026-07-28 Protocol Rebaseline

> **Status:** Proposal  
> **Current evidence:** Not re-run  
> **Reason:** Protocol truthfulness is prerequisite to all later compatibility claims.

## 1. Research basis

Official MCP `2026-07-28` changes the lifecycle substantially:
- stateless core;
- no legacy initialize lifecycle for the modern protocol;
- per-request protocol/client capability metadata;
- `server/discover`;
- MRTR;
- routing headers;
- list caching metadata;
- authorization hardening;
- formal extensions;
- Tasks as extension;
- several legacy features deprecated.

The official Go SDK has 2026 support, but current SDK issues/compliance fixes demonstrate that “using official SDK” is not itself proof of complete conformance.

The official conformance project supports explicit `2026-07-28` selection and expected-failure baselines.

See `17-SOURCE-REGISTRY.md`.

## 2. Profiles

### Modern

```yaml
protocol: "2026-07-28"
transport: streamable-http
lifecycle: stateless
```

### Legacy, only if intentionally retained

```yaml
protocol: "<=2025-11-25"
lifecycle: initialize
```

Do not mix lifecycle assumptions.

## 3. Requirements

### ST-MCP-R001 — Explicit protocol registry
Runtime MUST explicitly define supported versions instead of accepting whatever an SDK happens to advertise.

### ST-MCP-R002 — Modern lifecycle
`2026-07-28` MUST NOT depend on `initialize` / `notifications/initialized`.

### ST-MCP-R003 — server/discover
A server claiming `2026-07-28` MUST implement `server/discover`.

### ST-MCP-R004 — Client discovery is optional
Tests MUST include a modern client sending a non-discover RPC first.

### ST-MCP-R005 — Stateless HTTP
The 2026 Streamable HTTP endpoint MUST use the SDK/runtime mode required for the modern stateless lifecycle.

### ST-MCP-R006 — Explicit application state
Branch/world state MUST not depend on protocol session identity.

### ST-MCP-R007 — Header/meta validation
Header/body/version inconsistencies MUST fail deterministically according to the selected protocol behavior.

### ST-MCP-R008 — Modern result shape
Required modern result discriminators/metadata MUST be validated by State Twin's own tests, not assumed from SDK use.

### ST-MCP-R009 — Cache semantics
Tool/list cache metadata MUST not bypass binding/drift policy.

### ST-MCP-R010 — Deprecated core dependencies
New State Twin core behavior MUST NOT depend on roots, sampling, logging or deprecated legacy HTTP+SSE.

### ST-MCP-R011 — MRTR
MRTR is capability/profile-gated.

### ST-MCP-R012 — Tasks
Tasks are optional extension behavior, not v0.1 core.

### ST-MCP-R013 — Hidden control plane
Snapshot/fork/reset/diff/clock/fault controls MUST stay absent from agent `tools/list`.

### ST-MCP-R014 — SDK evidence
Every verified protocol run records exact Go SDK module version.

### ST-MCP-R015 — Conformance evidence
Every claim records exact conformance package/commit, protocol version, suite/scenarios, expected failures and raw artifact.

## 4. Gap matrix

Use `templates/MCP-2026-07-28-GAP-MATRIX.md`.

Never mark PASS from README text alone.

## 5. Required test categories

- `server/discover`
- direct first RPC
- tools/list
- tools/call
- protocol version mismatch
- malformed metadata
- header mismatch
- unknown method
- result discriminator
- JSON Schema tool contracts
- legacy lifecycle only if supported
- deprecated transport rejection/policy
- extension negotiation
- cancellation behavior relevant to current server

## 6. Exit gate

Phase 0 is complete when:
- exact modern capability matrix exists;
- exact legacy policy exists;
- official conformance version is pinned;
- raw output is archived;
- gaps are either closed or explicit expected failures;
- README claim mirrors evidence.

## 7. Feasibility

**High.**

Main risk: fast ecosystem changes.  
Mitigation: explicit versions + repeatable conformance, not “latest”.
