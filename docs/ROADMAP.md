# MCP State Twin Roadmap

**Status:** planned work ordered by accepted lifecycle boundaries
**Authority:** ADR-0021 and the linked Phase SPECs
**Current release:** development preview; latest public prerelease
`v0.1.0-alpha.1`

MCP State Twin remains a deterministic, forkable, evidence-backed stateful
environment for testing tool-using agents. The roadmap does not turn planned
features into implementation claims; `IMPLEMENTATION-STATUS.md` and
`CLAIM-REGISTRY.md` control those claims.

## Release train

| Product line | Primary contract | Scope |
|---|---|---|
| `v0.1.x` | [Phase 1](PHASE-01-LOCAL-CORE.md) | local hermetic deterministic core |
| `v0.2.x` | [Phase 2](PHASE-02-DETERMINISM-RECOVERY.md), [Phase 3](PHASE-03-REMOTE-EXECUTION.md) | determinism/recovery completion and fenced remote Episodes |
| `v0.3.x` | [Phase 4](PHASE-04-PROVIDER-VALIDATION.md) | secure remote staging and exact provider/host evidence |
| `v0.4.x` | [Phase 5](PHASE-05-FIDELITY.md) | recorder/replay, drift, differential L2 |
| post-v0.4 | [Phase 6](PHASE-06-SCENARIO-FAMILIES.md) | scenario families and bounded multi-Agent evaluation |
| `v1.0.0` | [Phase 7](PHASE-07-V1-STABILITY.md) | stable formats, compatibility and maintainer contract |

## Phase 0 — specification and claim consolidation

Current progress: ADR-0021 through ADR-0034, SPEC-0019 through SPEC-0034,
unified traceability, claim, compatibility and decision ledgers, phase
contracts, conservative local ExecutionProfile, HTTP admission, and private
health/readiness exist. Exit still requires final link/translation review and
CI on the merged revision.

## Phase 1 — local core

Current progress: strict TwinSpec/CEL/JSON Schema admission, canonical digests,
SQLite atomic transitions, snapshots/forks/reset/diff, control isolation,
Scenario, TwinBundle, local Episode, two synthetic reference domains and direct
MCP wire tests are implemented. The exact candidate must pass every Phase 1
gate before a stable v0.1 tag.

Provider live evidence is explicitly not a v0.1 gate. The local release neither
claims nor requires Internet-ready data-plane security.

## Phase 2 — determinism and recovery completion

Partial preview capability exists for monotonic branch heads, a private virtual
clock, `sha256-ctr-v1` modeled entropy, a bounded branch-local deterministic
queue, atomic ordered due delivery, digest-bound inspection, runtime-bound
one-attempt local TwinSpec actions, two transaction fault phases and versioned
resource limits. Each action step is capped at 32 actions/256 total events and
zero cascade. Scheduled Agent/provider/external effects, automatic retry,
recurrence/dead letters, non-zero cascades, the full fault taxonomy and
cross-platform deterministic artifact evidence remain open.

The accepted operational-safety subset now includes a soft Go heap target and
bounded per-listener HTTP admission. Hard OS quotas, durable overload queues,
distributed fairness and empirical performance budgets remain open and require
separate platform evidence.

## Phase 3 — durable remote Episodes

Candidate implementation includes Journal schema v2, leases, heartbeat,
fencing, bounded hermetic recovery, cooperative cancellation, remote workers,
`COMMIT_UNKNOWN` and exactly-once terminal Evidence acceptance. It remains an
experimental candidate until merged and evidenced by CI. Multi-coordinator HA,
replication, retention and external-effect exactly-once remain excluded.

## Phase 4 — provider/host validation

OpenAI Responses and Anthropic Messages adapters have mock contract tests and
an opt-in harness. There are no dated admitted live reports. The complete
remote-staging security profile in SPEC-0020 is a prerequisite. ChatGPT, Codex,
Claude and Claude Code remain separate unverified product profiles.

## Phase 5 — fidelity

Recorder, cassette replay, redaction, upstream inspection, drift automation,
differential validation and L2 admission are not implemented. The current
reference twins remain `L1`, `unverified` and `unbound`.

## Phase 6 — scenario families

Issue-tracker and package-registry scenarios exist. Additional scenario
families, deterministic generation, held-out evaluation, metamorphic coverage
and shared-world multi-Agent scheduling require separate evidence.

## Phase 7 — v1 stability

v1.0 freezes only contracts that have survived preview releases and have real
consumers. Cloud hosting, marketplaces, general A2A orchestration, arbitrary
native plugins and production mirroring remain independent RFCs.

## Priority order

1. close Phase 0 and run the exact-candidate Phase 1 gates;
2. publish a narrow v0.1 local core instead of waiting for unsafe provider
   validation;
3. stabilize Phase 2 recovery and Phase 3 remote Episode semantics;
4. implement SPEC-0020 before collecting or claiming live provider evidence;
5. add fidelity only after recorder privacy and consent semantics are reviewable.

The adoption strategy is evidence-first: useful reference scenarios, small
reproducible releases, public issues/PRs, and compatibility claims tied to
exact profiles. The project does not use an AGI capability claim as a release
or adoption metric.
