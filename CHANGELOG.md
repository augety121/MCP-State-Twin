# Changelog

All notable changes will be documented in this file. The project has a public
alpha prerelease but no stable release yet.

## Unreleased

### Added

- ADR-0021 and RFC-0001 revision 3, establishing the bounded product identity,
  independent version dimensions and a non-circular v0.1–v1.0 release train;
- SPEC-0019 through SPEC-0021 for exact HostProfiles, remote-staging security,
  claim admission and evidence freshness;
- Phase 0–7 contracts plus unified requirement, claim, compatibility and
  decision ledgers;
- an independent durable local Episode Journal with SQLite identity/schema,
  immutable request digests, transactional lifecycle CAS, terminal Evidence
  atomicity, idempotent completed-request reads, explicit incomplete records,
  tamper admission, and `episode inspect`;
- SPEC-0017, ADR-0019 and `local-preview-v3` with a 10,000-record Journal bound;
- ADR-0020, SPEC-0018 and `local-preview-v4`, adding a fenced single-coordinator
  Episode profile with 16-attempt and 3,600-second lease bounds;
- authenticated remote Episode submit/claim/heartbeat/complete/fail/cancel
  control APIs, a hermetic remote worker, cooperative cancellation, safe lease
  recovery, external `COMMIT_UNKNOWN`, and exactly-once terminal Evidence
  acceptance without claiming exactly-once external effects;
- Journal schema-v1 fixture migration to schema v2, interrupted-migration
  recovery, task/attempt lineage admission, and foreign SQLite zero-mutation
  refusal evidence;
- OpenAI Responses and Anthropic Messages provider-smoke adapters with current
  remote-MCP contract tests, bounded redacted reports, and an opt-in live-smoke
  workflow; no dated live provider report is claimed yet;
- ADR-0022 and SPEC-0022 with a conservative `quiet` local execution default,
  validated `balanced`/`throughput` modes, exact `--max-procs` override,
  environment precedence and machine-readable `execution-profile` output;
- ADR/SPEC-0023 through 0025: mode-bound Go heap soft limits, independent
  non-queueing HTTP listener admission with redacted `SERVER_BUSY`, and
  authenticated control-plane live/readiness checks;
- `local-v2` ExecutionProfile fields and overrides for memory MiB and maximum
  in-flight requests, with explicit soft-governor and non-production claims;

### Changed

- live OpenAI-family and Anthropic-family evidence is no longer a local-core
  v0.1 gate; it targets the v0.3 secure provider/host profile;
- OpenAI cancellation idempotency is reported as `unknown`: the documented
  cancel endpoint does not by itself prove repeated-cancel semantics;
- deterministic TwinBundle `v1alpha1` build/verify commands with strict
  manifest and payload admission, portable path handling, exact member
  digests, resource bounds, deterministic ZIP bytes, and tamper/negative tests;
- one-shot local scripted EvaluationEpisode execution with a closed lifecycle,
  declared-Scenario admission, runtime revision identity, canonical evidence
  digest, and fail-closed output handling;
- RFC-0003, ADR-0018, and a structured v0.2 requirement/evidence
  ledger that keep durable/remote/provider/signing work explicitly deferred;
- governance, maintainer, support, code-of-conduct, and CODEOWNERS records;
- a synthetic package-registry reference Twin and release-lifecycle Scenario;
- ADR-0014 recording the second-domain decision and its non-claims;
- ADR-0015 accepting the L1-only v0.1 profile and explicitly deferring
  recorder/L0 and L2/L3 fidelity work;
- a direct slow-header connection test for the production HTTP timeout profile;
- ADR-0016 plus synthetic schema-v3 and tagged alpha schema-v4 fixtures, with
  process-exit migration recovery at two pre-commit kill-points;
- ADR-0017 and strict HostCompatibilityReport admission through
  `statetwin compatibility validate`, including bounded trials, remote-profile
  binding, immutable digests, and credential/private-key/email pattern rejection;
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
