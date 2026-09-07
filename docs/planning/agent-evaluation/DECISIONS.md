# 待接受的有限决策

状态：Proposal，2026-09-08。以下 AE-DEC 是评审编号，不是 accepted ADR。
本轮不把用户的“先走流程”解释为接受所有远期接口或授权付费执行。

后续实施更新：用户明确要求“推送、接着干活”后，
[ADR-0038](../../ADR-0038-AGENT-TASK-AND-OFFLINE-GRADING.md) 接受独立 Task/只读评分/
六任务 offline witness 的有限子集；[ADR-0039](../../ADR-0039-OFFLINE-AGENT-REGRESSION-LOOP.md)
随后接受合成 Responses 循环、有限证据重放与模型标签比较。真正的 provider transport/live、
通用远程恢复和发行决策仍待各自接受，
不把本轮原始评审记录改写为全生命周期已接受。

| ID | 推荐选择 | 替代与代价 | 实施前要求 |
|---|---|---|---|
| AE-DEC-01 | 保留内核，优先 Agent 回归闭环 | 先扩展更多域/调度会延后使用反馈 | 接受 B01–B12 的有限主线，不承诺全部生命周期完工 |
| AE-DEC-02 | 独立 AgentTask、AgentEpisode、AgentEvidence kind | 给旧 Scenario 加 model 会混淆 scripted/live 证据 | 不改变旧 Bundle/Journal schema；新格式 experimental、独立 reader |
| AE-DEC-03 | 首个 local API bridge 为 trusted harness、无 shell Agent | 原生 MCP 先行必须先满足远程安全门槛 | 明确外发工具/内容、private harness 边界、停止与网络 admission |
| AE-DEC-04 | 先一个 OpenAI Responses 非流式函数工具 adapter，模型 ID 不预设 | 可改选获准的另一 API，但须重新核实独立官方合约 | mock first；API 文档只证明调用形状；live 单独批准 |
| AE-DEC-05 | 合成 trace 独立保存政策，opaque continuation 内存私有 | digest-only 保存更少，但不能完整 world replay | 既有 ProviderSmokeReport 不扩展原文；拒存与 partial 语义有测试 |
| AE-DEC-06 | 只读声明式 oracle，受限业务对象与事件谓词 | 任意脚本 oracle 需另立执行/沙箱规范 | 不新增 CEL I/O 或动态 native plugin；正反例、预算、版本明确 |
| AE-DEC-07 | 产品阶段与发行版本分开 | 直接把 D2 写为稳定 v0.2 会覆盖既有 release train | B11 先做公共支持清单；版本映射经 accepted ADR 才改变 |

已经接受、无需重新决定：provider live 不阻塞局部 v0.1（ADR-0021）；不允许 hermetic
passthrough；未知行为显式失败；控制工具不进入 Agent MCP；remote exactly-once 只限
terminal Evidence 接受。

没有阻塞离线设计的外部未知：账号是否可用某模型、单价和预算、native remote 安全部署、
外部试用者、缺失原包的远期任务。它们分别阻塞 live/remote/external/完整来源审计，
不应该阻塞 mock、Task admission、评分器和离线比较器的实施。
