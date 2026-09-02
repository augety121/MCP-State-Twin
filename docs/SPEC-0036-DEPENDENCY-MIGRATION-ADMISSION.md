# SPEC-0036: Dependency Migration Admission

- **Status:** Accepted via ADR-0036
- **Date:** 2026-09-02
- **Scope:** reviewed source/dependency migrations and their evidence
- **Initial migration:** `github.com/google/cel-go v0.31.0` to `cel.dev/cel-go v0.32.0`

## 1. Observed incident

[Dependabot run 33498581757](https://github.com/augety121/MCP-State-Twin/actions/runs/33498581757)
reports `go_module_path_mismatch` when updating CEL. It also successfully
created SQLite PR #5; the aggregate failure is not evidence that creating
that PR failed. Upstream's tagged `go.mod` declares `module cel.dev/cel-go`,
and its v0.32.0 release notes explicitly require import-path migration.

The runtime MUST follow the declared module path. A permanent `replace`,
disabling checksum verification, switching to an unreviewed fork, or ignoring
all future CEL updates is not an accepted correction.

## 2. Admission stages

1. **Identify:** read exact failed step, old/new module path, version and
   upstream tagged manifest. Distinguish updater failure from candidate tests.
2. **Baseline:** add reviewed expression vectors on the current dependency;
   record any existing mismatch as a separate defect (see SPEC-0037).
3. **Migrate:** change direct module and every source import together, run
   `go mod tidy`, inspect the entire `go.mod`/`go.sum` diff, and verify checksums.
4. **Validate:** run semantic vectors, boundary tests, full suite, vet, race,
   both reference scenarios, hermetic-egress and conformance on the candidate.
5. **Integrate:** preserve the main-branch evidence and source revision;
   historical failed runs remain immutable diagnostic records.
6. **Observe updater:** a fresh dependency update job must use the new main
   manifest. Its result is separate from the main CI result. Do not claim a
   Dependabot success before observing that job.

For this migration, the direct CEL pin is exactly `cel.dev/cel-go v0.32.0`.
No old-path CEL package may remain in the authored `internal/` imports or
direct module requirements. `TestCELModuleMigrationIsComplete` gates this
bounded source-tree contract; `go list -m all` and checksum verification
provide module-graph evidence. It is not an SBOM or provenance attestation.

## 3. Compatibility matrix

| Surface | Required check | What it does not prove |
|---|---|---|
| Syntax/compilation | valid expressions, malformed syntax, 4,096-byte limit | all CEL language features |
| Values | scalars, maps/lists, null, virtual time, call index | every native/protobuf type |
| Evaluation errors | missing keys, division by zero, non-string keys | byte-identical upstream error messages |
| Resources | 10,000-cost interruption and oversized-source rejection | equal cost for every expression across releases |
| Trust | filesystem/network/process/host-clock/RNG functions unavailable | OS process sandboxing |
| Effects | rollback, invariant and schema checks; both reference domains | live service fidelity |
| Persistence | existing world/Journal reopen/migration suite | universal SQLite-version compatibility |
| Protocol | pinned SDK wire and tools-first conformance tests | provider/product live compatibility |

Use exact canonical results for success vectors and error/refusal categories
for failures. Do not freeze library prose as a protocol contract. CEL's new
optional extensions are not enabled merely because the dependency contains
them; the current environment registers only the existing declared variables.

## 4. Version and reproducibility rules

A source dependency update changes the runtime build identity, even if the
TwinSpec, storage schema and MCP surface stay unchanged. Tests establish only
their covered subset, not observational equivalence for arbitrary programs.
Compare evaluations under the same runtime revision and module graph. Existing
Evidence is immutable; regenerating a report is a new evaluation.

Do not bump storage schema or artifact format solely because a module path
changed. Conversely, a discovered result-semantic correction must have its
own compatibility note and decision rather than being hidden inside the bump.
The null-to-zero correction is adopted separately by ADR-0037.

## 5. Failure and rollback policy

| Failure | Required behavior |
|---|---|
| module path mismatch | inspect upstream declaration and migrate imports |
| checksum mismatch | stop; investigate authenticity/cache, never bypass sumdb |
| old/new module duplication | identify import owner, refuse casual merge |
| compilation/API change | explicit source adaptation and regression tests |
| changed golden value | investigate semantics, never regenerate blindly |
| cost/error change | assess supported profile and fail-closed behavior |
| new optional capability | remain disabled unless separately accepted |
| network/setup failure | mark validation incomplete, not code-compatible |
| unrelated dependency PR | test that candidate independently before merging |

Rollback uses a new reviewed revert restoring source imports and dependency
files together; never rewrite historical releases or evidence. Old commits
remain reproducible under their recorded toolchains. A rollback cannot infer
whether stored numeric zero was originally intended to represent null.

## 6. Maintenance boundaries

Dependabot remains enabled for Go modules and GitHub Actions. No ignore rule
is added to hide this incident. Maintainers review major Actions/runtime
changes separately and preserve immutable action pins. This SPEC does not
authorize merging unrelated open PRs, publishing a stable release, or closing
the provider-live/fidelity/remote-security gates.

Primary sources:
[CEL v0.32.0 release](https://github.com/cel-expr/cel-go/releases/tag/v0.32.0)
and [tagged module manifest](https://github.com/cel-expr/cel-go/blob/v0.32.0/go.mod).
