# SPEC-0049: 离线比较报告与分母核算

Status: accepted by [ADR-0049](ADR-0049-AUDITABLE-COMPARISON-REPORTS.md).

保留 ComparePlan v1alpha1、最多 32 pairs、每 pair repeat 1–16、唯一 trial/path 的规则。
不改变既有报告 format 或删除字段；新增字段为实验性扩展，严格旧 reader 应显式升级：

| 字段 | 含义 |
|---|---|
| `decisionPolicy` | 精确值 `offline-regression-v2` |
| `verificationProfile` | 独立资源版本 `offline-compare-v1`，不是世界状态或评分器版本 |
| `baselineModel` / `candidateModel` | 已验证的 mock 标签，不是真实模型 ID |
| `planBinding` | `validated-input-not-preregistration-proof` |
| `baselineCounts` / `candidateCounts` | 各侧 planned/notStarted/started/incomplete/terminal/validlyEvaluated |
| pair `reasons` | 固定顺序的有限原因码，无原始错误或文件路径 |
| pair `newPolicyFailures` | candidate 新失败的 verified policy ID，保持 oracle 顺序 |
| trial `blockedAttempts` / `committedViolations` / `failedPolicyChecks` | 仅重放成功后派生的风险事实 |

原 combined `counts` 每一字段都必须等于两侧之和。每侧满足
`planned = notStarted + started`、`started = incomplete + terminal`、
`0 <= validlyEvaluated <= terminal`。无效、缺失、未评分与完整失败都不得从计划删除。
`validlyEvaluated` 是证据和评分资格，不是 successful count；JSON/Markdown 都要表达这点。

Markdown 必须展示 model 标签、决策规则、双方分母、每条计划的 trial ID、validation、
outcome 和原因。只展示安全标识符和枚举，不输出 artifact 路径、Task 正文、表达式或密钥。
相同已冻结输入、相同运行时、未超出预算时，报告字节保持确定性；无 wall-clock 字段。

保留 `upgradeAllowed:false` 和 `mock-no-provider-call`。不得从一对 mock 推导真实价格、
实际模型回归、显著性、真实预注册时间或“双方都失败所以系统可用”。原工件保持不变。

验收：混合多 pair 决策优先级、两侧与总分母等式、确定性 JSON/Markdown、模型/ID 与
原因可定位、敏感路径不回显、重复 trial 拒绝、现有 CLI 不满足门禁仍返回非零。
