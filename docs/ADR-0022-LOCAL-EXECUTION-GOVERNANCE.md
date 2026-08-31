# ADR-0022 — Conservative local execution governance

- **Status:** Accepted
- **Date:** 2026-08-31
- **Scope:** all `statetwin` CLI processes; local and single-worker preview

## Context

SPEC-0015 bounds semantic inputs, state and artifacts, but those limits do not
prevent a process from scheduling runnable Go work on every logical CPU. That
is undesirable on maintainer laptops and workstations: it can create sustained
fan noise, thermal pressure and interference with interactive work.

A portable process cannot honestly promise an exact CPU percentage. Go's
runtime scheduler can bound simultaneous Go execution, but it does not control
the operating system scheduler, child processes, kernel work, firmware, or
future native libraries. A hard quota requires an independently implemented
and tested OS isolation backend such as a cgroup, container CPU quota, or
Windows Job Object policy.

## Decision

Every `statetwin` process applies an operational ExecutionProfile before it
starts a command, server, worker, or provider adapter.

The default profile is `quiet` and sets `GOMAXPROCS=1`. Other modes are opt-in:

- `balanced`: `ceil(logical CPUs / 2)`, capped at four slots;
- `throughput`: all detected logical CPUs;
- `--max-procs N`: exact validated override within `1..logical CPUs`.

Root command-line options take precedence over `STATETWIN_EXECUTION_MODE` and
`STATETWIN_MAX_PROCS`; environment settings take precedence over the default.
The resolved profile is inspectable with `statetwin execution-profile`.

The remote Episode worker remains single-claim and single-execution. This ADR
does not introduce parallel task execution.

## Identity boundary

The ExecutionProfile is operational evidence, not deterministic environment
identity. It must not change modeled state, generated IDs, virtual time, tool
results, or canonical Evidence digests. The semantic ResourceProfile and its
digest remain unchanged at `local-preview-v4`.

Wall-clock performance comparisons must record the ExecutionProfile because
different modes can change latency. A future benchmark report format may make
that record mandatory; the current Scenario and Episode formats do not claim
empirical performance comparability.

## Failure behavior

Unknown modes, missing values, repeated root options, non-integer limits, zero,
and values above the detected logical CPU count fail before the requested
command starts. The runtime must not silently fall back to unrestricted mode.

## Explicit non-claims

This ADR does not claim:

- an exact CPU utilization percentage;
- an OS-enforced CPU, energy, temperature, or memory quota;
- a limit on child processes or non-Go native threads;
- protection from unrelated host processes;
- remote tenant fairness, admission control, or rate limiting;
- identical wall-clock latency across execution profiles.

Those capabilities require separate ADRs, platform-specific tests and release
evidence.

## Verification

Executable tests cover conservative defaults, mode calculation, precedence,
invalid input and application/restoration of `GOMAXPROCS`. CI also executes
the profile inspection command. Documentation must preserve the soft-governor
versus hard-quota distinction.

## Amendments: ADR-0023 and ADR-0024 (2026-08-31)

The profile advances from `local-v1` to `local-v2`. ADR-0023 adds a Go heap
soft limit; ADR-0024 adds a per-listener non-queueing HTTP admission bound.
Neither amendment creates an OS hard quota or changes semantic environment
identity. ADR-0025 separately adds authenticated control-plane health routes.
