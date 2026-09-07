# 来源与核验登记

日期：2026-09-08。下面 S 编号仅代表本轮实际查看；不假装恢复原稿 R01–R17/W01–W07。
代码观察基线为本地 `77d3ea0ec610fe64b48425f15c350e0b4c0d577f`，不是新执行证据。

| ID | 实际来源 | 可以支持的结论 | 本轮不支持的结论 |
|---|---|---|---|
| S01 | [episode.go](../../../internal/episode/episode.go) 的 Run/run/selectScenario | 当前从 Bundle 解码 Scenario 后执行 scenario.Run | 自主 AgentTask 已实现 |
| S02 | [smoke.go](../../../internal/provider/smoke.go) 的 Report/Run | report 是工具观察计数和摘要，不是业务目标 oracle | live 曾执行、模型能完成任务 |
| S03 | [data.go](../../../internal/server/data.go) 的 ServeHTTP/buildHandler | 路径绑定 branch，无主体授权检查；传递业务 result | 对恶意本地进程隔离、公网安全 |
| S04 | [protocol_test.go](../../../internal/server/protocol_test.go)、[go.mod](../../../go.mod) | 仓库的协议测试和依赖 pin 可定位 | 最新官方完整协议或本轮测试通过 |
| S05 | [issue Twin](../../../examples/issue-tracker/twin.yaml)、[state](../../../examples/issue-tracker/state.json) | 六个业务工具；close 已关闭对象冲突；有 get 无评论查询；原 fixture 仅一个 issue | 自动幂等、GitHub 等价、六任务已执行 |
| S06 | [package Twin](../../../examples/package-registry/twin.yaml) | installation 是 insert，无安装查询工具 | upgrade 已有安装、完整 SemVer/上游等价 |
| S07 | [ADR-0021](../../ADR-0021-UNIFIED-LIFECYCLE-AND-RELEASE-BOUNDARIES.md)、[ADR-0018](../../ADR-0018-TWINBUNDLE-AND-LOCAL-EPISODE-PREVIEW.md)、[RFC-0002](../../RFC-0002-V0.1-RELEASE-PROFILE.md) | provider gate 分线已接受，preview 与稳定 claim 需逐项映射 | 已满足 release 所有门槛 |
| S08 | [SPEC-0020](../../SPEC-0020-REMOTE-SECURITY-PROFILE.md)、[Phase 4](../../PHASE-04-PROVIDER-VALIDATION.md) | remote security/live 要求不能由本地 mock 代替 | 当前 profile 已安全部署 |
| S09 | [Implementation Status](../../IMPLEMENTATION-STATUS.md)、[store tests](../../../internal/store/store_test.go)、[Journal tests](../../../internal/episode/journal_test.go) | 历史证据清单和相应测试源码存在 | 本轮重跑过这些测试 |
| S10 | [RFC-0001](../../RFC-0001.md) I-1–I-16 | deterministic、隔离、unknown、immutable evidence、fencing 等硬不变量 | 所有路线都已实现 |
| S11 | [官方 Function calling 文档](https://developers.openai.com/api/docs/guides/function-calling)，本轮搜索并打开正文 | 应用执行函数，通过 call_id 关联结果，续接所需响应项须正确处理 | 当前账号可用、产品宿主兼容、项目 live 通过 |

外部文档仅用于本地 bridge 可行性和调用关联的有限依据；并未开展新一轮竞品/模型发布/
隧道/Anthropic/完整 MCP 官方协议研究。总稿中的相关历史研究不能充当本轮已验证事实。
原稿 R/W 标签保留为待解析线索；只有具备原来源或独立复验后才允许晋级为本轮依据。

本轮未查询最新远程分支/Issue/PR/release 状态，未复验历史 Actions run。相关数字和
链接若出现在导入总稿，指原作者历史记录，不是当前 GitHub 状态。进入实施或发行前
必须重新确认所选候选提交及 exact-revision CI。

上传的两份原文仍保留在用户下载目录；仓库中的是注明 v3.1 的编辑副本，非字节级归档。
本轮没有计算文件哈希或生成一致性哈希清单。既有业务 canonical/digest 合约原样保留。
