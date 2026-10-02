# SPEC-0078 迭代计划与验收矩阵

**状态：已授权实施；逐项状态见交付台账。日期：2026-10-02。**

本文件是 [主 Spec](../../SPEC-0078-AGENT-BASELINE-PLUGIN.md) 的执行视图，不是另一套权威契约。[ADR-0078](../../ADR-0078-AGENT-BASELINE-PLUGIN-PROPOSAL.md) 已接受实施。表中的测试名称为计划入口，实际文件/测试结果由交付台账记录，不能当作已执行的测试。

## 1. 依赖与整批交付边界

```text
N0 当前基线/契约接受
 └─ N1 描述与准入 → authority dispatcher → stdio/session → evidence → Inspect → 24 Task 验收
      └─ N2 family/variant + quality + disclosure
           └─ N3 冻结计划 + shards + 统计判定
                └─ N4 live/具名宿主/外部 first-value
                     └─ N5 所声明 profile 的 preview 发布资格

有限 L2：独立 reference 可获得后才启动；不阻止不宣称 L2 的插件 preview。
```

当前优先级是 **P0=N0/N1，P1=N2/N3，外部条件满足后=N4，发布门槛=N5**。不是六轮同时编码，也不是只写包装后结束。一个工作包只有文档、API stub 或 happy path 时不得关闭。

如果以后授权“整批实施本 Spec”，默认完成所有可执行工作，外部条件逐项记 blocked；不得用 N1 完成冒称 N0–N5 全部完成。如果只授权 N1，按下面 W00–W09 和相应验收完整交付，不自动做 live 或发布。

## 2. 工作包与建议 PR 切分

建议路径可以在接受时根据真实代码组织调整，但必须保留需求映射。原有 `internal/agenteval` 已较大，新特性不应全部堆进同一个源文件。

| 包 | 迭代/优先级 | 交付物及拟议入口 | 依赖 | 责任角色 | 验收 ID |
|---|---|---|---|---|---|
| W00 | N0/P0 | 当前 main/旧分支关系、协议与 adapter 版本决定、接受记录；docs | 无 | Maintainer | AB01–AB02 |
| W01 | N1/P0 | 严格 pack/profile/session schema 与 fixtures；`internal/baselinepack` | W00 | Runtime owner | AB03–AB05 |
| W02 | N1/P0 | Task 约束 dispatcher；复用 `agenteval/environment.go`、server 工具注册 | W01 | Runtime + Security | AB06–AB08 |
| W03 | N1/P0 | stdio MCP 传输、冻结工具面、限流/背压；`internal/plugin` | W02 | Runtime owner | AB09–AB11 |
| W04 | N1/P0 | 私有 lifecycle、单 session world、准入/finish/cancel；`internal/plugin` | W03 | Runtime owner | AB12–AB16 |
| W05 | N1/P0 | terminal/partial、重放、只读 inspect、有限错误；复用 evidence | W04 | Evaluation owner | AB17–AB20 |
| W06 | N1/P0 | `plugin check/describe/serve/inspect`、本地配置模板；`cmd/statetwin` | W01;W05 | Runtime owner | AB21–AB22 |
| W07 | N1/P0 | 固定版本 Inspect task/solver/scorer 生命周期；`adapters/inspect` | W04;W05 | Adapter owner | AB23–AB26 |
| W08 | N1/P0 | 两个 client profile × 24 Task 接入、旧例回归、边界/crash | W02–W07 | Evaluation + Security | AB27–AB29 |
| W09 | N1/P0 | 干净安装指南、诊断指南、性能基线、N1 PR 收尾 | W08 | Release owner | AB30–AB32 |
| W10 | N2/P1 | 24 family、有限参数 schema、72 instances、质量关联 | W09 | Evaluation owner | AB33–AB35 |
| W11 | N2/P1 | split/disclosure/freeze/deprecate 与包评审记录 | W10 | Evaluation + Maintainer | AB36–AB37 |
| W12 | N3/P1 | 冻结 baseline plan、配对配置、trial/shard 目录 | W11 | Evaluation owner | AB38–AB40 |
| W13 | N3/P1 | 固定分母、多轴结果与成本未知语义 | W12 | Evaluation owner | AB41–AB42 |
| W14 | N3/P1 | family 分层 bootstrap、绝对门槛、四态判定 | W13 | Evaluation reviewer | AB43–AB45 |
| W15 | N4/有条件 | live readiness、显式计划与实际六任务/pilot | W14;具体 live 授权 | Adapter + Security | AB46–AB48 |
| W16 | N4/有条件 | 一个实际产品宿主与三名非作者 first-value | W09;目标/环境/参与者 | Adapter + Maintainer | AB49–AB50 |
| W17 | N5/发布门槛 | 兼容声明、过期/撤销、版本/恢复/发布候选 | 所声明范围前置完成 | Release owner | AB51–AB54 |
| W18 | N5/独立子轨 | 一个领域≤3操作的独立 reference 与差分 | 获准且可复位 reference | Fidelity owner | AB55–AB56 |

建议 PR：N1 可按 W01–W02、W03–W05、W06–W07、W08–W09 四个审查切片交付；对外只在最后一片满足门槛时称插件闭环完成。不要在一个 PR 同时修改世界规则、任务 oracle 和 live 模型基线，以免混淆回归原因。切片之间维持可构建状态，测试各自契约。

## 3. 逐项验收矩阵

每条验收要求至少一个正例、一个针对该规则的反例和可定位证据。测试选择应与变更相称；不要为了每种 Markdown 显示组合都跑完整模型评测。

### N0：基线与契约

| ID | 必须证明的结果 | 拟议证据入口 |
|---|---|---|
| AB01 | 以实施当时 main 为基线，0077 已集成；不会覆盖新改动或把旧 CI 当新 head CI | Git ancestry/文件 diff、候选 PR、精确 run 元数据 |
| AB02 | N0 决定固定 runtime/SDK/协议/Inspect/Python 版本，新增格式与旧格式兼容政策明确 | Accepted ADR、锁定依赖、contract fixtures；无自动消费授权 |

### N1：插件闭环

| ID | 必须证明的结果 | 拟议测试/验收入口 |
|---|---|---|
| AB03 | 所有新格式严格解码；未知/重复/null/错误类型/超限失败 | `TestBaselinePackAdmission`；全部字段表和 N/N+1 |
| AB04 | 路径越界、软/硬链接、大小写别名、输出子树和自引用拒绝 | `TestPluginReferenceIsolation`；Windows/Unix fixtures |
| AB05 | 冻结后改 pack/Task/world/profile 不改变会话；同 revision 不同内容拒绝 | `TestPluginFrozenInputs`；无新 hash 清单 |
| AB06 | list 仅 Task 工具；直接调用未列出工具仍被拒绝且不改变状态 | `TestPluginTaskToolSurface` |
| AB07 | 参数级 authority 在服务端生效；合法调用、越权尝试和实际副作用分开 | `TestPluginAuthorityCannotBeBypassed`；直接 wire call |
| AB08 | 旧 in-process 与新插件对相同顺序调用产生相同世界和评分；旧 generic serve 不变 | `TestTaskDispatcherCompatibility`；旧回归 |
| AB09 | stdio 初始化/协商/list/call/error 符合声明 profile；未知版本不假装成功 | `TestPluginStdioProtocolProfiles`；MCP conformance |
| AB10 | stdout 无日志/隐私，长行与背压受限；broken pipe 不伪称输出成功 | `TestPluginStdioLimitsAndOutput` |
| AB11 | mutating calls 有确定串行顺序；队列超限/取消不会产生额外提交 | `TestPluginConcurrentAdmission`；race |
| AB12 | MCP 不能调用 lifecycle；错 token/错 session/非父进程通道拒绝 | `TestPluginControlIsolation`；各平台 IPC 权限 |
| AB13 | 两个 session 互不见状态、工具事件、输出和控制能力 | `TestPluginSessionIsolation` |
| AB14 | 只有准入成功才能 claim/start；最后一个引用坏时零执行零输出目录 | `TestPluginPreflightNoEffects` |
| AB15 | finish 只停止交互，目标没完成仍判失败；晚到调用不能改 terminal | `TestPluginFinishBarrier`；取消与提交竞态 |
| AB16 | EOF/crash/timeout/取消/cleanup 失败保留真实状态；重复 finish 不重评分 | `TestPluginTerminationMatrix` |
| AB17 | 原始世界、Task、runtime、评分绑定可重验；旧/伪造 terminal 无法替代 | `TestPluginEvidenceBinding` |
| AB18 | 写/短写/Sync/Close/发布/Remove 故障保留 partial 或 residue；不覆盖用户目录 | `TestPluginPublicationFaults` |
| AB19 | inspect 只读，不发 MCP/provider 请求，不重跑 qualification；缺证据失败 | `TestPluginInspectReadOnly` |
| AB20 | lifecycle/execution/evidence/cleanup 独立呈现，错误码有限且不回显敏感输入 | `TestPluginFailureProjection` |
| AB21 | check/describe 不联网、不读凭据、不写宿主配置；输出投影无 oracle | `TestPluginCLIPreflight` |
| AB22 | CLI JSON/可读输出计数和结论一致；短写不会回滚已发布结果 | `TestPluginCLIOutput` |
| AB23 | Inspect 用独立进程和真实 MCP wire 跑一个完整 Task，state 跨调用保留 | `test_inspect_session_lifecycle` |
| AB24 | task 投影无隐藏字段、其他 tools 和历史会话；额外工具未确认时拒绝资格 | `test_inspect_agent_projection` |
| AB25 | scorer 来自 State Twin terminal；模型自报 success 和框架日志不能覆盖 | `test_inspect_scorer_binding` |
| AB26 | adapter 异常/取消/finally/框架 resume/缓存行为不重试 live、不泄漏实例 | `test_inspect_cleanup_and_resume` |
| AB27 | 24 Task 均经两个 client profile 运行，覆盖有效正例、错误对象和预期拒绝；保留原 103 case | 接入矩阵与任务 case 测试；mock 标签明确 |
| AB28 | oracle/control/secret canary 不进入 Agent 工具、stdout、stderr、prompt 或公开日志 | `TestPluginDisclosureBoundary`；Inspect log 负例 |
| AB29 | 100 次短会话、10 次分阶段 crash 后资源不增长且无自有僵尸；不清理用户实例 | lifecycle soak 报告、跨平台进程归属检查 |
| AB30 | 干净环境从本地发行物按指南完成 check/connect/run/inspect；不需全局配置写入 | CLI/adapter 安装流程测试 |
| AB31 | 插件性能按固定环境≥30次，含进程/框架开销；满足门槛或有已接受修订 | benchmark 数据及 p50/p95/max/RSS/开销来源 |
| AB32 | 全部 N1 契约及旧回归通过；gofmt/vet/race、平台、安全、conformance 对最终 head 完整通过 | exact-head PR/CI；临时产物清理、无实施剩余 |

### N2：能力资产与基线维护

| ID | 必须证明的结果 | 拟议测试/验收入口 |
|---|---|---|
| AB33 | 24 family/72 instances 的数量及引用明确，变体不虚增独立能力数 | `TestBaselineFamilyInventory` |
| AB34 | 有限参数生成不执行脚本；目标/fixture/oracle 同步，非法参数拒绝 | `TestBaselineVariantAdmission` |
| AB35 | 每个变体正负例和 metamorphic 关系有效；合成变异不算 Agent 轨迹 | `TestBaselineVariantQualification` |
| AB36 | public/private disclosure 正确；公开后不再称未见样本，family 相关性保留 | `TestBaselineSplitDisclosure` |
| AB37 | 冻结包不可同 revision 原位更改；弃用/撤销保留旧证据；独立评审缺失可见 | `TestBaselineRevisionLifecycle` + 实际评审记录 |

### N3：实验与判定

| ID | 必须证明的结果 | 拟议测试/验收入口 |
|---|---|---|
| AB38 | config/Task/world/顺序/重复/阈值在开始前冻结；请求别名不伪装 observed snapshot | `TestBaselinePlanIdentity` |
| AB39 | shard 符合 240/20/12 上限；旧 project/campaign 的限额与 offline 格式不变 | `TestBaselineShardBudgets` |
| AB40 | crash 或未授权后续 shard 保留所有 planned rows；不自动重跑或续接 partial | `TestBaselineShardInterruption` |
| AB41 | 缺失/未评分/无证据/策略失败各自计数，不能裁剪成更好分数 | `TestBaselineFixedDenominators` |
| AB42 | timeout 不算快速成功；usage/价格缺失为 unknown；不混用不同成本单位 | `TestBaselineMetricsAvailability` |
| AB43 | family/领域分层 bootstrap、固定 seed 与版本可复现；不足样本不推断 | `TestBaselineClusteredInference`；独立数字 golden |
| AB44 | validity、policy、floor、非劣按顺序判定；pass/fail/inconclusive/invalid 均有实例 | `TestBaselineDecisionPolicy` |
| AB45 | 全成功/全失败/相关变体/不平衡缺样本/边界 delta 不能伪造显著改进 | `TestBaselineStatisticalCounterexamples` |

### N4–N5：实际采用与发布

| ID | 必须证明的结果 | 拟议证据入口 |
|---|---|---|
| AB46 | 未批准/过期/错误计划/预算耗尽时零 provider 请求；Inspect live 不继承其他 profile 批准 | live readiness 注入式负例与 outbound audit |
| AB47 | 实际六任务冒烟包括成功、拒绝、取消和不确定响应；失败如实保存 | 具名模型、时间、精确版本、sanitized 实际报告 |
| AB48 | 批次仅按批准的冻结计划执行，pilot/发布比较不互相冒充，重放与随机模型结论分开 | 实际 shard/实验报告、预算记录 |
| AB49 | 一个具名产品宿主独立测安装/工具/隔离/退出；未通过者 unverified，不借 API 结果 | 产品版本、步骤、正负例与 TTL 报告 |
| AB50 | 至少三名非作者 first-value 观察，记录≤10分钟目标与真实失败，不虚构用户 | 获授权的试用记录与问题闭环 |
| AB51 | HostProfile/adapter/协议/pack 任一绑定变更立即影响 claim；过期/撤销状态可见 | 兼容矩阵派生测试与真实证据 |
| AB52 | old/new decoder、copy-to-new 恢复、损坏/丢失/取消和不覆盖行为成立 | 版本兼容/恢复 fixtures；旧格式回归 |
| AB53 | 发布候选完整检查、所声明平台安装、秘密扫描、文档与候选版本一致 | exact-candidate CI、候选发行物安装证据 |
| AB54 | 维护角色、支持范围、弃用/撤销流程明确；preview 与 stable 宣称不混淆 | 发布说明审查、claim registry 映射 |
| AB55 | reference 来源独立、已获授权、可复位，语义和未覆盖范围固定 | reference 合约与重置测试；缺来源则 blocked |
| AB56 | ≤3操作的成功/失败/重复/权限差分成立，回归使 L2 子集失效 | golden differential 报告；不推广到整个领域 |

## 4. 实施验收记录模板

未来每个工作包填写：

| workItem | acceptanceIds | implementationFiles | executedTests | result | headRevision | evidence | remaining |
|---|---|---|---|---|---|---|---|
| W00–W18 | 对应 AB ID | 当前为空 | 当前为空 | not_started | 当前为空 | 当前为空 | 等待明确实施授权及相应外部条件 |

result 仅允许 `not_started / in_progress / passed / blocked`。只写测试名称没有执行结果不能记 passed。blocked 必须写清具体缺什么、已尝试方法、可继续的独立工作；不得将个人时间或“已经完成几项”当作阻塞。

每个 PR 描述列明完成/剩余 AB ID、行为变化、失败语义、兼容性、实际验证和精确 head。设计 PR 通过不等于实现通过；merge 之后仍区分提交检查和新主线检查。

## 5. 质量门禁与测试成本

- 格式/路径/类型/状态机用快速单元测试，错误注入不重复运行完整 24-task pipeline。
- 关键端到端用真实独立进程与 MCP wire；不能全部改成内部函数调用后宣称插件验收。
- 固定小型 fixture 做 race；插件平台清理以三平台实际进程测试证明。无平台证据的能力标 unsupported。
- provider/live 和外部参与者不进入普通 PR CI；mock readiness 不能关闭 AB47–AB50。
- 性能 soak 与 CI 总超时单独记录，不修改业务 deadline 来掩盖失败。
- 新需求必须说明替换了哪个旧需求或增加了哪个验收，不默默删除困难反例。

## 6. 完整交付定义与不依赖外部条件的工作

N1：AB01–AB32 全部有执行证据，W00–W09 完成，PR 已推送、最终 head 检查通过、指南可走通、自有临时实例清理，才算完成。N2/N3 同理要求各自全部 AB 通过。

N4 需要具体账号/预算/宿主环境/参与者。即使这些暂缺，N1–N3 仍有完整、独立、可交付的工程范围；live 所需的准入、mock、错误与取消 readiness 也可继续实现。N5 只允许按已通过范围作发行宣称，不能把外部证据缺口写成可忽略。

有限 L2 子轨单独声明；缺 reference 时，不阻止 L1 baseline plugin preview 发行，但 AB55–AB56 和 L2 宣称不得关闭。

2026-10-02 已明确收到整批实施授权，本轮按完整可执行范围推进与 PR 交付。
