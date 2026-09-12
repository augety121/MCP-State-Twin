# SPEC-0045: 发布计划、版本与说明准入

Status: accepted via [ADR-0045](ADR-0045-REVIEWED-RELEASE-PLAN.md), 2026-09-12.
实施范围为 B11/B32；不扩大 stable 的功能承诺。

## 1. 入口与路径

维护者只读命令：

```text
go run -p 1 ./cmd/releasecheck --root . --tag v0.1.0-alpha.N
```

`N` 为占位，不能直接执行。命令不访问 GitHub、不读取 token、不写文件、不打 tag。
默认 JSON 输出；`--format github` 只向 stdout 输出有限安全的 workflow output 行。
不使用 eval/source 执行配置。未知参数、位置参数或输出格式失败。

版本 tag 长度最多 128 ASCII bytes，必须为 `vMAJOR.MINOR.PATCH[-PRERELEASE]`。
核心数字及纯数字 prerelease component 不得有前导零；prerelease component 非空且
只含 ASCII 字母/数字/连字符。拒绝空组件、空白、路径、控制字符、非 ASCII、额外核心段。
本发布 profile **不支持 `+build` metadata**，这是本项目对 SemVer 的显式子集，
不是声称 SemVer 禁止它。不解析整数到机器 int，避免大数字溢出。

只读取 root 下固定 `releases/<tag>.json` 和 `releases/<tag>.md`；不从 plan 接受
任意 notes 路径。使用可信本地 root，拒绝 observed symlink、非 regular 文件及越界。
plan ≤16 KiB、深度 ≤8、严格单 JSON object；notes ≤64 KiB、合法 UTF-8。
读取有 max+1 边界；不扫描整个仓库。

## 2. Plan v1alpha1

| 字段 | 必须条件 |
|---|---|
| format | `statetwin.dev/release-plan/v1alpha1` |
| tag | 与命令 tag 完全一致 |
| profile | `local-core-v0.1`，目前唯一支持值，tag 必须为 v0.1.x |
| channel | `prerelease` 或 `stable`，与 tag 是否含 prerelease 一致 |
| claimsReviewed | true，维护者声明公共能力/四语 README 已复核 |
| compatibilityReviewed | true，维护者声明独立 storage/report/protocol 边界已复核 |
| stableGatesReviewed | stable 必须 true；prerelease 必须 false，避免误暗示稳定资格 |

布尔值必须是 JSON boolean，不能接受字符串 `"true"`。未知字段、重复 key、YAML、
多文档、错误类型、过深/过大/敏感模式全部拒绝。未批准模板保持 false，不自动补 true。

notes 必須包含且每个恰好一次、按以下顺序的非空二级章节：Scope、Verified changes、
Compatibility and migration、Security and hermeticity、Known limitations / deferred proposals、
Evidence、Contributors。章节至少有非空正文；代码围栏里的同名标题不算章节。
只接受这七个二级章节；更细分内容可用三级标题。该有界结构检查不是完整 Markdown
渲染器，也不能判断章节正文是否实质完成了人工审核。
拒绝模板标记 `REVIEW_REQUIRED` 和有限敏感模式；这只是结构与有限隐私检查，
不证明正文事实正确、无任意秘密、真实贡献者身份或审核签名。

## 3. 报告与审批边界

报告只含 tag、version、profile、prerelease、固定 notes 路径、draft=true、latest=false、
`approvalEvidence:repository-declaration-not-attestation`。不输出 notes 正文、原始解析
错误或机器路径。JSON 输出与 GitHub 行格式来自同一已验证对象。
只有 repository plan 准入不构成发布许可；还必须通过 SPEC-0046 的完整 CI。

v0.1 仍排除 Provider/product live、remote production、recorder/L0、L2/L3、外部副作用
exactly-once 和实验性 Agent eval 的稳定性承诺。必须在 notes 中人工写明实际范围，
不能靠布尔字段自动证明这些事实。没有自动版本选择、历史版本排序或回填已发布 release。

## 4. 验收与依据

正反例覆盖 tag grammar、所有 review/channel/profile 组合、严格 JSON 类型/重复字段、
路径/大小/UTF-8/敏感内容、notes 结构与代码围栏、CLI 输出和非零失败。默认不需要 Bash、
GitHub、密钥或网络。exact revision 的 CI 与本地测试分别记录。

版本规则依据 [SemVer 2.0.0](https://semver.org/spec/v2.0.0.html)；本地支持范围和
额外上限是本项目决定，不是上游兼容认证。
