# SPEC-0047: 无覆盖发布构建与失败保留

Status: accepted via [ADR-0047](ADR-0047-NO-CLOBBER-RELEASE-BUILDS.md), 2026-09-12.

入口保持 `bash scripts/build-release.sh TAG`，仅支持一个 tag 参数。新增先决条件：
在 repository root、来源工作区干净、tag peeled commit 与 HEAD 一致、SPEC-0045 plan/notes
已准入。只读检查完成前不能创建输出。无任何清理参数、任意输出根或 force 开关。
`git status` 本身失败也是拒绝；先检查命令退出状态，再检查其输出是否为空，不能将
失败命令的空 stdout 当作“工作区干净”。
Git 身份读取也先独立捕获结果再比较，保留命令本身的失败码；不能只保留最外层 shell
比较失败而丢失原始原因。初次 POSIX CI 的 missing-tag 用例已复现并推动这个修复。

`umask 077` 后用普通 `mkdir dist` 独占新目录；现有目录、文件、symlink 都拒绝。
不使用 `mkdir -p` 将原目录当作本次产物，不递归删除，失败也不回滚到“空目录”。
本地文件系统仍需可信，不防御同账户恶意并发写者，也不保证硬件断电安全。

保留 Linux amd64/arm64、macOS amd64/arm64、Windows amd64 五个现有交叉编译目标。
每次只有一个 Go build，`-p 1`，未指定 GOMAXPROCS 时默认 1；设置是 Go 软控制，
不是 CPU/温度/RSS 硬配额。保持 CGO_ENABLED=0、trimpath 与 version/revision ldflags。
修订号来自实际 HEAD，而非 tag 字符串或旧 CI。编译成功不等于该架构上已运行测试；
CI 目前的 Windows/macOS/Linux hosted runner 不能自动证明每个 arm64 发布目标。

首次失败立即非零退出；不忽略 build/写盘/既有 checksum 步骤错误，不上传半成品。
失败后保留本次部分输出供检查；重新执行仍因 dist 已存在而拒绝。维护者必须先检查确切
目标和数据归属再决定后续处理，文档不提供广域删除命令。

既有发行 checksum 步骤仍只在真实授权的发行构建中运行；本次不计算文件哈希、不生成
实际发行清单，也不以新文件哈希作为修改验收。测试使用合成 Git/Go 命令，在打包末段前
注入失败，验证工作区、tag、输出拒绝、五目标构建参数和失败传播。POSIX 路径在 Linux/
macOS CI 执行，Windows 记录 skip；跨平台原生 Go policy tests 独立运行。

默认不测试真实 tag 创建、全部二进制打包/上传、签名、公证、SBOM、可重现 bit-for-bit
发行或供应链 provenance；这些不能由脚本单元测试升级为 verified。
