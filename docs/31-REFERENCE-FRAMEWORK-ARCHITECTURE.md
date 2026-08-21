# MCP State Twin — Reference Framework Architecture

> **Status:** Proposal / Unverified  
> **Purpose:** Translate the lifecycle SPEC into a concrete but implementation-neutral system framework.  
> **Important:** Package/module names are suggested boundaries, not current repository claims.

## 1. Architectural objective

The framework should isolate fast-changing agent-host behavior from slow-changing deterministic world semantics.

The architecture is divided into seven planes:

```text
+--------------------------------------------------------------+
| 7. Compatibility & Release Governance                       |
+--------------------------------------------------------------+
| 6. Evaluation / Curriculum / Benchmark Plane                |
+--------------------------------------------------------------+
| 5. Host Compatibility Plane                                 |
|    Profiles / Adapters / Surface Projection / Isolation     |
+--------------------------------------------------------------+
| 4. Evidence Plane                                           |
|    World Trace / Host Observation / Evaluation / Claims     |
+--------------------------------------------------------------+
| 3. Protocol Plane                                           |
|    MCP Agent Data Plane / Private Control Plane             |
+--------------------------------------------------------------+
| 2. Deterministic World Plane                                |
|    State / Effects / Scheduler / Faults / Invariants        |
+--------------------------------------------------------------+
| 1. Storage & Artifact Plane                                 |
|    SQLite / Snapshots / Cassettes / Bundles / Artifacts     |
+--------------------------------------------------------------+
```

A higher plane may depend on a lower plane. The reverse dependency should be avoided.

## 2. Plane 1 — Storage & Artifact Plane

Responsibilities:

```text
database identity
storage schema
branch heads
snapshots
migration
audit persistence
cassette storage
bundle import/export
artifact content addressing
```

Should not know:
- Codex;
- Claude;
- Gemini;
- host approval UI;
- benchmark score.

## 3. Plane 2 — Deterministic World Plane

Responsibilities:

```text
TwinSpec compiled model
entities/state
tool transition engine
preconditions
effects
queries
postconditions
global invariants
virtual time
deterministic entropy
scheduled events
fault plan
canonical state
```

Core API concept:

```text
Execute(world, canonical_tool_call, semantic_profile)
    -> world_result
```

This plane is the long-lived technical heart of the project.

## 4. Plane 3 — Protocol Plane

### Agent Data Plane

Maps MCP requests to canonical world calls.

Responsibilities:
- protocol version;
- tool listing;
- tool calling;
- structured/unstructured result mapping;
- protocol errors;
- auth profile where enabled.

It MUST NOT own branch reset/fork/fault policy.

### Control Plane

Privileged harness/operator API:

```text
state
snapshot
fork
reset
diff
clock
fault
evidence control
```

Separate authorization and endpoint.

## 5. Plane 4 — Evidence Plane

Responsibilities:

```text
environment identity
world call trace
commit ordering
raw MCP results
host observation
assertions
terminal diff
claim references
redaction
evidence versioning
```

Evidence plane observes the runtime; it must not silently alter world semantics.

## 6. Plane 5 — Host Compatibility Plane

Subcomponents:

### Host Registry

Stores documented/probed host profiles.

### Surface Projector

Maps canonical server surface to host-specific representation.

### Compatibility Linter

Checks whether a Twin/scenario can be represented safely on target hosts.

### Host Adapter SPI

Configures/launches/attaches to agent hosts.

### Isolation Resolver

Determines:
- `mcp_only`;
- `declared_mixed`;
- `uncontrolled`.

### Surface Readiness Detector

Ensures required tool surface is ready before scoring.

This plane changes faster than the deterministic world plane.

## 7. Plane 6 — Evaluation Plane

### Episode Orchestrator

Creates one reproducible episode.

### Scenario Runner

Executes scripted scenarios.

### Live Agent Runner

Lets a host/model choose trajectory.

### Scenario Family Generator

Creates deterministic held-out instances.

### Scorer

Evaluates:
- state;
- invariants;
- safety;
- recovery;
- side-effect budgets;
- efficiency metrics.

### Curriculum Controller

Selects progressive episodes for capability improvement.

## 8. Plane 7 — Governance Plane

Responsibilities:

```text
SPEC status
requirement traceability
compatibility claim registry
evidence freshness
release gates
deprecation
migration
source registry
```

This layer prevents documentation from overstating implementation.

## 9. Suggested internal domain interfaces

### `WorldEngine`

Conceptually:

```text
ValidateSpec
InitWorld
ExecuteTool
InspectState
Snapshot
Fork
Diff
AdvanceTime
```

### `HostAdapter`

Conceptually:

```text
Describe
ProbeCapabilities
ProjectSurface
Prepare
WaitReady
Run
Collect
Cleanup
```

### `EpisodeRunner`

Conceptually:

```text
ResolveEpisode
PrepareWorld
PrepareHost
VerifyIsolation
RunObjective
Finalize
Score
PersistEvidence
Cleanup
```

### `EvidenceWriter`

Conceptually:

```text
BeginRun
RecordWorldEvent
RecordHostObservation
RecordAssertion
FinalizeRun
```

These are semantic responsibilities, not finalized Go interfaces.

## 10. Suggested dependency rule

Allowed:

```text
episode -> host adapter -> protocol client/config
episode -> world/control client
protocol server -> world engine
world engine -> storage
evidence -> stable event interfaces
```

Avoid:

```text
world engine -> codex package
world engine -> claude package
TwinSpec parser -> GitHub Copilot policy
storage -> host adapter
```

## 11. Canonical Call abstraction

To reduce protocol coupling, consider an internal call form:

```yaml
call:
  canonicalTool: close_issue
  arguments: {...}
  caller:
    agentId: null
  metadata:
    requestId: ...
```

MCP maps into this form.

A future non-MCP bridge could also map into it without changing Twin semantics.

Do not build a public second protocol until required.

## 12. Canonical Result abstraction

```yaml
result:
  status: success|domain_error|unmodeled
  output: ...
  errorClass: null
  preStateDigest: ...
  postStateDigest: ...
  commitSequence: ...
```

Protocol-specific formatting happens above it.

## 13. Host projection architecture

```text
TwinSpec Tool
   |
   v
MCP Native Tool
   |
   +---- source digest
   |
   v
Host Projection Rule
   |
   +---- transform log
   +---- projected digest
   +---- loss class
   |
   v
Model-visible Tool
```

Projection is evidence, never hidden implementation behavior.

## 14. Episode isolation architecture

```text
Evaluator Workspace
  - expected state
  - control token
  - evidence
        X inaccessible
Agent Workspace
  - code/task data
  - declared instructions
  - host configuration
        |
        v
Agent Host
        |
        v
MCP Data Plane
```

The control plane belongs outside the agent trust domain.

## 15. Hosted/provider path architecture

For remote provider APIs:

```text
Provider Host
     |
     | Internet / authenticated MCP
     v
Ephemeral Security-Qualified Staging Data Plane
     |
     v
State Twin World
```

This remote profile is separate from local-first runtime.

## 16. ACP composition architecture

```text
IDE
 |
 | ACP
 v
Coding Agent Process
 |
 | MCP
 v
State Twin
```

The ACP integration belongs in the Host Compatibility Plane.

## 17. A2A composition architecture

```text
Coordinator Agent
   |
   | A2A
   v
Worker Agent
   |
   | MCP
   v
State Twin
```

A2A never grants State Twin control-plane access implicitly.

## 18. Scaling strategy

Before distributed architecture:
- scale by logical branches;
- optimize snapshot/storage;
- run isolated workers with independent DBs if scenarios are independent;
- aggregate evidence outside the DB.

Only adopt distributed shared storage if a real use case demands shared remote state.

## 19. Extension strategy

Core supports declarative operations.

If extensibility pressure emerges:

```text
built-in operation
 -> versioned isolated extension interface
 -> sandboxed execution
```

Avoid loading arbitrary host code into the world engine.

## 20. Framework success test

The architecture passes its long-term design test when:

> Adding a new Agent host changes primarily the Host Compatibility Plane, not the deterministic World Plane.

That is the strongest structural guarantee of future maintainability.
