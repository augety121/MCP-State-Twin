# SPEC-0074: Frozen Offline Suite Export

Status: **Proposal — not accepted or implemented**；[ADR-0073](ADR-0073-OFFLINE-EVIDENCE-PORTABILITY-PROPOSAL.md)。

## 1. 范围与前置条件

拟议 `statetwin eval suite-export --root ROOT --out SUITE --archive DEST`，root 默认点，
其余必填且 portable，DEST 及 DEST.pending 必须在 SUITE 子树外，父目录须已存在。
输入只支持现有 offline suite。完整加载最多 128 MiB 的固定成员，冻结后只读审计
该同一视图；不能审核后重新读取活动目录来打包。所有操作共享 120 秒。

可导出条件：audit state published、reportVerification matched、无 staging residue、
所有计划 trial 证据完整。**不要求 no_regression 或候选通过**，正确的负面结果也值得
保留。残留 closure.json/terminal.pending.json/report.pending.json、未知成员、partial、
无效或不兼容 runtime 均拒绝，不删除它们来让导出成功。不接 API/live evidence。

## 2. 受限归档格式

无压缩 USTAR tar，最多 68 个常规文件，archive 最多 129 MiB、payload 最多 128 MiB。
拒绝 PAX/GNU 扩展、link、device、目录成员、绝对路径、反斜杠、点段、大小写冲突、
重复成员以及 tar 结束标记后的非零数据。不能只信 header size，流读取同样计预算。

成员为 manifest.json 和 payload 下原样保存的 suite 文件：根 claim.json、plan.json、
report.json，加每个 baseline-NN/candidate-NN 的 claim.json、terminal.json。
16 pairs 对应 3+32*2+1=68 个成员；计划少于 16 对时只能包含实际计划成员。
无 authoring Task、mock 响应、外部 expectation/目录、源绝对路径、mtime 或用户名。
terminal 已内嵌 bundle；不另行扫描其 Task.Bundle 路径，不从宿主补齐依赖。

manifest 严格 JSON、64 KiB、深度 32：format 为
statetwin.dev/offline-suite-archive/v1alpha1，profile 为 offline-suite-copy-v1，
members 为按 path 字典序的 `{path, bytes}` 数组，均必填。数组列出所有 payload 成员，
不包含 manifest 自身；bytes 为原始非负整数且等于实际字节长度。无摘要、文件 hash、
签名、导出时间和可执行指令。清单是结构声明，不是防篡改或来源证明。

写入顺序 manifest 在前、payload 按 path 字典序；固定 mode=0600、uid/gid/mtime=0、
uname/gname 为空，使用常规 USTAR header 和两个结束块，不带额外 padding 内容。
源 JSON 字节保持不变，不重新排版。相同冻结输入应输出相同归档字节，测试直接比 bytes。

## 3. 发布及错误

先完成全部准入、隐私检查和冻结视图 audit，再独占创建 DEST.pending；若 DEST 或
pending 已存在，一字节也不覆盖。依次写、Sync、Close，使用平台支持的原子 no-replace
发布（可复用 hard-link 方案），成功后移除本次 pending。能力不可用则明确失败，
不得降级为会覆盖的 rename。禁止“存在即先删”。

失败保留自己的 pending 供诊断；若最终文件已发布但移除 pending 失败，返回
published_with_residue，不能当作完整成功。不得删除来源或用户原有文件。
报告仅 format=statetwin.dev/suite-export-result/v1alpha1、state、files、bytes、
failureCode、provenance:not-proven；state 为 refused/failed/published_with_residue/
published，后者才退出零。stdout 失败仍非零；磁盘状态不能因打印失败被回滚覆盖。
错误使用 EXPORT_INPUT_INVALID、EXPORT_NOT_ELIGIBLE、EXPORT_RESOURCE_LIMIT、
EXPORT_DEST_EXISTS、EXPORT_WRITE_FAILED、EXPORT_PUBLISH_FAILED、EXPORT_CLEANUP_FAILED。

## 4. 验收

正面/负面但完整 suite 均可导出，外部期望不打入包；冻结后修改原目录不影响输出。
遍历、link、未知/遗漏成员、错误 size、超限、敏感哨兵、残留和不兼容 runtime 拒绝。
Write/Sync/Close/publish/remove 每步注入失败，验证旧 DEST 从不变化、状态忠实。
相同输入字节确定性、跨 Windows/Linux/macOS 路径约束、取消及 stdout 故障。
实施建议共享 bounded readonly view，让 audit 验证冻结文件，保留现有公开 API。
