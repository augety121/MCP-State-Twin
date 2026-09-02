# SPEC-0035: Bounded Fuzzing and Failure Evidence

- **Status:** Accepted via ADR-0035
- **Date:** 2026-09-02
- **Scope:** repository CI and local maintenance; not a runtime capability
- **Owner:** project maintainers
- **Evidence:** implementation/remote execution status is recorded in `IMPLEMENTATION-STATUS.md`

## 1. Problem and evidence boundary

[Run 33498653329](https://github.com/augety121/MCP-State-Twin/actions/runs/33498653329)
failed in `FuzzDecodeTwinSpec`, after approximately 11 seconds for a requested
10-second fuzz duration, with `context deadline exceeded`. The log records
192,614 executions and no saved crashing input. Other jobs in that PR run,
including Linux race tests, passed. This establishes the observed failure,
not a proof of a Go bug, parser hang, harmless flake, or SQLite regression.

The corrective policy separates the search budget from an outer watchdog.
It MUST NOT reinterpret a timeout as success or retry until a green result
appears. A missing counterexample remains an evidence gap.

## 2. Accepted smoke profile

| Dimension | TwinSpec parser | CEL compilation |
|---|---|---|
| Package | `./internal/spec` | `./internal/engine` |
| Exact target | `FuzzDecodeTwinSpec` | `FuzzExpressionCompilation` |
| Search budget (`-fuzztime`) | `200000x` | `10000x` |
| Minimization per attempt | `1000x` | `1000x` |
| Workers (`-parallel`) | 1 | 1 |
| Go scheduler slots | 1 | 1 |
| Build package concurrency | 1 | 1 |
| Go test watchdog | 3 minutes | 3 minutes |
| CI step watchdog | 5 minutes | 5 minutes |

The combined job has a 15-minute watchdog, including setup and artifact upload.
The count budget is the toolchain's iteration limit, not a promise that its
displayed execution total equals the budget: seed collection and minimization
are separate work. Corpus evolution is coverage-guided and MUST NOT be called
a deterministic enumeration or a coverage percentage.

The entry point is `bash scripts/run-bounded-fuzz.sh spec|engine`. Unknown or
missing targets fail with exit code 2. The wrapper prints the actual Go
version and chosen profile, passes through any Go failure, and never runs a
provider, production service or arbitrary user-selected package.

## 3. Failure taxonomy and response

| Observation | Classification | Required response |
|---|---|---|
| Assertion/panic with saved input | reproducible candidate | replay exact input, minimize/review, add regression |
| Deadline without saved input | unresolved timeout | retain log and environment; reproduce with bounded run |
| Compiler/download/setup failure | build/infrastructure | diagnose dependency/toolchain/network before rerun |
| Go-test watchdog | test did not complete | fail gate; inspect stack/log, never count as coverage success |
| CI cancellation | interrupted | no passing claim; distinguish from completed failure |
| Artifact upload failure | evidence unavailable | record missing artifact, do not invent a counterexample |
| Count budget completes with exit 0 | bounded smoke passed | claim only target/profile/revision tested |

The CEL step runs even if the parser step fails, unless the job is cancelled.
This provides independent evidence without `continue-on-error`. A failed
parser step still fails the job. No retry or exception filter is permitted
for `context deadline exceeded`.

## 4. Artifact and trust boundary

On job failure, upload only these directories when they exist:

- `internal/spec/testdata/fuzz/`
- `internal/engine/testdata/fuzz/`

The artifact name includes run ID and attempt; retention is 14 days. Missing
files are a warning, not a synthetic success report. No entire workspace,
environment dump, module cache, database, trace or credentials are uploaded.
Tests and seeds MUST remain synthetic. Generated data is untrusted input,
never shell instructions. Before committing a counterexample, review it for
secrets, licensing and personal data. Do not print private reproductions in
public Actions logs.

The authoritative run record includes event, PR/head and checkout revision,
run/attempt/job IDs, toolchain version, target, budget, exit status, error text
and artifact presence. A PR merge checkout is not identical to its head SHA;
record which one executed. A later green main run cannot rewrite a historical
failed PR run or establish a different dependency candidate as validated.

## 5. Acceptance tests

- `TestFuzzWorkflowPreservesFailureAndEvidence` checks job budget, independent
  target execution, no ignored failures and the artifact path allowlist.
- `TestBoundedFuzzScriptExitStatus` runs a synthetic command fixture on POSIX,
  verifies both target budgets, single-worker flags, bad-target rejection and
  exact nonzero exit propagation. Windows skips this shell-only test; Linux
  CI and macOS exercise it.
- Actual parser/CEL fuzz runs MUST exit 0 on the candidate revision before
  recording a remote pass. Wrapper tests alone are not fuzz evidence.
- Seed cases remain part of ordinary `go test`; disabling fuzz smoke MUST NOT
  be used to make a dependency PR green.

## 6. Limits and follow-up

This policy does not prove absence of parser vulnerabilities, bound all
possible input runtimes, or guarantee a fixed CPU percentage or fan speed.
Long-running fuzz campaigns, corpus retention beyond 14 days and hardened
untrusted-input isolation require separate resource/evidence policies.
Recurrent deadlines trigger investigation rather than ever larger automatic
budgets. Review budgets when changing parsers, CEL, Go or seed shape.

Source: [Go fuzzing documentation](https://go.dev/doc/security/fuzz/).
