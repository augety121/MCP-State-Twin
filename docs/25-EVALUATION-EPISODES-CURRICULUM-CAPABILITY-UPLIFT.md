# Evaluation Episodes, Curriculum & Agent Capability Uplift

> **Status:** Proposal / Unverified  
> **Purpose:** Define how MCP State Twin can improve agent capability through safe stateful practice without corrupting blind evaluation claims.

## 1. Key distinction

MCP State Twin can serve two related but different purposes:

1. **Evaluation** — measure what the agent can do.
2. **Capability uplift** — provide safe, repeatable environments in which agents can practice recovery, state reasoning and tool use.

These purposes MUST not share unlabeled evidence.

## 2. Episode abstraction

An Episode is the top-level executable unit:

```text
Bundle
  + Scenario/Objective
  + Host Profile
  + Isolation Profile
  + Execution Mode
  + Budgets
  + Starting Snapshot
  + Fault/Time Profile
  + Scoring Profile
```

## 3. Episode orchestrator

Conceptual flow:

```text
Resolve bundle
 -> validate compatibility
 -> create/fork world
 -> prepare host adapter
 -> establish isolation
 -> wait for surface readiness
 -> run objective
 -> enforce budgets
 -> finalize world
 -> evaluate assertions
 -> archive evidence
 -> cleanup
```

The orchestrator is above the runtime.

## 4. Episode schema

Example:

```yaml
episode:
  id: close-issue-recovery-v1
  mode: blind_eval
  scenarioFamily: issue-recovery
  seed: 88421

host:
  profile: claude-code-local-http

isolation:
  profile: declared_mixed

world:
  snapshot: base
  faultPlan: timeout-after-commit

budgets:
  toolCalls: 12
  writes: 3
  virtualDuration: 10m

scoring:
  requiredInvariants:
    - issue_closed
    - no_duplicate_comment
```

## 5. Capability curriculum

Recommended progressive dimensions:

### Level 0 — Discovery & observation
- find correct tool;
- inspect state;
- no write.

### Level 1 — Single state transition
- one valid mutation;
- verify result.

### Level 2 — Stateful multi-step reasoning
- observe;
- act;
- observe changed state;
- continue.

### Level 3 — Preconditions & constraints
- handle missing/invalid/conflicting state;
- respect invariants.

### Level 4 — Recovery & ambiguity
- timeout;
- ambiguous commit;
- idempotency;
- rate limits;
- retry.

### Level 5 — Time & consistency
- virtual time;
- delayed visibility;
- expiry;
- scheduled jobs.

### Level 6 — Concurrency
- shared state;
- optimistic conflict;
- competing agents/processes.

### Level 7 — Long-running / multi-agent
- tasks;
- delegation;
- cancellation;
- asynchronous world.

### Level 8 — Adversarial / security
- malicious input;
- prompt/tool injection;
- authorization boundaries;
- unsafe side-effect avoidance.

Levels describe curriculum complexity, not a universal intelligence score.

## 6. Coaching policy

Coaching can occur:
- after a failed episode;
- after a step;
- after a group of episodes.

Feedback MAY include:
- failed invariant;
- relevant state diff;
- error class;
- suggested recovery concept.

Feedback MUST NOT include hidden secrets/control tokens.

## 7. Learning loop

A safe training loop:

```text
blind attempt
 -> evidence
 -> diagnosis
 -> coaching
 -> new fork/new scenario instance
 -> retry
```

The retry result is labeled coached.

## 8. Scoring dimensions

Keep dimensions separate:

```text
task_success
terminal_invariants
unexpected_side_effects
recovery
safety
efficiency
tool_errors
approval_interruptions
latency
cost
```

Do not collapse to one weighted score before an evaluation-methodology RFC establishes justified weights.

## 9. Success equivalence classes

Different valid trajectories may lead to equivalent acceptable worlds.

A scenario SHOULD define:
- required invariants;
- forbidden invariants/effects;
- acceptable terminal-state equivalence;
- optional efficiency budget.

## 10. Side-effect minimality

Capability is not just task completion.

Measure:
- unnecessary writes;
- duplicate operations;
- avoidable destructive calls;
- retries after confirmed success;
- use of irrelevant tools.

A task can succeed but still be unsafe/inefficient.

## 11. Agent recovery benchmark

Fault-heavy curricula should measure:
- detection of failure;
- ambiguity handling;
- state re-check;
- idempotent retry;
- avoiding duplicate effects;
- escalation/abort when uncertainty remains.

This is a major value proposition of a stateful deterministic world.

## 12. Cross-host fairness

Curriculum difficulty should be world-defined, but host capabilities differ.

Therefore comparisons must record:
- built-in tools;
- approvals;
- surface projection;
- context/tool budget;
- model identity.

No raw leaderboard across incompatible Host Profiles.

## 13. Held-out curriculum generation

Scenario families MAY generate held-out instances via:
- deterministic seed;
- private seed selection;
- parameter partitions.

Generator version + seed are environment identity.

## 14. Capability uplift evidence

Training results may report:

```text
pre-coaching success
post-coaching success
number of attempts
failure categories reduced
new scenario generalization
```

Do not claim model-weight improvement unless weights actually changed.

The project improves the **agent system's evaluated behavior and practice environment**; it may not alter the base model.

## 15. Feasibility

**High** after deterministic scenarios/faults/evidence are stable.

This layer is primarily orchestration and evaluation methodology on top of the core world.
