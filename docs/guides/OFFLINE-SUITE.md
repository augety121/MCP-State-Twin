# 一次运行六任务离线回归

这套示例验证 mock 评测基础设施，不证明真实模型能力。无需 API key，不访问模型
服务。以下命令从仓库根目录运行，假设已构建的 `statetwin` 在 PATH 中；也可将
前缀换成 `go run -p 1 ./cmd/statetwin`。

## 准备和运行

```powershell
New-Item -ItemType Directory -Force examples/issue-tracker/.statetwin
statetwin bundle build --manifest examples/issue-tracker/bundle-agent.yaml --out examples/issue-tracker/.statetwin/agent-world.stb
statetwin eval suite-preflight --root examples/issue-tracker --suite agent-suite.json
statetwin eval suite --root examples/issue-tracker --suite agent-suite.json --out .statetwin/suite-01
```

已有 bundle 可直接复用。预检不创建输出或世界；预期 `plannedPairs: 6`、
`plannedTrials: 12`。执行按计划顺序，每对先 baseline 后 candidate，各自使用
独立世界。全部完成后预期 `executionStatus: completed`、`comparisonStatus: complete`、
比较决策 `no_regression_observed`。`upgradeAllowed` 始终 false。

目录包含 `claim.json`、生成的 `plan.json`、十二个 trial 子目录和成功发布的
`report.json`。后者先写入并同步临时文件，再通过不覆盖的硬链接发布。
只看到 stdout 中的报告对象不能证明文件已保存：务必同时检查命令退出状态。

## 重新核查证据

先对整套目录诊断，再执行严格门禁：

```powershell
statetwin eval suite-inspect --root examples/issue-tracker --out .statetwin/suite-01 --format markdown
statetwin eval suite-verify --root examples/issue-tracker --out .statetwin/suite-01
```

两条命令都只读，核查 claim、计划、报告、所有计划内 trial，以及重放生成的比较。
默认输出 JSON；两者均支持 `--format markdown`。没有报告或中断后也可 inspect，
但诊断成功不等于门禁通过。状态含义：

| 状态 | 含义 | inspect 退出码 | verify 退出码 |
|---|---|---|---|
| not_started | 目录尚不存在 | 0 | 非零 |
| incomplete_or_running | 无已发布报告，或仅有部分产物；不证明进程仍在运行 | 0 | 非零 |
| published_unverified | 已发布但未完成的执行，无法核验历史停止原因 | 0 | 非零 |
| published | 完整报告与重放证据一致 | 0 | 仅未观察到回归时为 0 |
| published_with_residue | 报告一致但仍有 staging 残留 | 0 | 非零 |
| invalid | 目录、元数据、报告或底层证据冲突/损坏 | 非零 | 非零 |

`reportVerification: matched` 不等于任务全部成功。合法回归报告同样可以 matched，
但不能通过 verify。门禁要求干净发布、报告一致和 `no_regression_observed`；
`upgradeAllowed` 始终 false。读取或取消错误不会被包装成成功诊断。

检查器先限制目录成员和证据尺寸，再重放；总 trial 产物尺寸上限 128 MiB，报告每份
512 KiB，claim/plan 每份 64 KiB，外层期限 120 秒。尺寸上限不是多次读取的总 IO
或进程内存配额。根目录应可信且停止并发写入，检查不保证原子快照或来源真实性。

也可以保留原来的单项复核方式：

```powershell
statetwin eval compare --root examples/issue-tracker/.statetwin/suite-01 --plan plan.json --format markdown
statetwin eval verify --root examples/issue-tracker/.statetwin/suite-01 --evidence baseline-01/terminal.json
```

compare 会核查计划内各 trial；单个 `report.json` 不是执行来源证明。需要 JSON
时将 `--format markdown` 改为 `--format json`。Task ID 汇总保留全部计划分母、
缺失证据和未评分数量。两组相同失败也可能没有回归，不等于任务成功。

## 制造候选回归

复制 `agent-suite.json` 为 `agent-suite-regression.json`，只把 `close-issue` 对的
`candidateResponses` 改成 `agent-mocks/omit-action.json`，然后执行：

```powershell
statetwin eval suite --root examples/issue-tracker --suite agent-suite-regression.json --out .statetwin/suite-regression-01
```

预期非零退出、比较结果 `regression`。任务失败保留完整证据并继续后面的任务，
这次仍应完成十二个 trial。执行、存储、预算或取消故障则停止启动新 trial；未启动项
保留在报告分母中。Ctrl-C 会请求取消；已经开始的清理和证据封存仍会尝试完成。

## 限制与故障处理

计划最多 16 对；`repeat` 是 1–16 的显式重复编号，不是自动展开次数。同一个
Task ID/repeat 不得重复。trial ID 按计划位置生成，例如 `baseline-02`，与 repeat
值无关。要重复同一任务，请增加不同 repeat 的计划项。

预检累计读取上限 64 MiB、累计 Bundle 解包内容上限 64 MiB，按引用计数；执行
累计证据内容写入上限 128 MiB（包括后来删除的 staging 内容），总执行期限 120 秒。
各文件和单 trial 的既有预算仍生效。这些限制不是进程内存硬配额。执行只使用预检
保留的私有字节；可信根目录必须避免并发写入，不保证敌对文件系统下的原子快照。

输出目录必须全新，父目录须已存在。冲突或中断后不覆盖、不续跑、不自动修复。
保留失败目录，使用 `eval inspect --root SUITE_DIR --out baseline-01` 等检查单个
trial，或用 `eval suite-inspect` 诊断整套目录；修复输入后选择新的 `--out` 再运行。
没有完整报告时也不要删除已有失败证据。检查器不会清理残留、提升 pending 文件或
修改已保存的结果；这不是 suite 恢复、live provider、并行执行或正式模型升级许可。
