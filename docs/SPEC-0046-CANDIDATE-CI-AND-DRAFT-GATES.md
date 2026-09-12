# SPEC-0046: 同一候选提交的完整 CI 与 draft 门禁

Status: accepted via [ADR-0046](ADR-0046-FULL-CANDIDATE-RELEASE-GATES.md), 2026-09-12.

## 1. 作业依赖

```text
tag push -> admit (plan + notes + HEAD/tag + main ancestor)
         -> gates (same-commit reusable ci.yml; all seven CI jobs)
         -> stage (recheck tag + build fresh artifacts + create draft only)
```

stage 同时 depends on admit 和 gates；默认 success 依赖语义不可被 `always()`、
`continue-on-error` 或 fallback 替代。测试失败或取消时不构建/上传 release assets。
根 permissions 为 contents:read；admit/gates 无 write。仅 stage 为 contents:write。
相同 ref 的 release workflow 使用 concurrency group 且 cancel-in-progress=false；
不承诺 GitHub 任意数量排队任务的公平/持久队列语义。

## 2. 身份与完整检查集合

checkout 固定 github.sha。admit 检查 HEAD 与 GITHUB_SHA、本地 tag peeled commit
一致，以及该 commit 属于最新 fetch 的 origin/main 祖先。祖先关系不证明有人 review；
SPEC-0045 的声明和维护流程仍独立。stage 重新 fetch 精确远端 tag 并核对 peeled commit，
发生移动或缺失就失败。检查与服务端最终创建之间仍可能竞争，不声称 tag 原子锁定。

不请求一个旧 CI run 的 success 来替代新验证。调用本仓库同一提交的可重用 CI，包含：

- Linux format、vet、race、文档、Scenario、Bundle/Journal、资源/协议 smoke 和 build；
- Windows 与 macOS tests/build；
- bounded TwinSpec/CEL fuzz；
- history secret scan 与 synthetic-fixture policy；
- Linux loopback-only hermetic 测试；
- 已固定的官方 MCP conformance subset。

现有 main/pull_request 触发保留。workflow_call 无需要传入的 secret 或 endpoint；
不添加 paid Provider 测试。某个 gate 失败就修复原因，不在 release notes 里豁免 required CI。

## 3. Draft 与错误

创建命令必须使用 `--verify-tag --draft --latest=false --notes-file`，预发布还需
`--prerelease`；不用自动生成 notes 替代 reviewed notes。缺失远端 tag 不允许 CLI 自动创建。
已有 release、不完整上传、限流、鉴权失败均原样非零；无 edit/delete/clobber 或自动重试。
GitHub Release 与多 asset upload 不是一个事务：可能遗留 draft/部分附件，维护者先检查，
不能将 job 失败当作“外部什么也没发生”。没有自动清理远程 draft 的操作。

成功只代表 draft 已准备；维护者仍要复核 scope、格式/迁移、artifact、exact CI 与兼容性
限制，再决定发布和 Latest 标识。本次实施不创建 tag、draft 或新发行包。

## 4. 验收与上游依据

使用 YAML 结构测试验证依赖图、同仓库可复用路径、完整 required job 集合、只读默认权限、
无失败绕过、tag 重检、draft/notes/prerelease 参数与 action pins。main CI 运行这些测试；
它不等于已经执行新的 tag workflow，后者必须有单独 run 证据。

GitHub 官方说明同仓库相对 reusable workflow 使用调用方同一提交，权限不可提升，见
[Reuse workflows](https://docs.github.com/en/actions/how-tos/reuse-automations/reuse-workflows)。
draft/prerelease/verify-tag/notes-file 语义见
[gh release create](https://cli.github.com/manual/gh_release_create)。已查阅于 2026-09-12；
这些文档不是当前仓库真实发布成功的证据。
