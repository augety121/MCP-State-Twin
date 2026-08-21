# Host Isolation & Benchmark Integrity

> **Status:** Proposal / Unverified  
> **Purpose:** Ensure live-agent results measure interaction with the declared world rather than hidden access to evaluator state, extra tools, persistent memory or answer leakage.

## 1. Threat model

A coding agent may have access to:
- shell;
- filesystem;
- repository files;
- web;
- editor APIs;
- persistent memory;
- host plugins/skills/hooks;
- environment variables;
- local process list/network;
- cached previous conversations.

Therefore "the MCP tools were correct" is insufficient for benchmark integrity.

## 2. Isolation profiles

### `mcp_only`

Strictest profile.

Agent has only the declared MCP business tools plus unavoidable host primitives.

Use where host supports this.

### `declared_mixed`

Agent also has declared built-ins such as shell/filesystem/editor.

All such capabilities are recorded and the scenario is designed so they do not reveal evaluator truth.

### `uncontrolled`

The harness cannot characterize significant extra capabilities.

Results MAY be diagnostic but MUST NOT be used for strict cross-host comparison.

## 3. Control secret isolation

### ST-ISO-R001
The evaluated agent process MUST NOT inherit the State Twin control-plane token.

### ST-ISO-R002
Control-plane URLs/credentials MUST not be present in model-visible project files.

### ST-ISO-R003
Expected terminal state and private assertions MUST not be placed in the agent workspace.

### ST-ISO-R004
Evidence directories containing answers/private state MUST be outside the agent-readable workspace under strict profiles.

## 4. Partial observability

State is divided into:

```text
agent-observable world state
evaluator-visible world state
runtime-internal metadata
host-only metadata
```

The agent learns world state only through allowed observation/actions.

Evaluator/control state is not a tool.

## 5. Workspace isolation

Record:
- repository commit;
- dirty worktree state/digest;
- generated files;
- instruction files;
- hidden/private test files;
- network policy;
- filesystem mounts.

Strict evaluation should use a fresh worktree/container/working directory when feasible.

## 6. Instruction contamination

Agent instruction files can change behavior.

Record digests for observable instructions such as:
- AGENTS.md;
- CLAUDE.md;
- GEMINI.md;
- project rules;
- skills/plugins/hooks.

A benchmark prompt MUST NOT assume identical instruction precedence across hosts.

## 7. Persistent memory

### ST-ISO-R020
Every Host Profile MUST state whether model/host memory reset is guaranteed, not guaranteed, or unknown.

### ST-ISO-R021
If reset is unknown, repeated trials cannot be described as fully independent.

### ST-ISO-R022
A fresh State Twin branch does not satisfy the host-memory reset requirement.

## 8. Scenario secrecy tiers

### Public / development

Scenarios and expected outcomes may be public.

Useful for:
- debugging;
- examples;
- agent coaching.

### Validation

Scenario definition may be public; exact generated instance/seed withheld until run.

### Held-out / private

Objective is supplied, but private assertions/expected state and possibly generated seed remain hidden from the agent.

Used for stronger benchmark claims.

The tier is evidence.

## 9. Scenario contamination

A scenario MUST NOT encode the answer through:
- filenames;
- branch names;
- tool descriptions;
- entity IDs;
- private comments;
- expected-state files;
- environment variable names.

## 10. Side-effect budget

A scenario MAY declare:

```yaml
budgets:
  toolCalls: 20
  writes: 5
  deletes: 0
  controlPlaneCallsByAgent: 0
```

Evaluation can fail or penalize unnecessary/destructive behavior without requiring exact trajectory equality.

## 11. Network isolation

Strict local profiles SHOULD deny network access except declared endpoints.

If web access is available:
- record it;
- scenario must not rely on hidden network restrictions;
- result belongs to `declared_mixed`, not `mcp_only`.

## 12. Built-in tools

Coding-agent hosts often provide shell/filesystem/editor capabilities.

The Host Profile must record them.

If built-ins can directly edit the simulated state DB, access expected state or invoke the control plane, the benchmark is invalid.

## 13. Training/coaching contamination

Runs using:
- step hints;
- post-error coaching;
- exposed expected diff;
- replay of previous solution;

must be labeled `coaching` or `curriculum`.

They MUST NOT count as blind evaluation.

## 14. Reproducible episode setup

Before run:

```text
fresh workspace
+ declared instructions
+ declared host config
+ isolated credentials
+ fresh State Twin branch
+ hidden evaluator artifacts
+ declared network policy
```

After run:
- collect world evidence;
- collect allowed host evidence;
- destroy/revoke ephemeral secrets;
- clean workspace.

## 15. Benchmark validity result

The harness SHOULD emit:

```text
VALID
VALID_WITH_DECLARED_MIXED_CAPABILITIES
INVALID_SECRET_EXPOSURE
INVALID_EXPECTED_STATE_EXPOSURE
INVALID_CONTROL_PLANE_ACCESS
INVALID_WORKSPACE_CONTAMINATION
INVALID_HOST_STATE_UNKNOWN
UNKNOWN
```

## 16. Feasibility

**High** for controlled local coding-agent evaluations.

Cloud-host agents may provide less isolation introspection; those profiles must state the limitation rather than infer equivalence.
