# SPEC-0007 — Virtual Time, Entropy & Deterministic Scheduler

> **Status:** Proposal  
> **Implementation baseline:** Partial foundation only; virtual-clock advancement/scheduler are not treated as verified.  
> **Verification:** Unverified

## Goal

Make simulator-controlled time, generated identifiers and modeled randomness reproducible.

This SPEC does **not** make the model deterministic.

## Time domains

- `host_wall_time`: operations/evidence metadata only.
- `world_time`: virtual time visible to modeled behavior.
- `logical_tick`: deterministic monotonic event ordering.
- `duration`: virtual elapsed duration.
- `evidence_time`: real time, excluded from deterministic state identity unless explicitly declared.

## Requirements

### ST-VTIME-R001
Modeled state transitions MUST NOT read host wall time.

### ST-VTIME-R002
World time changes only by accepted semantics:
- control-plane advance;
- transition-defined advance;
- scheduler event.

### ST-VTIME-R003
Clock controls MUST never be agent-visible business tools.

### ST-VTIME-R004
Control plane SHOULD support:
- read;
- advance by duration;
- advance to time;
- process due events;
- optionally jump to next due event.

### ST-VTIME-R005
Clock advance + due-event execution mode MUST be explicit and atomic at the declared transaction boundary.

### ST-VTIME-R006
Equal-time events MUST use stable tie-break ordering:

```text
(due_time, priority, creation_sequence, event_id)
```

Never OS scheduler/goroutine/map iteration.

### ST-VTIME-R007
Scheduled IDs are deterministic allocations or explicit state.

### ST-VTIME-R008
Modeled randomness MUST identify PRNG algorithm + seed.

### ST-VTIME-R009
Changing PRNG algorithm changes environment identity.

### ST-VTIME-R010
Modeled UUIDs/random IDs MUST come from deterministic allocation/entropy. Security credentials remain outside deterministic modeled data.

### ST-VTIME-R011
Canonical world timestamps SHOULD use UTC unless domain semantics require a zone ID.

### ST-VTIME-R012
DST/precision/rounding/overflow boundaries require golden tests if exposed.

### ST-VTIME-R013
Scheduled-event cancellation is itself a state transition.

### ST-VTIME-R014
Scheduled effects obey the same postcondition/invariant/transaction rules as tool effects.

## Snapshot additions

Snapshot SHOULD bind:
- `world_time`;
- scheduler queue/state;
- scheduler semantics version;
- PRNG algorithm;
- PRNG state/seed.

## Failure semantics

- backwards advance: typed invalid control input unless a future reversible-clock profile exists;
- overflow: typed limit/input error;
- event invariant failure: rollback event transaction;
- crash: governed by SPEC-0012.

## Test matrix

- two branches, same schedule -> same digests;
- equal timestamp ordering;
- many events same timestamp;
- event cancellation;
- event creates another event;
- event deletes target entity;
- fault + event interaction;
- scheduler crash killpoints;
- timezone/DST corpus where relevant;
- PRNG golden corpus.

## Feasibility

**High.** Main failure risk is accidental host-time/randomness leakage.
