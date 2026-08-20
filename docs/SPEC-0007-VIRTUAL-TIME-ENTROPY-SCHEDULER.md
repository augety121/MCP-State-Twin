# SPEC-0007: Virtual Time, Entropy, and Scheduler Boundary

- **Status:** Proposed; clock-control subset implemented
- **Implementation status:** partial
- **Verification status:** clock advancement is tested; scheduler and entropy are not implemented
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

## 3. Not yet implemented

- deterministic PRNG/entropy profile;
- scheduled-event queue;
- equal-time tie ordering;
- event cancellation and cascades;
- scheduled effects and due-event processing;
- scheduler snapshot identity;
- deterministic fault/time interaction.

Until those components are implemented and tested, the project MUST NOT claim
deterministic scheduled workflows or deterministic model randomness.

## 4. Required future invariants

Future scheduler work MUST bind algorithm IDs, seeds, queue state, and semantic
limits into environment identity. Equal-time events MUST be ordered by
`(due_time, priority, creation_sequence, event_id)`, and every cascade MUST have
a bounded budget. A scheduler failure MUST roll back its world transition.
