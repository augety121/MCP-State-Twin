# MCP State Twin — Master Lifecycle Specification

> **Revision:** Lifecycle Architecture Proposal v2  
> **Status:** Proposal / Unverified  
> **Research cut:** 2026-08-18  
> **Current implementation:** NOT re-tested by this document  
> **Purpose:** One integrated lifecycle specification for the evolution of MCP State Twin from the current development preview through protocol rebaseline, deterministic-runtime completion, fidelity evidence, real Codex/Claude host evaluation, security hardening, stable 1.0 contracts and forward compatibility with increasingly capable agents.
>
> **Truth rule:** This document distinguishes **source-reported current state**, **externally verified facts**, **proposed requirements**, and **future candidates**. A proposal is never a shipped feature; an implemented feature is never “verified” without pinned evidence.

## How to read this master document

This master document intentionally includes the earlier architecture proposal as **Part I**, then adds the lifecycle, host-evaluation, multi-agent, supply-chain and future-agent requirements as **Part II**. If a statement in Part I conflicts with Part II, Part II is the newer proposal and takes precedence for this pack.

The companion segmented documents are implementation-sized views of the same plan. They do not create separate semantics.

---

# Part I — Core architecture and phased feasibility baseline


> **文档状态：Proposal / Architecture Plan**
>
> 本文不是当前实现能力声明，也不修改现有协议或代码。
>
> 目的：定义 MCP State Twin 后续规范体系应该如何扩展、哪些内容必须先做、哪些应该延后，以及每项设计的可实施性与验证门槛。

---

# 1. 总体结论

MCP State Twin 当前最合理的发展路线不是“继续增加模拟功能”，而是把项目逐渐收敛成一个有明确边界的：

> **Evidence-backed deterministic world runtime for stateful agent evaluation.**

它最核心的长期价值应该保持在五个不变量上：

1. **相同环境身份 + 相同有序工具调用 ⇒ 相同可观察结果与最终状态。**
2. **Agent 永远不能访问 Simulation Control Plane。**
3. **每次比较运行必须能够从完全相同的逻辑初始状态开始。**
4. **Fidelity 只能由证据晋级，不能由模型推断或开发者声明自动晋级。**
5. **未知行为必须显式表现为 unknown / unmodeled，而不是“猜一个看起来合理的结果”。**

这与当前项目已经表达出的设计一致：当前 runtime 已有 immutable snapshot、isolated fork、canonical diff、SQLite atomic transitions、bounded CEL、MCP data/control plane separation，并明确把当前 reference twin 标记为 `L1 / unverified / unbound`。

---

# 2. 现在最优先的问题：Protocol Rebaseline

## 2.1 为什么必须先做 Phase 0

当前项目 README 声称设计基线为 MCP `2026-07-28`，但当前 CI conformance 仍然检查 `initialize`、`ping`、`tools-list`，并明确说明这些测试只证明到 `2025-11-25`。

正式 MCP `2026-07-28` 已经发生了结构级变化：

- 删除 `initialize` / `notifications/initialized`；
- 删除 `Mcp-Session-Id` 核心会话模型；
- 每个请求携带 protocol version / capabilities 等 `_meta`；
- 增加 `server/discover`；
- Streamable HTTP 的 2026 模式要求 stateless；
- 引入更正式的 header-based routing；
- list 类响应增加 cache metadata；
- roots / sampling / logging 被正式弃用；
- MRTR 替代旧式 server-initiated interaction；
- tool schema 完整对齐 JSON Schema 2020-12。

所以：

**在 MCP 版本语义没有重新锁定以前，新加 fault、recording、host harness 等 SPEC 会建立在不稳定地基上。**

---

# 3. 建议的规范治理模型

以后每一份 SPEC 都不应该只有一个简单的 `Status: Draft`。

建议至少分成四个正交状态：

| 字段 | 示例 | 含义 |
|---|---|---|
| `spec_status` | draft / accepted / superseded | 文本本身是否成为规范 |
| `implementation_status` | none / partial / complete | 是否存在实现 |
| `verification_status` | unverified / verified | 是否有可重复证据 |
| `release_scope` | v0.1 / v0.2 / future | 属于哪个发行阶段 |

这能避免最常见的一种开源项目问题：

> “规范写完了”被误读成“功能做完了”。

## 每份 SPEC 必须包含

### Header

```text
SPEC:
Title:
Status:
Owners:
Created:
Last-Updated:
Target-Release:
Normative-Dependencies:
Supersedes:
```

### 正文章节

1. Abstract
2. Motivation
3. Goals
4. Non-Goals
5. Terminology
6. Normative Requirements
7. State Machine / Data Model
8. Determinism Requirements
9. Failure Semantics
10. Security & Privacy
11. Resource Limits
12. Compatibility
13. Migration
14. Evidence Requirements
15. Conformance Tests
16. Release Gates
17. Rejected Alternatives
18. Open Questions

每条真正的规范要求应该拥有稳定 ID，例如：

```text
ST-VTIME-R001
ST-VTIME-R002
ST-FAULT-R001
ST-SEC-R014
```

以后测试、Implementation Status、Failure Mode Matrix、Release Checklist 都引用 requirement ID，而不是引用自然语言段落。

这是我认为你现在从“个人工程项目”走向“严谨基础设施项目”最重要的一次规范升级。

---

# 4. 现有 SPEC 家族：不建议推翻

你此前的 README 基线规划了：

| SPEC | 原有职责 |
|---|---|
| SPEC-0001 | TwinSpec Core |
| SPEC-0002 | Runtime Semantics |
| SPEC-0003 | MCP Boundaries & Compatibility |
| SPEC-0004 | Evidence / Fidelity / Release |
| SPEC-0005 | Scenario & Report |
| SPEC-0006 | Host Compatibility & Model Evaluation |

这个拆分本身是合理的。

**建议不是重写编号，而是先 Revision。**

---

# 5. Phase 0 — Normative Rebaseline

目标：

> 让“项目声称支持什么协议”与“代码和测试实际上证明什么”完全一致。

这是所有后续 SPEC 的前置条件。

## 5.1 SPEC-0003 Revision：MCP 2026-07-28

必须明确支持两套 lifecycle profile：

### Legacy profile

```text
<= 2025-11-25
initialize
notifications/initialized
session semantics
```

### Modern profile

```text
2026-07-28
server/discover
per-request protocol metadata
stateless request processing
no initialize lifecycle
```

官方 Go SDK 已经明确区分这两个 lifecycle，并要求实现 `2026-07-28` 的服务器支持 `server/discover`。

### 必须重新审核

- `initialize`
- `initialized`
- `ping`
- session ID
- resumability
- cancellation
- `server/discover`
- per-request `_meta`
- `Mcp-Method`
- `Mcp-Name`
- list cache semantics
- `resultType`
- structured output
- protocol downgrade
- unsupported-version rejection
- backwards compatibility

特别要注意：

当前的：

```text
initialize
ping
tools-list
```

不能再被描述成完整的 `2026-07-28` conformance。

---

# 6. Conformance 策略必须升级

官方 MCP conformance 当前已经可以使用：

```text
--spec-version 2026-07-28
```

并且支持：

- core
- extensions
- backcompat
- auth
- metadata
- expected-failures baseline

并明确要求客户端根据 protocol version 选择 legacy lifecycle 或 2026 stateless lifecycle。

因此 State Twin 不应该写：

> “passes latest MCP conformance”

而应该写成：

```text
conformance-framework:
  repository: modelcontextprotocol/conformance
  commit: <exact commit>
  protocol: 2026-07-28

suite:
  core: PASS
  metadata: PASS
  backcompat: PASS
  auth: NOT_APPLICABLE

expected_failures:
  - ...
```

**必须 pin commit / version。**

不能 pin `main`。

---

# 7. SPEC-0007 — Virtual Time, Entropy & Deterministic Scheduler

这是建议第一个真正新增的 SPEC。

## 7.1 为什么优先级最高

当前 TwinSpec 已经有：

```yaml
clock:
  mode: virtual
  initial: ...
```

而 README 明确说明 virtual-clock advancement 尚未实现。

如果没有真正的 virtual scheduler，就无法严谨模拟：

- retry delay；
- rate-limit reset；
- timeout；
- TTL；
- delayed job；
- eventual consistency；
- lease expiration；
- scheduled webhook；
- backoff；
- deadline。

## 7.2 规范必须覆盖

### 时间模型

区分：

```text
wall_time
monotonic_time
logical_time
```

推荐：

```text
Twin 世界：
wall_time     = virtual
monotonic     = deterministic logical duration
host clock    = forbidden from tool semantics
```

### API

建议 Control Plane：

```text
clock/read
clock/advance
clock/advance-to
```

不应该向 Agent 暴露控制操作。

### 时间排序

两个事件同一时间触发时：

```text
(timestamp, deterministic_sequence_id)
```

作为稳定 tie-break。

绝不能依赖：

- goroutine scheduling；
- map iteration；
- OS scheduler；
- 真实纳秒时间。

### Randomness

后续必须把以下内容纳入 environment identity：

```text
PRNG algorithm
seed
sequence state
```

UUID、随机 token、随机排序都不能直接调用不可控的系统随机源。

## 7.3 可行性

**高。**

这是纯 runtime 内部问题，不依赖外部服务。

现有 CEL 本身适合作为有界表达式引擎，因为 CEL 是 non-Turing-complete、无副作用，并且 cel-go 支持 cost tracking / cost limit。

---

# 8. SPEC-0008 — Deterministic Fault & Failure Injection

该 SPEC 应建立在 SPEC-0007 之后。

不能简单实现：

```text
randomly fail 10%
```

因为这样会破坏 reproducibility。

应该定义：

```text
fault =
  selector
  + trigger
  + phase
  + deterministic outcome
```

## Fault phases

至少覆盖：

```text
before-validation
after-validation
before-effect
after-effect-before-commit
after-commit-before-response
response-delivery
```

## Fault classes

### Domain

- NOT_FOUND
- CONFLICT
- PRECONDITION_FAILED

### Transport

- timeout
- connection reset
- cancellation
- unavailable

### Service

- rate limit
- overload
- temporary unavailable

### Consistency

- stale read
- delayed visibility
- reordered visibility

### Crash

- process crash before transaction
- crash during transaction
- crash immediately after commit
- crash before audit response delivery

## 一个重要设计原则

`partial-effect` **不能通过破坏 SQLite transaction atomicity 来模拟**。

正确方法应该是：

```text
业务世界明确具有：
step A
step B
step C

故障发生在 B 后

事务一次性提交：
A+B 可见
C 不可见
fault result
```

也就是说：

> 模拟的是“上游系统具有部分业务副作用”，而不是“让 State Twin 自己产生数据库半事务”。

---

# 9. SPEC-0009 — Recorder, Redaction & Cassette Replay

对应 Fidelity L0。

当前 README 明确把 recorder / cassette replay / trace redaction 标为未实现。

## Recorder 不应成为核心 runtime 的透明代理

建议结构：

```text
Upstream
   ↑
Recorder Adapter
   ↑
Authorized Test Client
```

而不是：

```text
Agent
 ↓
Twin → secretly proxy production
```

后者违背项目最核心的生产隔离原则。

## Cassette 必须包含

```text
format_version
source_type
recorded_at
tool_surface_digest
request
response
error
sequence
redaction_manifest
provenance
```

## 必须默认排除

- Authorization
- Cookie
- API keys
- bearer tokens
- session tokens
- credentials
- provider-specific secrets

对于未知 header：

**默认 deny，而不是默认记录。**

## L0 的严格定义

Replay 能证明：

> “这条已捕获路径能够重现。”

不能证明：

> “这个 Twin 等价于真实系统。”

---

# 10. SPEC-0010 — Upstream Surface Inspection & Drift

这是现在已经有基础、但还不完整的一块。

当前项目已经实现 canonical MCP tool-surface digest 与 fail-closed binding，但 automatic inspection / refresh 仍未实现。

## Inspector 只负责事实采集

例如：

```text
tool name
description
input schema
output schema
annotations
protocol version
server capabilities
```

然后生成：

```text
surface_digest
```

### 状态

```text
current
drifted
unknown
incompatible
```

## 自动刷新原则

自动检查可以：

```text
detect
report
generate candidate
```

但**不能自动修改 TwinSpec 的行为语义。**

也就是说：

> schema drift 可以机器发现；
>
> semantic drift 不能机器自行认证。

---

# 11. SPEC-0011 — Differential Validation & Fidelity Promotion

这是 MCP State Twin 真正形成技术壁垒的重要 SPEC。

Fidelity 不能只靠：

```text
looks correct
```

应该形成：

```text
Twin
 ↕ same test corpus
Sandbox / Authorized Upstream
```

然后比较：

### Surface

- tool schema
- annotations

### Result

- success/error class
- normalized output

### State

- externally observable state

### Temporal behavior

- visibility timing
- retry semantics

### Idempotency

- duplicate call outcome

## Differential result

不能只有 PASS / FAIL。

建议：

```text
MATCH
ACCEPTED_DIVERGENCE
UNMODELED
UPSTREAM_NONDETERMINISTIC
TEST_INVALID
FAIL
```

## L2 promotion

只有当某个明确 coverage set 满足：

```text
tool
× operation
× success
× error
× state transition
× relevant fault
```

才能将这一覆盖区域标记为 L2。

不能说：

> “整个 GitHub Twin 是 L2。”

应该说：

> “issue.close basic-state-transition coverage is L2 under profile X.”

这比全局 fidelity 标签严谨得多。

---

# 12. SPEC-0012 — Storage, Concurrency, Migration & Crash Recovery

SQLite 当前非常适合 State Twin 的**本地、单机、确定性 runtime**。

SQLite 本身提供事务隔离；WAL 模式允许读者与写者并发，但仍然只有一个 writer。

因此 SPEC 应明确：

## 支持模型

```text
single host
multiple branches
multiple readers
serialized writer semantics
```

## 同一 branch 并发写

必须明确选一种：

### 推荐方案

```text
branch_head_version
```

调用读取：

```text
expected_version = N
```

提交要求：

```text
head_version == N
```

否则：

```text
BRANCH_CONFLICT
```

这比依赖 goroutine 调度顺序更可解释。

## 必须测试

- fork during write
- snapshot during write
- reset during call
- two writes same branch
- writes different branches
- process crash before commit
- process crash after commit
- disk full
- permission failure
- interrupted migration
- foreign database
- future schema version
- corrupted database

## 明确不支持

不要把 SQLite WAL 文件放到网络共享目录并当成分布式数据库使用。

SQLite WAL 本身仍只有单 writer，而且对文件系统/主机语义有约束。

---

# 13. SPEC-0013 — Security & Network Boundary

这份 SPEC 不应该在 v0.1 阶段把项目直接变成公网 SaaS。

它应该先定义安全边界。

## Local Profile

默认：

```text
loopback only
data-plane auth: optional/local
control-plane auth: required
network egress: deny by default
```

## Remote Profile

未来如果启用：

```text
TLS required
authentication required
authorization required
origin validation
host validation
rate limiting
request limits
audit
```

## OAuth

MCP 官方安全要求明确禁止 token passthrough：

MCP Server 接收到的 token 必须是发给该 MCP Server 自己的；如果 MCP Server 再访问上游 API，应获取另一份上游 token，而不是把客户端 token 原样转发。

因此未来 Remote State Twin 必须明确：

```text
Client token
    ↓
State Twin

≠

Upstream token
```

## Threat model 至少覆盖

- confused deputy
- token passthrough
- SSRF
- DNS rebinding
- hostile Origin
- prompt injection
- tool injection
- control-plane discovery
- branch guessing
- snapshot ID guessing
- path traversal
- external `$ref`
- secret leakage
- report leakage
- audit leakage
- oversized payload DoS

---

# 14. SPEC-0014 — Evidence, Audit & Observability Export

State Twin 不应该把 OpenTelemetry 当作 canonical evidence format。

原因是 OpenTelemetry GenAI conventions 仍在持续演进，而且工具参数、工具结果、输入消息等字段可能包含敏感信息或 PII。

正确架构：

```text
Canonical State Twin Evidence
            |
            +---- JSON report
            |
            +---- audit DB
            |
            +---- optional OpenTelemetry exporter
```

而不是：

```text
OpenTelemetry = source of truth
```

## Canonical evidence 必须记录

```text
runtime_version
build_id
spec_digest
surface_digest
snapshot_digest
initial_state_digest
terminal_state_digest
scenario_digest
seed
ordered_call_trace
assertions
canonical_diff
storage_schema_version
protocol_version
```

Provider evaluation 再增加：

```text
provider
host
model
model_version
host_version
tool-policy
approval-policy
timestamp
run_count
```

---

# 15. SPEC-0015 — Resource Governance & DoS Limits

这一份很容易被小项目忽略，但“大厂级”规范必须有。

当前 CEL 已经有 source-size 与 cost limit，这是很好的开始。

后续需要明确：

```text
max_spec_bytes
max_tool_count
max_entity_count
max_state_bytes
max_input_bytes
max_output_bytes
max_schema_depth
max_json_depth
max_diff_entries
max_report_bytes
max_expression_bytes
max_expression_cost
max_calls_per_scenario
max_forks
max_concurrent_calls
max_audit_rows/profile
```

所有超限行为必须：

```text
deterministic
typed
documented
testable
```

不能 OOM 或 hang 以后再交给操作系统处理。

---

# 16. SPEC-0016 — Reproducible Evaluation Bundle

这是建议后期新增、但价值非常高的一份规范。

真正可以分享的 evaluation 不应该是：

```text
“你 clone 这个 repo 然后试试”
```

应该是一个 immutable bundle：

```text
TwinBundle
├── manifest
├── TwinSpec
├── fixture
├── scenarios
├── schema
├── expected digests
├── runtime compatibility
└── evidence policy
```

Manifest：

```yaml
bundleVersion: statetwin.dev/v1alpha1

runtime:
  minVersion: ...

spec:
  digest: sha256:...

fixture:
  digest: sha256:...

scenario:
  digest: sha256:...

protocol:
  mcp: "2026-07-28"
```

这会让：

```text
machine A
machine B
CI
benchmark runner
provider harness
```

能够确认运行的其实是**同一个环境定义**。

---

# 17. SPEC-0006 Revision — Host / Model Evaluation

不建议新建另一份 provider SPEC。

应该扩展原来的 SPEC-0006。

因为：

> MCP protocol compatibility ≠ host compatibility ≠ model behavior compatibility.

Host Profile 应该长这样：

```yaml
host:
  provider: anthropic
  product: messages-api
  profileVersion: ...
  observedAt: ...

transport:
  streamableHTTP: true
  stdio: false

features:
  tools: supported
  resources: unsupported
  prompts: unsupported

evidence:
  runId: ...
  result: ...
```

而不是写：

```text
Claude compatible ✅
```

---

# 18. 后期可选 SPEC：不是近期目标

以下内容应该被**设计考虑，但暂时不要进入 v0.1 critical path**。

## Long-running Tasks / MRTR

MCP `2026-07-28` 已经引入新的 MRTR 模型，并把 Tasks 放入 extension 体系。

State Twin 可以以后支持。

但只有 reference domain 真的需要：

- async jobs
- approvals
- long-running actions

时才值得做。

## Extension / Plugin Runtime

未来可能有人希望：

```text
custom effect
custom evaluator
custom query
```

不要现在开放：

```text
arbitrary Go plugin
```

否则会同时破坏：

- hermeticity
- determinism
- portability
- security

如果以后确实需要扩展：

优先考虑：

```text
declarative built-ins
→ isolated extension protocol
→ sandbox/WASM（再评估）
```

而不是动态加载任意本地代码。

---

# 19. Remote Multi-Tenancy

技术上可行。

但目前不应该做。

原因不是“做不了”，而是它会把项目从：

```text
deterministic evaluation runtime
```

变成：

```text
distributed infrastructure product
```

需要新增：

- tenancy
- RBAC
- persistent identity
- quotas
- encryption
- backup
- scheduler
- worker lifecycle
- multi-node consistency
- HA
- database operations

并且 SQLite 的单 host / single-writer 模型不会自然扩展为大型多租户远程 backend。

因此：

**Multi-tenancy 应该是独立 product decision，而不是自然的 v0.x feature。**

---

# 20. 必须系统考虑的 Edge Cases

任何后续 SPEC 都不能只写 happy path。

## Determinism

必须考虑：

- real clock leak
- timezone
- DST
- timestamp precision
- PRNG
- UUID
- unordered Go maps
- Unicode normalization
- floating-point representation
- `-0`
- NaN / Infinity
- JSON key ordering
- array ordering
- concurrent execution ordering
- retries
- duplicate delivery
- cancellation

## State

- missing entity
- duplicate entity
- broken foreign relation
- large state
- invalid initial fixture
- invariant failure
- mutation rollback
- snapshot during mutation
- reset during mutation

## Concurrency

- parallel calls same branch
- parallel calls different branch
- fork + write
- snapshot + write
- reset + write
- diff while write
- cancelled caller after commit

## Protocol

- invalid protocol version
- downgrade
- unsupported capability
- malformed `_meta`
- missing discover
- legacy initialize
- cache expiration
- duplicate JSON-RPC ID
- unknown method
- malformed tool result
- arbitrary JSON structured output
- cancellation

## Faults

- fail before effect
- fail after effect
- commit but response lost
- retry after ambiguous timeout
- rate limit
- stale read
- delayed visibility
- duplicate request
- process crash
- disk failure

## Security

- malicious tool description
- prompt injection
- SSRF
- credential leakage
- token replay
- token audience mismatch
- path traversal
- external schema reference
- giant JSON
- deep schema recursion
- report secret leakage

## Evidence

- partial report
- interrupted run
- unsupported environment
- changed runtime
- changed Go SDK
- changed spec
- changed fixture
- changed tool surface
- changed model version
- changed host version

---

# 21. 分阶段 Release Plan

## Phase 0 — Protocol Correctness

目标：

**先把协议基线弄对。**

范围：

- MCP 2026-07-28 rebaseline
- SPEC-0003 revision
- versioned lifecycle
- final Go SDK audit
- exact conformance pin
- legacy + modern compatibility suite
- exact claim language

### Exit Gate

必须能明确回答：

```text
Which MCP versions are supported?
Which are actually verified?
By which SDK version?
By which conformance commit?
Which scenarios passed?
```

---

## Phase 1 — Deterministic Runtime Complete

对应首个稳定核心。

实施：

- SPEC-0007 Virtual Time
- deterministic entropy
- scheduler
- SPEC-0008 Fault Injection
- crash killpoints
- SPEC-0012 concurrency / recovery
- SPEC-0015 hard limits
- second independent stateful reference domain

### Exit Gate

同一个：

```text
runtime version
spec digest
fixture
snapshot
seed
ordered calls
```

在支持平台重复运行：

```text
results identical
state digest identical
audit ordering identical
```

---

## Phase 2 — Fidelity & Evidence

实施：

- SPEC-0009 Recorder / Replay
- redaction
- SPEC-0010 Upstream Inspector
- drift detection
- SPEC-0011 Differential Validation
- L0 workflow
- first bounded L2 coverage
- reproducible bundle

### Exit Gate

至少拥有：

```text
one L0 cassette-backed workflow
one evidence-backed L2 coverage subset
```

注意是：

**coverage subset**

而不是声称整个 service 已达到 L2。

---

## Phase 3 — Real Host Evaluation

实施：

- SPEC-0006 revision
- OpenAI host harness
- ChatGPT host profile
- Claude API profile
- Claude Code profile
- repeated-run evaluation
- provider evidence reports
- optional OTel exporter

### Exit Gate

每个“verified host”必须存在：

```text
host version/profile
protocol version
date
model
scenario
environment digest
run evidence
```

不能靠 README 手写兼容性。

---

## Phase 4 — Security-Hardened Remote Mode

只有当项目真的需要远程运行时才开始。

实施：

- TLS
- data-plane auth
- OAuth profile
- audience validation
- RBAC
- SSRF protection
- Origin/DNS rebinding protection
- egress policy
- quotas
- secret management
- threat model
- security tests

### Exit Gate

在完成：

```text
threat model
security test suite
operational runbook
```

之前，不能叫：

```text
Internet-safe
production-ready remote service
```

---

## Phase 5 — 1.0 Stability

重点不是新功能，而是稳定。

要求：

- stable TwinSpec version
- migration policy
- compatibility policy
- deprecation window
- bundle format
- conformance suite
- security policy
- reproducible release
- SBOM / provenance
- upgrade/downgrade story
- documentation freeze criteria

---

# 22. 可行性矩阵

| 能力 | 技术可行性 | 风险 | 判断 |
|---|---|---|---|
| MCP 2026 rebaseline | 高 | SDK / conformance 正在快速演化 | **立即做** |
| Virtual clock | 高 | clock leakage | **v0.1 必做** |
| Deterministic scheduler | 高 | concurrent ordering | **v0.1 必做** |
| Fault injection | 高 | semantic complexity | **v0.1 必做基础版** |
| Crash killpoints | 中高 | OS / storage variance | **v0.1 做受控矩阵** |
| Recorder | 高 | privacy / secrets | **v0.2** |
| Cassette replay | 高 | path-only fidelity | **v0.2** |
| Surface inspector | 高 | semantic inference 不可靠 | **v0.2** |
| Drift detection | 高 | schema normalization | **v0.2** |
| Differential validation | 中高 | upstream nondeterminism | **v0.2 核心** |
| L2 promotion | 中 | coverage proof 较难 | **证据驱动** |
| OpenAI host tests | 高 | provider behavior/version变化 | **v0.3** |
| Anthropic host tests | 高 | connector feature subset | **v0.3** |
| OTel export | 高 | semconv 演化 / PII | **可选** |
| OAuth remote mode | 中高 | security complexity | **v0.4+** |
| Multi-tenancy | 中 | 架构复杂度剧增 | **暂缓** |
| Arbitrary plugins | 中 | 安全/确定性风险高 | **暂缓** |
| MCP Tasks / MRTR | 中高 | 需求尚不明确 | **按领域需求** |
| L3 fidelity | 取决于领域 | 需要真实 reference logic | **不能自动实现** |

---

# 23. 哪些东西永远不能靠 AI 自动完成

这是项目必须写进规范的边界。

AI 可以：

- 建议 TwinSpec
- 推断 candidate transition
- 生成测试候选
- 发现可能 drift
- 总结 differential failures

AI **不能自行证明**：

```text
“This behavior matches production.”
“This twin is L2.”
“This twin is L3.”
“This host is compatible.”
“This implementation is secure.”
```

这些必须来自：

```text
human-reviewed contract
+ executable evidence
+ pinned upstream
+ repeatable verification
```

---

# 24. Claim Policy

项目以后所有公开声明建议只允许以下术语：

### `specified`

规范已经定义。

### `implemented`

存在实现与代码级测试。

### `verified`

存在固定版本、可复现证据。

### `compatible`

存在指定 host/profile/version 的测试证据。

### `L2`

存在限定 coverage 下的 differential evidence。

### `production-ready`

必须另外满足安全、运维、升级、恢复等 release gate。

这样：

```text
implemented ≠ verified
verified ≠ compatible
compatible ≠ equivalent
```

---

# 25. 最重要的非目标

至少在 1.0 之前，不建议 MCP State Twin 变成：

- Agent framework；
- Agent orchestrator；
- model runtime；
- RAG framework；
- hosted SaaS；
- MCP registry；
- secret manager；
- generic workflow engine；
- production reverse proxy；
- 自动复制整个真实 SaaS 的系统。

这些都会稀释它真正最有价值的抽象：

> **可分叉、确定性、可证据化的外部 Agent 世界。**

---

# 26. 最终推荐 SPEC 路线

建议最后形成这样的规范树：

```text
CORE
├── SPEC-0001 TwinSpec Core
├── SPEC-0002 Runtime Semantics
├── SPEC-0003 MCP Protocol Compatibility
├── SPEC-0004 Evidence & Fidelity
├── SPEC-0005 Scenario & Report
└── SPEC-0006 Host & Model Evaluation

DETERMINISM
├── SPEC-0007 Virtual Time & Entropy
└── SPEC-0008 Deterministic Fault Injection

FIDELITY
├── SPEC-0009 Recording & Replay
├── SPEC-0010 Upstream Surface & Drift
└── SPEC-0011 Differential Validation

RUNTIME
├── SPEC-0012 Storage, Concurrency & Recovery
├── SPEC-0013 Security & Network Boundary
├── SPEC-0014 Audit & Observability
└── SPEC-0015 Resource Governance

DISTRIBUTION
└── SPEC-0016 Reproducible Evaluation Bundle
```

再往后的：

```text
Tasks / MRTR
Extension runtime
Remote multi-tenancy
Distributed backend
Registry
Hosted service
```

都暂时不要给正式编号。

先证明需求，再 RFC，再决定是否升为 SPEC。

---

# 27. 下一步执行顺序

**不要先写 SPEC-0007。**

正确顺序应该是：

```text
1. Inventory current docs/code/tests
          ↓
2. MCP 2026-07-28 gap analysis
          ↓
3. Revise SPEC-0003 / ADR-0001
          ↓
4. Pin versioned conformance
          ↓
5. Revise SPEC-0004 claim/evidence rules
          ↓
6. Finalize SPEC-0007 virtual time
          ↓
7. SPEC-0008 fault model
          ↓
8. Storage/crash semantics
          ↓
9. v0.1 release gate
          ↓
10. Recorder / drift / differential fidelity
```

这是目前风险最低、最容易验证、也最符合 MCP State Twin 项目定位的工程路径。

---

# 28. Feasibility Verdict

综合当前代码基础、正式 MCP `2026-07-28`、官方 Go SDK、MCP conformance、CEL、JSON Schema、SQLite、OpenAI/Anthropic MCP 接入形态来看：

## 核心产品方向

**可行性：高。**

因为项目核心依赖的每个基础 primitive 都已经存在成熟技术基础：

- MCP 提供标准 tool surface；
- Go SDK 已支持新协议；
- CEL 适合有界、无副作用规则；
- JSON Schema 2020-12 能提供严格 contract validation；
- SQLite 足以支撑 local deterministic transactional world；
- snapshot/fork/diff 已有工作实现；
- provider 已实际存在 remote MCP tool integration。

## 最大技术风险

不是“做不出来”。

而是：

> **随着 feature 增加，是否还能持续证明 determinism、fidelity 和 claim correctness。**

因此项目真正应该投资的不是越来越多的模拟功能，而是：

```text
specification
evidence
conformance
failure semantics
differential validation
```

如果这五个体系做扎实，MCP State Twin 才会从一个“不错的 MCP mock/runtime 项目”真正变成一个有独立技术定位的 Agent Evaluation Infrastructure。

---

# 29. 资料与验证来源

> 本节用于记录本 Proposal 的外部验证来源。落地为正式 SPEC 前，应进一步固定到具体版本、commit 或发布日期。

- MCP 官方规范与 2026-07-28 版本资料  
  https://modelcontextprotocol.io/
- MCP 官方规范仓库  
  https://github.com/modelcontextprotocol/modelcontextprotocol
- 官方 MCP Go SDK  
  https://github.com/modelcontextprotocol/go-sdk
- MCP Conformance  
  https://github.com/modelcontextprotocol/conformance
- CEL-Go  
  https://github.com/google/cel-go
- SQLite WAL  
  https://www.sqlite.org/wal.html
- OpenTelemetry GenAI Semantic Conventions  
  https://opentelemetry.io/docs/specs/semconv/gen-ai/
- OpenAI MCP / Responses API 文档  
  https://platform.openai.com/docs/
- Anthropic MCP Connector 文档  
  https://platform.claude.com/docs/
- MCP State Twin Repository  
  https://github.com/augety121/MCP-State-Twin

---

# 30. 文档使用说明

本文是**架构规划 Proposal**，不能替代：

- 已接受 RFC；
- 已接受 ADR；
- 正式 SPEC；
- Implementation Status；
- executable tests；
- pinned conformance evidence。

任何未来 README 或 Release 声明都应遵守：

> **Roadmap 不等于 Feature。Specified 不等于 Implemented。Implemented 不等于 Verified。Verified 不等于 Equivalent。**


---

# Part II — Full lifecycle extension

## 31. Project lifecycle model

MCP State Twin should manage each externally observable capability through the following lifecycle:

```text
research
  -> proposal
  -> accepted specification
  -> implementation
  -> code-level validation
  -> evidence verification
  -> release qualification
  -> compatibility monitoring
  -> deprecation
  -> migration
  -> retirement
```

These states MUST remain independent. A feature MAY be specified but not implemented; implemented but unverified; verified under one host profile but not another.

### 31.1 Required lifecycle metadata

Every major specification SHOULD declare:

```yaml
spec_status: draft
implementation_status: none
verification_status: unverified
release_scope: future
owners: []
created: "<date>"
last_updated: "<date>"
normative_dependencies: []
supersedes: []
```

### 31.2 No automatic status promotion

No CI job, AI agent, generated patch, trace parser, differential test or model output may unilaterally promote:
- fidelity level;
- compatibility claim;
- production-readiness claim;
- security claim.

Automation MAY collect evidence and recommend a transition. Promotion remains a governed project decision.

---

## 32. Normative requirement identity

Each future requirement SHOULD use a stable ID so tests and release gates can reference semantics directly.

Recommended namespaces:

```text
ST-GOV-*      governance / claims / lifecycle
ST-MCP-*      MCP protocol compatibility
ST-VTIME-*    virtual time / entropy / scheduler
ST-FAULT-*    deterministic failures
ST-REPLAY-*   recorder / cassette / redaction
ST-SURFACE-*  surface inspection / drift
ST-DIFF-*     differential validation / fidelity
ST-STORE-*    storage / concurrency / recovery
ST-SEC-*      security / trust boundaries
ST-EVID-*     evidence / audit / observability
ST-LIMIT-*    resource governance
ST-BUNDLE-*   evaluation bundles
ST-HOST-*     host evaluation
ST-FUTURE-*   agent-evolution constraints
ST-REL-*      release gates
```

Requirement IDs MUST NOT be reused after retirement.

---

## 33. Master deterministic contract

The earlier contract should be extended from an ordered-call-only identity to a complete semantic profile.

Conceptually:

```text
EnvironmentIdentity =
    runtime_semantic_version
  + TwinSpec_digest
  + canonicalization_id
  + storage_schema_version
  + snapshot_digest
  + world_time_state
  + scheduler_semantics_version
  + entropy_algorithm_and_seed
  + fault_plan_digest
  + protocol_profile
  + scenario_profile
  + world_commit_schedule
```

For a deterministic world profile:

```text
execute(EnvironmentIdentity)
    -> ordered world-observable results
     + terminal state digest
     + canonical audit trace
```

The `world_commit_schedule` qualifier matters for future concurrent or multi-agent evaluation. The project MUST NOT claim that independently running language models will generate identical concurrency or identical tool trajectories.

---

## 34. Agent scheduler vs world scheduler

Future agents may parallelize tasks or spawn subagents.

The specification MUST distinguish:

- **Agent scheduler:** chosen by Codex, Claude, another host, an agent framework or model runtime.
- **World scheduler:** deterministic State Twin mechanism for virtual-time events and modeled background behavior.

MCP State Twin owns the second. It does not own or claim determinism of the first.

For a shared branch, concurrent tool writes require explicit serialization/conflict semantics. Evidence MUST record the actual world commit order.

---

## 35. Host neutrality contract

TwinSpec core MUST NOT depend on:
- Codex configuration syntax;
- Claude Code configuration syntax;
- OpenAI API response item types;
- Anthropic connector beta identifiers;
- a particular model name;
- hidden chain-of-thought;
- provider-private planning state.

Provider/host-specific details belong in **Host Profiles** and adapters.

A new model generation SHOULD require only a Host Profile refresh unless the external-world contract changes.

This is the primary long-term protection against rapid agent evolution.

---

## 36. Host capability lattice

Compatibility MUST be modeled as dimensions rather than a boolean.

At minimum:

```yaml
protocol:
  version: ...
  transport: ...
tools:
  list: supported|unsupported|unknown
  call: supported|unsupported|unknown
  structured_output: ...
discovery:
  server_discover: ...
tool_surface:
  eager: ...
  deferred_or_search: ...
authorization:
  type: ...
approval:
  mode: ...
mrtr:
  supported: ...
tasks_extension:
  supported: ...
concurrency:
  multiple_calls_per_turn: ...
  parallel_calls: ...
output_handling:
  raw_passthrough: ...
  transforms_large_results: ...
retry:
  host_automatic_retry: ...
cancellation:
  behavior: ...
```

A compatibility report MUST state the exact tested subset.

---

## 37. Real-host evidence contract

A live Codex/Claude/API run SHOULD record, where observable:

```text
provider
product
host_version
model_configured_id
resolved_model_snapshot_if_exposed
host_profile_schema_version
host_config_digest
MCP_config_digest
MCP_protocol_version
transport
server_instructions_digest
server_surface_digest
host_visible_surface_digest
allowed/enabled/disabled_tools
approval_policy
timeout_policy
max_turns_or_budget
environment_digest
snapshot_digest
fault_plan_digest
ordered_server_observed_calls
host_visible_results_or_transforms
terminal_state_digest
canonical_diff
completion_status
error_class
wall_clock_metadata
data_handling_profile
```

Secrets MUST NOT be persisted.

Raw server result and host-visible result MUST be distinguishable because hosts may transform, truncate, persist or summarize tool output.

---

## 38. Codex evaluation profile

The first Codex lane SHOULD remain local-first where current host capabilities permit it.

### 38.1 Initial sequence

```text
fresh immutable snapshot
 -> fresh isolated branch
 -> local State Twin Streamable HTTP endpoint
 -> pinned/recorded Codex host profile
 -> verify model-visible business tools
 -> issue bounded objective
 -> capture server-observed calls
 -> compute terminal assertions
 -> archive evidence
 -> destroy/reset branch
```

### 38.2 Test tiers

1. connection and surface discovery
2. read-only world task
3. single deterministic write
4. state-dependent multi-step task
5. conflict / retry task
6. ambiguous commit-result task after deterministic fault support
7. repeated-run outcome distribution
8. parallel/subagent workload only after shared-branch concurrency semantics are stable

### 38.3 Claim language

A passing test may establish:

> “Codex `<host-version/profile>` passed scenario `<id>` against environment `<digest>`.”

It MUST NOT establish:

> “Codex compatible” globally.

---

## 39. Claude evaluation profiles

Claude is not one compatibility target.

### 39.1 Claude Code

Treat as a local/desktop/CLI MCP host profile.

Prefer the same Streamable HTTP data plane used by other hosts to reduce transport confounding. A stdio profile MAY be tested separately.

Record:
- exact Claude Code version;
- MCP config scope/source;
- effective permission policy;
- model selection;
- run bounds;
- host-visible tool surface.

### 39.2 Anthropic Messages MCP connector

Treat as a remote HTTP API host profile.

At the 2026-08-18 research cut, the connector publicly documents tool-call support rather than the entire MCP feature set. Therefore test and claim only the connector subset actually supported by the selected beta/profile.

### 39.3 Anthropic Managed Agents

Treat as a third host profile.

Its MCP permission policy and output transformations are part of evidence. Large-output transformation means raw MCP server output cannot be assumed to be what the model directly receives.

### 39.4 No transitive compatibility

```text
Claude Code PASS
!= Messages MCP connector PASS
!= Managed Agents PASS
```

---

## 40. OpenAI API evaluation profile

Remote MCP through the OpenAI API is separate from local Codex testing.

The harness SHOULD record:
- remote deployment identity;
- model;
- allowed tool filter;
- approval setting;
- MCP list/call event identifiers;
- server errors;
- environment digest;
- terminal world result.

A single model API request may involve multiple MCP tool calls. The evaluation model therefore MUST NOT assume `1 model turn = 1 tool call`.

---

## 41. Repeated-run model evaluation

The world may be deterministic while the agent is not.

For live model evaluation:

```text
N independent trials
same snapshot
same host profile
same objective
same world semantic profile
```

Report:
- terminal-state success rate;
- invariant pass rate;
- unexpected side-effect rate;
- recovery rate after faults;
- tool error rate;
- call-count distribution;
- trajectory diversity;
- approval interruption rate;
- latency/cost when available and meaningful.

Exact call-sequence equality is diagnostic only.

---

## 42. Multi-agent evaluation

Support four conceptual modes.

### A. Isolated comparison
One fork per agent. Preferred benchmark mode.

### B. Cooperative shared world
Agents share one branch. Requires branch-head conflict/serialization semantics.

### C. Adversarial shared world
Used for race/security evaluation. Every commit interleaving is evidence.

### D. Hierarchical
Parent agent delegates to subagents. Evidence includes parent/child identity where host exposes it.

For B–D, evidence SHOULD include:

```text
agent_id
parent_agent_id
host_sequence
request_id
world_commit_sequence
branch_head_before
branch_head_after
```

The runtime can replay recorded world ordering. It cannot prove the host will choose the same ordering on another run.

---

## 43. Long-running operations

Long-running operations SHOULD remain optional.

If a future reference domain needs them:
- use an explicitly negotiated MCP Tasks extension or then-current equivalent;
- persist task handle as explicit state;
- use virtual time for deterministic lifecycle evaluation;
- define cancellation races;
- define `input_required` / MRTR behavior where supported;
- never return a task shape to a host that did not negotiate support.

No v0.1 core behavior depends on Tasks.

---

## 44. Tool-search and deferred-discovery future

As agent hosts scale to larger tool sets, a host may hide/filter/defer tool descriptions.

State Twin SHOULD distinguish:
- canonical full server surface;
- authorization-scoped server surface;
- host-visible surface;
- model-selected surface.

A host hiding a tool does not change the business semantics of that tool.

Tool-surface evidence should therefore contain both server and host perspectives.

---

## 45. Computer-use and GUI boundary

The current product models an MCP tool world.

It does not automatically model:
- browser pixels;
- DOM;
- desktop windows;
- mobile UI;
- robotics;
- physical systems.

If future AGI evaluation requires these, create separate domain adapters and fidelity specifications. Do not stretch TwinSpec claims beyond observable modeled semantics.

---

## 46. Self-modifying / tool-authoring agents

A future agent may propose new tools, state entities or simulation rules.

Such artifacts MUST enter as **candidate specifications**.

They cannot execute as trusted native extensions merely because the agent authored them.

Preferred extension path:

```text
declarative TwinSpec
 -> reviewed built-in primitive
 -> isolated versioned extension protocol
 -> sandboxed extension runtime only after independent security RFC
```

Arbitrary native Go plugin execution remains out of core.

---

## 47. Evidence architecture

Canonical State Twin evidence is project-owned and versioned.

Layers:

### Environment
versions, digests, protocol, scheduler, entropy, fault plan.

### World trace
call, input digest, pre/post state, commit sequence, virtual time, faults.

### Host observation
host/model/config, approvals, filtered tools, transformed result.

### Evaluation
assertions, invariants, terminal diff, score.

Optional OpenTelemetry export MUST be redacted and MUST NOT replace canonical evidence.

---

## 48. Evidence integrity

Every evidence schema version MUST be immutable once released.

Digests MUST declare:
- hash algorithm;
- canonicalization identifier.

Evidence produced by an interrupted run remains a valid artifact with status `incomplete`.

Skipped assertions are not passes.

An artifact signature can prove origin/integrity under its trust policy; it cannot prove fidelity.

---

## 49. Reproducible evaluation bundles

A future `TwinBundle` SHOULD package:

```text
manifest
TwinSpec
fixture
scenarios
schemas
policy
expected digests
runtime compatibility
evidence policy
optional provenance/signature
```

Import must:
- reject path traversal;
- verify component digests;
- reject unsupported versions;
- remain hermetic unless an explicit profile authorizes network retrieval.

This bundle is the portable unit for developer laptops, CI, Codex, Claude and benchmark runners.

---

## 50. Software supply chain

Stable releases SHOULD later add:
- SBOM;
- build provenance;
- reproducible build inputs where practical;
- signed release artifacts;
- optional signed bundles.

SLSA/SPDX/Sigstore can support this lifecycle, but they are not semantic correctness proofs and SHOULD NOT block early deterministic-runtime development.

---

## 51. Security lifecycle

### Local-first
Before remote mode:
- loopback;
- synthetic data;
- no production upstream;
- separate control authentication;
- egress-deny hermetic CI.

### Remote profile
Requires:
- TLS;
- authentication;
- authorization;
- token audience validation;
- no token passthrough;
- SSRF protections;
- Origin/Host controls where applicable;
- rate/body limits;
- credential lifecycle;
- threat model;
- security tests;
- operational runbook.

Host tool approval never substitutes for server authorization.

---

## 52. Recorder and upstream safety

The evaluated agent SHOULD NOT be transparently proxied to production.

Recorder/differential lanes are privileged harness workflows:

```text
Authorized Harness -> Recorder/Differential Adapter -> Explicit test/sandbox upstream
```

Unknown headers default to not recorded.

Redaction occurs before durable persistence.

Production access, if ever permitted, requires an explicit separate profile and authorization.

---

## 53. Resource governance

Stable runtime must bound all attacker/user-controlled growth axes, including:
- spec size;
- tool count;
- schema depth/bytes;
- JSON depth/bytes;
- CEL source/cost;
- state/entity count;
- output size;
- diff size;
- scenario length;
- scheduled events;
- faults;
- forks;
- concurrent calls;
- audit/report size;
- cassette/bundle extraction.

Semantic limits that affect outcomes become part of environment identity.

---

## 54. Storage / concurrency lifecycle

SQLite remains appropriate for the local deterministic profile because it provides transactional isolation and serialized writes.

But explicit semantics are still required:
- branch-head versions;
- typed write conflict;
- snapshot/write races;
- reset/write races;
- crash killpoints;
- migration interruptions;
- disk-full/permission/corruption behavior;
- WAL checkpoint operational policy.

SQLite WAL is not a remote multi-host database. Multi-tenancy requires a separate architecture RFC.

---

## 55. Failure classification

Infrastructure must distinguish:

```text
SPEC_ADMISSION
DOMAIN
PRECONDITION
INVARIANT
UNMODELED
PROTOCOL
AUTHORIZATION
CONCURRENCY
STORAGE
RESOURCE_LIMIT
INJECTED_FAULT
UPSTREAM
HOST
HARNESS
EVIDENCE_INTEGRITY
```

An agent task failure and a broken test harness are not the same result.

---

## 56. Full release lifecycle

### Phase 0 — Protocol truthfulness
MCP 2026-07-28 gap matrix; exact SDK/conformance pins.

### Phase 1 — v0.1 deterministic runtime
Virtual time, entropy, scheduler, bounded faults, concurrency/recovery, critical limits, second domain.

### Phase 2 — v0.2 fidelity/evidence
Recorder/replay, redaction, surface drift, differential L2 subset, evidence schema, bundle alpha.

### Phase 3 — v0.3 real-host matrix
Codex, Claude Code, OpenAI API, Anthropic Messages, optional Managed Agents; repeated-run evidence.

### Phase 4 — remote security, only if justified
TLS/auth/OAuth/SSRF/quotas/security review.

### Phase 5 — v1.0 stability
Stable TwinSpec/evidence/bundle/storage migration, deprecation policy, release provenance.

### Phase 6 — post-1.0
Optional Tasks/MRTR profiles, advanced multi-agent, sandboxed extensions, new world adapters.

---

## 57. Version dimensions

These MUST stay independent:

```text
runtime_semver
TwinSpec_api_version
storage_schema_version
canonicalization_id
evidence_schema_version
scenario_schema_version
bundle_schema_version
MCP_protocol_version
MCP_extension_versions
host_profile_schema_version
host_product_version
model_identifier
```

No evidence artifact is interpretable without the relevant dimensions.

---

## 58. Compatibility freshness

Host compatibility evidence decays.

Every verified host profile SHOULD have:
- observed/tested date;
- exact host version if available;
- exact protocol profile;
- model identifier;
- evidence artifact;
- revalidation policy.

A provider alias or host update may invalidate a compatibility claim without changing State Twin code.

---

## 59. Claim-generation future

Long term, README compatibility/status tables SHOULD be generated or CI-checked from an evidence inventory.

Example:

```yaml
claims:
  - id: mcp-2026-core
    status: verified
    evidence: evidence/conformance/...
  - id: codex-local-http-profile
    status: unverified
```

CI SHOULD reject a public `verified` claim whose evidence reference is missing or stale.

---

## 60. Data governance

Default data class:

```text
synthetic-only
```

Later profiles MAY include:
- authorized test data;
- restricted data.

Every recorder/live-provider run declares a data-handling profile.

Credentials are never canonical evidence.

---

## 61. Performance and scalability

Performance targets SHOULD be release-specific empirical budgets, not universal promises.

Measure:
- tool-transition p50/p95 under fixed corpus;
- fork/snapshot latency;
- diff scaling;
- state-size scaling;
- scheduler scaling;
- memory peak;
- WAL growth/checkpoint behavior;
- evidence overhead.

Performance regressions matter, but deterministic correctness has higher priority.

---

## 62. Migration and deprecation

Stable formats require:
- version reader policy;
- writer policy;
- migration path;
- rollback/recovery;
- evidence invalidation rules.

Pre-1.0 can break faster, but MUST NOT silently reinterpret historical snapshots/evidence.

Deprecation SHOULD define:
- announcement;
- replacement;
- warning window;
- removal target;
- migration guide.

---

## 63. Definition: ready for live Agent evaluation

A scenario is ready for Codex/Claude testing only if:

- immutable starting snapshot exists;
- business tool surface is stable/digested;
- control plane is not agent-visible;
- objective success is terminal-state/invariant based;
- report schema is versioned;
- branch reset/fork is automated;
- run has time/call/budget bounds;
- no production writes;
- host config can be hashed/redacted;
- harness can distinguish task failure from infrastructure failure.

---

## 64. Definition: ready for v1.0

Minimum:

- stable semantic core specs accepted;
- stable TwinSpec API version;
- current protocol conformance evidence;
- migration/recovery evidence;
- deterministic corpus across declared platforms;
- explicit security boundary;
- stable evidence and bundle formats;
- all P0 requirements traceable;
- compatibility rows evidence-backed;
- reproducible release/provenance process;
- deprecation/migration policy.

`1.0` means stable contracts, not “every possible feature”.

---

## 65. Anti-hallucination rules for Twin generation

As AI becomes more involved in Twin authoring:

1. inferred behavior is `candidate`;
2. schema similarity does not prove semantic equivalence;
3. absent upstream evidence remains `unknown`;
4. generated transitions cannot self-promote to L2/L3;
5. unmodeled behavior fails explicitly;
6. provider compatibility is never inferred from another provider;
7. security is never certified by the model that authored the code.

---

## 66. Exhaustiveness without pretending omniscience

It is impossible to enumerate literally every future failure/agent behavior.

The project SHOULD instead maintain:
- a living Failure Mode Matrix;
- requirement IDs;
- closed-world semantics;
- explicit `unknown/unmodeled`;
- future RFC registry;
- source freshness policy;
- compatibility revalidation;
- release blockers for unresolved P0/P1 cases.

This is more rigorous than claiming a finite document covers “all possible worlds”.

---

## 67. Future AGI-facing success criterion

The future-facing success criterion is:

> A materially more capable agent can be connected by adding/updating a Host Profile and adapter while the deterministic external-world contract remains valid.

The project fails provider neutrality if every model generation requires core TwinSpec changes.

---

## 68. Final execution order

When implementation begins:

```text
1. inventory current code/docs/tests
2. create current evidence inventory
3. MCP 2026-07-28 gap matrix
4. revise SPEC-0003 / protocol ADR
5. pin Go SDK + conformance artifacts
6. revise SPEC-0004 claim/evidence rules
7. adopt governance + requirement IDs
8. implement SPEC-0007
9. implement bounded SPEC-0008
10. close SPEC-0012 concurrency/crash gaps
11. close SPEC-0015 critical limits
12. qualify v0.1 candidate
13. implement recorder/drift/differential
14. stabilize evidence/bundle formats
15. only then run live Codex/Claude/provider matrix
16. remote security only after demonstrated need
17. stabilize v1.0 contracts
```

---

## 69. Final feasibility judgment

The core direction remains technically feasible.

The project already reports working primitives for explicit state, transactions, forks, digests, scenarios and MCP exposure. The official MCP ecosystem now provides a dated 2026 protocol, Go SDK support and conformance tooling; current major agent hosts expose MCP integration surfaces.

The dominant long-term risk is not implementation impossibility.

It is **semantic drift and overclaim**:

> as runtime, provider, protocol and agent behavior evolve, can MCP State Twin continue to prove exactly what is deterministic, exactly what is compatible, and exactly what fidelity evidence covers?

The lifecycle architecture in this pack is designed around that risk.

---

# Part III — Universal Agent Compatibility Framework

## 70. Why protocol compatibility is no longer enough

The next architecture layer must explicitly model the fact that an MCP host may transform what the model sees.

The full path becomes:

```text
TwinSpec semantics
      |
      v
Canonical World Runtime
      |
      v
Canonical MCP Server Surface
      |
      v
Authorization-Scoped Surface
      |
      v
Host Projection
      |
      v
Host-Visible / Model-Visible Surface
      |
      v
Agent trajectory
      |
      v
World Commit Trace
      |
      v
Terminal State + Evidence
```

A protocol-conformant MCP server does not prove that two hosts expose an identical effective tool contract.

## 71. Universal compatibility target

The project SHOULD target:

> **Systematic compatibility with heterogeneous agent hosts through versioned Host Profiles, surface projections, isolation profiles, adapters and evidence.**

It MUST NOT target an unscoped statement such as:

> "Compatible with all agents."

Unsupported/unknown remains a legitimate compatibility result.

## 72. Portable Tools Baseline

The broad portable baseline SHOULD require only:

```text
MCP tools
JSON input contracts
typed/structured tool results
explicit business errors
stateful world semantics
terminal-state evaluation
```

The baseline MUST NOT require:
- prompts;
- resources;
- roots;
- elicitation;
- Tasks;
- MRTR;
- Apps/UI;
- provider-specific memory;
- host-specific instruction formats.

These become optional capability profiles.

## 73. Canonical vs projected tool identity

The runtime MUST preserve a stable canonical tool identity even if a host:
- prefixes;
- sanitizes;
- truncates;
- aliases;
- groups;
- filters;
- lazily loads

the tool.

Example:

```text
canonical: close_issue
Gemini-visible example projection: mcp_statetwin_close_issue
```

The projected name is evidence, not canonical identity.

## 74. Schema projection

Every known host schema rewrite MUST be representable as evidence.

```yaml
canonicalSchemaDigest: sha256:...
projectedSchemaDigest: sha256:...
transforms:
  - remove:$schema
  - remove:additionalProperties
lossClass: syntactic
```

Loss classes:

```text
none
syntactic
semantic-risk
unsupported
unknown
```

A `semantic-risk` projection blocks strict fidelity claims until tested.

## 75. Surface layers

The compatibility layer SHOULD distinguish:

```text
S0 TwinSpec canonical surface
S1 MCP-native server surface
S2 authorization-scoped server surface
S3 host-projected surface
S4 model-visible/selected surface
```

Every observable layer SHOULD have a digest.

## 76. Surface readiness / quiescence

Some hosts dynamically discover or progressively load tools.

A live episode MUST NOT start scoring the agent until its required surface-ready policy is satisfied.

Example policy:

```text
target server connected
+ all required tools visible
+ no unresolved discovery error
+ host projection resolved
```

If not satisfied, classify:

```text
HOST_NOT_READY
```

not agent failure.

## 77. Host Adapter SPI

A host adapter exists above the world runtime.

Conceptual interface:

```text
Describe
ProbeCapabilities
ResolveProfile
ProjectSurface
RenderConfiguration
PrepareRun
StartOrAttach
WaitUntilSurfaceReady
InvokeObjective
CollectHostObservation
NormalizeObservation
Cleanup
```

Adapters MAY adapt:
- configuration;
- transport;
- auth;
- approval;
- projection;
- observation.

Adapters MUST NOT change Twin business semantics.

## 78. Compatibility Registry

Recommended registry:

```text
compat/hosts/
  codex/
  claude-code/
  anthropic-messages/
  anthropic-managed-agents/
  openai-responses/
  gemini-cli/
  github-copilot-vscode/
  github-copilot-cloud-agent/
  cursor/
  windsurf/
  cline/
  amazon-q/
  jetbrains-junie/
  zed/
  opencode/
  custom-mcp/
```

Each contains:
- primary source registry;
- profile;
- known limitations;
- projection fixtures;
- adapter tests;
- real evidence when available.

## 79. Host-specific capability transforms

The architecture MUST be prepared for current host behavior such as:
- tool-name namespace/prefixing;
- schema sanitization;
- host tool-count limits;
- host allow/deny filtering;
- approval/no-approval execution;
- progressively loaded tools;
- host-side output transforms;
- MCP forwarded through an editor-agent protocol.

These behaviors belong in Host Profile/evidence.

## 80. Host isolation profile

A coding agent often has more than MCP.

Define:

### `mcp_only`
Only declared MCP business tools plus unavoidable host primitives.

### `declared_mixed`
Built-in shell/filesystem/editor/web tools are enabled and explicitly recorded.

### `uncontrolled`
Significant capabilities cannot be characterized.

Strict cross-host benchmark evidence requires `mcp_only` or a carefully controlled `declared_mixed` profile.

## 81. Evaluator secret isolation

The agent MUST NOT receive:
- control-plane token;
- private expected state;
- private assertions;
- held-out scenario answers;
- evaluator evidence containing answers.

These artifacts must live outside the agent-readable workspace under strict evaluation.

## 82. Workspace identity

Coding-agent behavior is affected by the workspace.

Host evidence SHOULD include:

```text
repo commit
worktree digest
instruction files
skills/plugins/hooks
built-in tools
network policy
host memory/reset state
```

Where instruction files are observable, record digests rather than assuming all agents interpret them the same way.

## 83. Agent memory boundary

A fresh world is not a fresh agent.

Host Profile MUST record:

```yaml
memory:
  freshProcess: true|false|unknown
  previousConversationCleared: true|false|unknown
  persistentMemoryResetGuaranteed: true|false|unknown
```

If memory reset is unknown, repeated trials are not fully independent.

## 84. Benchmark integrity tiers

Scenario secrecy profile:

```text
public-development
validation-held-seed
private-heldout
```

Private evaluator state MUST never be present in filenames, comments, environment variables, tool descriptions or accessible files.

## 85. Execution modes

Every episode MUST identify one of:

```text
blind_eval
diagnostic_eval
coaching
curriculum
stress
```

A coached run cannot be compared directly to a blind run.

## 86. Fidelity vs augmentation

### Fidelity mode
Models the bound upstream contract under evidence-supported coverage.

No extra answer hints.

### Augmentation mode
May provide richer feedback and synthetic coaching designed to improve agent behavior.

Augmentation evidence MUST NOT be used as upstream-equivalence evidence.

## 87. Evaluation Episode

The top-level executable abstraction becomes:

```text
Episode =
  TwinBundle
+ Scenario/Family Instance
+ Host Profile
+ Host Adapter
+ Surface Projection
+ Isolation Profile
+ Execution Mode
+ Starting Snapshot
+ Time/Entropy/Fault Profile
+ Budgets
+ Scoring Profile
```

This Episode abstraction sits above the host-neutral world runtime.

## 88. Episode lifecycle

```text
resolve
 -> validate
 -> compatibility lint
 -> fork world
 -> prepare host
 -> establish isolation
 -> wait surface ready
 -> invoke objective
 -> enforce budgets
 -> finalize
 -> score
 -> archive evidence
 -> cleanup
```

## 89. Capability-uplift curriculum

State Twin can safely improve an agent system's behavior through progressive synthetic worlds.

Recommended curriculum axes:

```text
L0 discover/read
L1 one state mutation
L2 stateful multi-step
L3 constraints/preconditions
L4 recovery/idempotency/ambiguity
L5 virtual time/consistency
L6 concurrency
L7 long-running/multi-agent
L8 adversarial/security
```

These are curriculum levels, not a universal intelligence scale.

## 90. Evaluation dimensions

Do not immediately collapse evaluation into one score.

Track independently:

```text
task success
terminal invariants
unexpected side effects
recovery
safety
efficiency
tool errors
approval interruptions
latency
cost
trajectory diversity
```

A future methodology RFC may define justified composite metrics after empirical data exists.

## 91. Side-effect budget

An episode MAY constrain:

```text
tool calls
writes
deletes
simulated irreversible effects
wall time
virtual duration
```

This lets the harness detect agents that succeed through unnecessary or unsafe actions.

## 92. Scenario families

Replace dependence on a small fixed scenario set with deterministic families.

Identity includes:

```text
family id
generator version
seed
parameter partition
metamorphic transforms
```

Same version+seed MUST generate the same case.

## 93. Metamorphic coverage

Useful transforms include:
- identifier renaming;
- irrelevant state addition;
- irrelevant ordering;
- non-semantic text changes;
- virtual-time origin shift;
- transient fault insertion;
- permission/state partition changes.

Assertions derive from accepted semantics, never model-generated guesses.

## 94. Held-out generation

Scenario families MAY provide:
- public training seeds;
- validation seeds;
- held-out private seeds.

This reduces simple memorization while retaining reproducibility.

## 95. Compatibility lint

A future static command SHOULD be considered:

```text
statetwin compat lint \
  --spec <twin> \
  --targets codex,claude-code,gemini-cli,cursor
```

It would inspect:
- tool counts;
- projected names/collisions;
- schema projection;
- feature dependencies;
- transports;
- auth profile;
- output representation;
- known host limits.

This is a proposed interface, not a current feature.

## 96. Compatibility CI layers

```text
L0 source/profile lint
L1 surface compatibility lint
L2 adapter fixture tests
L3 MCP conformance
L4 real-host connection smoke
L5 real-host stateful scenario
L6 repeated trials
L7 stress/multi-agent
```

Real-host jobs may be scheduled/manual because they require credentials, rate limits and cost.

## 97. Evidence staleness

Compatibility evidence can become stale because of:
- host version;
- model alias/snapshot;
- adapter;
- tool projection;
- protocol;
- approval policy;
- connector beta;
- tool surface;
- State Twin runtime.

Staleness is identity-driven, not merely age-driven.

## 98. Current agent ecosystem strategy

Current major coding-agent products should be treated as separate compatibility targets, including at least:
- Codex;
- OpenAI API remote MCP;
- Claude Code;
- Anthropic Messages MCP connector;
- Anthropic Managed Agents;
- Gemini CLI;
- GitHub Copilot IDE/CLI;
- GitHub Copilot cloud agent;
- Cursor;
- Windsurf Cascade;
- Cline;
- Amazon Q Developer;
- JetBrains AI Assistant / Junie;
- Zed;
- OpenCode;
- custom MCP clients.

The list is an onboarding registry, not a promise that every row is verified.

## 99. Protocol-role separation

Future agent systems increasingly use several interoperability protocols.

Maintain:

```text
MCP = agent/host <-> tool/world
ACP = editor/client <-> coding agent
A2A = agent <-> agent
```

MCP State Twin remains the external world/tool server.

It does not become an ACP agent or an A2A peer by default.

## 100. ACP composition

A valid architecture is:

```text
Editor
  |
  | ACP
  v
Coding Agent
  |
  | MCP
  v
State Twin
```

If the editor forwards MCP configuration over ACP, that forwarding path becomes Host Profile evidence.

## 101. A2A composition

Future remote agent delegation can be:

```text
Agent A
  |
  | A2A
  v
Agent B
  |
  | MCP
  v
State Twin
```

The A2A layer manages agent collaboration; State Twin still owns world semantics.

## 102. Cross-protocol credentials

Credentials MUST NOT inherit automatically across:
- ACP;
- A2A;
- MCP Agent Data Plane;
- State Twin Control Plane.

Every trust boundary is explicit.

## 103. Future non-MCP bridge

If a valuable agent does not support MCP, a provider-function/tool bridge MAY be added.

Rules:
- canonical tool identity preserved;
- projection logged;
- host-specific evidence;
- no MCP conformance claim;
- semantic loss classified.

## 104. Multimodal results

Future tool results may include:
- text;
- image;
- audio;
- files;
- embedded resources/artifacts.

Large binary data SHOULD be represented via content-addressed artifact references rather than placed directly in canonical JSON world state.

Host transformations remain evidence.

## 105. AGI-facing compatibility definition

The project MUST NOT use “AGI compatible” as a verified claim.

The defensible future-facing goal is:

> A new, materially more capable agent can be onboarded through a Host Profile, projection and adapter while the same deterministic external-world semantics continue to hold.

## 106. New required design artifacts

This Part III is implemented as planning documents:

```text
20 Universal Agent Compatibility Architecture
21 Portable MCP Tools Profile
22 Host Adapter SPI & Registry
23 Current Agent Host Matrix
24 Host Isolation & Benchmark Integrity
25 Evaluation Episodes / Curriculum
26 Scenario Families / Metamorphic Coverage
27 MCP / ACP / A2A Boundary
28 Compatibility CI / Freshness
29 Multimodal Artifact Profile
30 Surface Projection / Compat Lint
```

They remain Proposal / Unverified until explicitly adopted and implemented.

## 107. Updated execution order

The implementation sequence becomes:

```text
Protocol truthfulness
  -> deterministic world
  -> faults/storage/limits
  -> canonical evidence
  -> portable MCP tools profile
  -> surface projection model
  -> host adapter SPI
  -> episode/isolation harness
  -> host fixture tests
  -> real Codex/Claude/Gemini/Copilot/... smoke
  -> repeated evaluation
  -> curriculum/stress
  -> optional cross-protocol/multi-agent extensions
```

This preserves rigor while expanding practical agent compatibility.

## 108. Final compatibility invariant

> **State Twin does not make every agent identical. It makes host differences explicit enough that a shared external world can still be tested, compared and evolved without lying about equivalence.**
