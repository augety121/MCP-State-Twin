# Unified Requirement Traceability

**Status:** executable mapping for accepted requirements
**Last reviewed:** 2026-08-31

| Requirement | Contract | Implementation/evidence | Current state |
|---|---|---|---|
| I-1 no hidden live writes | RFC-0001 | hermetic namespace job, source absence policy | verified local profile |
| I-2 control isolation | RFC-0001 / ADR-0002 | server discovery negative tests | verified |
| I-3 deterministic transition | RFC-0001 / SPEC-0002 | deterministic replay corpus | verified serial subset |
| I-4 explicit unknown | RFC-0001 / SPEC-0002 | engine unmodeled/error tests | verified modeled subset |
| I-5 surface binding | ADR-0008 | canonical tool-surface digest tests | verified local fingerprint |
| I-6 atomic transition | SPEC-0002 | rollback/invariant/schema tests | verified |
| I-7 branch isolation | SPEC-0002 | 100-fork test | verified |
| I-8 state oracle | SPEC-0004/0005 | Scenario state assertions/diff | verified scripted subset |
| I-9 secret exclusion | ADR-0009 / SPEC-0020 | sanitizer and provider-report tests | partial; remote profile open |
| I-10 error preservation | SPEC-0002/0018 | domain/internal/unknown tests | verified implemented paths |
| I-11 immutable Evidence | SPEC-0017/0018 | duplicate/conflict/tamper tests | experimental candidate |
| I-12 bounded execution | SPEC-0015 / SPEC-0022–0024 | semantic limit tests, ResourceProfile digest, CPU/heap governor tests, admission saturation/recovery tests and inspection smoke | accepted local soft-governor subset; no OS hard quota or distributed fairness |
| I-13 fencing | SPEC-0018 | stale-worker/concurrent-claim tests | experimental candidate |
| I-14 external ambiguity | SPEC-0018 | external expiry/failure tests | experimental candidate |
| I-15 scoped compatibility | SPEC-0019 | compatibility validator/matrix | partial; live evidence open |
| I-16 evidence-backed claims | SPEC-0021 | claim registry and release review | implemented as governance |
| ST-VTIME-001 forward-only virtual clock | ADR-0011 / SPEC-0007 | clock control/head/audit tests | verified bounded subset |
| ST-VTIME-002 deterministic modeled entropy | ADR-0026 / SPEC-0026 | spec/engine equal-draw, persistence and rollback tests | verified bounded subset |
| ST-VTIME-003 total signal ordering and lifecycle | ADR-0027 / SPEC-0027 | scheduler ordering/cancel/fork/control HTTP tests | verified bounded subset |
| ST-VTIME-004 atomic bounded due delivery | ADR-0028 / SPEC-0028 | delivery ordering and 257-event rollback tests | verified bounded subset |
| ST-VTIME-005 parsed-time order and per-instant admission | ADR-0029 / SPEC-0029 | sub-second order, capacity and cancellation-release tests | verified bounded subset |
| ST-VTIME-006 bounded next-due recovery | ADR-0030 / SPEC-0030 | preview, two-step legacy drain, empty/CAS and HTTP tests | verified bounded subset |
| ST-VTIME-007 consistent bounded inspection | ADR-0031 / SPEC-0031 | filter/page/stale/malformed cursor tests | verified bounded subset |

## Release use

A release gate is closed only when the evidence passes on the exact release
candidate revision. `Partial`, `experimental` and `unverified` rows may ship
only when excluded from the stable profile and called out in release notes.
