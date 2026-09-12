# MCP State Twin Documentation

## Start here: current project, not the future pack

If you are evaluating or contributing to the current repository, read these in
order:

1. [`PROJECT-MAP.md`](PROJECT-MAP.md) — product boundary, architecture and the
   AGI-facing rationale without AGI claims;
2. [`IMPLEMENTATION-STATUS.md`](IMPLEMENTATION-STATUS.md) — the only current
   implementation/evidence ledger;
3. [`DOCS-GOVERNANCE.md`](DOCS-GOVERNANCE.md) — authority, status and claim
   rules;
4. [`REQUIREMENT-TRACEABILITY.md`](REQUIREMENT-TRACEABILITY.md),
   [`CLAIM-REGISTRY.md`](CLAIM-REGISTRY.md), and
   [`COMPATIBILITY-MATRIX.md`](COMPATIBILITY-MATRIX.md) — exact requirement,
   public-claim and host-profile evidence;
5. [`RFC-0002-V0.1-RELEASE-PROFILE.md`](RFC-0002-V0.1-RELEASE-PROFILE.md) —
   current release boundary and blockers;
6. [`ADR-0015-V0.1-SCOPE-AND-FIDELITY.md`](ADR-0015-V0.1-SCOPE-AND-FIDELITY.md)
   — accepted L1-only v0.1 scope and explicit fidelity exclusions;
7. [`ADR-0016-V0.1-STORAGE-COMPATIBILITY.md`](ADR-0016-V0.1-STORAGE-COMPATIBILITY.md)
   and [`ADR-0017-HOST-COMPATIBILITY-REPORT-ADMISSION.md`](ADR-0017-HOST-COMPATIBILITY-REPORT-ADMISSION.md)
   — accepted local storage evidence and host-report admission boundaries;
8. [`STORAGE-COMPATIBILITY-MATRIX.md`](STORAGE-COMPATIBILITY-MATRIX.md) — the
   separate world-store and Episode-Journal schema/migration evidence ledger.
9. [`RELEASE-MANAGEMENT.md`](RELEASE-MANAGEMENT.md) and the root
   [`RELEASE.md`](../RELEASE.md) — maintainer and publication workflow.
10. [`RFC-0003-V0.2-LOCAL-EVALUATION-PLATFORM.md`](RFC-0003-V0.2-LOCAL-EVALUATION-PLATFORM.md),
   [`ADR-0018-TWINBUNDLE-AND-LOCAL-EPISODE-PREVIEW.md`](ADR-0018-TWINBUNDLE-AND-LOCAL-EPISODE-PREVIEW.md),
   [`ADR-0019-DURABLE-LOCAL-EPISODE-JOURNAL.md`](ADR-0019-DURABLE-LOCAL-EPISODE-JOURNAL.md),
   [`SPEC-0017-EPISODE-JOURNAL.md`](SPEC-0017-EPISODE-JOURNAL.md),
   [`ADR-0020-REMOTE-EPISODE-EXECUTION.md`](ADR-0020-REMOTE-EPISODE-EXECUTION.md),
   [`SPEC-0018-REMOTE-EPISODE-COORDINATOR.md`](SPEC-0018-REMOTE-EPISODE-COORDINATOR.md),
   and [`V0.2-REQUIREMENTS.md`](V0.2-REQUIREMENTS.md) — v0.2 proposal,
   accepted artifact/Journal/remote-coordinator subsets, and the
   requirement/evidence ledger.

The accepted normative documents are the ADRs and SPECs linked from the status
ledger. The large pack below is deliberately preserved as proposal material so
that design work remains reviewable without silently changing the runtime.

## Agent regression product proposal — reviewed 2026-09-08

The next product slice is proposed in
[`planning/agent-evaluation/REVIEW.md`](planning/agent-evaluation/REVIEW.md).
Read the [32-chapter master](planning/agent-evaluation/MASTER-SPEC.zh-CN.md),
[phase contracts](planning/agent-evaluation/PHASE-SPECS.md),
[six task cards](planning/agent-evaluation/TASK-CATALOG.md), and
[B01–B32 backlog](planning/agent-evaluation/IMPLEMENTATION-BACKLOG.md).
The [acceptance additions](planning/agent-evaluation/ACCEPTANCE.md) and
[source register](planning/agent-evaluation/SOURCE-REGISTER.md) distinguish
new design requirements from missing source material and existing evidence.

The original review was documentation-only. Subsequent implementation accepts
the [ADR-0038](ADR-0038-AGENT-TASK-AND-OFFLINE-GRADING.md) /
[SPEC-0038](SPEC-0038-AGENT-TASK-OFFLINE-ADMISSION.md) offline Task/grading/witness
subset and [ADR-0039](ADR-0039-OFFLINE-AGENT-REGRESSION-LOOP.md) /
[SPEC-0039](SPEC-0039-OFFLINE-AGENT-REGRESSION.md) synthetic Responses loop,
bounded evidence replay and model-label comparison. Start with the
[offline regression guide](guides/OFFLINE-AGENT-REGRESSION.md).
The separate [ADR-0040](ADR-0040-OPT-IN-LOCAL-API-PLAN.md) /
[SPEC-0040](SPEC-0040-LIVE-PLAN-AND-APPROVAL.md) and
[ADR-0041](ADR-0041-BOUNDED-PROVIDER-TRANSPORT-EVIDENCE.md) /
[SPEC-0041](SPEC-0041-PROVIDER-TRANSPORT-AND-EVIDENCE.md) accept the opt-in local
Responses plan, fixed transport and separate evidence contract. See the
[local API bridge guide](guides/LOCAL-API-BRIDGE.md); only contract tests have
run, not real model trials. Native product hosts and general live configuration
comparison remain unverified/unimplemented. Other proposed CLI commands are
not current quickstart commands. Existing Scenario, Bundle, Journal, release
gates and native-provider security boundaries remain.

The evidence hardening increment accepts three additional bounded contracts:

- [ADR-0042](ADR-0042-EVIDENCE-STORAGE-FAILURE-BOUNDARY.md) /
  [SPEC-0042](SPEC-0042-EVIDENCE-STORAGE-AND-TERMINAL-FAILURES.md): failure-preserving
  publication, independent terminal context, 22 filesystem fault cases and five
  subprocess-exit cut points; not hardware power-loss or all-storage compatibility;
- [ADR-0043](ADR-0043-READ-ONLY-EVIDENCE-INSPECTION.md) /
  [SPEC-0043](SPEC-0043-READ-ONLY-EVIDENCE-INSPECTION.md): read-only `eval inspect`,
  claim/staging consistency and explicit partial/invalid states; never resume or repair;
- [ADR-0044](ADR-0044-STRUCTURED-CREDENTIAL-ADMISSION.md) /
  [SPEC-0044](SPEC-0044-STRUCTURED-CREDENTIAL-ADMISSION.md): decoded JSON credential
  patterns checked before artifact writes, including escaped keys and embedded JSON.

Use the [evidence failure diagnosis guide](guides/EVIDENCE-FAILURE-DIAGNOSIS.md)
when a run leaves claim/closure/pending files. Inspection is not task success,
provider provenance, automatic recovery or an atomic directory snapshot.

## Unified lifecycle adopted on 2026-08-31

ADR-0021 adopts the bounded product definition, independent version dimensions
and release split. Read the phase contracts in order:

1. [`PHASE-00-SPEC-CONSOLIDATION.md`](PHASE-00-SPEC-CONSOLIDATION.md)
2. [`PHASE-01-LOCAL-CORE.md`](PHASE-01-LOCAL-CORE.md)
3. [`PHASE-02-DETERMINISM-RECOVERY.md`](PHASE-02-DETERMINISM-RECOVERY.md)
4. [`PHASE-03-REMOTE-EXECUTION.md`](PHASE-03-REMOTE-EXECUTION.md)
5. [`PHASE-04-PROVIDER-VALIDATION.md`](PHASE-04-PROVIDER-VALIDATION.md)
6. [`PHASE-05-FIDELITY.md`](PHASE-05-FIDELITY.md)
7. [`PHASE-06-SCENARIO-FAMILIES.md`](PHASE-06-SCENARIO-FAMILIES.md)
8. [`PHASE-07-V1-STABILITY.md`](PHASE-07-V1-STABILITY.md)

New cross-cutting contracts are
[`SPEC-0019`](SPEC-0019-HOST-PROFILE-AND-LIVE-EVIDENCE.md),
[`SPEC-0020`](SPEC-0020-REMOTE-SECURITY-PROFILE.md), and
[`SPEC-0021`](SPEC-0021-CLAIM-REGISTRY-AND-FRESHNESS.md). They do not claim that
the planned remote security or live-provider profiles are already implemented.

ADR-0022 and
[`SPEC-0022`](SPEC-0022-LOCAL-CPU-AND-EXECUTION-GOVERNANCE.md) separately
define the accepted conservative local execution policy. `quiet` is the
default and uses one Go scheduler slot; this is explicitly not an OS hard CPU
quota.

The same operational-safety batch also includes
[`SPEC-0023`](SPEC-0023-GO-HEAP-MEMORY-GOVERNANCE.md),
[`SPEC-0024`](SPEC-0024-HTTP-ADMISSION-AND-BACKPRESSURE.md), and
[`SPEC-0025`](SPEC-0025-OPERATIONAL-HEALTH-AND-READINESS.md), with one accepted
ADR per contract. They cover a Go heap soft target, independent listener
backpressure, and authenticated redacted health/readiness.

The Phase 2 deterministic-world batch is defined by
[`SPEC-0026`](SPEC-0026-DETERMINISTIC-ENTROPY-STREAMS.md),
[`SPEC-0027`](SPEC-0027-BRANCH-LOCAL-SIGNAL-SCHEDULER.md), and
[`SPEC-0028`](SPEC-0028-ATOMIC-DUE-SIGNAL-DELIVERY.md) through
[`SPEC-0034`](SPEC-0034-SCHEDULED-ACTION-BUDGET-AND-ZERO-CASCADE.md), with
accepted ADR-0026 through ADR-0034. It implements modeled entropy, private
future signals, bounded progress/inspection and a one-attempt runtime-bound
local TwinSpec-action subset. It does not implement cryptographic randomness,
scheduled Agent/provider/external execution, recurrence/automatic retry or a
distributed workflow queue.

Maintenance hardening is specified by
[`SPEC-0035`](SPEC-0035-BOUNDED-FUZZ-AND-FAILURE-EVIDENCE.md),
[`SPEC-0036`](SPEC-0036-DEPENDENCY-MIGRATION-ADMISSION.md), and
[`SPEC-0037`](SPEC-0037-CEL-NULL-AND-JSON-BOUNDARY.md), accepted through
ADR-0035–0037. These cover count-bounded fuzz evidence, CEL module-path
migration and a reproduced null-to-zero conversion defect. Historical CI
failures and fresh verification are separate records; this batch does not
close provider-live, remote security or fidelity gates.

# MCP State Twin Lifecycle SPEC Pack

> **Status:** Proposal / Unverified
> **Research cut:** 2026-08-18  
> **Default language:** 简体中文  
> **Important:** This pack does not modify the implementation and does not claim that proposed features have been tested.

这是 MCP State Twin 的完整生命周期规范设计包。

## 两层阅读方式

### 一次性看完整设计
先读：

- [`00-MASTER-LIFECYCLE-SPEC.md`](00-MASTER-LIFECYCLE-SPEC.md)

它包含从当前开发预览到：
- MCP 2026 协议重基线；
- 确定性 runtime；
- fault / virtual time；
- fidelity / differential evidence；
- Codex / Claude / API Host Evaluation；
- multi-agent；
- security-hardened remote mode；
- v1.0；
- 未来更强 Agent / AGI-facing infrastructure

的完整设计。

### 分阶段落地
按以下顺序读：

| 顺序 | 文件 | 目的 |
|---:|---|---|
| 1 | `01-SPEC-GOVERNANCE-CLAIMS-VERSIONING.md` | 先规定什么叫 specified / implemented / verified |
| 2 | `02-PHASE-0-MCP-2026-REBASELINE.md` | 先把协议基线弄对 |
| 3 | `03-SPEC-0007-VIRTUAL-TIME-ENTROPY-SCHEDULER.md` | 完整确定性时间模型 |
| 4 | `04-SPEC-0008-DETERMINISTIC-FAULTS.md` | 可重放故障 |
| 5 | `08-SPEC-0012-STORAGE-CONCURRENCY-RECOVERY.md` | 并发/崩溃/迁移 |
| 6 | `11-SPEC-0015-RESOURCE-GOVERNANCE.md` | 资源上限 |
| 7 | `05-SPEC-0009-RECORD-REPLAY-REDACTION.md` | L0 recorder/replay |
| 8 | `06-SPEC-0010-UPSTREAM-SURFACE-DRIFT.md` | 上游 surface drift |
| 9 | `07-SPEC-0011-DIFFERENTIAL-FIDELITY.md` | L2 fidelity |
| 10 | `10-SPEC-0014-EVIDENCE-AUDIT-OBSERVABILITY.md` | 证据体系 |
| 11 | `12-SPEC-0016-REPRODUCIBLE-EVALUATION-BUNDLE.md` | 可移植评测包 |
| 12 | `13-HOST-EVALUATION-CODEX-CLAUDE-OTHER-AGENTS.md` | Codex/Claude/其他 Host |
| 13 | `09-SPEC-0013-SECURITY-NETWORK-BOUNDARY.md` | Remote 安全边界 |
| 14 | `16-RELEASE-LIFECYCLE-AND-GATES.md` | 发布门槛 |

## 非规范但必须持续维护

- `14-MULTI-AGENT-LONG-RUNNING-FUTURE-AGI.md`
- `15-FAILURE-MODE-EDGE-CASE-CATALOG.md`
- `17-SOURCE-REGISTRY.md`
- `18-FEASIBILITY-DEPENDENCY-MATRIX.md`
- `19-OPEN-QUESTIONS-DECISION-GATES.md`

## Templates

- `templates/HOST-PROFILE.yaml`
- `templates/EVIDENCE-MANIFEST.yaml`
- `templates/TWIN-BUNDLE-MANIFEST.yaml`
- `templates/MCP-2026-07-28-GAP-MATRIX.md`
- `templates/REQUIREMENT-TRACEABILITY.md`
- `templates/RELEASE-EVIDENCE-INVENTORY.yaml`

## 采用方式

不要一次性把整个包标成 Accepted。

建议：

```text
Review Master
   ↓
Adopt Governance
   ↓
Run Phase 0 Gap Analysis
   ↓
Revise existing SPEC-0003 / SPEC-0004
   ↓
Accept + implement each new SPEC independently
```

这能保证“写了规范”不会被误认为“已经实现”。


## Universal Agent Compatibility Extension

这轮新增的核心架构：

| 文件 | 目的 |
|---|---|
| `20-UNIVERSAL-AGENT-COMPATIBILITY-ARCHITECTURE.md` | 全 Agent 兼容架构，不用虚假的“all agents compatible” |
| `21-PORTABLE-MCP-TOOLS-PROFILE.md` | 工具优先的最大公约数 MCP Profile |
| `22-HOST-ADAPTER-SPI-AND-REGISTRY.md` | Host Adapter SPI 与兼容性注册表 |
| `23-CURRENT-AGENT-HOST-MATRIX-2026-08.md` | 当前 Agent 官方能力研究矩阵（非测试结果） |
| `24-HOST-ISOLATION-BENCHMARK-INTEGRITY.md` | 防止 shell/files/secret/expected-state 泄漏污染评测 |
| `25-EVALUATION-EPISODES-CURRICULUM-CAPABILITY-UPLIFT.md` | Episode、课程和 Agent 能力提升模式 |
| `26-SCENARIO-FAMILIES-METAMORPHIC-COVERAGE.md` | 确定性生成、held-out 和 metamorphic coverage |
| `27-CROSS-PROTOCOL-ACP-A2A-BOUNDARY.md` | MCP / ACP / A2A 职责边界 |
| `28-COMPATIBILITY-CI-EVIDENCE-FRESHNESS.md` | 兼容性 CI 与证据过期模型 |
| `29-MULTIMODAL-ARTIFACT-OUTPUT-PROFILE.md` | 多模态/Artifact 输出前瞻 Profile |
| `30-PORTABLE-SURFACE-PROJECTION-AND-COMPAT-LINT.md` | Host surface 投影和兼容性 lint |

新增模板：

- `templates/AGENT-COMPATIBILITY-PROFILE.yaml`
- `templates/HOST-SURFACE-PROJECTION.yaml`
- `templates/EVALUATION-EPISODE.yaml`
- `templates/SCENARIO-FAMILY.yaml`
- `templates/COMPATIBILITY-CLAIM.yaml`

### 新的核心原则

```text
Canonical World
  -> Canonical MCP Surface
  -> Authorization Surface
  -> Host Projection
  -> Model-visible Surface
  -> Episode
  -> Evidence
```

兼容性不再是一个布尔值，而是一个可以被检测、投影、测试、过期和重新验证的工程对象。


## Reference Framework & Execution Planning

- `31-REFERENCE-FRAMEWORK-ARCHITECTURE.md` — 七层 reference framework
- `32-UNIVERSAL-COMPATIBILITY-REQUIREMENT-CATALOG.md` — 新兼容层 requirement IDs
- `33-IMPLEMENTATION-WORKSTREAMS-AND-EXIT-CRITERIA.md` — 可并行 Workstreams、依赖和 Exit Criteria

这三份文件把“架构思想”进一步转换成以后可以逐条给 Codex / Claude 实现与验收的工程结构。

- `VNEXT-DELTA.md` — 本轮 Universal Agent Compatibility 架构变化摘要
