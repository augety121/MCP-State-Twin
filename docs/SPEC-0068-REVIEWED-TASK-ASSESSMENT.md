# SPEC-0068: Replay-backed Reviewed Task Assessment

Status: **Proposal — not accepted or implemented**，依赖
[SPEC-0066](SPEC-0066-INDEPENDENT-TASK-CATALOG.md)；设计决策为
[ADR-0066](ADR-0066-REVIEWED-TASK-BINDING-PROPOSAL.md)。当前 `suite-assess` 不具有本能力。

## 1. 可复现缺口与目标

给同一 Task ID 的所有 baseline/candidate/repeat 一起替换 oracle，例如把目标条件
改得更宽松，现有 expectation 身份和 repeat 一致性仍可匹配。SPEC-0065 正确地按
实际保存的 Task 评分，却无法知道它与操作者独立选定的内容不同。

提议新命令，必须同时满足现有绝对达标门禁和完整 Task 的外部绑定：

```text
statetwin eval suite-assess-reviewed --root ROOT --out OUT --expect EXPECT --tasks CATALOG --policy candidate-pass-v1|both-pass-v1 --format json|markdown
```

root 默认 `.`，format 默认 json；out/expect/tasks/policy 必填。不接受旧 assessment
JSON 作为已核验缓存，不自动从结果内寻找目录，也不从 SPEC-0067 报告推断通过。
两种 policy 只影响结果要求；无论 policy 如何，**两侧全部计划 trial 都必须绑定 Task**。

## 2. 一次回放的数据流

1. 参数和外部输入先准入；目录文件及全部引用必须位于 out 子树外。
2. 一个 120 秒 context 包含 expectation、目录读取、suite audit、回放和报告生成。
3. 使用 `inspectSuiteObserved` / `compareObserved` 既有内部观察链路，一次核验同时
   喂给 SPEC-0064 consistency collector 和新 Task binding collector。
   不先公开调用 AssessSuite 再第二遍重放；不改变公开 Compare/InspectSuite 输出。
4. 仅通过实际重放且身份正确的 `RunDefinition.Task` 可贡献绑定。valid 和
   not_evaluated 均可证明 Task 内容；后者依旧不能通过绝对结果门禁。
   partial/invalid/缺失/身份错配不贡献定义，声明计数或 saved report 不替代证据。
5. 完整 Task 与冻结目录按 SPEC-0066 比较，不重评分、不注入 reviewed oracle，
   不把实际结果改成“若用了目录会成功”的反事实结果。

复用现有 suite/evidence IO 预算和 4 MiB consistency reference；独立目录采用
SPEC-0066 的预算。binding collector 只保留每个 trial 的有限诊断，不留原始 trace
或第二套实际 Task 副本；输出最多 1 MiB，超限 `REVIEWED_ASSESSMENT_RESOURCE_LIMIT`。
所有新取消/资源错误必须穿过内部 audit 层，不能被转成可继续的普通无效样本。

## 3. 绑定语义与分母

taskBinding.status 为 matched/mismatched/unverifiable。字段包括：

- plannedTrials：始终为独立 expectation 的 pairs 数乘 2。
- verifiedTrials：计划身份完全匹配时，拥有重放核验 Task 的 expected trial 数。
- matchedTrials / mismatchedTrials / unavailableTrials：按 expected 样本逐一分类。
- missingTaskIds / extraTaskIds：目录集合差异；前者按 expected 首现顺序，后者字典序。
- trials：expected plan 顺序、每对 baseline 在前，每行 taskId/repeat/trialId/status/
  differences；status 为 matched/mismatched/unverifiable，differences 为有限类别数组。

计数恒等式为 matchedTrials + mismatchedTrials + unavailableTrials = plannedTrials；
verifiedTrials 单列，不与 unavailable 简单互补：重放已验证但目录缺项的 trial 仍
不能完成绑定。verifiedTrials 不证明结果通过，也不证明所验证 Task 符合目录。

当 saved plan 与 expectation 身份不一致时，全部 expected trial 绑定为 unverifiable，
verifiedTrials 为 0，不按相似 ID 重新配对。预算不一致但身份相同不阻止 Task 比较，
旧 assessment 仍因预算不匹配而失败。计划损坏/不可用也保留 expected 分母。

已验证 Task 与对应合法目录不同，该行 mismatched；缺失证据或参考，该行 unverifiable。
有任何确定内容差异或目录集合差异，整体 mismatched；否则有任一不可核验行则
unverifiable；只有覆盖精确且每个 expected trial 完成绑定才 matched。
已知差异遇到另一缺失样本不能降为“只有证据不足”。

## 4. 独立结果封套与门禁

提议 format `statetwin.dev/agent-reviewed-assessment/v1alpha1`，profile
`offline-reviewed-task-v1`，必填字段：format、profile、policy、decision、reasons、
taskBinding、assessment、upgradeAllowed、provenance。

assessment 嵌入与 SPEC-0065 相同的 SuiteAssessment；其旧格式、reason 顺序、
audit、decision 不因目录结果变化。decision 只有 passed/failed，公式为：

```text
passed = assessment.decision == passed AND taskBinding.status == matched
```

顶层 reasons 按顺序包含 `base_assessment_failed`（一次，细节在旧封套）、
`catalog_coverage_mismatch`（一次），然后按 expected trial 顺序附带 taskId/trialId
输出 `reviewed_task_mismatch` 或 `reviewed_task_unverifiable`。空列表序列化为 []。
upgradeAllowed 始终 false，provenance 始终 not-proven；禁止命名为“审批完成”。

JSON/Markdown 展示绑定覆盖与原有门禁，两者不能相互遮蔽。仅显示有限标签/类别，
不打印目录路径、Task 文本或底层异常。结构化失败先输出诊断，再返回
`REVIEWED_ASSESSMENT_GATE_NOT_SATISFIED`；参数错误为
`REVIEWED_ASSESSMENT_ARGUMENTS_INVALID`。stdout 错误优先；取消/准入/资源错误
允许无封套返回，绝不能输出虚构通过结果。

## 5. 兼容与解释

不修改 SuiteReport、Comparison、SuiteInspection 或 SuiteAssessment 的序列化，
也不把新字段写回旧证据目录。现有 CLI 参数、默认策略要求及退出结果保持不变。
新命令本身只读，无 provider、恢复、清理或自动升级行为。

Task.Bundle 字符串匹配不等于对 bundle 内容的独立期待；bundle 重放正确性及跨样本
一致性仍由原有机制处理。即使顶层 passed，也不能宣称世界内容、runtime、模型身份
获得独立审批、样本统计独立或 oracle 质量正确。目录是可信根下的输入声明，不是证书。

## 6. 后续实施验收矩阵

| 反例或正例 | 旧 assessment | 新封套 |
|---|---|---|
| 六任务、独立 Task 副本、完整证据 | passed | passed |
| 所有重复一致替换 oracle，revision 保持不变，实际结果仍通过 | passed | failed，Task mismatch |
| 一致替换 goal/authority/Task budget 且实际通过 | 允许 passed | failed，对应差异类别 |
| 只换一个 repeat 定义 | failed，heterogeneous | failed，保留旧原因和绑定诊断 |
| 基线失败、候选通过且 Task 全部匹配 | 按两种 policy 分别判断 | 完全沿用两种结果 |
| 双方都失败但没有相对回归 | failed | failed，不被 Task matched 掩盖 |
| replay-valid、unscorable、Task 匹配 | failed | Task 可 matched，顶层 failed |
| 缺首/中/尾/全部 trial，或 partial/无效 | failed | 保留全部分母、不可核验行 |
| 已证 Task mismatch 再缺一个 trial | failed | binding mismatched，保留 unavailable 数 |
| 目录少项/多项、计划替换/不可读 | failed 或原本 passed | failed，不重映射样本 |
| matched Task 但被改 bundle 内容 | 由既有回放/一致性决定 | 不虚称独立 world 绑定 |

还须覆盖最大 16 对、interleaved Task、重复诊断稳定排序、两种格式/策略、先准入再读
证据、取消/资源错误穿透、stdout 失败、输入字节不变和旧封套相等。用 observer/spy
证明每个 admitted trial 只核验一次；纯格式/排序矩阵用单元测试，关键反例保留真实
回放与 CLI 集成测试，不只构造最终 report JSON。

建议落点 `internal/agenteval/reviewed_assessment.go` 和
`cmd/statetwin/agent_assess_reviewed.go`。后续交付必须包含正例及 coherent Task
替换负例的可运行指南，运行 gofmt、vet、tests、race 与对应提交 CI 后才能标为实现。
