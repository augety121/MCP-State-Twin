# ADR-0073: Bounded Offline Evidence Portability

- Status: **Accepted for the bounded offline subset**
- Date: 2026-09-29
- Accepted: 2026-09-30; maintainer explicitly requested implementation of SPEC-0066–0076
- Scope: B28/B29 的本地合成 suite 证据子集，不接受删除、迁移或 live 归档。
- Contracts: [0073](SPEC-0073-SUITE-INVENTORY.md)、[0074](SPEC-0074-SUITE-EXPORT.md)、
  [0075](SPEC-0075-SUITE-IMPORT.md)、[0076](SPEC-0076-RETENTION-PREVIEW.md)。

## 调查依据

现有 `suite_inspect.go` 只审计显式单个目录；`inspect.go` 拒绝未知 trial 成员，
`storage.go` 已有 exclusive write、sync/close、hard-link 发布和故障测试接缝。
`evidence.go` 的 terminal 内嵌 bundle 并严格绑定当前 runtime 身份。因此完整合成
suite 可以在不依赖原 authoring 文件的前提下重放，但当前没有受限批量清点和搬运契约。

提议显式 registry 清点，随后支持干净已发布 suite 的固定结构归档/导入。
不扫描整个磁盘、不把稀疏失败目录包装为可重放备份；失败样本的诊断保留另有边界。
证据搬运允许有效的负面任务结果，不等同于候选达标。

## 决策提案与取舍

归档使用无压缩受限 tar，固定成员清单，不新增文件 hash/签名/哈希清单。
导出前冻结字节并对该冻结视图审计，导入对归档的封闭视图重放后再写，防止审核 A、
复制 B。完整 128 MiB 输入冻结是明确的内存取舍，不是硬 RSS 保证。
复用既有业务 bundle digest，不增加其他校验体系，也不改旧格式。

导入必须独占新目录、逐成员 no-clobber，最后发布原 report；不承诺整目录原子替换。
中断留下可诊断的不完整目标，绝不自动覆盖/恢复。不同 runtime 不能绕过原 replay
兼容约束，归档不提供历史版本自动迁移或长期可执行性保证。

保留预览只计算显式注册范围的建议和引用保护；没有可靠的全局所有权、活跃租约与
可恢复备份依据，本批不实现删除，更不能把未发现引用等同于可安全删除。

## 实施顺序

0073 是独立只读工作；0074/0075 共用有界归档 reader 和只读文件视图，先完成格式与
畸形输入拒绝，再实现写入生命周期；0076 只复用 registry，不依赖归档已实现。
接受每份 ADR 后仍须代码、故障注入、平台测试、用户指南和同提交 CI 才能声称完成。
本次接受的写路径仅限用户指定的本地新输出位置，禁止覆盖、恢复和删除旧数据。
