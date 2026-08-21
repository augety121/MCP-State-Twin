# Host Evaluation Blueprint — Codex, Claude, APIs & Future Agents

> **Status:** Proposal / planned expansion of existing SPEC-0006  
> **Implementation baseline:** source baseline reports no live Codex/OpenAI/Claude/Claude Code smoke-test evidence  
> **Verification status:** unverified

## 1. Purpose

This document defines how MCP State Twin should be tested against real agent hosts without contaminating core world semantics with provider-specific behavior.

It deliberately separates:

```text
World Runtime
Host Adapter
Host Product
Model
Evaluation Harness
Evidence
```

A compatibility result applies to a named combination of these elements.

## 2. Core architecture

```text
Agent / Model
      |
      v
Host product / API
      |
      | MCP
      v
State Twin Data Plane
      |
      v
Deterministic World Runtime

Test Harness ------> State Twin Control Plane
     |
     +-------------> Host Adapter / CLI / API
     |
     +-------------> Evidence Collector
```

The evaluated model never needs the control plane.

## 3. Host profile schema

Compatibility MUST be a capability vector, not a boolean.

Use `templates/HOST-PROFILE.yaml`.

At minimum capture:

```yaml
identity:
  provider: ...
  product: ...
  hostVersion: ...
  observedAt: ...

model:
  configuredId: ...
  resolvedSnapshot: ...

mcp:
  protocolVersion: ...
  transport: ...
  extensions: ...

tools:
  enabled: []
  disabled: []
  allowed: []
  approvalPolicy: ...
  discoveryMode: ...

run:
  timeoutSeconds: ...
  maxTurns: ...
  budget: ...
  parallelismPolicy: ...
```

## 4. Host-independent requirements

### ST-HOST-R001 — Fresh world
Every independent comparison run MUST start from a new branch/fork of the same immutable snapshot.

### ST-HOST-R002 — Stable server surface
The model-visible State Twin business-tool surface MUST be digested before the run.

### ST-HOST-R003 — Host-visible surface
If the host filters, defers, renames or otherwise transforms the surface, the transformed/visible surface MUST be recorded separately where observable.

### ST-HOST-R004 — Control-plane isolation
Host configuration MUST expose only the Agent Data Plane.

### ST-HOST-R005 — Host configuration identity
The effective host configuration MUST be captured/digested after redacting secrets.

### ST-HOST-R006 — Model identity
The configured model ID/alias MUST be recorded. If the provider exposes a resolved snapshot/version, record it separately.

### ST-HOST-R007 — Run bounds
Live-agent runs MUST have explicit time/call/turn/budget bounds appropriate to the host.

### ST-HOST-R008 — Server-side trace
Evaluation MUST retain State Twin's own server-observed call trace independently of the model transcript.

### ST-HOST-R009 — Terminal scoring
Primary success SHOULD be based on terminal state/invariants, not exact trajectory equality.

### ST-HOST-R010 — Failure attribution
Evidence MUST distinguish:
- agent task failure;
- host failure;
- provider/API failure;
- MCP protocol failure;
- State Twin runtime failure;
- harness failure;
- evidence-integrity failure.

### ST-HOST-R011 — No transitive claims
A PASS on one host product/profile MUST NOT imply another product/profile from the same provider passes.

### ST-HOST-R012 — Freshness
Compatibility evidence MUST include observation date and SHOULD be revalidated after material host/model/provider changes.

## 5. Codex profile

### 5.1 Initial recommended topology

Start with a **local Streamable HTTP State Twin** if supported by the chosen Codex environment/profile.

Rationale:
- preserves local-first safety;
- avoids premature public exposure;
- tests actual MCP trajectory choice;
- keeps the world runtime identical to other HTTP-host tests.

### 5.2 Codex evidence

Record:
- exact Codex application/CLI/host version if exposed;
- official MCP configuration fields used at test time;
- effective server URL;
- tool filtering policy;
- approval/autonomy policy;
- model configuration;
- run bounds;
- tool calls observed by State Twin;
- final State Twin world state/diff;
- Codex completion/error metadata that can be safely collected.

Do not freeze undocumented CLI/config behavior into the core SPEC. Revalidate exact fields against current OpenAI Codex documentation when implementation begins.

### 5.3 Codex test ladder

#### C0 — Connectivity
- host can connect;
- correct server chosen;
- expected business tools visible.

#### C1 — Read only
Objective can be completed without mutation.

#### C2 — Single mutation
One state-changing tool call; verify terminal state.

#### C3 — Stateful multi-step
Agent must read world state, mutate, re-read or branch on changed state.

#### C4 — Preconditions
Agent encounters a valid modeled domain failure and recovers or reports appropriately.

#### C5 — Ambiguous outcome
After SPEC-0008: commit occurs, response is lost; evaluate retry/idempotency behavior.

#### C6 — Repeated trials
Multiple independent trials from one snapshot; report success and trajectory distribution.

#### C7 — Parallel/subagent
Only after shared-branch concurrency/evidence semantics are mature.

### 5.4 Codex claim

Allowed example:

> Verified: Codex `<version/profile>` completed scenario `<id>` in `N/M` runs against environment `<digest>` on `<date>`.

Not allowed:

> Codex compatible.

## 6. Claude Code profile

Claude Code is a distinct MCP host.

### 6.1 Initial topology

Prefer local Streamable HTTP first so the same State Twin network data plane is exercised.

A separate stdio profile MAY exist later.

### 6.2 Capture

Record:
- Claude Code version;
- MCP server config scope/source;
- project/user/local configuration as relevant;
- configuration approval state;
- effective permission policy;
- model selection;
- run timeout/turn/budget settings;
- exact test harness invocation;
- host-visible tools;
- server-observed calls;
- terminal world result.

### 6.3 Claude Code test ladder

Use the same C0–C7 shape, plus:
- host permission denied;
- tool approval flow;
- project-config approval;
- fresh-process reconnection;
- output-size handling.

### 6.4 Claim rule

Claude Code evidence never automatically applies to Anthropic's API MCP connector.

## 7. Anthropic Messages MCP connector profile

Treat as a remote-host API profile.

At the 2026-08-18 research cut, first-party Anthropic documentation describes the connector as supporting remote MCP tool use under a documented connector/beta profile rather than the entire MCP feature set.

Therefore:

### ST-HOST-R030
Only capabilities actually documented and tested for the selected connector profile may be claimed.

### ST-HOST-R031
Connector beta/version headers or profile identifiers that affect behavior MUST be recorded.

### ST-HOST-R032
Remote deployment identity and authorization profile MUST be evidence.

### ST-HOST-R033
Allow/deny/per-tool settings MUST be included in host profile.

### ST-HOST-R034
Unsupported resources/prompts/extensions MUST remain explicitly unsupported/unknown rather than inferred.

## 8. Anthropic Managed Agents profile

Managed Agents are another distinct product surface.

Evidence should capture:
- managed-agent product/profile version;
- MCP permission policy;
- enabled toolsets;
- output transformations;
- sandbox/file handling for large results;
- agent/task budget;
- server-observed trace;
- final state.

If a large MCP result is transformed into a file/preview by the host, store/digest:
1. raw State Twin result;
2. host-visible transform metadata.

## 9. OpenAI Responses/API remote MCP profile

This profile is separate from Codex.

### 9.1 Remote requirement

Use a properly security-qualified staging endpoint before live API testing.

### 9.2 Capture

Record:
- remote server deployment identity;
- model ID;
- MCP server URL identity, not secrets;
- allowed tool filters;
- approval policy;
- MCP list/call response item/event identifiers where applicable;
- server-observed calls;
- final state;
- API error/completion status.

### 9.3 Multi-call behavior

A single model API request may involve more than one MCP operation.

The harness MUST NOT assume:

```text
one model request == one tool call
```

## 10. Generic future host onboarding

A new host (Cursor, VS Code, custom agent, future AGI host, etc.) enters through:

1. primary-source capability review;
2. Host Profile;
3. adapter;
4. smoke matrix;
5. evidence;
6. only then compatibility table entry.

No provider/host gets a compatibility badge from documentation inspection alone.

## 11. Repeated-run protocol

For model-driven scenarios:

```text
same bundle
same starting snapshot
same host profile
same objective
same world limits/fault plan
N independent runs
```

Report:
- task success;
- terminal invariant pass rate;
- unexpected side effects;
- recovery success;
- calls/run;
- tool errors;
- approval interruptions;
- latency;
- cost where exposed;
- trajectory diversity.

Do not imply statistical significance from a small `N`.

## 12. Parallel calls and subagents

When the host makes concurrent calls:

Evidence SHOULD record:

```text
agent_id
parent_agent_id
host_sequence
request_id
branch_head_before
world_commit_sequence
branch_head_after
```

World determinism is conditional on the actual serialized commit schedule.

The model/host scheduler remains outside State Twin's deterministic claim.

## 13. Approval semantics

A host may:
- require human approval;
- allow a configured tool;
- deny a tool.

Approval is part of the **host profile**, not TwinSpec business semantics.

Server authorization still applies independently.

## 14. Host retries

Hosts may automatically retry network/tool operations.

If observed:
- retry behavior should be recorded;
- server-side call IDs/counts remain source of truth for actual world interactions;
- duplicate/idempotency semantics are evaluated by State Twin.

## 15. Output transforms

Hosts may:
- truncate;
- summarize;
- store as file;
- encode structured result differently;
- hide internal transport metadata.

Evidence MUST separate:
- server raw semantic result;
- host-delivered representation.

## 16. Data handling

Live provider tests default to synthetic state.

Do not send:
- real credentials as task data;
- production traces;
- personal data;
- unreleasable third-party recordings.

## 17. Feasibility

| Host profile | Feasibility | Main dependency |
|---|---|---|
| Codex local MCP | High | current official host configuration/version |
| Claude Code local HTTP | High | current version/permission policy |
| OpenAI API remote MCP | High after secure staging | remote endpoint/auth |
| Anthropic Messages MCP | High for documented subset | connector scope/beta |
| Managed Agents | Medium-high | product evolution/output transforms |
| Generic MCP host | High when protocol surface exists | adapter/profile |
| Non-MCP future host | Adapter-dependent | mapping to world contract |

## 18. Release gate for any compatibility row

A row cannot say `verified` until it has:
- exact Host Profile;
- environment digest;
- scenario;
- fresh run evidence;
- terminal assertions;
- date;
- no unresolved infrastructure error.
