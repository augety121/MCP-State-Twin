# ADR-0038: Independent AgentTask and Offline Grading

- Status: Accepted for the bounded offline subset below
- Date: 2026-09-08
- Basis: explicit maintainer request to implement the reviewed proposal
- Stable v0.1 impact: none; experimental authoring/validation commands only

## Decision

Accept B01 product direction and the B02/B03 offline subset in
[the reviewed plan](planning/agent-evaluation/PHASE-SPECS.md): independent
AgentTask alpha format, strict structural/surface admission, bounded declarative
read-only CEL grading, scalar resource authorization, six synthetic task/witness
pairs and a no-network MCP witness runner. The exact executable contract is
[SPEC-0038](SPEC-0038-AGENT-TASK-OFFLINE-ADMISSION.md).

Scenario, TwinBundle, scripted EpisodeEvidence and world/Journal formats keep
their existing meaning. AgentTask is a sidecar, never a new interpretation of
Scenario. The witness source MUST be `scripted-witness`, not a live model.

The offline runner uses the official MCP SDK and data-plane HTTP handler over
an in-process transport with a fixed route, no listener, DNS or HTTP fallback.
Its private grading view can inspect modeled effects; that view is not an
agent-visible tool result. Unknown, budget, privacy, and grading failures
cannot become success. Synthetically modeled response loss is not a claim of
real network failure injection.

## Explicit exclusions

This acceptance does not accept arbitrary provider calls, new live compatibility,
native remote deployment, full RunDefinition/ComparisonKey implementation,
autonomous AgentEpisode, captured trace sealing/replay, monetary governance,
automatic recovery, signed evidence or changes to stable release gates.
Those remain separate reviewed work packages. Existing terminal-Evidence-only
exactly-once boundaries remain unchanged.

Solvability is demonstrated only for the supplied witnesses, not decided for an
arbitrary natural-language task by a parser. The maintainer owns the final
semantic review of task/oracle pairs; generated tests are not independent audit.
