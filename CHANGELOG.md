# Changelog

All notable changes will be documented in this file. The project has a public
alpha prerelease but no stable release yet.

## Unreleased

### Added

- a synthetic package-registry reference Twin and release-lifecycle Scenario;
- ADR-0014 recording the second-domain decision and its non-claims;
- versioned local resource-governance profile and `statetwin limits` command;
- fail-closed `RESOURCE_LIMIT` enforcement for JSON/state/query/diff/report and
  branch/snapshot budgets;
- deterministic fault preview and resource-profile ADR/SPEC evidence;
- maintainer evidence ledger, release operations checklist, and Dependabot
  configuration;
- project map and documentation-governance rules separating accepted contracts
  from the vendored vNext proposal pack;
- pull-request/issue templates and a tag-driven release workflow that reruns
  gates, builds multi-platform binaries, and publishes SHA256 checksums;
- strict TwinSpec `v1alpha1` YAML decoding and validation;
- canonical JSON digests for specs and state;
- bounded CEL expression compilation;
- SQLite-backed atomic state transitions and append-only tool-call audit;
- immutable snapshots, isolated forks, reset, and canonical state diff;
- MCP `2026-07-28` stateless Streamable HTTP data plane through the official
  Go SDK;
- separately authenticated HTTP control plane;
- issue-tracker reference TwinSpec and synthetic fixture;
- CLI commands for validation, initialization, calls, state, snapshots, forks,
  diffs, and serving;
- unit and MCP HTTP integration tests.
- strict shared YAML admission that rejects aliases, anchors, explicit tags,
  multiple documents, and unknown fields;
- bounded Scenario v1alpha1 execution with deterministic evidence reports,
  expected error classes, JSON Pointer state assertions, and a reference
  issue-closing scenario.
- proposed host-compatibility and cross-model evaluation evidence contract in
  SPEC-0006; this does not claim live provider support.
