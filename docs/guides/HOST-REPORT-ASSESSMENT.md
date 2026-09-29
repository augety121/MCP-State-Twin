# 判断一份 Host 报告是否仍适用于目标配置

本指南是本地、只读、无 provider 请求的声明核对流程，不是实时兼容性认证。
对应 [SPEC-0053](../SPEC-0053-HOST-COMPATIBILITY-TARGET.md) 与
[SPEC-0054](../SPEC-0054-HOST-REPORT-SCOPE-ASSESSMENT.md)。

## 1. 三件事分开判断

| 问题 | 命令/字段 | 不能证明什么 |
|---|---|---|
| 报告结构是否完整且自洽 | `compatibility validate` | 真实运行发生过 |
| 声明是否在时间窗口内 | `assess --at` / `claimTimeEligible` | 仍适用于目标配置 |
| 声明是否与指定配置逐项一致 | `assess --target --at` / `scope` | 目标就是当前真实部署；来源已认证 |

所有结果都保留 `publicationAllowed: false` 和 `provenance: not_verified`。
不要把本工具的零退出码写成“ChatGPT/Claude 已验证兼容”。

## 2. 准备目标

独立整理你**准备验收的配置**，按
[合成格式示例](../../internal/hostcompat/testdata/synthetic-target.yaml) 填写目标：
runtime 版本与 revision、TwinSpec/工具面/初态身份、host/profile/provider/模型、
协议/传输/部署身份、验证过程与 trial 策略和全部预算。

示例中的值是测试占位符，不是项目的真实验证报告。不要直接复制旧报告当作目标并
据此宣称没有变化；程序无法辨别操作者是否提供了真实、独立的目标。
没有完整身份时应停止当前性判断，不使用 `unknown`、缺失预算、零的默认值蒙混过去。
可选 requestedModel/loopback deployment identity 缺失意味着空值，仍参与比较。
旧格式没有独立 adapter/capability 身份，不能用它完成 HostProfile v1alpha2 认证。

## 3. 执行与退出码

```text
statetwin compatibility assess --report report.yaml --target target.yaml --at 2026-09-26T00:00:00Z
statetwin compatibility assess --report report.yaml --target target.yaml --at 2026-09-26T00:00:00Z --require-current
```

使用实际想审计的 UTC 时间替换示例日期。第一条只诊断：即使过期或配置不符，合法的
诊断结果仍为零退出。第二条要求 declared verified、时间合格、目标声明完全匹配；
否则先输出诊断，再以 `HOST_REPORT_NOT_CURRENT` 非零退出。无 target 会在文件读取前失败。

报告的创建/过期时间及 `--at` 都使用 `YYYY-MM-DDTHH:MM:SS[.fraction]Z`；小数秒
可省略，填写时只接受 1–9 位。逗号小数、单数字小时和超过纳秒的精度会直接拒绝，
不会截断后再判断生效或过期。

`--require-fresh` 仍是**仅时间**门槛；即使与 target 不符也可能零退出。
要在 CI 阻断配置变化，使用 `--require-current`。两者同时指定时 current 门槛优先。

例如修改 runtime version 后，输出的关键部分是：

```json
{
  "scopeStatus": "mismatched",
  "scope": {
    "policy": "host-report-scope-v1",
    "mismatches": ["runtime.version"],
    "claimCurrentEligible": false
  }
}
```

这只是完整输出的摘录。所有不一致字段都会按固定顺序列出，不回显原始值、路径或内容。
不存在自动更新、放宽版本范围、忽略模型变化或将旧报告改为成功的行为。

## 4. 无网络的命令演示

在仓库根目录可用现成合成 fixture 验证命令行为，无需生成工件或计算文件哈希：

```text
go run ./cmd/statetwin compatibility assess --report internal/hostcompat/testdata/synthetic-report.yaml --target internal/hostcompat/testdata/synthetic-target.yaml --at 2026-08-24T00:00:00Z --require-current
```

预期 scope matched、claimCurrentEligible true，但 publicationAllowed false。
把 `--at` 改为 `2026-09-26T00:00:00Z`：仍 matched，但 expired、current false、非零退出。
这些 fixture 的 verified 只是测试输入标签，不代表真实 provider 跑过。

## 5. 数值与文件边界

预算、trial index、断言计数必须显式填写整数；小数、科学计数浮点、字符串、布尔、null
和遗漏均拒绝。曾复现 `failed: 0.5` 被 YAML 转成零的问题；现在读取原始类型并拒绝，
不会悄悄取整或改写旧文件。YAML 整数拼写沿用解析库，数值必须可表示为 Go int。

文件来自操作者选择的可信、静止本地目录；报告与 target 读取不是原子快照。
两者均拒绝观察到的 symlink/非普通文件和超过 1 MiB 的内容；文件错误不回显路径。
父路径和并发替换并非完整沙箱。
报告必须显式填写布尔值 `redaction.secretsDetected: false`；遗漏、null、字符串和
YAML 的 `no`/`off` 均拒绝。`true` 也会拒绝，不能将“未提供检查结果”当成“检查通过”。
这些只是声明准入规则，不证明秘密扫描真实执行过，也不会自动修补旧报告。
当前性结果不检查撤销列表、真实上游变化、独立签名或所引用工件的存在与真实性。
这些仍是完整 B15 和 live 验收的前置条件。
