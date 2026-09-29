# 离线候选达标验收

`suite-assess` 在现有重放审计之上，检查独立预期计划、重复试验定义和候选结果。
本指南使用纯合成 mock，无 API key 或网络模型调用。从仓库根目录执行；需要 PATH
中的 `statetwin`，也可将前缀替换为 `go run -p 1 ./cmd/statetwin`。

## 六任务正例

仓库的 `agent-expectation.json` 是单独维护的输入，包含两组模型标签、六对任务、
重复编号、trial ID 和每次 1024 output-token 上限。审阅它再运行，不要从运行后的
`plan.json` 自动生成预期文件，否则不能发现计划替换。

```powershell
New-Item -ItemType Directory -Force examples/issue-tracker/.statetwin
statetwin bundle build --manifest examples/issue-tracker/bundle-agent.yaml --out examples/issue-tracker/.statetwin/agent-world.stb
statetwin eval suite --root examples/issue-tracker --suite agent-suite.json --out .statetwin/assessment-01
statetwin eval suite-assess --root examples/issue-tracker --out .statetwin/assessment-01 --expect agent-expectation.json --policy candidate-pass-v1 --format markdown
statetwin eval suite-assess --root examples/issue-tracker --out .statetwin/assessment-01 --expect agent-expectation.json --policy both-pass-v1
```

已有 bundle 可复用，已有输出目录不能覆盖。预期两种策略均 `passed`，12 个定义经
重放核查，六组均为 `single_pair`。这不是六组重复实验，也不证明真实模型能力。

| 策略 | 所需结果 |
|---|---|
| candidate-pass-v1 | 所有计划内候选均已评分且 success 或 expected_abstention |
| both-pass-v1 | 所有计划内基线和候选均满足同样要求 |

两者都要求干净发布、报告一致、独立预期匹配、定义一致，以及未观察到回归。
合法的预期不操作由原有评分器确认，不是把任意拒绝算作通过。双方都失败时，旧比较
仍可能没有相对回归，但这两种达标策略都失败。默认 JSON；Markdown 只改变展示。

## 重复定义冲突负例

下面的样例运行同一 close-issue 的两个 repeat：第二次使用 revision v2，其他行为
相同。每对基线/候选内部一致，跨重复的定义却不同。这是故意构造的负例。

```powershell
statetwin eval suite --root examples/issue-tracker --suite agent-suite-repeat-negative.json --out .statetwin/repeat-negative-01
statetwin eval suite-verify --root examples/issue-tracker --out .statetwin/repeat-negative-01
statetwin eval suite-assess --root examples/issue-tracker --out .statetwin/repeat-negative-01 --expect agent-repeat-expectation.json --policy candidate-pass-v1 --format markdown
```

前两条预期成功；最后一条**预期非零退出**，decision failed，定义组 heterogeneous，
原因 definitions_heterogeneous。不应修改报告来消除这个结果。修正真实计划/任务后，
在新目录重新运行，并重新审阅独立预期。

## 如何阅读失败和边界

`expectation` 区分 matched、mismatched、unverifiable，保留预期分母与已验证数量。
预算不匹配仍可以有完整验证覆盖；验证过不等于符合预期。计划不匹配时，不把另一份
计划下的证据算成预期 trial 的已验证样本。

`consistency` 保留全部已知计划样本；缺失定义不可推断相等。已发现差异又有缺失时，
identityStatus 仍为 heterogeneous，coverage 为 incomplete。原始 oracle、Task 和
trace 不写入 assessment。规范化只去掉模型标签和 trial ID，其他定义字段都保留。

命令只读，不恢复、不修复、不清理残留。预期文件必须在输出子树之外，根目录应可信
且停止并发写入。64 KiB 预期输入、4 MiB 累计保留定义、120 秒协作期限以及原有
审计资源限制共同生效；不是原子文件系统快照或硬 RSS 配额。中断/读取错误明确失败。

`upgradeAllowed` 始终 false，provenance 为 not-proven。预期只绑定计划标识和预算，
不会独立批准 oracle；同定义也不证明独立随机样本、统计显著性或真实模型来源。
退出非零时，若已经生成诊断会先打印；stdout 写失败同样失败，不会被通过结果掩盖。
