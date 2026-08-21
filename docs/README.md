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
4. [`RFC-0002-V0.1-RELEASE-PROFILE.md`](RFC-0002-V0.1-RELEASE-PROFILE.md) —
   current release boundary and blockers;
5. [`RELEASE-MANAGEMENT.md`](RELEASE-MANAGEMENT.md) and the root
   [`RELEASE.md`](../RELEASE.md) — maintainer and publication workflow.

The accepted normative documents are the ADRs and SPECs linked from the status
ledger. The large pack below is deliberately preserved as proposal material so
that design work remains reviewable without silently changing the runtime.

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
