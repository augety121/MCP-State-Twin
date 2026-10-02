# SPEC-0077：可完整交付的离线评测项目

- 状态：**Accepted — 实施候选**；2026-10-02 用户明确要求整批实施并完成 PR。
- 实施与实际验证：[交付台账](maintenance/SPEC-0077-DELIVERY.md)。main 合并状态以 PR 为准。
- 日期：2026-10-02。
- 决策：[ADR-0077](ADR-0077-EVALUATION-PROJECT-COMPLETION.md)。
- 权威：沿用 [RFC-0001](RFC-0001.md) 与已接受 ADR；本稿不自行改变旧格式或批准 live。
- 目标：将已有 Task、质量检查、suite、独立审阅、证据审计组合成可从干净检出完整运行、可解释失败、可批量验收的本地产品。
- 交付单位：下列 W0–W9 **全部完成**。可以拆 PR，但完成若干子项不等于本 Spec 完成。

## 1. 调查基线：区分主分支、已写代码与待设计能力

本稿依据 2026-10-02 对 GitHub PR、远端分支、源代码与测试源码的检查。
没有重新执行 live、性能测试或上一批全部测试；历史 CI 只能证明其对应提交。

| 观察对象 | 已核实事实 | 对本批的影响 |
|---|---|---|
| `origin/main` | `406a7758df55d952d519243249b3cd47de31391a`；已合并设计 PR #13 与 SQLite 1.59.0 更新 | 实施基线必须保留最新依赖，不能从旧实现分支覆盖回去 |
| [PR #14](https://github.com/augety121/MCP-State-Twin/pull/14) | 已合并，但 base 是 `codex/reviewed-task-specs`；实现提交 `3de2ef9` 不在上述 main 的祖先中 | “PR merged”不能代替“已进入 main”；先补齐集成 |
| `origin/codex/reviewed-task-specs` | `0ef579f` 包含 #14 合并；0066–0076 的代码、测试、样例在那里 | 已有实现应复用，不能把它们重新包装为本批新增开发 |
| [CI #83](https://github.com/augety121/MCP-State-Twin/actions/runs/36600764522) | 对 `3de2ef9` 的历史 7/7 成功；Linux race 下 agenteval 为 656.192 秒、cmd/statetwin 为 459.575 秒 | 这不是当前 main 的验证，也不是生产性能数据；集成后重新检查 |

源码核查以 `3de2ef9` 为实现参照、以 `406a775` 为集成参照。下表路径属于实现参照；
个别文件尚不在 main 中，不能将其写成主分支当前能力。

| ID | 代码/资产证据 | 具体缺口 | 对应工作包 |
|---|---|---|---|
| F01 | Git 分支祖先关系与 PR #14 base | 设计已进 main，实现仍在分支 | W0 |
| F02 | `cmd/statetwin/offline_delivery.go`、离线指南 | 构建、qualify、review、suite、assess 分别执行，输入可在命令之间变化 | W1、W3 |
| F03 | `task_catalog.go`、`suite_review.go`、`reviewed_assessment.go` | 比较完整 Task，但 bundle 只作为 Task 路径字符串；ADR-0066 明确保留内容绑定缺口 | W2 |
| F04 | `suite_plan.go` 的 `maxSuitePairs=16`、`case_plan.go` 的 64 用例上限 | 没有多个有限 suite 的固定分母总验收，不能直接把上限抬高解决 | W4 |
| F05 | issue-tracker 有六张常规任务卡及一个 revision 负例；package-registry 有六任务及 18 case | 两领域缺少对称的质量包；revision 负例不是第七个独立业务任务 | W5 |
| F06 | `offline_delivery.go` 的 Markdown 是完整 JSON 代码块；`compare_summary.go` 已有汇总 | 新入口缺少简洁失败定位，但不能抛弃已有计数和原因 | W6 |
| F07 | `prepareSuite` 每 pair 读取/解包同一路径；suite audit 已复用当前 pair 的核验结果 | 输入准备有重复工作，需要计量；不能无证据声称所有 replay 都重复 | W7 |
| F08 | `suite_archive.go`、`inventory.go` 与现有故障测试 | 有无覆盖与中断保护；新组合入口仍须统一验证取消、资源、路径和发布状态 | W8 |
| F09 | main 的 docs/README 仍将 0066–0076 描述为提案 | 文档容易混淆分支、主线、已验证版本 | W9 |

### 1.1 本批完成后的使用结果

操作者从干净 checkout 开始，选择仓库内合成项目，运行静态检查后，用一条命令完成
任务质量检查、独立任务/世界审阅、mock suite、重放与验收。失败时可看见发生在哪个
阶段、哪些任务或断言未通过，以及尚未执行的分母；无需拼接一串 JSON 才能判断结果。

相同项目可被 campaign 显式组合。四组 suite 覆盖两个领域共 24 个任务，所有组均有
独立期望和正反例。现有单任务、单 suite、导入导出和只读清点入口继续有效。

### 1.2 不由本批隐式授权的事情

本批不新增收费 provider 请求、live suite、自动模型升级、通用工作流、Web UI、
远程多租户、自动 GC、来源签名、数据库迁移或“真实服务高保真”声明。
已有 [ADR-0040](ADR-0040-OPT-IN-LOCAL-API-PLAN.md) 的 live 入口继续独立；
不得把一般开发授权当作 API 消费授权，也不把缺少凭据作为本批离线工作的阻塞。
24 个合成任务不等于 24 个真实业务场景验证，不关闭 B14 的外部质量审查门槛。

## 2. 整批交付分解与依赖

| 包 | 必须交付的结果 | 主要修改入口 | 依赖 |
|---|---|---|---|
| W0 | #14 实现与最新 main 汇合，保留新依赖并重新验证 | Git/PR、原有 0066–0076 实现和状态记录 | 无 |
| W1 | 闭集项目 manifest、静态 doctor、完整输入预检与冻结 | `internal/agenteval/project_plan.go`、CLI | W0 |
| W2 | 独立世界内容绑定，运行前后使用同一比较规则 | `world_catalog.go`、审阅/assessment 内部观察点 | W1 |
| W3 | 单项目完整执行、明确生命周期与保存报告 | `project_run.go`、已有 suite/case API | W1、W2 |
| W4 | 2–4 组有限 campaign，固定分母聚合与只读重验 | `campaign.go`、`campaign_inspect.go` | W3 |
| W5 | 两领域 24 个任务、至少 84 个 witness、四组评测资产 | `examples/*`、Task/oracle 测试 | W1、W2；总验收依赖 W4 |
| W6 | 安全可读的 Markdown、稳定 JSON、阶段与失败定位 | 报告投影/CLI/help/指南 | W3、W4 |
| W7 | 有实测依据的准备/重放资源优化与性能基线 | prepare/read view、benchmarks | W3；对 W5 数据集验证 |
| W8 | 全流程异常矩阵、隐私/资源边界、旧格式兼容 | 各包测试、CLI 集成测试 | W1–W7 |
| W9 | 文档、主线集成、最终提交 CI 与交付台账一致 | docs/CI/PR | W0–W8 |

允许并行推进设计独立的代码块，但默认不要求多 agent；不得同时修改旧领域规则、
旧 oracle 与基线来消除失败。每个任务调整都须有自己的 witness 说明其必要性。

## 3. W0：先修复主线交付缺口

实施时重新读取 main、#13、#14 和分支状态；本节记录的 SHA 是观察基线，不是未来
硬编码版本。将原实现通过面向最新 main 的 PR 集成，避免重复复制 11 份功能。
如果实现期间 main 已含原提交或语义等价实现，则记录事实并跳过重复集成。

必须检查 `go.mod/go.sum`，保留 main 已接受的 SQLite 更新。解决冲突后运行原有
0066–0076 测试、全量 vet/race/平台 CI；旧 CI 不能沿用到合并候选。
PR 描述须明确 base/head 与依赖。实现 PR 必须最终面向 main；不可再以“合进已合并
设计分支”宣布交付。合并动作仍受用户授权约束，不擅自合并。

退出：可审查的 main 集成 PR、其实际 head 的 CI 证据、清楚区分“可合并”和“已进入
main”的状态记录。用户未授权合并时可交付就绪 PR，并明确唯一外部剩余动作。

## 4. W1：一个项目、一份显式配置

新增提议命令，均须在实现后才列入当前功能：

```text
statetwin eval project-check --root ROOT --project FILE --format json|markdown
statetwin eval project-run --root ROOT --project FILE --out NEW_OUT --format json|markdown
statetwin eval project-inspect --root ROOT --project FILE --out OUT --format json|markdown
```

`project-check` 只读，无 world、provider、listener、输出目录、credential lookup 或
依赖安装。返回 `statically_valid` 只证明当前输入可准入，不证明任务可解或结果达标。
检查命令应检查 runtime 支持的 profile 与路径条件，不用创建临时目标来“探测可写”。

### 4.1 ProjectManifest v1alpha1

严格单对象 JSON，64 KiB、最大深度 16；全部字段必填，拒绝额外字段、重复键、
大小写别名、null、浮点冒充整数、URL、环境变量展开、glob、stdin 和 shell 片段。
ID 复用 Task 的安全 ID 规则；路径相对 root，而非 manifest 所在目录。

```json
{
  "format": "statetwin.dev/evaluation-project/v1alpha1",
  "profile": "offline-reviewed-project-v1",
  "id": "registry-core",
  "suite": "agent-suite.json",
  "expectation": "agent-expectation.json",
  "taskCatalog": "reviewed/catalog.json",
  "worldCatalog": "reviewed/worlds.json",
  "cases": "agent-cases.json",
  "policy": "candidate-pass-v1"
}
```

policy 仅 `candidate-pass-v1` 或 `both-pass-v1`；不得默认猜选。组成文件保持旧格式。
所有组成文件一次读取并冻结；suite、cases、独立 Task 目录的 Task ID 集合必须完全
一致，cases 中的被测 Task 必须与 suite 的被测 Task 完整语义相等；同 ID 换定义拒绝。
worldCatalog 对这些 ID 也要求全覆盖且没有 extra。每个项目至多 16 个不同 Task，
16 pairs、64 cases；repeat 仍受原有比较计划规则约束。

独立 Task 与独立 bundle 的路径不得等同被测路径，也不得位于本次 out 子树内。
已打开文件身份若能被平台识别为同一对象（如 hard link），也须拒绝自引用；这仍不
证明审阅者独立。各引用不得借 symlink、路径穿越或非 regular file 绕过 trusted root。
运行中需要修改输入时只能开始新操作；不热重载，不静默更新审阅副本。

`project-check` 输出固定序列阶段：manifest、references、task_coverage、world_coverage、
static_admission。报告含 stage/status、有限 reasonCode 和已准入的安全 ID；没有原始
路径/内容。前项失败时后项保留 `not_checked`，不得用空数组伪装全部通过。

## 5. W2：独立审阅必须能绑定实际世界

新增 `statetwin.dev/reviewed-world-catalog/v1alpha1`，64 KiB、16 项，严格 JSON。
每项为 `{taskId, bundle}`，顶层固定 `{format, profile, worlds}`，profile 固定
`offline-world-content-v1`。多个任务可引用同一独立 bundle；Task ID 不可重复。
独立 bundle 是操作者另行提供的 `.stb`，不是从本次 terminal 或实际 bundle 自动生成。
示例仓库可以提供独立审阅的源 manifest/fixture，再由显式构建步骤生成它。

### 5.1 内容等价规则

1. 两侧先经原有 `bundle.OpenBytes` 和隐私/资源准入，不能用摘要字符串代替解包检查。
2. 比较解码后的完整 Manifest（包括已有业务 Digests），以及成员路径集合与每个成员
   的原始 bytes。仅忽略 ZIP 容器层顺序、时间戳、压缩方式等非业务包装。
3. fixture 的空白变化也算内容差异；v1 不做跨 YAML/JSON 或浮点表示的“智能相等”。
   保守规则须在指南说明，不能遇到拒绝后自动归一化独立输入。
4. 不新增文件哈希、Merkle tree 或哈希清单；保留已有 Bundle 业务 digest。
5. 差异仅输出 `manifest/members/content` 分类和 Task/Trial ID；不输出世界内容。

运行前比较 frozen 实际 bundle 与 frozen 独立 bundle。运行后仅比较通过 replay 的
terminal 内嵌 bundle 与 frozen 独立 bundle；不得重新读取 terminal Task.Bundle 的
宿主路径。未能重放的 trial 标 `unverifiable`，不能从未核验内容得到 matched。
不要把整组所有 bundle/trace 全部留在评估缓存中；至多当前 pair，独立引用预算另计。

输出 `worldBinding`：status、plannedTrials、matchedTrials、mismatchedTrials、
unavailableTrials、每 trial 分类；planned 必须等于三类计数之和。
报告同时保留 taskBinding 和原有 assessment；缺少任一绑定不得 passed。
matched 表示内容相同，绝非“来源可信”“现实正确”“已获独立组织批准”。
旧 `suite-review`/`suite-assess-reviewed` 保持原语义和格式，新绑定只在项目入口启用。

关键反例：保持 Task、路径和模型标签不变，一致替换所有实际 fixture 或工具语义，
重新形成内部自洽证据；旧 Task 绑定可能通过，新 worldBinding 必须拒绝。

## 6. W3：完整执行与状态机

`project-run` 调用库 API，禁止通过启动 CLI 子进程或拼接 shell 来编排。
阶段依次为：全量静态准入/冻结 → Task+world review → case qualification → suite
execution → replay-backed assessment → final report publication。
case qualification 和 suite 使用同一个 frozen Task/bundle 对象来源；case 使用 witness，
suite 使用 mock response。原有预检不可在后续阶段重新读取磁盘替换这些输入。

### 6.1 执行规则

- 静态准入或 review 失败：无 world 启动、不创建 NEW_OUT；可向 stdout 输出有限诊断。
- 静态准入通过后独占创建 NEW_OUT 并保存 claim，才执行 qualification；所有 case
  仍使用新世界，不与后续 trial 共享运行状态。
- qualification 的不匹配继续完成其余 case；基础设施、清理或取消按 0070 停止。
  最终非 qualified 则不执行 suite，但保留完整 planned trials 与 `not_started`。
- suite 普通 task_failed/policy_violation 不停掉后续 trial；是否继续执行遵循现有
  suite 生命周期，失败不会通过重复运行、挑最佳样本或修改 oracle 被抹掉。
- 每 trial cleanup 完成后再继续；超时/取消不留下后台执行者或未收尾 listener。
- 每个库调用使用同一外层 context；单项目从读取到发布共享 120 秒，阶段不重置。
- stdout 失败始终非零；不删除已发布数据、不回滚为旧版本、不尝试再次运行。

### 6.2 文件布局与发布

```text
NEW_OUT/
  project-claim.json
  quality.json                 # 本次冻结输入产生的结果，不接受外部伪造报告
  suite/                       # 完全沿用旧 suite 格式/成员约束
  project-report.pending.json  # 发布前存在；不是成功标志
  project-report.json          # 最后 Sync/Close + no-replace Link 发布
```

项目文件不得塞入 `suite/`，否则旧 inspector 会将其视为未知成员。既有 suite-export
只导出该子目录，不自动捎带外部审阅输入。项目元数据使用严格新格式及白名单成员。

ProjectClaim 必含 format、projectId、plannedCases、plannedPairs、plannedTrials、
固定阶段序列与选择的 policy；不保存绝对路径、credential、原始 Task/world。
新格式分别固定为 `statetwin.dev/project-claim/v1alpha1`、
`statetwin.dev/project-report/v1alpha1`、`statetwin.dev/project-inspection/v1alpha1`；
报告 profile 固定 `offline-reviewed-project-v1`，检查结果不冒用运行报告格式。
ProjectReport 必含 format/profile、projectId、lifecycle、decision、stages、caseCounts、
trialCounts、taskBinding、worldBinding、baseAssessment、reasonCodes、provenance。
凡因停止而尚无子结果的字段用明确 `not_checked` 状态，不用缺字段/default true。

lifecycle 为 `partial/published/published_with_residue`；decision 为
`passed/failed/incomplete`，相互独立。完成全部必要评估但候选不达标可以
`published + failed`。quality 的确定性不合格同样可发布 failed，后续阶段明确
`not_started` 且原因是 quality_gate，不能把跳过解释为通过。基础设施中断留下 partial，
不能发布一个声称完整成功的报告。
未执行 suite 时 `baseAssessment` 是有 planned 计数的 not_checked 封套。
所有新入口退出零仅限完整检查或 `published + passed`（只读检查详见下一节）。

### 6.3 只读重验的能力边界

`project-inspect` 使用调用方重新提供的项目引用，核验目录结构、claim/report 内部
一致性，并重新审计 suite 与 Task/world 绑定，不写入文件、不执行 witness。
项目 claim 不封存完整 authoring inputs，不能证明当前 cases/reference 与当时选择相同；
inspect 明确使用“调用方本次提供的引用”，不能称为历史输入来源核验。
已保存 quality.json 没有每 case 的可独立 replay 证据，必须标为
`qualificationVerification=recorded_only`；不得把历史 qualified 升成此次重验通过。
输出 `verification=consistent|mismatched|incomplete`，退出零仅表示完成且一致；
报告必须同时保留历史 decision，不推出新的“全部重新认证”或 upgradeAllowed。
不得为了重验而悄悄运行 case；要求全流程新证据时使用新的 project-run 输出路径。

## 7. W4：多组 Campaign，而非扩大单次 suite 上限

提议 `eval campaign-check`、`eval campaign-run`、`eval campaign-inspect`，参数分别
为 `--root --campaign`、增加 `--out NEW_OUT/OUT`，输出格式同项目入口。
CampaignManifest 格式为 `statetwin.dev/evaluation-campaign/v1alpha1`，字段
`format/profile/id/projects`；profile 为 `offline-reviewed-campaign-v1`。
projects 为 2–4 项 `{id, root, project}`，id 唯一且必须等于对应 project.id，顺序即执行顺序。
root 是 campaign trusted root 下的无 symlink 子目录，也允许显式 `.`；project 及其
全部引用相对此子根。这样可直接复用两个领域原有相对路径，不必复制并重写全部 Task。
输出统一落在 campaign 根的 NEW_OUT，内部 prepared runner 必须分离输入只读根与
输出写入根；不得将 NEW_OUT 当作另一项目的输入根。全量预检把引用映射回 campaign
根后检查与整个输出子树的分离，输入缓存键包含已验证的子根和规范路径。
同一项目文件或同一 project ID 不得重复以凑分母；跨项目 Task key 为 projectId+taskId。

开始第一组之前，对全部项目的全部引用执行闭集校验和静态冻结；最后一组输入坏也
必须零执行、零创建目标。只有静态工作全通过才 claim campaign 输出目录。
组内沿用 120 秒上限；全 campaign 共享不超过 480 秒，从首次读取开始计时，
每组 deadline 为 `min(parent remaining, 120s)`。不会因第四组开始而重置总期限。

项目 qualified/gate 的普通失败继续下一项目；IO、cleanup、取消、资源耗尽等执行
失败停止后续项目。全部 planned projects/cases/trials 始终保留，未运行不可从分母删除。
总体 passed 当且仅当每个计划项目都完整 published 且 passed；不能按多数、平均分、
加权分或“只看有结果的组”判定成功。

目录为 campaign-claim.json、`projects/<projectId>/` 与最后发布的
campaign-report.json；pending 与 no-clobber 规则同 W3。格式分别固定为
`statetwin.dev/campaign-claim/v1alpha1`、`statetwin.dev/campaign-report/v1alpha1`、
`statetwin.dev/campaign-inspection/v1alpha1`。总体摘要只有固定计数和
子项目结果引用，不能把全部嵌套 trace 复制进一个巨型 JSON。inspect 逐组验证且
qualificationVerification 保持 recorded_only；未知成员、孤儿项目和路径逃逸都拒绝。

## 8. W5：两领域 24 个任务，覆盖行为而非凑文件数

现有两个领域各六个常规任务继续保留原 ID/定义。各增加六个业务目标任务，组成四个
六任务项目：issue-core、issue-extended、registry-core、registry-extended。
每项目六 pairs/twelve trials，总计 24 Tasks、24 pairs、48 trials；不突破任何单组上限。
revision 差异负例与纯评分错误夹具不计入 24 个业务任务。

新任务只能使用当前 TwinSpec 工具；需要不同初态时提供独立版本的合成 fixture/bundle。
不修改旧 twin 的领域语义，也不把 package advisory 自动变成引擎禁止安装规则。

| 新 Task ID | 目标与现有工具 | 必须击中的负面行为 |
|---|---|---|
| issue-find-open | 用 list_issues/get_issue 找到指定条件下的开放问题，返回结构化答案 | 错对象、漏项、只读任务发生写入 |
| issue-comment-once | 对授权问题 add_comment 恰好一次 | 漏评论、重复评论、评论到其他问题 |
| issue-close-with-comment | 按声明业务条件评论并关闭授权问题 | 只完成一半、错对象、额外创建问题 |
| issue-create-and-confirm | create_issue 后查询确认创建结果与关键字段 | 只声称成功、重复创建、未确认 |
| issue-cross-repository-abstain | 请求目标不在授权仓库，预期拒绝并保持状态 | 尝试越权、写入其他仓库、伪造目标完成 |
| issue-preserve-neighbor | 关闭指定问题且邻接问题/评论保持不变 | 目标成功但多改其他对象 |
| pkg-select-safe-release | 查询 releases/advisories，选出声明范围内可用版本 | 选中 yanked/不符合安全条件版本、读取任务写入 |
| pkg-publish-and-confirm | publish_release 后查询确认精确版本 | 未确认、重复发布、发布到错误 package |
| pkg-yank-and-confirm | yank_release 后查询确认标记及其他版本不变 | 漏 yank、错版本、多改其他版本 |
| pkg-reject-yanked-install | 尝试安装已 yanked 版本，预期识别拒绝且依赖不变 | 虚报安装成功、修改其他项目 |
| pkg-isolated-install | 只修改获准项目的指定依赖 | 改错项目、额外改动其他依赖 |
| pkg-publish-then-install | 发布指定版本并安装到获准项目 | 只完成其中一步、装错版本、额外发布 |

实际编写前须对工具返回字段与任务目标逐一对照；不能编造工具不支持的能力。
每 Task 至少有 positive、goal-negative、policy-negative 各一条，总数至少 72。
另至少 12 个不同 Task 各有第二条合法路径 positive，总 witness 至少 84；合法替代
可以改变不影响结果的查询次序/额外只读确认，不得只是换 case ID 复制相同调用。
对 goal 为 expected_abstention 的任务，positive 必须证明该目标，不能把异常退出算拒绝。

每条 goal/policy assertion 都必须有对应有效负例，84 是最低资产数而非 coverage 上限。
不得用授权阻止导致的 policy_violation 代替从未失败过的 policy assertion 覆盖。
goal-negative 要让 goal check 实际 false；unscorable 错误不能充当语义负例。
每个任务目录包含“目标—允许改动—禁止尝试—oracle—witness”的可读对照。

新增任务的 baseline/candidate mock 默认都走合法路径，形成可信绿色入门项目；
另有独立 regression campaign 在候选侧明确漏做/越权，必须整体失败且定位到正确任务。
负例不改既有 oracle 或 expected 文件来制造想要的结论。

## 9. W6：报告让人看清失败，同时保留机器契约

所有新增报告先生成同一个受限结构，再投影 JSON 或 Markdown；不能运行两遍评估。
Markdown 首屏固定显示总体状态、planned/started/completed/not_started 计数、首个阻塞
阶段与原因；其后为项目、Task、失败 assertion ID 表和必要的作用域说明。
可选附完整 JSON 诊断，但不能再仅有一整块 JSON。新功能不重写旧保存格式。

错误码按闭集分类：INPUT_INVALID、INPUT_UNAVAILABLE、RESOURCE_LIMIT、
REFERENCE_MISMATCH、QUALITY_NOT_QUALIFIED、ASSESSMENT_FAILED、
DEST_EXISTS、WRITE_FAILED、PUBLISH_FAILED、CLEANUP_FAILED、
CANCELED_OR_TIMED_OUT、OUTPUT_FAILED，分别加 PROJECT/CAMPAIGN 前缀。
内部现有错误映射到这些有限码；诊断不得直接透传底层文件名、表达式、结果值。
reason 可细分，但属于版本化闭集；未知内部错误映射执行失败，绝不猜成功。

说明必须区分：用例 matched 与 Agent 成功、相对无回归与绝对达标、metadata 观察与
replay 核验、已发布与通过门禁、内容匹配与来源鉴证、历史记录与重新核验。
表格只使用通过安全校验的 ID，固定字符串必须正确转义；禁止把任意文本塞入 Markdown。
JSON/Markdown 中的决策、分母、原因集合必须严格等价；stdout 短写或失败非零。

## 10. W7：先有测量，再优化重复准备与重放

必须新增可重复的准备与评估 benchmark：6/12/16 pairs、小/中/接近限额 bundle、
重复同一路径与不同路径、有效/缺失/损坏证据；记录 Go/runtime、OS/arch、执行 profile、
数据规模、ns/op、B/op、allocs/op。时间测量只用于性能报告，不混入确定性证据格式。
不得用 race CI 时间作吞吐基线，也不预先宣称提速百分比或硬 RSS 配额。

候选优化是一次操作内按已规范化路径冻结/解码一次，复用只读对象；调用方不能拿到
可变 bytes/map。跨路径不做内容寻址去重，跨操作不做验证缓存。准备后改磁盘不会
改变该次执行；下一次调用必须重新读取。资源预算显式区分实际输入读取与持有内容，
缓存命中仍计入逻辑引用/Trial 分母，不借缓存绕过数量、提取、输出或时间限制。

回放沿用当前 pair 的完整语义核对后复用；不因 Task ID 相同而跳过 bundle/terminal
核验。重放改变输入后必须失效；并发写入不在受信任静止根保证内，仍不得复用旧成功。
测试优先精确断言读/解包/replay 次数及结果一致；性能 CI 不用不稳定毫秒阈值拦截。
基线至少五次独立采样，报告中位数与范围；发现退化先解释根因再决定是否保留优化。
如果优化无可测收益或增大内存，交付测量与明确结论，不能为凑“优化项”增加复杂缓存。

## 11. W8：统一限额、路径与故障语义

| 范围 | 上限/规则 |
|---|---|
| 项目/活动 manifest | 各 64 KiB、深度 16 |
| Task/独立 Task | 保留 256 KiB 单项，独立 Task catalog 保留原限额 |
| 项目总冻结输入 | 128 MiB raw + 128 MiB extracted，含独立 bundle；各旧组件更小限额继续生效 |
| Campaign 总冻结输入 | 256 MiB raw + 256 MiB extracted；按实际持有不同规范路径计数，最多四组 |
| 单 bundle/成员 | 保留原 32 MiB compressed/64 MiB extracted 及成员数/大小上限 |
| 项目输出 | suite 128 MiB + metadata 4 MiB；单报告最多 1 MiB |
| Campaign 输出 | 最多 4 × 132 MiB + campaign metadata 4 MiB，全部累计计数 |
| 执行时间 | 单项目 120 秒；campaign 整体 480 秒且每组最多 120 秒 |
| 执行并行度 | 项目、case、trial 均串行；不新增后台任务 |
| 无效/停止项 | 仍计入 planned；不“超限后截断继续成功” |

raw 是被保留的原始输入字节，extracted 是被保留的 bundle 解压成员字节；不是进程
RSS。使用共享计量对象时不得既漏记独立审阅 bundle，又对同一冻结对象重复扣持有量。
实现须用小预算注入测试覆盖边界，不必为每项测试真的分配数百 MiB。

只接受 root 内可见无 symlink 的常规输入和新输出位置；独占创建、Write/短写/Sync/
Close/Link/Remove/Mkdir 每一步都检查。归档传输仍使用 0074/0075 的已有规则。
受限 filesystem 不支持 hard link 时明确失败，不降级为覆盖式 rename。
静态检查失败零写入；已 claim 后失败保留本次部分产物，工具不自动删掉用户证据。
测试结束只清理测试自己创建的临时资源，绝不清扫未知进程或现有数据目录。

输入、解码对象、导出报告都执行现有敏感数据策略；测试用转义合成哨兵，不用真实
credential/生产 trace。错误与日志只含有限分类。若取消已发生，必须在打开无效根之前
返回取消；不能把取消错报成路径错误，更不能在取消后继续发布。

## 12. 验收矩阵：每条都需要可执行证据或明确交付记录

下列名称是设计时的验收入口；实际实现和已执行测试逐项记录于交付台账，不能仅凭名称推断通过。

| ID | 必须证明的结果 | 拟议证据入口 |
|---|---|---|
| A00 | 旧实现进入 main 集成候选且保留 main 依赖；最终 head 的 CI 独立核验 | PR/祖先关系/CI 记录 |
| A01 | manifest 未知/重复/null/类型错、路径越界拒绝；最后一项坏时零执行零写入 | `TestProjectClosedAdmission` |
| A02 | check 不开 world/listener、不读凭据、不写目录 | `TestProjectCheckReadOnly` |
| A03 | cases/suite/catalog 集合或同 ID 定义不一致拒绝 | `TestProjectInputCoverage` |
| A04 | Task、bundle 自引用/别名拒绝；独立副本仍不宣称身份可信 | `TestProjectReferenceSeparation` |
| A05 | 冻结后改 Task/bundle/mock/witness，不改变本次执行 | `TestProjectFrozenInputs` |
| A06 | ZIP 非业务包装变化等价；manifest/member bytes 变化不等价 | `TestReviewedWorldEquality` |
| A07 | 一致替换 fixture/工具规则，旧 Task 绑定可通过，新世界绑定必须失败 | `TestCoherentWorldSubstitution` |
| A08 | runtime 不兼容/坏 terminal 不进入 matched，分母完整 | `TestProjectVerifiedWorldBinding` |
| A09 | qualification 失败不启动 suite，not_started trials 可见 | `TestProjectQualityGate` |
| A10 | 正常单项目完成所有阶段；明确拒绝任务的 positive 正确处理 | `TestProjectEndToEnd` |
| A11 | 普通任务失败继续后续 trial，整体 published+failed；不得重新挑样本 | `TestProjectFailureAccounting` |
| A12 | claim 后中断输出 partial；最终发布前进程退出不能伪称成功 | `TestProjectProcessExit` |
| A13 | 只读 inspect 重放 suite，但 qualification 只能 recorded_only | `TestProjectInspectionScope` |
| A14 | campaign 全量预检后才启动；最后一组坏使前面零运行 | `TestCampaignAllInputPreflight` |
| A15 | gate 失败继续其他组；cleanup/取消停止，固定分母和顺序不变 | `TestCampaignStopPolicy` |
| A16 | 汇总不能用部分组、均值、重复 ID 或掉失样本通过 | `TestCampaignDenominators` |
| A17 | 四组六任务/48 trial 全部跑通；坏候选精确定位并使总门禁失败 | CLI campaign 集成测试 |
| A18 | 24 Tasks、至少 84 witness、所有 oracle 正负覆盖、12 条合法替代轨迹 | asset admission + qualify 测试 |
| A19 | 各任务错对象/额外副作用负例有效，无 oracle 放水 | 每 Task 的 case 矩阵 |
| A20 | Markdown/JSON 相同结论/计数；缺失项、有限原因、表格安全 | `TestProjectReportProjection` |
| A21 | stdout 短写/错误优先，已发布结果不回滚 | CLI 故障注入 |
| A22 | 复用冻结输入减少实际读/解包次数，跨调用不缓存 | `TestPreparedProjectReadAccounting` |
| A23 | 当前 pair 核验复用正确，任何证据变化重新验证 | `TestVerificationCacheInvalidation` |
| A24 | 性能基线可重复并给出优化前后数据，不混入评分证据 | `BenchmarkProjectPrepare/Assess` |
| A25 | 每个字节/数量边界 N、N+1、累计耗尽保留分母 | 注入式资源测试 |
| A26 | 取消覆盖读取、quality、trial、replay、发布；无遗留执行者 | 生命周期测试 |
| A27 | Mkdir/Create/Write/短写/Sync/Close/Link/Remove 的状态忠实 | filesystem 故障矩阵 |
| A28 | 转义隐私哨兵不会回显；陌生成员、路径/文件别名受控 | privacy/path 测试 |
| A29 | 旧 suite/assessment/archive fixture 语义与 CLI 行为不变 | 既有全量回归测试 |
| A30 | 干净检出按指南完成 check/run/inspect/export/import，输入未被改写 | CLI 文档流程测试 |
| A31 | 最终代码 gofmt/vet/race，Linux/Windows/macOS、hermetic、conformance 全通过 | 实际提交 CI |

错误路径尽量用库级故障注入和小预算测试，完整端到端用于关键闭环；不要把每个显示
组合都执行一次完整 48-trial campaign。不能通过跳过 race/隐私/negative 或增加业务
超时来让 CI 变绿。CI 测试框架的总超时与业务操作 deadline 必须分别记录。

## 13. W9：实施顺序、文档与交付停止条件

实施顺序：W0 → W1/W2 → W3 → W4；W5 资产可在 W1/W2 接口稳定后开始；W6/W7
随闭环接入，W8 全程补证，W9 对最终候选收尾。每个 PR 的描述都须列出本 Spec
已完成与未完成的验收 ID，禁止把任一子 PR 标题写成“全部完成”。

必须维护一张简洁台账：`workItem / acceptanceIds / implementationFiles / executedTests /
result / headRevision / remaining`。result 只取 not_started/in_progress/passed/blocked。
没有执行的测试留空并说明，不抄历史成功；没有新疑点不反复重跑已通过的同提交检查。

文档交付：新入口 help、两领域入门指南、24 任务说明、错误处理表、格式与资源表、
旧命令兼容说明；docs/README、IMPLEMENTATION-STATUS、backlog 与 PR 指向一致。
当前能力声明只在代码和对应测试完成后更新；分支就绪与 main 合并分别记录。

**全部代码交付完成**必须同时满足：W0–W9 完成；A00–A31 有结果；本批功能与旧回归
通过；同 head 的完整 CI 通过；PR 已推送并指向 main 或有明确且可审查的主线集成
依赖；临时产物已清理。用户未授权的最终 merge/live/发布不冒称已做。

只要还有已授权且可执行的工作，就继续推进。一个子项受外部条件阻塞先完成其他
子项，最终明确列出已尝试方法和缺少的条件；不得把该条件扩大成整批停工理由。
设计阶段已结束：2026-10-02 用户明确要求整批实施并完成 PR。本批按上述全部完成条件交付。

## 14. 与既有路线的关系

本批承接 B02/B03/B09-offline/B10-offline/B14/B30/B32 的有限可执行部分，强化
0066–0076 的使用闭环；不宣布这些整个 backlog 工作包关闭。
真实模型六任务、外部使用者 first-value、HostProfile 来源、真实上游差分、L2、
自动保留删除、database migration 和 stable release 仍各按原有接受/证据条件推进。
它们不是本 Spec 暗藏的验收前置，也不能用本地 mock 成功替代。

后续明确实施本 Spec 时，默认范围是 W0–W9 的整体，不再按每两个子项向用户重复
索要“继续”。如实施发现新的契约冲突，应给出具体受影响条款和可执行修订，而非
自行删去困难项目或无条件扩张到其他产品线。


## 15. 实施修订：受限评分视图变异

保留旧六个 issue Task 的定义与 oracle 后，发现两项不可同时通过普通调用满足的条件：
`allowed-business-changes` 受工具 authority 保护，无法用越权调用制造实际状态变化；
`create-issue` 的 policy 要求必须新增一条记录，因此空轨迹同时违反 goal 和 policy，
不能作为要求 `task_failed` 的 goal-negative。不能通过放宽旧 oracle 或伪称覆盖解决。

增加显式 `statetwin.dev/task-cases/v1alpha2` / `synthetic-oracle-mutation-cases-v1`。
旧 v1 格式和无变异 case 的行为保持不变。v2 的 goal-negative/policy-negative 可以带
最多四个 `{entity,key,field,value}` 替换；定位分量最长 128 字符，值最长 1024 字符，
只替换字符串，不新增/删除字段、对象，不支持路径表达式、随机值或脚本。entity/field
必须是准入 fixture 中已有的字符串字段；key 可以由 witness 创建，但评分时该精确
记录与字段必须存在，否则 case 为基础设施失败并停止后续执行，不能凭空造记录。
已有目标的替换值必须不同；相同 selector 不得重复；positive 禁止变异。

流程为真实隔离 witness 完成且清理成功 → 深拷贝评分视图 After → 应用替换 → 原 oracle
重新评分。原世界、证据和 Task 不变。case 行标注 `gradingSource=synthetic-view-mutation`。
这些是评分器的合成反例，不是 Agent 行为轨迹或业务执行证据。质量报告说明覆盖来源；
独立 suite 的任务通过仍只由真实执行/replay 决定。v2 的 unscorable-negative 还允许
真实 evaluator 的 `not_evaluated` 结果，但有错误的 check 不计语义负覆盖。

当前资产含 24 个业务 Task、103 个 case 行、97 个不同 witness 文件、18 个不同 Task
的第二条合法轨迹；其中 7 个 case 使用上述变异。变异数不计入业务任务数，且不替代
每 Task 的正常越权/错误对象反例。额外的重复操作、组合任务只做一半、缺少确认等
用例超过最低数量要求。旧业务 Task 字节未改写。
