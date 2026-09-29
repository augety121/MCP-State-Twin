# SPEC-0070: Isolated Task Case Runner

Status: **Proposal — not accepted or implemented**；依赖 [0069](SPEC-0069-TASK-CASE-MANIFEST.md)。

## 1. 拟议接口

`statetwin task cases --root ROOT --cases FILE --format json|markdown`

root 默认点、format 默认 json；cases 必填，拒绝未知参数和位置参数。只读来源文件，
stdout 报告，不保存质量结果到证据目录。使用现有 RunWitness 本地 MCP 路径；
不调用 RunMock、provider、外网，也不新增 Agent 可见控制工具。

## 2. 生命周期

全量准入后按 manifest 顺序串行运行，每 case 单独 provision/close，不共享变更后的
world、事件、答案或 oracle 反馈。所有读取与执行共享 120 秒 deadline；Task 自有
更短 deadline 仍生效。64 cases 是数量上限，不承诺任何任务组合都能在期限内完成。

实际评分与 expected 不同是 assertion mismatch：继续后续 case，以便一次看完问题。
环境、IO、cleanup、resource 错误或取消是 infrastructure failure：停止后续执行。
清理失败不得被 expected evaluator_error 吞掉；保留原始首因及有限 cleanup 状态。
共享 context 超时后不会为剩余 case 分配新的 120 秒。

预计的错误评分是测试成功，不是运行失败：只有 execution completed、cleanup complete、
真实 evaluator 结果存在且满足 0069 outcome/check 精确匹配时才标 matched。
对 unscorable-negative 额外要求至少一个 check 有 evaluation error；不能以普通 goal
false 冒充。不能从异常字符串推断 evaluator_error。

## 3. 报告契约

format `statetwin.dev/task-case-report/v1alpha1`，source scripted-witness-cases，
profile synthetic-witness-cases-v1。必填字段：format/source/profile/decision、
planned/matched/mismatched/failed/notStarted、cases、provenance。
decision 为 matched/mismatched/incomplete，provenance 为 not-proven。

cases 每行含 caseId/taskId/role、state（matched/mismatched/failed/not_started）、
expectedOutcome、actualOutcome（无结果为空串）、failedCheckIds、errorCheckIds、
executionStatus、cleanupStatus、failureCode；无信息用空串/数组，不伪造 completed。
所有计划行从开始就占位；总数守恒。任何 failed/not_started 令 decision incomplete，
否则任意 mismatch 为 mismatched，否则 matched。计数是用例符合预期数，不是 Agent
任务成功率，禁止把反例的 policy_violation 算成模型改善。

隐藏 View、answer、输入、原始错误以及 Task 表达式，只输出 admitted ID 和有限状态。
报告最多 1 MiB；超限返回 CASE_RESOURCE_LIMIT，不能截断后报 matched。
结构化非匹配先打印，再返回 CASE_GATE_NOT_SATISFIED；取消/资源/无报告错误明确
失败，stdout 错误优先。纯 stdout 路径无输出目录恢复或重试选项。

## 4. 验收与实施约束

- 正例与故意漏做、错对象、额外效果、无权限尝试、oracle 运行错误一起运行，精确计数。
- 第一 case 修改状态，下一 case 仍看到原始 fixture；顺序互换不改变各自语义结果。
- mismatch 后继续；第 N 项 cleanup/IO 失败后保留 N+1..end 的 not_started。
- cancel/timeout 覆盖预检、回合和清理边界；晚到结果不能把 incomplete 改为 matched。
- 两种输出、stdout 故障、无网络、输入字节不变，临时 world 在成功/失败/取消均清理。
- 与旧 `task witness` 同输入的 Evaluation 一致，旧 Report/CLI 行为不变。

实现建议 `internal/agenteval/case_run.go`、CLI `task cases` 分派与指南。
测试必须同时有真实本地 MCP witness 和可控失败注入，不只手工构造评分报告。
