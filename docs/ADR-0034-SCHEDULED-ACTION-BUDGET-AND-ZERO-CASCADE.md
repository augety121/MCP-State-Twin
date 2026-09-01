# ADR-0034: Cap Scheduled Actions and Enforce Zero Cascades

- **Status:** Accepted
- **Date:** 2026-09-01
- **Related:** ADR-0013, ADR-0022, ADR-0030, ADR-0032, ADR-0033, SPEC-0034

## Decision

Advance the semantic profile to `local-preview-v7`: at most 32 scheduled
actions and 256 total events execute in one step, each action has one attempt,
and cascade depth is zero. Batch selection is a contiguous total-order prefix.

The store hashes scheduler state before and after each runtime callback and
rolls back if the callback changed it. Only scheduler-owned terminalization is
allowed.

## Consequences

- action-heavy steps have a deterministic semantic work bound;
- equal-time action sets remain live through repeated explicit steps;
- tools cannot create hidden recursive work;
- automatic retry, recurrence, dead letters and compensation remain future
  policy work rather than accidental behavior.
