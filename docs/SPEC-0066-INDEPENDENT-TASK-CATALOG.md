# SPEC-0066: Independent Task Catalog

Status: **Accepted — implemented in the bounded offline subset**，决策入口为
[ADR-0066](ADR-0066-REVIEWED-TASK-BINDING-PROPOSAL.md)。本文件定义已接受的受限离线契约；实现证据见 IMPLEMENTATION-STATUS.md。

## 1. 目标与边界

为 SPEC-0067/0068 提供独立的完整 Task 期望，补足 SPEC-0063 只绑定计划身份和
预算的边界。目录表示调用方选定的内容，不证明审阅权限、oracle 正确性或预注册。
不读取结果报告来生成目录，不添加 `accept-current`、自动修订或批量批准模式。

## 2. 目录格式

采用严格 JSON 格式，单对象、64 KiB、最大深度 32；所有下列字段必填。
拒绝重复键、未知键、大小写别名、null、多文档、非 JSON 和超限输入。

```json
{
  "format": "statetwin.dev/agent-task-catalog/v1alpha1",
  "profile": "offline-task-binding-v1",
  "tasks": [
    {"taskId": "close-issue", "task": "reviewed/tasks/close-issue.json"}
  ]
}
```

`tasks` 为 1–16 项，taskId 使用现有 Task ID 规则，不重复；task 是相对于命令
`--root` 的 portable path，**不是相对于目录文件的位置**。只允许一层文件引用，
没有 glob、URL、目录扫描、递归 include 或环境变量插值。引用路径按大小写折叠后
必须唯一。目录条目顺序不决定执行顺序；报告按 expectation 首次出现的 Task ID 排列。

目录覆盖集合必须恰好等于独立 expectation 的不同 Task ID 集合：不允许少项、
多项或某个 repeat 独享另一份 Task。缺项/多项是可报告的覆盖不匹配，不是静默过滤。
结构合法的目录可被单独解析；结合 expectation 才判定集合关系。

## 3. 引用 Task 的准入与冻结

1. 使用 `task.ReadFile` 的 trusted-root、regular-file 和可见 symlink 拒绝规则。
   不接受 FIFO、设备、目录或跟随链接来绕过边界；不宣称抵抗可信根被并发替换。
2. 每个 Task 文件最多 256 KiB，严格 JSON、最大深度 32；保持既有 AgentTask
   字段与 `Task.Validate` 语义。此新入口不扩大旧 YAML/JSON Task 的能力。
3. 所有非 optional Task 字段都必须存在；仅 context、faultTool 可省略。
   拒绝 null、重复/未知键和大小写别名。所有 Budgets 字段必须是原始 JSON 整数，
   拒绝 `1.0`、指数和字符串数值；不能依赖解码后的整数值掩盖源类型。
   authority.equals 保持现有受限 scalar 语义，不引入表达式执行。
4. 文件内 Task.ID 必须等于条目 taskId；否则整个输入无效。对原始文本和解码对象
   都执行现有 synthetic/privacy 检查，包括转义后的敏感文本；任何错误不得回显值。
5. 一次读取并冻结所有引用内容。禁止评估每个 trial 时重读目录，也不能把目录快照
   写入结果目录。冻结表示本次操作不再重读，不是原子文件系统快照。
6. post-run 命令要求 expectation、目录以及每个 Task 引用都在 `--out` 子树外，
   所有平台按大小写折叠后的路径比较；拒绝与 out 相同的路径。pre-run 也采用同样
   的预留 out 边界，并额外拒绝引用路径与待运行 suite 的 Task 路径相同。

路径不同不证明物理独立：hard link、目录别名及操作者同时改写两边仍不构成可信审批。
此版本不增加文件身份跟踪；可信、无并发写入的根目录仍是前提。

## 4. 相等规则

比较完整 `task.Task` 的 canonical semantic JSON；比较过程不计算新 digest。
对象键顺序和 JSON 空白不影响相等；数组顺序、所有文本、bundle 路径、revision、
objective、context、tools、authority、budgets、oracle、expectedOutcome、faultTool
均参与比较。不排序 tools/oracle、不规约 CEL、不去掉路径、不把 revision 当替代品。
可省略的空 context/faultTool 与其显式空字符串按现有 Task 序列化视为相同。
JSON 字符串转义等价按解码值处理。不存在“语义相近”或自定义忽略字段。

新 Task 字段默认参与相等；如果未来字段改变准入或解释，必须审阅目录 profile
兼容性，不能通过手写字段白名单忽略它。目录 admission 不编译 oracle；运行前的
既有 Preflight 与运行后的重放分别核验与实际 bundle 的兼容性。

与 reviewed Task 不同只输出 taskId/trialId 和有限差异类别：
`identity`、`goal`、`tools`、`authority`、`budgets`、`oracle`、`other`，按此顺序去重。
identity 含版本/格式/id/revision/domain/bundle/mode；goal 含 objective/context/
expectedOutcome；tools 含 tools/faultTool。未归类的新字段归 other，仍必须拒绝相等。
类别只用于诊断，不替代完整相等判断。禁止渲染原始表达式、目标文本、文件路径或值差异。

## 5. 资源与错误

| 资源 | 上限/处理 |
|---|---|
| 目录 | 64 KiB、16 项、深度 32 |
| 每个引用 Task | 原始输入及保留 canonical 表示各 256 KiB |
| 引用累计输入 | 4 MiB，按实际读取累计；不得通过别名绕过 |
| 保留 Task canonical 表示 | 4 MiB，超限前拒绝，不截断 |
| 运行预算 | 使用调用方同一个 120 秒 context，目录加载计入；不能每阶段重置 |

逐文件检查取消。一次操作最多保留每个 Task ID 一份独立 Task；SPEC-0064 的
4 MiB observed reference 预算另算，不能借用。两者与已有 suite/evidence IO 上限
同时生效，不声称上述数值是硬 RSS 限额。编码临时空间受单 Task 边界约束。

有限错误：`TASK_CATALOG_INVALID`（含格式、结构、Task 不合法和读取失败）、
`TASK_CATALOG_PATH_INVALID`、`TASK_CATALOG_RESOURCE_LIMIT`、`DATA_POLICY_REJECTED`。
取消/期限保留 context 原因。CLI 仅打印有限错误码，不包含路径、内容或底层 OS 错误。
集合缺项/多项进入结构化绑定结果；不能靠空目录伪造“没有需要检查的任务”。

## 6. 实施验收

| 场景 | 必须得到的结果 |
|---|---|
| 同 Task、键重排/空白/等价转义、optional 空值 | 相等 |
| revision 不变而修改 goal/oracle/authority/预算 | 不同，类别准确且无原文 |
| 改数组顺序、bundle 路径或 expectedOutcome | 不同，不做隐式归一化 |
| 改未知的新字段 | 准入拒绝或升级后的完整比较捕获，不静默忽略 |
| 目录 ID 与文件 ID 不符、重复 ID/引用 | 整体准入拒绝 |
| 缺项、多项、条目重排 | 前两者覆盖不匹配；后者不改变报告顺序 |
| null/浮点预算/缺字段/重复键/别名/深层/超限 | 有限拒绝，不创建输出 |
| 转义敏感哨兵、路径内含敏感文本 | 错误不回显；无内容落盘 |
| out 子树、跨平台大小写、symlink、nonregular | 拒绝，输入不变 |
| 冻结后改磁盘文件、取消、累计上限边界 | 不重读；取消/超限明确失败 |

实施建议落点：`internal/agenteval/task_catalog.go` 及对应测试；复用 task/logging/
strict decoder，不修改既有 Task 解码兼容性。对应实现与测试见实现台账。
