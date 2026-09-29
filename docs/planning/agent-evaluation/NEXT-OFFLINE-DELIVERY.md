# 下一批离线能力设计与实施清单

状态：**Proposal，未授权实施，未实现**。2026-09-29；基于已合并 PR #12。
本页组织 SPEC-0066–0076 共 11 份设计，不改变实现台账，不表示功能数量或完成率。
当前可用能力仍以 [实现台账](../../IMPLEMENTATION-STATUS.md) 为准。

## 三条交付线

| Spec | 用户得到什么 | 接入位置与依赖 | 核心验收反例 |
|---|---|---|---|
| [0066](../../SPEC-0066-INDEPENDENT-TASK-CATALOG.md) | 独立完整 Task 期望 | task_catalog；无新 Spec 前置 | revision 不变但 oracle 变化 |
| [0067](../../SPEC-0067-OFFLINE-SUITE-REVIEW.md) | 执行前发现计划/任务偏差 | PrepareSuite；0066 | 最后一个输入错误时零执行 |
| [0068](../../SPEC-0068-REVIEWED-TASK-ASSESSMENT.md) | 真实重放后核对独立 Task | 内部 definition observer；0066 | 全部重复一致换掉 oracle |
| [0069](../../SPEC-0069-TASK-CASE-MANIFEST.md) | 可版本管理的正反用例 | case_plan；已有 Task/Witness | 错预期、错 ID、未知 assertion |
| [0070](../../SPEC-0070-TASK-CASE-RUNNER.md) | 一次执行整批隔离用例 | RunWitness；0069 | mismatch 继续，基础设施失败停 |
| [0071](../../SPEC-0071-ORACLE-CASE-COVERAGE.md) | 检查 oracle 是否有必要反例 | case_coverage；0070 | constant true 或漏测 assertion |
| [0072](../../SPEC-0072-PACKAGE-REGISTRY-TASK-KIT.md) | 第二领域完整任务闭环 | package-registry 资产；0069–0071 | 有 advisory 仍安装、越权 project |
| [0073](../../SPEC-0073-SUITE-INVENTORY.md) | 一次清点多套本地证据 | suite audit；无新 Spec 前置 | 未回放不能标 verified |
| [0074](../../SPEC-0074-SUITE-EXPORT.md) | 可携带且原字节保真的合成 suite | 冻结只读视图、artifact writer | 审核后换文件、覆盖既有目标 |
| [0075](../../SPEC-0075-SUITE-IMPORT.md) | 隔离还原后可继续核验 | 0074 归档 reader | 恶意 tar、断写、runtime 不兼容 |
| [0076](../../SPEC-0076-RETENTION-PREVIEW.md) | 看清保留依赖与信息不足 | 0073 metadata | 引用环/未知观察不能误标可删除 |

0067 与 0068 都依赖 0066，0068 不信任 0067 的保存报告；0072 的旧 suite 路径不依赖
0066–0068，可独立验证第二领域。0074 不需要 registry，可直接导出显式单个 suite。
0076 不需要归档完成，也不会调用删除。没有隐藏的全组循环依赖。

## 建议实施顺序与完整交付

1. 先接受 [ADR-0066](../../ADR-0066-REVIEWED-TASK-BINDING-PROPOSAL.md)，实现独立绑定线，
   保留旧 suite-assess 行为，并交付 coherent oracle 替换负例。
2. 接受 [ADR-0069](../../ADR-0069-TASK-QUALITY-PROPOSAL.md)，完成用例准入、runner、
   coverage 和 package-registry 六任务资产。不能停在只有格式或只有正例。
3. 接受 [ADR-0073](../../ADR-0073-OFFLINE-EVIDENCE-PORTABILITY-PROPOSAL.md)，完成 inventory、
   export/import 往返和故障矩阵、只读 retention 预览。不能用解压成功替代审计成功。

这是设计依赖顺序，不是工期承诺，也不自动接受三份 ADR。后续若明确授权实施整批，
就按全部 11 份验收交付；若有外部阻塞先做其他工作并说明剩余项，不把部分功能标完成。
新的格式、命令、示例和测试必须在接受后的同批变更中对应，不提前修改公开 current claims。

## 共同边界与测试策略

- 保持 RFC-0001 不变量：无生产写入、无 provider、无 Agent 可见控制面、未知行为明确失败。
- 先全量准入、固定分母、确定性顺序、有限隐私错误；禁止为得到绿色结果弱化 oracle。
- 每操作一个 120 秒预算，模块预算叠加，不在循环中重置；大用例集超时必须显示未执行项。
- 文件比较使用 bytes 或已有 canonical semantic JSON，不新增文件 hash/哈希清单。
  现有 bundle 业务 digest 不删除、不改义。
- 只读命令不写证据；导出/导入只写用户明确指定的新位置，独占创建，不覆盖原文件。
- 真实 MCP/witness/replay 覆盖关键正反链路；解析、排序、计数矩阵用单元测试，
  IO 发布和清理用实例级故障注入，避免把每个排版组合都变成全量重放。
- 修改 Go 时执行 gofmt、vet、针对性/full tests、race 和同提交 CI。现有 Linux CI
  整包 20 分钟是 race instrumentation 预算，不能据此提高业务操作期限。
- 保存格式兼容、stdout 故障、取消、无覆盖、输入不变和敏感哨兵是交付验收，不是可选优化。

## 明确保留的后续门槛

本批不完成 live provider 六任务、外部用户试用、24 任务全集、真实上游 fidelity、
Recorder/L0、HostProfile 来源鉴证、world/Journal schema migration、自动 GC、签名或
全局备份恢复。它们仍按 [backlog](IMPLEMENTATION-BACKLOG.md) 的各自授权与证据门槛推进。
不把缺少这些外部条件当作本批本地设计的阻塞，也不因写了 Spec 就宣称它们已可用。
