# Unified Requirement Traceability

**Status:** executable mapping for accepted requirements
**Last reviewed:** 2026-09-12

| Requirement | Contract | Implementation/evidence | Current state |
|---|---|---|---|
| I-1 no hidden live writes | RFC-0001 | hermetic namespace job, source absence policy | verified local profile |
| I-2 control isolation | RFC-0001 / ADR-0002 | server discovery negative tests | verified |
| I-3 deterministic transition | RFC-0001 / SPEC-0002 | deterministic replay corpus | verified serial subset |
| I-4 explicit unknown | RFC-0001 / SPEC-0002 | engine unmodeled/error tests | verified modeled subset |
| I-5 surface binding | ADR-0008 | canonical tool-surface digest tests | verified local fingerprint |
| I-6 atomic transition | SPEC-0002 | rollback/invariant/schema tests | verified |
| I-7 branch isolation | SPEC-0002 | 100-fork test | verified |
| I-8 state oracle | SPEC-0004/0005 | Scenario state assertions/diff | verified scripted subset |
| I-9 secret exclusion | ADR-0009 / SPEC-0020 / SPEC-0044 | sanitizer, decoded JSON sentinel, Task-rule negative and provider-report tests | finite tested patterns; universal secret detection and remote profile open |
| I-10 error preservation | SPEC-0002/0018 | domain/internal/unknown tests | verified implemented paths |
| I-11 immutable Evidence | SPEC-0017/0018 | duplicate/conflict/tamper tests | experimental candidate |
| I-12 bounded execution | SPEC-0015 / SPEC-0022–0024 | semantic limit tests, ResourceProfile digest, CPU/heap governor tests, admission saturation/recovery tests and inspection smoke | accepted local soft-governor subset; no OS hard quota or distributed fairness |
| I-13 fencing | SPEC-0018 | stale-worker/concurrent-claim tests | experimental candidate |
| I-14 external ambiguity | SPEC-0018 | external expiry/failure tests | experimental candidate |
| I-15 scoped compatibility | SPEC-0019 | compatibility validator/matrix | partial; live evidence open |
| I-16 evidence-backed claims | SPEC-0021 | claim registry and release review | implemented as governance |
| ST-VTIME-001 forward-only virtual clock | ADR-0011 / SPEC-0007 | clock control/head/audit tests | verified bounded subset |
| ST-VTIME-002 deterministic modeled entropy | ADR-0026 / SPEC-0026 | spec/engine equal-draw, persistence and rollback tests | verified bounded subset |
| ST-VTIME-003 total signal ordering and lifecycle | ADR-0027 / SPEC-0027 | scheduler ordering/cancel/fork/control HTTP tests | verified bounded subset |
| ST-VTIME-004 atomic bounded due delivery | ADR-0028 / SPEC-0028 | delivery ordering and 257-event rollback tests | verified bounded subset |
| ST-VTIME-005 parsed-time order and per-instant admission | ADR-0029 / SPEC-0029 | sub-second order, capacity and cancellation-release tests | verified bounded subset |
| ST-VTIME-006 bounded next-due recovery | ADR-0030 / SPEC-0030 | preview, two-step legacy drain, empty/CAS and HTTP tests | verified bounded subset |
| ST-VTIME-007 consistent bounded inspection | ADR-0031 / SPEC-0031 | filter/page/stale/malformed cursor tests | verified bounded subset |
| ST-VTIME-008 runtime-bound scheduled TwinSpec actions | ADR-0032 / SPEC-0032 | runtime admission/schema/spec-drift, ordinary-advance refusal, private HTTP and due-time execution tests | verified bounded local-hermetic subset |
| ST-VTIME-009 scheduled terminal evidence and atomicity | ADR-0033 / SPEC-0033 | success/domain-failure/after-effect-fault, call/audit linkage and infrastructure rollback tests | verified one-attempt subset |
| ST-VTIME-010 scheduled action budgets and zero cascade | ADR-0034 / SPEC-0034 | 33-action split, mixed prefix, callback-mutation and oversized-result rollback tests | verified `local-preview-v7` subset |

## Release use

The separate [AgentTask offline contract](SPEC-0038-AGENT-TASK-OFFLINE-ADMISSION.md)
maps AE-001/002/004/005/006/007 and the local path/authority portion of AE-012
to `internal/task`, `internal/evaluator`, `internal/agenteval` and their tests.
AE-003 has six executable authoring witnesses; arbitrary-task solvability and
independent semantic review remain distinct. These are experimental local
results, not admission of full B04–B12 or new stable/live claims.

The subsequent [offline Agent regression contract](SPEC-0039-OFFLINE-AGENT-REGRESSION.md)
adds tested portions of AE-004/008–013/015–019/021–024 through `internal/agenthost`,
`internal/agenteval` and CLI integration tests. AE-014/020 remain limited to
joined in-process work and bounded terminal inspection; general remote drain,
complete OS/disk fault coverage and cleanup recovery are not closed. No full 109-item
source audit, independent semantic review, live profile or external use is claimed.

[SPEC-0040](SPEC-0040-LIVE-PLAN-AND-APPROVAL.md) and
[SPEC-0041](SPEC-0041-PROVIDER-TRANSPORT-AND-EVIDENCE.md) add the local API
readiness portions of AE-009/011–020/023: explicit approval/count caps, fixed
transport, private continuation, failure receipts, non-reusable plan directories
and separate evidence replay. `internal/agentapi`, `internal/agenteval/live_test.go`
and `cmd/statetwin/agent_live_test.go` contain executable contract checks.
AE-023 still lacks actual approved live evidence; AE-014/019/020 retain the
remote/OS fault limitations above. No incomplete work package is closed wholesale.

[SPEC-0042](SPEC-0042-EVIDENCE-STORAGE-AND-TERMINAL-FAILURES.md),
[SPEC-0043](SPEC-0043-READ-ONLY-EVIDENCE-INSPECTION.md) and
[SPEC-0044](SPEC-0044-STRUCTURED-CREDENTIAL-ADMISSION.md) extend the tested local
portions of AE-014/016/018/019/020: 22 filesystem fault injections, five real
subprocess-exit cut points, independent terminal context, preserved first cause,
read-only interrupted-directory diagnosis and pre-write decoded JSON privacy checks.
Tests are in `internal/agenteval/{storage,terminal,inspect}_test.go`,
`internal/logging/sanitize_test.go` and CLI tests. Injected `ENOSPC` is not an
actual full disk, process exit is not power loss, and inspect never authorizes
resume or proves provider origin. General remote drain/reconciliation, all-OS
crash recovery, broad DLP and actual live evidence remain distinct open work.

| Maintenance requirement | Authority | Evidence | Status |
|---|---|---|---|
| ST-MAINT-001 bounded failure-preserving fuzz | SPEC-0035 | workflow contract + POSIX wrapper exit tests + actual fuzz job | verified CI run 33579877698 |
| ST-MAINT-002 dependency migration admission | SPEC-0036 | module/import checks, checksum verification and expression vectors | verified CI run 33579877698; updater run 33579893053 |
| ST-MAINT-003 null/zero distinction | SPEC-0037 | engine null/nested/schema/stored-state regression tests | verified CI run 33579877698 |
| ST-RELEASE-001 reviewed declaration and channel binding | SPEC-0045 | native releasepolicy/releasecheck strict input and notes tests | implemented maintenance subset; not signed human review |
| ST-RELEASE-002 complete candidate gates and draft boundary | SPEC-0046 | workflow dependency, permission, action-pin and flag tests; reusable CI source | implemented; exact-candidate CI and real tag execution are distinct |
| ST-RELEASE-003 preserve owned release output | SPEC-0047 | twelve POSIX fake-command refusal/failure paths; five serial target argument checks | requires Linux/macOS CI; real packaging and publishing not run |

These release-maintenance contracts address the B11/B32 and AE-024 tooling
subset. They do not close B11 stable qualification, B10 external use, actual
provider evidence or missing-source audits by themselves.

A release gate is closed only when the evidence passes on the exact release
candidate revision. `Partial`, `experimental` and `unverified` rows may ship
only when excluded from the stable profile and called out in release notes.
