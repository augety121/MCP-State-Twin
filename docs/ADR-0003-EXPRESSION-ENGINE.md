# ADR-0003: Bounded CEL for TwinSpec Expressions

- **Status:** Accepted and implemented for the development preview
- **Date:** 2026-08-17

## Context

TwinSpec requires conditions, keys, value construction, queries, and
postconditions. Arbitrary embedded code would make deterministic execution,
static review, and supply-chain isolation substantially harder. A completely
new expression language would add parser and semantic risk.

## Decision

Use `cel-go` as a declarative expression engine with dynamic JSON-shaped
variables:

- `input`
- `state`
- `vars`
- `item`
- `clock`
- `call_index`

Programs are compiled when the TwinSpec is loaded. Each program has a CEL cost
limit of 10,000. No filesystem, network, process, reflection, or arbitrary Go
function access is registered.

## Consequences

- expressions are reviewable and deterministic for the supported value domain;
- CEL syntax becomes part of the alpha TwinSpec contract;
- dynamic values defer some type errors until validation/runtime;
- custom domain logic is not available in `v1alpha1`;
- future native adapters require a separate trust model and ADR.
