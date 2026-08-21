# ADR-0014: Package Registry as the Second Reference Domain

- **Status:** Accepted for the development preview
- **Date:** 2026-08-21
- **Decision owners:** repository maintainers
- **Scope:** examples, scenario evidence and release-gate coverage only

## Context

The v0.1 release profile requires a second independent stateful reference
domain. A second domain is necessary to detect an abstraction that only works
for issue trackers, but it must not introduce a production connector, provider
dependency, trademark claim or unbounded simulation surface.

## Decision

Adopt a synthetic **package-registry** domain as the second reference domain.
The current TwinSpec models:

- package lookup and release listing;
- publishing a release;
- yanking a release without deleting history;
- installing a non-yanked release into a project; and
- querying advisories for a package/version pair.

The release-lifecycle Scenario covers successful transitions, terminal-state
assertions, an advisory result, and the negative path that rejects installation
of a yanked release.

## Boundaries

This decision does **not** claim:

- compatibility with npm, PyPI, crates.io, Maven, NuGet or another registry;
- semantic-version resolution, dependency solving, signatures or provenance;
- an upstream differential-fidelity result;
- network access or production package writes; or
- that the domain is a complete package-manager implementation.

The fixture is synthetic and remains `L1`, `unverified`, and `unbound`.

## Evidence and promotion

The executable evidence is:

- `examples/package-registry/twin.yaml`;
- `examples/package-registry/state.json`;
- `examples/package-registry/scenario-release-lifecycle.yaml`;
- local `statetwin scenario` execution; and
- the CI reference-domain smoke step.

The v0.1 gate is considered covered at the domain-selection level. Differential
validation, 20+ multi-step scenarios, and L2 fidelity remain separate gates.
