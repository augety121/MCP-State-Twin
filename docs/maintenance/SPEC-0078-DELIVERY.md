# SPEC-0078 实施台账

2026-10-02 用户明确授权整批实施。基线 main `5ad90baa691a8df8ece1942aa74e2d99bcc3502e`；分支 `codex/agent-baseline-plugin`。本表随真实实现和验证更新，不把 proposed 测试名当执行证据。

| 范围 | 状态 | 实现/验证 | 剩余 |
|---|---|---|---|
| N0 / W00 | implemented, final-head pending | 主线已核对；Go 1.26.5 / SDK 1.8.0 / Python 3.12 / Inspect 0.3.275 / MCP Python 1.26.0 / go-winio 0.6.2 固定 | 最终 head CI |
| N1 / W01–W08 | implemented, local tests passed | 严格准入、Task authority、stdio/私有 IPC、屏障/回放/inspect、两 Go profile ×24 Task、Inspect ×24 Task 与实际框架评分、故障矩阵；异常/async cancel/concurrency 五项 Python 测试通过 | 最终 head CI |
| N1 / W09 | in_progress | 100 次会话 soak、30 性能样本、10 次分阶段 crash；本地 wheel 构建安装；指南和三平台 CI | 最终 CI；1.5 倍性能门槛未通过，不能记 N1 完成 |
| N2 / W10–W11 | implemented, local tests passed | 24 family/72 独立 Task、有限变体、三 split 的 12 组 309 正负/变异 case 全匹配；可执行 qualify、公开披露、revision 冻结/弃用/撤销 | 最终 head；明确 self-reviewed，无私有 holdout 或独立评审宣称 |
| N3 / W12–W14 | implemented, local tests passed | 原始输入冻结、AB/BA、144/240 上限、串行 shard、固定分母/未知费用、真实 shard 缺 seal 反例、family bootstrap 数字 golden 与四态判定；完整 144-trial scripted pilot 全部重验，结果 inconclusive/degenerate | 最终 head CI；scripted pilot 不代替真实模型 pilot |
| N4 / W15 | implementation ready, external evidence blocked | 独立 API bridge live shard；全片批准早于凭据/claim；到期/未批准/错 profile/耗尽/错误绑定零请求反例；原有六任务 mock/不确定响应契约 | 缺具体模型、账号、费用/合成数据/有效期批准；实际六任务及 pilot 未执行 |
| N4 / W16 | external evidence blocked | 用户委托选择后，核对本机 Codex CLI 0.159.0-alpha.12.1 和官方配置；当前聊天存在 shell/文件/其他工具，不能充当 tools-only 产品实测；产品/非作者 first-value 输入契约、TTL 与隔离边界 | 缺隔离产品会话证据、三名非作者及联系/记录授权 |
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

2026-10-08 后续改进：PR #19 已合并，后续以 `dfb3f496ea4509919914d76a88584164e85a4bc3` 为基线。修复 executable qualify 的去重身份：相同 manifest 路径在不同 root 下仍须分别执行，不能漏验第二套 witness/world。新增真实正例 root 与失败 shadow root 的回归测试。Inspect 辅助 CLI 改为同时读取有界 stdout/stderr，超过 1MiB/256KiB 立即终止并等待自有子进程，120 秒超时返回有限错误码；不再退出后才检查临时文件大小。四项独立进程测试覆盖精确边界、两流超限、错误信息不泄露与超时。

本轮按照用户授权启用 PR 的 Codex 自动代码审核，检查完成且审核问题处理后可合并 main；不改变 stable、真实模型、非作者或 L2 的验收条件。Jev 仅收到跨 root 去重的合成问题描述，返回 semantic_gap（confidence=1），只用于问题优先级判断，实际正确性仍由执行测试证明。

PR #21 的 Codex Code Review 指出 P2：继承管道的后代可能让读取线程 join 超时失效。改为 Python 3.12 支持的非阻塞管道，在同一操作 deadline 内公平读取两流；即使后代保留写端也不等待线程或无限 EOF。辅助 CLI 的直接子进程仍 kill/wait，流总是关闭；不把该 helper 宣称为任意可执行程序的进程树沙箱。新增真实继承管道和调用者 KeyboardInterrupt 反例，六项 Python 进程测试通过。测试后代通过专用停止文件退出，未影响其他进程。

2026-10-03 后续修复：`4dc0c6e9f593e94cf168439f7671f8a5317d1c83` 的 [18 项 CI 全部通过](https://github.com/augety121/MCP-State-Twin/actions/runs/37035109984)，包括完整分片 race、三平台 Inspect 与其他安全检查。随后实现单次准入内精确字节复用、去除重复 oracle 编译，并补上跨 root 缓存隔离反例；本地 gofmt、go vet、整仓 Go 测试再次通过。24-entry 准入微基准中位数下降约 24.8%，100 会话 soak 和 30 次 Inspect 框架复测通过。新提交的 race 结果以 PR 最终 head 检查为准。性能原始≤1.5倍门槛仍未满足，详见性能记录的优化复测段。

上述 targeted Go 测试通过，`go vet ./...` 通过。稳定候选的 `go test -p 1 -timeout=20m ./...` 全部通过。Inspect 本地 wheel 构建/安装通过；最终五项 Python 测试通过（114.15 秒），包含 24 Task、完整 Task/Solver/Scorer、异常、取消与并发拒绝。第一次整仓测试与代码新增并行，遇到 source file inventory 尚未包含新增 live 文件及指南代码被链接检查误识别的问题；对应问题已处理，不把失败轮次称为通过。

2026-10-03 完整离线 pilot：12 shards ×12 trials，baseline/candidate 各 72 planned、72 scorable、72 successes，missing/unverified/interrupted 均为 0；没有 policy failure 或非法实际副作用。24 family、每个配置 3 repeats，结果为 inconclusive/degenerate，符合全成功样本的保守判定。脚本没有 provider usage，tokens/cost 明确 unavailable/unknown。

Python ctypes 被沙箱 DLL 策略阻止，测试在同一项目 venv 的获准进程中运行；中文 Windows 固定进程级 `PYTHONUTF8=1`，没有改全局 Python 或系统编码。

性能首次完整链路 p95=1.1484264 秒；停止并行本地重负载后补测阶段与 RSS：100 独立 session、30 样本全部执行通过，完整链路 p95=0.8982404 秒、启动 p95=0.3443768 秒、新增 p95=0.8651119 秒、倍率=27.1138。Inspect 另测 30 次完整框架链路，p95=2.194704 秒，原≤1.5倍及新增≤2秒均未通过；未得到明确同意前维持失败。soak 测试通过不等于性能验收通过。详细测量、开销定位与限制见 [性能记录](SPEC-0078-PERFORMANCE.md)。

代码与文档已推送 [PR #19](https://github.com/augety121/MCP-State-Twin/pull/19)。首候选 832c618 的三平台 Inspect、Windows/macOS 回归、hermetic-egress、fuzz、conformance、secret-policy 均通过；最终 head 的完整 CI 由 PR 检查记录核对，不能用首候选替代。

首轮 Linux race+atomic coverage 中 cmd/statetwin 与 internal/agenteval 触及包累计 20 分钟超时，日志没有 data race；前者仍在两个 profile ×24 Task 中，后者仍在 Project cancellation 用例中。CI 改为两个重包各 4 个完整清单分片，其他包保留整包 race。脚本动态读取 Test/Fuzz/Example，验证分片并集和互斥；唯一展开项为两个各含完整 24 Task 的 protocol profile。没有删除用例、放宽业务 deadline 或去掉 race/coverage。修复后以新 head 的全部 18 个 CI job 为准。

新增 public qualify 的整包正例在 race instrumentation 下触及 120 秒操作上限；改为按 dev/regression/evaluation 三个 split 分别调用公开接口，每次仍受 120 秒生产限制，合计仍断言 12 组 309 case 全匹配。非 race 的整包 72-instance/309-case public qualify 已在本地通过（20.49 秒含生成与测试准备）；没有调整生产超时或把超时视为通过。

实际 provider 请求次数为零；没有读模型凭据、邀请外部参与者、操作生产服务或发布 tag/包。外部缺项已询问，仍可独立完成的开发/测试/PR 工作继续推进。

配套：[安装与使用](../guides/BASELINE-PLUGIN.md)、[验收矩阵](../planning/agi-baseline/ITERATION-AND-ACCEPTANCE.md)、[外部 reference 合同](../planning/agi-baseline/REFERENCE-QUALIFICATION.md)。
