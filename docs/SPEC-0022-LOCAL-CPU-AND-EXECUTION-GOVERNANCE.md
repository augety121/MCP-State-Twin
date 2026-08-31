# SPEC-0022 — Local CPU and Execution Governance

- **Status:** Accepted by ADR-0022
- **Version:** `statetwin.dev/execution-profile/v1alpha1`, `local-v1`
- **Applies to:** `statetwin` CLI, servers, coordinator, Episode workers and
  provider-smoke adapter code running in the same Go process

## 1. Goal

Keep local execution conservative by default so normal project use does not
schedule Go work across every logical CPU. The contract is portable and
fail-closed, while remaining explicit that it is not a hard OS CPU quota.

## 2. ExecutionProfile

The machine-readable profile has these fields:

| Field | Meaning |
|---|---|
| `format` / `version` | exact profile schema and behavior revision |
| `mode` | `quiet`, `balanced`, or `throughput` |
| `logicalCpus` | logical CPUs observed during resolution |
| `maxProcs` | Go scheduler execution-slot limit actually applied |
| `workerConcurrency` | simultaneous Episode claims executed by this worker |
| `hardCpuQuota` | always `false` in `local-v1` |
| `appliesToGoRuntime` | `true` |
| `appliesToChildProcesses` | `false` |
| `source` | `default`, `environment`, or `command-line` |

`statetwin execution-profile` MUST emit the resolved structure as JSON after
the same policy has been applied to that process.

## 3. Resolution and precedence

The implementation MUST resolve settings in this order:

1. root CLI options `--execution-mode` and `--max-procs`;
2. `STATETWIN_EXECUTION_MODE` and `STATETWIN_MAX_PROCS`;
3. default mode `quiet`.

Root options MUST appear before the command:

```text
statetwin --execution-mode balanced scenario ...
statetwin --max-procs 2 episode worker ...
```

An explicit maximum overrides the mode-derived slot count but does not rename
the selected mode. Duplicate root options are invalid.

## 4. Mode semantics

For `C = max(1, detected logical CPUs)`:

```text
quiet      = 1
balanced   = min(4, ceil(C / 2))
throughput = C
```

An explicit maximum MUST be an integer in `1..C`. Invalid configuration MUST
terminate before command dispatch. There is no `unlimited`, `auto`, or zero
sentinel in this version.

## 5. Enforcement boundary

The process MUST call `runtime.GOMAXPROCS(maxProcs)` before it creates command
goroutines, HTTP servers, workers or provider clients. The Episode worker MUST
continue to execute at most one claimed task at a time.

The following are outside this enforcement boundary:

- the `go` compiler/test driver that launches the binary;
- shell scripts and child processes;
- operating-system, filesystem and network kernel work;
- native threads from future cgo libraries;
- other programs on the host.

Maintainer test instructions SHOULD use `GOMAXPROCS=1` and `go test -p 1` for
a quiet local validation pass. CI may use a separate explicit policy because
it does not consume the maintainer workstation.

## 6. Determinism and evidence

ExecutionProfile MUST NOT alter semantic ResourceProfile digesting. A test
corpus run under quiet and throughput modes MUST produce equivalent modeled
results and canonical state when it uses the same accepted deterministic
inputs.

Performance or thermal claims MUST NOT be inferred from this contract. Any
future latency benchmark MUST record the exact ExecutionProfile and host
identity separately from semantic environment identity.

## 7. Overload and shutdown

This version bounds CPU parallelism; it does not add a queue or discard work.
Existing request-size, lease, timeout, cancellation and shutdown behavior
continues to apply. A future concurrency/admission controller MUST define:

- queue capacity and ordering;
- overload error type and retry advice;
- cancellation while queued;
- per-tenant fairness;
- metrics that do not expose secrets or hidden state.

It MUST NOT be added as an undocumented semaphore because admission order can
affect wall-clock behavior and availability.

## 8. Acceptance tests

The accepted implementation requires executable evidence for:

1. default mode resolves to one slot on multi-core input;
2. balanced and throughput calculations at boundary CPU counts;
3. CLI-over-environment precedence;
4. invalid mode, missing value, duplicate option, zero and above-host refusal;
5. application of the selected `GOMAXPROCS` value;
6. machine-readable `execution-profile` smoke;
7. unchanged deterministic Scenario outcomes across at least quiet and
   throughput modes before performance claims are introduced.

Items 1–6 are required now. Item 7 is a gate for future benchmark/performance
claims, not a claim made by the current release.
