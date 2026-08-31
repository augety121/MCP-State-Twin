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
| DEC-011 | Deterministic scheduler/interleaving profile | open | SPEC-0007 / Phase 2 |
| DEC-012 | Recorder consent/redaction and incomplete-trace semantics | open | proposed SPEC-0009 / Phase 5 |
| DEC-013 | L2 admission coverage thresholds | open | proposed SPEC-0011 / Phase 5 |
| DEC-014 | TwinBundle signing and publisher identity | open | Phase 7 |
| DEC-015 | Supported v1.0 platform and deprecation windows | open | Phase 7 |

Open decisions are roadmap work. They MUST NOT be silently resolved by an
implementation-only change.
