# SPEC-0067: Offline Suite Review Before Execution

Status: **Proposal — not accepted or implemented**，依赖
[SPEC-0066](SPEC-0066-INDEPENDENT-TASK-CATALOG.md) 和
[ADR-0066](ADR-0066-REVIEWED-TASK-BINDING-PROPOSAL.md)。下述命令当前不存在。

## 1. 用户结果

现有 `suite-preflight` 证明输入能被静态接受，不能发现它与操作者选定的任务不同。
提议在运行前一次性审阅：执行计划、output-token 上限以及每个 repeat 的完整 Task
是否匹配外部输入。审阅通过只表示此刻冻结输入的静态匹配，不表示模型能完成任务。

拟议语法：

```text
statetwin eval suite-review --root ROOT --suite SUITE --out OUT --expect EXPECT --tasks CATALOG --format json|markdown
```

root 默认 `.`，format 默认 json；其余参数必填，不接收位置参数或未知格式。
out 仅用于检查独立输入与计划输出的路径边界，**不枚举、创建或清理输出目录**，
也不承诺它在之后执行时仍不存在。现有 `eval suite` 保持自己的 no-clobber 检查。

## 2. 流程与零执行边界

1. 校验参数和 portable paths，建立覆盖全流程的 120 秒 context，支持中断。
2. 先冻结 SPEC-0063 expectation 和 SPEC-0066 目录；校验所有外部输入在 out 外。
3. 读取/解码 suite；核对目录引用路径不等于任一 suite Task 路径（大小写折叠）。
   调用共享 `PrepareSuite` 路径，保留 Task、bundle、mock 响应的全量准入和累计预算。
   显式引用的 suite 输入仍会读取；out 参数本身不触发目录状态探测。
4. 比较 prepared ComparePlan 与独立 expectation 的完整有序计划及输出预算。
   对每对已冻结 Task 与目录期望做完整相等判断；不得使用重新打开的 Task 文件。
5. 输出审阅结果；任一结构化不匹配返回非零。输入无效、取消、资源失败可不输出结果。

不执行 mock 回合、不创建世界运行实例、不生成 trace/claim/report，也不调用 provider。
复用 Preflight 的解析、bundle 验证、工具/oracle 静态检查，不调用 `RunSuite` 或 `RunMock`。
不得为了预检复制任务到临时根、建立持久数据库或制造一套成功终态证据。
suite 的 64 MiB 输入/提取预算与目录预算分别生效；输出上限 1 MiB，超限返回
`SUITE_REVIEW_RESOURCE_LIMIT`，不输出截断成功。deadline 到期不得再输出 passed。

## 3. 结果格式

提议 format `statetwin.dev/agent-suite-review/v1alpha1`，profile
`offline-task-review-v1`，顶层字段全部存在：

| 字段 | 规则 |
|---|---|
| format / profile | 上述固定值 |
| decision | `matched` 或 `mismatched`；不用 passed/success 描述执行结果 |
| planStatus | `matched` 或 `mismatched`，含有序计划与 maxOutputTokens |
| catalogStatus | `matched` 或 `mismatched`，表示 Task ID 集合覆盖 |
| plannedPairs / checkedPairs | expectation 分母；仅身份相同的 pair 可计入 checked |
| tasks | 按 expectation pair 顺序的全部诊断行 |
| extraTaskIds | 目录中多余 ID，按字典序；无则空数组 |
| reasons | 有限、有序、去重的 code 对象，可附 taskId/repeat |
| executionPerformed | 始终 false |
| provenance | 始终 `not-proven` |

每行字段：taskId、repeat、status（matched/mismatched/unverifiable）、differences
（SPEC-0066 类别）。suite 计划不匹配时不进行样本重映射：checkedPairs 为 0，所有
expected 行为 unverifiable。计划匹配且目录缺该 Task 时，该行也为 unverifiable。
有合法参考且 Task 已比较即计 checked，内容不同仍计入；缺失不能缩减分母。
prepared 中单 pair Task 同时用于两侧，静态报告采用 pair 分母，不虚称两个已执行 trial。

reasons 依次为 `plan_mismatch`、`output_budget_mismatch`、
`catalog_coverage_mismatch`，再按 expected pair 顺序输出 `task_mismatch` 或
`task_unverifiable`。仅预算不匹配但计划身份相等时，仍允许核对并计入 Task；planStatus
仍为 mismatched。全部匹配才得到 decision matched；不提供无条件通过的空输入路径。

Markdown 和 JSON 传达同样的分母、状态与非执行声明，不显示原始 Task 内容。
stdout 失败优先于 matched；结构化不匹配输出后返回 `SUITE_REVIEW_NOT_MATCHED`。
参数无效返回 `SUITE_REVIEW_ARGUMENTS_INVALID`。其他有限输入错误沿用其所属模块；
禁止把错误降为“未发现问题”。

## 4. 时间与授权边界

审阅输出不作为执行许可或可加载缓存：两个 CLI 进程之间输入可能改变，suite 仍需
正常冻结输入，运行后仍需 SPEC-0068 重新核验。不得把 review matched 接成自动执行。
本阶段不改 `suite`、`suite-preflight`、`suite-assess` 的默认行为。

共享 prepared 内部数据时只能加只读内部访问/观察点，不能导出可变 bytes 或把目录
Task 替换进实际计划。发现差异必须报告，不能“修复”目标、oracle、预算或 mock 响应。

## 5. 后续实施验收

- 使用六任务计划证明 expectation、目录及不同 task 文件路径的合法副本能匹配；
  用两次 close-issue repeat，其中第二次修改 oracle/revision，证明在零执行下拒绝。
- dropped/reordered/extra pair、模型、repeat、预算变化分别有测试；预算单独不匹配
  仍检查 Task，身份不匹配不错误配对。目录缺项/多项保持 expected 分母。
- 无效 mock 脚本、损坏 bundle、oracle 静态不兼容不能被“任务匹配”掩盖。
- 以可观测 fake/spy 证明没有 RunMock、provider、输出写入调用；检查前后输入字节，
  不使用文件哈希；out 不存在和已有无关文件两种情况都不得变化。
- 冻结后替换磁盘 Task，审阅仍使用同一 prepared 输入；下一次新调用读取新内容。
- CLI 两种格式、缺必填参数、未知参数、取消、超限、stdout 失败与有限隐私错误。
- 旧 `suite-preflight` 输出与行为不变。适量真实 CLI 测试覆盖链路，矩阵纯比较使用
  单元测试；不为每个文本排版样本重复运行昂贵的真实回放。

建议落点为 `internal/agenteval/suite_review.go`、`cmd/statetwin/agent_review.go` 及
测试。实施必须执行 gofmt、vet、必要测试、race 和对应提交的 CI；本 Spec 只定义验收。
