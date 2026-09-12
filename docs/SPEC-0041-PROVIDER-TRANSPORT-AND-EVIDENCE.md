# SPEC-0041: 有界 Provider 传输、停止与独立证据

Status: accepted experimental subset via
[ADR-0041](ADR-0041-BOUNDED-PROVIDER-TRANSPORT-EVIDENCE.md), 2026-09-08.
Authority: SPEC-0040 governs permission; SPEC-0038/0039 govern the inherited
Task, codec, local world and replay semantics except the explicit differences below.

## 1. Profile 与协议基线

`openai-responses-local-bridge-v1alpha1`，单任务、单 owner、串行非流式请求。
固定 `POST https://api.openai.com/v1/responses`。只发送 Responses function tools，
没有 server-side remote MCP URL，不接受工具内容改变网络路由。

2026-09-08 核对的官方资料：

- [Responses create](https://developers.openai.com/api/reference/typescript/resources/responses/methods/create)：请求和响应 envelope；
- [Function calling](https://developers.openai.com/api/docs/guides/function-calling)：function call / output、call_id 和 schema 投影；
- [Reasoning/stateless continuation](https://developers.openai.com/api/docs/guides/reasoning)：本次 session 私有 output continuation。

这些来源支持 wire 合约设计，不支持“已 live 兼容”的声明。不得根据官方文档推断
任意模型/账户可用性、当前价格、产品宿主兼容或真实测试分数。

## 2. 传输资源与网络边界

| 项目 | 必须行为 |
|---|---|
| Endpoint | 编译期固定 HTTPS host/path；不读自定义 URL/proxy 环境变量 |
| TLS | 正常证书和 hostname 验证，最低 TLS 1.2；无 insecure 开关 |
| Dial | 仅允许 `api.openai.com:443`；系统 DNS/根证书受信任 |
| Redirect | 所有 3xx 作为失败；不转发认证给 Location |
| Retry | 无应用重试；禁用 body replay、连接复用，不设置幂等重试头 |
| Parallelism | client 串行；每 host 最多一连接；无 worker pool |
| Time | dial、TLS、header、整个请求均有界，继承 Episode 更短 deadline |
| Size | headers ≤16 KiB；body 有界读取 `ResponseBytes+1`，超限拒绝 |
| Encoding | 非流式 JSON；压缩响应拒绝，不执行解压工作 |
| Credentials | 仅 Authorization header；不放 body/URL/工件 |

请求显式 `store:false`、`stream:false`、`parallel_tool_calls:false` 和计划的输出上限。
仅接受固定 instructions、模型、工具名称及顺序。实际工具 schema 来自已 admission
的私有 world MCP discovery，codec 保留名称/输入 schema；client 不是独立的 schema
真实性证明器。上游新增 executable item 或字段无法表达时拒绝，不偷偷降级成功。

## 3. 成功、错误与未知结果

每次进入 HTTP 调用前记录单调 sequence、批准时刻和 `acceptance_unknown`。
只有收到状态 200、合法有界 JSON、合法 metadata/usage、通过敏感模式检查后，
receipt 才成为 `response_received`。随后 codec 独立校验 completed status、output、
call IDs、参数及批次，任何失败禁止这个批次的业务 dispatch。

非 200 不读取或保留错误正文；只保留 HTTP status 和规范化错误码。
DNS、连接、TLS、超时、截断等原始错误文本不输出，避免泄漏 host/path/credential。
网络错误统一保守标为接受情况未知，即使某些低层失败可能发生在发送前。
HTTP 错误也不证明没有计费。没有自动重试、后台 polling 或远程 cancel 调用。

若 usage 缺省/null，tokens 缺省；若对象存在却字段不全、负数、溢出、总数不符则拒绝。
模型返回的 `model` 仅记录为 `reportedModel`，不是已认证 snapshot。
请求 model 与返回 model 可以因 alias resolution 不同；snapshot 固定 unknown。

## 4. 执行生命周期

计划独占 claim → 新建私有 world/session/client → 请求 admission → 单次 POST →
完整响应 admission → 按顺序工具尝试/授权/MCP dispatch → 保存本地结果供下次请求 →
完成或停止 → 有界终态检查/评分 → replay closure → 资源关闭 → terminal publication。

每次工具尝试独立事务，后续失败不回滚已经提交的工具效果。拒绝的尝试也计数，
fault 的 before/after effect 语义继续由本地事件证据决定，不能凭模型说法推断。

`delivered` 在此 profile 中是保守的 transport acknowledgement：先保留 false，
只有承载该结果的后续请求收到通过 transport admission 的响应才转 true。
构造/发送请求不够；超时后仍为 false，但不等于断言远端肯定没收到。
这个布尔字段不证明模型正确理解了结果，也不证明 provider 的内部持久化。
已有 offline witness/mock 的字段语义不重解释。

达到请求 cap 后不再构造第 cap+1 次请求，即使 oracle 此刻为真也不能补造模型完成。
主 context 取消或 deadline 后禁止新调用，已有本地提交保留；本地 HTTP 请求被取消
不保证远端停止计算或计费。所有本地工作同步 join，无 detached provider goroutine。
cleanup 使用独立有界 context，不能复用取消的执行 context 来跳过终态检查。

## 5. 独立工件与验证

| 对象 | 新 format |
|---|---|
| RunConfig | `statetwin.dev/agent-run-live/v1alpha1` |
| Episode | `statetwin.dev/agent-episode-live/v1alpha1` |
| Evidence | `statetwin.dev/agent-evidence-live/v1alpha1` |

Evidence 是严格有界 JSON（32 MiB），字段为 format、plan、startedAt、receipts、
原始已验证 Bundle 的 base64 和 Episode。运行 source 为 provider-live；测试注入路径
为包私有且生成 contract-test。旧 mock reader/comparer MUST 拒绝 live kind。

保留：Task/世界身份、模型请求配置、业务事件、独立 grading、请求 frontiers、
公开计数、内容无关 receipt（sequence、admittedAt、outcome、HTTP status、reportedModel、
可选整数 tokens、cost unknown）。不保留：raw body、私有 reasoning、call/response IDs、
Authorization、header、完整错误正文、API key、私有 oracle 在模型侧的输出。

保留的 Task/oracle 是操作者侧证据，不进入模型请求。合成数据人工审批与有限敏感模式
检测同时适用；模式检测不是对任意 PII/秘密的完整识别。不得把真实业务 trace 当作合成
fixture。检测拒绝时宁可留下 partial/claim，不替换内容后谎称等价 replay。

完成验证 MUST 交叉检查：

1. 计划结构、开始时批准有效；归档后到期不阻碍历史验证。
2. Plan Task/Bundle/model/config 与 Episode definition 完全一致。
3. 正确 live format/source/profile/isolation/runtime identity；snapshot unknown。
4. Completed 且无 failure，receipt 数与 model admissions 相符并在 cap 内。
5. Receipt sequence 无缺口，时间非倒退且在批准窗口内，成功状态/usage 合法。
6. 继承世界 replay 校验：原始 Bundle、初态、全部业务事件、终态、request frontiers、
   计数、授权、错误、提交、delivery 和只读评分逐项一致。
7. Terminal 必须 cleanup complete、closure checked、evidence complete。

验证只重放确定性 world，不重放模型。Unsigned 工件即使完全一致，也不独立证明
Provider 来源、账号身份、实际费用、人类审批或构建来源；不能据 source 字段自动颁发兼容认证。
真实评分失败可有完整证据；执行/传输/隐私/清理失败只能 partial。

## 6. 持久化与恢复边界

沿用 SPEC-0039 的 exclusive claim → sync closure → close → sync pending →
同目录 hard-link 无覆盖 publication。支持失败返回并保留现场，不改写既有 terminal。
不承诺跨文件系统目录项 power-loss durability、每请求 WAL、自动恢复、resume API、
远端 reconciliation 或 exactly-once 外部副作用/计费。原世界和旧 Journal schema 不升级。

后续 [SPEC-0042](SPEC-0042-EVIDENCE-STORAGE-AND-TERMINAL-FAILURES.md) 补充独立终态
context、首因保留、22 个存储注入故障和五个测试子进程退出点；
[SPEC-0043](SPEC-0043-READ-ONLY-EVIDENCE-INSPECTION.md) 提供只读目录检查。
这些验证不发送真实 Provider 请求，也不恢复或重发中断执行。

## 7. Failure matrix 与验收

| 风险 | 行为 / 证据 | 当前验收边界 |
|---|---|---|
| 未批准/过期/未来批准 | 零 POST；不创建有效执行记录 | plan/client/CLI tests |
| 错 Bundle/Task/oracle/model | admission 拒绝；不构造 client | live loop negative tests |
| 缺凭据 | 明确失败；可能保留已 claim 目录，不允许复用 | CLI negative test |
| 请求 cap / 输出参数被修改 | 不发送超限请求或错误配置 | client/loop tests |
| 重定向 / 401 / 429 / 500 | 无重试、无错误正文保留 | transport contract tests |
| malformed/duplicate JSON | 不 dispatch | codec/client tests |
| usage 缺失 / 部分 / 不合法 | unknown / 拒绝 / 拒绝 | client tests |
| 敏感正文 | 停止，正文不保留 | client/loop tests；不宣称完整 DLP |
| 超大响应 / 截断 / 超时 | bounded failure / acceptance unknown | response limit and transport failure tests |
| 批次后半段非法 | 完整 admission 前零 dispatch | inherited codec tests |
| 工具提交后 Provider 超时 | 保留 effect，delivery false，partial | live loop tests |
| cap 用尽但 goal 已达成 | 不冒充模型完成 | live loop budget test |
| 重复/中断 output | 零新 POST，不重用 | single-use tests |
| 换 Task/model/receipt/state | cross-binding 或 replay 失败 | tamper tests |
| 进程退出 / 注入 ENOSPC / link 失败 | 保留 claim/staging；绝不自动重发 | SPEC-0042 共享 writer 的 22 个注入故障和五个子进程退出点；非真实满盘/硬件断电证明 |
| 远端继续执行/计费 | 未知；本地无法担保 cancel/exactly-once | 明确不支持 |
| 真 Provider API/模型版本变化 | 返回可解释失败；独立重新验证 | 尚无实际 live 证据 |

默认 CI 仅使用内存 HTTP test double、合成响应和本地 world，包含无外网 namespace 回归。
不下载新模型、不读取实际 key、不使用测试成功替代 live。当前六任务仅证明 contract
路径与 world oracle；B08 仍需获准账户/模型/预算/六任务报告，B09-live 需预注册可比实验，
B10-external 需真实独立使用反馈。不得声称所有 109 条未提供原文的需求已完成。
