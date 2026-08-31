# SPEC-0023 — Go Heap Memory Governance

- **Status:** Accepted by ADR-0023
- **Profile:** `statetwin.dev/execution-profile/v1alpha1`, `local-v2`

## 1. Requirements

Before starting any command goroutine or HTTP listener, the CLI MUST call
`debug.SetMemoryLimit` with the resolved `softMemoryLimitBytes`.

Resolution precedence is:

1. root option `--memory-limit-mib`;
2. environment variable `STATETWIN_MEMORY_LIMIT_MIB`;
3. selected-mode default.

The value MUST be an integer in `64..1048576` MiB. Zero, negative, malformed,
overflowing and above-profile values MUST fail before command dispatch.

## 2. Semantics

The limit is operational metadata. It MUST NOT enter the semantic
ResourceProfile digest, virtual time, modeled state, tool result, state digest,
or canonical Episode Evidence.

The resolved byte count and `hardMemoryQuota: false` MUST appear in
`statetwin execution-profile`, with field-level configuration provenance. The
project MUST use the words **soft memory limit** in public claims.

## 3. Failure model

Reaching the soft limit may increase garbage collection and latency. It does
not guarantee graceful recovery from process, kernel, driver, SQLite, native,
or child-process memory exhaustion. The runtime MUST NOT translate an actual
process OOM into a modeled tool error or success.

Per-request/state/artifact limits from SPEC-0015 remain the primary fail-closed
admission controls. This SPEC does not replace them.

## 4. Evidence

Tests MUST verify mode defaults, override precedence, lower/upper-bound
rejection, applied runtime value, previous-value restoration, JSON inspection,
and documentation of the non-claim.
