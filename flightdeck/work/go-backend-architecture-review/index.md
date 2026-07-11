# Index — Go 后端架构深度审查

## State

升级实施进行中。审查结论见 `review.md`，完整升级/修复方案见 `plan.md`。批次 A-C 已完成；批次 D 的本地平台 seam 与 GUI compile gate 定义已完成，首次远端 runner 验证待推送；批次 E 的后台与交互资源已纳入统一应用生命周期。

## Next

继续批次 E：审计 executable App/LogSink 最终 drain、活动 automation held-input 释放与退出故障注入；同时保留 Linux/macOS GUI compile gate 首次远端运行与宿主 smoke 待办。

## Read now

- `plan.md`
- `review.md`
- `../../knowledge/agent/codex-working-agreement.md`
- `../../knowledge/build/go-cover-bom-trap.md`
- `../../knowledge/architecture/go-multiplatform-boundary.md`
- `../../knowledge/architecture/go-module-identity.md`
- `../../knowledge/architecture/application-lifecycle.md`

## Read if

- `../../knowledge/build/build.md` — 开始编译、测试、静态分析前
- `../../knowledge/nodes/node-system-architecture.md` — 审查节点注册、派发与 capability 模型时
- `../../knowledge/nodes/error-model.md` — 审查节点错误传播与恢复语义时
- `../../knowledge/nodes/contract-verification-traps.md` — 评估节点契约测试是否可能假绿时
- `../../knowledge/platform/windows-abi-pointer-copy.md` — 修改 Win32 callback/native DLL ABI 时
- `../../knowledge/testing/realtime-tests-and-scheduler-delay.md` — 修改实时时序、QPC 或回放调度测试时

## Progress

Current:

- `plan.md` 阶段 2 / 批次 E；批次 D 的首次远端 GUI runner 验证并行待办。
- `internal/appruntime` 已接管 Worker、MCP HTTP、ScheduleDaemon、hotkey、recording、calibration 与 tools presentation；下一步闭合 App/LogSink 最终 drain 与 held-input 退出证据。

Done:

- 批次 E（第二批）：runtime ownership 扩展到 hotkey registry、recording、calibration 与 tools presentation；逆序关闭会先取消临时 capture、关闭/等待 secondary windows、停止校准与录制 hook，再释放中央 hotkey。关闭后的服务拒绝新 native 资源，录制退出 Cancel 且 finalizing drain 后不再落半成品。
- 批次 E（首批）：新增 application runtime 状态机与 managed HTTP server；构造完成后统一启动 Worker→MCP→Schedule，MCP bind error 同步失败并触发 rollback；Wails 正常/错误退出都按 Schedule→MCP→Worker 关闭。Worker 使用 lifetime context 闭合 queued→active shutdown 竞态，Worker/Schedule/HTTP Close 均幂等且响应 context。
- 批次 D（第十三批）：将 16 个启动期 callback/interface 注入方法移出 Wails service 方法集，bindings 收敛到 107 methods / 0 warnings，并让 CI 拒绝 warning 回归；同时修复 LogSink 按 seq 生成却并发乱序交付及 shutdown 不等待尾日志的问题。
- 批次 D（第十二批）：新增 Ubuntu 24.04 amd64 与 macOS 15 arm64 原生 GUI compile gate；安装宿主依赖、核对实际 Wails CLI、生成 bindings/frontend、以 production tag 编译并归档 Unix 产物。
- 批次 D（第十一批）：Wails library/release CLI/README pin 统一到 `v3.0.0-alpha2.117` 并加入一致性脚本；root template capture/game provider 移除直接 `lxn/win`；更新 GUI 宿主构建基线与失效的透明窗 knowledge。
- 批次 D（第十批）：App event transport 与 tools semantic window presenter 从 Wails 抽离；GUI options/policy 留在 executable adapter；统一 attempt/generation-aware window slot 消除并发 open/close 竞态；services/tools 进入 portable-core CI。
- 批次 D（第九批）：recording event/stop contract 与 Win32 recorder/hook/raw-input adapter 分离；service 使用 canonical target contract；非 Windows recorder 返回 typed unsupported；Linux/macOS 进入 portable-core CI。
- 批次 D（第八批）：tools 的 mouse/pixel/window-capture native 实现进入 `_windows.go`，非 Windows adapter 使用 typed unsupported；WindowResolver 改用 canonical target contract；tools 平台守卫已建立。
- 批次 D（第七批）：calibration state 与 Win32 raw-input/hotkey adapter 分离；非 Windows service 返回统一 typed unsupported；Linux/macOS 进入 portable-core CI。
- 批次 D（第六批）：hotkey manager 与 Win32 RegisterHotKey 消息循环分离；非 Windows 返回统一 typed unsupported；registry 单测改用内存 loop，Linux/macOS 进入 portable-core CI。
- 批次 D（第五批）：RuntimeContext 移除 legacy input/capture backend 字段；Win32 provider 独占后端创建、适配与释放；vision/cursor 全部经 controller capability；新增 controller factory 注入 seam、动态 capability 与 runtime legacy-import 守卫。
- 批次 D（第四批）：target contract 接管 WindowHandle/WindowMatchSpec；runtime core 移除直接 `lxn/win`/winutil import；winutil 拆分 Windows/non-Windows adapter；container/runtime tests 可为 Linux/Darwin 编译，失败图从 14 收敛到 5。
- 批次 D（第三批）：提取跨平台 VK 解析；非 Windows `KillProcess` 使用 typed unsupported；`internal/nodes/input` 与 `internal/nodes/io` 在 Linux/Darwin 编译通过并进入 CI 原生测试矩阵。
- 批次 D（第二批）：新增统一 typed unsupported platform error；拆分 `pkg/input`/`pkg/capture` 公共 contract、`_windows.go` adapter 与非 Windows factory；mock capture 保持跨平台；Linux/Darwin dependency graph 无 `lxn/win`，CI 已加入三包原生测试。
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
- D 第二批双轴 review：Standards 无 finding；Spec 指出的 Windows mock invalid-HWND 语义回退与 `_windows.go` 命名未闭合均已修复，并补 Windows 回归测试及全仓旧路径检查。
- D 第三批后，Windows 全量 test/vet/staticcheck、受影响包 race/coverage 通过；Linux/Darwin 的 nodes/input、nodes/io、pkg/input、pkg/platform 编译通过。
- D 第四批后，Windows winutil/container/runtime 定向测试通过；Linux/Darwin 的 winutil/container/runtime 测试可交叉编译；架构守卫禁止 runtime core 重新直接 import Win32/winutil。
- D 第四批双轴 review：Standards 3 项、Spec 2 项均已修复——恢复导出契约注释、borderless state 编译期类型、target 构造归属、BringToFront typed unsupported，并补齐 gofmt 与回归测试。
- D 第五批后，Windows 全量 test/vet/staticcheck、受影响包 race/coverage 通过；CI portable-core 的 21 个 package（含 container/runtime 与 mcpserver）均为 Linux/Darwin 成功编译测试二进制；runtime core 守卫同时禁止重新 import legacy input/capture 与 Win32 packages。
- D 第五批双轴 review：修复旧符号注释、provider 单复数命名、Win32 profile/controller capability 漂移与缺 target 错误语义；测试 adapter 的少量重复保留在 `_test.go`，避免把 legacy backend import 重新带回平台中立生产代码；factory 注入 seam 保留给 embedder，并明确资源仍由调用方持有。
- D 第六批后，Windows hotkey test/race 通过，Linux/Darwin hotkey tests 成功交叉编译；架构守卫禁止 hotkey 非 Windows文件重新引入 Win32 packages。
- D 第六批双轴 review：清理与内存 loop 相反的旧测试注释；另修复 rebuild 持锁等待 loop 时、同步 dispatcher 再取同一 mutex 可能形成的退出死锁，并增加 in-flight dispatch 回归测试。
- D 第七批后，Windows calibration test/race 通过，Linux/Darwin calibration tests 成功交叉编译；架构守卫禁止 calibration 非 Windows 文件重新引入 Win32 packages。
- D 第七批双轴 review：将 OS-thread helper 收入 `_windows.go`；统一跨平台 package doc；移除文件名实现旁白；把裸 atomics 收进单一 calibration state store，native adapter 只通过状态行为更新快照。
- D 第八批后，Windows tools test/race 通过；Linux/Darwin 编译已不再命中 Yotta Win32 import，下一失败点稳定落在 Wails alpha.91 `pkg/application` 的 Linux CGO 实现；因此尚不加入 portable-core CI。
- D 第八批双轴 review：MousePos 改为 typed error contract 并让 HUD 显式呈现轮询/取色错误；capture 在检查 Wails app 前先判平台 capability；target tool router 统一传 canonical `target.Target`，删除 kind 字符串双权威与含混命名。
- D 第八批 binding/frontend 验证：Wails 正式生成后 MousePos 仍为 `CancellablePromise<MousePosInfo>`，`vue-tsc`、67 个 Vitest 文件/527 tests 与 production build 通过；另记录 CLI `-dry` 默认 clean 会清空 gitignored bindings 的陷阱。
- D 第九批后，Windows recording tests 通过，Linux/Darwin recording tests 成功交叉编译；架构守卫与 portable-core CI 已覆盖 recording，项目内非 main package 的 Win32 编译失败归零。
- D 第九批双轴 review：恢复 HookEvent/StopResult 的时间、判别字段、坐标与持久 ID 契约；保持 Start 无前台激活副作用、ValidateTarget 独占前台预检；清理重复 package doc，并以共享 lifecycle interface 在编译期约束各平台 Recorder surface。
- D 第九批复核后，全仓 `go test ./... -count=1`、`go vet ./...`、`staticcheck ./...`、recording race 与 Linux/Darwin 交叉编译均通过；recording statement coverage 为 34.1%。
- D 第十批后，`internal/services` 与 `internal/services/tools` 在 Linux/Darwin、`CGO_ENABLED=0` 下成功编译，tools dependency graph 不再包含 Wails；架构守卫禁止整个 `internal/services` 重新 import Wails。
- D 第十批双轴 review：修复窗口并发创建、Close 后排队 Open 重开、同批失败结果分裂、cleanup 重入死锁、打开中取消清理与旧代际 closing callback 清理新窗口的竞态；App presentation 生命周期改为 new/attached/closed 单向状态，emitter 与 LogMerger 原子发布；LogMerger shutdown 幂等 drain 并等待 worker；presentation port 使用具名 semantic window request，具体标题、路由、尺寸与材质策略归 executable adapter；最终 Standards/Spec 均无剩余 finding。
- D 第十批 binding/frontend 验证：Wails bindings 生成后 RPC 方法从 124 收窄为 123，已知 warning 仍为 10；`vue-tsc`、67 个 Vitest 文件/527 tests 与 production build 通过。
- D 第十批最终门禁：全仓 `go test ./... -count=1`、`go vet ./...`、`staticcheck ./...` 与 services/tools/architecture race 通过；tools 并发回归连续 100 次通过；Linux/Darwin 的 services/tools 交叉编译及 Wails-free dependency graph 检查通过。
- D 第十一批后，alpha2.117 同版 CLI 成功生成 123 methods / 10 条已知 warning；全仓 Go test/vet/staticcheck、app/Wails pin 校验、`vue-tsc`、67 个 Vitest 文件/527 tests 与 production build 通过；root 非 Windows `CGO=0` 失败已只剩 Wails GUI 宿主依赖。
- D 第十一批双轴 review：source verifier 改为检查每个受管文件的全部 pin，Task/release 强制核对 PATH 中实际 CLI；删除重复 foreground 实现并增加 root direct-Win32 import 守卫；最终 Standards/Spec 均无剩余 finding。
- D 第十二批本地验证：workflow YAML/matrix 解析与 `git diff --check` 通过；双轴 review 修复干净 runner 缺 `bin/`、Unix artifact 执行位与浮动 macOS 架构问题。Linux/macOS 原生编译仍需提交后由远端 runner 首次执行确认。
- D 第十三批 binding/frontend 验证：alpha2.117 正式生成 107 methods / 0 warnings；被移除的 16 个方法均为 Go 启动期装配入口，前端源码无调用；`vue-tsc`、67 个 Vitest 文件/527 tests 与 production build 通过。
- D 第十三批日志回归：确定性测试证明旧 LogSink 可在 seq=1 callback 阻塞时先交付 seq=2；FIFO delivery pump 修复后顺序/Flush 回归 20 次与 LogSink race 5 次通过，shutdown 使用包内 drain barrier 排空已入队事件。慢 emitter 后的待交付 batch 有界合并并报告 overflow；高精度 timer 在重负载并行门禁中一次超时，隔离重复 20 次通过。
- D 第十三批双轴 review：补齐 binding generator 非零输出、warning 单复数与 107-method 精确门禁；LogSink drain 不再对 emitter 暴露，Close 保持可重入，慢 callback 后的新建/合并 delivery 都受 2000 行及 backing-cap 上限约束；FIFO 测试使用确定性队列状态。最终 Standards/Spec 均无剩余 finding。
- D 第十三批最终门禁：全仓 `go test ./... -count=1`、`go vet ./...`、`staticcheck ./...`、受影响 package race、Wails/app 版本校验与 `git diff --check` 通过。
- E 首批定向验证：`appruntime`、execution、schedule 与 root tests 通过；三包 race 通过。Runtime 并发 Start/Close wait、rollback deadline、HTTP start deadline/request lifetime、并发 Close context、stable Done、Worker late-run cancellation、Schedule cleanup error/pre-start Reload 均有回归测试。
- E 首批双轴 review：修复 Worker queued→active 取消窗口；Runtime 状态等待改为 context-aware broadcast，rollback 使用独立 5s 上限；HTTP request lifetime 与 Start ctx 解耦、Done/Err 广播且并发 Close 各自遵守 deadline；HotkeyRegistry 不再吞 native unregister error或删除仍有 binding 的 entry；最终 Standards/Spec 均无剩余 finding。
- E 首批最终门禁：串行文件 I/O 环境下全仓 `go test -p 1 ./... -count=1`、`go vet ./...`、`staticcheck ./...`、appruntime/hotkey/execution/schedule race、Wails 107 methods / 0 warnings、版本校验与 `git diff --check` 通过。
- E 第二批双轴 review：修复窗口取消 cleanup 的重入死锁并分离 caller/cleanup barrier；线性化 shutdown 与 in-flight open/capture；部分 hotkey Pause 失败后 Shutdown 重试残留 binding；recording 在 native Stop drain 后再次检查 shutdown intent，禁止退出期持久化；capture cancellation 有界且不阻塞窗口/校准清理。
- E 第二批最终门禁：全仓串行 atomic coverage、`go vet ./...`、`staticcheck ./...`、hotkey/recording/calibration/tools race、Wails 107 methods / 0 warnings、版本校验与 `git diff --check` 通过；本批使 hotkey、recording、calibration、tools statement coverage 分别达到 78.9%、36.1%、24.1%、50.2%。

## Open questions

- `flightdeck/knowledge/build/code-style.md` 订阅的 `knowledge/coding/comments.md` 当前不存在；需决定恢复共享注释规范，还是删除断链并把完整规则收回本仓库。
- 外部扩展目标是只接受 in-tree contribution，还是允许 out-of-tree Go module / plugin？
- 首批正式支持的平台矩阵是什么：Windows + Linux，还是 Windows + Linux + macOS？Android/Browser 是 target adapter，不等同于宿主 OS 支持。
- Container package 的崩溃一致性目标采用 generation directory，还是较轻的 lock-last commit + load-time validation？
- 宿主 smoke 应只验证进程启动/资源装载，还是还要覆盖开窗与 WebView 首屏；Linux runner 是否引入虚拟 display？
