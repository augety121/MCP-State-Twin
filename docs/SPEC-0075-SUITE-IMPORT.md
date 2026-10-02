# SPEC-0075: No-clobber Offline Suite Import

Status: **Accepted — implemented in the bounded offline subset**；依赖 [0074](SPEC-0074-SUITE-EXPORT.md)。

## 1. 接口及准入

命令 `statetwin eval suite-import --root ROOT --archive FILE --out NEW_OUT`。
archive 为 root 下常规文件，out 是完全不存在的新目录，父目录已存在。
两者不得相同、大小写别名或使 archive 位于 out 子树；拒绝任一路径中可见 symlink。
不支持 URL、stdin、覆盖、merge、resume、自动迁移和 runtime compatibility 强制绕过。

先完整准入 0074 归档，冻结 payload，检查全部成员与 manifest/ComparePlan 精确一致，
再对内存只读视图执行现有 audit。单纯解压成功、tar header 有效或清单长度相同都不够。
要求与 export 同样的 clean published/report matched；有效回归可以导入，不能伪称达标。
runtime 不匹配失败，不能改 terminal RuntimeVersion/Revision 后继续验证。

归档 129 MiB、payload 128 MiB、68 成员；原有单 artifact 限额、隐私和 replay 预算
继续生效。读取、解析、核验、写入共享 120 秒；不为解包和 replay 分配独立新期限。
不调用 provider，也不读取 terminal 内嵌 Task.Bundle 对应的源宿主路径。

## 2. 写入生命周期

准入和重放全部通过后才 exclusive Mkdir NEW_OUT；竞争创建失败即拒绝，不接管已有目录。
逐文件使用 exclusive create，写入冻结原始字节，Sync/Close 全检查；创建的 trial 子目录
也须独占。先 suite claim/plan，再各 trial claim/terminal，**最后**用原始 report 字节
按既有 pending -> no-replace publish 协议发布 report.json。不增加 import marker 到
旧 suite 内部，因为旧 inspector 会拒绝未知成员。

这不是整目录原子发布：中途中断目标可能 partial；最后 report 发布前不应被正常 audit
视为 clean published。错误不得触发恢复、覆盖、自动删掉半成品或重写原 metadata。
所有异步工作结束后才退出，不允许取消之后继续发布。最终文件已发布而清理残留失败
必须报告 published_with_residue；下一次导入相同路径始终拒绝。

报告格式 statetwin.dev/suite-import-result/v1alpha1，字段 state、writtenFiles、
writtenBytes、failureCode、provenance:not-proven，状态同 0074；计数只统计成功写并
关闭的 payload，不把 pending 重复计入。发布成功不等于通过候选门禁。
stdout 错误优先，不尝试回滚一个已发布目录。有限错误前缀 IMPORT，分别覆盖 input、
resource、incompatible、dest_exists、write、publish、cleanup；无路径/内容回显。

## 3. 往返与故障验收

- 六任务真实 export -> 新目录 import -> InspectSuite，审计语义相同，所有 payload
  原始 bytes 相同；用字节比较，不另算 hash。原 source 与 archive 不变。
- 有效 negative suite 往返仍 negative；不可改变 report 和模型标签来“修好”。
- 所有坏归档在创建 out 前拒绝：路径穿越、重复/别名、未知成员、长度错、超限、
  额外尾数据、敏感内容、bundle/终态篡改及 runtime 不兼容。
- 中途 Write/Sync/Close/Mkdir/发布/清理和进程退出，目标不错误宣称完整；已存在的
  空目录也拒绝。模拟竞争创建不得覆盖用户文件。
- 取消/超时停在可解释状态，无后台写；重复调用不恢复半成品。平台不支持安全发布
  时明确失败，不运行有覆盖风险的 fallback。

实施建议 archive reader 与 0074 共用，写入使用实例级 FS 接缝，不引入进程全局 hook。
本 Spec 不提供 world-store/Journal 数据库导入、schema migration 或备份恢复承诺。
