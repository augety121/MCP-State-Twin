# SPEC-0076: Explicit Retention Preview Without Deletion

Status: **Accepted — implemented in the bounded offline subset**；依赖 [0073](SPEC-0073-SUITE-INVENTORY.md)。

## 1. 只读范围

命令 `statetwin eval retention-preview --root ROOT --registry FILE --policy FILE
--format json|markdown`。root/format 默认同 inventory，另两个文件必填。
不提供 apply、delete、force、prune 或后台周期任务；既有 world/Journal/suite 原样保留。
目标是让用户看见显式注册范围里哪些需要保护、哪些信息不足，而非自动判断可删除。

policy 严格 JSON 单对象，64 KiB、深度 32，format 为
statetwin.dev/retention-intent/v1alpha1，entries 必填，1–16 条：

```json
{
  "format": "statetwin.dev/retention-intent/v1alpha1",
  "entries": [{"id": "trial-set-a", "intent": "keep", "active": false,
               "references": []}]
}
```

id 与 registry 集合精确一致，intent 为 keep/review，active 原始 JSON bool 必填，
references 是已注册 ID 的集合；每项最多 16 个，不重复，禁止 dangling reference。
允许环并有界遍历。引用表示“保留当前项还需要目标项”；不从文件名、mtime、
wall clock 或 trace 文本推导 owner/活跃/时间，不把声明 active=false 当作租约证明。

## 2. 保护算法

一次 metadata inventory 先覆盖全部注册项，不进行 replay 或内容搬运。保护起点为
intent keep、声明 active=true，以及缺失/损坏/不可读/观察不完整的项。
沿 references 传递保护，最多 16 节点/256 边，不因循环超时或丢弃引用。
每项结果为 protected、review_candidate、unknown；异常观察项自身 unknown，但其
依赖仍受保护。正常且被保护项 protected，剩余正常项才是 review_candidate。
unknown 优先于该项其他显示状态，原因列表仍保留 keep/active/reference 事实。

未被保护的引用环整体可以列 review_candidate，但明确必须作为依赖组一并人工评估，
不能暗示任一节点可独立删除。报告附有序组成员 ID，按 registry 首现顺序。
没有闭合的注册范围证明，所有 review_candidate 都仍是建议，不输出 safeToDelete=true。

## 3. 输出、资源与错误

format statetwin.dev/retention-preview/v1alpha1，completion、entries、dependencyGroups、
deletionPerformed:false、globalReferenceCoverage:unknown、provenance:not-proven 必填。
entries 含 id、status、reasons（keep_requested/active_declared/observation_unknown/
referenced_by_protected/not_protected_in_registry），引用原因列安全 ID，不暴露路径。
观察字节数若展示，必须保留 sizeComplete；不汇总为“可释放空间”。

全部读取/元数据观察共享 120 秒，沿用 inventory 累计预算、4096 目录项和 1 MiB
报告上限。全量完成返回零，即便 unknown；完成度不足/取消/超限非零，并保留已知
诊断。policy 错为 RETENTION_POLICY_INVALID，参数/IO 使用有限错误，不打印输入。
无可执行删除计划、删除 token 或可重放写指令；stdout 失败不会产生任何清理。

## 4. 验收与不完成项

覆盖单项保留、活跃项、链式保护、多入口、引用环、自引用、未注册引用、空/多/少
policy 项、大小写别名、损坏目录和总预算耗尽。验证输入与所有注册目录 bytes 不变，
可观测 FS seam 必须证明未调用任何 Remove/Rename/Write；metadata 不触发 replay。
同输入和目录状态输出稳定，无现在时间参与决策。

这只是 B28 设计前置子集。真实 GC 还需要独立所有权、并发活跃锁/租约、完整引用图、
保留政策和恢复证据的接受决策；不能因本预览完成便把 B28/B29 标为完成。
