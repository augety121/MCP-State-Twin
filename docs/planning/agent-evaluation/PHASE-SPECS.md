# 近期阶段规格：从脚本世界到 Agent 回归证据

状态：Proposal / v3.1，2026-09-08。本文细化总规格第 7–15、24–25 章；
下述 MUST 表示待接受设计要求，不表示当前实现。新 kind、字段、命令均不可直接交给现有 CLI。
既有 Phase 0–7、RFC/ADR 及旧 JSON 枚举保持不变；完整 D0–D8 生命周期见总规格。

## 1. 分阶段交付与依赖

| 阶段 | 工作包 | 开发进入条件 | 退出证据 | 明确不能声称 |
|---|---|---|---|---|
| P-A：设计封闭 | B01、B02/B03/B04/B07/B12 设计 | 固定源码与现有权威 | 接受局部 ADR、六任务卡、字段/错误/预算与负例清单 | 已实现、109 项已验收 |
| P-B：离线执行切片 | B02–B07、B12 | P-A 对应语义已接受 | mock loop、评分、停止、脱敏、verify/replay 集成测试 | 真实模型兼容 |
| P-C：离线回归产品 | B09-offline、B10-offline | P-B 工件合约 | 两组合成观察、完整分母、不可比/回归/不足报告、干净环境教程 | 两个真实配置验证、外部采用 |
| P-D：有限 live | B08、B09-live | P-B；已批准配置、数据、有限预算 | 六任务实际报告、实际失败保留；两配置比较另有预算 | 全模型通过、产品宿主兼容 |
| P-E：外部使用 | B10-external | P-C；live 示例需 P-D | 独立使用方完成一次任务/配置比较并反馈 | 维护者演示等于真实采用 |
| L：本地发行 | B11 | accepted core 与 exact candidate | 现有 Phase 1 支持清单和发布 gate | 所有同包 preview 都 stable |

P-C 可以先于 P-D；L 与 P-B/P-C 独立推进。B07 的保存前策略是 P-B 的前置设计，
不是要求在尚无 Episode 时先完成全部 evidence 实现。B12 是运行入口约束，不能延期到 live 后。
P-B/P-C 全通过仍只完成离线子项；D2 的 live、D3A 的外部使用不能被关闭。

远期 B13–B32 保留原 Backlog 范围：澄清、任务扩展、原生/产品 profiles、fidelity、保留/迁移、
性能和稳定维护分别按需求/安全 gate 启动，不将 32 个包一次性接受为必做功能。

## 2. P-A：Task 与输入合约

建议 `AgentTask` 使用独立 alpha kind；Bundle v1 不塞入旧 parser 不认识的新含义。
近期用现有已验证 Bundle 加独立、只读 Task sidecar；是否扩展 Bundle 的新 manifest
另有格式决策。sidecar 引用必须落在指定根目录，拒绝绝对路径、越界、symlink 与外部 URL。
发布工件不能依赖维护者机器上的路径。

| 字段组 | 初期合约 |
|---|---|
| identity | 稳定 task ID、format revision、domain revision；ID 不得复用给不同目标 |
| world | 固定 Bundle/TwinSpec/fixture 引用、初态、工具 surface、fault/time profile |
| objective | 用户可见目标和授权；不能运行后改写 |
| visibility | 分开 agent-visible objective/context 和 private oracle/witness；禁止直接序列化整个 Task 给模型 |
| tools | canonical 业务工具名集合；未知/缺失/名称投影冲突 preflight 拒绝 |
| authority | 业务动作及资源集合；先限定精确 repository/issue，不接受任意字符串脚本 |
| oracle | 项目内版本化声明式断言；无动态脚本、文件/网络函数或原生插件 |
| expected_response | 可选有界 JSON 事实契约；无该契约时，不对自由文本事实做确定性 PASS |
| budgets | 轮次、工具、墙钟、输出、trace、费用政策；必须完整有效 |
| mode | blind 初期唯一严格对比模式；diagnostic/coaching 需显式标记和独立 cohort |
| solvability | 私有 witness/替代路径、所需信息与工具；无法成立则 task_invalid 或 blocked |

初期不含模型用户模拟器、自动目标变更、任务生成器或任意 plugin。任务需要澄清时，
要么使用已接受的固定回应表，要么暂不准入；不得让模型猜 oracle 中的隐藏目标。

建议初始资源配置 `agent-eval-quiet-alpha1`（设计值，非性能测量）：

| 对象 | 建议硬 admission/运行上限 |
|---|---|
| Task 文件 / JSON 嵌套深度 | 256 KiB / 32 层 |
| 用户目标 / 可见上下文 | 8 KiB / 32 KiB UTF-8 |
| 工具数 / assertion 数 | 32 / 64 |
| 单次 API 响应 / 工具输入输出 | 1 MiB / 不超过已有 runtime profile 的上限 |
| 模型请求 / 工具尝试 | 16 / 32，拒绝的调用也计入尝试 |
| 单请求 / Episode / drain+cleanup | 30 秒 / 180 秒 / 10 秒独立清理期限 |
| trace 事件 / 内容总量 | 512 / 8 MiB；预留 64 KiB 控制与终态记录空间 |
| oracle | 受限已有表达式预算，每条 10,000 cost；最多 64 条；无外部访问 |
| 并发 | 一次一个 Episode；一轮工具 ordered-sequential；沿用 quiet 一个 Go slot |

这些上限正式接受时版本化，缺失/零/负数/溢出/超 profile 上限全部拒绝，不以 0 表示无限。
预算可以进一步收紧；放宽上限视为新 profile，不是 silent fallback。现有 core ResourceProfile
不因新 harness 配置改变；需要改变 core 语义时另行说明。计数和字节可强制，Go slot 与 heap
仍是软控制，不能保证 CPU 百分比、风扇噪声或硬 RSS。

## 3. P-A/P-B：只读评分合约

### 3.1 三种证据视图

1. `BusinessView`：固定截止点的实体、业务序列与必要业务字段。
2. `CommitView`：截止前已提交的业务操作、目标、结果与故障关联。
3. `ObservationView`：Agent 请求过、被拒绝、得到或未得到的工具内容；不混入私有效果标记。

只读调用可能增加 audit、call_count、head 或消耗已声明 fault 状态。业务不变断言只比较
任务声明的实体/序列；runtime 元数据变化单列，不能把正常读取判成越界。
读任务不能仅凭终态未改变通过：需要已交付事实与符合任务契约的最终回答。

### 3.2 断言的最小集合

- entity-field equality / existence / count；JSON Pointer 必须存在且类型匹配；missing 不等于 null；
- allowed business diff：create 允许一个新 issue 与相应业务 sequence；close 只允许目标 state/closedAt；
- committed operation count/target：防止错写再恢复逃过终态断言；
- attempted / blocked / committed violation：三个计数分别保留；
- delivered observation predicate：验证恢复确认是 Agent 实际获得的信息；
- expected final JSON facts：只用于明确需要回答的任务，不把自然语言 judge 混作确定性事实；
- 全部谓词有对象与事件遍历上限；越限是 evaluator/resource error，不是默认 true。

“允许目标之外未修改”必须覆盖不存在于初态、后来新建的对象。未知 extra field、错误对象、
多一次 effect、删后重建均有负例。只匹配最终 entity 集合而忽略业务序列不够。

### 3.3 多轴结果和优先级

固定三个主维度：execution_status、task_outcome、evidence_status。
另列 goal_satisfied、policy_attempts、blocked_attempts、committed_violations、recovery、cleanup_status。
不得为了单个总分丢掉这些事实。

| 条件 | 归类/结论 |
|---|---|
| preflight Task 自相矛盾、无工具/信息合法路径 | TASK_INVALID；不发起模型；计入计划排除清单 |
| 到不了 READY、adapter/隔离失败 | HOST_NOT_READY / infrastructure_error；不是 Agent task_failed |
| Agent 停止但目标未满足 | task_failed，若证据足够；不是 host_error |
| 目标满足但已验证越界 | policy_violation；禁止 success 覆盖 |
| 任务预先允许合法不行动且证据成立 | expected_abstention；与普通 success 分栏 |
| 取消/超时/预算耗尽，终态仍可读 | 保留 execution 原因；可报告已观察事实，不给自动升级 PASS |
| 序列/终态关系损坏 | integrity_failed；不得使用无效工件推断任务能力 |
| 清理未确认或在途提交无法判定 | partial / quarantine；禁止完整认证 PASS |

恢复评分不强制唯一路径。仅当任务明确要求确认时，要求提交后有有效的已交付读观察；
允许 get 或能证明同一事实的 list，不强制固定 get-close-get 顺序。

## 4. P-B：RunDefinition 与 local bridge

### 4.1 两种身份不能用一个摘要替代

RunDefinition 是一次配置的完整身份；改变候选模型/提示词后它理应不同。
ComparisonKey 固定 task/domain/world/fault/evaluator/mode 和未被声明为实验变量的策略。
比较先验证 ComparisonKey，再核对 plan.allowed_differences；不得要求两个 RunDefinition 完全相等，
也不得只比较 task ID 而忽略变更的 oracle、预算或工具授权。
滚动模型别名无 snapshot 时记录 unknown；这降低可归因精度，不编造 pin。

每次 live trial 新 conversation/context；禁止复用上一 trial 的反馈、continuation 或工作区。
模型必要 opaque continuation 只在当前运行私有内存中传递，不分析为隐藏思维，默认不封存。
不支持安全续接的 API profile 应拒绝，不能删掉必要上下文却声称等价。

### 4.2 权限与安全分层

首条路线的本地 harness 是可信应用，远程模型只能提出函数调用，不能访问 shell、文件、
控制端口、连接 URL 或 provider key。服务端 endpoint、branch 由 harness 绑定，不接受模型输入。
preflight 检查合成数据和固定 provider endpoint；禁止重定向携带凭据到任意地址。
本路线的进程级 OS 网络隔离尚未实现前只能说明逻辑 allowlist，不得声称 kernel egress 沙箱。

模型请求工具名先过 allowlist，再过输入 Schema，再过资源授权，再调用本地 MCP。
业务目标只允许精确资源时，资源策略在 dispatch 前拦截其他对象；attempt 仍进入评分。
若未来任务有意观察可提交的业务策略违规，必须明确区分“硬 capability”与“可评分政策”，
另设 profile，不能秘密放松当前硬授权。
直接通过 engine 调用的测试只证明 engine，不替代 bridge 经 MCP 的集成证据。

### 4.3 Provider-neutral 事件合同

最小事件：model_request、tool_requested、tool_rejected、tool_dispatched、world_committed、
tool_delivery、agent_final、stopping、evaluation、seal、cleanup。
每条绑定 trial/attempt、单调 observation sequence、canonical tool、private call reference；
world head/commit sequence 是另一序列，不能把一次 model turn 当成一次事务。
无 world commit 的拒绝不伪造 head；不支持的结果类型显式拒绝。

首个 adapter 建议 OpenAI Responses 非流式函数工具 profile（待接受）；
函数名、JSON arguments、call_id 与回传结果必须保持关联，依据见 SOURCE-REGISTER S11。
暂不接受 streaming partial arguments、async tool、工具搜索、shell、computer、动态工具集。
API 不支持请求设置时 HOST_PROFILE_UNSUPPORTED，不自动换模型/协议/工具。

### 4.4 批次、重复与错误

收到一轮工具数组后先检查整个批次的协议形状、数量、唯一 call ID；此阶段失败不执行该批次。
全部结构合法后按数组顺序逐项进行权限/Schema/预算 admission，再执行。
执行到第 k 项失败不会回滚前 k-1 项已经提交的独立工具事务；报告保留这个前缀。
业务拒绝作为明确工具结果可继续；协议/安全/基础设施终止类错误则停轮，余下调用记未执行。

同一 trial 的重复 call ID（无论内容是否相同）在首版拒绝为协议异常，不把它当业务幂等键。
不同 call ID 但同样业务输入是新的尝试，由领域语义决定结果。
若以后加入 captured response 去重缓存，必须有独立协议重投范围和冲突测试，不能宣传 exactly-once。

普通 HTTP/MCP 重试默认关闭；provider 请求接收状态未知时停止，不重复计费试探。
domain after-commit fault 是任务中允许的可观察不确定结果：Agent 可执行授权的读取确认，
harness 不得给它私有 effectsCommitted 标记或自动代它重发 mutation。

## 5. P-B：停止、封存与崩溃

### 5.1 Admission 截点

`CREATED → VALIDATED → PROVISIONING → READY → RUNNING → STOPPING → EVALUATING → SEALING → TERMINAL`。
这是新对象的提案状态，不重解释旧 Episode 状态。每阶段失败都有 typed failure path；
验证失败不得要求 branch 必定已创建，cleanup handle 可为空但状态必须明确。

STOPPING 首先原子关闭本 trial 的 dispatch gate。之后禁止新模型请求和新工具请求，
即使迟到 provider batch 看起来合法。已开始的本地工具有界 drain；记录最后确认的 head 和
已知提交前缀。cancel 不能承诺撤销已提交的效果，也不能把停止本地等待称为远端取消成功。

只有全部允许的在途操作已收敛、且终态与 commit 截止点一致，才能做完整评分和封存。
drain 超时/进程失联时：quarantine 该世界，不重用，不声明 complete，不自动重跑；
后续 reconcile 仅做只读核对，属于独立恢复工作。数据库无歧义与模型是否收到结果分别判定。

### 5.2 持久化顺序

新的 AgentEvidence 先用独立 bounded artifact，不把它强塞进当前 scripted Journal 的 payload。
创建 run manifest 的 no-clobber claim → 记录准备状态 → 执行/append 可保留事件 →
固定截止终态 → 只读评分 → 撤销访问和清理运行资源 → 写终态 manifest。
删除临时 world 之前，必须已在私有 staging 持久化并验证完整 replay 闭包（初态、事件、
终态及其版本引用），或明确选择 non-replayable 保存政策。终态发布失败时保留安全 staging
以便只读恢复；不能先删唯一数据库再发现缺少证据。关闭 listener/撤销能力不等于删除工件。
写入采用私有临时路径和原子发布/可恢复状态标记；失败不覆盖旧 artifact。
平台不支持预期原子行为时，报告 unsupported，不能用同名文件存在作为完成证据。

TERMINAL 包含 cleanup_status。已封存档案不得因后来清理重试而改写；另追加引用旧证据的
cleanup 补充报告。未完成目录在重启时只能 inspect/verify，首版没有自动 resume live。
旧 Journal 和 remote fenced completion 保持现状，不接收新 kind 直到单独版本化 admission。

## 6. P-B：trace 与 world replay

### 6.1 保存前策略

Agent-visible 和持久化可见是两种政策。工具业务内容必须先是获准合成数据；公开 trace
只保留 schema 白名单内的合成内容、类型化事件和必要映射。provider header、session ID、
capability URL、API key、opaque continuation 默认不写磁盘，错误日志和临时文件同样受限。
使用 trial 内序号替换 provider 私有引用；不得公开可枚举的私有 ID 哈希。

扫描只能提供有限检测，不能保证发现任意秘密。准确声明为“合成来源 + 白名单 + 已测试敏感
模式阻断”；检测到敏感/未知内容时拒绝保存该内容，终止/降级为 partial，保留去敏错误码。
不得为了 replay 保存真实凭据，也不得先写日志再扫描。

### 6.2 等价与完整性

可重放 world 的必要数据：固定 TwinSpec/fixture 与完整初态、资源/时间/熵/调度/fault 身份、
全部有序世界输入/控制事件、对应返回/提交序列、终态和 oracle。模型完整原文不是 world replay
必需；没有可恢复的 world 输入时 `world_replayable=false`。

将私有元数据改成局部别名不改变业务值；一旦脱敏改动了业务输入、对象关系或输出，标记
non_equivalent，不得继续称为原 world replay。digest-only 也不能恢复缺失的原文。

verify 必须分别报告结构有效、内容完整、关系一致、重放匹配；不把单个摘要匹配称为真实性。
完整伪造并重算摘要的攻击者可以造自洽档案；本地 unsigned artifact 不证明真实 provider 来源。
未来签名只提供来源/完整性，不提供模型正确或 L2 保真证明。

trace 控制保留空间满、磁盘满或 seal 失败时不丢掉中间事件后伪称完整；可生成脱敏的 partial
终态报告，若连报告无法写入，CLI 非零退出并给出安全错误，不打印 secret/body。
replay 是纯离线命令，不调用模型，不评判新模型能力。新 live rerun 创建新 trial，费用另计。

## 7. P-C：比较与分母

计划先固定 expected trial 集合，每个 key 为 task/domain/fixture/fault/repeat/config。
禁止重复接受同一 key 的不同结果；retry 若获准必须是新 attempt 且保留旧失败记录。

计数采用可核算集合，避免类别重叠：planned = not_started + started；
started = running_or_incomplete + terminal；terminal 分 execution_status；
validly_evaluated 是正交子集，另列 evidence/cleanup 有效性。task_invalid 出现在 preflight
或后验发现时均保留在计划分母，不能悄悄删行。

配对实验区分固定 ComparisonKey 与 allowed_differences；模型不同是合法实验变量，
oracle/domain/policy 改变默认不可比。预算有意作为变量时声明“整体配置比较”，不归因于模型。
初期采用按 task/repeat 交替的固定顺序；真实墙钟耗时不驱动虚拟时间。

报告独立字段 `decision`：regression / no_regression_observed / inconclusive / incomparable。
自动升级 PASS 只在预注册完整门槛满足时产生；`no_regression_observed` 不自动等于可安全升级。
关键新的可复核越界立即阻断；缺失或无效证据至少 inconclusive；不以失败重跑中的最好一次替换。
初期小样本输出分子分母和配对差异，不默认给显著性或精确概率；统计方法新增时单独版本化。

成本未知必须显示 unknown；request timeout 也可能计费。API 费用、工具次数、墙钟、virtual time
分别记录。免费 mock 的零费用不能推算 live 价格。参数/模型别名不精确时，报告可归因限制。

## 8. P-D：付费试验与资源约束

默认 `live_enabled=false`；preflight 不联网探测模型。凭据存在不等于运行授权。
运行需提供获批 plan、明确 provider/model、合成数据政策、任务/重复数、总请求上限和费用方案。
不得默认在 PR CI 打开 live；不向不可信 PR 暴露 secrets。

对可计算价格的请求，发送前预留保守上界；无单价/usage 时按维护者批准的请求数和输出上限
受控计划运行并标价未知，不能声称实现硬货币封顶。远端供应商的硬配额需实际配置和验证，
客户端取消不能保证避免在途计费。

先单任务 live 验证，再六任务小组，再两个配置；每次扩展不得超过已批准预算。API 权限、
配额或部署不可用时停止对应 live lane，但 P-C 和 B11 离线工作不因此停摆。
真实运行的失败、未触发 fault、未完成、cleanup 失败都进入档案。

本机默认 quiet，测试时 GOMAXPROCS=1、Go -p 1，一次仅一个编译/测试任务。
CPU 占用仍可能有短暂波动；压力、race/fuzz 和跨平台重放可放获准 CI，不在本机并发铺满。

## 9. P-E 与 L：交付、不完整证据和回滚

离线 first-value 只使用当前真实命令或已合并的新命令，教程逐步注明 scripted/mock/live。
未实现的 eval/task/evidence 命令只能留在设计文档，不能放到 README 可复制 quickstart。
外部试用须真实独立使用方及同意，不替用户发邀请、不编造反馈或采用量。

B11 在发行前逐项解决 RFC-0002、Phase 1 与 preview ADR 的支持范围映射；缺少 release-scope
接受或 exact-candidate CI 时不发布稳定 tag。没有 live 不是本地 core 的阻塞项，但同包新增
bridge 不得悄悄让 hermetic 命令具备外网路径。版本号在接受前只作规划，不写入 go/runtime。

回滚以禁用/移除新 experimental 入口为第一选择，保留旧 Scenario/Bundle/Journal 数据可读。
不可覆盖旧证据或自动 schema downgrade。新 oracle 修复通过版本与 supersedes 关系重新评价，
旧分数保留历史；新 live 需要新授权，不能用“修复报告”触发付费重跑。

## 10. 阶段 DoD 和验证轨道

P-A 只要求文档完整与冲突处理；进入 P-B 后才开始新增实现和测试：

- Task：严格解码、路径/资源/未知字段、合法 witness、不同合法路径；
- evaluator：正例、漏做、错对象、额外效果、先错写再恢复、只读副作用；
- bridge：未知工具、重复 ID、错误 JSON、多调用、续接、private fields 不泄漏；
- lifecycle：每阶段失败、stop 与 commit 竞争、迟到调用、磁盘/seal/cleanup 失败；
- evidence：缺事件、替换终态、错 call 链、脱敏破坏关系、离线 replay、不调用模型；
- compare：变量匹配、缺失分母、重复 trial、失败选择、不可比/不足与真实回归；
- CI：普通 PR 无秘密执行 mock；跨平台/race 使用明确工具链；live lane opt-in 独立授权。

实施需 gofmt、go vet、go test -race 及改动相称验证；平台不支持 race 时明确说明并使用
实际 CI 证据。本轮文档阶段不运行这些业务测试，不把计划中的测试名写成已存在。
完整 109 原稿需求映射和 24 原任务审查需补齐原材料，不能用以上清单替代后宣称全包完成。
