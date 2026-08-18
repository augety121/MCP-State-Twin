# ADR-0010: Bounded scenario artifacts and evidence reports

- **Status:** Accepted and implemented for the development preview
- **Date:** 2026-08-18
- **Related:** SPEC-0002, SPEC-0004, SPEC-0005, RFC-0002

## Decision

Add a strict `Scenario` artifact and a scripted runner before adding provider
SDK harnesses. Scenarios use declarative tool calls, expected canonical error
classes, and bounded JSON Pointer state assertions. They run in a fresh
in-memory store and emit a deterministic evidence report.

The runner does not expose control-plane operations to MCP, execute arbitrary
assertion code, contact an upstream service, or claim a scripted trajectory was
performed by an AI model. Provider identity is recorded separately from the
environment digest in future harnesses.

## Consequences

- reference twins can ship executable, state-scored examples;
- CI can detect semantic changes without using an LLM judge;
- scenario/report formats become compatibility surfaces and require an ADR for
  semantic changes;
- full traces may contain sensitive payloads, so v1alpha1 remains synthetic-
  fixture only and has a 16 MiB trace budget.

## Evidence

- strict parser tests cover unknown fields, aliases, tags, multiple documents,
  invalid pointers, and resource bounds;
- runner tests cover deterministic replay, expected modeled errors, unexpected
  error failure, state assertions, and state diff;
- the issue-tracker scenario runs as a CLI/CI smoke test.
