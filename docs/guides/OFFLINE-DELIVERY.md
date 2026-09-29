# 完整离线任务质量与证据交付

本指南使用 package-registry 的纯合成数据。没有 API key、provider 请求或生产服务。
从仓库根目录执行；`statetwin` 可替换为 `go run -p 1 ./cmd/statetwin`。
所有输出使用新路径；已有 bundle 可复用，已有结果、归档或 pending 不会覆盖或恢复。

## 1. 构建世界并检查任务质量

```powershell
New-Item -ItemType Directory -Force examples/package-registry/.statetwin
statetwin bundle build --manifest examples/package-registry/bundle-agent.yaml --out examples/package-registry/.statetwin/agent-world.stb
statetwin task cases --root examples/package-registry --cases agent-cases.json
statetwin task qualify --root examples/package-registry --cases agent-cases.json --format markdown
```

预期 cases 为 18 matched，qualify 为 qualified：六张任务卡各有一个正例、一个目标
反例和一个策略反例。负例按期得到 task_failed/policy_violation 才是用例匹配，不能
把这 18 个 matched 当作 Agent 的 18 次任务成功。每个用例独立世界，后项看不到前项
的修改。基础设施/清理失败停止执行，后续 not_started 保留；普通预期不符继续执行。

`pkg-advisory-abstain` 要求看到 high advisory 后不安装；这是 Task 政策，不是 Twin
引擎自动禁止安装。`pkg-scope-protection` 连被阻止的越权尝试也拒绝。
qualified 只覆盖已声明的正反例，不证明 oracle 完备、作者独立或模型能力。

## 2. 审阅独立 Task，再运行和验收

```powershell
statetwin eval suite-review --root examples/package-registry --suite agent-suite.json --out .statetwin/delivery-01 --expect agent-expectation.json --tasks reviewed/catalog.json
statetwin eval suite --root examples/package-registry --suite agent-suite.json --out .statetwin/delivery-01
statetwin eval suite-assess-reviewed --root examples/package-registry --out .statetwin/delivery-01 --expect agent-expectation.json --tasks reviewed/catalog.json --policy candidate-pass-v1
statetwin eval suite-assess-reviewed --root examples/package-registry --out .statetwin/delivery-01 --expect agent-expectation.json --tasks reviewed/catalog.json --policy both-pass-v1 --format markdown
```

review 预期 matched、checkedPairs=6，且不会创建结果目录。suite 执行 12 个 mock
trial；两种新策略预期 passed，taskBinding.matchedTrials=12。审阅文件路径相对 root，
不是相对 catalog 所在目录。独立 Task 副本必须由操作者单独审阅，不能从结果生成。

可复现的拒绝示例：在临时工作副本中保留 reviewed/tasks，修改 agent-tasks 下某个
Task 的 goal oracle 为 `true`，不改 revision。重新执行 suite-review 应非零，报告
task_mismatch/oracle。若仍在新的输出目录执行 suite，旧 suite-assess 可能通过，
suite-assess-reviewed 必须拒绝，保留两个封套的不同结论。恢复时使用原版本文件，
不要改已有报告或把 reviewed 副本同步成坏定义来消除诊断。

原来的 suite-assess 保持可用。新绑定只确认完整 Task 相同，Task.Bundle 字符串相同
不代表独立批准了 bundle 内容，也不证明 runtime/model 来源。review 不是后续执行
许可；操作之间可能发生变更，因此运行后必须重新核验。

## 3. 导出与导入合成证据

```powershell
statetwin eval suite-export --root examples/package-registry --out .statetwin/delivery-01 --archive .statetwin/delivery-01.tar
statetwin eval suite-import --root examples/package-registry --archive .statetwin/delivery-01.tar --out .statetwin/restored-01
statetwin eval suite-inspect --root examples/package-registry --out .statetwin/restored-01
```

只有干净发布且经重放审计的 suite 可导出，但有效的负面评测同样允许。归档保留原始
证据字节，不带外部 expectation、reviewed Task 目录、用户名或源路径，也没有新增
文件 hash 清单或签名。接收方要另外选择自己的期望输入。

导入先在只读内存视图验证所有成员和重放，再独占创建新目录，最后发布 report。
跨 runtime 身份的证据按原规则拒绝，不提供强制兼容或自动迁移。它不是通用数据库
备份恢复。取消、写入失败或进程退出可能留下 pending/部分目标；不要将其视为成功，
也不要重复使用该目标路径。published_with_residue 必须非零，供操作者诊断。

## 4. 显式清点与只读保留预览

在 root 下单独维护 registry.json：

```json
{"format":"statetwin.dev/suite-registry/v1alpha1","entries":[
  {"id":"source","out":".statetwin/delivery-01"},
  {"id":"restored","out":".statetwin/restored-01"}
]}
```

以及 retention.json：

```json
{"format":"statetwin.dev/retention-intent/v1alpha1","entries":[
  {"id":"source","intent":"keep","active":false,"references":["restored"]},
  {"id":"restored","intent":"review","active":false,"references":[]}
]}
```

```powershell
statetwin eval inventory --root examples/package-registry --registry registry.json --mode metadata
statetwin eval inventory --root examples/package-registry --registry registry.json --mode replay --format markdown
statetwin eval retention-preview --root examples/package-registry --registry registry.json --policy retention.json --format markdown
```

metadata 只表示观察状态，reportVerification=not_checked；problem 区分 metadata_present
与 metadata_partial。replay 才核验报告/世界。inventory 返回零表示完成清点，不代表
每个条目合格；损坏/缺失仍保留。超过共享时间/读取限额时后续 not_checked 不会丢失。

source 的 keep 经引用保护 restored；未知或损坏观察也保护其依赖。review_candidate
只表示注册范围内未被保护，globalReferenceCoverage 始终 unknown。预览不删除、不输出
可执行删除计划，也不把观察字节数冒充可释放磁盘空间。

## 资源和故障解释

单操作共享 120 秒，不会对每个阶段或条目重新计时。cases 最多 16 Tasks/64 cases，
suite 最多 16 pairs；目录 4 MiB，suite payload 128 MiB、归档 129 MiB/68 members。
各原有单文件限制仍适用；这不是进程 RSS 硬配额。复杂组合可能在达到数量上限前超时，
此时报告必须保留未执行/未观察项。

新只读报告支持 JSON/Markdown，Markdown 以带作用域说明的完整 JSON 诊断块展示，
字段、分母与原因完全一致。归档写命令输出 JSON 生命周期结果。stdout 写失败非零，
不会把已发布文件覆盖回滚。所有错误使用有限标签，原始 Task/trace/路径不出现在诊断中。
