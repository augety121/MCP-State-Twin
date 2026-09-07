# 本地 Responses bridge：先审计划，再选择是否付费运行

这是实验性功能，**目前只有本地 contract tests，没有本次新增 profile 的真实 API 报告**。
默认 CI 不需要密钥、不访问模型。若只想验证项目，请先跑
[纯离线回归教程](OFFLINE-AGENT-REGRESSION.md)。

这条路线把合成任务的可见目标、授权、工具 schema 和工具结果发给 OpenAI API，
业务工具在私有 Twin world 内执行；不创建真实 GitHub issue/PR，不部署 remote MCP。
账户/模型可用性、费用、服务端行为仍须实际核验，不能等同 ChatGPT/Codex 产品兼容。

## 1. 准备（不调用模型）

需要仓库构建的 `statetwin` 位于 PATH，或用 `go run -p 1 ./cmd/statetwin` 替换前缀。
从仓库根目录执行。保留默认 quiet 模式；本地构建/测试可设置 `GOMAXPROCS=1`。
先检查目标文件是否已存在：本项目拒绝覆盖已有 Bundle/evidence。

```powershell
New-Item -ItemType Directory -Force examples/issue-tracker/.statetwin/live
statetwin bundle build --manifest examples/issue-tracker/bundle-agent.yaml --out examples/issue-tracker/.statetwin/agent-world.stb
```

若已完成离线教程，复用已经验证的同一 Bundle，**不要重复 build 覆盖它**。

## 2. 生成未批准计划（不读密钥、不调用模型）

将 `YOUR_APPROVED_MODEL` 换成你实际账户可用且愿意付费使用的模型。下面 4 次请求、
每次最多 1024 输出 token 是**示例上限，不是推荐费用或本次执行授权**。
本地 plan 模型 grammar 校验不证明这个 ID 是真实模型。

```powershell
statetwin eval live-plan --root examples/issue-tracker --task agent-tasks/close-issue.json --id my-reviewed-close-01 --model YOUR_APPROVED_MODEL --max-requests 4 --max-output-tokens 1024 --valid-for 1h
```

命令只向 stdout 输出 JSON。将它保存为该 root 下新的 UTF-8 文件，例如
`reviewed-plan.json`；不要覆盖旧计划，不要把 key 写进去。
PowerShell 旧版的 `>` 可能写出 UTF-16；请使用编辑器明确保存为 UTF-8。

逐项审阅完整 Task、Bundle 身份、模型、上限、有效期、数据与未知费用风险。
只有你实际同意后，才手动将 `approved`、`syntheticDataApproved`、
`unknownCostApproved` 三项设为 true。修改任务/模型/预算意味着新的审批内容。

```powershell
statetwin eval live-preflight --root examples/issue-tracker --plan reviewed-plan.json
```

`locally-valid-not-provider-verified` 是结构结果。还需检查 `approvalCurrentlyValid`；
即使 true，也没有测试账户、网络、模型、费用，更不会自动开始运行。

## 3. 可选付费执行（这一步会发送真实请求）

通过你自己的安全方式在当前进程环境提供 `OPENAI_API_KEY`，不要把值发进聊天、
提交 Git、写进计划，或放在命令参数中。确认全部批准后才执行：

```powershell
statetwin eval live --root examples/issue-tracker --plan reviewed-plan.json --allow-live
```

只接受固定 OpenAI HTTPS endpoint；忽略环境 proxy，不支持自定义网关。
使用需要企业代理/自定义路由的环境时本 profile 可能无法连接，不要为了连通关闭 TLS。
命令非零退出时先检查规范化 failureCode 和已有目录；**不要自动重跑**。
超时可能已经被远端接受或计费，不能将“没收到响应”当作零费用。

输出位于 `.statetwin/live/<plan-id>/`。相同目录禁止任何复用；缺凭据、崩溃或写盘失败
也可能留下 claim/staging。先保留并人工检查，新尝试必须有新的审阅与预算。
请求次数只是本地单计划 cap，不是账户级货币上限；token usage 缺失和费用保持 unknown。

## 4. 独立重放（不读密钥、不调用模型）

```powershell
statetwin eval live-verify --root examples/issue-tracker --evidence .statetwin/live/my-reviewed-close-01/terminal.json
```

它检查 Bundle/Task/计划/receipt/事件/终态/评分的一致性，并重放本地工具世界。
一个模型任务失败也可以拥有完整有效证据；partial 执行不能通过完整验证。
`providerProvenance:not-proven` 和 `modelSnapshot:unknown` 是刻意保留的边界：
无签名工件不是独立的服务端执行凭证，world replay 不是 model replay。

已有 `eval verify` / `eval compare` 只接受 offline-kind，不能混用 live artifact。
六任务实际 live 矩阵、两真实配置比较、原生 remote MCP 与产品宿主仍需独立验收。
契约细节见 [SPEC-0040](../SPEC-0040-LIVE-PLAN-AND-APPROVAL.md) 和
[SPEC-0041](../SPEC-0041-PROVIDER-TRANSPORT-AND-EVIDENCE.md)。
