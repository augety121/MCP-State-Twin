# Implementation Status

**Build status:** development preview; latest public prerelease `v0.1.0-alpha.1`; no stable release
**Last verified locally:** 2026-08-31; merged/public CI evidence is still required
**Authority:** this file reports implementation evidence. RFC-0001 is the
umbrella design; RFC-0002 is the accepted v0.1 release profile. RFC-0003 is a
v0.2 proposal whose deterministic TwinBundle and local scripted Episode subset
is accepted by ADR-0018, the durable Journal subset by ADR-0019, and the fenced
remote coordinator subset by ADR-0020. ADR-0011 through ADR-0020 accept only
the bounded subsets they name; the rest of the vNext pack remains proposal
material. ADR-0021 additionally governs the release train and claim vocabulary.
SPEC-0019 through SPEC-0021 are accepted contracts, but acceptance is not an
implementation claim: complete remote-staging security and dated live
HostProfiles remain absent.

## Implemented and tested

| Capability | Evidence |
|---|---|
| Strict TwinSpec YAML decoding | 1 MiB limit, exactly one document, unknown fields rejected by `yaml.v3`; spec tests and fuzz target |
| TwinSpec structural validation | API/kind/name/fidelity/upstream/entity/tool validation tests |
| Full tool schema validation | JSON Schema 2020-12 input/output compilation, nested/format/rollback tests |
| Hermetic schema loading | external `$ref` resource test fails closed |
| Canonical spec and state digest | golden map-order tests |
| Bounded declarative expressions | 4,096-byte source limit; CEL programs compiled once with a cost limit of 10,000 |
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
| Deterministic fault preview | branch-local bounded plans; `before-validation` and `after-commit-before-response`; atomic counters/events; stable plan digest; private HTTP integration tests |
| Versioned resource profile | `statetwin limits`; profile digest in Scenario environment identity; state/input/output/query/effect/diff/report/storage bounds; typed `RESOURCE_LIMIT` failures |
| Maintainer/release automation | release checklist, docs authority map, PR/Issue templates, Dependabot, and tag-driven multi-platform release workflow are present; no stable release has been published |
| HTTP server bounds | 1 MiB bodies/headers, read/write/idle timeouts, configuration tests, and a direct slow-header connection test that proves the application handler is not reached |
| Host compatibility report admission | strict 1 MiB single-document schema, immutable revision/digest checks, bounded profile checks, remote deployment-profile binding, credential/private-key/email pattern rejection, and `statetwin compatibility validate` |

## Partially implemented

| Capability | Current boundary |
|---|---|
| Virtual time | private forward-only clock advancement is implemented; scheduler, entropy, due events, and scheduled effects are not implemented |
| Deterministic faults | two transaction phases and three canonical outcomes are implemented; latency, partial effects, idempotency collapse, crash/cancellation, scheduled visibility, and eventual consistency are not |
| Upstream surface discovery | local canonicalization and binding enforcement work; upstream inspection and automatic refresh are not implemented |
| Hermeticity | there is no upstream connector or passthrough code; only-loopback Linux CI job passed in run #6 |
| Secret/fixture policy | pinned Gitleaks history scan and synthetic-fixture heuristic passed in run #6 |
| Snapshot storage | immutable logical snapshots work; copy-on-write/delta optimization and GC are not implemented |
| MCP protocol coverage | direct 2026-07-28 wire smoke tests pass; the pinned conformance framework still covers legacy-era scenarios and does not establish every modern optional feature |
| Resource governance | `local-preview-v4` adds bounded Episode attempts and leases to existing local/bundle limits; OS quotas, multi-tenant fairness, scheduler/cassette quotas, retention, and empirical performance budgets are not implemented |
| Portable evaluation artifacts | deterministic unsigned TwinBundle, scripted EpisodeEvidence, and bounded remote transport are implemented; signatures, provenance attestations, registry transport, and publisher identity are not |
| Episode delivery semantics | hermetic tasks support bounded at-least-once claim delivery with fenced exactly-once terminal Evidence acceptance; arbitrary provider calls, HTTP delivery, tool effects, and external side effects are not exactly-once |
| Provider live evidence | executable OpenAI/Anthropic adapters and opt-in CI workflow exist; only mock contract tests have run in this repository state, because provider credentials and a public synthetic MCP endpoint are absent |

## Not implemented

- recorder and trace redaction;
- L0 cassette replay;
- remaining deterministic fault phases, idempotency semantics, crash/cancellation injection, and eventual consistency;
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
go test ./internal/spec -run=^$ -fuzz=FuzzDecodeTwinSpec -fuzztime=10s
go test ./internal/engine -run=^$ -fuzz=FuzzExpressionCompilation -fuzztime=10s
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
