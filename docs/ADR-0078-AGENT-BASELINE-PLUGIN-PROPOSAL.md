# ADR-0078：以环境插件和评测适配器推进 Agent 基线产品化

- **状态：Accepted for implementation；实际能力以验收台账为准。**
- 日期：2026-10-02。
- 关联：[SPEC-0078](SPEC-0078-AGENT-BASELINE-PLUGIN.md)。
- 接受授权：2026-10-02 用户明确要求“按照上面的spec开始干活，一次性给我搞完”，授权实施全部可执行工作并通过 PR 交付。

## 背景

SPEC-0077 已形成完整离线项目闭环并随 PR #18 进入 main，但真实外部 harness 接入仍没有统一的会话、权限、评分与清理契约。现有 Task authority 位于内部执行路径；直接对外暴露整个 TwinSpec server 不能等同于有约束的 Task 执行。

用户希望项目成为“AGI 的基线插件”。RFC-0001 已将产品定义为 Agent 测试世界基础设施，明确排除 AGI 系统、生产代理和通用编排。因此本提案采用可安装、可重复、可核验的 Agent 基线工具含义，不建立“AGI 认证”的新宣称。

## 接受的决策

1. 保留 Go 世界内核，增加 stdio 环境插件；每个会话只暴露 Task 授权业务工具。
2. 控制和评分交给可信 harness，使用与 MCP 分离的私有生命周期通道。新通道不能列出世界、重置、任意执行或跨会话控制。
3. 首个外部框架参考集成为 Inspect，以独立 Python 包完成任务装载、连接、清理与评分结果投影；不在适配器重写世界语义。
4. 抽取共享 Task authority dispatcher，防止新传输绕开旧约束。旧 generic serve 和旧证据保持契约。
5. 建立声明式 Baseline Pack、冻结实验计划和独立结果封套；新格式不改变现有 offline Suite/Project/Campaign 的含义。
6. 先交付 tools-only 可信宿主模式；有 shell/filesystem 能力的 Agent 需要独立 OS 隔离证据，不能只靠 prompt 或隐藏文件。
7. 任务族、统计判定、真实宿主与发行资格按 N2–N5 推进，分别有退出门槛；mock、live、身份鉴证、L2 和 stable 不互相替代。
8. 保留所有固定分母、原 oracle、错误证据和旧资源上限；不新增文件 hash 清单。

## 考虑过的替代方案

| 方案 | 优点 | 拒绝作为首批默认的原因 |
|---|---|---|
| 只增加 skill/安装说明 | 改动小，易演示 | 没有跨进程会话、权限制约、cleanup 和独立评分闭环 |
| 把 eval/reset/grade 全部做成 MCP 工具 | 一个接口看似简单 | 将判分答案和控制权暴露给被测 Agent，违反 ADR-0002 |
| 先做网页控制台或云多租户 | 容易展示，多人共享 | 不解决基线有效性，提前增加身份、隔离、数据与运维负担 |
| 重写成 Python 通用 Agent harness | 与模型生态接近 | 重复已有框架，丢失 Go 内核与证据价值，扩大维护范围 |
| 先接某个桌面产品的私有配置 | 容易做单产品 demo | 未确定精确目标与隔离能力，容易误把安装成功称为兼容 |
| 任意外部脚本插件 ABI | 扩展自由 | 破坏声明式、可界定和资源有界的产品边界 |
| 一次增加很多领域 | 数量好看 | 现阶段缺口主要是接入与有效性，领域数不等于测量质量 |

## 权威与兼容性影响

本提案不修改 RFC-0001 的硬不变量，不接受完整 remote-staging、production write、任意脚本或通用多 Agent。新增私有 lifecycle 属于可信 operator 控制面，不属于 agent-facing MCP。

接受后将新增 alpha 合约和可选依赖；实施必须保留旧 CLI、解码与文件语义，并给 stdio 单独协议证据。若实际实现要求突破上述边界，必须修订本 ADR 或另立 RFC，不能用“小重构”绕过。

N0 固定 Python 3.12、Inspect 0.3.275、MCP Python 1.26.0、Go SDK 1.8.0；独立 stdio 测试覆盖 2025-11-25 和 2026-07-28，Inspect 使用其已验证的 2025-11-25 profile。依赖安装只发生在项目隔离虚拟环境。私有 finish 补充有界 answer 以支持已有只读 Task，评分仍只由原 oracle 和可重放世界决定。具体执行结果见台账。

## 接受条件

- 维护者明确授权实施并接受 N0/N1 的边界与接口。
- 记录选定 adapter/runtime/protocol 版本以及需要新增的契约。
- 认可逐项验收、原测试回归、exact-head CI 和文档更新是完整交付条件。
- live、产品宿主环境、外部参与者、发布操作分别满足已有授权规则；不能由本提案自动获得。

接受日期：2026-10-02。基线 main `5ad90baa691a8df8ece1942aa74e2d99bcc3502e`。固定 Go 1.26、MCP Go SDK 1.8.0、Python 3.12、Inspect 0.3.275；Windows 私有管道使用 go-winio 0.6.2。实际实施与外部证据分开记录于 [交付台账](maintenance/SPEC-0078-DELIVERY.md)。接受不代表验收通过，也不自动授权付费模型、对外联系或发布。
