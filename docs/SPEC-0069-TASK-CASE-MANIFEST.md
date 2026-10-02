# SPEC-0069: Task Case Manifest

Status: **Accepted — implemented in the bounded offline subset**；[ADR-0069](ADR-0069-TASK-QUALITY-PROPOSAL.md)。

## 1. 输入及用途

新输入把 Task、已有 TaskWitness 和预期评分关联起来。它是 authoring test，
不属于 AgentEvidence，不改变 Task oracle，也不支持任意执行脚本。
严格 JSON 单对象，256 KiB、深度 32，拒绝未知/重复/大小写别名字段、null 与多文档。

```json
{
  "format": "statetwin.dev/task-cases/v1alpha1",
  "profile": "synthetic-witness-cases-v1",
  "tasks": [{"taskId": "close-issue", "task": "agent-tasks/close-issue.json"}],
  "cases": [{
    "caseId": "close-positive", "taskId": "close-issue",
    "role": "positive", "witness": "agent-witnesses/close-issue.json",
    "expected": {"outcome": "success", "failedChecks": []}
  }]
}
```

上例仅演示语法，不满足 0071 的完整质量门槛。全部展示字段必填；tasks 为 1–16 项，
cases 为 1–64 项。caseId/taskId 使用 Task ID 规则，分别唯一；case.taskId 必须引用
目录内 Task，每个 Task 至少有一个 case。单 Task 在整个 manifest 中只能有一份定义。
所有路径相对显式 trusted root，portable、常规文件、可见 symlink 拒绝；无远程引用。

role 闭集：positive、goal-negative、policy-negative、unscorable-negative。
expected.outcome 闭集：success、expected_abstention、task_failed、policy_violation、
evaluator_error；不允许 not_evaluated 充当合格用例。positive 的 outcome 必须等于
Task.expectedOutcome，其 failedChecks 为空；goal-negative 必须 task_failed；
policy-negative 必须 policy_violation；unscorable-negative 必须 evaluator_error。

failedChecks 是该 Task oracle 中不同的 assertion ID 集合，至多 64 个；语义为
实际 Checks 中 Passed=false 的**精确集合**，包含运行错误导致的 false。
goal-negative 至少列一个 goal check；policy-negative 可以为空，因为未授权尝试
可独立触发 policy_violation；unscorable-negative 至少列一个产生错误的 check。
不接受“任意失败算命中”、文本正则或动态 CEL 作为预期匹配器。

## 2. 全量准入与冻结

执行前读取所有 Task、bundle、witness，先用现有 Decode/Admit/DecodeWitness 核验，
检查 witness.TaskID、调用数量和隐私策略。Task/witness 各 256 KiB；bundle 沿用
32 MiB 压缩、64 MiB 提取及成员限制。总原始输入 64 MiB、总提取 64 MiB，
读取相同引用可缓存冻结字节一次，缓存键只用规范路径；不同路径不做内容去重。
所有新 manifest 整数必须原始整数；既有 Task/Witness 兼容规则不在此偷偷升级。

冻结表示本次不再重读，不证明跨文件原子快照。所有角色、ID、expected set 和目录
覆盖先检查完才执行第一条；最后一个坏用例也必须使零用例启动。不让 decoder 的
底层错误泄漏值或路径；新边界使用 CASE_MANIFEST_INVALID、CASE_INPUT_INVALID、
CASE_RESOURCE_LIMIT、DATA_POLICY_REJECTED 及 context 原因。

## 3. 比较与验收

case 输出顺序必须等于输入顺序；failedChecks 比较集合，诊断顺序按 Task oracle。
被测评分始终使用实际 Task，不以 expected 改写评分器。

必测：64/65 cases、16/17 tasks、非法 role/outcome 组合、重复/不存在 assertion、
缺任务/多任务、空 positive 反例、Task ID 错配、最后一项坏文件、预算上限前后、
目录穿越/symlink/nonregular、转义敏感哨兵、取消和冻结后磁盘修改。
正例覆盖同 Task 多条不同合法 witness、expected_abstention、策略尝试无需失败
assertion 的情况。测试输入均合成，不引入生产 trace。

建议 `internal/agenteval/case_plan.go`；解析/冻结对象不导出可变原始 bytes。
交付时需把上述规则映射到可执行测试，不能把示例解析通过当作 runner 已实现。
