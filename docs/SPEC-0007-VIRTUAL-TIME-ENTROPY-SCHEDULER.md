# SPEC-0007: Virtual Time, Entropy, and Scheduler Boundary

- **Status:** Accepted bounded subsets; complete scheduled-effect runtime remains proposed
- **Implementation status:** partial
- **Verification status:** clock, deterministic entropy, signal queue and bounded due delivery are tested; scheduled effects are not implemented
- **Source:** `MCP-State-Twin-Lifecycle-SPEC-Pack-vNext/03-SPEC-0007...`

## 1. Boundary

State Twin has separate time domains:

- host wall time: operational metadata only;
- world time: virtual time visible to TwinSpec expressions;
- logical ordering: deterministic branch/head order;
- evidence time: non-deterministic report metadata.

TwinSpec behavior MUST NOT read host wall time. The current runtime exposes
world time to CEL as `clock`; the private control plane may advance it. Clock
control is never an agent-facing MCP tool.

## 2. Implemented clock profile

The local preview supports `POST /v1/clock/advance` with exactly one of:

```json
{"branch":"main","by":"1h","expectedHeadVersion":0}
```

or:

```json
{"branch":"main","to":"2026-08-02T00:00:00Z","expectedHeadVersion":1}
```

The operation is forward-only, limited by `store.MaxClockAdvance`, updates the
branch head atomically, and appends `clock.advance` control audit evidence.
Stale `expectedHeadVersion` returns `BRANCH_CONFLICT`; backwards or oversized
advances return `CLOCK_INVALID`.

## 3. Implemented bounded additions

ADR-0026 through ADR-0031 add:

- `sha256-ctr-v1` modeled-world entropy streams;
- branch-local `signal-queue-v1` events with deterministic lifecycle/order;
- atomic bounded due-signal delivery during private clock advancement;
- parsed UTC order and per-instant admission;
- bounded next-due preview/drain and digest-bound scheduler pages;
- scheduler/entropy limits in `local-preview-v6` and environment identity.

Signals are opaque private-harness records. Delivery does not execute a tool,
wake an Agent or create an external effect.

## 4. Not yet implemented

- cascading scheduled effects and recurring events;
- scheduled TwinSpec tool execution;
- deterministic fault/time interaction.

Until those components are implemented and tested, the project MUST NOT claim
deterministic scheduled workflows or deterministic model randomness.

## 5. Required future invariants

Future scheduled-effect work MUST preserve the accepted identity and ordering
contracts. Every cascade MUST have a separate bounded budget, and a scheduled
transition failure MUST define whether the event is terminal, retryable or
dead-lettered without inventing success.
