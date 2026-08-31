# ADR-0023 — Go heap soft-limit governance

- **Status:** Accepted
- **Date:** 2026-08-31
- **Scope:** memory managed by the Go runtime in a `statetwin` process

## Context

CPU parallelism alone does not prevent a large heap from increasing garbage
collection pressure, latency and fan noise. SPEC-0015 bounds modeled payloads,
but aggregate runtime memory includes parsers, SQLite buffers, HTTP state and
temporary canonical representations.

Go exposes a portable runtime memory limit. It is a soft garbage-collector
target rather than a reservation or an operating-system enforcement boundary.

## Decision

The ExecutionProfile `local-v2` applies a Go runtime soft memory limit before
command dispatch:

| Mode | Default soft limit |
|---|---:|
| `quiet` | 512 MiB |
| `balanced` | 1,024 MiB |
| `throughput` | 2,048 MiB |

`--memory-limit-mib` or `STATETWIN_MEMORY_LIMIT_MIB` may select an exact value
from 64 MiB through 1 PiB. The CLI value wins. Invalid values fail before work
starts.

## Non-claims

The limit is not an RSS limit, container quota, Windows Job Object limit,
allocation guarantee or out-of-memory recovery protocol. It does not cover
memory owned by the kernel, mapped files, child processes, future native code,
or every SQLite allocation. `hardMemoryQuota` remains `false`.

## Consequences

The operational profile must remain separate from deterministic world
identity. Performance evidence must record it. Tests must restore the previous
process setting so package tests do not leak global runtime state.
