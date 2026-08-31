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
