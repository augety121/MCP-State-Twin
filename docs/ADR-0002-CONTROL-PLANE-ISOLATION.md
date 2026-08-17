# ADR-0002: Separate Agent Data Plane from Simulation Control Plane

- **Status:** Accepted for v0.1
- **Date:** 2026-08-17

## Context

A stateful simulation needs privileged operations such as snapshot, fork, reset, state inspection, virtual-time advance, and fault injection. If those operations are exposed as normal MCP tools to the tested agent, the agent can accidentally or deliberately modify the benchmark environment, inspect hidden answers, or reset failures.

This is both a security issue and an evaluation-validity issue.

## Decision

The project uses two physically/logically distinct interfaces:

### Agent data plane

Exposes only simulated application tools that correspond to the upstream agent-facing surface.

### Simulation control plane

Exposes privileged lifecycle and test operations to the test harness/CI operator.

Control operations are never registered in the default agent-facing `tools/list`.

Remote control-plane exposure requires independent authentication and authorization. Local development defaults to loopback/private transport.

## Hard consequences

- Scenario assertions and hidden expected state are control-plane data.
- The model cannot create, delete, reset or inspect arbitrary branches unless a test explicitly makes such a capability part of the simulated business domain.
- Fault schedules are installed by the runner, not by the agent under test.
- Data-plane server instructions must not mention hidden control capabilities.

## Rejected alternative

A single MCP server with administrative tools hidden only by prompt instructions is rejected. Tool visibility and authorization must be enforced in code; model compliance is not a security boundary.

## Verification

Release tests must assert that:

1. agent `tools/list` contains no control functions;
2. data-plane credentials cannot call control endpoints;
3. control-plane state changes are audited;
4. hidden scenario assertions are absent from agent-visible resources/results.