# SPEC-0071: Declared Oracle Case Coverage

Status: **Proposal — not accepted or implemented**；依赖 [0070](SPEC-0070-TASK-CASE-RUNNER.md)。

## 1. 门槛目的

用例全符合预期仍可能只有正例。新命令 `statetwin task qualify --root ROOT --cases FILE
--format json|markdown` 在同一次 0070 执行中计算覆盖，不读取外部自报 report，
也不靠扫描文件名推断反例。全程沿用 0070 时间、IO 和无 provider 边界。

固定 profile `oracle-cases-v1`，无百分比阈值、忽略列表或随意 waiver。
每个 Task 必须同时满足：

1. 至少一个 matched positive，actualOutcome 为该 Task 期待的成功/预期不操作。
2. 每个 goal assertion 至少有一个 matched goal-negative，该 assertion false 且
   无 evaluation error，证明这条检查实际拒绝了一次已声明反例。
3. 每个 policy assertion 至少有一个 matched policy-negative，该 assertion false
   且无 evaluation error。仅有未授权尝试计数不能替代逐 policy assertion 覆盖。
4. 整个 case run decision matched；任意失败、缺失、未运行或额外 mismatch 均失败。

一个反例可以覆盖多个 assertion，但不证明这些 predicate 相互独立或均有必要；
evaluator_error 不贡献拒绝覆盖。unscorable-negative 可测试错误处理，不能补齐
goal/policy coverage。某任务无法产生满足条件的反例时，输出 uncovered，不降低门槛。

## 2. 新封套

format `statetwin.dev/task-qualification/v1alpha1`，profile oracle-cases-v1，
decision qualified/not_qualified；fields 包括 format/profile/decision、caseReport、
tasks、reasons、provenance:not-proven。旧 CaseReport 不增字段。
tasks 按 manifest tasks 顺序，每行 taskId、positiveCaseIds、checks；checks 按 oracle
顺序，每项 id/category、status covered/uncovered、matchedNegativeCaseIds。
caseIds 按 manifest cases 顺序去重，不能排除计划中失败样本再计算一个“干净子集”。

reasons 先 case_run_not_matched，再逐 Task 的 positive_missing 和逐 assertion 的
negative_coverage_missing；附 safe taskId/checkId，不渲染表达式。qualified 当且仅当
上述门槛全部成立。原始负例轨迹不进入摘要。总输出 1 MiB，stdout/取消失败规则同 0070。

## 3. 解释与兼容

qualified 只表示本版本声明的合成正反用例满足门槛，不证明 oracle 没有漏洞、
独立人工批准、测试完备性或模型能力。正反 witness 都可能由同一作者写错。
不将此字段混入 SuiteAssessment 或视为运行授权；0068 仍需独立 Task 绑定。
未来若要把两类门禁组合，必须有单独接受的契约，而不是解析这份报告自动升级。

## 4. 验收

必须覆盖只有正例、只测一个 goal 的多 goal Task、遗漏 policy assertion、
“constant true”导致反例无法击败 oracle、“constant false”导致正例失败、
错误评分冒充负例、policy attempt 有结果但无 assertion 覆盖，以及同一反例覆盖多项。
六任务用例集以真实运行证明 qualified；删一个必要反例则 not_qualified。
用 spy 证明 qualify 未重复运行 cases；旧 cases 输出不变。
纯覆盖聚合用单元测试，至少一个完整正例和一个未覆盖反例有 CLI 验收。

实现建议在 `internal/agenteval/case_coverage.go` 保持纯聚合，仅消费本次冻结 Task
和 runner 内部结果，不公开“信任传入报告”的产品入口。
