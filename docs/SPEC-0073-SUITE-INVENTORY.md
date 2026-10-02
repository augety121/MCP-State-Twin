# SPEC-0073: Explicit Suite Inventory

Status: **Accepted — implemented in the bounded offline subset**；[ADR-0073](ADR-0073-OFFLINE-EVIDENCE-PORTABILITY-PROPOSAL.md)。

## 1. 注册范围与接口

提议 `statetwin eval inventory --root ROOT --registry FILE --mode metadata|replay
--format json|markdown`。root 默认点、format 默认 json；registry/mode 必填。
没有 glob、递归发现、隐式 home/cache 搜索。registry 为严格 JSON 单对象、64 KiB、
深度 32，拒绝未知/重复/大小写别名字段、null 及多文档。

```json
{
  "format": "statetwin.dev/suite-registry/v1alpha1",
  "entries": [
    {"id": "trial-set-a", "out": "runs/a"},
    {"id": "trial-set-b", "out": "runs/b"}
  ]
}
```

全部字段必填，entries 为 1–16，id 为 Task 风格安全标签且唯一，out 为 root 下
portable path；大小写折叠后不得重复或互为祖先，避免双算。只支持 offline suite。
所有输入走 trusted-root regular-file/symlink 拒绝，未知引用不自动创建。

## 2. 两种观察强度

metadata：读取 bounded 目录项、固定元数据和成员大小，不运行 world replay；最多
分类为 missing/present/partial/invalid，reportVerification 固定 not_checked，
comparisonDecision 为空。即使 report 声称成功，也不能填 verified 或达标。
Task/trace 正文不能作为 metadata 摘要输出；文件数量和 bytes 只是本次观察的原始
长度，不是磁盘占用、可回收空间或文件一致性证明。

replay：对每个条目调用既有 suite audit，沿用其 state/reportVerification 和比较
decision；失败结果也保留。不能把无回归提升为候选通过，不自动使用任何 expectation。
所有条目共用 120 秒、累计读取 256 MiB、最多 4096 目录项，既有单 suite 128 MiB
和单文件限制仍生效。不能通过为每个 audit 重置预算规避总限额。

条目缺失/损坏可继续观察其他条目；权限/IO 错误给该条目有限 unreadable 结果并继续。
全局资源耗尽/取消则停止，后续保留 not_checked。不重放 metadata 模式。

## 3. 输出及错误

format `statetwin.dev/suite-inventory/v1alpha1`，mode、completion complete/incomplete、
plannedEntries、observedEntries、entries、provenance:not-proven 必填。
entries 顺序等于 registry；每行 id、observationState、auditState、reportVerification、
comparisonDecision、observedFiles、observedBytes、sizeComplete、problem。
metadata 的 auditState 为空；未能完整枚举时 sizeComplete=false，不外推总量。
observationState 为 observed/missing/unreadable/invalid/not_checked。
不输出用户路径、文件内容、模型文本或文件系统异常字符串；problem 为固定 code。

完整观察即返回零，即便其中存在 invalid；这是清点命令，不是 gate。completion
incomplete 输出已有结果后非零，INV_INCOMPLETE；准入失败为 INV_INPUT_INVALID，
参数为 INV_ARGUMENTS_INVALID，资源为 INV_RESOURCE_LIMIT。报告 1 MiB，stdout 错误
优先；格式说明必须防止调用方用退出零推断全部 suite 合格。

## 4. 验收

真实 published、有效回归、双方失败、partial、损坏、缺目录混合清点；metadata
不触发 replay，replay 结果与单目录 InspectSuite 一致。后项 IO 失败不丢前项，
全局预算/取消保留未观察项，16/17 条目、祖先路径/大小写别名、symlink/nonregular
明确拒绝。输入字节不变，没有目录遍历出注册范围或删除调用。
候选实现 `internal/agenteval/inventory.go`；不引入持久 registry 数据库或后台 watcher。
