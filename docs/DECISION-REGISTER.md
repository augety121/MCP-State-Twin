# Decision Register

**Status:** maintained index of resolved and open cross-cutting decisions

| ID | Decision | State | Authority |
|---|---|---|---|
| DEC-001 | Product direction remains deterministic stateful MCP evaluation environments | resolved | RFC-0001 rev 3 / ADR-0021 |
| DEC-002 | v0.1 is local hermetic core; provider live evidence moves to v0.3 | resolved | ADR-0021 / RFC-0002 rev 2 |
| DEC-003 | Hermetic mode has no upstream passthrough | resolved | I-1 / AGENTS.md |
| DEC-004 | Control plane is not Agent-visible MCP | resolved | ADR-0002 / I-2 |
| DEC-005 | Exactly-once means terminal Evidence acceptance only | resolved | ADR-0020 / SPEC-0018 |
| DEC-006 | Remote ambiguous effects become `COMMIT_UNKNOWN` | resolved | ADR-0020 / SPEC-0018 |
| DEC-007 | API and product HostProfiles are separate | resolved | SPEC-0019 |
| DEC-008 | Provider mock tests do not satisfy live compatibility | resolved | SPEC-0019 / SPEC-0021 |
| DEC-009 | Complete remote-staging security controls | open | SPEC-0020 / Phase 4 |
| DEC-010 | Operator reconciliation contract for `COMMIT_UNKNOWN` | open | Phase 3 follow-up |
| DEC-011 | Full scheduled-effect/interleaving profile beyond opaque signals | open | SPEC-0007 / Phase 2 |
| DEC-012 | Recorder consent/redaction and incomplete-trace semantics | open | proposed SPEC-0009 / Phase 5 |
| DEC-013 | L2 admission coverage thresholds | open | proposed SPEC-0011 / Phase 5 |
| DEC-014 | TwinBundle signing and publisher identity | open | Phase 7 |
| DEC-015 | Supported v1.0 platform and deprecation windows | open | Phase 7 |
| DEC-016 | Local processes default to a one-slot soft Go runtime governor; OS hard quotas remain a separate capability | resolved | ADR-0022 / SPEC-0022 |
| DEC-017 | Go heap pressure uses a documented soft runtime target, never an RSS/hard-quota claim | resolved | ADR-0023 / SPEC-0023 |
| DEC-018 | HTTP overload is rejected immediately per listener rather than queued without bound | resolved | ADR-0024 / SPEC-0024 |
| DEC-019 | Health/readiness remain authenticated private control routes and never Agent tools | resolved | ADR-0025 / SPEC-0025 |
| DEC-020 | Modeled entropy uses explicit public seed plus `sha256-ctr-v1`; never host RNG or security entropy | resolved | ADR-0026 / SPEC-0026 |
| DEC-021 | Virtual scheduler is a bounded branch-local private signal queue, not an Agent/workflow scheduler | resolved | ADR-0027 / SPEC-0027 |
| DEC-022 | Clock advance and all bounded due-signal lifecycle changes commit atomically; over-budget batches fail whole | resolved | ADR-0028 / SPEC-0028 |
| DEC-023 | Scheduler order uses parsed UTC time; new equal-time pending sets are capped at one delivery batch | resolved | ADR-0029 / SPEC-0029 |
| DEC-024 | Ordinary clock jumps remain atomic; an explicit advance-next operation provides bounded legacy drain | resolved | ADR-0030 / SPEC-0030 |
| DEC-025 | Scheduler inspection uses bounded digest-bound cursor pages | resolved | ADR-0031 / SPEC-0031 |

Open decisions are roadmap work. They MUST NOT be silently resolved by an
implementation-only change.
