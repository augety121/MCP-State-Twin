# SPEC-0050: 比较的 IO、资源和取消边界

Status: accepted by [ADR-0050](ADR-0050-BOUNDED-COMPARISON-EXECUTION.md).

资源版本为 `offline-compare-v1`，通过 JSON/Markdown 的 `verificationProfile` 标识；
独立于 `decisionPolicy`。不改写既有 world ResourceProfile `local-preview-v7` 或其摘要。
后续改变本规范的字节、数量、时间上限必须更新比较资源版本与测试。

## 1. 根目录与成员

计划静态验证后只打开一次操作者选择的 trusted root；失败返回有限
`COMPARE_ROOT_UNAVAILABLE`，而非伪装成一组未开始的运行。逐段 Lstat 已验证的 portable
relative 路径。只有确定缺失的普通目录组件才记 `not_started/missing`；已观察到 symlink、
非目录、无法检查的路径保持 `incomplete/invalid`，不得跟随 symlink 再把 dangling target
当 missing。现有目录中缺 terminal 也是 incomplete。terminal 必须是非 symlink 普通文件。

维持单文件 32 MiB 上限；一次 Compare 累计最多接受 128 MiB terminal 内容进入解码。
按计划顺序、baseline 再 candidate 顺序扣预算。预算跨 pair 累计，超限返回
`COMPARE_RESOURCE_LIMIT`，不丢掉失败项并继续生成“成功”报告。有限读取可能已分配至多
额外一个单文件 buffer；此限制不是 RSS 硬限，也不限制 OS 元数据调用阻塞。

## 2. 取消

一次 Compare 使用父 context 与 120 秒 verification deadline 中较早者。每个 trial 前后、
读取/解码后、重放返回后和最终输出前检查 ctx。父级取消/整次超时必须返回原 context
error 和 nil report；不把它降为某条 `invalid` 后返回一个完成报告。单工件内独立 replay
失败仍按无效证据处理。回放停止协作式执行，无 detached goroutine、provider 请求或恢复。

文件读取/解码/第三方代码并非全部可抢占，不能承诺硬实时取消；可信根下原证据不改变。
上下文、字节预算和缺失目录导致的结果不同有明确测试，不靠真实 sleep 制造时序。

## 3. 验收与兼容

覆盖 pre-canceled、重放期间取消、最后一份重放完成瞬间取消、父截止时间、累计超限、
最大合法预算边界、root 缺失/非目录、普通父目录缺失、文件父组件、内部/外部/dangling
symlink、目录冒充 terminal、terminal 过大、原工件无修改。
Windows 缺 symlink 权限时记录 skip；Linux CI 必须实际运行。无 SHA 文件验收或哈希清单。
仅限实验性 `eval compare`；不修改旧 Scenario、SQLite、Journal 或 MCP 语义。
