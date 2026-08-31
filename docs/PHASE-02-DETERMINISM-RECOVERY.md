# Phase 2: Determinism and Recovery Completion

- **Target:** v0.2 development line
- **Entry:** stable local serial core
- **Purpose:** make every environment-affecting source explicit and replayable

## Required scope

- versioned VirtualClock with deterministic timer ordering;
- deterministic entropy streams and ID derivation;
- an explicit serial scheduler contract and equal-time tie-break rules;
- complete before-effect/after-effect fault distinction;
- cancellation and retry ordering;
- snapshot lineage validation and corruption refusal;
- migration interruption and process-crash recovery evidence;
- versioned ResourceProfiles covering every bounded subsystem;
- EnvironmentIdentity coverage for clock, entropy, scheduler and faults.

## Current accepted progress

ADR-0026 through ADR-0031 implement a bounded deterministic entropy profile,
branch-local signal scheduler, parsed-time total ordering, equal-instant
admission, cancellation, atomic due-signal delivery, explicit next-due legacy
drain and digest-bound inspection pages. These signals do not execute scheduled
tools or Agents. Full cascading effects, retry/dead-letter semantics and
remaining fault classes stay open.

ADR-0022's operational ExecutionProfile is separate from EnvironmentIdentity.
It may change wall-clock latency but MUST NOT change modeled outcomes. Any
future performance comparison must record it as host evidence.

## Safety rules

Wall clock, Go map order, goroutine completion order, process-global randomness
and database-generated IDs MUST NOT become simulation semantics. Operational
timestamps remain outside deterministic equality and invisible to TwinSpec.

## Exit evidence

- 100 identical runs produce identical result/event/state digests;
- supported Windows/Linux executions agree for canonical artifacts;
- every declared crash point reopens to an old-valid or new-valid state;
- timeout before effect and timeout after effect are observably different;
- configured and fired faults are separately recorded;
- limits fail closed without partial commits.
