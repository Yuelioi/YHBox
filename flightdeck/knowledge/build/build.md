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
- Wails Go/CLI 固定 v3.0.0-alpha2.117，frontend runtime 固定 3.0.0-alpha.97；scripts/verify-wails-version.ps1 -CheckInstalled 验证实际 CLI。
- 开发入口 task dev。
- Workflow WebView smoke：powershell -NoProfile -ExecutionPolicy Bypass -File ./scripts/smoke-workflow-editor.ps1。使用正式 Windows DEV build、独立 exe/data/profile 与 loopback CDP；断言目录点击 0→1、拖放 1→2，拒绝 JS error/rejection/console.error，并必须实际查看 PNG。
- 完整本地门禁 task check：supply-chain、contracts、AI eval、版本/Wails、Go tests + global 65% + vet/staticcheck、bindings、format/lint/typecheck/i18n/Vitest/production bundle。
- 正式构建只用 task build；它生成 bindings/frontend/syso，并构建 Yotta.exe、ScriptWorker、WasmPluginRunner、capture DLL 与 ADB。不要裸 go build -o Yotta.exe。
- task package 要求前后 worktree 全干净，从已测试 staging allowlist 生成 manifest/archive；公开 stable 仍受许可证、证书、迁移与 owner 设置阻塞。

## bindings 与 generated contracts

frontend/bindings 由 Wails 生成且 gitignore，不手改。node frontend/scripts/generate-bindings.mjs 固定生成 TypeScript；pnpm -C frontend bindings:check 对比 tracked contracts/wails-rpc.json。2026-07-17 基线为 14 services / 95 methods / 109 models。

Workflow/Node durable contracts 由同一 Go generator 供 runtime validator 与 tracked JSON Schema/TypeScript；task contracts:check 拒绝漂移。plugin Proto/WIT/SDK/reference/conformance 由 task plugins:check 拒绝漂移。

## 测试基线

2026-07-17 插件阶段后的可信基线：task check 全绿；global Go coverage 65.0%（门槛 65%）、根包 34.1%，package floors、go vet、staticcheck 全部通过。覆盖统计合并 plugin shared conformance profile、按 source block 去重，并排除带标准 Code generated ... DO NOT EDIT 标记的生成文件；不降低阈值，也不把 protoc getter 当人工代码。

Go 门禁由 task check 统一编排。CI 另含 race group、parser/package/MCP fuzz、Linux/macOS portable core 与三平台原生 GUI compile。race 清单使用稳定 internal/noderuntime 名称，不得恢复 nodes31 等发布号包名。

Linux/macOS portable core 已纳入 node/package/plugin contract；没有等价 sandbox 时 Process/Wasm capability 必须 fail closed。原生 Linux/macOS production GUI 结果以 CI gui-build matrix 为准，Windows cross-compile 不能替代原生 CGo/WebKit 宿主。

前端基线：28 files / 106 tests；i18n 1269 keys、0 中文 residue；Wails 14/95/109；tracked no-explicit-any debt 24。production bundle entry 262852 gzip bytes（limit 350000），editor 96843（limit 200000，target 125000）。raw chunk 超过 500 kB 的 Vite 通用 warning 非阻断，以 bundle:check 为准。

## 运行 / smoke

- Windows Process/Wasm plugin smoke：task windows:smoke:plugins，必须走真实 LPAC/AppContainer + Job isolation。
- Workflow smoke 基线：100 catalog nodes、2 canvas nodes、AI review panel 可达；截图应显示实际顶栏、目录、画布、节点、review panel 与日志层。
- WebView2 截图前必须 bringToFront、focus emulation、两次 requestAnimationFrame 加 settle，避免 DOM 绿但 PNG 黑屏。
- Wails dev 的可选 custom.js/favicon 404 非阻断；阻断信号是 JS error/rejection/console.error、节点计数不变、CDP 不可达或截图实际布局不可用。
