# Implementation Status

**Build status:** development preview; latest public prerelease `v0.1.0-alpha.1`; no stable release
**Last verified:** 2026-08-26
**Authority:** this file reports implementation evidence. RFC-0001 is the
umbrella design; RFC-0002 is the accepted v0.1 release profile. RFC-0003 is a
v0.2 proposal whose deterministic TwinBundle and local scripted Episode subset
is accepted by ADR-0018. ADR-0011 through ADR-0018 accept only the bounded
subsets they name; the rest of the vNext pack remains proposal material.

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
| Resource governance | local profile and fail-closed enforcement are implemented; ADR-0018 adds bounded bundle file/compressed/extracted/member admission; OS quotas, distributed fairness, scheduler/cassette quotas, and empirical performance budgets are not |
| Portable evaluation artifacts | deterministic unsigned local TwinBundle and scripted EpisodeEvidence are implemented; import/export migrations, signatures, provenance attestations, registry transport, and remote execution are not |

## Not implemented

- recorder and trace redaction;
- L0 cassette replay;
- remaining deterministic fault phases, idempotency semantics, crash/cancellation injection, and eventual consistency;
- provider/model live harnesses and live-agent trajectory capture;
- full MCP 2026-07-28 conformance coverage beyond the pinned official subset;
- live ChatGPT, OpenAI API, Claude, or Claude Code smoke tests;
- evidence-derived host compatibility matrix and admitted live provider reports;
- differential validation against an upstream fixture service;
- L2 promotion workflow and coverage report;
- full import/export/migration tooling;
- durable Episode persistence, concurrent workers, remote runners, cancellation/retry recovery, and scheduling;
- generic HostProfile execution or provider SDK adapters;
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
```

The three artifact commands above refuse to overwrite existing output paths;
use fresh paths in automation and delete test artifacts after inspection.

Any README claim should be traceable to this matrix or to a reproducible command.
