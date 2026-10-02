# SPEC-0078 插件性能与 soak 记录

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
