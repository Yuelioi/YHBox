# Build checklist

编译 / 验证产物时前置读这份。

脚本职责、正式 Task/CI 入口与副作用索引见 `scripts/README.md`。日常门禁由增量 `task check` 定义；
CI/发布完整门禁由 `task check:full` 定义。

- frontend 包管理只用 pnpm（Node 24.18.0 / pnpm 11.1.2，engine-strict）；安装与 CI 一律 frozen lockfile。
- Wails Go/CLI 固定 v3.0.0-alpha2.117，frontend runtime 固定 v3.0.0-alpha.97；scripts/verify-wails-version.ps1 -CheckInstalled 验证实际 CLI。
- 开发入口 task dev。
- Workflow WebView smoke：日常运行 `task webview:smoke`，需要悬浮启动器纵向旅程时运行 `task webview:smoke:full`；底层入口是 `scripts/smoke-workflow-editor.ps1`。使用正式 Windows DEV build、独立 exe/data/profile 与 loopback CDP；CDP 可能只监听 IPv4 `127.0.0.1` 或 IPv6 `::1`，wrapper 必须从实际 TCP listener 选择 endpoint，不能写死一种 loopback。断言损坏 Source 恢复面、显式“添加节点”与 Tab 快速添加、左侧五个工作区工具、停靠式子图管理和三类独立资源入口，拒绝 JS error/rejection/console.error，并必须实际查看工作流列表、编辑器、资源工具、资源库与计划编辑器 PNG；full 额外检查 launcher workflow 执行/隐藏复用。production manifest 需要 UAC，因此隐藏 CDP 旅程使用同编译输入的无 manifest smoke host；它不能替代 `task build` 和 manifest 检查。空白连线落点必须扫描画布可用区域，不能依赖少量固定坐标。PowerShell wrapper 调用 go run/其它探针后必须立即检查 `$LASTEXITCODE` 并转成失败；不要让 finally/清理命令覆盖子进程退出码造成假绿。
- WebView 多选必须发送真实 modifier `keyDown → mouse events → keyUp`，不能只在鼠标事件上填写
  modifier 位；框选能力验证与后续 destructive action 的目标选择应分开，避免布局变化让包围矩形误选根节点。
- 交互式 WebView 调试：`task dev` 在 loopback `9227` 开放 CDP（可用 `WEBVIEW_DEBUG_PORT=<port>` 覆盖），开发窗口按 `Ctrl+Shift+I` 打开 Wails/WebView2 DevTools。对运行中的开发 WebView 执行 `task webview:screenshot` 可生成 `.task/webview/current.png`；多窗口时用 `URL_CONTAINS=<substring>` 精确选择。调试入口只在非 production build 存在；production 不启用快捷键，也不接受仓库自定义 CDP 环境变量。
- 日常 `task check` 读取相对 HEAD 的 staged、unstaged、untracked 文件；设置 `CHECK_BASE=<ref>` 时还纳入
  `<ref>...HEAD`。它先打印计划，再只运行相关门禁：Go 包及反向依赖 test/vet、前端快速检查，以及按路径
  触发的 contracts、bindings、依赖、版本、Wails、插件、AI 或 Rust 检查。
- `task check:full` 才运行 supply-chain、contracts、AI eval、版本/Wails、Go 全仓 tests + global 65% +
  vet/staticcheck、bindings、format/lint/typecheck/i18n/Vitest/production bundle。CI、`task package`、发布候选或
  用户明确要求完整验收时使用；普通代码修改不得无条件使用。
- 正式构建只用 task build；它生成 bindings/frontend/syso，并构建 Yotta.exe、Yotta.CLI.exe、ScriptWorker、WasmPluginRunner、capture DLL 与 ADB。不要裸 go build -o Yotta.exe。
- task package 要求前后 worktree 全干净，依次执行 task check:full、production build、staging、manifest/archive 与 frozen-payload smoke；公开 stable 仍受许可证、证书、canonical identity、维护者/owner 设置和原生宿主 smoke 阻塞。
- task release:sign-and-stage 只签已经冻结的 payload，签名后 restage 并重复 candidate smoke；不得以 sign task 隐式 rebuild。

## bindings 与 generated contracts

frontend/bindings 由 Wails 生成且 gitignore，不手改。node frontend/scripts/generate-bindings.mjs 固定生成 TypeScript；pnpm -C frontend bindings:check 对比 tracked contracts/wails-rpc.json。service/method/model 数量只作诊断信息；tracked RPC schema 与签名 diff 才是契约，不能把某次计数硬编码成长期门禁。

Workflow/Node durable contracts 由同一 Go generator 供 runtime validator 与 tracked JSON Schema/TypeScript；task contracts:check 拒绝漂移。plugin Proto/WIT/SDK/reference/conformance 由 task plugins:check 拒绝漂移。

## 测试基线

覆盖率、前端测试数、RPC 数量和 bundle 字节数会随实现变化，不在 Knowledge 固定某次运行快照。权威门槛位于 Task/CI/config；阶段结果写入对应 Topic/Slice。覆盖统计合并 plugin shared conformance profile、按 source block 去重，并排除带标准 Code generated ... DO NOT EDIT 标记的生成文件；不降低阈值，也不把 protoc getter 当人工代码。

Go 日常门禁由 task check 选择受影响包及其反向依赖；全仓 coverage/staticcheck 由 task check:full 编排。
CI 另含 race group、parser/package/MCP fuzz、Linux/macOS portable core 与三平台原生 GUI compile。race 清单使用稳定 internal/noderuntime 名称，不得恢复 nodes31 等发布号包名。

Windows 本地可用 go test -c 逐包生成 linux/amd64、darwin/arm64 测试二进制，只作为 portable-core 编译证据；不能直接 GOOS=... go test 后尝试运行外平台二进制，也不能把 wrapper 跳过执行冒充原生测试。原生 Linux/macOS portable-core 与 production GUI 结果以 CI runner 为权威；Windows cross-compile 不能替代 CGo/WebKit 宿主。

Linux/macOS 没有等价 sandbox 时 Process/Wasm capability 必须 fail closed。

前端测试、i18n 和 no-explicit-any debt 进入增量前端门禁；production build 与 bundle budgets 由
`task check:full` 管理。raw chunk 超过 500 kB 的 Vite 通用 warning 本身非阻断，以 repository bundle budget gate 为准。

## 运行 / smoke

- Windows automation native smoke：`task windows:smoke:automation`。修改 `pkg/input`、`pkg/winutil`、Windows adapter、窗口捕获或 recorder native path 后，在阶段末批量运行。该 smoke 使用全局 SendInput、foreground 与 native hook，必须串行且占用独立桌面；不要与其它 UI smoke 并行或中途强杀。当前前台若是更高完整性进程，应以与 production 相同的管理员完整性运行 smoke，不得靠反复重试刷绿；中断后先清理精确测试进程并确认输入状态。
- Windows Process/Wasm plugin smoke：task windows:smoke:plugins，必须走真实 LPAC/AppContainer + Job isolation。
- Frozen candidate smoke：task release:smoke；校验 manifest exact file set/size/SHA-256，并从 staging copy 运行 ScriptWorker、Process/Wasm plugin、CLI strict legacy rejection 与 desktop startup。smoke 不得修改 staging。
- Workflow WebView smoke 只能证明页面/创作入口；工作区工具数和 canvas node 数是观测值，不是产品能力。录制、模板、Windows/ADB 输入等宿主能力还必须通过各 Stage 的真实纵向旅程。
- Android ADB 真机/模拟器 smoke 只在已授权设备可用时运行：`powershell -NoProfile -ExecutionPolicy Bypass -File ./scripts/android-adb-smoke.ps1 -Serial <exact-serial> -Package <exact-package>`。它必须走 Source → Compiler → Admission → installed provider → journal，覆盖 exact identity、应用发现、activate、PNG capture、template click、drag、InputClip playback 与 stop-app；controller mock 或 controller-only smoke 不能替代。
- Browser CDP smoke：Chrome 使用 `powershell -NoProfile -ExecutionPolicy Bypass -File ./scripts/browser-cdp-smoke.ps1`；Edge 传 `-BrowserPath 'C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe' -Port <free-port>`。脚本必须创建独立 profile、恢复调用前环境变量并精确清理 profile/process；测试走 Source → Compiler → Admission → provider → journal，并核对页面真实副作用。
- WebView2 截图前必须 bringToFront、focus emulation、两次 requestAnimationFrame 加 settle，避免 DOM 绿但 PNG 黑屏。
- Wails dev 的可选 custom.js/favicon 404 非阻断；阻断信号是 JS error/rejection/console.error、节点计数不变、CDP 不可达或截图实际布局不可用。
