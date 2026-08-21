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
disabled, not unlimited. Bundle, cassette, scheduler-event, and future-task
limits are therefore zero while those features are not implemented.

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
