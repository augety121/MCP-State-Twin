# Failure Mode Matrix

**Project:** MCP State Twin  
**Date:** 2026-08-17  
**Purpose:** 这是设计输入，不是测试完成报告。每一个 P0/P1 failure mode 在 release 前必须映射到测试、监控或明确的 unsupported 行为。

Severity:

- **P0:** 可能触达真实系统、泄露秘密、破坏 benchmark 完整性或产生错误安全结论。
- **P1:** 可能让结果错误但通常局限于测试环境。
- **P2:** 可用性、性能、调试或兼容问题。

| ID | Area | Failure mode | Severity | Detection | Required behavior / mitigation |
|---|---|---|---|---|---|
| F-001 | Isolation | Hermetic mode 意外把 write 调到 upstream | P0 | network deny + egress audit | 默认无 upstream connector；CI deny egress；任何 live-write code path release gate 禁止 |
| F-002 | Isolation | Agent 看到 snapshot/reset/fault control tools | P0 | `tools/list` snapshot test | 控制面独立 endpoint/auth；默认绝不注册到 agent data plane |
| F-003 | Isolation | Read passthrough 实际触发 upstream side effect | P0 | allowlist + contract metadata + integration fixture | v0.1 hermetic 模式禁 passthrough；以后 read passthrough 也需显式 allowlist |
| F-004 | Secrets | Authorization header 被 recorder 保存 | P0 | secret scanning + golden redaction tests | Authorization/Cookie 等强制 drop，用户配置只能增加 redaction 不能取消内建高风险规则 |
| F-005 | Secrets | Secret 出现在 JSON body | P0 | configurable field policies + secret scanner | trace persistence 前 redact；export 再扫一次；命中高置信 secret 时 fail export |
| F-006 | Privacy | Production PII 被 commit 到 repo | P0 | `.gitignore` template + pre-commit scan | raw traces 默认不适合 source control；推荐 synthetic fixture；提供 sanitize/export 流程 |
| F-007 | Tenant | Branch A 读取 Branch B state | P0 | namespace property tests | 所有 state query 必须显式 branch ID；storage layer enforcement；不可由 agent 参数覆盖 |
| F-008 | Integrity | Agent 通过 tool input 注入 Twin DSL/expression | P0 | parser separation + fuzz | tool input 只作为值，不作为表达式源码执行 |
| F-009 | RCE | TwinSpec native extension 执行恶意代码 | P0 | extension trust metadata | v0.1 declarative only；native adapter 后续独立进程/allowlist/checksum |
| F-010 | Integrity | LLM 自动推断的 transition 被当作 verified | P0 | provenance field validation | inferred 必须显式状态；L2 promotion 需要人审 + tests |
| F-011 | Correctness | 未建模 hidden side effect 被“合理猜测” | P0 | unmodeled coverage markers | 返回 `UNMODELED_BEHAVIOR`；不得生成伪 success |
| F-012 | Correctness | Upstream schema/description 变化，Twin 仍静默使用 | P1 | surface digest | L2 CI 默认 fail `SPEC_DRIFT` |
| F-013 | Correctness | 只 hash schema，漏掉 description 变化 | P1 | canonical surface includes descriptions | description 是 model-facing contract，纳入 digest |
| F-014 | Correctness | Tool 输出 shape 与 upstream 不一致 | P1 | JSON schema + differential tests | validate structured output；差异列入 fidelity report |
| F-015 | Correctness | Upstream 没有 output schema | P1 | runtime metadata | 以 observed/declared schema 标记，不伪称 authoritative |
| F-016 | Correctness | Preconditions 漏规则 | P1 | differential tests + mutation tests | fidelity report 显式 uncovered; 不自动提升 L2 |
| F-017 | Correctness | Global invariant 漏掉 cross-entity 约束 | P1 | property tests | L2 要有 domain-reviewed invariant set；未知处列 limitation |
| F-018 | Correctness | Tool call 失败后留下半状态 | P1 | crash/failure injection | normal mode DB transaction rollback；只有显式 partial fault 可留下 partial state |
| F-019 | Correctness | Postcondition 检查在 commit 之后才失败 | P1 | transition tests | postconditions/invariants 在 commit 前验证 |
| F-020 | Determinism | 使用 wall clock 导致不同结果 | P1 | deterministic replay test | 所有 `now()` 走 VirtualClock |
| F-021 | Determinism | 使用 OS random/UUID | P1 | static lint + replay | deterministic ID provider from seed/branch/call counter |
| F-022 | Determinism | Map iteration 顺序影响 output/hash | P1 | cross-run hash tests | canonical JSON sorting；禁止依赖语言 map 顺序 |
| F-023 | Determinism | 并发 goroutine 调度改变 state order | P1 | race + replay tests | v0.1 state transitions serial scheduler；异步工作显式排序 |
| F-024 | Determinism | Fault injection 依赖真实 sleep | P1 | virtual time tests | fault schedule 绑定 virtual clock/call index |
| F-025 | Determinism | Pagination cursor 每次不同 | P1 | golden test | deterministic cursor derived from branch/state/version |
| F-026 | Snapshot | Fork 修改 parent snapshot | P0 | immutability property tests | snapshots immutable；copy-on-write only child |
| F-027 | Snapshot | Snapshot ID 不绑定 spec version | P1 | metadata validation | digest 必须绑定 TwinSpec/runtime export format |
| F-028 | Snapshot | Import 旧 snapshot 误解释新 schema | P1 | version guard | unknown/newer schema refuse；known old schema explicit migrate |
| F-029 | Snapshot | State diff 因字段顺序产生噪声 | P2 | canonical diff tests | canonical ordering + typed diff |
| F-030 | Snapshot | Branch explosion 占满磁盘 | P2 | storage quota/GC metrics | branch quotas、retention、reachable-delta GC |
| F-031 | Concurrency | 两个 eval run 共用一个 mutable base | P1 | namespace tests | 每 run 从 immutable snapshot fork |
| F-032 | Concurrency | Multi-agent race 被 OS 随机顺序决定 | P1 | scheduler assertion | v0.2+ 必须显式 deterministic interleaving policy |
| F-033 | Idempotency | 重试 create 产生重复对象 | P1 | duplicate-call tests | 只有 TwinSpec 声明 idempotency 时 dedupe；否则模拟真实 non-idempotent 行为 |
| F-034 | Idempotency | Runtime 自作主张认为 tool 幂等 | P1 | schema/spec validation | default = unknown/non-idempotent；禁止 inference promotion |
| F-035 | Time | Eventual-consistency 延迟使用真实时间 | P1 | fake clock tests | pending effects 依赖 virtual clock |
| F-036 | Time | Timezone/DST 导致场景漂移 | P2 | timezone tests | internal UTC; scenario input explicit timezone conversion |
| F-037 | Errors | Timeout after effect 被当作 before effect | P1 | explicit fault class tests | `TIMEOUT_BEFORE_EFFECT` 与 `TIMEOUT_AFTER_EFFECT` 分离 |
| F-038 | Errors | Upstream error 映射丢失关键 fields | P1 | differential error fixtures | canonical internal error + configurable upstream-like renderer |
| F-039 | Errors | Twin 内部 bug 返回业务 4xx | P1 | error taxonomy test | `INTERNAL_TWIN_ERROR` 明确，不伪装 domain failure |
| F-040 | Errors | Unknown behavior 返回 fabricated fake data | P0 | negative tests | fail explicit `UNMODELED_BEHAVIOR` |
| F-041 | MCP | `tools/list` 与真实 surface 不一致 | P1 | surface snapshot/diff | preserve tool-facing descriptor by default |
| F-042 | MCP | Current protocol upgrade break | P1 | official conformance in CI | pin supported MCP versions, upgrade via compatibility ADR |
| F-043 | MCP | 手写 transport 与 spec 偏差 | P1 | dependency choice | 使用官方 Tier-1 SDK，尽量不手写 wire layer |
| F-044 | MCP | ChatGPT 与 Claude 对同一 descriptor 选择不同 tool | P2 | provider smoke/eval matrix | 这是模型行为，不是 twin bug；只保证 surface same，不保证 selection same |
| F-045 | MCP | Claude connector 当前只支持 tool subset | P2 | compatibility matrix | v0.1 tools-first；resources/prompts 不作为跨-provider hard requirement |
| F-046 | MCP | Host 不支持最新 2026 protocol | P2 | negotiated compatibility tests | SDK backward compatibility adapter；文档列出测试过的 protocol matrix |
| F-047 | MCP | Streaming response/cancellation 处理不一致 | P1 | protocol integration tests | runtime cancellation 不得提交未完成 transition；streaming semantics 后续明确 |
| F-048 | MCP | Tool annotations 被误当安全保证 | P0 | design review | annotations 只做 metadata；安全来自 twin isolation/control plane |
| F-049 | Auth | 模拟权限比真实系统宽 | P1 | auth-state scenarios | L2 twin 必须显式 permission model 或声明 `auth fidelity unsupported` |
| F-050 | Auth | OAuth token 进入 fixture | P0 | redaction | 只保存 synthetic principal/scopes，不保存 bearer token |
| F-051 | Auth | 不同 tenant 的 synthetic identity 冲突 | P1 | fixture constraints | principal ID namespaced per scenario |
| F-052 | Recorder | Recording 本身改变请求/响应 timing/behavior | P2 | transparent proxy benchmark | recorder metadata 与 payload 分离；记录其存在，必要时 direct differential validate |
| F-053 | Recorder | 只录 happy path，推断出过度简化模型 | P1 | coverage report | trace coverage 仅显示 observed；不能自动称 L2 |
| F-054 | Recorder | Binary/large payload 把 artifact 撑爆 | P2 | size limits | content-addressed blob + truncation policy + metadata；默认限制 |
| F-055 | Recorder | Streaming payload 不完整 | P1 | completeness marker | incomplete trace 不允许用于 verified output template |
| F-056 | Compiler | Entity key 推断错误 | P1 | human review + differential tests | inferred key 标 `candidate`; duplicate/collision tests |
| F-057 | Compiler | LLM 把 prompt injection 当系统规则 | P0 | untrusted-data boundary | recorder output 永不作为 compiler instruction；结构化 sandbox prompt + human review |
| F-058 | Compiler | 自动生成 DSL 超出资源限制 | P1 | static validator | expression depth/steps/query cardinality limits |
| F-059 | Compiler | Trace 中偶然 correlation 被推断为 causation | P1 | provenance | inferred transition 低信任；需要 counterexample/differential validation |
| F-060 | Compiler | 未观察到的 enum 被错误封闭 | P1 | schema authority precedence | authoritative schema > observed trace；observed enum 不自动当 closed world |
| F-061 | State | Referential integrity 被破坏 | P1 | invariant/DB constraints | declared FK-like constraint checked transactionally |
| F-062 | State | Unique constraint 未模拟 | P1 | invariant tests | TwinSpec schema 支持 unique keys/index constraints |
| F-063 | State | Floating point 导致跨平台 hash 差 | P1 | numeric canonicalization tests | money/precise values 推荐 decimal/fixed-point；canonical number encoding |
| F-064 | State | Locale-dependent sorting/parsing | P1 | locale matrix | internal locale-neutral semantics |
| F-065 | State | Null/missing/empty string 混淆 | P1 | schema property tests | JSON semantics explicit；DSL 区分 missing/null |
| F-066 | State | Large state query OOM | P2 | resource budget | query cardinality/response size limits；indexes |
| F-067 | State | Cyclic references 导出递归爆炸 | P2 | serializer tests | ID references + bounded export traversal |
| F-068 | Scenario | Agent 自己修改 acceptance criteria | P0 | separate runner config | assertions 不暴露为 writable tool data |
| F-069 | Scenario | Agent 读取 hidden expected state 答案 | P0 | control/data separation | hidden assertions/control state 不进入 data plane |
| F-070 | Scenario | Test prompt 包含 snapshot internals | P1 | harness review | provider harness 只给任务/正常工具，不给 oracle |
| F-071 | Scenario | 模型随机性被误算成环境随机性 | P2 | N trials + environment digest | 报告拆分 model variance 与 environment determinism |
| F-072 | Scenario | 终态正确但过程违反显式约束 | P1 | optional trajectory assertions | Scenario 支持 forbidden tools/max calls/policy assertions，但与 state correctness 分开报告 |
| F-073 | Scenario | Exact trajectory assertion 过度约束合法解法 | P2 | review lint | 默认 terminal-state assertion；轨迹只用于必要约束 |
| F-074 | Scenario | Trial 没有 reset 到同一 base | P1 | digest assertion | 每 trial 输出 initial snapshot digest 并校验一致 |
| F-075 | Eval | LLM judge 成为唯一评分 | P1 | architecture rule | 核心 success 必须 deterministic；LLM judge 只能 supplementary |
| F-076 | Eval | 模型升级后 tool behavior drift | P2 | model matrix | 这是项目应用场景；记录 model/provider/version |
| F-077 | Eval | Host 自动注入不同 server instructions | P2 | host metadata log | provider-specific harness 记录可见配置；不宣称行为相同 |
| F-078 | Performance | 大量 snapshots 导致 fork 慢 | P2 | benchmark | delta/COW 优化在 measurement 后做，不预先过度设计 |
| F-079 | Performance | Expression evaluator 被恶意输入拖死 | P1 | step/time budget | bounded evaluator, no unbounded recursion |
| F-080 | Performance | Huge tool result 塞爆 model context | P2 | response limit | 模拟 upstream size，但 runner 提供 hard byte cap/expected error |
| F-081 | Crash | Runtime 在 commit 前 crash | P1 | kill-point tests | transaction rollback；重启恢复到 prior committed head |
| F-082 | Crash | Runtime commit 后 result 未返回 | P1 | crash-point tests | ledger 记录 call ID/outcome；是否重放按 tool idempotency semantics |
| F-083 | Crash | Snapshot metadata 与 DB commit 不一致 | P1 | transactional metadata | state head + metadata 同 transaction 或 WAL-backed recovery |
| F-084 | Upgrade | Runtime 更新改变相同 spec 结果 | P1 | golden replay corpus | semantic change 必须 bump runtime/export compatibility and documented migration |
| F-085 | Upgrade | DSL 新版本重新解释旧 expression | P1 | versioned evaluator | TwinSpec `apiVersion`; old semantics retained or explicit migrate |
| F-086 | Upgrade | Dependency update 改 canonical JSON | P1 | hash golden tests | canonicalization 自己定义、测试，不依赖第三方默认 serializer |
| F-087 | Observability | Log 泄露 secrets | P0 | structured logging/redaction tests | same redaction policy applies to logs/traces |
| F-088 | Observability | State mutation trace 缺失导致不可调试 | P2 | audit completeness tests | commit 记录 changed entities + hashes；large data 可 reference blob |
| F-089 | Observability | Fault schedule 配了但没触发，用户误以为测过 | P1 | run report | 报告 configured faults 与 fired faults 分开 |
| F-090 | Observability | Fidelity status 过期 | P1 | `validatedAt` + drift check | CI 可设置 max validation age；不能隐藏 `UNKNOWN` |
| F-091 | Supply chain | 恶意 TwinSpec 来自第三方 | P0 | signature/trust metadata later | v0.1 本地 review；remote package 未来需 provenance/signature policy |
| F-092 | Supply chain | Native adapter dependency 漏洞 | P1 | SBOM/scanning | native adapters 是代码依赖，走普通 supply-chain controls |
| F-093 | UX | 用户误把 Twin 当生产系统操作 | P1 | endpoint/banner/server metadata | server name/metadata 明确 `SIMULATED`; remote endpoint domain 明确区分 |
| F-094 | UX | 过度宽松 mock 让 agent 看起来虚假高分 | P1 | fidelity report | 默认不自动纠正错误参数；严格 schema/precondition；差异测试 |
| F-095 | UX | 过度严格 twin 比真实系统更难 | P1 | differential tests | 约束来自 declared/observed evidence；记录 deliberate divergence |
| F-096 | Open world | Agent 需要未被 twin 覆盖的外部系统 | P1 | missing tool/error | 明确 fail/unsupported；不要偷偷联网补答案 |
| F-097 | Open world | 多 MCP server 之间共享状态关系缺失 | P1 | scenario validation | v0.1 单 twin/domain；跨 twin composition 后续需 shared world contract |
| F-098 | Open world | 外部事件在真实世界发生，twin 不知道 | P1 | virtual event input | 通过 scenario/event fixture 注入；不宣称 live mirror |
| F-099 | Legal | 录制第三方 SaaS 数据违反条款/隐私政策 | P0 | docs + user responsibility | 推荐 synthetic/test account；不默认 crawl/record arbitrary services |
| F-100 | Product | 试图“一键自动 twin 任意 SaaS”导致不可兑现 | P0 product trust | scope review | marketing 明确 trace-assisted bootstrap + human validation，而非 universal auto-equivalence |

## Extended hazards for the next release profile

These rows are design requirements, not evidence that the current preview has
implemented the corresponding feature.

| ID | Category | Hazard | Priority | Required control |
|---|---|---|---|---|
| F-101 | Parser | YAML alias or deeply nested input exhausts memory | P0 | document/depth/alias limits and fuzz corpus |
| F-102 | Canonicalization | Unicode confusable names create identity collisions | P1 | identifier policy, normalization tests, collision rejection |
| F-103 | Canonicalization | JSON integer/float edge values change digest across runtimes | P1 | supported JSON value domain and golden vectors |
| F-104 | Storage | disk-full or SQLite corruption produces a false success | P0 | fail-closed commit/recovery tests and integrity checks |
| F-105 | Control | bearer token replay or rotation leaves an old operator authorized | P1 | token lifecycle, rotation semantics, and audit coverage |
| F-106 | Control | path traversal reads a fixture or export outside its root | P0 | canonical-path confinement tests |
| F-107 | Runtime | cancellation or client disconnect leaves a live transaction | P1 | context cancellation and kill-point tests |
| F-108 | Runtime | recursive/reentrant tool calls deadlock or exceed limits | P1 | explicit no-reentrancy rule or bounded call depth |
| F-109 | Runtime | collection query creates unbounded CPU or result amplification | P1 | cardinality, output-byte, and cost budgets |
| F-110 | Protocol | backpressure/stream cancellation leaks resources | P1 | transport cancellation and bounded-body tests |
| F-111 | Scheduler | starvation or livelock makes a scenario non-terminating | P1 | step/time/call budgets and termination report |
| F-112 | Audit | log injection or mutable audit rows falsify evidence | P1 | canonical fields, escaping, append-only/tamper checks |
| F-113 | Export | retention/deletion policy leaves sensitive snapshots behind | P0 | deletion tests and documented recovery/retention semantics |
| F-114 | Surface | host injects instructions or schemas not present in the digest | P1 | host metadata capture and compatibility report |
| F-115 | Provider | tools-first success is mistaken for model behavior equivalence | P1 | separate protocol and agent-trajectory claims |
| F-116 | Supply chain | transitive dependency or action tag drifts after review | P1 | immutable pins, SBOM, dependency update review |
| F-117 | Build | binary cannot be reproduced from the declared source/version | P1 | reproducible-build metadata and checksums |
| F-118 | Upgrade | old TwinSpec is silently reinterpreted by a new evaluator | P0 | semantic versioning, golden vectors, explicit migration |
| F-119 | Benchmark | an easy twin inflates agent scores through missing constraints | P1 | fidelity report, negative cases, differential coverage |
| F-120 | Legal | third-party trace or schema is redistributed without permission | P0 | synthetic fixtures by default, consent/license gate |

## Release interpretation

“考虑所有可能性”在工程上不能理解为穷举未来所有未知事故；合理标准是：

1. 建立系统性 failure taxonomy；
2. 对安全/正确性高风险点给出 hard invariant；
3. 对未知行为 fail explicit；
4. P0/P1 有测试或明确 unsupported；
5. 新 incident 必须回填本表与 regression test。

这份矩阵应当持续演进，而不是 v0.1 写完后冻结。
