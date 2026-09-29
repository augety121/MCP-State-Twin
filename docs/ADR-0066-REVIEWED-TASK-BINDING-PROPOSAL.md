# ADR-0066: Independently Supplied Task Binding

- Status: **Proposal — not accepted or implemented**
- Date: 2026-09-29
- Contracts: [SPEC-0066](SPEC-0066-INDEPENDENT-TASK-CATALOG.md),
  [SPEC-0067](SPEC-0067-OFFLINE-SUITE-REVIEW.md),
  [SPEC-0068](SPEC-0068-REVIEWED-TASK-ASSESSMENT.md)
- Baseline: PR #12 merged as `ae9968f`; ADR-0063 remains the current authority.

## 问题与代码证据

当前 `assessment.go` 的预期只绑定 ComparePlan 和 output-token 预算；
`assessmentCollector.observe` 比较同组 RunDefinition，却没有独立的期望 Task。
因此，把所有重复试验一致地换成更宽松的 oracle，仍可能同时满足计划匹配、
定义一致和候选通过。这是 SPEC-0063 已声明的边界，不是现有门禁实现失效。

`suite_plan.go` 已能冻结 Task、bundle 和 mock 响应并执行静态 preflight，
但 `suite-preflight` 没有把这些输入与外部期望比较。`compareObserved` 已提供
只接收重放核验定义的内部观察点，可以补充绑定而不重新读取每条证据。

## 提议

引入一个由操作者独立维护的任务目录，引用完整 AgentTask 快照；运行前把计划输入
与目录比较，运行后把重放核验的 Task 与同一目录比较。目录不从结果目录生成。

1. SPEC-0066 定义目录准入、冻结、完整 Task 相等规则和资源边界。
2. SPEC-0067 提议只读 `eval suite-review`，在执行之前发现计划或任务偏差。
3. SPEC-0068 提议只读 `eval suite-assess-reviewed`，同时要求现有绝对达标门禁
   和独立任务绑定通过；使用独立输出格式，不扩展保存的 Comparison。

“reviewed”描述调用方提供的审阅输入，不代表程序验证过审阅人的身份或判断质量。
统一使用 `taskBinding: matched`，不用 `oracleApproved` 或 `trusted`。
现有 `suite-assess` 不接受新参数、不升级策略、不改变退出码或序列化。

## 取舍与不纳入的范围

- 相比只锁定 revision，完整 Task 比较能发现不改版本号却改 oracle 的情况。
- 相比把 Task 塞入 64 KiB 的 expectation，独立目录可复用现有 256 KiB Task
  上限，并保持 SPEC-0063 文件兼容。
- 相比在 assessment JSON 上做后处理，内部观察点只使用真正重放核验的 Task。
- 不把所有 Task 字段加入忽略列表；只允许 JSON 对象键顺序和排版差异。
- 不增加文件哈希、签名、时间戳证明、审批工作流、执行脚本或自动接受工具。
  既有业务 digest 逻辑保持原样。
- Task 中的 bundle 路径参与比较，但目录不独立绑定 bundle 内容、runtime 或模型
  来源；现有重放和跨重复一致性检查继续承担自己的范围。不能将新结果描述为
  “整个测试世界已获独立批准”。
- 不调用 provider，不支持 live suite、resume、自动升级或生产写入。

## 实施顺序与退出条件

后续明确授权实施后，先实现 0066 准入与纯比较，再接入 0067 静态审阅，最后接入
0068 单次重放和 CLI。三份 Spec 的验收矩阵都是交付范围，不能用目录解析通过代替
真实回放反例。具体证据目标见各 Spec；实施时再接受本 ADR 并更新实现台账。

文档合并不构成实施授权，也不使三个拟议命令/格式成为当前能力。此次只交付设计。
