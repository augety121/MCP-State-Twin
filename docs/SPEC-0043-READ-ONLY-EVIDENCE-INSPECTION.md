# SPEC-0043: 只读证据目录诊断

Status: accepted via [ADR-0043](ADR-0043-READ-ONLY-EVIDENCE-INSPECTION.md), 2026-09-12.
对象是 SPEC-0039/0041 的 Agent artifact directory，不是旧 Episode Journal 或远端 worker。

## 1. 操作面与输入约束

```text
statetwin eval inspect --root DIR --out RELATIVE_DIRECTORY
```

只读诊断。无 `--repair`、`--resume`、endpoint、credential、删除或批准参数。
只使用本地 root 下的数据；不加载 API key，不调用模型，不新建持久 world 数据库。
可重放 artifact 时使用原有私有内存 world/MCP 路径，结束即关闭。

路径必须为现有 portable relative grammar，逐级拒绝 observed symlink、非目录和越界。
root 无法访问是操作错误，不能报告“尚未开始”。root 有效但指定目录不存在时为 not_started。
父目录缺失亦不得自行创建。

只接受四个固定成员：claim.json、closure.json、terminal.pending.json、terminal.json。
目录读取最多五项即可识别超过四项的非法布局；不递归扫描。
未知文件名、子目录、symlink、设备等均拒绝，报告不回显未知名称。
claim 最大 272 KiB；每个其余成员最大 32 MiB，读取前后做 regular-file/size admission。
总检查 context 30 秒约束 replay 和阶段检查；不能保证中断故障设备的阻塞读取 syscall。

## 2. 报告与退出码

新格式：`statetwin.dev/agent-artifact-inspection/v1alpha1`。

| 字段 | 含义 |
|---|---|
| state | not_started / incomplete_or_running / invalid / published / published_with_residue |
| lane / trialId | 由合法 claim 派生的本地 API/offline 分线与有界 trial ID |
| files | 固定叶名称、absent/valid/invalid/partial_unverified/verified、已知状态枚举、worldReplay |
| evidenceComplete | 当前目录没有已发现冲突，且 published terminal 完整验证通过 |
| stagingResidue | 观察到 closure/pending；不等于运行资源未关闭 |
| resumeAllowed | 始终 false |
| snapshotAtomic | 始终 false |
| providerProvenance | 始终 not-proven |
| problem | 有限规范化诊断码，不包含 raw error/content/path |

正常完成一次诊断可返回 0，即使 not_started 或 incomplete_or_running；这不等于评测 PASS。
invalid 目录在输出诊断后非零退出；路径/读取/context 错误非零退出。
需要自动化判断完整证据时必须读 `evidenceComplete` 或使用专门的 verify 命令；
判断任务成功还必须看已验证 evaluator 结果。诊断不输出原始 task、answer、模型正文或 key。

## 3. 校验顺序

1. 校验目录布局；缺 claim 的非空目录不合法，空目录是 incomplete_or_running。
2. 严格解析 claim：offline RunConfig 或 live Plan。模型/字段的既有 admission 保留；
   过期的 live Plan 仍可作历史检查，不需要当前付费批准。
3. 每个存在的 artifact 严格解析对应 format，并与 claim 的完整配置内容比对。
   不能只看相同 trialId 或只比较某个 digest。
4. 相邻已读 staging/terminal 比较完整语义核心：允许清理状态、证据完成标志和清理
   失败字段发生已定义的后续变化，其他世界/事件/评分/计划/receipt 不可变。
5. pending 与 published 同时存在时，完整语义值必须相同；它们本应来自无覆盖发布。
6. closure 使用封存前 replay 模式；pending/terminal 使用完整终态验证模式。
   partial 只标 partial_unverified，不补造完整结果或通过评分。
7. 只有 terminal.json 完整验证通过且没有目录级冲突才设置 evidenceComplete。
   单独可验证的 pending 仍是 unpublished；不能据此自动 Link/rename 为 terminal。

完整但失败的任务可为 published；存在无害 staging 副本可为 published_with_residue。
staging 内容损坏或与 terminal 冲突时目录整体为 invalid，但不删除或修改任何一份文件。
standalone verify 对单个 artifact 的原有边界不变，目录诊断是额外的跨文件检查。

## 4. 并发、信任和安全

使用静止、受信任的目录；该检查没有跨文件锁/快照，无法可靠区分还在写入和已经崩溃。
因此 incomplete_or_running 不代表“进程已死亡”。并发变更可能产生保守 invalid，
操作者应先停止写入，再做只读检查，不能根据一次诊断自动恢复付费工作。
OpenRoot 不是防御恶意本机写者的完整沙箱。

未知或敏感内容不输出。非法 status 字符串也不能当作错误信息回显。
只检查本地数据及确定性 world，不证明 unsigned artifact 真实出自某个 Provider。
不扫描整个工作区，不生成文件哈希清单，不遍历远端数据库，不接收模型指令。

## 5. 验收

`inspect_test.go`：published 只读、缺失/空/claim/closure/pending/partial 布局、
缺 claim、错 claim、corrupt/冲突 staging、重复副本、未知成员、敏感状态不回显、
live-kind contract 工件、路径越界、超大文件、平台支持时的 symlink 拒绝。
`storage_test.go` 的五个真实进程退出点通过此命令实现的同一检查函数诊断。
CLI quickstart 覆盖正常 published、not_started 和损坏目录非零退出；
已取消的 context 在文件系统准入前被拒绝，不误报为尚未开始。

仍不包含：自动 cleanup、修复、恢复、claim 租约、分布式目录快照、跨主机锁、
真实 Provider 证明、旧 SQLite/Journal 迁移或完整 OS 故障恢复矩阵。
