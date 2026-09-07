# SPEC-0040: 本地 API 运行计划、授权与预算

Status: accepted experimental subset via [ADR-0040](ADR-0040-OPT-IN-LOCAL-API-PLAN.md).
Date: 2026-09-08. 实现证据以 [台账](IMPLEMENTATION-STATUS.md) 为准；
本规范不声称已经执行真实 API，也不改变本地 v0.1 发行门槛。

## 1. 目标、范围与信任边界

让操作者能够在**执行之前**审阅一次模型运行的任务、数据、模型、最大请求数、
有效期和未知费用风险。只接受 OpenAI Responses 非流式函数调用本地 bridge：
模型请求可以出网；业务工具只访问本次私有 Twin world。Agent 看不到 oracle、
世界全量状态、故障控制、API 凭据或证据目录。

这不是 ChatGPT/Codex 产品宿主、原生 remote MCP、Anthropic adapter、任意代理服务、
云端 worker 或任意脚本执行接口。已有 `eval mock`、`task witness`、Scenario、
Journal 和 hermetic worker 的协议与默认网络边界保持不变。

本地操作者、构建产物、系统证书/DNS 和 artifact root 属于受信任边界。
不防御能修改本进程/凭据/计划文件的本机攻击者，不将布尔批准当作密码学授权。

## 2. 计划格式与字段

独立 JSON format：`statetwin.dev/agent-live-plan/v1alpha1`。
最大 272 KiB、深度 32；重复字段、未知字段、尾随文档和非法数值 MUST 拒绝。
Task 继续遵循 SPEC-0038，授权标量数字按统一数字语义校验。

| 字段 | 必须满足的约束 |
|---|---|
| `format` | 上述固定格式，不与 offline run config 混用 |
| `id` | 小写字母起始，后跟至多 63 个小写字母/数字/连字符 |
| `provider` | `openai` |
| `profile` | `openai-responses-local-bridge-v1alpha1` |
| `model` | 操作者明确指定，1–128 个允许的 ASCII 标识字符，不允许 `mock-` 前缀 |
| `task` | 完整、有效的独立 AgentTask；包含目标、授权、预算和私有 oracle |
| `bundleDigest` | 已验证 TwinBundle 的现有业务身份；不能只检查字符串形状 |
| `issuedAt` / `expiresAt` | 带时区 RFC3339Nano，后者严格大于前者，窗口不超过 24 小时 |
| `maxRequests` | 1 至 Task 的 model-request 上限，且受 Task 硬上限 16 约束 |
| `maxOutputTokens` | 1–8192，作为每次请求的 API 参数；不等于货币上限 |
| `costPolicy` | `request-capped-cost-unknown` |
| `approved` | 操作者审阅本次执行后明确设为 true |
| `syntheticDataApproved` | 操作者审阅发送和保留的数据后明确设为 true |
| `unknownCostApproved` | 操作者接受请求有费用且无法由本地精确预估后明确设为 true |

模型 grammar 只是本地 admission，不证明模型存在、账户有权限、支持这些参数或
返回固定 snapshot。禁止添加“最新模型”默认值或把 API 文档当作 live 测试。

任务与 Bundle 必须在创建 world、构造 provider client、写证据之前完成 admission：
校验 Bundle 内容及身份，工具 surface、资源授权、CEL oracle、Task budgets，以及
provider function projection。运行时冻结计划和任务副本，外部调用者之后修改原对象
不能改变已接受的上限。计划是这次运行的 Task 真值来源，不额外偷偷加载另一张卡。

## 3. CLI 合约

| 命令 | 行为 | 是否读密钥 / 发送请求 |
|---|---|---|
| `eval live-plan` | 读取本地 Task/Bundle；要求 id/model/请求数/输出上限/有效期；输出未批准 JSON | 否 / 否 |
| `eval live-preflight` | 检查计划与 Bundle 的本地结构和绑定；报告批准此刻是否有效 | 否 / 否 |
| `eval live` | 经明确授权后读取 `OPENAI_API_KEY`，执行一次计划 | 是 / 最多计划上限 |
| `eval live-verify` | 严格读取 live-kind 工件并重放本地 world | 否 / 否 |

`live-preflight` 的退出成功只表示本地结构合法，`approvalCurrentlyValid:false`
可以与结构成功同时存在。它不测试网络、账户、价格或模型可用性。
`live` MUST 再校验批准和 `--allow-live`，不能依赖一次过去的 preflight。

命令不接受自定义 endpoint、proxy、credential JSON、provider key 命令行参数、
复用旧 output、自动批准、自动生成新的 retry ID 或自动选择模型。
计划和工件相对路径继续遵守受信任 root 的有界读取及 observed-symlink 拒绝规则。

## 4. 授权时序和凭据

结构校验 → 当前批准窗口 → 固定计划目录独占 claim → client/codec/world →
每次 POST 前再次校验批准、context、请求形状和计数。

有效窗口是 `[issuedAt, expiresAt)`。到期拒绝**新请求**；不声称撤销已经被远端接受
的请求。时钟调整可能导致保守拒绝或工件无法完成时间交叉验证，不能因此降低校验。
API key 只从固定环境变量读取，绝不复制进计划、日志、工件、错误正文或模型输入。
client 停止后丢弃自己的 key 引用；不声称 Go 字符串、进程环境或 GC 内存已安全擦除。

## 5. 预算语义

请求 cap 在单个 Client 中串行计数，每次进入实际 HTTP 调用前消耗一次。
HTTP 失败、网络超时或无法确认远端接受情况也消耗该次，不退款，不重试。
Task 自身的 tool attempt、Episode deadline、响应/内容字节和清理约束继续生效。
provider 请求 deadline 不超过 Task 的 request seconds，并继承更短的 Episode context。

模型请求 admission 数与 actual HTTP attempts 不强行混成一个数字：编码成功之后仍
可能因授权到期而零 POST；partial report 保留各自计数。完成工件必须逐项一致。
未报告 token usage 使用缺省/unknown；若报告 usage，三个整数必须完整、非负、有界，
且 input + output = total。费用始终 unknown，不能由缺失 usage 算作零费用。

本地 request cap 不是账户级 spending limit。输出上限是发送给 Provider 的限制参数；
不能证明服务端正确执行，也没有估算或约束输入费用、缓存、税费、其他进程消耗。
操作者仍需独立设置和检查其账户控制；本工具不替账户购买、充值或创建凭据。

## 6. 一次性 claim 与失败

输出固定为 `.statetwin/live/<plan-id>`，父目录必须已存在且受信任。
已有目录一律拒绝，包括 complete、partial、只有 claim 的中断目录。
没有覆盖、自动恢复、重命名重试、复制凭据或按计划 ID 幂等重发 Provider 请求。
拷贝计划到另一 root、删除目录或更换 ID 属于新的人工操作，不能称为跨账户去重。

进程崩溃后只能说该计划可能已发送 0 至 cap 次；没有逐请求持久日志时不能补造精确数。
保留现场供检查，下一次真实尝试必须重新评审未知副作用、费用与预算。

## 7. 验证与未关闭门槛

已实现的可执行 contract tests：`internal/agentapi/client_test.go`、
`internal/agenteval/live_test.go` 和 CLI live-plan/preflight tests。
覆盖完整/缺失批准、过期、未来窗口、错误模型、超预算、开放字段、Bundle/Task 不匹配、
零请求负例和完成/中断目录不可复用。默认测试无真实凭据、无外部请求。

仍需要外部条件：实际账户/模型选择、操作者费用授权、六任务真实报告及独立审阅。
更广泛的 monetary governor、多用户授权、签名计划、原生 MCP/product-host 和 remote
security 不在这个 accepted profile 内。通过测试不将 B08、B09-live 或 B10-external 关闭。
