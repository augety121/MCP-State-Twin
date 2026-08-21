# MCP State Twin Project Map

**Status:** current product and architecture map
**Authority:** subordinate to accepted ADRs and RFC-0002; implementation claims
come only from `IMPLEMENTATION-STATUS.md` and executable tests.

## 1. One-sentence definition

MCP State Twin is a hermetic, deterministic, stateful MCP environment for
testing and comparing AI-agent tool behavior without writing to production
systems.

The project owns the **world model used for an evaluation**. It does not own
the model, the agent loop, or the real upstream service.

## 2. The problem

Agent evaluations are difficult to trust when every run shares a live service:

- the starting state changes between runs;
- one model can leave data behind for another model;
- retries can create real tickets, branches, or messages;
- a tool response can look valid while the hidden state is wrong;
- a text judge can disagree with the actual terminal state; and
- a benchmark can report model behavior without preserving enough evidence to
  reproduce it.

State Twin addresses the environment and evidence layer. It does not claim to
solve model intelligence, planning quality, or general agent alignment.

## 3. Product primitives

| Primitive | Responsibility | Current boundary |
|---|---|---|
| TwinSpec | Declarative entities, tools, schemas, transitions, invariants | Bounded YAML; no arbitrary scripts |
| State engine | Canonical state, preconditions, effects, postconditions | SQLite local profile |
| Snapshot/fork | Immutable starting point and isolated episode branches | Logical snapshots; no distributed storage claim |
| MCP data plane | Model-visible business tools | Tools-first, loopback-oriented development profile |
| Control plane | Snapshot, fork, reset, diff, clock and fault controls | Private and separately authenticated; never agent-facing |
| Scenario runner | Ordered calls, assertions, expected errors, report digest | Scripted deterministic scenarios; no live-model score claim |
| Evidence layer | Canonical digests, audit records, protocol evidence and status | Partial preview; recorder/bundle/OTel remain open |
| Release governance | SemVer gates, CI, changelog and evidence inventory | Development preview; no stable `v0.1.0` yet |

## 4. End-to-end lifecycle

```text
TwinSpec
  -> validate and bind tool surface
  -> initialize canonical state
  -> create immutable snapshot S0
  -> fork one isolated branch per episode/agent
  -> expose only business tools through MCP data plane
  -> apply atomic, bounded state transitions
  -> assert invariants and expected outcomes
  -> compute canonical terminal diff/report
  -> preserve evidence and update implementation status
  -> release only when the release profile gates pass
```

The comparison target is normally **terminal state plus declared assertions**,
not identical trajectories. Two agents may take different valid paths and still
reach the same acceptable world state.

## 5. Trust boundaries

```text
Agent / host
     |
     | MCP data plane: modeled business tools only
     v
State Twin runtime  ---- SQLite state and audit
     ^
     | private authenticated control plane
Harness / maintainer
```

The following are never exposed as agent-facing tools: snapshot, fork, reset,
state inspection, diff, fault configuration, clock control, and expected-answer
controls. This is an integrity boundary, not a prompt convention.

Hermetic mode has no production passthrough. If behavior is not modeled, the
runtime returns an explicit error; it must not invent a successful response.

## 6. Compatibility model

The compatibility claim is layered:

1. **Wire compatibility:** the MCP transport and message profile used by the
   server.
2. **Surface compatibility:** the canonical tool names, schemas and annotations
   admitted by the Twin.
3. **Host compatibility:** a specific ChatGPT, Claude, Codex, SDK or custom host
   can actually connect and render/use that surface.
4. **Behavioral compatibility:** the host/model produces a useful trajectory in
   a defined scenario.

The repository currently has evidence for the first two layers only. It must
not advertise “ChatGPT compatible”, “Claude compatible”, or “all agents
compatible” without a versioned host profile and an executable smoke test.

## 7. Why this is an AGI-facing foundation without claiming AGI

Long-running agents need a reliable external-world substrate: explicit state,
safe tool effects, reproducible failures, isolated branches, and evidence that
can survive model changes. State Twin supplies part of that substrate.

It is therefore reasonable to describe the project as **AGI-facing evaluation
infrastructure** or a **foundation layer for tool-using agents**. It is not
reasonable to call the repository an AGI system, a world model for the real
world, or a general-purpose autonomous agent runtime.

## 8. Risk model

The project handles “all possibilities” through bounded engineering controls,
not an impossible promise of enumerating every future failure:

```text
threat model
  -> hard invariants
  -> failure taxonomy
  -> fail-closed semantics
  -> negative/property/fuzz tests
  -> CI evidence
  -> incident-to-regression workflow
```

The detailed catalog is in `FAILURE-MODE-MATRIX.md` and the lifecycle proposal
pack. A catalog entry is not considered covered until it has executable evidence
and appears in `IMPLEMENTATION-STATUS.md`.

## 9. Maturity levels

| Level | Meaning | Current status |
|---|---|---|
| L0 | Cassette replay of previously observed calls | Not implemented |
| L1 | Stateful template Twin with new valid trajectories | Current development profile (`unverified`, `unbound`) |
| L2 | Contract-backed Twin with differential evidence and reviewed invariants | Planned / blocked |
| L3 | Native/reference implementation sharing domain semantics | Future; not claimed |

Automatic generation can produce a draft Twin, but it cannot promote a Twin to
L2 without human review, differential tests and coverage evidence.

## 10. Decision rule for future work

Every new capability must answer all five questions before implementation:

1. What exact user or maintainer problem does it solve?
2. Which boundary owns it: Twin, host, agent, or upstream connector?
3. What is the smallest accepted semantic subset?
4. Which invalid, adversarial and failure paths are tested?
5. What evidence permits a README or release claim?

If any answer is missing, the item stays proposal/blocked rather than silently
becoming part of the product.
