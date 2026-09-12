# 第一次跑出 Agent 回归报告（纯离线、实验性）

这条演示验证评测基础设施，不证明真实模型能力。无需 API key，不访问模型服务，
不会创建真实 GitHub issue/PR。需要已构建且在 PATH 中的 `statetwin`，或将命令前缀
替换为 `go run -p 1 ./cmd/statetwin`。请从仓库根目录运行；已有同名输出会被拒绝覆盖。

## 1. 准备合成世界

```powershell
New-Item -ItemType Directory -Force examples/issue-tracker/.statetwin
statetwin bundle build --manifest examples/issue-tracker/bundle-agent.yaml --out examples/issue-tracker/.statetwin/agent-world.stb
statetwin eval preflight --root examples/issue-tracker --task agent-tasks/close-issue.json --config agent-runs/baseline.json
```

预期 `offline-statically-valid`。这不是模型账号、live 或任务可解性的检测结果。
比较计划已在 `examples/issue-tracker/agent-comparison.json` 固定两个 trial 和输出位置。

## 2. 跑正确基线

```powershell
statetwin eval mock --root examples/issue-tracker --task agent-tasks/close-issue.json --config agent-runs/baseline.json --responses agent-mocks/close-issue.json --out .statetwin/baseline
statetwin eval verify --root examples/issue-tracker --evidence .statetwin/baseline/terminal.json
```

预期 task `success`、evidence `complete`，verify 显示 world replay 和 grading matched。
工具调用经过实际本地 MCP SDK；模型响应来自明确标注的合成文件。

## 3. 注入“忘记做事”的候选回归

```powershell
statetwin eval mock --root examples/issue-tracker --task agent-tasks/close-issue.json --config agent-runs/candidate.json --responses agent-mocks/omit-action.json --out .statetwin/candidate
```

这条命令**预期非零退出**：候选直接结束、没有关闭 issue，应该得到 `task_failed`。
它仍保留完整、可核查的失败证据，不要为了让命令成功而替换结果。

```powershell
statetwin eval verify --root examples/issue-tracker --evidence .statetwin/candidate/terminal.json
statetwin eval compare --root examples/issue-tracker --plan agent-comparison.json --format markdown
```

verify 预期成功：证据有效与任务成功是不同问题。compare 预期显示 `regression`，
planned/started/terminal/validlyEvaluated 均为 2，并非零退出。`upgradeAllowed` 始终 false。

如果没跑候选就比较，会得到 `inconclusive`，不会把缺失的一组从分母中删掉。
需要 JSON 时，将 `--format markdown` 改为 `--format json`。

## 4. 扩展与边界

`agent-mocks/` 还有读取、新建、已关闭不动作、注入越权提示和提交后确认等合成响应。
六个任务的完整 mock 回归在测试中覆盖。手工做新运行时使用新 trial ID、新配置和
新输出目录，并在运行前固定比较计划；不要复用旧证据目录。

输出目录只包含合成 claim/terminal 工件，异常中断可能保留 closure/pending 文件。
可用 `statetwin eval inspect --root examples/issue-tracker --out .statetwin/baseline`
只读检查目录，详见 [故障诊断指南](EVIDENCE-FAILURE-DIAGNOSIS.md)。
这些未完成目录不会自动恢复或覆盖。根目录必须可信；未知输入和敏感
模式会被拒绝。完整资源、保存和验证限制见 [SPEC-0039](../SPEC-0039-OFFLINE-AGENT-REGRESSION.md)。

从这个演示到真实模型评测，可审阅独立的 [本地 API bridge](LOCAL-API-BRIDGE.md)。
它已有 contract-tested transport，但仍需具体 model/profile、合成数据许可、
请求/费用授权和实际 live 报告；这里的 `mock-*` 标签不是可调用模型 ID。
