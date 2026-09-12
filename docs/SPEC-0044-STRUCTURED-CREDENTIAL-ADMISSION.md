# SPEC-0044: 结构化凭据检测与保存前拒绝

Status: accepted security correction via
[ADR-0044](ADR-0044-STRUCTURED-CREDENTIAL-ADMISSION.md), 2026-09-12.

## 1. 已复现的问题

原 text regex 识别 `api_key=...`，但对 JSON 的 `"api_key": "..."` 以及 escaped key
缺少结构化识别。新 writer refusal test 因此失败。必须修复检测，而不是替换哨兵值
来使测试通过，也不能先写入后清理。

## 2. 分层识别

保留已有 authorization/key-value、已知 token 前缀、private-key、email 模式。
增加常见 quoted JSON credential field 的文本模式，并在输入是合法 JSON 时使用
流式 token decoder 识别解码后的 key/value，不构造第二棵任意 JSON 对象树。

键名忽略大小写与 `_` / `-` 分隔符，有限集合为：authorization、apiKey、accessToken、
refreshToken、clientSecret、privateKey、token、secret、password。
非空凭据值被拒绝；authorization 的字符串/header 字符串数组属于敏感内容，
Task 使用的对象型资源规则数组本身不当作凭据。规则内部仍继续做既有敏感检测。
`tokens`（复数）不是 `token`，usage 整数不应误报成秘密。

字符串值解码后再次做 text-pattern 检查；字符串若承载 JSON，则在有限深度内继续识别，
覆盖 function arguments 一类双层编码内容。结构 stack 不超过 128 层，最多下探四层
嵌入 JSON；超出检测边界时保守拒绝，不能跳过后当作“无敏感内容”。
上层对象的 32 层/文件字节边界继续有效，这不是放宽 Task/Evidence schema 深度。

## 3. 拒绝与日志行为

`ContainsSensitive` 供 artifact/trace/request admission 使用：检测到受限模式则拒绝。
不得修改业务值后继续生成等价 replay；错误正文、私有临时文件和最终工件同样受控。
Evidence writer 检查整个 serializable envelope，然后才打开新文件；Bundle 成员原有
保存前检查保持不变，不把 base64 外壳扫描当作成员检测。

Operational `Redact` 遇到含敏感字段的完整 JSON，整体替换为 `[REDACTED_JSON]`，
避免手工字符串编辑错误留下片段。普通文本错误继续保留非敏感上下文并替换已知模式。
这是日志展示路径，不改变 MCP 业务响应、审计记录或模型输入的业务语义。
遗留敏感工件不会被后台扫描/改写；新 reader/verifier 可以保守拒绝它们。

## 4. 验证

`TestStructuredJSONCredentialAdmission` 覆盖普通 JSON、Unicode escaped key、
Authorization header/数组、refresh-token、数字密码、嵌套 secret 对象、
JSON 字符串中的 JSON、转义等号；断言敏感哨兵不出现在 Redact 结果。
负误报回归覆盖 Task authorization 规则数组、tokens usage、普通元数据和标量数组。
`TestJSONCredentialScanResourceBounds` 覆盖过深结构和嵌入 JSON 的保守拒绝。
storage、六任务 mock/contract、旧 provider 和 CLI 回归必须共同通过。

## 5. 明确限制

不是完整 DLP、任意编码探测、通用 PII 分类、secret manager 或安全擦除。
有限关键词可能产生保守拒绝；未匹配内容也不能据此证明真实数据可公开。
合成来源与人工数据审批仍是必要条件。没有自动生产 trace recorder、任意 base64
解码探测、模型辅助秘密猜测或网络访问。默认测试只用合成哨兵，不使用真实凭据。
