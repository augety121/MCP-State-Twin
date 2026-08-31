# ADR-0013 — Versioned resource governance profile

- **Status:** Accepted
- **Date:** 2026-08-21
- **Scope:** development-preview profile; stable release claims remain narrower

## Context

Resource limits were previously spread across packages as independent
constants. That made it possible for an evaluation report to omit a semantic
limit, or for a new caller to bypass a bound. SPEC-0015 requires limits to be
typed, applied before unbounded work where practical, and included in
environment identity when they can change outcomes.

## Decision

The repository defines one deterministic
`statetwin.dev/resource-profile/v1alpha1` profile in
`internal/limits`. Its digest is exposed by `statetwin limits` and included
in Scenario `EnvironmentIdentity`.

The accepted local limits include:

- TwinSpec/tool/schema/expression budgets;
- JSON depth/member/byte budgets;
- input, output, state, audit, report, and diff byte budgets;
- effect and query-result counts;
- entity records, branches, snapshots, fault plans, and concurrent-call
  bounds.

A zero value in the profile means the corresponding future feature is
disabled, not unlimited. Cassette and future-task limits remain zero while
those features are not implemented.

## Enforcement boundary

- TwinSpec admission rejects oversized schemas and effects.
- Runtime input and output are checked before commit.
- World state is checked at initialization, after every transition, and by the
  storage boundary.
- Diff generation fails closed when entry or encoded-byte limits are exceeded.
- SQLite branch/snapshot/fault-plan counts are bounded transactionally.
- Scenario reports include the profile digest and are rejected if oversized.
- Resource exhaustion uses the typed `RESOURCE_LIMIT` class and is not
  converted into a modeled business error.

## Non-claims

This ADR does not claim OS-level memory isolation, CPU quotas, distributed
multi-tenant fairness, scheduler limits, bundle import safety, or cassette
limits. Those require the corresponding feature to exist and independent
acceptance evidence.

## Amendment: ADR-0018 (2026-08-26)

ADR-0018 advances the profile identifier to `local-preview-v2` and enables
only deterministic local TwinBundle file-count, compressed-byte,
extracted-byte, and per-member limits. The remaining non-claims above are
unchanged; bundle signatures, publisher identity, registries, and remote import
remain outside the accepted boundary.

## Amendment: ADR-0019 (2026-08-26)

ADR-0019 advances the profile identifier to `local-preview-v3` and enables a
10,000-record bound for the independent local Episode Journal. It does not
enable remote task quotas, scheduling, retention automation or multi-tenant
fairness.

## Amendment: ADR-0020 (2026-08-26)

ADR-0020 advances the profile identifier to `local-preview-v4`, caps one
Episode task at 16 attempts, and caps a coordinator lease at 3,600 seconds.
These bounds cover the accepted single-coordinator preview only. They do not
establish distributed quotas, tenant fairness, retention, HA, or external
effect retry safety.

## Related operational decisions: ADR-0022 through ADR-0024 (2026-08-31)

The separate `ExecutionProfile` adds a default one-slot Go runtime policy, Go
heap soft target and bounded per-listener admission. It is intentionally not
folded into this semantic ResourceProfile: operational settings may affect
latency while modeled outputs and environment identity must remain unchanged.
OS/RSS hard quotas and distributed fairness remain non-claims.

## Amendment: ADR-0026 through ADR-0028 (2026-08-31)

The deterministic-world profile advances to `local-preview-v5`. It enables 64
entropy streams per branch, 32 bytes per draw, 1,024 retained scheduler events
per branch and 256 due-signal deliveries per clock advance. These are semantic
bounds and change Scenario environment identity. They do not enable scheduled
tool effects, Agent wakeup, recurring timers, retention/GC or a distributed
queue.

## Amendment: ADR-0029 through ADR-0031 (2026-09-01)

The deterministic-world profile advances to `local-preview-v6`. It adds a
256-pending-event per-instant admission bound, scheduler inspection pages of
100 by default and 256 at most, and a 2,048-byte cursor-input bound. Parsed-time
ordering and explicit bounded next-due drain close scheduler correctness and
liveness gaps; they do not authorize scheduled business effects.
