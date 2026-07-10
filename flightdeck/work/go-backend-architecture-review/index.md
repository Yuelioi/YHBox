# Index — Go 后端架构深度审查

## State

升级实施进行中。审查结论见 `review.md`，完整升级/修复方案见 `plan.md`。批次 A-C 已完成；canonical repository/module identity 已最终确认为 `github.com/yottaapp/yotta`。批次 D 已建立平台依赖守卫，并完成 platform、input、capture 的宿主 OS seam。

## Next

继续批次 D：移除 container/runtime 对 `lxn/win`、旧 input/capture bootstrap 与 winutil 的直接理解，让 Win32 能力只从 controller adapter 进入 runtime。

## Read now

- `plan.md`
- `review.md`
- `../../knowledge/agent/codex-working-agreement.md`
- `../../knowledge/build/go-cover-bom-trap.md`
- `../../knowledge/architecture/go-multiplatform-boundary.md`
- `../../knowledge/architecture/go-module-identity.md`

## Read if

- `../../knowledge/build/build.md` — 开始编译、测试、静态分析前
- `../../knowledge/nodes/node-system-architecture.md` — 审查节点注册、派发与 capability 模型时
- `../../knowledge/nodes/error-model.md` — 审查节点错误传播与恢复语义时
- `../../knowledge/nodes/contract-verification-traps.md` — 评估节点契约测试是否可能假绿时
- `../../knowledge/platform/windows-abi-pointer-copy.md` — 修改 Win32 callback/native DLL ABI 时
- `../../knowledge/testing/realtime-tests-and-scheduler-delay.md` — 修改实时时序、QPC 或回放调度测试时

## Progress

Current:

- `plan.md` 阶段 1 / 批次 D。
- 完整 Linux/Darwin build 的剩余链集中在 hotkey、winutil、container/runtime、recording/tools 与 Wails GUI 壳；input/capture package 自身已可跨平台编译测试。

Done:

- 批次 D（第二批）：新增统一 typed unsupported platform error；拆分 `pkg/input`/`pkg/capture` 公共 contract、Windows adapter 与非 Windows factory；mock capture 保持跨平台；Linux/Darwin dependency graph 无 `lxn/win`，CI 已加入三包原生测试。
- 批次 D（首批）：autostart/admin/console/shell 平台文件隔离；`pkg/platform` 三平台编译；新增 platform-neutral import 守卫测试。
- 批次 B：新增 Windows quality/race 与 Linux/macOS portable-core CI；新增只读版本一致性脚本；release 改为校验 tag，不再临时修改源码，并补齐 Node/pnpm runner 环境。
- 批次 C：module path 切换到 `github.com/yottaapp/yotta`；全量同步 Go import、前端 binding、CI、README 与知识链接；重新生成 Wails bindings，Go/前端验证全绿。
- 批次 A：移除 Go BOM；清零 29 个 staticcheck finding；用 Win32 ABI 内存复制 helper 消除 4 个 vet unsafe warning；将 inputclip 调度测试改为确定性 clock/wait；恢复 coverage 与关键 race 全绿。
- 盘点 624 个 Go 文件、59 个 package、入口、依赖方向、平台 adapter、持久化、并发/生命周期与 CI。
- 完成节点框架、controller、runtime Ctx、execution、settings/container store 的重点审查。
- 核对 README、release workflow 与节点 knowledge，记录明确漂移。
- 产出 `review.md`，给出分级问题、目标架构与四阶段治理顺序。

Verified:

- 当前分支为 `main`，启动审查前工作区干净。
- 隔离 `go test ./... -count=1` 通过；inputclip runtime `-count=20` 通过。
- services/container/runtime/execution/schedule/inputclip/hotkey 的 race 测试通过。
- Linux/macOS 交叉编译失败于 Win32 依赖；`go vet` 4 处 unsafe warning；staticcheck 非零。
- `staticcheck ./...`、`go vet ./...`、`go test -cover ./...` 已通过。
- inputclip runtime `-count=20` 与关键 package race 已通过。
- `go list -m` 返回 `github.com/yottaapp/yotta`；旧 owner 引用为 0；Wails bindings 只生成在新 module 目录。
- module/path/platform 首批改动后，`go test ./... -count=1`、`go vet ./...`、`staticcheck ./...`、`go test -coverprofile=coverage.out ./...`、关键 package race、`task version:verify` 与 `git diff --check` 均通过；coverage profile 已清理且 `/coverage.out` 已加入 `.gitignore`。
- `pnpm -C frontend typecheck` 与 67 个测试文件、527 个 Vitest 通过。
- D 第二批后，Windows `go test ./... -count=1`、`go vet ./...`、`staticcheck ./...`、受影响包 race/coverage 均通过；Linux/Darwin 的 input/capture/platform tests 可交叉编译且 dependency graph 无 Windows 包。

## Open questions

- 外部扩展目标是只接受 in-tree contribution，还是允许 out-of-tree Go module / plugin？
- 首批正式支持的平台矩阵是什么：Windows + Linux，还是 Windows + Linux + macOS？Android/Browser 是 target adapter，不等同于宿主 OS 支持。
- Container package 的崩溃一致性目标采用 generation directory，还是较轻的 lock-last commit + load-time validation？
- Wails dependency 固定为 alpha.91，而本机 CLI 是 alpha2.112；进入跨平台 GUI 构建前需要统一 CLI/library 版本并确认 Linux CGO/WebKit 构建基线。
- Wails bindings 虽生成成功，但仍对 function type 和暴露给绑定的 non-empty interface 参数报告 10 条 JSON 编码警告；后续应收窄可绑定 service surface，避免把仅供 Go 内部装配的方法暴露给前端生成器。
