# SPEC-0078 实施台账

2026-10-02 用户明确授权整批实施。基线 main `5ad90baa691a8df8ece1942aa74e2d99bcc3502e`；分支 `codex/agent-baseline-plugin`。本表随真实实现和验证更新，不把 proposed 测试名当执行证据。

| 范围 | 状态 | 实现/验证 | 剩余 |
|---|---|---|---|
| N0 / W00 | implemented, final-head pending | 主线已核对；Go 1.26.5 / SDK 1.8.0 / Python 3.12 / Inspect 0.3.275 / MCP Python 1.26.0 / go-winio 0.6.2 固定 | 最终 head CI |
| N1 / W01–W08 | implemented, local tests passed | 严格准入、Task authority、stdio/私有 IPC、屏障/回放/inspect、两 Go profile ×24 Task、Inspect ×24 Task 与实际框架评分、故障矩阵 | 最终 race/平台；async cancel 新反例最终结果待记录 |
| N1 / W09 | in_progress | 100 次会话 soak、30 性能样本、10 次分阶段 crash；本地 wheel 构建安装；指南和三平台 CI | 最终 CI；1.5 倍性能门槛未通过，不能记 N1 完成 |
| N2 / W10–W11 | implemented, local tests passed | 24 family/72 独立 Task、有限变体、三 split 全部正负/变异 case、公开披露、revision 冻结/弃用/撤销 | 最终 head；明确 self-reviewed，无私有 holdout 或独立评审宣称 |
| N3 / W12–W14 | implemented, local tests passed | 原始输入冻结、AB/BA、144/240 上限、串行 shard、固定分母/未知费用、真实 shard 缺 seal 反例、family bootstrap 数字 golden 与四态判定 | 完整 144-trial scripted pilot 与最终 head CI 正在执行 |
| N4 / W15 | implementation ready, external evidence blocked | 独立 API bridge live shard；全片批准早于凭据/claim；到期/未批准/错 profile/耗尽/错误绑定零请求反例；原有六任务 mock/不确定响应契约 | 缺具体模型、账号、费用/合成数据/有效期批准；实际六任务及 pilot 未执行 |
| N4 / W16 | external evidence blocked | 产品/非作者 first-value 输入契约、TTL 与隔离边界 | 缺具名产品版本/可隔离环境、三名非作者及联系/记录授权 |
| N5 / W17 | implemented, qualification pending | binding/TTL/撤销评估、copy-to-new 恢复/不覆盖、支持与维护流程 | 最终候选 CI/安装、性能决定；不自动 tag/发包/stable |
| N5 / W18 | external evidence blocked | 独立 reference/reset/三操作差分合同已写明 | 缺获准且可复位的独立服务；无实际差分，不宣称 L2 |

本机 Windows 无可用 CGO，race 将由 Linux CI 验证；没有结果之前不记通过。任何失败先查根因，保留原业务 deadline 和失败分母。只清理本轮自己的临时文件与进程。

## 实现入口与已执行的测试

- `internal/agenteval/plugin_world.go`：旧 authority/local MCP world；`TestPluginWorldAuthorityAndReplay`、`TestPluginWorldExistingWitnessEquality`、`TestPluginConcurrentAdmissionBudgetAndFinish`。
- `internal/baselinepack`：严格字段/required/未知/重复/null/路径/硬链接、schema、变体、revision；`TestBaselinePackAdmission`、`TestPluginReferenceIsolationAndFreeze`、`TestBaselineVariantQualification`、`TestBaselineRevisionLifecycle`。
- `internal/plugin`：stdio/IPC/storage/inspect/recover；`TestPluginStdioLifecycleAndInspection`、`TestPluginEOFAndAuthority`、`TestPluginControlIsolationAndSequence`、`TestPluginPublicationFaults`、`TestPluginStdioLimitsAndOutput`、`TestPluginTenStagedCrashesReapOwnedChildren`。
- `cmd/statetwin`：`TestPluginIndependentProcessProfilesAll24`、`TestPluginCLIProjectionAndOutput`、`TestBaselineRealShardAndFixedDenominators`、可选 `TestPluginReleaseSoakAndPerformance`。
- `adapters/inspect`：固定版本、fresh session、Task/Solver/Scorer；真实进程 24 Task、日志、自报评分、异常/取消、额外工具。
- `internal/baseline`：`TestBaselinePlanIdentityAndFreeze`、`TestBaselineDecisionPolicy`、`TestBaselineStatisticalCounterexamples`、`TestBaselineShardBudgetsAndNoEffects`、`TestBaselineLiveReadinessZeroOutbound`、`TestBaselineLiveShardApprovalBeforeCredentialsAndClaim`、`TestBaselineClaimBindingFreshnessAndRevocation`。

## 验证记录与限制

上述 targeted Go 测试通过，`go vet ./...` 通过。Inspect 本地 wheel 构建/安装通过；24 Task 及完整 Task/Solver/Scorer 测试通过。第一次整仓测试与代码新增并行，遇到 source file inventory 尚未包含新增 live 文件及指南代码被链接检查误识别的问题；对应问题已处理，稳定候选整仓检查正在执行，不把失败轮次称为通过。

Python ctypes 被沙箱 DLL 策略阻止，测试在同一项目 venv 的获准进程中运行；中文 Windows 固定进程级 `PYTHONUTF8=1`，没有改全局 Python 或系统编码。

性能第一次实测：Windows/amd64、Go 1.26.5、GOMAXPROCS=1、100 独立 session、30 样本；完整链路 p95=1.1484264 秒，预加载进程内 p95=0.0400349 秒，新增=1.1083915 秒，倍率=28.6856。新增≤2秒通过；原总耗时≤1.5倍未通过。启动与 RSS 分阶段测量已补入测试，最终数据待记录。已向用户提出门槛修订问题，未得到明确同意前维持原门槛失败。

实际 provider 请求次数为零；没有读模型凭据、邀请外部参与者、操作生产服务或发布 tag/包。外部缺项已询问，仍可独立完成的开发/测试/PR 工作继续推进。

配套：[安装与使用](../guides/BASELINE-PLUGIN.md)、[验收矩阵](../planning/agi-baseline/ITERATION-AND-ACCEPTANCE.md)、[外部 reference 合同](../planning/agi-baseline/REFERENCE-QUALIFICATION.md)。
