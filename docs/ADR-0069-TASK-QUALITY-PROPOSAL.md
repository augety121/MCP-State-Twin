# ADR-0069: Executable Task Quality Cases

- Status: **Proposal — not accepted or implemented**
- Date: 2026-09-29
- Scope: B02/B03/B14 的有限合成任务质量子集；不关闭整个工作包。
- Contracts: [0069](SPEC-0069-TASK-CASE-MANIFEST.md)、[0070](SPEC-0070-TASK-CASE-RUNNER.md)、
  [0071](SPEC-0071-ORACLE-CASE-COVERAGE.md)、[0072](SPEC-0072-PACKAGE-REGISTRY-TASK-KIT.md)。

## 调查与选择

`internal/agenteval/witness.go` 已提供受限 scripted witness，经真实本地 MCP 路径执行、
收集状态并评分。`witness_test.go` 已验证 issue-tracker 六任务及部分错误轨迹。
`internal/evaluator/evaluator.go` 已提供逐 assertion 结果和策略尝试计数。
缺口是作者不能用一个版本化输入表达“这些正例应通过、这些反例应被 oracle 捕获”，
也缺少拒绝缺少反例的任务集准入门槛。当前测试存在不等于已经提供这套产品接口。

选择显式用例目录，而非自动生成任务或让模型判断 oracle 好坏。沿用 witness，
每个用例新世界，完整保留未执行样本和预期失败。独立任务绑定 0066–0068 解决
“是否用了选定的 Task”，本组解决“选定的 Task 是否通过声明的正反用例”。两者互补，
不能用用例通过证明目录来源可信，也不能用目录匹配证明 oracle 有效。

第二领域采用仓库已有 package-registry 的六个工具，不修改其业务规则来迎合测试。
补任务、witness、mock 与新质量用例；不声称 npm/PyPI 真实等价、L2 或 live 验证。

## 实施与退出

先做 0069 严格准入，接 0070 执行，再做 0071 同次运行覆盖计算；0072 使用三者
交付具体领域资产。0069–0071 不依赖尚未实现的 0066，接受时可单独排期。
禁止通过读取别人保存的质量报告来跳过本次执行；不改变旧 witness 格式和退出行为。

执行代码、正反真实测试、CLI、指南与对应提交 CI 都完成后才更新实现台账。
当前仅提案，无执行授权，无代码变更。
