# 有限 L2 与外部验收的输入契约

本文件落实 SPEC-0078 W18 的准备工作。没有实际独立 reference 时，不运行外部服务，不把同一个 TwinSpec 引擎重新包装为 L2。

## 具名产品宿主

提交精确产品/版本/OS、安装方式、公开工具列表、fresh session 方法、额外 filesystem/shell/browser/MCP 的实际禁用证据、退出/中断行为。逐 Task 附观察步骤和原始脱敏会话记录。无法验证 tools-only 则 unverified；有 shell 的 Agent 需要单独 OS 身份/容器，不能挂载 oracle/控制能力/评分目录。

API TTL 30 天、产品 14 天；任一 runtime/head/adapter/protocol/pack/oracle/evidence 绑定变更立即 stale。claim-check 只评估本地声明，不能自己证明产品来源。

## 非作者 first-value

至少三名独立非作者，事前授权邀请和数据记录；使用本地合成数据。每人记录版本、依赖已安装与否、开始/结束时间、check/connect/run/inspect 各步骤、是否在 10 分钟内得到首个可解释结果、失败原因。使用匿名参与者 ID，不记录联系方式；失败同样保留。作者自己重复三次或 mock 用户不能替代。

## 独立 reference 与差分范围

首个候选范围固定 issue-tracker 的 `get_issue`、`close_issue`、`add_comment`，最多三操作；仍须用户提供或批准实际可复位的合成测试服务。契约需要：

1. 独立实现来源、版本、负责人/授权依据；明确未使用本项目 TwinSpec/engine/store 作为实际行为实现。
2. 允许访问的测试 endpoint、账号范围与合成数据；禁生产 trace、生产写入和真实个人数据。真实服务凭据不放进仓库。
3. 可验证 reset 方法：两次 reset 后目标 issue、邻居 issue、评论集合、序列与权限观察一致；reset 仅给可信 operator。
4. 固定 initial state、调用顺序与归一化规则：时间戳只在事前规定允许范围内归一化，不删除错误/权限/重复调用差异。
5. 成功、找不到对象、权限拒绝、重复 close、重复 comment、提交后未知响应分别记录 reference/Twin 的结果与副作用。任何不匹配使该三操作子集失效。
6. 保留明确未覆盖范围：仓库搜索、分页极限、webhook、真实竞态、完整 GitHub 行为等均不据此升级为 L2。

当前未提供独立 reference、reset 授权和观测数据。此合同本身不构成 reset 测试、差分报告或 L2 证据；W18/AB55–AB56 保持 blocked。也不据此阻止不宣称 L2 的 L1 插件候选实现。
