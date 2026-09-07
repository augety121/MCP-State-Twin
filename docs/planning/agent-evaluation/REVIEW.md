# Agent 回归产品线：规格评审与采用边界

- 日期：2026-09-08；状态：Proposal / design-reviewed，尚未 accepted。
- 代码审阅起点：`main @ 77d3ea0ec610fe64b48425f15c350e0b4c0d577f`。
- 本轮仅文档审查、设计和整合；没有实现 AgentTask，没有执行 live，没有发布版本。
- 设计权威仍为仓库指令、accepted ADR/RFC/SPEC；本目录不覆盖其语义。

## 1. 结论与阅读入口

保留现有确定性世界、Scenario、Bundle、Journal 和 bounded coordinator。
下一条产品主线建议是“给真实 Agent 目标，观察实际操作，用独立评分和证据支持升级决策”。
这不是已经实现的模型评测产品，更不是通用 AGI 或所有宿主兼容的声明。

| 文件 | 职责 |
|---|---|
| [总规格](MASTER-SPEC.zh-CN.md) | 保留原稿 32 章全生命周期设计，纳入 v3.1 修正 |
| [Backlog](IMPLEMENTATION-BACKLOG.md) | 保留 B01–B32 工作包及原稿需求引用 |
| [近期分阶段合约](PHASE-SPECS.md) | 开发顺序、接口、停止/评分/证据语义、阶段退出条件 |
| [首批任务卡](TASK-CATALOG.md) | 六个候选任务的可解性、witness、正反例及未实现项 |
| [补充验收清单](ACCEPTANCE.md) | AE 编号的补充需求与计划验证，不冒充原稿 109 项正文 |
| [来源登记](SOURCE-REGISTER.md) | 本轮实际代码依据、外部依据和缺失材料 |
| [待接受决策](DECISIONS.md) | 新路线接受清单，不占用正式 ADR 编号 |

本目录内同一事项的详细合约以 PHASE-SPECS 和本评审修正为准；Backlog 是任务视图，
总规格是全生命周期视图。若未来发现相互冲突，先修正文档再接受对应条款，不能选择较宽松版本实现。

## 2. 输入完整性

完整读取上传的 Master（32 章）与 Backlog（32 包），以及附带说明文字。
Backlog 含 109 个不同 `MST26-*` 引用，但以下原包成员未随本轮上传：

- `REQUIREMENTS-AND-ACCEPTANCE.md`、`requirements.yaml`：缺少逐项规范正文；
- `SCENARIO-CATALOG.md`：缺少 24 项原任务定义；
- `SOURCE-AND-AUDIT-REGISTER.md`：无法还原原稿 R/W 引用的全部指向；
- `templates/`：无法确认原拟议 schema、字段和示例。

本轮独立补写六张任务卡和 AE 验收条款，不声称恢复了缺失原文。全包逐条覆盖审计保持 open；
可执行下一轮有限实施不必等待全部远期材料，但必须先接受其实际使用的合约。
原稿建议的三位外部试用者、24 个任务、两次改进案例都不是现有用户或采用量。

## 3. 主要发现与修正

| 编号 | 发现 | v3.1 处理 | 是否改变当前实现 |
|---|---|---|---|
| RV-01 | Scenario Episode 跑固定脚本，smoke 只统计调用 | AgentTask/AgentEpisode 单独 kind，禁止重释旧成功字段 | 否 |
| RV-02 | 原稿 B09 依赖 live，阻塞本可离线做的比较器 | B09/B10 分 offline、live、external 验收；完整 D3A 不提前完成 | 否 |
| RV-03 | 总图把 D1 当 D2 前置，文字却允许并行 | D1 独立发行通道；D2 依赖核心合约，不依赖先打稳定 tag | 否 |
| RV-04 | B12 预算在列表后面，B06 未列依赖 | B06 前有预算策略和数据政策；B08 前所有安全控制生效 | 否 |
| RV-05 | 本地稳定发行分线被当成新建议 | ADR-0021 已明确 provider live 非 v0.1 gate；继承，不再投票 | 否 |
| RV-06 | Bundle/Episode 等 preview 与 Phase 1 支持表有交叠 | B11 逐项映射支持清单；同包分发不等于全部稳定 | 否 |
| RV-07 | 读取任务只看终态会漏掉根本没读/没答 | 输出事实和已交付观察分别评分；无自由文本确定性契约则候选 blocked | 否 |
| RV-08 | “其他状态不变”可能误含调用计数/audit/head | 分开业务实体、副作用、运行元数据；不要求整个 state 字节相等 | 否 |
| RV-09 | “没有额外终态变化”漏掉先写错再恢复 | 终态 diff 与提交轨迹共同检查，拒绝尝试另计 | 否 |
| RV-10 | 封存前没有规定未知在途提交处理 | close admission → bounded drain → consistency cut；不确定则 partial，禁止 pass | 否 |
| RV-11 | 工具数组中的重复 ID、预算与顺序未固定 | 整批协议结构检查后按序执行；预算逐调用预留；不做隐式业务幂等 | 否 |
| RV-12 | 读取 hidden oracle 的权限风险 | 远程模型只有窄工具投影；本地 trusted harness 不能号称 OS 沙箱 | 否 |
| RV-13 | trace 脱敏与 replay 等价有张力 | 合成白名单内容才可封存；敏感/不可等价内容拒存并降级，不靠 hash 恢复 | 否 |
| RV-14 | live 成功可能被误作所有宿主/模型兼容 | local bridge、native MCP、产品宿主独立 profile；原 claim 枚举不变 | 否 |
| RV-15 | B32 依赖稳定版才做维护 | 基础维护立即适用；稳定弃用和退役条款按阶段激活 | 否 |

另有两处现有派生摘要失配：ROADMAP 对已合并 coordinator 的“待合并”表述，以及
VNEXT-TRACEABILITY 末尾笼统否认整个 fault/scheduler 类别。本轮只修正其文字范围，
不新增测试通过或能力晋级声明。

## 4. 旧规范去向：不重写内核

| 既有依据 | 本轮处理 |
|---|---|
| RFC-0001 I-1–I-16 | 全部保留；只在外围新增评测链路 |
| ADR-0002，SPEC-0003，SPEC-0020 | 控制面隔离继续适用；local bridge 不能给远程数据面安全盖章 |
| ADR-0015/0018/0019/0020/0021，RFC-0002/0003 | 既有 release/Bundle/Journal/remote 语义保持；AgentEpisode 新 kind 待接受 |
| SPEC-0019/0021 | 保留现有 public claim vocabulary；新增 test_level 不替换旧状态枚举 |
| ADR-0022–0034 | 保留 quiet 资源政策、时间/熵/调度 bounded subset；不从零重做 |
| ADR-0035–0037 | 保留 fuzz、依赖迁移及 CEL null 修复；本次不复跑其测试 |
| 旧 `00-*` 至 `33-*` 研究包 | 历史 proposal；D0–D8 是新规划坐标，不重编号旧 Phase 0–7 |
| Phase 4 native provider gates | 不以 local-outbound 报告关闭；本地路径是待接受的新有限路线 |

## 5. 完成与暂停条件

本轮 Spec 交付的完成条件：两份输入完整审阅、已发现冲突有明确处理、近期各包有接口和
负例验收、缺失材料与授权分界显式标记、Markdown 链接有效、业务代码/依赖/CI 配置不变。
文档校验不是 runtime、live、安全隔离或发布验收。

接下来实施需要维护者明确要求实施本轮修订稿；此时先接受 DECISIONS 中限定范围，再做
B02/B03/B04 与 mock bridge。无 API 凭据时继续所有离线工作；不得伪造 live 或外部反馈。
付费模型、公开 endpoint、外部消息、发布或大矩阵必须有相应明确授权和有限预算。
本轮不 push、不创建 Issue/PR、不发布 release。

## 6. 本轮实际检查记录

- 完整阅读输入，核对本地基线；本轮工作区原先干净。
- 结构清点：32 章、32 个 B 工作包、109 个原稿引用 ID、6 张新候选任务卡、24 条 AE 补充要求。
- 单 Go slot、`-p 1`、禁用模块下载下，以下现有文档检查通过：
  `TestLocalMarkdownLinksResolve`、`TestUnifiedLifecycleDocumentsExist`、
  `TestHardInvariantTraceabilityIsComplete`（`internal/doccheck`）。
- `git diff --check` 通过；检查范围是现有跟踪文件的 diff，新文档另作结构与内容检查。
- 仅修改 docs：8 份新规划文档、4 份既有文档的入口/摘要；业务源码、依赖与 workflow 未改。
- 未执行 runtime 全套测试、race/fuzz、provider live、上游差分或 GitHub 远程验收。

以上检查证明文档入口与既有追踪规则没有被破坏，不证明新提案可直接运行或已达到发布标准。
