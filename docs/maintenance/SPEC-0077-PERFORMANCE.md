# SPEC-0077 性能记录

2026-10-02，Go 1.26.5，Windows/amd64，AMD Ryzen 7 5700X3D，GOMAXPROCS=1。
单次操作串行；不启用 race/coverage，不使用 CI race 耗时推算吞吐，不声明 RSS 硬配额。
这是开发工作树的本机实验。应用闪退后保留完整五样本组并恢复未完成的矩阵；相同配置
采用同批完整五次样本，不拼接不足五次的组。后台系统负载未做硬隔离，范围如实列出。

## 方法

```powershell
$env:GOMAXPROCS='1'
go test ./internal/agenteval -run '^$' -bench '^BenchmarkProjectPrepare' -benchtime=1x -count=5 -timeout=60m
go test ./internal/agenteval -run '^$' -bench '^BenchmarkProjectAssess' -benchtime=1x -count=5 -timeout=30m
```

Prepare 覆盖 6/12/16 pairs × 三个 bundle 大小 × 重复/独立 Task、mock 路径 × 缓存命中
开关，共 36 格，每格五次。对照用同一校验代码禁用操作内缓存命中，不是未经验证的
旧版本吞吐推测；两边使用相同有效输入、断言与旧组件限额。所有缓存都在单次操作结束
后释放，操作间仍重新读取；独立审阅 bundle 始终单独持有和比较，不做内容寻址去重。

distincttrue 为每 pair 使用不同 Task/mock 文件路径，bundle 仍由同领域 Task 共享，
独立世界参考使用另一条物理路径。small 是实际扩展样例；medium 为 fixture 追加
64 KiB JSON 空白；near-suite-limit 追加 3 MiB 空白。它保持业务状态相同，测量原始
成员处理成本。16 pairs 的逻辑展开超过 48 MiB，接近旧 suite 64 MiB 累计预算；
不通过放大单 bundle/成员上限构造大样本，也不声称这个数据集接近单包压缩字节上限。
Benchmark 只测静态准备，使用无外层执行 deadline 的 benchmark context，不承诺任意
接近限额的项目能在真实 120 秒执行预算内完成；CLI 仍执行原 deadline。

Assess 覆盖 6/12/16 pairs 的正常、缺失、损坏 terminal，共九格，每格五次，使用真实
已落盘的小样例 suite 重放。缺失/损坏样本仍保持 planned 分母；损坏样本的更短耗时来自
明确失败和提前停止，不是成功路径提速。源构建与 suite 初次运行不计入操作计时。

原始五次数值见[样本 JSON](SPEC-0077-PERFORMANCE-SAMPLES.json)。

## 准备阶段：中位数与五次范围

B/op 为 Go 累积分配字节，不是峰值常驻内存；allocs/op 为分配次数。

| 配置 | ns/op 中位数（min–max） | B/op 中位数 | allocs/op 中位数 | reads/op | decodes/op |
|---|---|---|---|---|---|
| pairs6/small/distinctfalse/cachefalse | 420596800 (398171800–430797300) | 50174568 | 884962 | 77 | 13 |
| pairs6/small/distinctfalse/cachetrue | 302293400 (297511300–311401900) | 40809872 | 728218 | 60 | 2 |
| pairs6/small/distincttrue/cachefalse | 402915700 (396262900–415684400) | 50150784 | 884535 | 77 | 13 |
| pairs6/small/distincttrue/cachetrue | 317437800 (310436600–320540800) | 40969456 | 732771 | 66 | 2 |
| pairs6/medium/distinctfalse/cachefalse | 995048100 (970255300–1078981000) | 71007320 | 885489 | 77 | 13 |
| pairs6/medium/distinctfalse/cachetrue | 642329100 (638450200–654322000) | 51010624 | 728492 | 60 | 2 |
| pairs6/medium/distincttrue/cachefalse | 974633500 (963722300–990390500) | 70994176 | 885124 | 77 | 13 |
| pairs6/medium/distincttrue/cachetrue | 635839500 (619547300–650707900) | 51173976 | 733057 | 66 | 2 |
| pairs6/near-suite-limit/distinctfalse/cachefalse | 25959632200 (25676344800–28417278500) | 821327152 | 886194 | 77 | 13 |
| pairs6/near-suite-limit/distinctfalse/cachetrue | 15221269600 (14825068000–16196909700) | 403622112 | 728940 | 60 | 2 |
| pairs6/near-suite-limit/distincttrue/cachefalse | 26519385200 (26106267500–26821886200) | 821302200 | 885797 | 77 | 13 |
| pairs6/near-suite-limit/distincttrue/cachetrue | 15005610900 (14933246500–15119358500) | 403795704 | 733560 | 66 | 2 |
| pairs12/small/distinctfalse/cachefalse | 645353300 (632710300–651219800) | 81159880 | 1436184 | 101 | 19 |
| pairs12/small/distinctfalse/cachetrue | 461182200 (447473700–480527900) | 66346328 | 1185939 | 60 | 2 |
| pairs12/small/distincttrue/cachefalse | 621022200 (614150600–625637000) | 81120536 | 1435415 | 101 | 19 |
| pairs12/small/distincttrue/cachetrue | 464725400 (452820600–473704800) | 66926736 | 1200868 | 84 | 2 |
| pairs12/medium/distinctfalse/cachefalse | 1556331000 (1535898600–1615961300) | 113739504 | 1437011 | 101 | 19 |
| pairs12/medium/distinctfalse/cachetrue | 1003398600 (983355300–1021826800) | 82482120 | 1186295 | 60 | 2 |
| pairs12/medium/distincttrue/cachefalse | 1620302700 (1526796100–2230704300) | 113697096 | 1436222 | 101 | 19 |
| pairs12/medium/distincttrue/cachetrue | 1044200000 (1027995700–1058265800) | 83074368 | 1201282 | 84 | 2 |
| pairs12/near-suite-limit/distinctfalse/cachefalse | 46034441500 (43238934000–47680440200) | 1282689336 | 1437945 | 101 | 19 |
| pairs12/near-suite-limit/distinctfalse/cachetrue | 26720721200 (25229441900–27143290100) | 636840224 | 1187014 | 60 | 2 |
| pairs12/near-suite-limit/distincttrue/cachefalse | 42892111100 (41400496900–45657310200) | 1282646616 | 1437167 | 101 | 19 |
| pairs12/near-suite-limit/distincttrue/cachetrue | 24396448200 (24333606400–25143550400) | 637433696 | 1201988 | 84 | 2 |
| pairs16/small/distinctfalse/cachefalse | 792417200 (774539900–815305100) | 102302456 | 1809011 | 117 | 23 |
| pairs16/small/distinctfalse/cachetrue | 545231700 (537557400–549336800) | 83858448 | 1496352 | 60 | 2 |
| pairs16/small/distincttrue/cachefalse | 779774900 (759509600–802410000) | 102253232 | 1808001 | 117 | 23 |
| pairs16/small/distincttrue/cachetrue | 576370800 (565936500–600144200) | 84728896 | 1518310 | 96 | 2 |
| pairs16/medium/distinctfalse/cachefalse | 1928408800 (1923374800–2012357200) | 142714312 | 1810013 | 117 | 23 |
| pairs16/medium/distinctfalse/cachetrue | 1245815400 (1227858400–1290583800) | 103965760 | 1496851 | 60 | 2 |
| pairs16/medium/distincttrue/cachefalse | 1916934100 (1883354100–1988685900) | 142662568 | 1809012 | 117 | 23 |
| pairs16/medium/distincttrue/cachetrue | 1314651800 (1300066900–1333838300) | 104832616 | 1518781 | 96 | 2 |
| pairs16/near-suite-limit/distinctfalse/cachefalse | 54990084900 (52604798100–58959849000) | 1590753032 | 1811089 | 117 | 23 |
| pairs16/near-suite-limit/distinctfalse/cachetrue | 31009297400 (30697621600–31226411600) | 792790640 | 1497610 | 60 | 2 |
| pairs16/near-suite-limit/distincttrue/cachefalse | 53426102600 (52652651300–59397289300) | 1590703416 | 1810096 | 117 | 23 |
| pairs16/near-suite-limit/distincttrue/cachetrue | 30531122200 (30410871200–31197976800) | 793682072 | 1519602 | 96 | 2 |

## 重放评估：中位数与五次范围

| 配置 | ns/op 中位数（min–max） | B/op 中位数 | allocs/op 中位数 |
|---|---|---|---|
| pairs6/valid | 692513200 (672131500–881079900) | 167354368 | 2079610 |
| pairs6/missing | 816864700 (636013900–976946200) | 158300520 | 1932295 |
| pairs6/corrupt | 85938400 (74616400–95768100) | 14493696 | 192737 |
| pairs12/valid | 1361818800 (1345484300–1394881500) | 335855056 | 4137344 |
| pairs12/missing | 1323650300 (1303301800–1335647900) | 319772984 | 3990032 |
| pairs12/corrupt | 96643100 (93461500–100013100) | 14316008 | 219271 |
| pairs16/valid | 1879928300 (1797670400–1920054700) | 447168016 | 5529482 |
| pairs16/missing | 1764201000 (1737483800–1767997300) | 433405840 | 5382100 |
| pairs16/corrupt | 109065000 (106467100–111427100) | 15147824 | 236877 |

## 保留的优化与限制

保留项目级规范路径读取复用和已验证 bundle 的操作内解码复用。实际计数测试证明每条
物理路径读取一次，实际 bundle 与独立参考分别解码，跨调用重新读取。分母和旧组件
更小限额不因缓存命中减少。上表同时报告时间、分配和范围，不以不稳定毫秒阈值阻断 CI。

已有 suite audit 的当前 pair 缓存继续通过完整语义相等判断复用，未增加跨操作缓存。
`TestVerificationCacheInvalidation` 证明稳定证据每 trial 一次重放；读取间变更 terminal
会再次重放并拒绝先前成功。这里没有把已有 pair 缓存包装为新增性能优化。

仍有成本：每个 trial/Task 的 admission、工具表/表达式校验及隐私扫描不能被路径复用
替代，大输入下准备仍然昂贵。本轮没有跳过这些验证，没有新增任意脚本、全局内容缓存
或自动调高业务 deadline。性能数据不进入确定性评估报告，也不等同 Agent 能力证据。
