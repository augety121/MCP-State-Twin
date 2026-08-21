# Forward Architecture — Multi-Agent, Long-Running Agents & Future AGI Hosts

> **Status:** Non-normative forward architecture / RFC candidate registry  
> **Implementation:** None implied  
> **Purpose:** Keep the project adaptable without pretending to know future AGI architecture.

## 1. Forward-looking position

MCP State Twin should not attempt to predict the internals of future AGI.

It should provide a durable external-world primitive with properties that remain useful as agents improve:

- explicit state;
- explicit effects;
- explicit permissions;
- deterministic external-world behavior;
- forkable starts;
- typed failure;
- replayable evidence;
- host-neutral contracts;
- capability/version negotiation.

This is an **AGI-facing infrastructure goal**, not an AGI capability claim.

## 2. No hidden-reasoning dependency

Normative evaluation MUST NOT require:
- chain-of-thought;
- hidden reasoning tokens;
- provider-private planner state;
- exact internal memory representation;
- exact subagent reasoning graph.

Only externally observable contract events may be required for verification.

## 3. Agent evolution axes

The architecture should tolerate:

- larger context windows;
- context compression;
- persistent memory;
- deferred tool discovery;
- tool search;
- programmatic tool calling;
- multiple tool calls per turn;
- parallel calls;
- subagents;
- long-running goals;
- human approval/input;
- asynchronous tasks;
- multimodal tool outputs;
- richer protocol extensions;
- changing model providers.

These should first change Host Profiles or optional extensions, not TwinSpec core.

## 4. Shared-world multi-agent modes

### Mode A — Isolated comparison

```text
Snapshot S0
  -> Agent A fork
  -> Agent B fork
  -> Agent C fork
```

Preferred for model comparison.

### Mode B — Cooperative shared branch

Multiple agents share one mutable world.

Requires:
- branch-head versioning;
- explicit conflicts;
- world commit sequence;
- agent attribution.

### Mode C — Adversarial shared branch

Used for:
- race-condition tests;
- permission conflicts;
- adversarial coordination;
- security testing.

### Mode D — Hierarchical

Parent agent delegates to subagents.

Harness MAY expose:
- `agent_id`;
- `parent_agent_id`;
- delegation metadata.

The mode is part of environment/evaluation identity.

## 5. Agent scheduler vs world scheduler

This distinction is foundational.

```text
Agent scheduler
= host/model decides which agent/tool action occurs next

World scheduler
= State Twin processes deterministic virtual-time events
```

State Twin owns only the second.

## 6. Concurrency replay

A shared-world run may be nondeterministic because hosts race calls.

For reproducibility, archive the actual:

```text
world_commit_sequence
```

A deterministic world replay can then reproduce the same interleaving.

Do not claim the agents will naturally race identically next time.

## 7. Long-running tasks

If a domain needs asynchronous operations:
- prefer the then-current MCP Tasks extension or equivalent explicit capability;
- durable task handle is explicit world state;
- task creation durability precedes successful handle return;
- task state transitions are modeled;
- virtual time drives deterministic timeout/expiry;
- cancellation races are specified.

No Tasks dependency belongs in core unless concrete use cases justify it.

## 8. Interactive server requests / MRTR

If future scenarios require server-side input from a client/human:
- capability-gate the behavior;
- define timeout/cancellation;
- record host/user response as evidence;
- avoid hidden out-of-band state.

Do not fake MRTR when the host cannot support it.

## 9. Large tool surfaces

Future agents may face thousands of tools.

State Twin should preserve:
1. canonical full server surface;
2. authorization-scoped server surface;
3. host-visible filtered/deferred surface;
4. selected/called surface.

A host's tool-search optimization does not change the business meaning of hidden tools.

## 10. Self-modifying agents

Future agents may generate:
- TwinSpecs;
- schemas;
- effects;
- tests;
- extension code.

Generated artifacts enter as **candidate**.

Admission sequence:

```text
generate
 -> static validation
 -> security/limit validation
 -> human semantic review
 -> executable tests
 -> evidence
 -> optional acceptance
```

No self-approval.

## 11. Extension runtime

Preferred evolution path:

```text
declarative TwinSpec
 -> new reviewed built-in primitive
 -> versioned isolated extension protocol
 -> sandboxed WASM/capability runtime only after separate RFC
```

Arbitrary native Go plugins should remain unsupported by default.

## 12. Computer-use boundary

Current TwinSpec is a tool-world model.

It does not imply fidelity for:
- browser pixels;
- DOM;
- desktop GUI;
- mobile devices;
- operating system;
- robotics;
- physical environment.

Future computer-use/robotics twins need separate state/observation/action/fidelity contracts.

## 13. Multimodal outputs

If tools later return:
- images;
- audio;
- files;
- binary artifacts;

the spec needs:
- content identity/digest;
- MIME/type;
- size limits;
- storage/reference semantics;
- redaction;
- host-transform evidence.

Do not encode arbitrarily large binary data into canonical JSON state by default.

## 14. Persistent agents

A host may persist an agent beyond one evaluation.

Evaluation must still define a reset boundary:
- model/host memory reset or declared persistent;
- State Twin world reset/fork;
- host cache state;
- MCP config state.

“Fresh world” does not guarantee “fresh model memory” unless the host profile establishes it.

## 15. Cross-session learning

If a future host learns across runs, evaluation profiles must explicitly mark:

```yaml
agent_memory:
  reset_guaranteed: true|false|unknown
```

Unknown host memory means repeated trials are not fully independent.

## 16. Future protocol independence

MCP is the current protocol integration boundary.

Long-term world runtime architecture should still expose an internal semantic interface so another future agent protocol could be adapted without rewriting state semantics.

This is a design goal, not a commitment to implement a second protocol.

## 17. Future RFC candidate registry

Do not allocate new SPEC numbers yet for:

- MCP Tasks integration;
- advanced MRTR scenarios;
- MCP Apps/UI evaluation;
- multi-tenant remote service;
- distributed storage;
- registry/marketplace;
- sandboxed extension runtime;
- browser/computer-use world adapter;
- robotics world adapter;
- cross-protocol adapter;
- benchmark marketplace.

Promote only after a real use case and feasibility RFC.

## 18. Long-term success criterion

MCP State Twin remains future-relevant if:

> A materially more capable agent can be integrated by changing a Host Profile/adapter while the external-world semantics, determinism and evidence contracts remain valid.

If every model generation requires changes to core TwinSpec, provider neutrality has failed.

## 19. What this document does not claim

It does not claim:
- AGI exists;
- MCP will be the final agent protocol;
- all future agent behaviors are predictable;
- State Twin is a general intelligence benchmark;
- deterministic external worlds make agents deterministic.

It defines robust engineering boundaries under uncertainty.
