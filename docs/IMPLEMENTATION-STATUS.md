# Implementation Status

**Build status:** development preview; latest public prerelease `v0.1.0-alpha.1`; no stable release
**Last verified committed baseline:** 2026-09-08; local single-core suite plus public CI
[run 34165668106](https://github.com/augety121/MCP-State-Twin/actions/runs/34165668106)
for implementation commit `7ec3fd3` completed successfully, including Linux
race, Windows/macOS, fuzz, secret policy, hermetic egress and MCP conformance
**Authority:** this file reports implementation evidence. RFC-0001 is the
umbrella design; RFC-0002 is the accepted v0.1 release profile. RFC-0003 is a
v0.2 proposal whose deterministic TwinBundle and local scripted Episode subset
is accepted by ADR-0018, the durable Journal subset by ADR-0019, and the fenced
remote coordinator subset by ADR-0020. ADR-0011 through ADR-0020 accept only
the bounded subsets they name; the rest of the vNext pack remains proposal
material. ADR-0021 additionally governs the release train and claim vocabulary.
ADR-0022 through ADR-0025 accept the bounded local operational-safety batch;
ADR-0026 through ADR-0034 accept the modeled entropy, deterministic scheduler,
bounded-inspection and runtime-bound local TwinSpec-action batch. SPEC-0019
through SPEC-0034 are accepted only to the exact
states recorded below; acceptance is not an implementation claim. Complete
remote-staging security and dated live HostProfiles remain absent.

**2026-09-02 maintenance verification:** ADR-0035–0037 accept count-bounded fuzz
smoke, the `cel.dev/cel-go v0.32.0` module migration and the CEL null-to-JSON
correction. Local single-core vet/full tests, module checksum verification,
200,000-iteration parser fuzz and 10,000-iteration CEL fuzz budgets pass.
Local race testing is unavailable because CGO is disabled; the Linux race
evidence for that maintenance batch is
[CI 33579877698](https://github.com/augety121/MCP-State-Twin/actions/runs/33579877698). The fresh
[Dependabot run 33579893053](https://github.com/augety121/MCP-State-Twin/actions/runs/33579893053)
also succeeded on `61aac2d`, explicitly checking `cel.dev/cel-go`. Historical
failed runs remain unchanged. SQLite PR #5 was refreshed for independent
[candidate CI](https://github.com/augety121/MCP-State-Twin/actions/runs/33580014685)
and has not been merged; main remains on SQLite v1.56.0.

## Implemented and tested

**2026-09-08 offline Task increment (ADR/SPEC-0038):** independent AgentTask
strict parsing and surface admission, scalar resource rules, bounded read-only
goal/policy grading, six synthetic witnesses and `task validate` / `task witness`
are present. Local tests and exact-commit CI
[34163406749](https://github.com/augety121/MCP-State-Twin/actions/runs/34163406749)
on `fcde8ae27a44e8bbc1bbc1536c72204e002c658c` pass, including Linux race,
Windows/macOS, fuzz, secret policy, hermetic egress and MCP conformance.
The witness runner traverses the official MCP SDK with no listening port or
network fallback. It reports `scripted-witness`, not autonomous/live evidence.
This does not close independent B02/B03 semantic review or live/external gates.

**2026-09-08 offline regression increment (ADR/SPEC-0039):** the offline portions
of B04/B05/B06/B07/B09/B10/B12 now include strict mock run configuration,
Responses function projection/continuation, sequential SDK-MCP loop, count/byte/
deadline admission, retained synthetic world evidence, no-clobber publication,
replay verification, fixed-plan model-label comparison and the clean-root CLI
guide. Tests exercise success, actual task failure, committed-but-undelivered
prefix, protocol/privacy stops, replay tampering and incomplete denominators.
Exact-commit CI 34165668106 passed on `7ec3fd3`; it is independent of `fcde8ae`.
That offline increment did not implement paid transport or claim real model scores, product-host compatibility,
general remote drain/recovery, all-filesystem crash guarantee or automatic
upgrade. Full AE/B work-package completion is not implied.

**2026-09-08 API readiness increment (ADR/SPEC-0040–0041):** explicit synthetic
live-plan generation/preflight/approval, a fixed-endpoint Responses transport,
per-plan request caps, failure/unknown-cost receipts, private continuation,
the local SDK-MCP loop and independent live-kind evidence/replay are implemented.
Local single-core contract tests cover six tasks, approval/binding refusals,
HTTP/cancellation/usage/privacy failures, committed-but-undelivered prefixes,
single-use output, receipt/state tampering and CLI admission. Exact-candidate CI
must be checked for this new increment before claiming cross-platform/race success.
No actual model request has been executed as validation of this increment; no
new profile is live-compatible, no paid budget is inferred, and no release is tagged.
This closes bounded implementation prerequisites, not B08 real six-task evidence,
B09-live, B10-external, full storage crash recovery or product-host validation.

| Capability | Evidence |
|---|---|
| Strict TwinSpec YAML decoding | 1 MiB limit, exactly one document, unknown fields rejected by `yaml.v3`; spec tests and fuzz target |
| TwinSpec structural validation | API/kind/name/fidelity/upstream/entity/tool validation tests |
| Full tool schema validation | JSON Schema 2020-12 input/output compilation, nested/format/rollback tests |
| Hermetic schema loading | external `$ref` resource test fails closed |
| Canonical spec and state digest | golden map-order tests |
| Bounded declarative expressions | 4,096-byte source limit; CEL programs compiled once with a cost limit of 10,000 |
| CEL migration/value regression | literal/nested null, map/list/arithmetic/error vectors, nullable output-schema and stored-null tests; `TestCELModuleMigrationIsComplete`; SPEC-0036/0037 |
| Bounded fuzz maintenance | count budgets, one worker, independent watchdogs, failure propagation and synthetic counterexample artifact contract; SPEC-0035; both targets passed in CI run 33579877698 |
| Canonical MCP tool-surface digest | order-independent name/description/schema/annotation digest; mutation tests |
| Surface admission enforcement | matching `current` binding accepted; mismatch, `drifted`, and `unknown` fail `SPEC_DRIFT` |
| Top-level input validation subset | required/additionalProperties/type/enum tests through engine calls |
| Atomic state transitions | SQLite transaction; failed-outcome rollback test |
| Declarative effects | allocate, insert, update/merge, delete |
| Preconditions/postconditions/global invariants | engine tests and reference TwinSpec |
| Immutable snapshots and isolated forks | sibling-fork isolation test |
| Reset and canonical state diff | store implementation and diff tests |
| Storage identity/version | SQLite application ID, current-version admission, foreign/future refusal, v1/v2/v3 migration, tagged alpha schema-v4 reopen fixture, and process-exit migration recovery tests |
| Storage compatibility ledger | separate world-store schema-v4 and Episode-Journal schema-v2 identities, fixtures, migration paths, zero-mutation refusals and explicit non-claims in `STORAGE-COMPATIBILITY-MATRIX.md` |
| Privileged control audit | snapshot/fork/reset audit written in mutation transaction |
| Deterministic replay | 1,000-call corpus replayed on two branches with equal digest at every step |
| Concurrent branch isolation | 100 forks mutated concurrently without sibling/base leakage |
| MCP data plane | official Go SDK, stateless Streamable HTTP integration test |
| MCP conformance subset | official framework `v0.1.16`; initialize, ping, tools-list, JSON Schema 2020-12 pass on Linux CI |
| Control-plane isolation | MCP `tools/list` test rejects all control functions |
| Control authentication | independent bearer-token HTTP test |
| Control auth grammar | missing scheme and raw-token negative tests; constant-time token comparison |
| Operational log boundary | CLI errors pass through secret/identifier redaction; redaction unit tests |
| Strict YAML safety | shared decoder rejects multiple documents, unknown fields, explicit tags, anchors, and aliases |
| Reference environment | six-tool issue-tracker TwinSpec with synthetic fixture; package-registry domain is recorded by ADR-0014 |
| Second reference domain | package-registry TwinSpec with publish, yank, install, advisory-query flows and negative scenario assertions |
| CLI closed loop | init → snapshot → two forks → different calls → canonical diff run locally |
| Scripted scenario runner | bounded Scenario v1alpha1 parser, expected error classes, JSON Pointer assertions, deterministic environment/report digests, ordered trace, and state diff |
| Deterministic TwinBundle | strict v1alpha1 manifest; portable paths; regular non-symlink members; exact declaration set; member SHA-256; TwinSpec/fixture/Scenario semantic admission; compressed/extracted/member bounds; byte-equality reproducibility and tamper tests |
| Local scripted EvaluationEpisode | declared-Scenario admission, closed lifecycle, deterministic Scenario execution, runtime version/revision identity, canonical evidence digest, terminal-outcome tests, and fail-closed existing-output behavior |
| Durable Episode Journal | independent SQLite application identity; schema-v1 fixture to schema-v2 migration; immutable request digest; transactional lifecycle CAS; terminal Evidence atomicity; reopen/idempotent replay/incomplete/conflict/tamper/foreign/future-schema tests; migration process-exit recovery and `integrity_check`; `episode inspect` |
| Fenced remote Episode coordinator | immutable submit policy; one active lease; concurrent claim serialization; heartbeat extension; monotonic fencing; stale-attempt refusal; bounded hermetic recovery; external `COMMIT_UNKNOWN`; queued/leased cancellation; exact duplicate completion lookup; task/attempt/parent consistency admission |
| Provider-neutral remote worker | authenticated coordinator client; hermetic-only claim filter; runtime identity refusal; heartbeat-driven cooperative cancellation; bounded TwinBundle revalidation; atomic Evidence completion; loopback hermetic CI smoke |
| Coordinator network boundary | bearer authentication with constant-time comparison; unknown-field/body bounds; typed sanitized errors; plaintext restricted to loopback; TLS required for non-loopback listeners; token is not echoed or persisted |
| Provider smoke contract harness | OpenAI Responses background create/retrieve/cancel and remote MCP result parsing; Anthropic Messages MCP connector current-beta request/result parsing; capability differences; strict HTTPS endpoint admission; bounded redacted report; mock positive/cancel/error/no-secret tests |
| Claim and compatibility governance | requirement traceability, public Claim Registry, exact-profile Compatibility Matrix, Decision Register and Phase 0–7 exit contracts |
| MCP 2026 wire evidence | raw `server/discover`, direct modern `tools/list`, result discriminator, header/body mismatch, and 2025-11-25 initialize compatibility tests; pinned SDK evidence CLI |
| Monotonic branch head | SQLite schema v4 `head_version`, CAS updates for calls/reset/clock/fault configuration, snapshot source-head binding, migration tests |
| Private virtual-clock advance | bounded forward-only `/v1/clock/advance`, expected-head conflict, and transactional `clock.advance` audit tests |
| Deterministic modeled entropy | opt-in `sha256-ctr-v1`, public synthetic seed, branch-local persistent counters, 1..32-byte draw, fork equality and failed-transition rollback |
| Branch-local signal scheduler | private create/list/cancel routes, total due/priority/sequence/ID ordering, snapshot/fork/reset identity and typed lifecycle conflicts |
| Atomic due-signal delivery | clock and all due event lifecycle changes in one transaction; 256-delivery bound with whole-operation rollback |
| Scheduler temporal/liveness hardening | parsed UTC ordering, 256-pending per-instant admission for new events, cancellation capacity release, and deterministic bounded recovery for legacy overfull instants |
| Scheduler bounded inspection | authenticated next-due preview plus 100-default/256-maximum status-filtered pages with digest-bound versioned cursors and stale-page refusal |
| Runtime-bound scheduled TwinSpec actions | authenticated private admission validates the loaded modeled tool/input schema, injects the canonical spec digest, refuses ordinary clock jumps over due actions, and executes only through runtime-backed bounded `advance-next` |
| Scheduled action terminal evidence | one-attempt `completed`/`failed` outcomes record call index, result, typed error, effect-commit ambiguity and optional fault identity; effects, fault consumption, tool/fault audit, lifecycle, clock, call count and aggregate head commit in one SQLite transaction |
| Scheduled action budget and zero cascade | `local-preview-v7` caps one step at 32 actions/256 total events, preserves the total-order prefix, rejects callback scheduler mutation even on reported domain failure, and rolls back oversized results or infrastructure errors |
| Deterministic fault preview | branch-local bounded plans; `before-validation` and `after-commit-before-response`; atomic counters/events; stable plan digest; private HTTP integration tests |
| Versioned resource profile | `statetwin limits`; profile digest in Scenario environment identity; state/input/output/query/effect/diff/report/storage bounds; typed `RESOURCE_LIMIT` failures |
| Conservative local execution profile | `local-v2`: `quiet` applies `GOMAXPROCS=1`, 512 MiB Go heap soft target and four in-flight requests per listener; validated mode/exact overrides, environment/CLI precedence and `statetwin execution-profile`; soft process-local boundary only |
| HTTP admission/backpressure | independent non-queueing pool per data/control/coordinator listener; typed redacted `503 SERVER_BUSY`, retry header, active/rejected counters and capacity-recovery tests |
| Operational health/readiness | authenticated control routes `/v1/health/live` and `/v1/health/ready`; readiness uses context-bound store ping and redacts storage details; absent from MCP tools |
| Maintainer/release automation | release checklist, docs authority map, PR/Issue templates, Dependabot, and tag-driven multi-platform release workflow are present; no stable release has been published |
| HTTP server bounds | 1 MiB bodies/headers, read/write/idle timeouts, configuration tests, and a direct slow-header connection test that proves the application handler is not reached |
| Host compatibility report admission | strict 1 MiB single-document schema, immutable revision/digest checks, bounded profile checks, remote deployment-profile binding, credential/private-key/email pattern rejection, and `statetwin compatibility validate` |

## Partially implemented

| Capability | Current boundary |
|---|---|
| Virtual time | private clock, modeled entropy, signals, and one-attempt runtime-bound local TwinSpec actions have parsed-time ordering, bounded atomic next-due progress, terminal evidence and consistent inspection; scheduled Agents/providers/external effects, recurrence, automatic retry/dead letters and non-zero cascades are not implemented |
| Deterministic faults | `before-validation` and `after-commit-before-response` are implemented for ordinary and scheduled local TwinSpec actions with explicit effect-commit evidence; latency, partial effects, idempotency collapse, crash/cancellation and eventual consistency are not |
| Upstream surface discovery | local canonicalization and binding enforcement work; upstream inspection and automatic refresh are not implemented |
| Hermeticity | there is no upstream connector or passthrough code; only-loopback Linux CI job passed in run #6 |
| Secret/fixture policy | pinned Gitleaks history scan and synthetic-fixture heuristic passed in run #6 |
| Snapshot storage | immutable logical snapshots work; copy-on-write/delta optimization and GC are not implemented |
| MCP protocol coverage | direct 2026-07-28 wire smoke tests pass; the pinned conformance framework still covers legacy-era scenarios and does not establish every modern optional feature |
| Resource governance | `local-preview-v7` bounds semantic resources including entropy, signal/action delivery, actions per step, attempts, zero cascade, per-instant admission and scheduler inspection; separate `local-v2` ExecutionProfile adds conservative CPU, Go heap soft target and per-listener admission. OS hard quotas, RSS/native/child controls, distributed/tenant fairness, durable queues, cassette quotas, retention, and empirical performance budgets are not implemented |
| Portable evaluation artifacts | deterministic unsigned TwinBundle, scripted EpisodeEvidence, and bounded remote transport are implemented; signatures, provenance attestations, registry transport, and publisher identity are not |
| Episode delivery semantics | hermetic tasks support bounded at-least-once claim delivery with fenced exactly-once terminal Evidence acceptance; arbitrary provider calls, HTTP delivery, tool effects, and external side effects are not exactly-once |
| Provider live evidence | executable OpenAI/Anthropic adapters and opt-in CI workflow exist; only mock contract tests have run in this repository state, because provider credentials and a public synthetic MCP endpoint are absent |

## Not implemented

- recorder and trace redaction;
- L0 cassette replay;
- remaining deterministic fault phases, idempotency semantics, crash/cancellation injection and eventual consistency;
- scheduled Agent/provider/process/external effects, recurrence, automatic retry/dead letters and non-zero cascades;
- dated OpenAI-family and Anthropic-family live provider reports and live-agent trajectory capture;
- full MCP 2026-07-28 conformance coverage beyond the pinned official subset;
- live ChatGPT, OpenAI API, Claude, or Claude Code smoke tests;
- evidence-derived host compatibility matrix and admitted live provider reports;
- complete SPEC-0020 remote-staging deployment/security profile;
- differential validation against an upstream fixture service;
- L2 promotion workflow and coverage report;
- full import/export/migration tooling;
- multi-coordinator HA, Journal replication, remote worker identity/attestation, retention and scheduling;
- automatic retry after ambiguous external effects, operator reconciliation for `COMMIT_UNKNOWN`, or exactly-once external effects;
- generic HostProfile execution;
- TwinBundle signatures, publisher identity, registry distribution, and provenance attestations;
- data-plane authentication, TLS, remote multi-tenancy, or cloud deployment;
- native/reference L3 adapters;
- OpenTelemetry integration.

The vNext proposal adoption boundary is documented in
[VNEXT-ADOPTION.md](VNEXT-ADOPTION.md), with the complete file-level matrix in
[VNEXT-TRACEABILITY.md](VNEXT-TRACEABILITY.md); proposal text is not
implementation evidence.

## Verification commands

```bash
go test ./...
go test -race ./...
bash scripts/run-bounded-fuzz.sh spec
bash scripts/run-bounded-fuzz.sh engine
go vet ./...
go build ./cmd/statetwin
go run ./cmd/statetwin validate --spec examples/issue-tracker/twin.yaml
go test ./internal/hostcompat ./internal/store
go run ./cmd/statetwin bundle build --manifest examples/issue-tracker/bundle.yaml --out issue-tracker.stb
go run ./cmd/statetwin bundle verify --bundle issue-tracker.stb
go run ./cmd/statetwin episode run --bundle issue-tracker.stb --id local-episode-001 --out episode-evidence.json
go run ./cmd/statetwin episode run --bundle issue-tracker.stb --id durable-episode-001 --journal episodes.db
go run ./cmd/statetwin episode inspect --journal episodes.db --id durable-episode-001
go run ./cmd/statetwin episode submit --bundle issue-tracker.stb --id remote-episode-001 --journal remote.db --effect-profile hermetic
STATETWIN_COORDINATOR_TOKEN=synthetic-local-token go run ./cmd/statetwin episode coordinator --journal remote.db --addr 127.0.0.1:8092
STATETWIN_COORDINATOR_TOKEN=synthetic-local-token go run ./cmd/statetwin episode worker --coordinator http://127.0.0.1:8092 --id worker-001 --once
go test ./internal/provider ./internal/episode ./internal/store
```

Artifact/evidence output commands refuse to overwrite existing paths; use fresh
paths in automation and delete local test artifacts after inspection.

Any README claim should be traceable to this matrix or to a reproducible command.
