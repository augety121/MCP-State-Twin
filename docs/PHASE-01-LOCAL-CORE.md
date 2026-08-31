# Phase 1: v0.1 Local Deterministic Core

- **Target:** `v0.1.x`
- **Profile:** local hermetic, single process, serial transition scheduler
- **Entry:** Phase 0 complete

## Required scope

- strict bounded TwinSpec and CEL admission;
- JSON Schema 2020-12 input/output validation;
- canonical TwinSpec, surface, state and Evidence digests;
- SQLite atomic transitions and explicit database identity;
- immutable snapshots, isolated forks, reset and canonical diff;
- monotonic branch head and CAS;
- explicit modeled errors and `UNMODELED_BEHAVIOR`;
- separately bound MCP data and simulation control planes;
- direct modern and legacy tools-first MCP wire tests;
- bounded Scenario, TwinBundle and local scripted Episode;
- local durable Journal and supported forward migrations;
- synthetic issue-tracker and package-registry reference twins.

## Explicit exclusions

Provider/product live compatibility, remote hosting, distributed workers,
recorder/L0, L2/L3, multi-tenancy, arbitrary plugins, production passthrough,
deterministic multi-agent interleaving and external-effect exactly-once are not
part of v0.1.

## Exit evidence

```text
gofmt -l . == empty
go vet ./...
go test -race ./...
bounded TwinSpec/CEL fuzz
fresh and historical database migration tests
interrupted-migration recovery
modern and legacy MCP wire tests
hermetic no-egress job
TwinBundle adversarial admission
secret/fixture policy scan
claim/documentation audit
```

All evidence must pass on the exact candidate commit. Older green runs cannot
close a release gate.
