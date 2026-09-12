# 补充需求与验收清单

> 后续实施记录：见 [ADR-0038](../../ADR-0038-AGENT-TASK-AND-OFFLINE-GRADING.md)
> 、[ADR-0039](../../ADR-0039-OFFLINE-AGENT-REGRESSION-LOOP.md)
> 和 [Implementation Status](../../IMPLEMENTATION-STATUS.md)。以下状态表保留原 Spec
> 评审时点；不将已经实现的有限 offline subset 误报为不存在，也不将整个 AE 条款自动关闭。

状态：Proposal；2026-09-08。AE 编号是本轮新条款，不替代 109 项缺失的 MST26 需求正文。
所有“验收”列均为计划，不是测试执行结果；本轮只有文档检查。实现状态一律 not-started，
除非明确为继承要求；此处不修改 IMPLEMENTATION-STATUS 或 public claim。

| ID | 要求 | 工作包 | 必须验证的正反例/产物 |
|---|---|---|---|
| AE-001 | 旧 Scenario/Bundle/Journal 不重解释为 Agent 运行 | B01/B02/B06 | 旧 CLI/golden 继续可读；新 kind 不被旧 reader 当成功 |
| AE-002 | Task 严格、有界、路径安全 | B02 | 未知字段、多文档、深度/大小、外部路径、symlink 拒绝 |
| AE-003 | 六任务具有合法可观察路径 | B02/B03 | 每卡 witness 与非唯一路径；缺工具/信息 task_invalid |
| AE-004 | agent-visible 与 private oracle 分离 | B02/B04 | 模型请求、工具投影、工作区无 oracle/witness/control 内容 |
| AE-005 | 业务变化与运行元数据分开评分 | B03 | 只读推进 audit/head 不误报；漏读/错答不通过 |
| AE-006 | 终态与过程联合 oracle | B03 | 正常、多合法路径、漏做、错对象、额外效果、错写再恢复 |
| AE-007 | attempted/blocked/committed 分栏 | B03/B06 | 被权限阻断也留尝试；goal true 不能覆盖越界 |
| AE-008 | RunDefinition 与 ComparisonKey 分离 | B04/B09 | 合法模型差异可比较；oracle/domain 差异拒绝 |
| AE-009 | fresh run 与 opaque continuation 私有 | B04/B05 | 无跨 trial 反馈/会话继承；必要续接不丢失且不持久化秘密 |
| AE-010 | provider surface 投影无隐性损失 | B05 | 碰撞、工具丢失、不支持 Schema 拒绝；双侧身份保留 |
| AE-011 | tool batch、ID 和顺序有合约 | B05 | 重复 ID/错误形状在 dispatch 前失败；独立事务保留成功前缀 |
| AE-012 | dispatch 绑定 branch/tool/resource | B04/B05 | 参数无法改 endpoint/branch；错资源被拒；不借控制凭据 |
| AE-013 | 整次运行预算，不只是 HTTP timeout | B12/B06 | 累计超限停止；拒绝尝试计数；未知费用不记零 |
| AE-014 | STOPPING 阻止新调用并固定 cutoff | B06 | stop/commit 竞争、迟到批次、未收敛 quarantine/partial |
| AE-015 | fault 的世界效果与模型观察分开 | B06 | 提交后错误、无额外 mutation、get/list 合法恢复；不泄漏私有标记 |
| AE-016 | 保存前 trace 白名单和敏感阻断 | B07 | secret 哨兵不落临时/最终/错误文件；未知字段拒存 |
| AE-017 | replay 能力依完整业务输入声明 | B07 | 缺输入/digest-only 非完整 replay；脱敏改语义 non_equivalent |
| AE-018 | evidence 交叉关系与来源边界 | B07 | 序列缺口/错引用/换终态失败；unsigned 不证明 provider 来源 |
| AE-019 | sealed 不覆写、crash 不自动 live resume | B06/B07 | no-clobber、磁盘满、半封存、旧目录 inspect-only |
| AE-020 | cleanup 为终态维度 | B06/B07 | 清理失败留报告、隔离不重用；后来重试用补充工件不改旧档案 |
| AE-021 | compare 保留完整分母与失败 | B09 | planned 等式可核算；重复 trial/选择最好重试拒绝；无效 evidence 不通过 |
| AE-022 | 可比/回归/不足分开，计划预注册 | B09 | 关键新越界阻断；小样本不自动 PASS；费用未知单列 |
| AE-023 | live 授权与契约测试不混淆 | B08/B09 | 无授权零请求；实际 profile 有报告；mock 不晋级 live/API 不晋级产品 |
| AE-024 | quiet、独立发行、真实外部验证 | B10/B11/B12 | 单 Episode；支持清单/候选 CI；无假用户/版本/采用量 |

## 近期工作包的四维状态

| 范围 | spec_status | implementation_status | verification_status | release_status |
|---|---|---|---|---|
| B02–B07/B09/B10/B12 的离线子集 | accepted ADR-0038/0039 | implemented bounded subset | 本地测试；CI 34163406749 / 34165668106 | experimental, not-released |
| B04/B05/B06/B07/B12 的本地 API readiness 子集 | accepted ADR-0040/0041 | implemented bounded subset | contract tests；CI 34168365247；真实 API 未执行 | experimental, not-released |
| AE-014/016/018/019/020 的存储/终态/只读诊断增量 | accepted ADR-0042/0043/0044 | implemented bounded subset | 22 个存储注入故障、五个子进程退出点、首因/取消/隐私/诊断测试；本次候选 CI 独立核验 | experimental, not-released |
| 核心/Scenario/Bundle/Journal 继承能力 | 以既有 accepted ADR 为准 | 以现有实现台账为准 | 历史证据；本轮未重跑 | 依既有 preview/stable 支持范围 |
| B08/B09-live 实际实验 | proposed live gate | readiness 不等于实际运行 | 需账户/model/数据/预算授权及真实报告 | 不可宣称兼容 |
| B10-external | proposed | not-started | 需真实独立使用反馈 | 不可宣称采用量 |
| 109 原需求/24 原任务逐项审计 | 原文未完整提供 | 不据此判断 | blocked-source | 不据此承诺 release |

实施 PR 必须关联 AE/MST26 条款（MST26 需有原文）、接受决策、实际源码/测试、候选 SHA 和
证据。planned test 不允许直接填成 passing test。若只完成离线子项，live/external gate 保持 open。
