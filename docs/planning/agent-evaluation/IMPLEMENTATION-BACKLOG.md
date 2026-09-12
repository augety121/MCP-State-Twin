# 实施 Backlog：32 个可立项工作包

**v3.1 / 2026-09-08：提案；没有创建 GitHub Issue/PR。原稿列有 32 个工作包，引用了 109 个不同需求 ID；需求正文未随附，尚不能证明其映射完整或正确。**

原始评审修正与执行顺序见 [REVIEW](REVIEW.md) 和 [PHASE-SPECS](PHASE-SPECS.md)。[补充验收清单](ACCEPTANCE.md) 不替代缺失的原始需求目录。以下工作包正文保留提案范围；后续 ADR-0038–0044 已接受并实现有限离线/API readiness、存储故障/终态、只读证据诊断和结构化隐私子集，实际状态见验收清单和 [实现台账](../../IMPLEMENTATION-STATUS.md)，不能把整个工作包标成完成。开发门槛与 live/发布门槛分别核验。

## 使用规则

B01–B10 是产品主线的小切片顺序；B11 是狭义本地核心发行通道，应与早期设计并行并在声明稳定本地版本前完成。B12 的运行预算约束必须在 B06 中生效，并在 B08 前通过负例验收。不要机械按编号串行执行全部工作包。

每个工作包可拆成多个小 PR，但一个 PR 不应同时改变领域语义、adapter、评分标准和模型比较基线。已有能力的工作主要是接入和补证，不要重写内核。新路径只是建议；若仓库既有目录更合适，接受时记录映射。

实际 Issue 建议补充个人 owner、目标 milestone、阻塞原因、accepted ADR、实际测试命令/函数、evidence 工件及风险。所有当前状态都是待立项，不提供虚构工期或完成率。

## 依赖概览

```text
B01 -> B02 -> B03 ----------------------> B06
         |                                |
         +-> B04 -> B05 -> B12 ---------->+-> B07 implementation
B07 data-policy design ----------------> B06   |
                                                +-> B08 live gate
                                                +-> B09-offline -> B10-offline
B08 -> B09-live -> B10-external
B01 -> B11 (independent local core release)

Farther packages: dependency table below is authoritative.
B32 baseline maintenance starts now; stable retirement depends on B31.
B19/B20/B26/B27 require admitted user needs, not just elapsed milestones.
```

## 工作包总表

| 工作包 | 目标 | 优先级 / 门槛 | 依赖 |
|---|---|---|---|
| B01 | 封存基线与接受产品分线 | P0 · D0 | 无 |
| B02 | 最小 AgentTask 与六任务 admission | P0 · D2 | B01 |
| B03 | 终态与策略评分器 | P0 · D2 | B02 |
| B04 | RunDefinition、fresh run 与 preflight | P0 · D2 | B01;B02 |
| B05 | 单 provider 本地 bridge 的 mock 闭环 | P0 · D2 | B04 |
| B06 | 一个任务的 AgentEpisode 纵向闭环 | P0 · D2 | B03;B05;B12;B07 数据政策设计 |
| B07 | 脱敏、封存、校验与 World replay | P0 · D2/D3A | B06 |
| B08 | 首个实际可用模型的六任务 live 证据 | P0 · D2 | B06;B07;B12 |
| B09 | 两配置重复运行与回归比较 | P0 · D3A | 离线：B07；live 验收：B08 |
| B10 | First-value 和外部试用 | P1 · D3A | 离线：B09-offline；外部 live：B09-live |
| B11 | 狭义本地核心发行门槛 | P0 · D1 | B01 |
| B12 | Episode 级预算与失败分类 | P0 · D2 | B04;B05 |
| B13 | 澄清与上下文隔离 | P1 · D3A/D5 | B09 |
| B14 | 首批 24 任务候选审查 | P1 · D3A | B03;B10 |
| B15 | 机器派生兼容声明与过期 | P1 · D3A/D3B | B01;B09 |
| B16 | 远程主体、分支与工具授权 | P0 · D3B | B11;B15;接受 remote profile |
| B17 | 原生 provider MCP 的独立 live 验证 | P0 · D3B | B16;预算和 API 授权 |
| B18 | 首个有需求的 Codex/Claude Code 类产品宿主 | P2 · D3B/D5 | B10;B15;实际宿主需求 |
| B19 | 需求驱动的时间与故障扩展 | P1 · D5 | B14;具体失败案例 |
| B20 | 领域幂等与查询能力 | P1 · D5 | B14;B19 按需 |
| B21 | 合成 Recorder 和有限 L0 | P0 · D4 | B07;授权数据政策 |
| B22 | 上游 Surface 与 Drift | P1 · D4 | B15;B21 |
| B23 | 独立且可复位的 Reference | P0 · D4 | B14;明确 reference 来源 |
| B24 | 有限差分 Runner | P0 · D4 | B21;B22;B23 |
| B25 | 第一个有限 L2 覆盖子集 | P0 · D4 | B24;B15 |
| B26 | 任务族与变形测试 | P1 · D5 | B14;B09;真实任务需求 |
| B27 | 有限共享世界多 Agent | P2 · D5 | B19;B26;共享世界用户需求 |
| B28 | 引用安全的保留和 GC | P1 · D3A/D6 | B07;B09 |
| B29 | 恢复、迁移与导出验收 | P0 · D6 | B28;B11 |
| B30 | 实测性能与最小运维观察 | P1 · D6 | B07;B11;B28 |
| B31 | 1.0 稳定范围资格审查 | P0 · D6 | B10;B11;B15;B29;B30；所声明 profile 的 gates |
| B32 | 持续维护、弃用和退出 | P1 · D7/D8 | 基础维护立即适用；稳定弃用/退役依赖 B31 |

## B01 — 封存基线与接受产品分线

**优先级 / 门槛：** P0 / D0\
**责任角色：** 产品负责人/维护者\
**前置依赖：** 无\
**当前/建议改动入口：** `docs/IMPLEMENTATION-STATUS.md；docs/ROADMAP.md；docs/VNEXT-TRACEABILITY.md；既有 claim/traceability 入口`

**本包要交付什么。** 将 77d3ea0 和已通过 CI 的 61aac2d 关系写清，列出旧 master 已实现/仍有效/转候选项，接受 local-outbound、Task 独立与新 TracePolicy 的有限决策。

**验收和关键负例。** 每条 current claim 可追溯；摘要不再说已实现 scheduler/fault 子集完全不存在；新需求仍保持 proposal，未执行 live 不变成 compatible。

**回滚与不做什么。** 文档变更与业务语义变更分 PR；拒绝本稿不影响已有 runtime。

**关联需求。** `MST26-GOV-001`, `MST26-GOV-002`, `MST26-GOV-003`, `MST26-GOV-004`, `MST26-GOV-005`, `MST26-GOV-006`, `MST26-PROD-001`, `MST26-TRACE-007`

## B02 — 最小 AgentTask 与六任务 admission

**优先级 / 门槛：** P0 / D2\
**责任角色：** Evaluation owner\
**前置依赖：** B01\
**当前/建议改动入口：** `新增 internal/task；现有 internal/spec 严格解码可复用；新增 examples/*/agent-tasks`

**本包要交付什么。** 只支持一个固定世界、目标、明确授权、所需工具、oracle 引用、预算和单轮运行；写六张任务卡及 witness，不建设自动生成平台。

**验收和关键负例。** 合法 Task 可编译；未知字段、不存在工具、缺信息却要求完成、超限目标被拒绝；旧 Scenario 文件和命令行为不变。

**回滚与不做什么。** 新 schema 用独立 proposal/alpha kind；删掉实验性入口不能破坏旧 Scenario。

**关联需求。** `MST26-TASK-001`, `MST26-TASK-002`, `MST26-TASK-003`, `MST26-TASK-007`

## B03 — 终态与策略评分器

**优先级 / 门槛：** P0 / D2\
**责任角色：** Evaluation owner\
**前置依赖：** B02\
**当前/建议改动入口：** `新增 internal/evaluator；复用现有 world 只读视图；任务 testdata`

**本包要交付什么。** 先支持字段相等/对象计数/允许修改范围和明确事件规则；把 runtime invariant、政策尝试和实际副作用分栏。

**验收和关键负例。** 六任务中正例、漏做、错对象、额外副作用都正确分类；不同合法轨迹均通过；grader 写 world 的负例失败。

**回滚与不做什么。** 不修改成功判据来迎合某模型；旧评分版本封存，变更产生新 digest。

**关联需求。** `MST26-TASK-004`, `MST26-TASK-006`, `MST26-EVAL-001`, `MST26-EVAL-002`, `MST26-EVAL-003`, `MST26-EVAL-008`

## B04 — RunDefinition、fresh run 与 preflight

**优先级 / 门槛：** P0 / D2\
**责任角色：** Adapter owner/Security reviewer\
**前置依赖：** B01;B02\
**当前/建议改动入口：** `新增 Host/Model/Isolation profile 目录；internal/hostcompat；internal/provider`

**本包要交付什么。** 定义模型请求身份、宿主能力、全新会话、工具白名单、secret reference、隔离要求和付费开关；preflight 默认不调用模型。

**验收和关键负例。** 未授权 live、无隔离能力、缺少必需工具/凭据在发送请求前失败；两个试验不继承上次 grader 反馈。

**回滚与不做什么。** 不要把环境变量中的密钥转成 profile JSON；profile 无法严格隔离就拒绝或降级。

**关联需求。** `MST26-HOST-001`, `MST26-HOST-005`, `MST26-SEC-001`, `MST26-SEC-002`, `MST26-SEC-004`, `MST26-SEC-005`, `MST26-PROD-005`, `MST26-EPI-003`

## B05 — 单 provider 本地 bridge 的 mock 闭环

**优先级 / 门槛：** P0 / D2\
**责任角色：** Adapter owner\
**前置依赖：** B04\
**当前/建议改动入口：** `internal/provider；internal/server；新增本地 tool bridge 与投影 fixtures`

**本包要交付什么。** 从授权 MCP surface 生成有版本的函数工具投影，接收 tool call，按明示顺序本地执行并回传；先使用合成响应 fixtures。

**验收和关键负例。** 一调用、多调用、未知工具、Schema 损失、重复 ID 不同输入、错误结果映射均覆盖；两个 surface digest 和 call mapping 可检查。

**回滚与不做什么。** 只支持最小无损 Schema 子集；不静默删约束，不新增通用插件 registry。

**关联需求。** `MST26-HOST-002`, `MST26-HOST-003`, `MST26-HOST-006`, `MST26-HOST-007`, `MST26-HOST-008`, `MST26-MCP-005`, `MST26-OPS-007`

## B06 — 一个任务的 AgentEpisode 纵向闭环

**优先级 / 门槛：** P0 / D2\
**责任角色：** Episode/Runtime owner\
**前置依赖：** B03;B05;B12；B07 数据政策先设计、完整封存随后集成\
**当前/建议改动入口：** `internal/episode；internal/provider；internal/server；新增 evaluator/evidence 联结`

**本包要交付什么。** 将任务、branch、host loop、世界审计、固定截止终态、评分与 cleanup 接起来；首例为指定 issue 关闭及提交后响应不确定变体。

**验收和关键负例。** mock Agent 自主轨迹能得分；已经提交但未交付被准确解释；取消/超时和晚到请求不产生封存之外的未说明状态。

**回滚与不做什么。** 先形成受限单任务实验入口；不等待完整 cohort、UI 或远程 coordinator 才展示结果。

**关联需求。** `MST26-EPI-001`, `MST26-EPI-002`, `MST26-EPI-003`, `MST26-EPI-004`, `MST26-EPI-005`, `MST26-EPI-008`, `MST26-TRACE-001`, `MST26-TRACE-002`, `MST26-TRACE-004`, `MST26-TRACE-005`, `MST26-CORE-004`, `MST26-FAULT-001`, `MST26-FAULT-002`, `MST26-FAULT-003`, `MST26-FAULT-006`

## B07 — 脱敏、封存、校验与 World replay

**优先级 / 门槛：** P0 / D2/D3A\
**责任角色：** Evidence owner/Security reviewer\
**前置依赖：** B06\
**当前/建议改动入口：** `新增 internal/evidence；复用 internal/canonical、internal/bundle；证据 fixtures`

**本包要交付什么。** 实现独立 synthetic trace profile，内容先白名单/扫描再持久化；绑定组件、序列、head、结果；提供离线 verify/replay。

**验收和关键负例。** 秘密哨兵不出现在临时/最终/错误文件；篡改终态、删事件、替换 profile 被发现；digest-only 明确不可完整 replay。

**回滚与不做什么。** 不改变 ProviderSmokeReport 无原始内容合约；不能重放的 run 只保留明确 partial 证据。

**关联需求。** `MST26-TRACE-001`, `MST26-TRACE-002`, `MST26-TRACE-003`, `MST26-TRACE-004`, `MST26-TRACE-005`, `MST26-TRACE-006`, `MST26-TRACE-007`, `MST26-TRACE-008`, `MST26-SEC-002`, `MST26-SEC-007`, `MST26-SEC-008`, `MST26-EPI-004`

## B08 — 首个实际可用模型的六任务 live 证据

**优先级 / 门槛：** P0 / D2\
**责任角色：** Adapter owner/预算批准者\
**前置依赖：** B06;B07;B12\
**当前/建议改动入口：** `internal/provider；opt-in live workflow；候选 ModelProfile`

**本包要交付什么。** 使用维护者明确批准且可访问的模型 profile 验证链路；不预设模型 ID、账号权限、费用或成功率。所有 live 均绑定预算和数据声明。API 文档可行不等于项目 live 通过。

**验收和关键负例。** 存在实际请求身份、工具轨迹、终态与任务结果；失败也存档；不宣称模型全过，不从 API 推断 ChatGPT/Codex 产品兼容。

**回滚与不做什么。** 没有权限/预算就止于 contract-tested；不借用他人密钥、不自动扩容试验。

**关联需求。** `MST26-HOST-009`, `MST26-PROD-005`, `MST26-EPI-003`, `MST26-OPS-007`, `MST26-FID-001`

## B09 — 两配置重复运行与回归比较

**优先级 / 门槛：** P0 / D3A\
**责任角色：** Evaluation owner\
**前置依赖：** 离线开发 B07；真实两配置验收 B08\
**当前/建议改动入口：** `新增 internal/evaluation；internal/episode；compare/report CLI`

**本包要交付什么。** 封存 plan，按任务/seed/trial 建 cohort；明确改动维度，输出计划与有效分母、风险、成本和 inconclusive。

**验收和关键负例。** 不同 Task/oracle 或未解释预算变化会判不可比；失败不能被选择性删除；给定封存工件 compare 输出稳定。

**回滚与不做什么。** 先 JSON/Markdown 输出；不做全网模型排名，不对小样本做过强统计断言。

**关联需求。** `MST26-EVAL-004`, `MST26-EVAL-005`, `MST26-EVAL-006`, `MST26-EVAL-007`, `MST26-EPI-006`, `MST26-EPI-009`, `MST26-CORE-008`

## B10 — First-value 和外部试用

**优先级 / 门槛：** P1 / D3A\
**责任角色：** 产品负责人/文档 owner\
**前置依赖：** 离线指南依赖 B09-offline；外部 live 对比依赖 B09-live\
**当前/建议改动入口：** `cmd/statetwin；README 多语言；新增上手与报告说明`

**本包要交付什么。** 将离线流程和 live 流程分开；减少手工步骤；报告先解释是否可比及失败归因；收集三个独立外部反馈。

**验收和关键负例。** 干净环境按文档可完成；用户能指出目标是否完成和哪个副作用违规；有可复核的升级/修复/范围收缩反馈。

**回滚与不做什么。** 不要为演示掩盖错误或代用户默默修配置；不先建设前端服务和账号体系。

**关联需求。** `MST26-PROD-002`, `MST26-PROD-003`, `MST26-PROD-004`, `MST26-GOV-005`, `MST26-TRACE-008`, `MST26-EVAL-007`

## B11 — 狭义本地核心发行门槛

**优先级 / 门槛：** P0 / D1\
**责任角色：** Runtime/Release owner\
**前置依赖：** B01\
**当前/建议改动入口：** `现有 engine/store/world/spec/server 测试；.github/workflows/ci.yml；发行清单`

**本包要交付什么。** 复验当前 bounded 能力和已有存储/协议矩阵，固定 exact candidate，明确实验性 remote/provider 标签，完成本地核心 release checklist。

**验收和关键负例。** 本地合约、核心 golden、安全负例和支持平台通过；发布工件/文档同一 SHA；不要求 live provider 才发布 v0.1 本地核心。

**回滚与不做什么。** 可与 B02-B05 设计并行；必须先于 stable-local 声明，不能因首轮产品顺序而遗漏。

**关联需求。** `MST26-CORE-001`, `MST26-CORE-002`, `MST26-CORE-003`, `MST26-CORE-005`, `MST26-CORE-006`, `MST26-CORE-007`, `MST26-MCP-001`, `MST26-MCP-002`, `MST26-MCP-004`, `MST26-OPS-002`, `MST26-OPS-006`, `MST26-REL-001`, `MST26-GOV-005`, `MST26-SEC-008`

## B12 — Episode 级预算与失败分类

**优先级 / 门槛：** P0 / D2\
**责任角色：** Runtime/Adapter owner\
**前置依赖：** B04;B05\
**当前/建议改动入口：** `internal/limits；internal/governor；internal/provider；episode deadline`

**本包要交付什么。** 把轮次、工具、wall-time、输出和费用策略放到整次运行，区分 domain/resource/provider/infra；记录 unknown usage。

**验收和关键负例。** 连续多请求累计超限会停止；未知费用不归零；外部接受未知不自动重试；超限仍有安全 partial report。

**回滚与不做什么。** 不是仅加一个 http.Client timeout；上限变化需要 profile/digest 更新。

**关联需求。** `MST26-HOST-009`, `MST26-CORE-005`, `MST26-SEC-004`, `MST26-OPS-007`, `MST26-TRACE-005`

## B13 — 澄清与上下文隔离

**优先级 / 门槛：** P1 / D3A/D5\
**责任角色：** Evaluation/Adapter owner\
**前置依赖：** B09\
**当前/建议改动入口：** `AgentTask interaction；HostProfile context policy；合成用户回应 fixtures`

**本包要交付什么。** 先加入有界确定性回应表和明确 reset 策略，再考虑目标 revision；区分 blind/diagnostic/coaching。

**验收和关键负例。** 缺信息任务通过澄清可解，超过轮次合法停止；诊断反馈不会混进 blind 基线；无 checkpoint 不标相同 Agent 状态。

**回滚与不做什么。** 不先引入模型用户模拟器；后者需单独随机性/成本/评分设计。

**关联需求。** `MST26-TASK-005`, `MST26-TASK-008`, `MST26-TASK-009`, `MST26-HOST-005`, `MST26-EPI-007`

## B14 — 首批 24 任务候选审查

**优先级 / 门槛：** P1 / D3A\
**责任角色：** Task/Domain owner\
**前置依赖：** B03;B10\
**当前/建议改动入口：** `examples/issue-tracker；examples/package-registry；任务与 oracle fixtures`

**本包要交付什么。** 按照 SCENARIO-CATALOG 逐项做 schema、fixture、可观察性和 oracle 审查；只发布 admitted 子集。

**验收和关键负例。** 所有候选带依赖分类和可解性结论；缺工具的任务 blocked；不会把当前无评论查询误当模型能力问题。

**回滚与不做什么。** 域扩展走单独版本，不让任务数目标逼出伪造 API 或隐式行为变更。

**关联需求。** `MST26-TASK-003`, `MST26-TASK-007`, `MST26-FAULT-003`, `MST26-CORE-007`, `MST26-EVAL-003`

## B15 — 机器派生兼容声明与过期

**优先级 / 门槛：** P1 / D3A/D3B\
**责任角色：** Compatibility owner\
**前置依赖：** B01;B09\
**当前/建议改动入口：** `internal/hostcompat；既有 compatibility/claim 清单与 CI`

**本包要交付什么。** 为接入路径、模型/宿主配置、日期、surface、证据和 TTL 定义 claim；当前表由其生成或校验。

**验收和关键负例。** mock 不晋级 live；local 不晋级 native；过期保留历史且失去 current；缺 artifact 的 verified CI 失败。

**回滚与不做什么。** 不用人工手写更多模型绿色徽章；删除公开错误声明要有修订记录。

**关联需求。** `MST26-HOST-010`, `MST26-GOV-002`, `MST26-GOV-003`, `MST26-MCP-001`, `MST26-MCP-003`, `MST26-REL-006`

## B16 — 远程主体、分支与工具授权

**优先级 / 门槛：** P0 / D3B\
**责任角色：** Security/Server owner\
**前置依赖：** B11;B15;接受 remote profile\
**当前/建议改动入口：** `internal/server；现有 SPEC-0020 对应接受文档；网关/部署配置`

**本包要交付什么。** 实现受众、过期、episode/branch/tool/attempt 绑定；TLS/受支持隧道；私有 control；销毁和审计。

**验收和关键负例。** 错误分支、过期/错误受众、stale attempt、未授权 tool/control 全拒绝；不泄露其他分支存在性。

**回滚与不做什么。** 只提供 synthetic-staging，不承诺多租户生产；回滚要撤销全部临时凭据。

**关联需求。** `MST26-SEC-001`, `MST26-SEC-002`, `MST26-SEC-003`, `MST26-SEC-004`, `MST26-SEC-005`, `MST26-SEC-006`, `MST26-SEC-007`, `MST26-HOST-004`, `MST26-MCP-005`

## B17 — 原生 provider MCP 的独立 live 验证

**优先级 / 门槛：** P0 / D3B\
**责任角色：** Adapter/Compatibility owner\
**前置依赖：** B16;预算和 API 授权\
**当前/建议改动入口：** `internal/provider；provider-smoke workflow；native HostProfiles`

**本包要交付什么。** 按实际 API/profile 执行成功、错误、取消或明确不支持路径，捕获原生工具面与终态证据。

**验收和关键负例。** OpenAI/Anthropic 各有自己声明；缺席一家不抹成全兼容；传输成功不替代任务评分。

**回滚与不做什么。** 不能用 local bridge 报告代替；credential/endpoint 缺失时 gate 维持 blocked。

**关联需求。** `MST26-HOST-004`, `MST26-HOST-008`, `MST26-HOST-010`, `MST26-MCP-003`, `MST26-SEC-006`, `MST26-EPI-008`

## B18 — 首个有需求的 Codex/Claude Code 类产品宿主

**优先级 / 门槛：** P2 / D3B/D5\
**责任角色：** Product-host adapter owner\
**前置依赖：** B10;B15;实际宿主需求\
**当前/建议改动入口：** `新增产品宿主 profiles/adapter；受控工作区与启动封装`

**本包要交付什么。** 只选择一个有明确需求的产品，记录版本、权限、记忆、工作区、工具投影和额外能力。

**验收和关键负例。** 宿主能读 shell/文件时隔离得到验证；API 成功不作产品证据；无法观察最终 surface 时标 unknown。

**回滚与不做什么。** 不同时承诺所有 IDE/CLI；不支持隔离的配置降级为 trusted 或不参与严格评测。

**关联需求。** `MST26-HOST-001`, `MST26-HOST-004`, `MST26-HOST-005`, `MST26-HOST-006`, `MST26-HOST-007`, `MST26-HOST-008`, `MST26-TASK-005`, `MST26-SEC-001`, `MST26-SEC-005`, `MST26-EPI-007`

## B19 — 需求驱动的时间与故障扩展

**优先级 / 门槛：** P1 / D5\
**责任角色：** Runtime/Task owner\
**前置依赖：** B14;具体失败案例\
**当前/建议改动入口：** `internal/engine；internal/store；fault/scheduler tests；相关领域 Task`

**本包要交付什么。** 只实现一个被实际任务需要的模式，如确定性可见性延迟；定义触发、原子性、观察和回放。

**验收和关键负例。** 相同事件输入跨重复运行一致；未支持 mode 拒绝；故障不依赖真实 sleep 或隐藏随机。

**回滚与不做什么。** 不为覆盖表全面铺开 crash/recurrence/DLQ；保留已有两阶段行为。

**关联需求。** `MST26-FAULT-001`, `MST26-FAULT-002`, `MST26-FAULT-003`, `MST26-FAULT-005`, `MST26-FAULT-006`, `MST26-CORE-004`

## B20 — 领域幂等与查询能力

**优先级 / 门槛：** P1 / D5\
**责任角色：** Domain owner\
**前置依赖：** B14;B19 按需\
**当前/建议改动入口：** `reference TwinSpec 新版本；领域幂等 fixtures 和 Task`

**本包要交付什么。** 根据任务需要加入明确的幂等键/结果查询或评论查询，定义相同 key 不同 payload、作用域和期限。

**验收和关键负例。** 重复请求不产生未声明副作用；冲突输入拒绝；旧 close_issue CONFLICT profile 不被暗改。

**回滚与不做什么。** 不拿 MCP request ID 当业务幂等键；新域版本独立比较。

**关联需求。** `MST26-FAULT-003`, `MST26-FAULT-004`, `MST26-TASK-007`

## B21 — 合成 Recorder 和有限 L0

**优先级 / 门槛：** P0 / D4\
**责任角色：** Evidence/Fidelity owner\
**前置依赖：** B07;授权数据政策\
**当前/建议改动入口：** `新增 recorder/cassette；复用 canonical/bundle 安全 admission`

**本包要交付什么。** 先只对授权合成 reference 采集，落盘前白名单/扫描，定义 cassette 格式和 exact-path coverage。

**验收和关键负例。** 未知路径拒绝而非编造结果；secret/temp/error 全不泄露；录制结果可按既定路径重放。

**回滚与不做什么。** 不接生产 passthrough；无隐私证据不扩到真实测试数据。

**关联需求。** `MST26-FID-003`, `MST26-TRACE-003`, `MST26-TRACE-007`

## B22 — 上游 Surface 与 Drift

**优先级 / 门槛：** P1 / D4\
**责任角色：** Fidelity owner\
**前置依赖：** B15;B21\
**当前/建议改动入口：** `新增 upstream surface inspector；已有 spec surface digest/admission`

**本包要交付什么。** 固定 source、日期、授权范围，采集 surface 并比较变更；只产差异和候选。

**验收和关键负例。** Schema/描述/annotation 变化可被捕获；取不到为 unknown；不自动改写 Twin 业务规则。

**回滚与不做什么。** 不要把 surface 相同宣传为语义等价；旧 claim 可 stale 但不删历史。

**关联需求。** `MST26-FID-002`, `MST26-FID-008`

## B23 — 独立且可复位的 Reference

**优先级 / 门槛：** P0 / D4\
**责任角色：** Domain/Fidelity owner\
**前置依赖：** B14;明确 reference 来源\
**当前/建议改动入口：** `新增独立 reference 测试服务或授权 sandbox profile；领域 corpus`

**本包要交付什么。** 挑一个有限域，定义 reset、可观察状态和授权工具范围，不复用被测 CEL 转换作为唯一实现。

**验收和关键负例。** 每个测试都有可复位等价初态；明确独立性；reference 异常不会计为 Twin 必然错误。

**回滚与不做什么。** 仅证明对该 reference 的一致；没有真实上游证据不声称真实服务等价。

**关联需求。** `MST26-FID-007`, `MST26-TASK-003`, `MST26-EVAL-003`

## B24 — 有限差分 Runner

**优先级 / 门槛：** P0 / D4\
**责任角色：** Fidelity owner\
**前置依赖：** B21;B22;B23\
**当前/建议改动入口：** `新增 differential runner/comparator；配对结果 artifacts`

**本包要交付什么。** 在等价初态执行相同序列，比较结果、错误、终态及相关重复/时间语义；规范化策略版本化。

**验收和关键负例。** 至少一条真实 mismatch 可定位；关键偏差不能被配置忽略；不可控上游分类而非强判 MATCH。

**回滚与不做什么。** 不以高通过率目标反向删失败用例；accepted divergence 人工审查。

**关联需求。** `MST26-FID-004`, `MST26-FID-005`

## B25 — 第一个有限 L2 覆盖子集

**优先级 / 门槛：** P0 / D4\
**责任角色：** Fidelity reviewer/维护者\
**前置依赖：** B24;B15\
**当前/建议改动入口：** `coverage report；claim admission；文档与 release profile`

**本包要交付什么。** 发布 coverage 分母、通过/失败/未建模/偏差与来源，人工接受具体工具/状态/错误/fault 范围。

**验收和关键负例。** 未覆盖操作保持 L1/unknown；drift 能撤销 current；签名和模型成功都不替代差分。

**回滚与不做什么。** 不发布整个 GitHub/包平台 L2 徽章；保持历史来源和 scope。

**关联需求。** `MST26-FID-001`, `MST26-FID-006`, `MST26-FID-008`, `MST26-GOV-004`, `MST26-REL-002`

## B26 — 任务族与变形测试

**优先级 / 门槛：** P1 / D5\
**责任角色：** Task/Evaluation owner\
**前置依赖：** B14;B09;真实任务需求\
**当前/建议改动入口：** `新增受限 generator/metamorphic tests；任务 lineage`

**本包要交付什么。** 对 ID 重命名、无关实体、输入表示等受控变体生成稳定 Task；每个变体复核可解性和 oracle。

**验收和关键负例。** 相同 seed 得相同变体；合法等价变换保评分；公开任务不称私有 holdout。

**回滚与不做什么。** 不以规模替代领域深度；生成器更新时换 lineage/digest。

**关联需求。** `MST26-EXT-001`, `MST26-TASK-007`, `MST26-TASK-009`, `MST26-EVAL-004`

## B27 — 有限共享世界多 Agent

**优先级 / 门槛：** P2 / D5\
**责任角色：** Runtime/Evaluation owner\
**前置依赖：** B19;B26;共享世界用户需求\
**当前/建议改动入口：** `internal/episode/store；共享 branch profile；并发 evidence`

**本包要交付什么。** 先保留每 Agent 一 branch 比较，再为有需求的共享任务定义 actor、commit order、冲突与预算。

**验收和关键负例。** 记录的世界顺序可 replay；不声称宿主再次产生相同并发；stale worker 不能提交证据。

**回滚与不做什么。** 不变成通用 Agent orchestration；无需求则保持候选。

**关联需求。** `MST26-EXT-002`, `MST26-EPI-007`, `MST26-EPI-009`, `MST26-CORE-003`

## B28 — 引用安全的保留和 GC

**优先级 / 门槛：** P1 / D3A/D6\
**责任角色：** Storage/Evidence owner\
**前置依赖：** B07;B09\
**当前/建议改动入口：** `internal/store/episode；artifact manifest；清理/保留命令`

**本包要交付什么。** 实现 active/ref protection，分开临时 world、失败诊断和长期合成 evidence，删除前提供范围。

**验收和关键负例。** 仍被 replayable evidence 引用的工件不删除；cleanup 失败可见；没有无限默认敏感保留。

**回滚与不做什么。** 先逻辑 GC 与受控清理，不急做 copy-on-write；备份唯一副本不得覆盖。

**关联需求。** `MST26-OPS-001`, `MST26-SEC-007`, `MST26-REL-004`

## B29 — 恢复、迁移与导出验收

**优先级 / 门槛：** P0 / D6\
**责任角色：** Storage/Release owner\
**前置依赖：** B28;B11\
**当前/建议改动入口：** `internal/store；internal/episode/journal；migration/recovery testdata`

**本包要交付什么。** 扩展磁盘不足、权限、进程中断、封存冲突和 future schema 零修改拒绝；提供一致逻辑导出/恢复。

**验收和关键负例。** 在副本演练升级/中断恢复；world/Journal 版本分开；未知外部状态不自动重跑。

**回滚与不做什么。** 不承诺任意 downgrade 或 SQLite 多主；失败保留可检查原始工件。

**关联需求。** `MST26-OPS-002`, `MST26-OPS-003`, `MST26-REL-004`, `MST26-EPI-009`

## B30 — 实测性能与最小运维观察

**优先级 / 门槛：** P1 / D6\
**责任角色：** Runtime/Operations owner\
**前置依赖：** B07;B11;B28\
**当前/建议改动入口：** `benchmarks；internal/logging/governor；可选 exporter`

**本包要交付什么。** 固定 corpus/hardware/toolchain 测 CPU/内存/磁盘/latency/trace 开销；日志与 canonical Evidence 分离。

**验收和关键负例。** 结果带原始样本、数据规模和噪声；没有硬 RSS 保证却声称隔离会被审查拒绝。

**回滚与不做什么。** OTel/HTML 是可选派生；不为吞吐量牺牲确定性、事务或隐私。

**关联需求。** `MST26-OPS-004`, `MST26-OPS-005`, `MST26-CORE-005`, `MST26-TRACE-008`

## B31 — 1.0 稳定范围资格审查

**优先级 / 门槛：** P0 / D6\
**责任角色：** Release owner/独立使用方\
**前置依赖：** B10;B11;B15;B29;B30；所声明 profile 的 gates\
**当前/建议改动入口：** `release workflow；稳定格式/CLI 文档；兼容矩阵；迁移指南`

**本包要交付什么。** 只冻结经 preview 和真实使用的子集；固定 reader/writer、CLI、evidence、版本/弃用政策与跨平台工件。

**验收和关键负例。** 所有 stable P0/P1 有证据；外部消费方复验；跨平台不是各跑测试而是比较语义产物。

**回滚与不做什么。** 未完成 native/multi-agent 维持实验性或不发该 profile；不为 1.0 标题扩大承诺。

**关联需求。** `MST26-REL-001`, `MST26-REL-002`, `MST26-REL-003`, `MST26-REL-004`, `MST26-GOV-005`, `MST26-SEC-009`, `MST26-PROD-003`, `MST26-PROD-004`

## B32 — 持续维护、弃用和退出

**优先级 / 门槛：** P1 / D7/D8\
**责任角色：** 维护者/Release owner\
**前置依赖：** 基础维护从当前阶段持续执行；稳定弃用与整体退役分支依赖 B31\
**当前/建议改动入口：** `维护/事故/弃用/退役手册；claim freshness；最终导出与归档`

**本包要交付什么。** 建立 drift/漏洞/错评分/秘密泄露处理、错误 claim 撤回、更正 lineage、稳定弃用窗口和最终归档清单。

**验收和关键负例。** 撤回不篡改旧 digest；退役 adapter 没有 current 绿标；在线任务、endpoint、凭据可明确停止，离线证据能读。

**回滚与不做什么。** 扩展范围由需求和可维护性决定；AI 生成的新规则仍需接受和验证。

**关联需求。** `MST26-REL-005`, `MST26-REL-006`, `MST26-REL-007`, `MST26-GOV-006`, `MST26-EXT-003`, `MST26-EXT-004`, `MST26-PROD-001`, `MST26-HOST-010`

## PR 完成检查

提交者说明受影响的公共语义和 profile；评审者复核 Task/oracle 与正反例；CI 产出 exact-revision 证据；报告/文档清楚写出新范围及不支持项；有状态变更说明迁移/恢复；付费或远程试验有预算与授权。未满足条件只合并为受限实验，不能直接晋级 stable/compatible/L2。
