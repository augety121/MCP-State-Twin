# Changelog

All notable changes will be documented in this file. The project does not yet
have a tagged release.

## Unreleased

### Added

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
