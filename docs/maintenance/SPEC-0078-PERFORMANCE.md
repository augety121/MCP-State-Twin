# SPEC-0078 插件性能与 soak 记录

## 2026-10-23 运行期准入复用与可重算判定

基线为已合并 PR #22（`283d39b`），本轮仅复用一次操作内的已验证定义：world 持有独立的已解码 TwinSpec 和编译 oracle，结束评分复用该 oracle；Verify 使用刚完整解包并比对的 expected bundle 创建全新世界。RunWitness 同步优化，不能用旧版 RunWitness 数值当新倍率的分母。每个 world 的 store/runtime/session、Task、TwinSpec、故障状态均独立；真实输入读取、authority、资源和原始证据重放没有省略。

Pack 准入另外复用完全相同 raw bytes 的 reviewed Task 解码，以 Task ID 筛选候选后仍完整比较定义，对同一非空 artifact 指针省去重复内容遍历。物理输入仍分别读取、排除硬链接并扣预算；格式重排仍按语义比对。没有跨 session 缓存、文件 hash 清单或扩大原资源门槛。

同机 Windows/amd64、AMD Ryzen 7 5700X3D、Go 1.26.5、GOMAXPROCS=1、GOMEMLIMIT=512MiB，顺序运行 `go test -p 1 ./internal/baselinepack -run '^$' -bench '^BenchmarkPackPreparation$' -benchmem -count=3`：

| 指标 | PR #22 基线三轮 | 本轮三轮 |
|---|---|---|
| 单项 ns/op | 152887543 / 149484043 / 147444450 | 142558812 / 136200562 / 141721512 |
| 24 项目录 ns/op | 612790900 / 609915250 / 606000150 | 570168950 / 619688600 / 570271850 |
| 24 项目录 B/op | 57504320 / 57504540 / 57501076 | 50725308 / 50723872 / 50726132 |

目录中位约 609.915→570.272ms（约 6.5%），分配约 57.50→50.73MB/op（约 11.8%）；耗时范围重叠，不宣称统计显著性或任意硬件 SLA。新增 `BenchmarkPluginPreparation` 以三次迭代建立后续回归入口：read/close/after-commit 的 lifecycle 为 38.333/45.013/45.293ms，replay 为 52.915/51.949/52.715ms，当前 RunWitness 为 17.181/17.787/21.885ms。这组没有旧版本同名测量，不宣称对应提升百分比。

新增 `plugin assess-performance` 和 `plugin-performance/v1alpha2`。采样保存顺序 pair ID、两侧原始秒数、阶段数据、CPU/Go/build/workload 及 race 状态，严格检查字段、数量、时长和阶段总量；报告使用 nearest rank。沿用启动 p95≤2 秒、两组 p95 比≤1.5、两组 p95 差≤2 秒的原解释；逐对差的 p95 仅是诊断字段。历史 Go/Inspect v1alpha1 报告不可独立重算，判为 incomplete，保留原文件而不补造数据。范围限 Go stdio 合成 contract、本地操作员声明来源，不能外推为 Inspect、模型或发布资格。

本轮 100 次独立会话通过（测试主体 75.23 秒），前 30 次配对；原始记录为 [SPEC-0078-PERFORMANCE-RUNTIME.json](SPEC-0078-PERFORMANCE-RUNTIME.json)，buildLabel 如实为 `283d39b+working-tree-runtime-efficiency`、runtimeRevision 仍为 unknown。完整链路 p50=0.697740s、p95=0.753283s、max=1.461288s；当前 RunWitness p50=0.017332s、p95=0.020638s、max=0.022418s。启动 p95=0.291899s，p95 差=0.732644s，这两条通过；p95 比=36.4993，仍大于 1.5，命令判定 failed。逐对开销 p95=0.737232s，不用于替换门槛。子进程观测峰值 working set 最大 33,382,400 bytes，父 goroutine 2→3。

复核：`go run ./cmd/statetwin plugin assess-performance --root docs/maintenance --performance-report SPEC-0078-PERFORMANCE-RUNTIME.json` 输出上述失败，返回非零属于符合预期的验收结果。新的进程内基线更快，因此即使完整链路耗时减少，倍率也可能升高；不得冻结旧分母或以 soak PASS 冒充 AB31 通过。该结果不升级原先 Inspect 的历史数据；完整性能与外部证据门槛仍未完成。

## 同日优化复测

在原始候选 `4dc0c6e` 基础上，单次 Prepare 内复用字节完全相同的 bundle 解码及敏感数据扫描，仍逐物理输入扣除资源预算；去除已由 case 准入完成的重复 Task 编译。case 缓存按 root 和引用共同隔离，新增不同根引用和无效 oracle 反例。没有跨会话缓存或省略独立进程核验。

同一台 AMD Ryzen 7 5700X3D、GOMAXPROCS=1，顺序执行原提交源码与修改后源码的 `BenchmarkPackPreparation -benchtime=3x -count=3`，均不启用 profiler，fixture 构建排除在计时外。24-entry 准入三轮中位数从 806.205ms 降至 606.129ms（约 24.8%），范围分别为 787.239–814.960ms 和 591.176–619.345ms；分配从约 86.47MB/op 降至 57.51MB/op。单 entry 中位数 179.014ms→152.872ms，范围重叠；三轮微基准不构成硬件 SLA 或统计显著性结论。

优化后重新执行 100 次会话（75.93 秒），记录前 30 次配对样本：完整链路 p50=0.751730s、p95=0.884805s、max=1.463888s；预加载 RunWitness p95=0.035217s；启动 p95=0.303452s；新增 p95=0.849588s，倍率 25.1242。子进程峰值 working set 最大 43,839,488 bytes，父 goroutine 2→3。原始数据见 `SPEC-0078-PERFORMANCE-OPTIMIZED.json`。

Inspect 30 次完整框架复测通过（63.22 秒），p50=1.883136s、p95=1.974539s、max=4.785316s；范围与下文原测量相同，保留首轮较慢样本，启动/RSS 仍未分别测量。见 `SPEC-0078-INSPECT-PERFORMANCE-OPTIMIZED.json`。相对本次 Go 进程内 p95 的诊断差值为 1.939321s；这是跨运行差值，不能代替严格配对验收。原始≤1.5倍门槛仍失败，AB31 仍未通过，不修改门槛或宣称 stable。

## 原始候选测量（保留历史）

2026-10-03；Windows/amd64，AMD64 Family 25 Model 33 Stepping 2，Go 1.26.5，GOMAXPROCS=1。独立子进程，单 Task pack（issue read）、Go SDK 1.8.0、合成 witness；没有 provider 请求。测量源码 `cmd/statetwin/plugin_soak_test.go` 和 `internal/plugin/contract.go`。

执行 `STATETWIN_PLUGIN_SOAK=1 go test -p 1 ./cmd/statetwin -run '^TestPluginReleaseSoakAndPerformance$' -count=1 -timeout=8m -v`，100 次 session 完成并等待自有子进程退出，耗时 88.98 秒；前 30 次记录分阶段数据，每次配对原 RunWitness。另一个确定性测试覆盖 5 个阶段各两次强制 crash 并验证 Wait 完成。

| 指标 | p50 秒 | p95 秒 | max 秒 |
|---|---:|---:|---:|
| 插件完整链路 | 0.815081 | 0.898240 | 0.902616 |
| 预加载 Task/world 的原进程内 RunWitness | 0.025220 | 0.033129 | 0.035809 |

启动 p95=0.344377 秒；完整链路减原进程内 p95=0.865112 秒；p95 倍率=27.1138。30 个子进程峰值 working set 最大 43,692,032 bytes，RSS 测量状态 measured；父进程 goroutine 从 2 到 3（包含 SDK 常驻资源），100 次后未超出测试的 +4 上限。该检查不能证明无限期运行内存不增长，也没有测量父进程总 RSS。

分阶段近似 p95：父进程完整准入 0.18 秒、子进程启动/独立准入/MCP/控制握手 0.34 秒、工具 0.01 秒、评分封存 0.12 秒、退出回收 0.01 秒、只读重验 0.27 秒。分位数不可直接相加。原始 30 条测量保存在同目录 `SPEC-0078-PERFORMANCE.json`。

主要开销位于完整资产准入、子进程独立准入、评分和证据重放；实际工具调用只占约 5ms。现有优化只在单次准入中复用冻结原始输入、已打开 bundle 和相同 case 组；没有跨 session 复用可变文件，也没有省略 reviewed/quality 引用检查。以预加载的 33ms 进程内世界为分母，1.5 倍仅允许约 50ms 完整链路，低于当前任一个完整准入阶段。简单消除日志或工具调用开销不能满足该门槛；不能为使数字通过而改分母、延长 Task deadline 或绕过独立核验。

原 Spec 12.2 的启动≤2秒、新增≤2秒通过；总耗时≤1.5倍失败。因此 AB31 及依赖它的发布资格保持未通过。门槛修订已询问用户，尚未获明确同意；此记录不修改 Spec。若维持原门槛，需要另行完成准入/评分编译复用与等价性验证后复测。

Inspect 另执行 30 次真实 Task/Solver/Scorer + 独立 MCP 子进程（mock 模型、每次新 session），测试耗时 64.86 秒；Python 3.12.3 / Inspect 0.3.275。完整框架链路 p50=1.952603 秒、p95=2.194704 秒、max=4.229975 秒。范围包括计划/投影、eval、评分、scorer 重放、日志与清理，排除 fixture 构建和 Python 初次导入；启动与框架 RSS 未分离，保持 unavailable。原始样本见 `SPEC-0078-INSPECT-PERFORMANCE.json`。相对上述原进程内 p95，框架新增约 2.161576 秒，新增≤2秒也未通过。不同运行的这个差值只是候选诊断，不替代同批严格配对测量。

复现：设置 `STATETWIN_INSPECT_PERFORMANCE=1`、`STATETWIN_TEST_BINARY`、`PYTHONUTF8=1`，运行 `python -m pytest adapters/inspect/tests/test_session.py -k framework_performance -q`。普通 CI 跳过此性能测试，仍执行五项功能/安全边界测试；性能阈值由候选报告明确判断，不因 opt-in 测试返回成功就宣布达标。没有任意硬件 SLA 或性能提升宣称。
