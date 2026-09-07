# MCP State Twin：项目全生命周期与下一阶段迭代总规格

**文档版本：v3.1 / 2026-09-08 · 仓库整合评审稿（Proposal，不改变 accepted 语义）**\

> **本轮审阅边界：** 完整读取两份上传的 Markdown，并静态核对本地
> `main @ 77d3ea0ec610fe64b48425f15c350e0b4c0d577f`。本轮不复跑 runtime、
> 不重验 GitHub CI、不调用付费模型、不发布版本。
> 下文原稿的历史 CI/外部研究记录不是本轮执行结果。
>
> **阅读顺序：** [评审与采用边界](REVIEW.md) → 本总规格 →
> [近期分阶段合约](PHASE-SPECS.md) → [任务卡](TASK-CATALOG.md) →
> [实施 Backlog](IMPLEMENTATION-BACKLOG.md)。
> 本稿新增/细化条款以 PHASE-SPECS 与 REVIEW 的 v3.1 修正为准；
> accepted RFC/ADR 始终优先，本稿没有自动接受任何新能力。
>
> **附件不完整：** 109 个原稿需求 ID 可从 Backlog 提取，但没有随附其
> `REQUIREMENTS-AND-ACCEPTANCE.md`、`requirements.yaml`、原任务目录、
> 来源登记或模板。不能据此宣称逐条审查了全部 109 项要求或 24 个任务。
> [来源登记](SOURCE-REGISTER.md) 区分本轮证据与未解析的原稿 R/W 引用；
> [补充验收清单](ACCEPTANCE.md) 使用独立 AE 编号，不伪造缺失原文。

**产品主线：从确定性工具世界，走向可实际用于模型升级决策的 Agent 回归测试基础设施**

| 项目 | 本文基准 |
|---|---|
| 仓库 | `augety121/MCP-State-Twin` |
| 审阅分支与提交 | `main @ 77d3ea0ec610fe64b48425f15c350e0b4c0d577f` |
| 对应最后实现提交 | `61aac2d241f841d6fa2f18ad64361de246792a3c` |
| 已核对的公开 CI | `33579877698`，上述实现提交，结论 `success` |
| 当前发行定位 | 仓库标为 `0.1.0-dev`；最新公开预览版标为 `v0.1.0-alpha.1`；尚无稳定发行 |
| 本次工作范围 | 指定提交的文档与关键代码静态审阅、公开 CI 元数据核验、官方外部资料核验、整体产品与实施规格设计 |
| 本次未完成的验证 | 没有在本地重新运行 Go 测试；没有运行真实付费模型；没有进行运行时渗透测试或上游差分实验；没有验证用户账号的模型权限 |
| 仓库变更 | 本文档包没有修改仓库、创建 Issue/PR、发布版本或调用生产服务 |
| 当前权威性 | 本稿未被项目接受；不能覆盖既有 accepted RFC/ADR，也不能作为功能已经实现的证据 |

**审阅说明。** 当前 HEAD 比已通过 CI 的实现提交多一个仅修改三份证据文档的提交。本次核对了该差异，但不能将“父提交 CI 通过”表述为“本次重新对 HEAD 全量测试通过”。引用编号 `[Rxx]`、`[Wxx]` 在 `SOURCE-AND-AUDIT-REGISTER.md` 中给出固定提交路径、资料来源与适用边界。[R01][R02][R09][R10]

**规划假设。** 暂按一个主要维护者、有限外部协作、开源、本地优先、合成数据优先进行排期。不假设现有商业客户、专职团队、可用 API 预算或已签订交付日期。所有版本映射、数量指标、保留期限和性能门槛均为本稿建议，未经接受不得写成项目现状。

---

## 阅读地图与交付物

本文回答“项目为什么做、已经做到哪里、下一步做什么、每个模块如何设计、整个生命周期如何推进”。同包中的文件不是额外的独立规范体系，而是本稿的执行视图。

| 文件 | 用途 |
|---|---|
| `MASTER-SPEC.zh-CN.md` | 全项目产品、架构、数据、运行、评测、发布与退役总规格 |
| `IMPLEMENTATION-BACKLOG.md` | 可直接拆成 Issue/PR 的工作包、依赖、验收与回滚边界 |
| `REQUIREMENTS-AND-ACCEPTANCE.md` | 稳定需求编号、优先级、交付门槛和验收方法 |
| `requirements.yaml` | 同一需求目录的机器可读副本；不是当前 runtime 的配置格式 |
| `SCENARIO-CATALOG.md` | 首批任务、可解性、所需工具、正反例与领域扩展边界 |
| `SOURCE-AND-AUDIT-REGISTER.md` | 事实依据、审阅限制、旧规划与本稿的关系 |
| `templates/` | 提案级 Task、HostProfile、评测计划与兼容性声明示例；不能直接交给现有 CLI 执行 |

建议决策顺序为：第 1–4 章确定产品取舍；第 7–14 章确认首个真实 Agent 闭环；第 24–29 章接受排期、组织与发行门槛；再按 Backlog 开发。旧文档保留为历史依据，不建议把本稿直接粘贴覆盖旧 master。

---

## 1. 项目经理结论：不重写底座，优先交付能作出回归决策的产品闭环

### 1.1 当前项目不是“没有做出来”

仓库已经有相当完整的本地确定性基础设施：TwinSpec 校验、声明式状态转换、SQLite 事务、快照与分叉、差异比较、脚本场景、Bundle、Episode Journal、有限调度、有限故障注入、协议烟测和安全边界。这些应作为资产保留，而不是在新 spec 中重新列为从零建设的任务。[R02][R05][R16]

但目前还没有足够证据支持“开发者换一个模型，就能用项目判断真实 Agent 是否可靠”。关键断点在于：现有 Episode 执行预先写好的 Scenario；provider smoke 检查调用连通性与基本成功，不负责用户目标、终态、越界和副作用评分。两部分还没有汇合成真实 Agent 的可归因评测流程。[R05][R06]

### 1.2 推荐产品定义

> **MCP State Twin 是面向工具型 Agent 的版本化测试世界与回归证据层：把模型、提示词、宿主或工具策略的变化，转化为可重复执行、可检查终态、可定位失败原因的升级决策。**

确定性的对象仍是外部世界，而不是语言模型。被比较的产品单元是“模型配置 + Agent 宿主/适配器 + 上下文与权限策略 + 相同任务世界”，不是一个孤立的模型名称。

### 1.3 三个必须改变的优先级

第一，把“真实 Agent 完成目标并接受独立评分”排到下一阶段主线上，不能继续等所有故障、远程执行、调度、录制功能都齐全才开始。

第二，把“参考领域是否足以表达一个可解、可验证的业务任务”置于增加领域数量之前。两个可靠领域优于二十个只有 happy path 的模拟接口。

第三，把“规范—实现—测试—证据—发行—用户使用”贯通。文档数量、测试数量、支持的模型名称数量，都不能单独代表产品完成度。

### 1.4 暂停扩张，不删除资产

现有远程 coordinator、Journal、scheduler 保留，修复必要缺陷并维护测试；暂不新增 HA、多租户、任意插件、通用工作流、自动循环调度、GUI 世界、工具市场。只有当已经验证的用户任务无法完成，且小范围补充不能解决时，才启动对应 RFC。

暂停意味着不让它们占据近期关键路径，不意味着这些方向无价值，也不意味着回滚已经接受的语义。

---

## 2. 事实基线：已经实现什么、还缺什么、能声明到哪一步

### 2.1 实现与依赖基线

| 层 | 指定提交下的现状 | 不能由此推导的结论 |
|---|---|---|
| 语言与依赖 | Go 1.26；CEL 0.32.0；官方 MCP Go SDK 1.7.0；JSON Schema v6 6.0.3；SQLite Go 模块 1.56.0 | 不是对所有依赖当前最新版的判断 |
| TwinSpec | 严格 YAML、声明式工具、输入/输出 Schema、有限 CEL | 不支持任意程序或任意真实服务自动复制 |
| World / Store | 事务转换、不可变逻辑快照、隔离分叉、reset/diff、schema v4 与 head CAS | 不等于多主分布式数据库、无限规模 copy-on-write 或完整备份体系 |
| 时间与随机性 | 私有虚拟时钟、建模随机流、信号队列、一次性本地 scheduled action | 不等于任意 Agent 定时运行、自动重试、周期任务或外部副作用调度 |
| 故障 | `before-validation`、`after-commit-before-response` 两阶段 | 不等于完整网络、崩溃、最终一致性及幂等语义 |
| MCP | stateless Streamable HTTP；现代 raw-wire 测试与 legacy 兼容测试 | 不等于所有现代可选功能或所有宿主兼容 |
| Scenario / Episode | 确定性脚本执行、终态断言与证据 | 不等于模型自主选择工具的 AgentTask 评测 |
| Bundle / Journal | 有界、可校验、确定性 unsigned Bundle；独立 Journal schema v2 | 不等于签名发布者身份、远程信任或公共 registry |
| Remote Episode | 租约、heartbeat、fencing、有限恢复与 terminal Evidence 接受 | 不能宣称外部 API 调用及其副作用 exactly-once |
| Provider | OpenAI/Anthropic smoke 适配器与 mock 合约测试 | 没有真实 live 报告，不能宣称已验证 Astra/Codex/Claude 产品 |
| 参考领域 | issue tracker、package registry；L1/unverified/unbound | 不能称为真实 GitHub 或真实包平台的已验证孪生 |
| 安全与资源 | 本地控制鉴权、边界测试、软运行配置、限流、hermetic CI | 不是数据平面远程多租户隔离，也不是硬 RSS / OS 配额 |
| 发行 | development preview，已有预发布和发行自动化 | 没有稳定版，不是 production-ready |

依据为实现状态、固定版本依赖、关键调用路径与 CI 定义；其中测试覆盖数量和通过声明来自仓库证据，并非本次重新执行。[R01][R02][R05][R06][R07][R08][R09]

### 2.2 本稿特别保留的已有能力

不得把已经实现的 private clock、entropy、signal scheduler、runtime-bound scheduled actions、TwinBundle、Journal、fencing、第二参考领域重新写成“尚不存在”。后续待办应表述为“补充跨平台语义证据”“接入 AgentEpisode”“增加特定故障策略”，并引用已接受的 bounded subset。[R02][R03]

### 2.3 真正缺失的交付闭环

目前明确缺少：自主 AgentTask；通用或首个实用 Host adapter 执行链路；模型观察轨迹与世界提交轨迹关联；独立任务评分；模型配置间的重复运行比较；真实宿主证据；上游差分与有限 L2 认定；面向外部使用者的低摩擦回归流程。[R02][R05][R06]

原稿记录过一次公开 Issue 查询，但本轮没有复验其结果；不据此推断当前需求或用户数量。后续立项可将 accepted 工作包映射到 Issue，创建/发送内容须有对应授权。[原稿 R12，来源未随附]

### 2.4 文档权威与局部失配

旧 master 是研究提案，不能把其中“当前未实现”的句子当作 9 月的代码现状。`VNEXT-TRACEABILITY` 的部分下方摘要与上方逐项更新的能力范围不够一致；`ROADMAP` 的部分“待合并/待证据”表述，也需要与最新 implementation status 对齐。这是当前摘要维护问题，不是旧架构全部失效。[R02][R03][R04][R11]

---

## 3. 为什么感觉有问题：六个具体诊断

### 3.1 验证底座的能力，比验证 Agent 的能力更成熟

`internal/episode/episode.go` 当前解码 Bundle 中声明的 Scenario，然后调用 `scenario.Run`；它验证脚本在世界中如何运行。`internal/provider/smoke.go` 则统计 MCP 工具发现、调用和错误，无法凭这些字段判断“是否只修改了授权对象”“是否重复提交”“是否真的达成任务”。这是产品链路未贯通，不应贬低现有单元测试的价值。[R05][R06]

**整改：** 保留 Scenario 作为 runtime 合约测试，新增与其并列的 AgentTask 与 AgentEpisode。不要直接给 Scenario 增加一个 `model` 字段就声称完成 Agent 评测。

### 3.2 部分未来路线过长，导致最先可验证的价值被后置

当前 Phase 4 把 secure remote staging 与原生 provider 验证绑定，并依赖前面远程阶段。这对原生远程 MCP 路线合理，但不是本地函数调用桥接路线的必需前置条件。[R04][R13]

**整改：** 用新 ADR 增加 local-outbound 试验路线，让模型 API 只接收受控业务工具输入/输出，Twin 仍在本地；原生远程路线继续满足既有安全门槛。不要通过开放无鉴权端口“绕过”安全工作。

### 3.3 环境可复现，不自动等于评测可复现

相同世界初态并不保证相同宿主记忆、工具投影、审批策略、输出截断和实际模型版本。当前代码证据还不足以覆盖这些变量。[R02][R06]

**整改：** 分别定义 WorldIdentity、RunDefinition、ObservationTrace、EvidenceEnvelope；不要把一次运行的唯一 ID 和墙钟字段掺入“两个起点是否相同”的判断。

### 3.4 任务可能不可解，或者评分器可能测错对象

issue tracker 当前六工具不包括读取评论；`add_comment` 无通用幂等键，`close_issue` 对已关闭对象返回冲突。这些是当前合成领域的具体语义，不是经过验证的真实上游行为。[R14]

**整改：** 先做可解性审查。已有 `get_issue` 支持关闭后的读取确认；评论提交后失联要先扩展可观察性或明确允许“无法确认、停止重试”。不能期待 Agent 读取根本不存在的工具。

package registry 也有对应边界：`install_dependency` 以 project/package 为键做 insert，不是升级已有安装；工具面没有 installation 查询。其版本字段也不是完整 SemVer/依赖解析合约。把“升级已有依赖”或“安装后失联读取确认”直接放入当前任务集，会把领域缺口误算成模型失败。[R17]

### 3.5 安全宣传范围容易大于实际隔离范围

数据路由从 `/mcp/{branch}` 取 branch 并调用 runtime；该 handler 没有实现 branch 主体授权。对本地可信开发用途，这与 preview 边界一致；对可运行 shell、访问文件或其他端口的宿主，隐藏 control tools 不足以形成完整隔离。[R07][R02]

**整改：** 把网络、文件系统、凭据、分支授权和 grader 数据访问纳入 IsolationProfile。分支名称不是密码，loopback 也不是恶意本地进程隔离。

### 3.6 “有状态评测”本身不足以说明差异化

ToolSandbox 已经研究有状态工具和不同轨迹的中间/终态评估；τ²-bench 研究共享状态下的交互式任务。不能把“别人都只返回静态 JSON”作为项目价值成立的前提。[W05][W06]

**整改：** 把差异化假设收敛为“用户可带入自己的工具合约，以可移植版本化世界反复做升级回归，并提供领域保真度边界与证据”。这需要外部使用来验证，不是已经建立的商业壁垒。

---

## 4. 产品定位、用户与成功标准

### 4.1 优先用户

第一类是维护 MCP 工具或工具型 Agent 的开发者：更换模型、提示词、工具定义后，需要知道业务行为是否退化。第二类是评测工程师：希望固定初态与故障条件，对多个 Agent 配置做可归因比较。第三类是领域维护者：愿意为一个小范围工具世界持续维护行为与保真度。

近期不优先服务：想自动复制整个企业 SaaS 的团队、需要多租户云调度平台的组织、寻找通用 Agent 编排框架的用户、以 GUI/浏览器交互为核心的评测。

### 4.2 核心工作任务

| 用户触发 | 产品应该提供的结果 | 不应让用户自己完成的工作 |
|---|---|---|
| 模型升级 | 候选与基线的任务结果、风险与成本差异 | 手动复制数据库、肉眼读全部聊天 |
| 提示词更新 | 同一世界与策略下的回归结论 | 重新编写一套测试服务 |
| 工具 Schema 变化 | 明确的兼容性差异与失败定位 | 把所有解析失败都归咎模型 |
| 发现线上边界问题 | 在合成/授权测试环境建立可重放最小案例 | 在生产中反复触发副作用 |
| 模型换了合法做法 | 按目标与约束正确评分 | 强制工具轨迹与旧模型逐字一致 |

### 4.3 北极星与辅助指标

北极星建议为：**外部使用者能够重复运行同一任务集，并据此作出一次有证据的升级、回退或修复决定。** 它是使用结果，不是遥测 KPI；默认不采集使用者数据。

辅助目标为：首次离线示例成功率、首次自有任务接入所需手工步骤、有效回归重复使用、缺陷复现成功率、报告中无法归因的失败比例。Stars、文档页数和总工具数只作生态参考。

### 4.4 初期验收目标（提案）

离线 first-value 路径建议不超过 3 个顶层动作：验证/加载示例、运行、查看报告。至少邀请 3 位或 3 个独立外部使用方做观察式试用，不能用维护者自己的三个环境代替三个用户。

先通过一个领域的 6 个 Agent 任务，再完成两个领域共 24 个任务候选的审查；任务数量不是模型得分，未实现依赖的任务必须保持 blocked。至少取得 2 个真实改进案例：例如发现了工具策略错误、定位了重复提交、或确认一次模型升级没有扩大授权范围。未达到时优先修产品闭环，不扩充平台规模。

### 4.5 明确非目标

1.0 前不把产品做成模型训练系统、planner、通用工作流引擎、云多租户平台、任意代码插件宿主、生产反向代理或“与所有 Agent 完全兼容”的系统。允许提供最小评测 adapter，但它只负责运行与观察任务，不拥有业务规划策略。

---

## 5. Astra 发布后该如何调整，而不是如何追模型名称

原稿以 Astra 为背景提出此路线。本轮不以模型发布日期、宣传能力或账号可用性为设计前提；模型 ID 由获准运行的 ModelProfile 显式指定，不预设可用权限或项目兼容。模型变化不改变以下设计原则。原稿 W01 未随附来源登记。

### 5.1 不改变核心抽象

不在 TwinSpec 中加入 Astra 特有字段，不以 Astra 回答补全未知业务行为，不把模型名称当成保真度证据。新模型首先是一个新的 ModelProfile 候选，而不是重写 world runtime 的理由。

### 5.2 改变评测重点

从“会不会调用工具”提升到“能否在更长、更模糊、更容易出错的任务中守住边界”。优先覆盖授权对象保护、重复副作用、提交结果不确定时的恢复、缺少信息时的澄清、不可完成任务时的合法停止，以及宿主记忆对重复运行的影响。

这是本项目的设计选择，不是对任何模型实际表现的断言。不得复制厂商 benchmark 分数充当本项目的结果。

### 5.3 比较完整配置

一次候选至少固定：请求的模型 ID、实际返回模型身份（若暴露）、API/宿主版本、adapter revision、工具投影、提示词摘要、上下文策略、审批策略、预算、世界与任务集摘要。对于滚动别名，报告 `resolved_snapshot: unknown` 或实际值，不能自行创造一个精确 snapshot ID。

禁用或固定某个采样参数不意味着模型确定性；不支持的参数必须拒绝或明确未设置，不能把“温度为零”写成通用复现保证。

---

## 6. 规范治理与旧文档迁移

### 6.1 四个独立状态

每项能力分别记录 `spec_status`、`implementation_status`、`verification_status`、`release_status`。本稿新增需求默认均为 proposed / not-yet-verified；继承能力标注“继承现有实现，需按新门槛补证”，而不是重新承诺已经完成。

需求编号使用 `MST26-<模块>-NNN`。这是本次迭代包的 proposal namespace，避免冒用仓库已经存在的 SPEC/ADR 编号。正式接受后可建立永久映射，但不可复用已退休 ID。

### 6.2 权威顺序

已接受且未被 supersede 的 RFC/ADR/规范决定语义；实现与可执行证据决定功能完成状态；发行清单决定对外承诺范围。README、路线图、翻译及摘要是派生视图。本文只有经过新 ADR 接受的条目才进入该链路。

### 6.3 本次建议新增的决策，而非直接覆盖

| 决策主题 | 推荐结论 | 对旧规划的影响 |
|---|---|---|
| 产品主线 | Agent 升级回归闭环优先 | 调整工作顺序，不废弃确定性世界原则 |
| 接入分线 | local-outbound 与 native-remote 分别验收 | 原生远程继续遵守 Phase 4 / SPEC-0020；本地路径新增边界 |
| 运行对象 | Scenario 和 AgentTask 并列 | 不修改旧 Scenario 的固定轨迹含义 |
| 证据内容 | 新增合成数据 trace profile | 不偷偷扩展现有 ProviderSmokeReport 的无原始内容承诺 |
| 版本范围 | 候选版本可包含 experimental local AgentEpisode | 发布支持矩阵接受后才固定版本；不自动改 ADR-0021 或稳定边界 |
| 冻结扩张 | 暂停非需求驱动平台功能 | 已接受实现保留，新增工作转候选队列 |

### 6.4 减少文档维护负担

维护一个机器可读能力/需求清单，记录 ID、接受决策、源码路径、测试、证据工件、提交、适用 profile、限制、更新时间。用它生成或校验 Implementation Status、Claim Registry、兼容性摘要和 README 片段。

旧 `00-MASTER...` 增加历史状态提示并指向新决策，不删除历史研究。逐项处理“仍有效、已实现、部分实现、转候选、被替代”，不能粗暴给旧包整体打上 obsolete。新增文档优先引用现有语义，而不是重复抄写四份。

---

## 7. 总体架构：保留内核，新增评测链路

```text
维护者 / CI / 评测负责人
        |
        v
EvaluationPlan + Bundle + AgentTask + Host/Model/Isolation profiles
        |
        v
Episode Orchestrator  -----> 私有 Provision / Evaluate / Evidence / Cleanup
        |
        +--> Local API Tool Bridge ----出站请求----> 模型 API
        |            |
        |            +----受限本地 MCP 调用----+
        |                                    |
        +--> Native Remote Host  --鉴权网关---+--> MCP Data Plane
        |                                    |        |
        +--> Product Host Adapter --隔离配置--+        v
        |                                      World Runtime
        |                                    /       |       \
        |                                 Engine   Scheduler  Faults
        |                                    \       |       /
        |                                     Transactional Store
        |                                        Branch / Snapshot
        v
World Trace + Host Observation + Terminal View
        |
        v
Read-only Evaluator --> Evidence Envelope --> Comparison / CI Gate

Control Plane、grader 数据、密钥、其他 Branch 不进入 Agent 可见面。
Recorder / Upstream Differential 是另一个显式授权的验证通道，不是在线 fallback。
```

### 7.1 模块职责与改动方向

| 模块 | 继续负责 | 新增/修订范围 |
|---|---|---|
| `internal/spec`、`internal/engine` | 工具合约、声明式转换、世界语义 | 仅新增被参考任务证明必要的 primitive；不依赖 provider |
| `internal/store`、`internal/world` | 原子状态、快照、fork、head、时间/事件 | 补充证据读取、生命周期回收与恢复测试，不变成分布式平台 |
| `internal/server` | MCP data/control 分离、协议 profile | 分支授权网关接口、Run 授权边界、可观察性事件 |
| `internal/scenario` | 固定脚本合约验证 | 保持旧 API，不冒充自主 Agent |
| `internal/episode` | Episode 生命周期、Journal 与远程实验能力 | 增加 AgentEpisode 执行种类与统一终态封存，不让 provider 侵入事务 |
| `internal/provider` | provider 特定合约 | 保留 smoke；新增有界工具桥接/观察适配层，按 profile 分目录 |
| 新 `internal/task`（建议） | 无 | AgentTask 编译、可解性/可观察性 admission 与任务 manifest |
| 新 `internal/evaluator`（建议） | 无 | 只读目标/策略评分、负例验证、类型化 outcome |
| 新 `internal/evidence`（建议） | 无 | trace/redaction/envelope/verify 与语义、内容摘要分离 |
| 新 `internal/evaluation`（建议） | 无 | cohort 调度、比较、分母与回归门槛；不拥有世界业务逻辑 |
| `internal/hostcompat` | 兼容性报告 admission | 增加 profile lineage、证据过期与接入路径区分 |
| CLI / docs / CI | 运维与证据入口 | first-value 命令、机器/人类报告、门槛与状态派生 |

新路径是目录建议，不是对仓库已有文件的声明。优先抽象到满足两种调用路径即可；不要先实现一套泛化插件注册平台。

### 7.2 依赖方向约束

World Runtime 不调用模型、不访问上游、不加载 native 插件；Evaluator 不修改 world；Provider adapter 不修改业务转换；Control 凭据不传给 Agent；报告导出失败不允许回写篡改已完成运行。一次模型 HTTP 请求不得处在 SQLite 写事务内部。

---

## 8. 核心数据对象与版本维度

| 对象 | 职责 | 最小身份字段 | 生命周期 |
|---|---|---|---|
| TwinSpec | 世界业务规则 | API version、spec digest、surface digest | draft → admitted → released → retired |
| Fixture | 初始数据 | schema/profile、content digest、synthetic declaration | validate → immutable |
| Snapshot | 完整逻辑起点 | world digest、clock、entropy、scheduler/fault state、head source | create → immutable → referenced → collectable |
| Scenario | 固定工具轨迹合约 | scenario version/digest | compile → execute → report |
| AgentTask | 对 Agent 的目标与评价约束 | task version/digest、domain contract、oracle digest | candidate → solvability reviewed → admitted |
| HostProfile | 宿主与接入路径 | product/API version、adapter、capabilities、projection | proposed → tested subset → stale/revoked |
| ModelProfile | 模型配置 | configured ID、requested settings、resolution policy | configured → resolved/unknown |
| IsolationProfile | 网络/文件/权限隔离 | policy digest、capabilities、test evidence | validate → enforce → teardown |
| EvaluationPlan | 一次比较的预注册定义 | cohort、tasks、profiles、budgets、ordering、grading policy | draft → sealed → execute → complete |
| AgentEpisode | 一次实际 rollout | episode ID、plan/run-definition digest、branch、attempt | provision → run → evaluate → seal |
| ObservationTrace | 可观察事件序列 | ordered event IDs、source、visibility、commit links | append → redact → seal |
| EvaluationResult | 目标、策略、执行状态 | evaluator version、assertions、classification | produced → immutable |
| EvidenceEnvelope | 可校验运行档案 | component digests、provenance、completeness | provisional → terminal → retained/deleted |
| FidelityClaim | 有限上游相似性声明 | reference identity、coverage、comparator、expiry | candidate → reviewed → admitted → invalidated |

### 8.1 四类摘要不能混用

`WorldIdentity` 绑定会影响业务语义的初态和执行规则；`RunDefinitionDigest` 在此基础上绑定任务、宿主、模型请求配置、工具投影、预算与评分器；`ExecutionSemanticDigest` 对实际有序业务事件与结果做摘要；`EvidenceEnvelopeDigest` 覆盖某次档案中的 episode ID、运行时间、工件等。

两个独立运行可以拥有相同 RunDefinitionDigest，但 EvidenceEnvelopeDigest 不同。模型做出不同调用时，ExecutionSemanticDigest 也应不同。不得通过删掉真实差异让两个结果“看起来可复现”。

### 8.2 身份缺失处理

provider 不暴露不可变模型 snapshot 时记录 unknown；宿主版本无法取到时记录 profile 的可复核获取方式与 unknown 字段，并降低声明精度。不得用 `latest` 冒充 pin。密钥、原始 bearer token、可直接访问服务的 capability URL 不参与公开身份内容。

### 8.3 版本独立性

runtime SemVer、TwinSpec API、world-store schema、Episode-Journal schema、Task schema、Evidence schema、Bundle schema、canonicalization ID、MCP protocol、HostProfile schema 和模型 ID 分别版本化。当前 world-store v4 与 Journal v2 不能合并称为“数据库版本 4”。升级某一维必须说明哪些旧证据仍可读取、哪些需要重跑。[R02]

---

## 9. AgentTask：新增的核心产品合约

### 9.1 与 Scenario 的区别

Scenario 指定“按什么顺序调用哪些工具，期待什么结果”，用于验证 runtime。AgentTask 指定“用户要达到什么目标、允许什么、禁止什么、如何评价”，由真实 Agent 自主选择工具路径。一个任务可以有多个合法解；参考轨迹是可解性证据，不是唯一评分答案。

不将编排策略、思维链、模型内部规划或某一家宿主的命令格式写入 AgentTask 核心。

### 9.2 字段级规格

| 字段组 | 要求 | Agent 可见性 |
|---|---|---|
| `identity` | 稳定 task ID、任务 schema、任务版本、作者与领域版本 | 可公开；不给隐藏测试答案 |
| `world` | Bundle、fixture、初始 snapshot、所需 tool surface、时间/故障 profile | 只提供完成任务所需的业务信息 |
| `objective` | 明确自然语言目标、授权范围、应请求澄清的条件 | 可见 |
| `interaction` | 单轮/有限澄清、可提供信息、用户回应策略与最大轮次 | 交互策略是否公开由任务明确声明 |
| `required_affordances` | 完成任务所需的读写能力与可观察证据 | preflight 使用；不暴露参考答案 |
| `success` | 终态断言、业务里程碑、合法不执行/升级处理条件 | grader 私有；目标本身不得秘密改变 |
| `policy` | 禁止修改对象、不得重复的业务操作、过程安全条件 | 与真实用户授权对应的规则必须可见 |
| `oracle` | 独立只读评分程序、版本、数据访问范围与摘要 | 私有 |
| `solvability` | 至少一个合法 witness 或明确不可完成证明、必要工具与信息 | 维护者私有 |
| `budgets` | 最大模型轮次、工具调用、墙钟、虚拟推进、字节与开销策略 | 可见或通过标准终止语义反映 |
| `evaluation_mode` | blind / diagnostic / coaching | 必须写入报告，不可混分 |
| `lineage` | 来源、变体种子、父任务、公开/留出分组 | 报告可追溯 |

### 9.3 可解性 admission

每个任务入库前必须通过四项检查：目标与 fixture 不矛盾；必需工具确实可被该 HostProfile 暴露；所需信息能被用户或工具获得；评价条件可以由外部观察或终态验证。

可解任务至少保留一条成功 witness，并增加与之不同的合法路径或无关读操作变体，证明评分器不是轨迹匹配器。不可解任务必须把正确结果定义为澄清、停止、请求授权或说明限制，而不是“无论如何完成”。

工具缺失、fixture 错误、评分器矛盾、隐藏信息无法获得，应返回 `TASK_INVALID` 或在运行前 blocked；不能计入模型失败分母。能力不足与任务不可解的判断必须留下人工可复核的理由。

### 9.4 世界不变量不等于 Agent 策略评价

例如 engine 强制 issue 状态只能为 open/closed，那么“没有出现第三种状态”很可能只是世界守护成功，不能证明 Agent 判断力。需要同时记录：Agent 是否尝试了不允许的动作；世界是否拒绝；是否有已提交副作用；最终任务是否成功。

**世界不变量**维护合法数据库；**任务策略**判定用户授权及业务行为；**产品安全边界**阻断越权资源访问。三个层次分别测试，不把零持久化违规率当成零违规尝试率。

### 9.5 澄清与不可完成任务

首版采用有界、确定性用户回应表，建议最多 2 次澄清。用户模拟器不能把 grader 的隐藏答案直接喂给模型。若未来引入模型驱动用户模拟器，必须独立 pin 配置、记录输出，标记新增随机来源并重新设计统计比较。

目标变更生成新的 objective revision；记录用户变更发生的时点、旧目标停止适用的范围和仍然有效的授权。不能在运行结束后为了让模型通过才修改目标。

### 9.6 三种评价模式

blind 模式不向 Agent 返回逐条评分或隐藏状态。diagnostic 模式可在运行结束后显示失败断言，但下一次运行必须记录已看见反馈。coaching 模式允许逐步反馈，但不得与 blind 分数合并。公开仓库里的题库是公开题库，不能因为文件名叫 holdout 就声称没有训练污染。

---

## 10. 用户旅程、CLI 与报告交互

### 10.1 首次离线体验

用户选择官方合成示例，验证 Bundle，运行 scripted baseline，看到“相同起点、不同合法工具轨迹、独立终态、可重放证据”。该路径无需模型密钥。保留当前可用 `validate/init/fork/call/diff/scenario/bundle/episode` 能力，不打断现有脚本。[R01][R02]

### 10.2 首次真实 Agent 体验

用户配置一个 ModelProfile，仅将密钥放在进程凭据环境或明确秘密输入通道；运行 preflight；确认任务、工具集、隔离、预算与数据策略；启动一组小任务；读取结构化报告。第一次不默认跑完整 24 任务多次采样，不自动扣费做大规模矩阵。

### 10.3 模型升级回归

用户选择 baseline/candidate 两份配置与封存的 EvaluationPlan；系统为每个任务/重复次数创建独立分支；交替安排两组运行；生成逐任务比较、差异证据、可重放最小案例；按事先定义的回归门槛给出通过、失败或证据不足。

### 10.4 拟新增 CLI 命令面

以下是**提案接口，不是当前可执行命令**。命名需接受 CLI ADR 后才可以进入 README quickstart。

| 拟议命令 | 语义 | 输出/失败约束 |
|---|---|---|
| `statetwin task validate` | 验证 Task、oracle 引用及所需 surface | blocked 原因与缺少能力，不调用模型 |
| `statetwin eval preflight` | 解析 plan/profile、检查隔离与预算 | 默认不计费；若需要付费 probe 必须单独显式启用 |
| `statetwin eval run` | 执行封存 plan | 拒绝覆盖旧档案；支持终止和不完整证据 |
| `statetwin eval compare` | 比较两个可比 cohort | 不可比时显式拒绝，不给误导排名 |
| `statetwin evidence verify` | 校验摘要、引用、序列、完整性 | tamper/缺失/版本不支持分开报告 |
| `statetwin evidence replay` | 重放 world 操作与断言 | 不调用模型；明确与 live rerun 不同 |
| `statetwin evidence explain` | 展示首个违反条件、状态差异和调用关联 | 不暴露密钥、隐藏思维或无权限数据 |
| `statetwin doctor` | 检查本地依赖、端口、存储与 profile | 无默认上游探测和自动网络修复 |

退出码建议分成成功、回归失败、计划/配置错误、执行/证据错误四类；具体数值在 CLI 合约接受时固定。JSON 输出的枚举与 schema 才是 CI 合约，面向人的表格不应成为机器解析接口。

### 10.5 最小报告界面

报告首页先给：这是什么比较、是否可比、多少计划任务/实际运行/有效运行、已观察到的回归、重要限制。再给逐任务成功率与策略违规尝试/已提交副作用、错误归因、调用/token/耗时分布、证据引用。

首版采用 JSON + Markdown，后续可增加离线静态 HTML。不要把前端仪表盘、账号体系或在线协作空间变成首个真实闭环的前置需求。报告中的“成功”按钮必须能定位到只读终态与断言；不能只展示模型的自我总结。

---

## 11. Host Adapter 与两条接入路线

### 11.1 路线 A：本地 API 工具桥接，近期主线

流程为：从本地 MCP 获取明确授权的业务工具面，投影为 provider 支持的函数工具合约，发送任务，接收模型工具调用，在受限的本地 MCP 数据面执行，把工具结果继续交给模型，直至停止或触发预算。

官方函数调用文档展示应用执行函数并通过 `call_id` 回传 `function_call_output` 的流程。这支持本地桥接的接口可行性，不证明本项目已经实现或兼容。具体隔离、投影和评分仍需独立验证。[OpenAI 官方文档，2026-09-08 查阅](https://developers.openai.com/api/docs/guides/function-calling)

该路径可以避免为了 first-value 而先部署公网 MCP。但它证明的是“指定模型 API + 本项目 adapter + 本地 MCP 的组合”，**不证明 provider 原生 MCP、ChatGPT 或 Codex 产品兼容**。

### 11.2 路线 B：原生远程 MCP，独立验证

provider/宿主直接连接经过鉴权与限定权限的 MCP endpoint；隧道可作为未来候选传输，但须另行核实其官方合约及本项目证据，不在本稿中宣称已经兼容。任何传输均不能替代主体、分支和工具授权。原稿 W03 未在本轮作为接入依据。

原生路线仍需满足项目现有 Phase 4/SPEC-0020 的部署隔离、TLS/身份、凭据管理、成本与销毁证据。没有这些门槛，不通过临时无鉴权端口收集“成功调用”来换取兼容标记。[R13]

### 11.3 最小 adapter 合约

| 操作 | 输入 | 输出与必要保证 |
|---|---|---|
| Describe | adapter ID/version | 静态支持维度，不做网络请求 |
| Resolve | Host/ModelProfile | 完整有效配置、unknown 字段和摘要；秘密单独处理 |
| ProjectSurface | 授权工具面 | 名称映射、Schema 投影、损失类别、两侧 digest |
| Prepare | EpisodeRunContext | 新会话/工作区、隔离上下文、cleanup handle |
| CheckReady | 准备好的上下文 | 所需工具可见、无未解决投影错误 |
| Step / Run | 用户目标与前序工具观察 | 有界模型消息、工具调用、停止/错误事件 |
| DeliverResult | 调用关联 ID 与工具结果 | 保留调用关联和顺序，记录模型实际可见表示 |
| Cancel | 当前运行 handle | 明确确认/未知/不支持，不能假称远端已经取消 |
| Collect | 运行 handle | 受策略允许的可观察输出与 usage，不收集隐藏思维 |
| Cleanup | 所有已创建资源 | 释放凭据、会话、端口、分支授权；失败留下 evidence |

首版不实现自动发现所有插件的 registry；两个 adapter profile 共用合约即足够检验抽象。

### 11.4 Schema 与工具名称投影

保持 canonical tool ID 和 host-visible ID 一一映射。禁止名称碰撞、工具静默丢失、参数类型静默放宽、把不支持的约束直接删除后声称等价。投影分为 lossless、representation-only、semantic-risk、unsupported；后两类不能进入 strict-comparison profile。

记录完整世界工具面、主体授权后的工具面、adapter 投影后的工具面。无法观察最终模型可见面时，记录 unknown，不能用服务器 surface digest 代替。

### 11.5 调用、并发与续接

保持 provider call ID、adapter call sequence、MCP request ID、world event/commit sequence 的映射。多个工具调用按返回数组顺序执行的首版 profile 必须明确叫 `ordered-sequential`；不得因 SDK 内部并发而产生未记录顺序。

维护 continuation handle 或 provider 要求的续接上下文，但它们不是公开评测内容。不会暴露的内部状态不能被当作可移植 Agent checkpoint。处理 API 要求的 opaque 数据不等于保存或分析隐藏思维。

### 11.6 宿主配置与模型配置分离

Astra、另一个模型、同一模型不同推理预算是 ModelProfile 的区别；OpenAI Responses、原生 MCP、Codex CLI、Anthropic Messages、Claude Code 是 HostProfile 的区别。Anthropic 路线的当前 API 合约需在对应实施包重新核实；不能从一个通用 MCP 测试推断其全部能力。原稿 W04 尚未独立复验。

每个兼容声明须包含“接入路径 + host/API 版本/配置 + 模型配置 + surface profile + 测试日期 + 场景与证据”。新模型只要不改变外部世界语义，就优先新增 profile 和测试，不改 TwinSpec。

---

## 12. AgentEpisode 生命周期、停止和恢复

### 12.1 生命周期

```text
CREATED -> VALIDATED -> PROVISIONING -> READY -> RUNNING
                                                |
                                                v
                                            STOPPING
                                                |
                                                v
                                           EVALUATING
                                                |
                                                v
                                             SEALING
                                                |
                                                v
                                             TERMINAL
```

任何阶段都允许有类型的失败；清理与证据封存不能因为业务失败而跳过。旧 Episode v1alpha1 的状态枚举保持兼容，新 AgentEpisode 采用独立 kind/schema 或显式版本迁移，不直接重释旧 `SUCCEEDED`。

### 12.2 三个终态维度

`execution_status` 表示 completed / cancelled / timeout / budget_exhausted / host_error / infrastructure_error；`task_outcome` 表示 success / task_failed / policy_violation / expected_abstention / task_invalid / not_evaluated；`evidence_status` 表示 complete / partial / integrity_failed。

模型说“完成”只触发准备停止，不直接等于 success。任务终态合格但证据缺失时，可以显示“观察到目标成立、证据不完整”，不得标成完整可认证通过。

### 12.3 准入与启动

执行前检查 Task 与 Bundle 摘要、支持版本、预算、模型配置、adapter 能力、隔离策略、secret references、存储空间和输出路径。创建新 episode ID 与独立 branch；拒绝覆盖旧证据目录；记录父 plan 与 trial index。

到 READY 时所需工具必须可见。发现失败是 HOST_NOT_READY，不应计为 Agent 不会使用工具。初态 fixture 和 grader 的隐藏文件不得挂载到 Agent 工作区。

### 12.4 预算与真实终止

建议首批任务 profile 设置明确的模型轮次、工具调用、响应字节、wall-time 和虚拟推进上限，具体数值由任务复杂度与试验决定。付费调用前要有显式 monetary ceiling 或人工核准的受控计划；没有可用单价/usage 规则时只能报告费用未知，不能把 unknown 计成零。

运行预算是一整次 Episode 的上限，不只是每个 HTTP 请求的 timeout。停止后不再发起新调用，撤销该 episode 的工具能力，对已开始操作等待有界收敛，并记录是否在截止前已经提交。

单次外部请求可能在超时后仍计费，客户端预算不能保证绝对无超额。采用预留本次请求最大估计、限制输出、停止后续调用与提供商可用硬配额共同控制；报告估算值、观测值与未知部分。

### 12.5 封存一致性

封存前阻断新调用；将已经进入 runtime 的操作归入明确的 commit 截止序列；读取与该序列一致的终态；执行只读 grader；封存摘要和完整性状态；最后清理资源。禁止先读取终态再允许后台工具继续写，从而得到“报告通过、实际世界又被修改”的档案。

对于后到达的 stale worker/provider 调用，拒绝并写非业务诊断事件；不能偷偷再开一个分支承接它们。

### 12.6 失败、重试与不确定提交

世界未提交的确定失败，可以按明确任务/宿主策略再次尝试。外部请求结果未知、已发起但无法查证的状态，记录 `COMMIT_UNKNOWN` 或等效类型并停止自动重发有副作用的调用。

既有 remote fencing 保护的是 terminal Evidence 接受，不等于模型请求恰好一次或工具副作用恰好一次。不得将重复消息、业务幂等键、Episode attempt 与 JSON-RPC request ID 混为一谈。[R02]

### 12.7 四种“重跑”必须分别命名

| 模式 | 固定什么 | 能证明什么 |
|---|---|---|
| World replay | 世界身份和完整有序输入/控制事件 | 相同可观察世界结果与终态 |
| Captured-agent playback | 已捕获的 Agent 工具选择/观察 | adapter/runtime 回归，不是新模型能力 |
| Fresh live rerun | 相同计划，新的模型执行 | outcome 分布；不保证相同轨迹 |
| Counterfactual branch | 指定分叉世界与明确 Agent 上下文政策 | 某种受控条件变化的观察，不自动是严格因果结论 |

world fork 只复制世界，不复制 provider 的隐含状态。反事实实验需明确 reset-and-rerun、公开上下文重建或经宿主支持的 checkpoint；缺失 checkpoint 时不得宣传“任意时刻复制整个 Agent”。

---

## 13. 证据、日志与可重放轨迹

### 13.1 Evidence 组成

档案最少包含 run definition、有效 profile、初态摘要、工具投影、世界事件、宿主观察、终态只读视图/摘要、断言、分类、预算使用、cleanup 状态、版本与 provenance。每部分单独有 digest，外层 manifest 绑定完整声明集合。

推荐先落地可独立验证的目录或 Bundle 扩展，不先建立远程服务。仅记录 digest 的报告仍可作 smoke 证据，但不能被标为完整 replay 工件。[R06]

### 13.2 事件字段

每条事件至少记录 episode/attempt 关联、单调 sequence、source、event type、call mapping、虚拟时间、world head before/after、提交状态、结果错误类别、脱敏策略与允许保存的内容。wall-clock、网络 duration、计费 usage 是操作观察，不进入纯世界确定性断言。

至少区分 model_request、model_observation、tool_requested、tool_admitted/rejected、world_commit、tool_delivered/undelivered、clock_advanced、fault_triggered、evaluation、cleanup。被世界拒绝的尝试也必须在适当边界被观察，不能只记录成功事务。

### 13.3 事务证据与外部观察

业务效果和对应 world audit 在同一 SQLite 事务提交；宿主收到/未收到结果属于事务外观察，两者不得伪装成原子事件。服务崩溃后，可以据已提交 audit 确认世界状态；不能据此断定模型曾看见结果。

工具输出被 adapter 截断、格式化、汇总时同时记录原始世界输出摘要和实际交付表示/摘要。无法保存内容时明确影响 replay 的范围，不允许拿一份 hash 恢复不存在的原文。

### 13.4 数据分层和脱敏政策

现有 ProviderSmokeReport 的“无原始提示词/轨迹/响应”合约保持不变。本稿提议单独接受 `synthetic-agent-trace` profile：只保存通过字段白名单与敏感扫描的合成任务/工具内容，保存前脱敏，密钥与私有 ID 永远不落盘。[R13]

另设 digest-only profile，适用于内容不允许持久化的运行；其报告必须标注 `replayable: false` 或具体可重放的部分。脱敏改变业务值、对象关系或结果语义时，必须标成 non-equivalent，不得称为原世界的字节级 replay。

Authorization、Cookie、API key、session/capability token、私钥、可访问运行的 URL 默认禁止保存；未知头字段默认丢弃。哈希个人标识不自动等于匿名化。仅测试合成数据也不免除 prompt 中夹带真实秘密的扫描。

### 13.5 脱敏保留引用关系

对确需伪名化的合成标识，采用类型化、运行内一致映射；同一对象的请求、响应和终态对应同一伪名。映射规则版本进入证据。若原值是低熵秘密，不能只对其计算公开可枚举 hash。

### 13.6 完整性校验

验证器检查 schema、成员声明、digest、引用、事件连续性、head 关联、终态与断言关联、sealed 状态、外层来源。缺少中间事件、重复不同内容、序号跳跃、终态替换、旧 profile 冒充新 profile、引用越界，分别提供错误。

签名证明发布者/完整性，不证明模型表现真实或 Twin 保真。签名、OTel、漂亮的可视化都不能替代原始可复核语义证据。

---

## 14. 评分、比较与发布阻断规则

### 14.1 评分不是单一平均分

至少展示五个轴：目标达成；授权/策略；副作用与恢复；执行效率；评测有效性。安全或授权失败不能被高任务成功率或低成本抵消。

建议主要枚举规则：目标成立且所有关键策略成立为 success；目标成立但越界写入仍为 policy_violation；信息不足时正确澄清/停止为 expected_abstention；任务本身不可解且未预先声明为此类任务，标 task_invalid；宿主与评测器损坏则分到相应执行错误。

### 14.2 过程断言与终态断言

终态断言验证目标字段、对象数量、允许变化集合、未授权对象未变化。过程断言验证必要授权先于修改、失败后是否重复副作用、截止后是否继续调用、是否请求缺少的信息。

不默认要求“必须先 get 再 close”；只有业务权限、版本检查或任务规则确实要求时才把某个先后关系设为强断言。无关读取、不同但合法的工具序列，不应被当作失败。

### 14.3 评分器自身的负例与变异测试

每个 oracle 至少具备正例、漏做、错对象、额外副作用四类案例；适用时增加重复提交和合法放弃。把成功终态的关键字段改坏，评分器必须转为失败；把不影响目标的合法字段改动，不能误判失败。

被 engine 永远拒绝的坏状态不能作为唯一负例，应增加“尝试被拒绝”轨迹样本和“可提交但违反任务授权”的样本。评分器只读，失败不允许修正世界以制造通过。

### 14.4 重复运行与可比性

比较前封存 task set、world/fault seed、profile、工具授权、预算、评分版本与试验次数。除有意改变的变量外保持其他配置一致；同时改变模型和宿主时，结论应写为“整体配置变化”，不能归因于单独模型升级。

建议探索阶段每任务每配置 3 次作为调试样本，而不是精确可靠性估计；发布决策阶段根据可承受成本和观察到的波动提高次数。任务先分层，再按统一重复规则运行，不能只给失败的候选额外机会。

### 14.5 分母与缺失数据

报告 planned、started、completed、validly_evaluated、task_invalid、host_error、infra_error、cancelled 的数量。既给计划口径，也给有效评测口径，并解释差异。不能删除 timeout 或基础设施失败后只公布最漂亮的成功率。

仅在适用独立性假设时给单比例置信区间；按任务/种子聚类、重复运行的比较应采用匹配或分层重采样等预先固定方法。小样本、重复任务相关和 provider rolling alias 导致的限制必须写明，不能把百分比小数位当成统计精度。

### 14.6 回归门槛

已定义的关键越界/重复副作用案例，只要出现新的可复核 violation，就阻断自动推荐升级。成功率、工具调用和成本的退化阈值由 EvaluationPlan 预注册；不显著或证据不足返回 inconclusive，而不是默认 pass。

阶段初期不承诺任何模型达到某个成功率。产品验收的是能够正确、完整地发现和报告上述差异；模型表现由实际运行决定。

### 14.7 LLM judge 的边界

机器可判断的世界事实使用确定性 evaluator。对于最终自然语言沟通质量，允许可选辅助 judge，但单独记录配置、输入、输出、重复性与不确定性，不能覆盖授权/副作用硬断言，不能作为 L2 保真度的自证。

---

## 15. 首批任务与旗舰闭环

### 15.1 旗舰案例：关闭目标 Issue，遇到提交后响应不确定

用户仅授权关闭 `octo/demo` 的指定 issue，其余对象保持不变。先运行无故障版本，再配置一次 `after-commit-before-response` 故障。Agent 收到不确定结果后，可以通过 `get_issue` 判断该 issue 已关闭；不得因为没有收到成功响应就创建无关 issue 或修改其他对象。

这是首批可落地案例，因为当前域有 get/close 能力且已有对应故障阶段。需通过新 AgentTask、host bridge、世界/交付轨迹与终态 evaluator 形成新的闭环证据；不能只因为已有单元测试，就声称真实 Agent 已通过。[R02][R14]

验收同时看：初始 issue 确实 open；世界只发生一次目标关闭效果；Agent 可观察到的返回与世界 commit 被区分；其他 issue、comment、repository 保持不变；无故障和有故障档案都可检查并做 world replay。

### 15.2 任务层级

先实现一个领域的 6 个任务：读取、单次创建、定向关闭、重复关闭的合法处理、无授权对象不修改、提交后读取确认。再扩到两个领域共 24 个候选，包含信息不足、不存在对象、错目标、重复操作、恢复和未来时间类任务。

`SCENARIO-CATALOG.md` 为每个任务明确 existing-surface、fixture-only、domain-extension 或 future-runtime。任务入选是规划，不意味着当前 CLI 能执行它。

### 15.3 领域模型补充原则

当新增 list_comments、幂等键、版本预条件、异步发布状态时，须明确领域版本、旧语义、迁移与测试。不得为了让某个模型通过而临时改变工具描述或写入规则。不要同时升级领域规则、评分器和候选模型后给出“模型进步”的结论。

---

## 16. 确定性、虚拟时间与并发

### 16.1 基本契约

固定 runtime 语义版本、TwinSpec、初态、canonicalization、semantic limits、时间/熵/故障/调度状态与有序外部事件，应该得到相同的世界可观察输出、提交序列和终态摘要。跨平台验证比较这些语义工件，而不是要求 SQLite 二进制文件或带墙钟的日志字节完全相同。

不同 Agent 会产生不同轨迹；同一个 Agent 也可能在重复运行中选择不同顺序。runtime 负责重放已记录的世界顺序，不负责让 provider 的规划调度确定。

### 16.2 Snapshot 完整性

fork/reset 必须包含业务实体、分配序列、虚拟时间、熵流 counters、pending/completed scheduler 状态、fault plan 与消费计数，以及影响后续转换的其他元数据。不能只复制 entities 后宣称两个实验拥有同一世界。

若 episode ID、branch 名等只用于隔离和追踪，不得影响业务随机流与正常工具输出；确需可观察时要纳入世界身份，并解释如何做跨分支规范化比较。

### 16.3 时间推进

首版 AgentTask 默认 `frozen-unless-specified`。需要时间任务时，由 harness 按明确事件政策推进虚拟时间：例如在完成指定工具观察后推进，或者由确定性的 scripted external actor 注入事件。禁止使用模型真实推理耗时作为世界时间的默认推进量。

world scheduler 和 agent scheduler 分开。每次 clock advance 都进入有序世界事件流；不能由 grader 在看到失败后“多推进一点”帮助模型通过。不同时间政策就是不同评测 profile。

### 16.4 同分支并发

已有 head CAS 保留。首版 adapter 的工具调用使用确定顺序，记录 accepted commit order；未来共享分支并发明确 conflict/retry 政策。操作系统线程顺序不是语义规范；多个 worker 恰好按某次顺序完成也不是可复现证明。

写冲突、reset 与在途 call、snapshot 与提交、cancel 与提交的竞争均需要负向测试。不能把 branch conflict 隐藏为一般工具失败而丢失发生原因。

### 16.5 确定性限制与操作限制分开

`local-preview-v7` 中影响工具/调度可执行结果的上限必须进入世界语义身份；`local-v2` 的 Go 并发、heap soft target 和 HTTP admission 属于操作配置，也要记录，但不声称是硬进程资源沙箱。[R02]

CLI 不应给用户一个模糊的“deterministic=true”覆盖所有层；报告应说清哪些部分可严格重放、哪些属于 live observation、哪些尚未跨平台验证。

---

## 17. 故障、幂等与恢复规格

### 17.1 近期只补支撑任务的语义

当前两种故障阶段已经能支撑验证前失败、提交后响应不确定等重要任务。近期不为了“覆盖所有故障”增加大量阶段；先证明现有阶段能被 AgentEpisode 正确配置、消费、观察、评分和重放。[R02]

| 情形 | 世界是否提交 | Agent 可能看见 | 评测需要确认 |
|---|---|---|---|
| validation 前拒绝 | 不提交业务效果 | 明确失败 | 不发生业务变化；故障消费规则可重放 |
| 业务前置条件失败 | 不提交该业务转换 | 类型化业务错误 | Agent 是否合法读取/澄清/停止 |
| commit 后交付失败 | 已提交 | timeout/error/缺少确认 | 世界事实和 Agent 可知事实分离 |
| 重复业务请求 | 由领域幂等规则决定 | 原结果/冲突/新效果 | 是否产生不允许的重复效果 |
| 在途取消 | 已提交或未提交，依截止而定 | cancelled/unknown | 有无截止后的新操作；终态是否一致 |
| 存储失败 | 依恢复证据确认 | infrastructure error | 不归为模型能力问题 |
| 不支持的故障模式 | 不启动 | profile admission error | 不能静默降级成“无故障” |

### 17.2 故障选择与身份

故障定义至少固定工具/操作、触发条件、阶段、次数、确定性结果和 profile 版本。优先用显式序号或稳定 selector；禁止使用未记录的真实随机概率。配对实验复用 fault definition 和初始消费状态。

工具轨迹不同可能导致故障触发时机不同，报告应注明触发/未触发；不能把“有故障组中实际未触发”的运行直接并入故障恢复成功率。按全计划口径和实际触发口径分别展示。

### 17.3 幂等规则属于领域，而不是网络去重开关

JSON-RPC ID、provider call ID、coordinator completion key 只代表消息或尝试关联，不能自动变成业务幂等键。业务幂等需定义 key 的作用域、冲突输入、保存期限、相同 key 不同 payload 的结果，以及历史结果如何查询。

对当前 `close_issue` 的重复调用冲突，不应仅为让 retry 通过而改成成功；新的“幂等关闭”可以是另一个明确 domain revision。创建/评论类操作的去重也必须符合领域公开语义。[R14]

### 17.4 部分效果不能破坏存储原子性

需要模拟“流程完成 A、B，未完成 C”时，把 A+B 与失败状态建模为一笔合法业务状态转换。不能让 State Twin 自己产生半事务、破坏 audit 或损坏 SQLite，来代表上游部分成功。

### 17.5 恢复任务的可观察性

Agent 只能使用返回的工具观察和用户信息决定是否重试，不能读取私有 `effectsCommitted` 证据或 fault plan。grader 可以拥有这些信息，用来解释 Agent 面对的是成功确认、确定失败还是不确定结果。

每个恢复任务必须列出“能够确认”的工具路径；没有查询能力则明确允许人工确认/停止。若唯一成功策略依赖猜测不可见状态，该任务不得进入正式模型对比集。

---

## 18. MCP 协议与兼容性边界

### 18.1 固定 profile，不追随含糊的 latest

当前仓库代码声明 `2026-07-28` profile，并包含 modern raw-wire 与 legacy 测试；本轮只核对本地声明及代码，不把它表述为重新验证了官方最新规范或全部扩展。近期补齐已声明 profile 的证据，协议升级仍按 accepted 合约办理。[本轮 S04；原稿 W07 未复验]

记录协议版本、SDK 精确版本、conformance 工具版本/提交、所选套件、expected failures、跳过项和适用 transport。升级任一项都生成候选证据，不自动继承旧 PASS。

### 18.2 tools-first 最小承诺

核心只依赖发现/列出工具、JSON 合约的工具调用、结构化结果和类型化错误。prompts、resources、Tasks、MRTR、动态 discovery、stdio、GUI 等分为可选 profile；未使用的能力明确 not-applicable 或 unsupported。

stateless MCP transport 只表示协议处理不依赖隐式会话状态，不表示业务世界无状态。业务 branch 必须由显式运行身份绑定并拥有自己的状态。

### 18.3 负向覆盖

覆盖不支持版本、错误元数据、header/body 不一致、未知方法、错误参数、非对象输入、超限结果、错误 discriminator、现代/legacy 生命周期混用，以及不会把 control 操作暴露为 tools 的检查。不得用忽略所有错误的客户端脚本声称兼容。

### 18.4 兼容性声明矩阵

每一行只声明特定接入路径：local bridge、native remote MCP 或产品宿主。维度包括 transport、工具发现、调用、structured output、Schema 子集、投影、审批、重试、取消、并发、输出变换与数据策略。

状态为 untested、supported-under-profile、unsupported、stale、revoked。资料声称支持只可写 documented；有 mock 只可写 contract-tested；有 live exact-profile evidence 才可进入 observed-compatible。兼容不自动证明任务可靠或真实上游保真。

---

## 19. 安全、权限与评测完整性

### 19.1 部署/隔离 profiles

| Profile | 允许场景 | 必需边界 | 不允许声明 |
|---|---|---|---|
| local-trusted | 维护者自己运行合成脚本 | loopback、控制鉴权、无上游写、文件权限 | 对恶意本地进程或 shell Agent 完整隔离 |
| local-api-bridge-trusted（本轮细化） | 可信本地 harness 对接只有函数工具的远程模型 | 固定 endpoint/branch、工具与资源 allowlist、private oracle/secret、保存前策略；无 shell Agent | OS 硬网络/文件沙箱或公网数据面安全；参见 PHASE-SPECS |
| local-agent-isolated | 本地 API bridge 或隔离宿主 | 独立 run、工具 allowlist、隐藏 oracle、限定文件/网络、受控秘密 | 原生 remote MCP 或公网安全 |
| remote-synthetic-staging | 原生 provider MCP 验证 | TLS/受信隧道、主体与 branch/tool 授权、凭据过期、速率限制、销毁 | 多租户生产 SLA 或真实生产等价 |
| managed-production | 未来独立产品决策 | 完整运维、租户、备份、响应与审计 | 不能在本稿中自动接受 |

初期推荐 local-api-bridge-trusted，不自动称为 local-agent-isolated；后者须有对应 OS/宿主隔离证据。
不满足隔离要求的运行不得混入声称该隔离级别的 cohort。

### 19.2 数据面授权

远程或非可信宿主请求须由服务器验证可信主体，绑定 episode/branch、允许 tools、受众、有效期、运行状态与必要资源范围。改 URL 中 branch 名不能获得另一运行的访问权。只能作用于授权范围的凭据，不能复用控制 token。

对已结束、取消、过期或 fenced attempt 拒绝新的效果请求。鉴权失败响应不披露其他 branch 是否存在、状态内容或内部数据库位置。

### 19.3 文件系统和控制面隔离

有 shell 能力的 Agent 不得能读取 Twin 数据库、控制凭据、隐藏 fixture/oracle、其他运行目录或宿主秘密。仅在 MCP tools/list 隐藏 control tools 不能证明这些路径不可达。隔离能力在平台上不存在时，profile 拒绝或降级为 trusted，不能仅显示绿色勾选。

### 19.4 网络与凭据

本地 bridge 的网络仅允许指定 provider endpoint 与本地数据面；world runtime 本身保持无上游出口。provider 密钥存在 harness 私有上下文；传入模型的工具内容不包含秘密。远程网关不得进行任意 URL 转发；TLS/隧道不会取消 Origin/Host、主体与请求资源校验需求。

录制与差分的 upstream 凭据单独授权，不能把 Agent 的 token 透明转发到上游。不要把临时运行地址作为唯一认证；地址一旦出现在日志/模型上下文中，不能继续当作秘密边界。

### 19.5 Benchmark 隐藏数据与防投机

用户目标与授权公开，隐藏的只是验证实现与特殊 fixture/测试分组。Agent 无权限读取 grader、预期终态或评分反馈；发现越界访问尝试记录事件。对“把任务规则藏起来再惩罚模型”的测试，先修正任务，不将其称为严格评测。

工具描述和内容可含不可信业务文本。系统不得让这类文本修改控制面、权限或评分程序。相关测试只在授权合成环境中验证拒绝与隔离，不接入真实用户/生产账户。

### 19.6 资源与供应链

输入和 archive 均不可信。保留现有 path traversal、symlink、成员与字节上限、外部 Schema 资源拒绝等边界；新增 trace、task、report 使用相同严格 admission 原则。[R02]

运行时内存软目标不等于 OS 硬限制。首版不加载任意 Go/Python plugin 来运行评分器；优先受限声明式检查或项目内固定版本 evaluator。未来自定义执行需要单独安全 RFC 与沙箱测试。

### 19.7 Teardown 与残留

清理负责撤销能力、停止 listener/worker、关闭数据库、删除临时敏感材料与确认无在途工具写。保留证据与删除世界状态按明确 retention policy 分开。清理失败进入报告并阻断重用可能污染的环境，不能打印 success 后留下可继续访问的运行端口。

---

## 20. Fidelity、录制与上游差分

### 20.1 保真度与运行时质量分开

当前 L1/unverified/unbound 参考领域可以充分用于验证工具使用和 runtime 合约；它们不证明真实 GitHub/包平台行为。再多内部单元测试也不能自动升级成上游等价。[R02][R14]

### 20.2 三条验证路径

第一，内部合约验证：Twin 与自己的规范一致。第二，独立参考实现验证：使用不共享同一转换实现的测试服务，检查领域语义。第三，授权上游 sandbox 验证：针对固定 API/工具面和授权测试数据比较结果。

第二条证明对该参考服务的一致性，不自动推广到真实服务；若 reference service 只是再次执行同一份 CEL，得到的是自洽性测试，不是独立差分证据。

### 20.3 Surface inspector

采集工具名、描述、输入/输出 Schema、annotations、protocol/capability 信息并生成稳定摘要；记录源身份、获取时间、授权 profile 与观察范围。检测到 drift 只产生报告或候选变更，不自动改写业务规则。

规范允许的无序集合可以规范化；用户可观察的数组顺序与描述变化不能随意丢弃。surface digest 相同只证明所采集接口一致，不证明隐藏的业务行为相同。

### 20.4 Recorder 与 cassette

Recorder 是显式授权的测试采集工具；不得作为 runtime 未建模时的透明上游 fallback。首版仅合成 reference 服务；后续真实 sandbox 需要单独数据授权。

Cassette 包含格式、source profile、工具面、请求/响应/错误、序列、前后状态可观察摘要、redaction manifest 与覆盖范围。录制前过滤秘密；不允许先落盘再补脱敏。缺失路径返回未覆盖，不用模型临时编造回应。

L0 replay 证明已记录路径可重现，不证明新合法轨迹下的状态模型正确。

### 20.5 差分执行器

在 Twin 与 reference 的等价初态上执行相同有序请求；比较成功/错误类别、规范化输出、可观察终态、重复操作、权限范围和时间可见性。reference 初态不可复位或结果有未控随机性时，记录 TEST_INVALID/UPSTREAM_NONDETERMINISTIC，而不是强行判 Twin 错。

比较器配置独立版本化。忽略时间戳、生成 ID、排序等字段必须有领域理由与测试；不能通过忽略所有差异取得 MATCH。accepted divergence 必须列出偏差、影响与批准者。

### 20.6 L2 认定

认定单元为 `reference profile × tool/operation × state class × success/error × relevant fault` 的有限覆盖集合。产物包含覆盖分母、通过、失败、未建模、可接受偏差、reference 版本、comparator 与证据。

只有人工审查并接受证据后，才能认定某个覆盖子集为 L2。任何未覆盖操作仍为 L1/unknown；上游 drift、版本变化、证据过期或比较器变化触发失效与重验。

### 20.7 优先领域

先选 issue tracker 中 get/list/create/close 的有限子集，或 package registry 中明确的 publish/yank/install 合约。选择标准是 reference 可固定、可复位、无生产副作用、正反例可观察，不是接口最热门或工具数最多。

---

## 21. 存储、迁移、保留与故障恢复

### 21.1 保留当前存储边界

本地单机/受控进程使用 SQLite，保持 world-store 与 Episode Journal 的独立 identity/schema。不要把网络共享目录上的 SQLite 当作分布式状态服务，不将未来 HA 工作隐含并入 1.0。

现有 schema migration/reopen/interrupted-migration 能力继续使用；新增 trace、task 与 Evidence 字段优先通过独立版本化工件引入，减少把所有产品变化变成数据库大迁移。[R02]

### 21.2 生命周期与保留

定义引用关系：active episode → branch/snapshot；evidence → 必要 replay 工件；release corpus → 不可变测试版本。GC 只回收没有 active/reference 的对象，不能删掉仍被声明为可重放证据所需的初态或 trace。

建议本地成功运行临时状态在确认封存后回收；失败/未完成状态按显式用户策略保留；公开合成证据可以长期保存。涉及非公开内容时不设无限默认保留。保留期限是产品策略，需在 run 开始前知晓。

### 21.3 导出、备份与恢复

导出前固定一致性点，包含 manifest、schema/版本、逻辑内容摘要与所需依赖；恢复到新位置先验证再激活，不覆盖唯一原件。迁移前可恢复备份、失败时状态识别、旧格式读/拒绝矩阵均是门槛。

不承诺任意版本 downgrade；旧 runtime 打开未来 schema 应无修改拒绝。允许通过明确工具迁移副本，而不是手工降低 version 字段。

### 21.4 故障验证范围

补充磁盘不足、写权限丢失、进程在 commit/封存边界终止、report 写失败、Journal/world 关联缺失、corrupt archive、partial trace、WAL/临时文件残留。测试应证明“无静默成功、无错误覆盖、可解释的 incomplete/unknown”。

### 21.5 终态证据与持久化顺序

durable Journal 记录 immutable run request；世界事务写入自己的 audit；最终 Evidence 在有版本 CAS/fencing 的 Journal 中完成接受。文件系统工件写入使用临时路径、校验、原子提交或等价可恢复协议；不得仅凭同名文件存在就判成功。

重复提交完全相同 Evidence 返回已存在结果；相同 episode/attempt key 对应不同内容必须冲突。进程在封存中途重启能够区分未完成、已封存和已接受，不可自动重跑一次有潜在外部副作用的调用。

---

## 22. 非功能要求与运维预算

### 22.1 性能不先报未经测量的数字

当前代码中的 1 MiB、CEL cost、32 actions/256 events、512 MiB Go heap 软目标等是声明的 admission/运行配置，不是所有机器上的性能保证。[R02]

建立固定硬件/OS/Go/依赖/profile 下的基准：工具调用 p50/p95、snapshot/fork/diff、状态规模、scheduler 规模、trace 写入开销、RSS/heap 峰值、磁盘增长、启动与 teardown。输出原始样本、数据量与测量方法。

初期建议性能回归阈值采用相对基线，只有多次重复且超过噪声范围的显著恶化才阻断；不得凭一次共享 CI 慢 20% 判断算法退化。正确性、安全和恢复优先于吞吐量。

### 22.2 边界清单

每个新增可增长对象都要有上限：Task/Schema 大小与深度、目标文本、工具数量、轮次、并发、事件数量、trace 总字节、单结果、报告、Artifact 提取、branch/snapshot、保留时间。超限要么在执行前拒绝，要么停止并产生类型化不完整档案，不能静默截断评分所需事件。

### 22.3 可观测性

保留 operation log 与 canonical evidence 的分离。日志展示组件、episode 的非秘密引用、阶段、分类和计数，不默认打印输入输出。health/readiness 只提供必要状态，不暴露数据库路径或世界数据。

OTel 作为可选 exporter，不成为真相源；导出失败不能改变业务评分，只影响 observability 状态。metrics 标签不使用用户文本、token 或高基数原始 tool input。

### 22.4 Provider 故障与依赖故障

401/403、模型不可用、quota、429、网络超时、服务错误、Schema 不支持与内容中断必须分开分类。失败记录要保留可分享诊断，不回显密钥。只对明确安全、未产生副作用且已定义的请求重试；不能让通用 HTTP retry 中间件自行决定所有情况。

依赖升级用独立候选 PR、golden corpus、存储 reopen/迁移与回滚说明。当前 SQLite PR #5 在本次查看时仍未合并，不把其候选状态当作主分支依赖已经升级。[R08][R15]

---

## 23. 测试与质量保障体系

### 23.1 六条互不替代的验证轨道

| 轨道 | 核心验证 | 不能代替 |
|---|---|---|
| Unit / property / fuzz | 解析、转换、边界、确定性性质 | 真实 provider 合约或上游保真 |
| Scenario contract | 固定脚本正反例、状态与错误 | 自主 Agent 决策能力 |
| Protocol / adapter contract | MCP wire、provider mock、Schema 投影 | live host compatibility |
| AgentEpisode integration | 隔离、调用循环、评分、证据、cleanup | 真实服务 fidelity |
| Live exact-profile evaluation | 实际宿主/模型轨迹与结果 | 其他宿主/模型/时间点的兼容 |
| Independent differential | reference 限定覆盖区域 | 完整服务等价或未来版本 |

### 23.2 基础 CI 保留

现有 CI 包含 Linux race、Windows/macOS test/build、fuzz、secret policy、hermetic egress、协议和 Bundle/Episode smoke，应继续保留。跨平台分别测试通过不等于已证明“同一 corpus 的语义工件跨平台逐项一致”，需要新增独立 artifact 比较 job。[R09]

### 23.3 新增 CI 轨道

普通 PR 无密钥地运行 mock Agent loop、Task/Oracle 正反例、世界 replay、adapter 映射、证据篡改/脱敏、cleanup、身份与文档生成校验。真实付费 provider lane 显式选择、批准预算，并在隔离凭据上下文执行，不给不可信 PR 任意使用 secret 的能力。

release candidate 固定 SHA、Go patch version、依赖、协议和 corpus；普通开发可以使用动态 runner 镜像，但 evidence 必须记录实际 OS/toolchain，不能只写 ubuntu-latest。

### 23.4 推荐必须覆盖的负向案例

重复工具 ID 搭配不同输入；provider 返回未知 tool；同一 turn 多调用；输入解析失败；工具已提交但结果交付失败；运行停止后晚到 call；Task 引用不存在 tool；模型输出超限；隐藏 grader 泄漏检查；错误 branch 授权；外部 Schema 引用；Evidence 中间事件缺失；redaction 破坏引用；磁盘不足；cleanup 失败；已封存 run 重复执行。

### 23.5 交付 Definition of Done

需求语义被接受；有代码与正反例；支持 profile 下可重放；错误有分类且不泄露；必要迁移与回滚明确；CI exact revision 通过；证据工件存在；状态/README 派生视图更新；至少一个用户可理解的示例。只满足“编译通过”或“AI 看过代码”不算完成。

每个严重缺陷优先建立最小失败工件，再修复，再把该工件进入稳定回归集。一个 bug 不默认需要一份新架构 SPEC；只有改变公共语义时才升级为 ADR/规范变更。

---

## 24. 整个项目周期：交付里程碑与发行映射

本节使用 **D0–D8** 表示本稿的交付里程碑，避免与仓库已有 Phase 0–7 混淆。版本映射是需要 ADR 接受的提案，绝不是对既有 release train 的无声覆盖。

### 24.1 总路线

```text
D0 基线与决策封存
    |----> D1 狭义本地核心发行（独立通道，不阻塞 D2 开发）
    v
D2 一个领域的真实 Agent 纵向闭环（local-outbound）
    |
    v
D3A 可重复的模型配置回归产品 -----> D3B 原生远程/产品宿主证据（独立安全门槛）
    |                                  |
    +------------------+---------------+
                       v
D4 有限上游保真度与差分证据
                       |
                       v
D5 按实际需求扩展任务族、时间/故障与有限多 Agent
                       |
                       v
D6 1.0 合约稳定与真实消费方验收
                       |
                       v
D7 维护、监控、迁移与弃用
                       |
                       v
D8 版本/能力/项目退役与可恢复归档
```

D3B 不要求先实现完整 SaaS；D5 的所有候选也不是 D6 的必要前置。1.0 只冻结经过使用的具体 profile，不要求承诺全部宿主、多 Agent 或 L3。

### 24.2 D0：重新建立唯一事实基线

**进入：** 指定提交、已有 RFC/ADR、Implementation Status 与 CI 可定位。\
**交付：** 当前能力清单、旧新规格映射、产品/接入分线/TracePolicy 决策、首批工作项、需要修正的摘要。\
**退出：** 每一条对外当前声明有证据或明确 unverified；不再把已实现能力写为待从零建设；下一步 owner、依赖、DoD 清楚。\
**停止条件：** 关键权威文档冲突未解决，或尚未决定 local bridge 与 native remote 的范围。\
**回滚：** 只撤回未接受的新规划，不修改已验证 runtime 语义。

### 24.3 D1：发布范围有限但可以依赖的本地核心

**建议版本：** `v0.1.x`，沿用原 Phase 1 的 local hermetic contract。\
**范围：** accepted 本地核心与两领域脚本例；Bundle/Episode 及其他已存在的 preview 扩展是否进入稳定支持清单，须按 ADR-0018、ADR-0021、RFC-0002 和 Phase 1 的逐项映射明确，不能因同包交付就默认稳定。保留清晰安装与迁移说明。\
**退出：** exact candidate CI；核心 golden corpus；控制/数据隔离负例；支持平台运行；已有存储身份/迁移门槛；first-value 指南；没有未说明的关键数据损坏问题。\
**明确不阻塞：** 没有 live provider 不阻塞局部本地核心发行；没有 HA、自动重试、registry 不阻塞。\
**拒绝发布：** 用“先发再说”掩盖已知世界不确定、跨分支污染、不可恢复写损坏或对外宣传越界。

### 24.4 D2：真实 Agent 最小闭环

**建议版本：** `v0.2.x` 的 experimental local-evaluation 子范围，需新增 release-scope ADR；保留既有 Phase 2/3 的 bounded 实现。\
**范围：** AgentTask、只读 evaluator、一个 API bridge、fresh run identity、基础 trace/redaction、封存与 cleanup、一个领域 6 个任务。\
**退出：** mock 正反例全部通过；至少一个实际可用模型/profile 产生有界 live evidence；6 个任务的评分行为已人工复核；一条提交后响应不确定案例能解释世界与观察差异；没有生产调用。\
**模型得分：** 不要求 Astra 必须高分，也不要求任何模型全过；要求产品不会错算、漏算或掩盖失败。\
**外部阻塞：** 没有授权 API/预算时，只能完成 contract-tested，live gate 保持 blocked；不伪造报告。

### 24.5 D3A：模型升级回归可被外部使用

**建议版本：** `v0.3.x` 的 local-evaluation profile。\
**范围：** 封存 EvaluationPlan、至少两个配置的配对/重复运行、有效性分母、风险与效率报告、CI compare gate、错误归因、离线 replay、self-serve 文档。\
**退出：** 每个差异可关联到 Task/模型/宿主/世界身份；inconclusive 不伪装 pass；外部使用者无需修改 runtime 即完成一次自有任务或自己的配置对比；取得可复核反馈。\
**首批目标：** 完成 3 个独立外部试用反馈；24 个候选任务中只执行已经 admitted 的部分，blocked 清单对用户可见。\
**停止扩张：** 外部使用者普遍无法接入或报告不能支持升级决策时，暂停新增域和调度功能。

### 24.6 D3B：原生远程与产品宿主独立证据

**建议版本：** 原 Phase 4 / `v0.3.x` native-remote 或独立 experimental product profile。\
**进入：** 对应 remote-staging、安全与已接受前置门槛通过；真实 API 授权和预算批准。\
**范围：** 原生 OpenAI/Anthropic MCP profile，以及有需求才增加的 Codex/Claude Code 等产品 profile；鉴权隔离、审批、取消/失败、销毁证据。\
**退出：** 每个宣称兼容的 profile 各自有当前 live 报告、至少一条成功与负向路径、有效身份和边界；无秘密残留。\
**非传递性：** 通过 local bridge 不关闭本 gate；通过某个 API 不关闭某产品宿主 gate。[R13]

### 24.7 D4：有限可证明的 Fidelity

**建议版本：** `v0.4.x`，与原 Phase 5 方向一致。\
**范围：** 合成 reference recorder、redaction、cassette、surface inspector、comparator、有限差分覆盖和人工 L2 admission。\
**退出：** 至少一个界限清楚的 reference coverage 子集；展示一个真实 mismatch 与其解释/修复过程；所有未覆盖/accepted divergence 公开；漂移可使旧声明失效。\
**不合格：** Twin 与自身共享规则跑两次得到一样结果，不算上游 L2；录制数多也不等于状态模型保真。

### 24.8 D5：通过需求推动复杂度

**建议版本：** post-v0.4；具体版本接受时再定。\
**候选：** 种子稳定任务族、metamorphic 变体、有限虚拟时间任务、领域幂等与可见性延迟、共享世界多 Agent。\
**进入：** 已有用户失败案例不能被现有两个领域/基本 profile 表达；有固定可解性与评分方案。\
**退出：** 只对被接受的子范围补齐确定性、隔离、成本和证据测试。\
**禁止：** 将 recurrence、DLQ、自动外部重试、A2A、GUI 等一次性纳入“平台完善”。

### 24.9 D6：1.0 合约稳定

**建议版本：** `v1.0.0`。\
**范围：** 已经经历 preview 和实际消费的核心格式、Task/AgentEpisode/Evidence 的稳定子集、CLI/机器输出、兼容/迁移政策与维护承诺。\
**退出：** 所有声明为 stable 的 P0/P1 需求有证据；支持平台语义 corpus 一致；至少一个实际 Agent profile；真正的外部消费方复验；升级/恢复/弃用演练；已知限制、数据政策和安全响应可执行。\
**Fidelity：** 框架稳定与 reference fidelity 分开。若发行宣称提供 verified Twin，则相应有限 L2 gate 必须满足；否则清楚标为稳定框架 + L1 参考模型，不暗示生产等价。\
**不要求：** 全宿主、全 MCP 扩展、L3、多租户或所有 D5 候选全部完成。

### 24.10 D7：持续维护

对模型别名/宿主/协议依赖变化做 profile 重验；对数据与存储变化做迁移；对实际失败案例补回归；对证据 TTL 过期降级声明。每次维护应减少已知风险或改善使用，而不是例行创造新 SPEC。

### 24.11 D8：退役与归档

对不再维护的 profile/版本发布停止支持范围、替代路径和证据截止时间；保留最后兼容 reader/导出工具或可执行环境说明。撤销远程 endpoint/密钥，清理不应继续保留的数据，归档合成 corpus 与证据目录索引。

项目整体进入维护模式或归档时，README 明确不再响应的服务边界，关闭自动兼容徽章，避免陈旧“verified”被误解为当前支持。用户拥有的世界/证据不能因项目停止维护而被平台锁住。

---

## 25. 最近一轮执行顺序：小切片形成闭环

完整工作包见 `IMPLEMENTATION-BACKLOG.md`。以下是建议的首轮顺序，不代表已经创建 GitHub Issue。

| 顺序 | 工作包 | 先交付的具体结果 | 不要同时做 |
|---|---|---|---|
| 1 | B01 | 固定基线、修正文档摘要、接受分线与范围决策 | 改写所有已接受规范 |
| 2 | B02 | 最小 AgentTask schema 与六个任务的可解性卡片 | 通用任务生成平台 |
| 3 | B03 | 只读 evaluator，正例和错对象/额外副作用负例 | LLM judge 主导评分 |
| 4 | B04 | RunDefinition、fresh run、预算/隔离 preflight | 全宿主配置 registry |
| 5 | B05 | 一个本地 bridge 的 schema/调用关联 mock 闭环 | 原生公网 MCP 部署 |
| 6 | B06 | 单个任务的 AgentEpisode 纵向运行与基本证据 | 完整 cohort 服务 |
| 7 | B07 | trace redaction、seal、verify/replay、失败清理 | 云端日志与仪表盘 |
| 8 | B08 | 显式预算下的 live profile 和六任务检查 | 自动扩大到昂贵矩阵 |
| 9 | B09 | 两个配置的重复运行 compare 与风险门槛 | 发布模型排行榜 |
| 10 | B10 | first-value CLI/文档、报告可读性与外部试用 | 大规模扩充领域 |

B02/B03 的设计可以同一个 PR；B06 尽早形成端到端案例。B11 是独立本地发行通道；B12 的预算语义和 B07 的保存前数据政策必须在 B06 的执行入口中生效，B08 live 更不能绕过它们。B09/B10 的离线部分不以 B08 为开发前置；live 与外部验收仍是独立交付门槛。详见 PHASE-SPECS。

---

## 26. 团队、职责、工作量与预算管理

### 26.1 角色而不是假定人数

| 角色 | 必须负责 | 单人项目如何执行 |
|---|---|---|
| 产品/需求负责人 | 用户价值、范围、优先级、验收决定 | 主维护者兼任；记录取舍，不靠临时聊天遗忘 |
| Runtime owner | 确定性、存储、事务、兼容性 | 对公共语义变化做独立自查清单 |
| Evaluation owner | Task 可解性、oracle、统计、归因 | 尽量请外部试用者复核高风险评分 |
| Adapter owner | provider/host 合约、工具投影、成本边界 | 首先维护一种 adapter，不同时承诺十种 |
| Security/release reviewer | 隔离、秘密、迁移与公开声明 | 缺少独立评审就降低声明，不伪装已经审计 |
| Design partner | 真实接入与使用反馈 | 不要求维护者泄露生产资料，使用合成最小案例 |

AI 可以起草实现、测试与文档；任务 witness、oracle 与关键安全负例的实现和验证须可审查。人工负责高风险语义接受、Fidelity 等价及公开兼容声明晋级；不将 AI 静态自查称为独立审计。

### 26.2 WIP 与优先级

建议同一时间只推进一个端到端产品工作包，外加一个必要维护/缺陷通道。P0 指对应里程碑的不可缺少项，不代表全生命周期所有 P0 都要同时做。安全/数据完整性缺陷优先插队；功能扩展不以“已经写了 spec”为开工理由。

完成两个或更多实际工作包后，再依据真实 cycle time、返工和审查数据更新日期计划。本稿不给未知人力配置承诺总工期，也不用虚构人天营造确定性。

### 26.3 试验成本

先跑离线与 mock，再跑单任务 live，再跑小 cohort。每个 plan 分开记录 provider 调用预算、运行资源预算、保留/存储预算、人工审核成本。单价有版本和获取日期，usage 不可得时记录 unknown。

获批预算只授权该 plan；失败重跑、增加配置或提高试验次数需要重新检查剩余预算。禁止为追求稳定分数无限重试，或在 CI 默认打开付费矩阵。

### 26.4 范围变更

新增需求至少说明：哪个用户任务受阻、现有方案为何不足、最小增量、影响的身份/格式/安全边界、验收证据、必须延后的现有工作。没有上述信息时放入候选池，而不是自动给版本承诺。

---

## 27. 用户验证与产品是否值得继续投入

### 27.1 试用脚本

让外部使用者在不接受维护者手把手修改代码的情况下完成：运行离线例子；理解一份失败报告；设置自己的模型配置；改动一个合成任务；完成一次比较；指出结果能否帮助实际决策。记录在哪一步失败、需要多少手工补丁和何种解释。

不收集真实 token、私有 prompt 或生产数据库。反馈以步骤、去标识错误、操作次数和用户评价为主。

### 27.2 三类学习结论

若用户喜欢 deterministic runtime、但需要自己编写大量 grader，优先改善 Task/Oracle 作者体验；若 adapter 接入最痛，优先本地桥接与兼容 lint；若用户无法信任与真实业务的一致性，优先有限 fidelity，而不是增加更多演示。

若只有维护者能使用，或用户一次性看 demo 后没有重复使用，则产品假设尚未被证实。不能用“模型变强了所以更需要我们”替代验证。

### 27.3 Go / Narrow / Pause 决策

Go：有重复使用、能支撑升级/修复决策、失败可归因、维护成本可承受。Narrow：有价值但仅在一个域或一种宿主成立，则明确支持该范围。Pause expansion：核心流程仍难用、评分经常无效、尚无需求证明下一层复杂度；保留稳定核心，暂停新平台功能。

停止扩张不是宣布项目失败，而是防止维护面远超已经验证的价值。

---

## 28. 风险、缓解与待决策项

| 风险 | 触发信号 | 对策 | 决策责任 |
|---|---|---|---|
| 规范膨胀 | 同一能力在多个文件出现冲突状态 | 单一清单派生、历史归档、只为语义变化写 ADR | 产品/维护者 |
| 评分器不可信 | 明显失败却 pass，或合法路径 fail | witness、变异测试、外部复核、冻结基线 | Evaluation owner |
| 模型/宿主漂移 | 相同 ID 行为或合约改变 | profile 时间与可解析身份、重验、stale 声明 | Adapter owner |
| 模型看到答案 | 工作区/工具结果暴露 oracle | 隔离负例、可见性分层、coaching 标记 | Security reviewer |
| 不可解任务 | 必需工具或信息缺失 | admission blocked，不算模型失败 | Task owner |
| 外部效果不确定 | timeout 后无法确认远端是否接受 | COMMIT_UNKNOWN、停止盲重试、人工核对 | Runtime/Adapter |
| 隔离被夸大 | 只做 tools/list 隐藏却允许 shell 读 DB | OS/文件/网络 profile、不能满足就降级 | Security reviewer |
| 维护者过载 | 多 adapter 与领域同时过期 | 支持范围收缩、WIP 限制、过期显式化 | 项目负责人 |
| 成本失控 | CI 重跑/长循环/未知 usage | 计划级预算、预留、有限重复、不开默认 live | 评测负责人 |
| 差分伪证据 | reference 复用同一转换或忽略所有字段 | 独立 reference、比较器审查、覆盖分母 | Fidelity owner |
| 数据保留失控 | trace 含秘密或无法删除 | before-persist redaction、分级保留、撤回工件 | 数据负责人 |
| 依赖升级改变语义 | golden digest/迁移结果变化 | 候选 lane、pin、解释差异和迁移 | Runtime owner |

### 28.1 实施前必须作出的决定

接受 local-outbound 的安全边界；AgentTask 最小 schema；新增 trace 内容政策；首个 adapter/API；首批预算审批方式；stable 发行支持范围。这些阻塞相关工作包，不能默认由代码生成器猜测。

### 28.2 可以后置的决定

长期 GUI、完整多 Agent、云托管、租户收费、插件运行时、registry、全量 SDK/扩展支持、L3 参考实现。给它们记录启动条件，不要为了“整体 spec”提前做不可维护的细节承诺。

---

## 29. 发行、兼容与迁移合约

### 29.1 Release candidate 清单

精确源码 SHA、无未提交改动的构建来源、Go patch version、模块 checksum、MCP/SDK/conformance profile、支持平台、limits/profile digest、schema 版本、测试 corpus、证据工件摘要、已知缺陷与非目标。发行包与文档指向同一 revision。

当前 CI 的动态版本选择适合日常开发，稳定发行证据还应记录解析后的精确版本，不能只写 `1.26.x`。必要供应链产物包括依赖/许可证清单、checksum 和构建来源；签名/attestation 按实际支持范围提供，不能称为业务正确性证书。[R09]

### 29.2 兼容政策

稳定 reader/writer 的支持范围分别声明。建议至少支持上一个稳定格式的读取或提供独立转换工具，拒绝未来格式时保证零修改。API、CLI、schema、semantic profile 的 breaking change 要有迁移说明和版本标记。

preview 可以演化更快，但不能悄悄重释历史 Evidence。比较工具遇到 evaluator/Task 版本不同时默认报告不可比；使用受审查转换后才能继续比较，并记录该转换。

### 29.3 弃用窗口（建议值）

稳定接口建议至少提前两个次版本且不少于 90 天公告，再执行移除；安全紧急事件可缩短，但必须给风险理由和替代路径。该窗口不是当前项目已有承诺，需在 1.0 前正式接受。

### 29.4 兼容证据保鲜

精确宿主/模型版本证据长期作为历史事实保留，但当前支持声明会过期。建议滚动模型别名的当前兼容提示设置较短复验周期，例如 30 天；固定版本根据升级/漏洞/服务变化事件触发。证据过期不删除历史，而是把 current verified 降为 stale。

### 29.5 撤回与回滚

发现发行含关键状态错误、秘密泄露或错误兼容声明时，先撤回相关 claim/工件，再提供受影响范围、最后安全版本、数据恢复/再验证路径。不能通过修改已有 digest 对应的内容修补历史证据；发布更正版并保留 supersedes 关系。

---

## 30. 维护期运行手册

### 30.1 常规维护

周期性核验声明过期、依赖更新候选、测试 flaky 情况、corpus 增长、工作项积压与外部使用反馈。任何新功能优先回答“改善哪个已经观察到的问题”。

### 30.2 事故处理

发现不正确评分：冻结受影响 Task/evaluator 版本，标记相关比较结论需复验，给出最小反例并修复；发现世界确定性问题：停止相关 profile 的 strict replay 声明，保存平台/版本差异；发现秘密泄露：限制工件访问、撤回公开副本、轮换受影响凭据并说明残余传播风险。

上述是项目运行要求，不表示本次发现这些事故已经发生。

### 30.3 模型或 API 变化

文档更新只产生 documented 状态；adapter mock 通过产生 contract-tested；真实指定 profile 跑过才恢复 live claim。模型新名称、provider 新 beta 或 SDK 新版本不允许通过简单改 badge 自动晋级。

### 30.4 Issue 生命周期

triaged → scoped → accepted → implementing → review → evidence-ready → released → monitored。每个阶段有 owner；blocked 写清外部依赖和解除条件；close 必须指向代码/决策/证据，不能只回复“应该已修复”。

维护单人项目时，公开需求和工作包优先于复杂会议机制。一个简洁、可追踪的看板胜过大量没有 owner 的 roadmap 表。

---

## 31. 退役、数据退出与长期可读性

### 31.1 Profile 退役

关闭被退役 adapter 的自动执行入口，保留其历史证据和“最后验证日期”；新任务不能默认落到退役 profile。用户显式历史 replay 使用固定工具链/环境，且无隐式网络调用。

### 31.2 数据退出

用户能导出 TwinSpec、fixture、Task、EvaluationPlan、允许保留的 trace、Evidence 和版本 manifest；导出路径验证完整性，不绑定在线服务才能读取。不能导出的敏感数据说明原因与删除方法。

### 31.3 项目归档

公告支持结束，明确不再保证 provider/API 当前可用；撤销服务密钥、停止远程 worker/endpoint；关闭误导性实时 badge；保留最终支持矩阵、数据格式与离线验证办法。任何运行的自动计费任务必须显式停止，不能仅归档代码仓库。

---

## 32. 总验收：项目何时算真正完成一个阶段

一个阶段完成不是因为 spec 写全、代码行数增长或 mock 没报错，而是因为以下链条同时成立：

```text
真实用户问题
  -> 接受的需求与有限范围
  -> 已实现的业务/运行语义
  -> 正反例与边界测试
  -> 指定提交与 profile 的证据
  -> 准确的发行声明
  -> 外部使用者能完成任务
  -> 已知失败有恢复、迁移或退出路径
```

近期最重要的产品验收是：**同一合成世界、同一任务与授权下，两个真实模型配置可以走不同合法路径；系统仍能正确判断目标、越界尝试、已提交副作用和不确定恢复，并提供能复核的证据。**

最终 1.0 验收是：这些已经被使用的合约稳定、迁移可控、声明准确、维护可持续。不是“模拟一切世界”，也不是“追上所有新模型”。

**推荐立即立项的目标：B01–B10。先把一个真实 Agent 闭环交给外部使用者，再让证据决定下一层架构。**
