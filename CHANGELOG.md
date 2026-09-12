# Changelog

All notable changes will be documented in this file. The project has a public
alpha prerelease but no stable release yet.

## Unreleased

### Added

- ADR/SPEC-0045–0047: version-bound reviewed release plans/notes and read-only
  `cmd/releasecheck`; full same-commit reusable CI before draft staging, explicit
  prerelease/not-latest flags and contents:write restricted to the staging job.
- Replaced unconditional deletion of `dist` with no-clobber packaging admission,
  clean-source/exact-tag checks and serial builds. Failures preserve partial output.
  Added native policy/CLI, workflow structure and POSIX synthetic-command tests;
  no real release artifacts, tag, draft or stable qualification are created by this change.

- ADR/SPEC-0042–0044: evidence storage/terminal-failure contracts, read-only
  `eval inspect` and structured JSON credential admission. Tests cover 22 injected
  filesystem failures and five actual test-subprocess exits without filling a disk.
  Inspection checks claim/staging consistency and world replay; it never resumes,
  repairs, calls a Provider or proves provider origin. SQLite schemas are unchanged.
- Fixed canceled execution context leaking into terminal replay, secondary
  terminal/cleanup errors overwriting the first failure, and undetected short
  evidence writes. Optional terminal/cleanup failure fields may be rejected by
  older strict preview readers; historical artifacts are not rewritten.
- Fixed known credential fields bypassing privacy admission when encoded as
  JSON, including escaped keys and bounded embedded JSON. Full envelopes are
  checked before file creation; this remains finite-pattern policy, not universal DLP.

- ADR/SPEC-0040–0041: explicitly approved local Responses plans, fixed no-retry
  HTTPS transport, bounded unknown-cost receipts, a private-world API loop and
  separate live-kind world replay. New `eval live-plan`, `live-preflight`, `live`
  and `live-verify` commands; tests use contract doubles, not actual paid APIs.
  No new live/product compatibility claim, stable release or automatic recovery.

- ADR/SPEC-0039: experimental offline Responses codec and sequential mock Agent
  loop, per-trial budgets, private continuation, retained synthetic evidence with
  no-clobber publication and world replay, fixed-plan JSON/Markdown comparison,
  and an executable first-value regression guide. That lane remains purely offline.

- ADR/SPEC-0038: independent experimental AgentTask admission, bounded read-only
  grading, six synthetic task/witness pairs and `task validate` / `task witness`;
  witness playback traverses the official MCP SDK with an in-process HTTP
  transport and is explicitly not live-agent or sealed replay evidence;

- ADR/SPEC-0035–0037: bounded fuzz evidence, dependency migration admission,
  and CEL null/JSON value-boundary contracts;
- expression compatibility vectors, module/import migration checks and
  failure-propagation tests for the bounded fuzz wrapper;

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
- ADR/SPEC-0026 through 0028: `sha256-ctr-v1` modeled entropy, a private
  branch-local `signal-queue-v1`, deterministic equal-time ordering,
  cancellation and atomic bounded due delivery;
- `local-preview-v5` semantic limits for entropy streams, draw bytes, retained
  scheduler events and deliveries per clock advance;
- ADR/SPEC-0029 through 0031: parsed UTC scheduler ordering, 256-pending
  per-instant admission, bounded next-due preview/drain, and digest-bound
  filtered pagination;
- `local-preview-v6` semantic limits for pending events per instant, scheduler
  inspection pages and cursor bytes;
- ADR/SPEC-0032 through 0034: private runtime-bound scheduled TwinSpec actions,
  spec/schema admission, ordinary-clock fail-closed behavior, one-attempt
  terminal evidence, atomic effect/fault/audit coupling and zero cascade;
- `local-preview-v7` semantic limits of 32 scheduled actions per step, one
  attempt and cascade depth zero, while retaining the 256-total-event bound;

### Changed

- CEL module/import path migrated from `github.com/google/cel-go v0.31.0` to
  the upstream-declared `cel.dev/cel-go v0.32.0`; no Dependabot ignore rule;
- fuzz smoke uses 200,000 parser / 10,000 compiler iterations, one worker,
  explicit watchdogs and narrow counterexample artifacts; timeout is still
  failure, and the compiler target runs independently after parser failure;
- fixed pre-existing CEL null conversion to numeric zero, including nested
  values and persisted effects. Affected results/state digests intentionally
  change; existing snapshots/Evidence are not rewritten. Compare under the
  same runtime revision, not merely the same TwinSpec digest;

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
