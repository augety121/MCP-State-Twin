# SPEC-0077 交付台账

授权：2026-10-02 用户要求整批实施并完成 PR。实施 PR 为
[#18](https://github.com/augety121/MCP-State-Twin/pull/18)，直接指向 main。
本表是本地验证与发布前快照；**最终 head 的完整 CI 结论与完成状态以该 PR 描述及 Checks 为准**，
不能将历史 run 当作最新 head 已通过。没有自动合并或生产发布。

## 版本与主线集成

- 集成基线 main：`406a7758df55d952d519243249b3cd47de31391a`。
- 旧实现：`3de2ef996138e8dda186edffa4147b42e2eac136`（原 #14 仅合入设计分支）。
- 实现提交：`dfef1cefdc12d29aefd17cace97d62b8fb40617f`；补充边界提交：`b3aa75c98170b6e034b0aaef2ac99d415780e9a9`。
- 已执行两项 `git merge-base --is-ancestor`，基线与旧实现均在当前候选祖先内。
- 已核对原两领域 12 个 Task 与旧实现之间的 Git diff，未改动原定义/oracle。
- Bot [#15](https://github.com/augety121/MCP-State-Twin/pull/15) 已修复 SDK pin 不匹配，
  Windows Test/Build 分步并加回归检查；`67339292d5804c10ed9e2afe64f74bd7e5a48072`
  的[完整 CI](https://github.com/augety121/MCP-State-Twin/actions/runs/36965127089)通过。
  SQLite 1.59.0 保留，没有用旧分支降级依赖。

## 工作包

| workItem | acceptanceIds | implementationFiles | executedTests | result | headRevision | remaining |
|---|---|---|---|---|---|---|
| W0 集成 | A00 | Git ancestry; protocol.go; ci.yml | ancestry; SDK pin/workflow regression; Bot CI | passed | b3aa75c / Bot 6733929 | main merge 由维护者决定 |
| W1 准入/冻结 | A01–A05 | project_plan.go; shared input readers | 下表对应测试 | passed | b3aa75c | 无业务剩余 |
| W2 世界绑定 | A04/A06–A08 | world_catalog.go | 独立路径、ZIP 等价、完整替换、坏 terminal | passed | b3aa75c | 无业务剩余 |
| W3 项目闭环 | A09–A13 | project_run.go; project_inspect.go | 门禁、执行、故障、只读重验 | passed | b3aa75c | 无业务剩余 |
| W4 多项目 | A14–A17 | campaign.go; campaign_inspect.go | 全量准入、串行停止、分母、跨项目别名 | passed | b3aa75c | 无业务剩余 |
| W5 两领域资产 | A18/A19 | examples; bounded case v2 | 24 Task / 103 cases / 97 witness / 18 alternate positives | passed | b3aa75c | 无业务剩余 |
| W6 可读报告 | A20/A21 | project_eval.go; project diagnostics | JSON/Markdown 投影、短写、门禁和磁盘发布分离 | passed | b3aa75c | 无业务剩余 |
| W7 计量 | A22–A24 | operation-local input cache; benchmarks | 实际读/解包/replay 次数；五次采样矩阵 | passed | 同一实现的 benchmark 工作树 | 无性能采样剩余 |
| W8 边界与回归 | A25–A29 | project/campaign tests; old regression suite | 全量本地测试及新增边界测试 | passed | b3aa75c | 最终平台 race 见 W9 |
| W9 文档/交付 | A30/A31 | guide; status/index; this ledger; PR | 本地 vet/test/指南 CLI；dfef1ce 完整 CI | passed | dfef1ce；后续 head 见 PR Checks | 最新 head 检查由 PR 描述记录 |

## 逐项验收

以下本地测试及采样均已执行并通过；初始实现 dfef1ce 的完整远端 CI 也已通过。A29 全量测试之后
新增的精确边界测试和门禁原因变更已定向验证；最终远端 CI 再对整个 head 运行全量检查。

| ID | workItem | executedTests / evidence | result |
|---|---|---|---|
| A00 | W0 | git ancestor checks; main-base PR #18; Bot #15 exact-head CI | passed |
| A01 | W1 | TestProjectClosedAdmission; TestCampaignAllInputPreflight | passed |
| A02 | W1 | TestProjectCheckReadOnly | passed |
| A03 | W1 | TestProjectInputCoverage | passed |
| A04 | W1/W2 | TestProjectReferenceSeparation; TestCampaignCrossProjectReferenceAlias | passed |
| A05 | W1/W4 | TestProjectFrozenInputs; TestCampaignFrozenInputs | passed |
| A06 | W2 | TestReviewedWorldEquality | passed |
| A07 | W2 | TestCoherentWorldSubstitution | passed |
| A08 | W2 | TestProjectVerifiedWorldBinding | passed |
| A09 | W3 | TestProjectQualityGate | passed |
| A10 | W3/W5 | TestProjectEndToEnd; TestCampaignAssetsQualification; TestFourProjectCampaign | passed |
| A11 | W3 | TestProjectFailureAccounting | passed |
| A12 | W3/W8 | TestProjectProcessExit | passed |
| A13 | W3 | TestProjectInspectionScope | passed |
| A14 | W4 | TestCampaignAllInputPreflight | passed |
| A15 | W4 | TestCampaignStopPolicy | passed |
| A16 | W4 | TestCampaignDenominators | passed |
| A17 | W4/W5 | TestFourProjectCampaign; TestRegressionCampaignCLI | passed |
| A18 | W5 | TestCampaignAssetsQualification; TestMutationIsolationAndAdmission; TestRealUnscorableCaseDoesNotCoverGoal | passed |
| A19 | W5 | TestCampaignAssetsQualification (103 exact outcomes/check sets); unchanged original Task diff | passed |
| A20 | W6 | TestProjectReportProjection; TestProjectFailureAccounting | passed |
| A21 | W6 | TestProjectGuideCLI; TestProjectRunStdoutFailurePreservesPublication | passed |
| A22 | W7 | TestPreparedProjectReadAccounting; TestProjectFrozenInputs | passed |
| A23 | W7 | TestVerificationCacheInvalidation | passed |
| A24 | W7 | BenchmarkProjectPrepare; BenchmarkProjectAssess (five samples per cell) | passed |
| A25 | W8 | TestProjectResourceBoundaries; TestProjectManifestInclusiveLimit; TestProjectPrivacyAndMetadataLimits; TestCampaignAllInputPreflight; existing suite/case/archive limits | passed |
| A26 | W8 | TestProjectCancellationStages; TestCampaignStopPolicy; existing suite cancellation/cleanup tests | passed |
| A27 | W8 | TestProjectPublicationFaults; TestCampaignPublicationFaults | passed |
| A28 | W8 | TestProjectPrivacyAndMetadataLimits; TestProjectReferenceSeparation; TestProjectInspectionScope | passed |
| A29 | W8 | go test -p 1 -timeout=20m ./... (24 packages); existing suite/assessment/archive regression tests | passed |
| A30 | W9 | TestProjectGuideCLI (clean copied sources, check/run/inspect/export/import, unchanged inputs) | passed |
| A31 | W9 | gofmt; go vet -p 1 ./...; local complete tests; dfef1ce all 7 CI jobs passed; final-head checks recorded on PR | passed |

## 执行环境与证据边界

本地 Go 1.26.5、Windows/amd64、GOMAXPROCS=1。`go vet -p 1 ./...` 和
`go test -p 1 -timeout=20m ./...` 已通过；24 个 package 均报告 ok。
`go test -race ./...` 在本机返回 `-race requires cgo`，未冒称通过；race 由 PR Linux
CI 执行。CI 同时覆盖 Windows/macOS、hermetic-egress、fuzz、secret-policy 和 MCP conformance。
性能不使用 race 时间，见[五次采样记录](SPEC-0077-PERFORMANCE.md)。

存储/中断测试只操作 t.TempDir 内的实例。CLI 测试复制源资产到临时干净根再构建，
不会改写仓库原始 Task/参考或用户运行数据。性能/命令日志是合成测试输出；交付前清理
本轮 `.tmp/spec77` 中不需要的临时文件，保留本表和性能汇总，不创建新文件 hash 清单。

质量统计与 Agent 成功率分开；独立内容一致性不鉴证来源；inspection 的 qualification
仅 recorded_only。受限评分视图变异的理由、界限与事实标签见
[Spec 第 15 节](../SPEC-0077-EVALUATION-PROJECT-COMPLETION.md#15-实施修订受限评分视图变异)。


## 远端验证记录

[CI 36972174058](https://github.com/augety121/MCP-State-Twin/actions/runs/36972174058)
对应 `dfef1cefdc12d29aefd17cace97d62b8fb40617f`，七个 job 全部 success，含 Linux
race/vet/format、Windows、macOS、fuzz、secret-policy、hermetic-egress、MCP conformance。
后续补充提交仍独立触发 CI；最终交付须检查 PR #18 实际 head，不能用上述历史结果
代替。为避免文件内自指提交号，最终 head 与 run URL 放在 PR 描述中维护。
