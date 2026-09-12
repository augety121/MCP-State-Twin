# Agent 证据目录：中断后先检查，不自动重跑

适用于实验性 `eval mock` 和 `eval live` 产物，不替代 SQLite 的 `episode inspect`。
本指南所有命令均只读，不读取 Provider key、不请求模型、不修改原世界或证据。

## 1. 选对目录，确保没有正在写入的运行

先查看原命令、退出状态和固定 output 路径。只检查自己控制的本地可信 root，
不要检查由不可信进程持续替换文件的目录。多个文件的观察不是原子快照；
该工具不检测进程存活，不能仅据 `incomplete_or_running` 判断可以杀进程。

离线教程产物：

```powershell
statetwin eval inspect --root examples/issue-tracker --out .statetwin/baseline
```

本地 API bridge 产物（将 ID 换成你的原计划 ID）：

```powershell
statetwin eval inspect --root examples/issue-tracker --out .statetwin/live/my-reviewed-close-01
```

命令只读固定的 claim/closure/pending/terminal 文件，不递归遍历任意文件；
输出有限枚举状态，不打印模型回答、工具参数、原始错误或未知成员名称。

## 2. 按诊断状态处理

| `state` | 可以得出的结论 | 不能得出的结论 |
|---|---|---|
| `not_started` | 在这个 root/out 没发现目标目录 | 不证明别的目录/机器/Provider 没有执行或计费 |
| `incomplete_or_running` | 没有可确认为完整发布证据的 terminal；可能只有 claim/staging 或 partial terminal | 不证明进程已退出，不允许自动重跑或把 pending 改名发布 |
| `published` | 已有完整 terminal，所见 claim 与内容一致，世界 replay 通过且无 staging 残留 | 不等于任务成功、Provider 来源真实、硬件断电后必定持久 |
| `published_with_residue` | 完整 terminal 通过检查，但仍看到一致的 staging 文件 | 不应重发任务，也不能把清理步骤失败抹掉 |
| `invalid` | 发现无法准入的文件、错误状态、内容冲突或验证失败 | 不应自行选择“看起来最好的”工件覆盖其余内容 |

`resumeAllowed` 和 `snapshotAtomic` 始终 false，`providerProvenance` 始终
`not-proven`。`partial_unverified` 表示仅能读取有限内容，未证明完整世界 replay。

`invalid` 会输出有限诊断并非零退出；路径/读取/取消失败也非零退出。其他状态的
退出码 0 **仅表示诊断完成**，不是 CI 验收通过。自动化必须读取完整字段，并根据
相应 evidence verifier 和 Task 评分作决定；不要只检查这条诊断命令的退出码。

## 3. 分开检查证据和任务结果

完整 offline-kind 证据继续使用：

```powershell
statetwin eval verify --root examples/issue-tracker --evidence .statetwin/baseline/terminal.json
```

完整 local-API-kind 证据使用：

```powershell
statetwin eval live-verify --root examples/issue-tracker --evidence .statetwin/live/my-reviewed-close-01/terminal.json
```

任务失败也能有完整有效证据；预算、传输、终态检查或清理失败不能冒充完成。
`failureCode` 保留首个执行失败；`terminalFailureCode` / `cleanupFailureCode`
分别记录后续失败。旧 strict preview reader 可能拒绝新字段；使用与产物相符的
runtime revision 检查，不编辑历史产物来“兼容”。

## 4. 保存现场与后续决策

先保留原计划、退出码和诊断输出；不要删除、覆盖、移动或复用 claim 目录。
现有命令没有“修复”“补发”“确认远端未收费”的能力。遇到写盘/权限问题，在确认
确切路径和操作权限后由维护者处理环境；不要执行针对整个 workspace 的清理命令。

若确需新运行，必须使用新 trial/计划 ID 和新输出目录，并更新预注册比较计划。
付费运行还需重新审阅模型、数据、请求上限、有效期和未知费用风险。旧尝试仍保留在
比较分母/失败记录中，不能只保留最好的一次。`eval compare` 仍是 artifact-level
offline 比较，不会自动改为 directory-level 诊断，也不能比较 live-kind。

已覆盖的 22 个故障注入和五个真实测试子进程退出点见
[SPEC-0042](../SPEC-0042-EVIDENCE-STORAGE-AND-TERMINAL-FAILURES.md)；目录状态契约见
[SPEC-0043](../SPEC-0043-READ-ONLY-EVIDENCE-INSPECTION.md)。这不是满盘、网络文件系统、
OS 崩溃或硬件断电的全组合保证。隐私检测也只是
[SPEC-0044](../SPEC-0044-STRUCTURED-CREDENTIAL-ADMISSION.md) 的有限模式，不接纳真实秘密或个人数据。
