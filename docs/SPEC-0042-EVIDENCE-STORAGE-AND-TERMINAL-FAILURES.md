# SPEC-0042: 证据保存、终态失败与中断验收

Status: accepted experimental subset via
[ADR-0042](ADR-0042-EVIDENCE-STORAGE-FAILURE-BOUNDARY.md), 2026-09-12.
本规范补齐 AE-014/019/020 的本地子集；不是完整 storage compatibility、HA 或自动恢复证明。

## 1. 需要修复的问题

原 Agent evidence writer 没有逐操作的失败注入入口，无法在不破坏真实磁盘的前提下
验证 write/sync/close/link/remove 的每种失败。终态路径还有两个具体问题：

1. world 检查使用独立 context，但 replay closure 仍使用可能已取消的执行 context，
   会在执行已经完成后丢失本可保存的有效闭包。
2. 终态检查/清理错误会覆盖原始 Provider、预算或取消错误，丢失先发生的失败原因。

新增存储测试接缝和错误字段必须保留旧的世界、MCP、模型和评分语义。
不得通过放松 verifier 或重跑模型来使证据“完成”。

## 2. 保存协议

固定文件：`claim.json`、`closure.json`、`terminal.pending.json`、`terminal.json`。
只有受信任操作者能提供根目录/相对输出目录；Agent 无对应控制能力。

| 阶段 | 必须完成的动作 | 失败后的状态 |
|---|---|---|
| admission | Task/Bundle、相对路径、序列化大小与敏感政策检查 | 不执行、不创建含拒绝内容的文件 |
| claim | 检查父路径；独占新目录；独占文件、完整写入、Sync、Close | 不进入 runner；已创建的目录保留且禁止复用 |
| execution | 一个私有 world/session，原有预算/权限/MCP 规则 | 不重复执行以补证据 |
| terminal inspection | 使用独立有界 context 读取终态、只读评分 | 明确 partial，保留原始失败 |
| closure | 独立重放检查；写入并同步完整闭包 | 原 world 随退出清理；仅保留已有 claim/staging，不伪造完整报告 |
| runtime cleanup | 关闭拥有的 world/session | 清理失败单列，不覆盖原始执行错误 |
| pending | 独占创建、完整写入、Sync、Close | 不发布；保留 closure 和已创建的 pending |
| publish | 同目录无覆盖 Link 为 terminal | 无 rename/copy 覆盖 fallback；错误可能留下已发布文件，先检查 |
| staging cleanup | 仅删除自己固定的 pending 和 closure 叶文件 | 返回清理错误；不回滚/删除已发布 terminal |

新文件 mode 0600，新目录 0700；Windows 的 ACL 语义不等同 POSIX mode，不能据此
宣称跨平台强权限隔离。root/父目录仍必须由操作者控制。
内容检查在打开每个文件之前完成，最大编码工件 32 MiB。检查完整 envelope，不能只
检查 Episode 后将 wrapper 中的额外内容未经检查写出。

write 返回短计数，即便 error 为 nil，也 MUST 作为失败；不得继续 Sync 后当成完整文件。
即使 Write/Sync 失败也尝试 Close 这个已打开的文件；不通过忽略错误继续下一阶段。
每次运行使用私有 filesystem interface 实例，没有全局故障变量、命令行注入开关或远端配置。

## 3. 已知、未知和不允许的推断

- `terminal.json` 存在不等于内容合法、任务成功或全部清理完成。
- Link 返回错误时不能假定 terminal 必定不存在；模拟的“操作已生效后返回错误”必须测试。
- remove 返回错误时不能假定 staging 必定仍存在；诊断依实际目录与内容。
- claim 写入失败后，目录仍不可复用；没有首次重试免费、隐式补偿或模型 exactly-once。
- 不能通过删除 claim、复用 trial ID 或覆盖 terminal 来“修复”失败。
- 合法 `task_failed`/`policy_violation` 可以有完整证据；execution/inspection/cleanup 失败
  只能留下 partial，不能被 goal 已满足覆盖。

## 4. 终态 context 与错误字段

在停止模型/工具 admission 后，终态读取、评分和 replay closure 共用一个独立的
`cleanupSeconds` context，而不是已取消的执行 context。已经完成的执行若在封存阶段
收到迟到取消，可以继续保存其确定闭包；这不允许再发模型请求或继续原执行的工具调用。
独立 verifier 仍可在新的内存世界重放已经记录的工具事件，不修改原世界。

| 字段 | 语义 |
|---|---|
| `executionStatus` | 原执行阶段的结果，不因后续错误伪造成功 |
| `failureCode` | 第一项执行错误；若执行无错则为首先发生的终态/清理错误 |
| `terminalFailureCode` | 仅终态读取/评分失败时为 `TERMINAL_INSPECTION_FAILED` |
| `cleanupFailureCode` | 仅 world/session 清理失败时为 `CLEANUP_FAILED` |
| `cleanupStatus` | pending / complete / failed，原有独立维度 |

两个新字段默认省略；旧工件无需补写。完整验证必须拒绝非空的新失败字段，不能只查
旧 `failureCode`。旧严格 reader 可能拒绝带新字段的失败工件；这是明确的 preview
兼容边界，不做静默字段删除或格式 downgrade。

暂存失败也必须关闭拥有的 world/session。若无法写出安全终态，返回规范化错误并保留
已落盘现场，不输出原始磁盘路径、系统错误正文或模型内容。
Go context 约束计算/状态读取/replay，但不能强制中断阻塞的 OS Sync/Close/Link；
因此不能声称所有设备故障都在 cleanupSeconds 内收敛。

## 5. 可执行故障矩阵

`TestEvidenceStorageFailureMatrix` 共 22 个注入结果：

- mkdir 失败；
- claim、closure、pending 各自 open/write/short-write/sync/close 失败（15 项）；
- Link 生效前、生效后返回失败（2 项）；
- 删除 pending/closure，各自生效前、生效后返回失败（4 项）。

测试使用真实临时目录和有界操作包装器模拟 ENOSPC 等失败，不填满使用者磁盘。
不将“包装器返回 ENOSPC”写成真实磁盘耗尽测试。
断言包括：claim 完成前 runner 零调用；后续不重复执行；既有 artifact 不覆盖；
已发布内容仍可验证；未发布不晋级；失败目录拒绝复用；原始 filesystem 错误不泄漏。

`TestEvidenceProcessExitCutPoints` 启动受控测试子进程，在以下真实操作后退出：

1. claim Sync；2. closure Sync；3. pending Sync；4. Link；5. pending Remove。

父进程读取现场并验证状态，检查新运行不会复用该目录。子进程只访问父测试拥有的
临时目录；不会结束用户应用进程。临时产物由父测试统一清理。

`terminal_test.go` 覆盖迟到取消仍可保存已完成闭包、失败原因保留、暂存失败关闭
world、新失败字段不能绕过完整验证。默认验证无真实 Provider、无外部副作用。

## 6. 仍不承诺

没有真实硬件断电/控制器缓存故障、全文件系统持久性、目录 fsync 承诺、加密、备份恢复、
分布式事务、远端 reconciliation、自动恢复或 exactly-once 计费保证。
世界 SQLite schema 4 和 Journal schema 2 不变；本次文件工件测试不能替它们关闭
未验收的 storage gate。独立只读诊断见 [SPEC-0043](SPEC-0043-READ-ONLY-EVIDENCE-INSPECTION.md)。
