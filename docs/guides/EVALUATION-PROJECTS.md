# 离线评测项目与 Campaign

本指南对应 [SPEC-0077](../SPEC-0077-EVALUATION-PROJECT-COMPLETION.md)。输入全部为
合成离线资产，不需要 provider 凭据或访问生产系统。单项目组合静态审阅、witness
质量门禁、baseline/candidate suite、Task/世界绑定和结果发布。

## 从干净检出准备

在仓库根运行。需要项目指定的 Go 版本；下列为 PowerShell 命令。Linux/macOS 将
`New-Item` 换成 `mkdir -p`，将 `./statetwin.exe` 换成 `./statetwin` 即可。
每条命令必须成功再执行下一条，不能用最后一次 build 的成功掩盖之前的失败。

```powershell
go build -o statetwin.exe ./cmd/statetwin
New-Item -ItemType Directory -Force .statetwin, examples/issue-tracker/.statetwin, examples/package-registry/.statetwin
./statetwin.exe bundle build --manifest examples/issue-tracker/bundle-agent.yaml --out examples/issue-tracker/.statetwin/agent-world.stb
./statetwin.exe bundle build --manifest examples/issue-tracker/reviewed/world/agent/bundle-agent.yaml --out examples/issue-tracker/.statetwin/reviewed-agent-world.stb
./statetwin.exe bundle build --manifest examples/package-registry/bundle-agent.yaml --out examples/package-registry/.statetwin/agent-world.stb
./statetwin.exe bundle build --manifest examples/package-registry/reviewed/world/agent/bundle-agent.yaml --out examples/package-registry/.statetwin/reviewed-agent-world.stb
./statetwin.exe bundle build --manifest examples/package-registry/bundle-project.yaml --out examples/package-registry/.statetwin/project-world.stb
./statetwin.exe bundle build --manifest examples/package-registry/reviewed/world/project/bundle-project.yaml --out examples/package-registry/.statetwin/reviewed-project-world.stb
```

`reviewed/tasks` 与 `reviewed/world` 是单独提交、可审阅的参考源。运行时不会自动从
待测输入生成或覆盖它们；修改世界后须有意识地审阅参考差异。内容相同不证明审批人
身份或可信来源。不能把参考路径指回实际 Task/bundle，也不能用硬链接绕过独立性。

## 单项目：两个领域都可独立运行

```powershell
./statetwin.exe eval project-check --root examples/issue-tracker --project project-core.json
./statetwin.exe eval project-run --root examples/issue-tracker --project project-core.json --out .statetwin/core-run
./statetwin.exe eval project-inspect --root examples/issue-tracker --project project-core.json --out .statetwin/core-run --format markdown
./statetwin.exe eval project-check --root examples/package-registry --project project-core.json
./statetwin.exe eval project-run --root examples/package-registry --project project-core.json --out .statetwin/core-run
./statetwin.exe eval project-inspect --root examples/package-registry --project project-core.json --out .statetwin/core-run --format markdown
```

输出目录必须不存在，父目录必须已经存在。重复运行请指定新目录，不会覆盖旧证据。
`project-extended.json` 运行各领域新增的六个组合/只读/拒绝任务；原六个 Task 保持原定义。
任务允许改动、禁止尝试、oracle 与 witness 的对照见
[issue 任务表](../../examples/issue-tracker/agent-tasks/README.md)与
[registry 任务表](../../examples/package-registry/agent-tasks/README.md)。

## 四组一起运行

```powershell
./statetwin.exe eval campaign-check --root . --campaign examples/evaluation-campaign.json
./statetwin.exe eval campaign-run --root . --campaign examples/evaluation-campaign.json --out .statetwin/green
./statetwin.exe eval campaign-inspect --root . --campaign examples/evaluation-campaign.json --out .statetwin/green --format markdown
```

四个项目按 manifest 顺序串行，共 24 个任务、103 个质量 case、48 个 trial。所有项目
输入先完成准入并冻结；最后一组有错时前面也不会执行。输出中 `projects/<id>` 保存
独立项目证据，根报告只汇总身份、状态和固定分母，不复制所有 raw trace。

独立回归样例使用 `examples/evaluation-regression.json` 与一个新输出目录：
issue-close-with-comment 的候选只评论、不关闭；pkg-publish-then-install 的候选只发布、
不安装。`campaign-run` 应返回非零，但完成所有四组、48 个 trial，并发布 `failed`。
两个失败项目报告精确定位各自 `candidate` 的 `objective`。随后 inspect 可以返回零：
它证明失败报告与当前证据一致，不把历史失败变成成功。

## 读报告与搬运证据

JSON 是完整有界结构，`--format markdown` 投影相同结论/计数并附 JSON。`completed`
指执行结束，不等同业务成功；质量 case matched 指符合预期，包括正确命中的负例。
`CaseFailures` 列未命中预期的 case；`baseAssessment.failedChecks` 列已核验 trial 的失败
断言；Task binding、World binding、base assessment 分别显示，不互相代替。

质量不合格会发布 failed 报告，但不创建 suite，所有 trial 仍计为 not_started。
基础设施、清理、取消、超限或发布错误立即停止，保留 partial；之后各项目保留完整分母。
`published_with_residue` 表示最终文件已出现而临时文件清理失败，需要人工检查。

inspect 在可信、静止的本地根目录中只读核验，重放 suite；qualification 仅为历史记录的
结构和计数核对，明确标为 `recorded_only`，不会再次运行 witness。它使用当前提供的
参考，不证明历史审批、原进程存活、归档来源或无并发修改。不能用它恢复/继续执行。

沿用既有 suite 归档格式（只搬运子 suite，不宣称归档了整个 project/campaign）：

```powershell
./statetwin.exe eval suite-export --root . --out .statetwin/green/projects/registry-core/suite --archive .statetwin/registry-suite.tar
./statetwin.exe eval suite-import --root . --archive .statetwin/registry-suite.tar --out .statetwin/restored-suite
./statetwin.exe eval suite-verify --root . --out .statetwin/restored-suite
```

## 格式与限额

| 范围 | 契约 |
|---|---|
| project | evaluation-project/v1alpha1；offline-reviewed-project-v1；所有九个字段必填 |
| campaign | evaluation-campaign/v1alpha1；offline-reviewed-campaign-v1；2–4 个独立 ID 项目 |
| 世界目录 | reviewed-world-catalog/v1alpha1；offline-world-content-v1；与 Task 集合精确一致 |
| manifest | 严格 JSON、64 KiB、深度 16；拒绝未知、重复、null、类型错与路径越界 |
| 项目输入 | raw/extracted 各 128 MiB；campaign 各 256 MiB；旧组件较小限额仍保留 |
| 项目输出 | suite 128 MiB + metadata 4 MiB；单报告 1 MiB；不得覆盖 |
| 时间 | project 120 秒；campaign 480 秒且每组 120 秒；串行、有界、无后台重试 |
| witness | 旧 v1 不变；显式 v2 支持受限评分视图字符串替换，见 Spec 第 15 节 |

七个 case 带 `gradingSource=synthetic-view-mutation`，用于旧 oracle 的状态变化反例；
先真实执行并清理，再只改评分副本。这些不是 Agent 执行证据，不可当作 Agent 成功率。
97 个不同 witness 中包含 18 个任务的第二条合法调用路径。质量覆盖要求每条断言被
有效语义反例命中；`not_evaluated`/错误评分不算语义覆盖。

## 有限错误与处理

错误前缀为 PROJECT 或 CAMPAIGN，输出不会透传路径内容、凭据或原始表达式。

| 后缀 | 处理 |
|---|---|
| INPUT_INVALID / INPUT_UNAVAILABLE | 检查 manifest、文件类型、路径、隐私规则；check 未执行任务 |
| REFERENCE_MISMATCH | 审阅 Task/世界/计划引用，避免别名、遗漏或意外修改 |
| RESOURCE_LIMIT | 缩小本次数据或拆分业务范围；不会截断分母后报成功 |
| DEST_EXISTS | 换用新的输出目录，旧目录保持原样 |
| CANCELED_OR_TIMED_OUT | 检查 partial；使用新目录重新发起，不自动恢复 |
| QUALITY_NOT_QUALIFIED / ASSESSMENT_FAILED | 看 case/任务/trial/断言表，修复实际问题而非改预期掩盖 |
| WRITE_FAILED / PUBLISH_FAILED / CLEANUP_FAILED | 检查磁盘、权限与残留，不能把 partial 当成功 |
| OUTPUT_FAILED | stdout 失败；已经发布的磁盘结果不会回滚 |

旧 suite、assessment、archive、inventory、retention 命令继续可用。项目包装没有新增
agent-facing 控制工具、上游 passthrough、生产写入、自动删除或 live 授权。
