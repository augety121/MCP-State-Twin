# Implementation Status

**Build status:** development preview, no tagged release  
**Last verified:** 2026-08-17  
**Authority:** this file reports implementation evidence; RFC-0001 remains the
design target.

## Implemented and tested

| Capability | Evidence |
|---|---|
| Strict TwinSpec YAML decoding | unknown fields rejected by `yaml.v3`; spec tests |
| TwinSpec structural validation | API/kind/name/fidelity/upstream/entity/tool validation tests |
| Full tool schema validation | JSON Schema 2020-12 input/output compilation, nested/format/rollback tests |
| Hermetic schema loading | external `$ref` resource test fails closed |
| Canonical spec and state digest | golden map-order tests |
| Bounded declarative expressions | CEL programs compiled once with a cost limit of 10,000 |
| Top-level input validation subset | required/additionalProperties/type/enum tests through engine calls |
| Atomic state transitions | SQLite transaction; failed-outcome rollback test |
| Declarative effects | allocate, insert, update/merge, delete |
| Preconditions/postconditions/global invariants | engine tests and reference TwinSpec |
| Immutable snapshots and isolated forks | sibling-fork isolation test |
| Reset and canonical state diff | store implementation and diff tests |
| Storage identity/version | SQLite application ID and schema-version/refusal tests |
| Privileged control audit | snapshot/fork/reset audit written in mutation transaction |
| Deterministic replay | 1,000-call corpus replayed on two branches with equal digest at every step |
| Concurrent branch isolation | 100 forks mutated concurrently without sibling/base leakage |
| MCP data plane | official Go SDK, stateless Streamable HTTP integration test |
| Control-plane isolation | MCP `tools/list` test rejects all control functions |
| Control authentication | independent bearer-token HTTP test |
| Control auth grammar | missing scheme and raw-token negative tests; constant-time token comparison |
| Reference environment | six-tool issue-tracker TwinSpec with synthetic fixture |
| CLI closed loop | init → snapshot → two forks → different calls → canonical diff run locally |

## Partially implemented

| Capability | Current boundary |
|---|---|
| Virtual time | every branch has a fixed virtual clock used by CEL; clock advancement and scheduled effects are not implemented |
| Migration coverage | schema v1 snapshot and untagged legacy audit layouts are upgraded; no tagged historical database fixture exists yet |
| Surface binding | metadata models `current/drifted/unknown/unbound`; upstream inspection and automatic drift calculation are not implemented |
| Hermeticity | there is no upstream connector or passthrough code; an automated egress-deny release test is not implemented |
| Snapshot storage | immutable logical snapshots work; copy-on-write/delta optimization and GC are not implemented |
| MCP conformance | official `v0.1.16` initialize/ping/tools-list/JSON-Schema checks pass locally; pinned Linux CI execution is pending |
| HTTP resource bounds | 1 MiB bodies/headers and read/write/idle timeouts are configured; oversize/config tests pass, direct slow-client test remains incomplete |

## Not implemented

- recorder and trace redaction;
- L0 cassette replay;
- deterministic fault injection;
- virtual-clock advancement and eventual consistency;
- scenario runner and provider harnesses;
- official MCP conformance-suite execution;
- live ChatGPT, OpenAI API, Claude, or Claude Code smoke tests;
- differential validation against an upstream fixture service;
- L2 promotion workflow and coverage report;
- full import/export/migration tooling;
- data-plane authentication, TLS, remote multi-tenancy, or cloud deployment;
- native/reference L3 adapters;
- OpenTelemetry integration.

## Verification commands

```bash
go test ./...
go test -race ./...
go vet ./...
go build ./cmd/statetwin
go run ./cmd/statetwin validate --spec examples/issue-tracker/twin.yaml
```

Any README claim should be traceable to this matrix or to a reproducible command.
