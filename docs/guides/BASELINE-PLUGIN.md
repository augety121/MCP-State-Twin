# Agent Baseline Kit：本地插件与冻结实验

状态：SPEC-0078 的候选实现，精确验证见[实施台账](../maintenance/SPEC-0078-DELIVERY.md)。支持合成世界、可信 tools-only harness；真实 provider、带 shell 的产品宿主、L2 和 stable 各需独立证据。

## 1. 安装和准备

固定 Go 1.26.x / MCP Go SDK 1.8.0。可选 adapter 固定 Python 3.12、Inspect 0.3.275、MCP Python 1.26.0。下面使用本地构建物，无 tag/包发布、账号探测或宿主配置修改。依赖安装需要网络；执行插件与 mock 不需要 provider 账号。

```text
go build -o .tmp/statetwin.exe ./cmd/statetwin
python scripts/prepare-baseline.py --binary .tmp/statetwin.exe --out .tmp/baseline
python -m venv .tmp/inspect-env
```

使用该虚拟环境的 Python 执行 `-m pip install './adapters/inspect[test]'`。Windows 路径为 `.tmp/inspect-env/Scripts/python.exe`，Unix 为 `.tmp/inspect-env/bin/python`。中文 Windows 启动 Inspect 时使用 `python -X utf8` 或进程环境 `PYTHONUTF8=1`，否则固定 Inspect 版本读取自带模型元数据时可能遇到 GBK 解码错误。

准备脚本复制合成源文件，独立构建 actual/reviewed world，生成 Go client 和 Inspect 会话计划。已有输出目录失败；请指定新目录，不删除既有证据。

```text
.tmp/statetwin.exe plugin check --root .tmp/baseline --pack baseline-pack.json --profile plugin-profile.json
.tmp/statetwin.exe plugin describe --root .tmp/baseline --pack baseline-pack.json --profile plugin-profile.json
.tmp/statetwin.exe plugin projection --root .tmp/baseline --session-plan inspect-session.json
.tmp/statetwin.exe plugin schema --type session
```

`schema` 支持 pack/profile/session/report/baseline-plan/baseline-report，输出 Draft 2020-12 的字段、类型、required 与封闭对象结构。语义枚举、资源边界、跨文件绑定与路径规则仍须经 `check`/`projection`，单独 JSON Schema 验证不授予运行资格。

## 2. 真实 Inspect 连接，免费 mock 验证

可信 harness 可使用：

```python
import asyncio
from inspect_ai.tool import ToolDef
from statetwin_inspect import Session, verified_report

async def run():
    session = Session(".tmp/statetwin.exe", ".tmp/baseline", "inspect-session.json")
    async with session.connect():
        tools = {ToolDef(tool).name: tool for tool in session.tools}
        read_issue = tools["get_issue"]
        await read_issue(owner="octo", repository="demo", number=1)
        await session.finish({"repository": "octo/demo", "number": 1,
                              "state": "open", "title": "Target issue"})
    print(verified_report(".tmp/statetwin.exe", ".tmp/baseline", "inspect-session.json"))

asyncio.run(run())
```

这是合成 scripted contract 轨迹，不是模型成功率。框架集成入口 `statetwin_inspect.task.plugin_task(binary, root, plan)` 使用 Inspect Task/Solver/Scorer。当前 solver 只接受 `mockllm/model`，fresh 单消息、无已有工具；每个 adapter 进程最多一个 session，不接受 resume、额外 tools、provider 默认模型或缓存评分。

Agent 只得到 objective/context 和当前 Task 工具。oracle、witness、控制 token、terminal、评分均在可信侧。最终 answer 最多 32 KiB，只作为原 Task oracle 的输入；任何自报 success/score 都不能覆盖评分。

`Session` 管理独立子进程、持久 MCP connection、私有握手、finally abort 和回收。不要手工将 private lifecycle 作为 MCP 工具，也不要用 `asyncio.wait_for` 在另一个 task 内进入 Inspect 的 MCP context。协议 stdout 不含 CLI 报告。

```text
.tmp/statetwin.exe plugin inspect --root .tmp/baseline --session-plan inspect-session.json
.tmp/statetwin.exe plugin recover --root .tmp/baseline --session-plan inspect-session.json --out recovered
.tmp/statetwin.exe plugin inspect --root .tmp/baseline/recovered --session-plan inspect-session.json
```

inspect 只做本地重放与评分核验，不重跑模型或 quality witnesses。recover 复制到新目录，保持原始失败/partial/residue；不能让 partial 变为可续跑任务。

## 3. 24 family / 72 instances

```text
.tmp/statetwin.exe baseline variants --root .tmp/baseline --pack baseline-pack.json --profile plugin-profile.json --out expanded
.tmp/statetwin.exe plugin check --root .tmp/baseline/expanded --pack baseline-pack.json --profile plugin-profile.json
```

固定枚举生成 original/dev、namespace-regression/regression、namespace-evaluation/evaluation。目标命名空间在 Task/authority/oracle/witness、actual/reviewed Bundle 全部同步替换；没有用户脚本、LLM 生成器或自由 CEL 模板。变体产生独立 Task ID、revision 和 pack entry ID，源 Task ID 保留为 family 的语义身份。两个领域仍是 24 family，不能称为 72 种独立能力。

evaluation 显式 `public-evaluation-split`；公开内容不能充当未见 private holdout。生成结果的 reviewed 文件是源评审资产的机械变换，仍为 self-reviewed，不冒充第二位评审者。

```text
.tmp/statetwin.exe baseline register --root .tmp/baseline/expanded --pack baseline-pack.json --profile plugin-profile.json --registry registry
.tmp/statetwin.exe baseline revision-state --root .tmp/baseline/expanded --registry registry --pack-id reference-agent-baseline --revision v2
```

注册保存冻结原始内容。同 identity/revision 重复登记相同内容幂等，不同内容拒绝。`transition --event event.json` 追加 deprecated/revoked 事件；弃用需声明两个不同 preview 版本和替代 revision，紧急撤销可直接追加原因。事件是本地维护者声明，不是密码学发布证明。历史证据不删除。

## 4. 冻结计划、逐 shard 与四态判定

`baseline-pilot.json` 固定 24 dev × 2 配置 × 3 repeats = 144 trials，12 trials/shard，共 12 shards。两配置按 pair 交替 AB/BA。当前运行入口只执行 Go client 的 scripted witness，报告始终 `mode=offline-contract`；promptVersion 字段用于身份绑定，脚本不因此变成 prompt 实验。

```text
.tmp/statetwin.exe baseline check --root .tmp/baseline --plan baseline-pilot.json
.tmp/statetwin.exe baseline freeze --root .tmp/baseline --plan baseline-pilot.json --out experiment
.tmp/statetwin.exe baseline shard --root .tmp/baseline/experiment --plan baseline-pilot.json --shard 1
.tmp/statetwin.exe baseline assess --root .tmp/baseline/experiment --plan baseline-pilot.json
```

每条 shard 命令只启动指定分片，显式依次执行 1–12；没有自动后续 shard、自动恢复或失败重试。并行运行由 experiment lease 拒绝；crash 后的 lease/claim 保留，重试须新计划/目录。旧 project 120 秒、campaign 480 秒不变。新计划最多 240 trials、20 shards、12 trials/shard、2700 秒/shard。

冻结目录保存原始输入；运行不再依赖可变源目录。完整 trial 也不能替代丢失的 shard seal。assess 保留全部计划行、缺失/未核验/中断与未知费用，读取 terminal 重新评分；不接受导入的自报分数。

四态顺序：validity → 新 policy failure/非法实际副作用/required Task → 分领域 floor → family 分层 bootstrap 非劣。算法 `stratified-family-bootstrap-v1` 使用固定 SplitMix64、不偏取样、10,000 次、nearest-rank 95% 区间；每域等权、cluster 内保留所有实例与 repeats。至少 20 family、每域 8 family、3 repeats。全成功/全失败导致退化分布时 inconclusive，不伪造无风险结论。适用总体只包含声明的两个合成领域。

## 5. 诊断、证据与真实使用边界

| 现象/错误 | 处理 |
|---|---|
| PLUGIN_PLAN_INVALID / REFERENCE_MISMATCH | 先 check，核对未修改的 Task/review/world/cases、字段、链接和输出路径 |
| PLUGIN_OUTPUT_EXISTS | 原目录不可覆盖，选择新 session ID/output |
| PLUGIN_CONTROL_DENIED / STARTUP_TIMEOUT | 核对私有 IPC 权限、固定 profile 和可信父进程；不把 token 写日志 |
| PLUGIN_INTERRUPTED | 保留 partial，inspect/recover；新计划重试，不自动 resume |
| PLUGIN_OUTPUT_RESIDUE | 保留发布物与 pending，不能宣称发布完整 |
| BASELINE_SHARD_BUSY_OR_INTERRUPTED | 另一个 shard 正在执行或上次 crash；不得删除未知进程的 lease 后继续旧实验 |
| invalid / inconclusive | 分别表示证据不合格 / 当前样本不能判定，不等于业务失败 |

API live-check 复用已有 local bridge 的授权、到期和请求预算，只做准入，不创建 provider client。Inspect live profile 尚未获得独立 egress/credential/预算授权；不能把既有 bridge 批准搬过来。具体六任务计划、账号/模型和费用授权齐备后才执行。

冻结计划另支持 `mode=api-bridge-live`：每个 trial 必须通过 `livePlans` 映射到原有严格 `agent-live-plan`，绑定相同 Task/world、独立 plan ID、明确模型、时效、请求/token 上限与全部批准字段。配置 provider=openai、adapter=`openai-responses-local-bridge-v1alpha1`、host=api-bridge/v1alpha1、promptVersion=blind-objective-v1。`baseline shard ... --allow-live` 在本 shard 全量授权检查通过后才读 `OPENAI_API_KEY`；到期、未批准、错误绑定或漏掉 flag 都不创建 provider client。只能走原固定 endpoint/no-retry 传输，不能传入 URL 或外部执行器。先经单 Task live lane 完成获批六任务冒烟，再获批执行具体 pilot；不预选模型或自动开启付费试验。

live 汇总重新验证原 live evidence、receipts 与冻结计划，缺失 receipt usage/cost 保持 unknown；报告明确 API bridge 观察不能代替 stdio/Inspect 或产品宿主兼容证据。

claim-check 分别绑定 runtime/head、Host/版本、adapter、MCP、pack/revision、oracle、statistics、evidence。API TTL 30 天，产品 14 天；绑定变化或到期 stale，撤销/回归 regressed。unsigned 本地声明最多 experimental/unverified，不自动发出独立 verified claim。

产品宿主实测、三名非作者 first-value 和≤3操作独立可复位 reference 尚需外部条件。不得联系外部参与者、读账号凭据、发模型请求或发布包来绕开这些授权。

## 6. 维护与发布

Runtime owner 维护协议/权限/cleanup；Evaluation owner 维护 Task/quality/统计；Adapter owner 维护固定 Inspect 版本；Maintainer 负责 review、撤销和发布；具体个人由仓库维护者分配，不虚构参与者。

当前仅 development preview 候选，原 alpha 发行物不因此被替换。最终 head 必须完成 Go fmt/vet/race、三平台 plugin/Inspect、旧回归、秘密策略与候选安装。性能门槛的实测及未解决项见实施台账。推送 PR 不等于发布 tag、包或 stable。
