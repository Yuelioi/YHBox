---
kind: checklist
summary: "编译 / 验证产物的前置约定 — task check/dev/build / pnpm / bindings contract / bundle budget / 测试套件 / smoke 留意"
activation: action
read_when: "before compiling / building / verifying production artifact / 跑 runtime 测试套件 / 跑前端 vitest / 真机 smoke"
recheck_when: "改构建命令 (task dev/build) / wails 配置 / vite 配置 / bindings 生成 / 测试套件入口 / 前端测试跑法时"
---
# Build checklist

编译 / 验证产物时前置读这份。

- frontend 包管理只用 pnpm（Node 24.18.0 / pnpm 11.1.2，engine-strict）；安装与 CI 一律 frozen lockfile。
- Wails Go/CLI 固定 v3.0.0-alpha2.117，frontend runtime 固定 v3.0.0-alpha.97；scripts/verify-wails-version.ps1 -CheckInstalled 验证实际 CLI。
- 开发入口 task dev。
- Workflow WebView smoke：powershell -NoProfile -ExecutionPolicy Bypass -File ./scripts/smoke-workflow-editor.ps1。使用正式 Windows DEV build、独立 exe/data/profile 与 loopback CDP；断言目录点击与拖放节点数递增，拒绝 JS error/rejection/console.error，并必须实际查看 PNG。空白连线落点必须扫描画布可用区域，不能依赖少量固定坐标。PowerShell wrapper 调用 go run/其它探针后必须立即检查 `$LASTEXITCODE` 并转成失败；不要让 finally/清理命令覆盖子进程退出码造成假绿。
- 完整本地门禁 task check：supply-chain、contracts、AI eval、版本/Wails、Go tests + global 65% + vet/staticcheck、bindings、format/lint/typecheck/i18n/Vitest/production bundle。
- 正式构建只用 task build；它生成 bindings/frontend/syso，并构建 Yotta.exe、Yotta.CLI.exe、ScriptWorker、WasmPluginRunner、capture DLL 与 ADB。不要裸 go build -o Yotta.exe。
- task package 要求前后 worktree 全干净，依次执行 task check、production build、staging、manifest/archive 与 frozen-payload smoke；公开 stable 仍受许可证、证书、canonical identity、维护者/owner 设置和原生宿主 smoke 阻塞。
- task release:sign-and-stage 只签已经冻结的 payload，签名后 restage 并重复 candidate smoke；不得以 sign task 隐式 rebuild。

## bindings 与 generated contracts

frontend/bindings 由 Wails 生成且 gitignore，不手改。node frontend/scripts/generate-bindings.mjs 固定生成 TypeScript；pnpm -C frontend bindings:check 对比 tracked contracts/wails-rpc.json。service/method/model 数量只作诊断信息；tracked RPC schema 与签名 diff 才是契约，不能把某次计数硬编码成长期门禁。

Workflow/Node durable contracts 由同一 Go generator 供 runtime validator 与 tracked JSON Schema/TypeScript；task contracts:check 拒绝漂移。plugin Proto/WIT/SDK/reference/conformance 由 task plugins:check 拒绝漂移。

## 测试基线

覆盖率、前端测试数、RPC 数量和 bundle 字节数会随实现变化，不在 Knowledge 固定某次运行快照。权威门槛位于 Task/CI/config；阶段结果写入对应 Topic/Slice。覆盖统计合并 plugin shared conformance profile、按 source block 去重，并排除带标准 Code generated ... DO NOT EDIT 标记的生成文件；不降低阈值，也不把 protoc getter 当人工代码。

Go 门禁由 task check 统一编排。CI 另含 race group、parser/package/MCP fuzz、Linux/macOS portable core 与三平台原生 GUI compile。race 清单使用稳定 internal/noderuntime 名称，不得恢复 nodes31 等发布号包名。

Windows 本地可用 go test -c 逐包生成 linux/amd64、darwin/arm64 测试二进制，只作为 portable-core 编译证据；不能直接 GOOS=... go test 后尝试运行外平台二进制，也不能把 wrapper 跳过执行冒充原生测试。原生 Linux/macOS portable-core 与 production GUI 结果以 CI runner 为权威；Windows cross-compile 不能替代 CGo/WebKit 宿主。

Linux/macOS 没有等价 sandbox 时 Process/Wasm capability 必须 fail closed。

前端测试、i18n、no-explicit-any debt 与 bundle budgets 由 `task check` 中的具体 gate 管理。raw chunk 超过 500 kB 的 Vite 通用 warning 本身非阻断，以 repository bundle budget gate 为准。

## 运行 / smoke

- Windows Process/Wasm plugin smoke：task windows:smoke:plugins，必须走真实 LPAC/AppContainer + Job isolation。
- Frozen candidate smoke：task release:smoke；校验 manifest exact file set/size/SHA-256，并从 staging copy 运行 ScriptWorker、Process/Wasm plugin、CLI strict legacy rejection 与 desktop startup。smoke 不得修改 staging。
- Workflow WebView smoke 只能证明页面/创作入口；catalog node 数和 canvas node 数是观测值，不是产品能力。录制、模板、Windows/ADB 输入等宿主能力还必须通过各 Stage 的真实纵向旅程。
- Android ADB 真机/模拟器 smoke 只在已授权设备可用时运行：`$env:YOTTA_ADB_SMOKE='1'; go test ./internal/automation/installed -run TestAndroidADBEmulatorSmoke -count=1 -v`。它必须通过 bundled/configured ADB 做 exact identity、分辨率、启动/停止、PNG 截图和通用输入操作；不得以 controller mock 替代该证据。
- WebView2 截图前必须 bringToFront、focus emulation、两次 requestAnimationFrame 加 settle，避免 DOM 绿但 PNG 黑屏。
- Wails dev 的可选 custom.js/favicon 404 非阻断；阻断信号是 JS error/rejection/console.error、节点计数不变、CDP 不可达或截图实际布局不可用。
