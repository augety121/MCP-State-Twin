# ADR-0006: Hermetic JSON Schema 2020-12 Validation

- **Status:** Accepted
- **Date:** 2026-08-17

## Context

The development preview originally validated only a top-level subset of tool
input schemas. That was insufficient for nested objects, arrays, numeric
bounds, combinators, formats, and output contracts. It could make a permissive
twin appear more faithful than its declared MCP surface.

Schema compilation must also remain hermetic. A `$ref` that downloads an
external resource during startup would add a hidden network and supply-chain
dependency.

## Decision

- Use `github.com/santhosh-tekuri/jsonschema/v6` pinned to `v6.0.3`.
- Default schemas without `$schema` to Draft 2020-12.
- Enable format assertions.
- Compile input and output schemas once when the runtime is constructed.
- Reject any schema resolution that requires an external loader.
- Validate input before effects.
- Validate successful output before commit.
- Map input violations to `INVALID_INPUT`.
- Map declared output violations to `INTERNAL_TWIN_ERROR` and roll back.

Local `$defs` and fragment references remain supported.

## Consequences

- Twin authors can use the full supported Draft 2020-12 vocabulary.
- Invalid output becomes a runtime contract failure, not fake success.
- Remote schema registries are not supported in v0.1.
- Updating the validator requires its upstream compliance evidence, local
  negative tests, and a dependency-review change.

## Evidence

- nested object and format assertion test;
- external `$ref` fail-closed test;
- invalid output rollback test;
- MCP integration tests using the same declared schemas.
