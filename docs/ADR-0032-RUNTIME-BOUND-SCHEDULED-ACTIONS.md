# ADR-0032: Bind Scheduled Actions to the Loaded TwinSpec Runtime

- **Status:** Accepted
- **Date:** 2026-09-01
- **Related:** ADR-0002, ADR-0027 through ADR-0031, SPEC-0032

## Decision

Add private `tool-call` scheduled events under `deterministic-queue-v2`. The
control plane admits them only when backed by the exact loaded runtime, validates
the existing modeled tool and input schema, and stores the runtime spec digest.

Actions execute only through bounded `advance-next`. Ordinary arbitrary clock
advance fails closed when an action is due. No scheduling control is exposed as
an Agent MCP tool.

## Consequences

- future world actions reuse TwinSpec semantics instead of inventing a second
  transition language;
- a branch cannot execute an action through a different spec revision;
- store-only harnesses continue to support signals but reject actions;
- the scheduler policy advances from signal-only `signal-queue-v1` to
  `deterministic-queue-v2`;
- provider calls, Agents and external writes remain outside this decision.
