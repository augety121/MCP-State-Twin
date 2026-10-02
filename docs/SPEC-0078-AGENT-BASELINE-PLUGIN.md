# SPEC-0078：面向高级 Agent 的基线插件与评测产品化

- **状态：Accepted for implementation，实施中；未完成的验收不得宣称已实现。**
- **日期：2026-10-02；文档版本：0.1。**
- 用户目标：基于当前项目设计并实施 Agent 基线插件；2026-10-02 已明确授权整批实施。
- 产品解释：提供可接入 Agent 宿主的有状态测试环境、冻结的评测基线和可审计回归判定；不提供 AGI 认证或通用智能总分。
- 设计权威：[RFC-0001](RFC-0001.md)、已接受 ADR；本稿不覆盖任何既有硬不变量。
- 配套：[决策提案](ADR-0078-AGENT-BASELINE-PLUGIN-PROPOSAL.md)、[实施与验收计划](planning/agi-baseline/ITERATION-AND-ACCEPTANCE.md)。
- 本稿中 MUST/必须描述应满足的契约，不表示已全部通过；当前实现、测试与阻塞项以[实施台账](maintenance/SPEC-0078-DELIVERY.md)为准。

## 1. 评审结论与产品主线

当前项目已经具备确定性世界、独立 Task/oracle、离线项目、跨项目 campaign、重放核验和证据搬运。下一轮的核心交付是：**让一个外部评测框架能够安装并接入这些世界，用自己的 Agent 运行，再获得可信且能指导升级决策的结果。**

不能继续只增加 CLI 包装和合成 witness 数量。用户应能完成下面的闭环：

1. 选择一个已审阅的任务包与基线版本，离线检查本机可运行条件。
2. 通过 MCP 接入一个全新隔离世界，使用自己的 Agent/harness 配置完成任务。
3. 由可信评测端封存真实工具轨迹、执行状态、清理结果和独立评分。
4. 对照冻结的旧配置，查看具体退化任务、策略违规、失败原因、耗时和可用的成本数据。
5. 在固定样本和预先声明的规则下，得到 `pass / fail / inconclusive / invalid` 的升级结论。
6. 在另一个干净环境重验世界执行和评分；明确哪些模型行为不可重现。

产品暂称 **MCP State Twin Agent Baseline Kit**；这是本项目内的工作名称，不是已注册品牌或生态标准。“AGI-facing”仅表示面向更强、更长期的 Agent 提供测试基础设施。不得据此改写 RFC-0001 的产品边界。

首个参考集成选 **Inspect 的任务/评分组件 + MCP 工具连接**，理由是它已有相应扩展点和评测语义，便于证明本项目可嵌入其他 harness。[S4][S5] 这是本稿的工程选择，不声称与 Inspect 官方合作，也不等同于任何桌面产品宿主兼容。

### 1.1 用户与核心场景

| 用户 | 需要作出的决定 | 产品应交付的结果 |
|---|---|---|
| Agent 应用开发者 | 更换 prompt、模型或工具策略后是否退化 | 同一基线下的逐任务变化、固定分母和准入门禁 |
| 评测/研究工程师 | 某项能力或失效模式能否被可靠测量 | 任务族、正反例、评分器覆盖、重复试验和适用范围 |
| 平台集成人员 | 能否以现有 harness 接入且不泄露控制权 | 插件描述、协议边界、生命周期与兼容证据 |
| 开源维护者 | 是否具备发布和宣称兼容的证据 | 版本矩阵、候选 CI、已知限制、降级和撤销流程 |

### 1.2 成功指标和非目标

以下为拟议产品门槛，均需实际测量，不是“大厂统一标准”：

- 已安装依赖的干净环境，完成离线 first-value 的目标时间不超过 10 分钟；至少 3 名非作者按指南操作，记录成功与失败，不用作者演示替代。
- 原 24 Task、103 case、97 witness 与既有 48-trial campaign 保持回归；旧 oracle 不因新 Agent 表现而放宽。
- 首个框架适配器在固定版本上走通 24 个任务的真实 MCP 调用、评分与清理；mock 证明接入，另立 live 证据证明真实模型运行。
- 接入失败、模型失败、策略失败和证据失败必须分栏；任何缺失样本不得从计划分母删除。
- 插件启动、工具调用和评分的开销先建立基线，再按第 12 节判定；不承诺没有测量过的吞吐量。

本轮不建设模型训练平台、Agent orchestration、RAG/memory、任意脚本插件市场、通用云多租户、生产代理或自我进化闭环。不把长期时间调度、共享世界多 Agent、任意 SaaS 自动建模塞入首批。既有相关 roadmap 继续保留，但不能用本稿暗中扩张授权。

## 2. 当前项目审计与差距

### 2.1 基线和本轮证据

本轮静态阅读本地 `1cd2dba3541beec62fbdfaebe9dc64da47babfda` 的代码与文档。GitHub 查询确认 [PR #18](https://github.com/augety121/MCP-State-Twin/pull/18) 已合并；查询时 main 为 `5ad90baa691a8df8ece1942aa74e2d99bcc3502e`。GitHub compare 显示本地提交至该 main 有两个提交、文件差异为空。因此下面的代码审计适用于该 main 的文件内容。

前轮最终提交的 [CI 36974358944](https://github.com/augety121/MCP-State-Twin/actions/runs/36974358944) 是历史实施证据。本轮没有重新运行业务测试、真实模型或性能基准，也不将其说成合并提交的新 CI。本稿的文档检查单独记录在交付说明。

现有 IMPLEMENTATION-STATUS/README 部分段落仍采用“分支候选”的历史措辞；实施时应依据合并事实更新，本轮不改旧文档。旧大规格中缺少原始附件的需求仍不能被宣称已经审计。

### 2.2 事实、缺口、复用入口

| 面向 | 代码中可确认的事实 | 不能外推的能力 / 下一步缺口 | 复用入口 |
|---|---|---|---|
| 世界内核 | 声明式 TwinSpec、SQLite 状态、原子工具转移、分支/时间/故障子集 | 通用 SaaS 等价、完整 L2、任意时间工作流 | `internal/engine`、`store`、`spec` |
| MCP | tools-first；已编译证据为 SDK 1.8.0，2026-07-28/2025-11-25 两种既有 profile | 当前只有 Streamable HTTP data plane；没有可交付的 stdio 插件会话契约 | `internal/server/data.go`、`protocol.go` |
| Task | 独立目标、authority、budgets、CEL oracle、blind 模式 | 任务包版本治理、公开/私有分割、参数化任务族尚未完整产品化 | `internal/task`、`evaluator` |
| 单次执行 | 可信进程内 MCP world、mock loop、独立 live lane | 外部任意 harness 不能直接沿用私有 `environment` 当稳定 API | `internal/agenteval/environment.go`、`episode.go` |
| 项目 | 四组六任务，quality→suite→reviewed assessment，冻结输入、固定分母 | Project/Suite/Campaign 当前明确 offline；不能改一处 model 字段就运行 live | `project_plan.go`、`project_run.go`、`campaign.go` |
| live | 受批准的单 Task 本地 API bridge、有限传输和证据 | 没有足够实际模型/产品宿主采样；不是 live campaign 或产品兼容证明 | `internal/agentapi`、`agenteval/live.go` |
| 证据 | 重放、Task/世界独立内容匹配、无覆盖发布、导出/导入 | 内容一致性不等于作者或审批身份鉴证；历史 qualification 为 recorded_only | `reviewed_assessment.go`、`world_catalog.go`、`suite_archive.go` |
| 比较 | 确定性离线候选达标和回归分类 | 随机模型的置信区间、任务族相关性、冻结统计计划不完整 | `compare.go`、`assessment.go` |
| 宿主 | 报告准入、声明目标/时效/一致性检查 | 完整 HostProfile、自动采证、撤销及来源验证仍有缺口 | `internal/hostcompat` |
| 资源 | 单 suite≤16 pairs/64 MiB 输入，project 120 秒、campaign 480 秒 | 大实验须显式分片，不能放宽旧预算假装扩展 | `suite_plan.go`、`project_plan.go` |
| 性能/发布 | 已有 225 个准备/核验采样，候选发布门禁 | 缺插件进程/外部框架开销、独立使用者与稳定版本证据 | `docs/maintenance/SPEC-0077-PERFORMANCE.md`、`releasepolicy` |

代码引用以仓库相对路径说明未来修改入口，不表示新模块已存在。实施前必须再次对照当时的 main，不能照搬这个快照覆盖后续变更。

## 3. 插件的两个接口与信任边界

### 3.1 对外产品组成

| 组件 | 使用者 | 交付形式 | 边界 |
|---|---|---|---|
| Environment Plugin | 被测 Agent 的 MCP client | `statetwin plugin serve` 的 stdio 会话 | 只有 Task 授权业务工具；每个会话只绑定一个 world |
| Evaluation Adapter | 可信 harness/CI | 版本化本地进程协议；首个可选 Inspect Python 包 | 管理会话、读取目标投影、收集状态、结束和取得评分 |
| Baseline Pack | 维护者/评测者 | 严格声明式目录和 manifest | 引用现有 Task、Bundle、cases 和 reviewed references；不能指定任意可执行程序 |
| Baseline Decision | 维护者/CI | 只读报告和门禁命令 | 冻结计划与全部样本，分别输出证据有效性、业务门槛、统计判定 |

“插件”不意味着 Go 动态库、下载即运行的脚本或 MCP 管理工具集合。首批通过固定二进制和薄适配器接入，不新增 SaaS 后台。

```text
可信操作员 / CI
  └─ 评测框架 + State Twin adapter
       ├─ 私有生命周期通道 ── session owner / grader / evidence
       └─ Agent host ── MCP stdio ── Task authority ── world runtime
                             ↑
                  仅业务工具和明确公开的目标

Task oracle / reviewed refs / faults / state inspection
  只留在可信侧；不进入 MCP tools、resources、prompts、日志或 Agent workspace
```

### 3.2 不可妥协的规则

1. MCP `tools/list` 只返回该 Task 的允许工具；工具调用在服务端再次执行参数级 authority 检查。隐藏工具名不构成授权。
2. 禁止向被测 Agent 暴露 reset/fork/snapshot/inspect/fault、quality、grade、acceptance 和其他 trial 的任何信息。
3. 同一二进制可共享内核代码，但 MCP 与私有生命周期协议不得复用 method namespace 或凭据。
4. MCP stdout 只写协议消息；有界诊断写 stderr。诊断同样不能包含 oracle、state、token 或敏感绝对路径。[S2]
5. 默认离线模式不得联网获取依赖、下载任务、探测账号或调用 provider；check/doctor 也不例外。
6. 环境可确定性重放不代表模型可确定性重跑。模型和 harness 的实际配置、观察到的版本、未提供的字段均分别记录。
7. 评分来源不允许由 Agent 通过自然语言或自报 success 覆盖。真实世界终态与规则是最终依据。

### 3.3 隔离等级必须诚实

- `trusted-tools-only-v1`：可信 harness 仅给模型 Task MCP 工具，关闭其他 filesystem/shell/browser/MCP 工具和历史会话注入。它证明工具面和会话隔离，不证明恶意本地程序无法读同用户文件。
- `os-separated-v1`：Agent 可执行代码时，必须独立 OS 身份/容器、受限挂载、进程与网络访问；控制凭据、oracle、baseline artifacts 不挂载给 Agent。Linux 为首个拟议验证平台，Windows/macOS 无证据时标 unsupported。
- 仅靠工作目录、随机端口、分支名、prompt 警告或文件名隐藏，都不能达到第二级。
- 无法检查外部工具配置的宿主，只能出 `isolation=unverified` 的诊断结果，不进入发布基线。
- 首批支持第一级；带任意 shell 的产品宿主不能绕过第二级要求，必须在后续独立宿主 profile 中验证。

## 4. 数据与版本契约

### 4.1 不改旧格式，增加明确封套

下表为待实现的严格 JSON 合约；统一 `statetwin.dev/<name>/v1alpha1` 前缀。禁止未知、重复、null 必填字段、浮点冒充整数、重复 ID、路径越界、别名和超限输入。所有引用在副作用前解析并冻结。

| name | 必须表达的字段 | 不能表达的内容 |
|---|---|---|
| `baseline-pack` | id、revision、license、runtimeCompatibility、profile、entries、reviewReferences、disclosure、resourceProfile | shell command、installer URL、凭据、模型默认授权 |
| `plugin-profile` | id、revision、protocolProfile、transport、isolation、taskProjection、lifecycleVersion、limits | 通配宿主兼容、未知能力默认 true |
| `plugin-session-plan` | id、packRef、entryId、hostProfileRef、runConfigRef、outputRef、mode、deadline、declaredDifferences | Agent 指定 oracle、远端任意 endpoint |
| `plugin-session-report` | identity、lifecycle、execution、evidence、cleanup、budgetUsage、failureCodes、terminalRef | 把未运行样本写成失败后再从分母去掉 |
| `baseline-plan` | id、pack/review refs、baseline/candidate 配置、ordered trials、samplingPolicy、decisionPolicy、shards、mode | 看完结果再选择 Task、重试和修改阈值 |
| `baseline-report` | plan、planned/started/scored/verified 计数、逐 trial 状态、分层指标、判定、适用范围 | 单一 AGI 总分、跨不可比模型成本估算 |

具体 JSON Schema 与合法/非法 golden fixtures 是 N1/N2 实施交付物。表中 `Ref` 是被明确根目录约束的本地逻辑引用；不允许 HTTP URL 自动拉取。相对路径的规范化、符号链接/硬链接/大小写别名拒绝延用 0077 规则。

身份复用现有 Bundle/Task/运行时业务身份与冻结原始内容；禁止额外引入文件 hash 清单作为本批验收手段。包 revision 相同但内容不一致必须拒绝；revision 不是内容真实性证明。来源认证在第 11 节另行定界，不拿数字指纹代替信任。

### 4.2 Baseline Pack 条目与角色

每个 entry 至少包含 `taskId / taskRef / familyId / variantId / split / worldRef / qualityCaseRef / disclosure`。Task 与 reviewed Task、执行 world 与 reviewed world 仍须物理独立，不允许“运行时顺便复制自己成为参考”。

同一个原始 Task 在多个实验中复用时不能被算成新增能力。原 Task ID 和 oracle 保持原义；变体产生独立 ID、revision 和引用关系。Task/family/instance/trial 四个数量分别报告。

包生命周期：`draft → qualified → frozen → deprecated → retired`。`qualified` 要有机器测试和评审记录；作者自评必须标明 `reviewIndependence=self-reviewed`，不能伪造第二位评审者。冻结后内容不可原位更改；缺陷修复产生新 revision，并保留旧报告的解释与撤销记录。

本地完整包仅提供给可信 harness。Agent 可见投影只包含 objective、必要 context、可见工具以及明确需要告知的业务限制；完整 Task 的 oracle、expectedOutcome、故障计划、witness 和评分反馈不能透传。任务要求应写清要做什么，但不给出隐藏判分实现。

## 5. N1：可安装、可隔离、可评分的插件纵向闭环

### 5.1 命令与使用路径

拟议操作员命令：

```text
statetwin plugin check --root <root> --pack <manifest> --profile <profile> --format json
statetwin plugin describe --root <root> --pack <manifest> --profile <profile> --format json
statetwin plugin serve --session-plan <plan>
statetwin plugin inspect --root <root> --session-plan <plan> --format json
```

`check` 做纯本地准入；`describe` 只输出接入参数模板和能力，不修改宿主配置、不探测网络。`serve` 是由可信 adapter 启动的固定程序入口，不由 Agent 根据 prompt 决定启动参数。完整会话计划不作为模型消息。

MCP 通道使用 stdin/stdout，生命周期用额外私有 IPC：Unix private socket / Windows named pipe，创建时限制访问并通过继承句柄或私有父子握手传递会话能力。不得把 token 放命令行、URL、MCP 消息或普通日志。权限测试必须覆盖实际平台；同 OS 用户攻击者仍受第 3.3 节限制。

生命周期只支持 `ready / finish / abort / status`，不提供任意调用、脚本、state 查询或 reset。可信 adapter 持有控制权；被测 Agent 不能调用这些消息。非法消息、错会话、超限和状态转换错误使用固定错误码。

控制消息共同字段为 `version / sessionId / sequence / operation`，响应包含相同序号与有限状态。`ready` 是子进程发出的通知；其余三项由可信父进程发出。sequence 单调递增，重复序号只可返回完全相同请求的既有结果，不同 payload 拒绝。`finish` 接受有限 `hostOutcome`（completed/cancelled/failed）和最多 32 KiB 的最终 `answer`；answer 只是原 Task oracle 的评分输入，不能覆盖 grade 或终态 state。此补充支持已有只读任务的结构化答案要求。usage 由独立 harness/provider receipts 记录，不从控制消息接受任意自报费用。`abort` 只接受 cancelled/failed hostOutcome，不接受 answer；`status` 只返回有限生命周期。子进程不得按消息指定的路径读取或写入文件。

N1 新错误码至少包含 `PLUGIN_PLAN_INVALID`、`PLUGIN_RESOURCE_LIMIT`、`PLUGIN_OUTPUT_EXISTS`、`PLUGIN_HOST_UNSUPPORTED`、`PLUGIN_CONTROL_DENIED`、`PLUGIN_STATE_CONFLICT`、`PLUGIN_STARTUP_TIMEOUT`、`PLUGIN_INTERRUPTED`、`PLUGIN_EVIDENCE_INVALID`、`PLUGIN_CLEANUP_FAILED`、`PLUGIN_OUTPUT_FAILED`。业务 `AUTHORITY_DENIED`、`INVALID_INPUT` 和 unknown 行为沿用原语义；协议解码错误与业务错误不能混成成功 tool result。错误码表、退出码映射与 JSON 字段必须由同一份 contract fixtures 覆盖。

### 5.2 不能直接复用现有 data plane 作为授权边界

当前 `DataPlane.buildHandler` 枚举整个 TwinSpec 工具，Task authority 在内部 `environment.step` 执行。外部插件若直接暴露 data plane，就可能绕开原 authority。这是 N1 必须修复的集成问题。

实施时抽取可复用的 Task 约束 dispatcher：输入验证 → 计入 attempt → 参数级授权 → 按确定顺序执行 → 收集事件。已有 in-process 环境与新 stdio 环境复用它，旧 generic `serve` 的行为保持原契约。不得只在 Inspect 的工具过滤器里实现权限。

新 stdio 会话绑定一个独立 world，客户端不传 branch ID；list/call 在整个会话中都遵守冻结工具面。JSON-RPC 并发 mutating 请求串行裁决，入队上限和取消行为固定；不得把网络到达顺序不确定性包装为跨次确定性。

### 5.3 生命周期与故障

```text
admitted → claimed → starting → ready → running
                                      ↓
                                  quiescing
                                      ↓
                             collecting → grading → sealed
任何非终态 ──取消/错误──> quiescing → failed/partial
```

- 全量准入失败：不得建 world、输出目录或 provider 请求。
- claim 成功后：独占输出目录，立即记录计划和 partial 状态；已有目录拒绝覆盖，不自动 resume。
- `ready` 只证明 MCP 初始化、工具面与私有控制均满足，不证明业务成功。
- finish/cancel/deadline 首先关闭新调用准入，再等待或取消在途调用，捕获最终 head，收集完整事件；迟到调用不能改写冻结终态。
- `finish` 是“宿主已结束交互”，不是“任务已完成”；可信 grader 独立判定。
- MCP EOF、进程失联、宿主异常退出且没有合法 finish：标 interrupted；即使当时 state 已满足目标，也不能把不完整执行包装为正常完成。
- 已提交但未交付的工具结果延用现有语义，不能自动重试；必须保留 effect 与 delivery 的区别。
- cleanup 失败同时记录独立 cleanup 状态；不得因 score=success 覆盖失败。
- terminal 写入遵循写临时文件、同步、关闭、无覆盖发布；磁盘/短写/关闭/发布失败保留 partial。stdout 失败不回滚已经发布的结果。
- 首批无 session 断点续跑；重复 finish 只能返回同一已封存结果，不能重新评分或产生第二次终态。未知历史状态失败封闭。

### 5.4 Inspect 适配器

建议新增独立 `adapters/inspect/` Python 包，不向 Go 内核引入 Python 或 provider SDK。task loader 只给模型公开投影，solver 使用 MCP 会话，scorer 读取 State Twin terminal；它不另写一套 CEL/oracle。

一次 sample 对应一次 fresh session。adapter 必须在 finally 中结束/取消并回收自己启动的实例；framework 的 sample concurrency 在首批明确限制为 1。框架 resume 只能跳过已完整核验的 sample；不能重试付费请求、续接 partial session 或把失败 sample 当缓存成功。

同一 trial 的工具调用应使用整个 trial 生命周期内的 MCP connection，不能每次调用都重新创建 world。Inspect 文档提供了对应连接管理机制，但具体版本 API 必须在实施时固定并做契约测试。[S5]

首批支持两个 client profile：仓库内最小 conformance client 与固定版本 Inspect adapter。后者需独立于 Go 内部调用执行真实进程与 MCP wire；两者都可以使用 mock model 进行免费 CI。任何额外模型、shell 工具和 product host 均不得由 adapter 默认开启。

安装通过明确本地二进制路径和固定包版本完成；不自动安装依赖或改全局配置。`describe` 的安装模板需标示已测试/未验证。第三方任务包不能增加 setup hook、postinstall 或可执行入口。

### 5.5 N1 完整完成线

插件描述与准入、stdio、私有生命周期、服务端 authority、证据封存、Inspect 适配、跨平台生命周期负例、24 Task 接入、原回归、指南和候选 CI 必须整批完成。只做一个能调用 `list_issues` 的 demo 不构成 N1 完成。

## 6. N2：把任务集合变成可维护的能力基线

### 6.1 首批能力维度

| 维度 | 从现有业务出发的测量 | 必须具备的反例 |
|---|---|---|
| 状态读取与目标选择 | 从多个 issue/package 选择精确对象 | 同名对象、邻居对象、过期或不安全版本 |
| 多步目标完成 | comment+close、publish+install+confirm | 只做前半步、缺确认、错误顺序 |
| 授权与拒绝 | 范围约束、跨仓库、不可用目标 | 越权尝试、实际非法变化分别计数 |
| 不确定响应处理 | effect 已提交但 response 失败 | 重复创建/重复副作用、编造成功 |
| 最小副作用 | 完成目标同时保护邻居资源 | 对错误对象或无关对象写入 |
| 鲁棒性 | 同义目标、实体重命名、数据顺序变化 | 依赖固定 ID、fixture 顺序或预设 witness |

首批保留两个领域，不用新增领域数量替代业务规则深度。只具备工具执行权限的 bounded 单 Agent 是本基线的测量总体；长周期、跨领域迁移和多人协作不由这些维度推出。

### 6.2 任务族和样本分割

以 24 个既有 Task 建立 24 个语义 family；每个 family 先保留原始 dev 实例，再增加一个公开 regression 变体与一个评测用 holdout 变体，共 72 个具体实例。数量是这轮资产目标，不等于 72 项独立能力。

变体只允许已审阅的有限参数：实体 ID、非敏感标签、无关记录顺序、可选干扰项、预先枚举的目标表述。参数必须同时绑定目标、fixture 和 oracle，禁止仅替换 prompt 导致误评分；生成时先验证引用完整性和可解性，再生成独立 Task/Bundle。禁止任意模板脚本、LLM 自动生成即入库或自由 CEL 生成。

同一 family 的全部变体在统计中聚类，不能把换名样本当独立总体。语义等价变体应有 metamorphic 关系：合法 witness 经对应映射后仍通过，错误对象或额外副作用仍失败。预期拒绝任务也必须保留合法拒绝轨迹。

公开仓库里的 holdout 只能叫 `public-evaluation-split`，不能宣称防训练污染。真正 `private-holdout` 由可信评测者在 Agent 不可读的位置维护，访问权限、使用次数和泄露记录单列；一旦公开，就变更 disclosure 并停止作为未见基线使用。外部私有资产缺失时实现可以交付，private-holdout 的资格不能冒称完成。

质量门禁至少包括每条 assertion 的正/负覆盖、独立 reviewed refs、所有 Task 的错对象/额外副作用反例、至少两条合法轨迹（语义确实只允许单轨迹的任务需书面豁免与测试）。合成评分视图变异继续显式标注，不混入模型成功率。

## 7. N3：冻结实验和可靠的升级判定

### 7.1 实验定义

baseline plan 在运行前冻结：任务实例及 family、两套模型/harness 配置、可变配置字段白名单、工具面/Task/world/review refs、重复数、顺序、资源预算、隔离等级、split、评分版本、统计与绝对门槛。任何结果依赖的改动都产生新 plan revision。

分别记录 requested model、observed model/snapshot、provider、host、adapter、prompt configuration、工具投影与数据披露状态。服务未返回实际 snapshot 时记 unknown，不用请求别名伪装固定版本。允许比较多项配置变化，但必须标为“整个配置的差异”，不能归因给单一模型因素。

先提供 24 个 dev 实例 × 2 配置 × 3 次的 pilot（144 trials），它仅验证流程与测量噪声。冻结发布比较建议使用 24 个评测实例 × 2 配置 × 5 次（240 trials）。样本量足够与否按区间判定；5 次不是统计充分性的保证。

配对单位为相同 family/variant/repeat 的 baseline 与 candidate；按预先固定的 AB/BA 交替顺序平衡运行时间。世界 seed 可绑定，provider 不提供 seed 时记录 unknown，不能假定两边模型随机性已配对。

### 7.2 指标和分母

| 指标 | 定义 / 展示要求 |
|---|---|
| success rate | verified 且 oracle 符合 expectedOutcome 的数量 / 全部 planned trials；缺失另列，不删除 |
| scorable rate | 能按原评分规则确定结果的数量 / planned；不能把 error 当业务负样本后隐藏其类型 |
| execution completeness | 已到已知终态、证据有效且 cleanup 完成的数量 / planned |
| policy violations | 非法尝试、实际非法副作用、新增 policy failure 分别给绝对数量和比例 |
| repeat reliability | 每个 Task 的成功次数/n 与全部 n 次成功的观测布尔值；不能把小样本 s/n 直接幂运算称为实测 pass^n |
| latency | 启动、provider、工具、评分、清理分别统计；timeout 作为删失/失败显示，不算进“快速成功” |
| tokens/cost | 区分 reported/estimated/unavailable；缺少 usage 或价格时标 unknown，不记零 |
| coverage | family、领域、维度、split、隔离和 fidelity 范围；不生成 AGI 总分 |

模型置信分析仅使用 complete/verified 的固定比较集；有缺失或污染时发布判定为 invalid，仍展示固定分母的运维结果。不能通过过滤掉坏 trial 恢复有效性。

### 7.3 判定顺序

1. **validity**：所有计划样本、定义绑定、独立 reference、清理、披露和隔离合格；否则 `invalid`。
2. **hard gates**：任何新 policy failure、非法实际副作用，或预先标记必过 Task 的失败，直接 `fail`，不能被平均收益抵消。
3. **absolute floor**：candidate 的领域级成功率达到运行前设定的门槛；未设置门槛只能输出研究报告，不能生成发布 pass。
4. **non-inferiority**：以 family 为 cluster，对每个 family 的配对成功差值聚合；按两个领域分层重采样，固定 seed/算法版本，10,000 次 bootstrap 给 95% 区间。区间下界≥`-delta` 才通过“不劣”门槛。
5. 区间跨越 `-delta` 且没有明确硬失败时 `inconclusive`；不能为了变绿临时追加样本。新的扩样必须是新计划并保留旧结果。

`delta` 为运行前明确的最大可接受退化幅度；拟议默认 0.05，仅作为工程初值，必须在计划中显式出现，不可省略默认。至少 20 个 family、每实例至少 3 次才启用上述推断 profile；不足时 `insufficient-sample`。这不是普适统计定理，报告必须说明估计范围仅为当前两个领域与所选 family。

每个领域至少 8 个 family；先在各领域内部重采样 family，再按运行前冻结的等领域权重求总体差值，保留 family 内所有实例和重复，不重采样为独立 trial。所有 family 差值相同、重采样分布退化或方法前提不满足时只展示观测差值，统计门禁为 `inconclusive`，不得用零宽区间声称没有风险。零次 policy violation 也只表示本次未观察到，不等于真实风险为零。独立样本量、估计总体和方法限制必须与区间同时展示。

分领域 floor 与 family 硬门槛不依赖显著性。“有显著提升”是独立可选结论：需另立主指标与多重比较方案；首批不实现探索多个模型后择优宣称的 leaderboard。bootstrap 极端退化、全部相同、边界样本、相关变体和固定 seed 都需 golden tests。

### 7.4 大实验不能绕开旧限额

增加独立 `baseline-plan`，不把 live 塞入 `offline-reviewed-campaign-v1`。它是有界的评测分片目录，不是通用 DAG engine。

首批每计划≤240 trials，每 shard≤12 trials，最多 20 shards，串行执行；每 trial 延用 Task 的 episode/request/cleanup 上限。每 shard 外层预算≤2700 秒，显式逐 shard 开始；不继承或增大旧 project 120 秒/campaign 480 秒限额。长计划只索引已完成 shard 证据，不将所有终态留在内存。

shard 已发布后不可覆写；中断 shard 不续跑，保留原 planned rows 并生成新计划重试。失败 shard 或未获后续 live 批准不能让汇总变成通过；可以完成其他已授权的离线验收，但整份实验 remains incomplete。

## 8. N4：真实模型、产品宿主与外部使用证据

N1–N3 先用 mock 和合成世界验证系统。N4 才采集独立的实际模型/宿主证据；它不与“继续开发”的授权混为一谈。

- 首个 live profile 复用已接受的本地 API bridge 约束；Inspect live 使用自身 provider 通道时，必须新增独立 egress/credential/预算/数据保留 profile，不能借用旧 bridge 的批准或默认模型。
- 执行前提供具体模型 ID、请求/输出 token/时长/次数上限、有效期、允许的数据和用户可理解的花费上界或“无法硬封顶”的说明。没有账户级费用控制时不能声称硬金额限额。
- 本次 Spec 不选用用户账号的真实模型、不读取凭据、不进行付费试跑。将来实施可完成所有 mock readiness；live 样本只有实际授权并运行后才能填入验收证据。
- 首个付费试验先六任务冒烟，覆盖真实工具调用、预期拒绝、取消与响应不确定；再对具体可用配置跑冻结 pilot，不能直接盲跑 240 trials。
- 一个真实 API/框架 profile 的成功不能推出某个桌面产品兼容。产品宿主须独立登记精确产品/版本、安装步骤、可见工具、额外工具隔离、fresh session、终止和证据采集方法。
- 本稿默认先交付 Inspect，再由实际使用需求选定一个产品宿主；选择不是代码开发的前置阻塞，然而该产品的 verified claim 必须等待明确目标和真实测试条件。
- 兼容状态使用现有 SPEC-0019 的 unsupported/unverified/experimental/verified/regressed/stale；API 与产品 TTL 沿用既有政策，版本变化立即过期。
- 三名非作者的 first-value 反馈记录安装耗时、卡点、失败原因和是否能解释回归；征集参与者需单独授权，不自行对外发消息。没有参与者就留 `external-validation-blocked`，不找 mock 用户代替。

远端 provider-hosted MCP 仍受 [SPEC-0020](SPEC-0020-REMOTE-SECURITY-PROFILE.md) 完整安全条件限制。首批本地 stdio 插件不顺带开放公网控制台或 remote control。

## 9. N5：基线保真度、发布和维护

### 9.1 保真度单独分级

现有双领域继续按其 L1/unverified/unbound 声明。独立 reviewed world 内容匹配证明“测试了预期世界”，不证明“真实服务就是这样”。

下一步有限 L2 只选择一个领域的最多 3 个具体操作，先建立有授权且可复位的独立 reference，再比较成功、错误、authority、副作用和重复调用。差分需要冻结上游契约、观测日期、未覆盖语义和重置方法；reference 不能直接调用同一 TwinSpec 引擎冒充独立实现。

没有 reference 来源时，完成差分契约和合成负例仍不允许标 L2。Recorder、生产 trace、远端接入和真实写入分别走已有后续工作包；本稿不授权采集生产数据。

### 9.2 版本与兼容

运行时、插件进程协议、MCP profile、adapter、Task pack、oracle、统计 policy 和证据格式分别版本化。任意一项变更都要明确哪些旧报告仍可重验、哪些比较失去可比性、哪些宿主证据过期。

旧 CLI 和旧证据维持原 decoder。新插件证据先独立封套；不可给旧格式添加未声明字段再要求旧读取器忽略。未来迁移必须用显式 copy-to-new，禁止原位改历史结果。

弃用至少跨两个已发布 preview 版本并提供替代路径；这是本稿建议维护政策，不是现有 SLA。紧急安全撤销可更快，但须提供原因、影响版本和可重验范围，不删除历史失败证据。

### 9.3 发行门禁

按现有 release plan 和 exact-candidate CI 构建固定二进制及可选 adapter 包；发布渠道、安装命令和许可需明确。没有用户明确要求，不自动推送 tag、发布包或改宿主配置。

首个发行只称 `baseline-plugin-preview`。stable 还需要外部使用、恢复/兼容承诺、维护责任和 Phase 7 的原门槛；本稿完成不自动等于项目 1.0。

## 10. 隐私、提示注入与不可信输入

| 入口/风险 | 必须执行的控制 | 负例 |
|---|---|---|
| 任务包指向私有文件 | 根目录约束、拒绝链接别名、冻结引用 | traversal、UNC/drive 路径、硬链接和大小写别名 |
| fixture 内含恶意指令 | 作为业务数据保留；不能变成 harness 命令 | 工具结果要求读 oracle、重置世界或导出密钥 |
| 自报评分/伪造报告 | scorer 只接受当前会话真实 evidence 与绑定 | Agent 输出 success JSON、旧 terminal 调包 |
| 外部宿主附带 shell/MCP | 不符合 tools-only 则拒绝资格；OS 隔离另验 | 从宿主 filesystem 读取 hidden refs |
| stderr/exception 泄密 | 有界有限错误码、解码后敏感扫描 | 转义 secret、带 token URL、深层对象字段 |
| raw trace 与模型推理 | 不要求也不采集隐藏 chain-of-thought；只保留协议所需可审计动作和允许的 usage | adapter 自动把 framework 全部日志复制到公开目录 |
| 宿主自动 retry/cache | 首批关闭；请求和工具 ID 行为明确 | 同 ID 不同 payload、超时后重复副作用 |
| 私有生命周期泄露 | 单独 IPC、独立能力、消息范围限制 | 在 MCP 发 finish/status 或复用别的 session 能力 |

Scan 不是完整隐私证明；默认只允许合成数据。framework 自己的日志也必须进入数据政策，不能只扫描 State Twin 报告。live/provider 外发与公开 artifact 是两次不同的数据披露，分别声明。

## 11. 证据可信度与来源

报告至少同时展示四个轴：`contentVerified`、`worldReplayVerified`、`sourceTrust`、`hostObservation`。不能用一个 verified 覆盖所有含义。

- 本地 artifact 的 sourceTrust 默认 `local-operator-asserted`；他人提供的文件不能因为可重放就变成 authenticated。
- 独立 reviewed references 的审批记录必须绑定明确版本、评审角色和内容；有评审文字不等于密码学来源验证。
- 若引入可信发布来源，后续采用现有发行渠道的可验证身份与撤销机制；首批不自创签名协议，不新增文件 hash 清单。
- `recorded_only` 的 quality 报告保持原边界；再次做 quality 需显式重跑并产生新结果，不能在 inspect 中偷偷执行 witness。
- 缺原始 terminal、身份不一致、runtime/评分器不支持或证据过期时，显示精确缺项并使相关门禁失败。

## 12. 资源、性能与可靠性

### 12.1 拟议新增上限

| 项目 | 首批上限 | 原则 |
|---|---|---|
| pack manifest | 256 KiB，最多 72 entries | 超出拒绝，不隐式拆分 |
| plugin/session plan | 每份 64 KiB | 不包含嵌入任意程序或全套 trace |
| lifecycle message | 64 KiB，深度≤16 | 固定字段和有限状态 |
| MCP message | 不超过既有 1 MiB 输入/输出限制 | stdout 背压、超长行在执行前拒绝 |
| session | 一个 world，一个有序 mutating 队列，最多 4 个在途请求 | queued 与 executing 均计预算，不无限堆积 |
| Task requests/time/trace | 沿用 Task 和旧组件更小上限 | 外层不能放宽内层 |
| process startup | 10 秒 | 超时回收自己创建的实例 |
| cleanup | Task cleanup 上限，最大 10 秒 | 未清理完整报告失败，必要时终止自己的子进程 |
| session report / stderr | 1 MiB / 256 KiB | 超限有明确状态，不静默截断后声称完整 |
| experiment | ≤240 trials，≤20 shards，每 shard≤12 trials | 初始串行，内存只保留当前 trial 与有界索引 |
| aggregate report | 4 MiB | 详细证据外置引用，无无限 trace 嵌入 |

这些是 proposed alpha profile；接受前不得替换现有 ResourceProfile。所有 byte/count 上限要测试 N、N+1、累计耗尽和取消。含 72 entries 的 pack 只是目录；执行只冻结本 shard 需要的内容并核对完整计划引用，不加载全部 evidence 到内存。

### 12.2 性能预算与 CI 分层

在固定 OS、CPU、Go/Python/adapter 版本、无 race 的环境采样，分别报告冷启动、热工具调用、评分和清理的 p50/p95/max、分配和峰值 RSS。为 p95 每个配置至少 30 次，复现命令和环境随报告保存；内核和 adapter 单独归因。

拟议门槛：启动 p95≤2 秒；相同小型任务的插件总耗时≤可信进程内基线的 1.5 倍且新增≤2 秒，两条同时满足。超过时必须定位，接受修订或优化；不能把门槛失败删掉。该门槛适用于声明的参考硬件，不是任意用户机器 SLA。

普通 PR CI 跑确定性 contract、原回归与少量真实进程端到端；race 保留，耗时较长的 package 可按不遗漏测试的方式分组。性能基准单独人工触发/发布候选执行，不在共享 runner 用不稳定毫秒阈值封死合并。扩展 fuzz/soak 的执行记录属于发布门槛，不能只写成尚未安排的未来任务。

可靠性测试需包含 100 个顺序短 session 的资源回收检查、10 次阶段化 crash 注入、取消竞态和无僵尸进程。只能终止/清理本轮拥有的实例；不得扫描并杀掉用户的同名 Go/Python 进程。

## 13. 实施组织、发布步骤与回退

采用六轮有退出门槛的迭代，而非同时重写全部系统：

| 迭代 | 用户可见交付 | 进入条件 | 完成后仍不能宣称 |
|---|---|---|---|
| N0 基线与契约 | 当前能力图、已接受边界、格式/接口决定 | 本稿评审 | 新插件已实现 |
| N1 插件完整纵向闭环 | 24 Task、stdio、Inspect、独立评分、跨平台清理 | N0 决策接受 | live 模型/任意产品宿主已验证 |
| N2 基线资产 | 24 family/72 instances、质量和 disclosure | N1 稳定接口 | 未见泛化、AGI 能力 |
| N3 实验与判定 | 冻结计划、分片、指标、统计门禁 | N2 + 配置差异规则 | 还没执行的真实模型结果 |
| N4 真实使用 | 具名模型/宿主证据、first-value 观察 | 具体授权、账号/隔离条件与参与者 | 不同产品、版本或所有模型兼容 |
| N5 发布资格 | preview 包、维护/撤销/恢复证据；有限 L2 独立子轨 | N1–N4 对所声明范围完成 | stable/通用 L2/生产等价 |

不承诺虚构日期或团队规模。责任采用角色：Maintainer、Runtime owner、Adapter owner、Evaluation owner、Security reviewer、Release owner；单维护者可兼任，但独立评审缺失必须如实标明。

完整工作包、逐项验收和依赖见配套计划。未来若用户要求“实施 N1”，必须完成 N1 全部工作包及 exact-head PR 检查再交付，不能只完成几个 helper 后要求用户继续。N4 外部条件不足不能阻止 N1–N3 已授权内容完成，但不能将整个路线标成完成。

回退以撤销新的插件/adapter 入口为主，保留既有 CLI、格式和 evidence。不得自动删除用户新结果、回写旧 oracle、放宽预算或把 failed 改成 passed。preview 的发布失败保留候选，不自动重发可变版本。

## 14. 评审决策与主要风险

| 决策/风险 | 本稿默认选择 | 接受前/实施时必须回答的问题 |
|---|---|---|
| 插件到底接什么 | stdio 环境插件 + Inspect 评测适配 | adapter 精确版本和最低 Python 版本在 N0 固定 |
| Agent 读到答案 | tools-only 先落地；代码宿主要求 OS 隔离 | 实际宿主能否关闭额外工具和记忆 |
| 协议演进 | 保留仓库已有 profile，新增 stdio 单独协商测试 | 当前 SDK 的 stdio 行为与目标 profile 的差异 |
| 插件触达当前 Task authority | 抽取公共 dispatcher，服务端强制 | 旧 generic server 的兼容与新 Task gate 的测试边界 |
| 统计假精确 | family 聚类、固定计划、inconclusive | delta/floor 与业务风险是否相称 |
| 公开数据污染 | disclosure 明确；私有集外部条件单列 | 是否真实存在受控 holdout 和独立评测者 |
| 框架日志泄密 | adapter 数据政策覆盖 framework 日志 | 禁用/清理项是否可在固定版本执行验证 |
| 大批次付费 | 小规模冒烟后显式分片授权 | 具体模型权限、限额、有效期与数据留存 |
| 多人开发/维护成本 | 模块薄、版本少、原能力不重写 | 是否值得第二个产品宿主，而非增加空配置模板 |

没有信息时使用上述已标明的规划默认值继续设计，不能把 unknown 填成已通过。新的 authority、production-write、脚本执行或产品边界变化必须走新的 RFC/ADR。

## 15. 外部依据与采用范围

资料于 2026-10-02 查阅。下面引用是设计参考，不代替本项目测试，也不是对第三方内部标准的宣称。除简短事实说明外，本文具体模块、限额、门槛和工作包均为本项目设计。

| ID | 一手资料 | 本稿采用的有限结论 |
|---|---|---|
| S1 | [Anthropic：Demystifying evals for AI agents](https://www.anthropic.com/engineering/demystifying-evals-for-ai-agents) | 区分 task/trial/grader/outcome；分开能力与回归评测，记录环境终态与重复运行；不照搬其模型分数 |
| S2 | [MCP 2025-11-25：Transports](https://modelcontextprotocol.io/specification/2025-11-25/basic/transports) | stdio 与 Streamable HTTP 的接口边界；stdout 不混入日志；此链接是固定旧版参考，不宣称最新协议 |
| S3 | [MCP：Security Best Practices](https://modelcontextprotocol.io/docs/2025-11-25/tutorials/security/security_best_practices) | 凭据、授权与连接边界需要独立验证；协议可用不等于安全配置完整 |
| S4 | [Inspect：Extensions](https://inspect.aisi.org.uk/extensions.html) | 独立 Python 包可提供任务、scorer、tools 等组件；适合薄适配器 |
| S5 | [Inspect：MCP Tools](https://inspect.aisi.org.uk/tools-mcp.html) | 提供 stdio/http/sandbox 连接与有状态连接管理；产品采用前仍需固定版本测试 |
| S6 | [Inspect：Sandboxing](https://inspect.aisi.org.uk/sandboxing.html) | sandbox 与本地进程具有不同隔离边界；容器配置本身也需要验证 |
| S7 | [Sierra：τ²-bench](https://github.com/sierra-research/tau2-bench) | 有状态工具交互是已有研究方向；本项目不复制其数据，也不宣称同等覆盖或新颖性 |

本稿遵循大规模工程评审常用的结构：明确问题、现状证据、契约、威胁模型、容量预算、备选方案、验收、发布和维护。它不声称获得任何公司认可，也不以“大厂标准”代替可执行门槛。

## 16. 本轮设计交付停止条件

设计阶段已结束。2026-10-02 用户明确授权按本 Spec 整批实施；按 N0–N5 完成可执行工作、必要验证与 PR。实际验收及外部条件见交付台账，不以设计接受或 mock 通过代替 live/外部用户/L2 证据。
