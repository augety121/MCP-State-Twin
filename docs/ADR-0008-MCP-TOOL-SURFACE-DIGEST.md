# ADR-0008: Canonical MCP tool-surface digest and startup binding

- **Status:** Accepted
- **Date:** 2026-08-17
- **Decision owners:** repository maintainers
- **Related:** RFC-0001 I-5, RFC-0002 §5, F-012, F-013

## Context

An MCP tool description is model-facing behavior. A change to a tool name,
description, input schema, output schema, or annotation can alter an agent's
trajectory even when the simulated transition code is unchanged. Hashing only
the TwinSpec file or only JSON Schemas would therefore miss relevant drift.

The development preview does not yet include an upstream MCP inspector. It
still needs a stable local representation so that a later inspector and CI
validator calculate the same value.

## Decision

`statetwin.dev/mcp-tool-surface/v1alpha1` is the canonical surface envelope.
It contains tools sorted by name. Each descriptor contains exactly:

- `name`;
- `description`;
- `inputSchema`;
- `outputSchema`, when declared;
- the effective `readOnly`, `destructive`, and `openWorld` annotations exposed
  by the MCP data plane.

Transition expressions, fixtures, fidelity metadata, upstream binding status,
and TwinSpec list order are excluded because they are not MCP tool descriptors.
Canonical JSON and SHA-256 follow ADR-0005.

The runtime applies these admission rules:

- `unbound`: start is allowed, but no upstream-equivalence claim is permitted;
- `current`: the declared digest MUST equal the computed TwinSpec surface;
- `drifted` or `unknown`: startup fails with `SPEC_DRIFT`;
- malformed digests are rejected during TwinSpec validation.

`statetwin validate` prints both `specDigest` and `surfaceDigest`. A future
inspector MUST use the same envelope and MUST NOT silently rewrite a reviewed
binding.

## Consequences

- Tool order and YAML formatting do not create false drift.
- Description-only and annotation-only changes do create drift.
- Changing a bound TwinSpec surface without updating the reviewed digest fails
  closed.
- This does not detect an upstream server change by itself. Until an inspector
  exists, upstream refresh remains a manual or external CI responsibility.

## Evidence

- order-independent surface digest tests;
- description, schema, and annotation mutation tests;
- matching `current` binding startup test;
- `current` mismatch plus `drifted`/`unknown` refusal tests;
- MCP server derives annotations from the same helper used by the digest.
