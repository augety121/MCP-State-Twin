# ADR-0077：以完整评测项目交付下一批离线能力

- Status: **Proposed — not accepted**。
- Date: 2026-10-02。
- Contract: [SPEC-0077](SPEC-0077-EVALUATION-PROJECT-COMPLETION.md)。
- 本轮授权：调查现状、形成大 Spec；本轮不实施新命令或改变运行环境。

## 现状与问题

2026-10-02 主分支为 `406a775`。上一批实现 #14 合入了设计分支，尚未进入该 main；
必须先补齐主线集成，而非把 merged 状态当作产品已交付。
实现分支已经具备 Task catalog、用例质量检查、suite 审阅/评估及证据搬运，下一轮
价值在于将这些能力组合起来，并补上独立世界绑定、对称任务资产和可读诊断。

## 提议决策

接受时以 W0–W9、A00–A31 为整批交付边界：主线集成、项目配置与 doctor、独立
bundle 内容绑定、冻结输入的完整执行、多组固定分母 campaign、24 个任务/至少 84
个 witness、报告、测量驱动优化、故障兼容矩阵及文档/CI 收尾。
新命令均是本地离线库 API 的有界组合，不是 shell 工作流引擎；旧格式与 policy 不变。
世界等价比较经过验证的 Manifest、成员集合与原始内容，不新增文件 hash 清单。
项目/活动分别共享 120/480 秒预算；campaign 不提高单 suite、Task 或 bundle 上限。

## 取舍

- 选择一个可走通的项目入口，减少手动拼接和跨命令输入变化；保持每个原入口可独立使用。
- 选择新增报告封套，不把新字段塞进已保存的旧 Comparison/SuiteReport。
- 选择逐组串行、有界 aggregate，不用并发 world 或大幅加限额追求漂亮数字。
- 选择明确 quality 重验限制：已保存 qualification 只能 recorded_only，不伪造来源与核验。
- 选择两领域的真实目标扩充与替代合法路径，不增加只有 happy path 的新领域。
- 先记录性能，再保留有收益的操作内缓存；不以复杂缓存本身作为完成指标。

## 保持的边界

RFC-0001 硬不变量、ADR-0039 离线边界、ADR-0040 显式 live 批准与原有资源限制保持。
本稿不批准 provider 消费、自动升级、生产写入、Agent 控制工具、任意脚本、自动 GC、
来源鉴证、数据库迁移、stable release 或 GitHub 自动合并。

## 接受与完成

用户明确要求实施后再将本 ADR 改为 accepted，并记录授权日期；当前不提前修改
IMPLEMENTATION-STATUS 的能力结论。实施可分 PR，但不能将子批完成代替整批完成。
最终交付必须用实际 head 的测试与 CI 证明，并把“已进入 main”与“等待合并”分清。
