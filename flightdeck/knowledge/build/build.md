# Build and verification

Taskfile、CI 和 schema 是门禁权威；本指南只说明如何选择入口。底层脚本职责与副作用见
[`scripts/README.md`](../../../scripts/README.md)。

## Default gate

普通修改从仓库根运行：

```powershell
task check
```

它读取相对 HEAD 的 staged、unstaged 和 untracked 文件；设置 `CHECK_BASE=<ref>` 时也纳入
`<ref>...HEAD`。它先打印计划，再按路径选择受影响 Go package 及反向依赖、前端快速门禁，以及 contract、
bindings、toolchain、plugin、AI/Rust 等专门检查。

该命令通常超过 60 秒。首次执行就使用可续接、可轮询的进程；外层调用超时不代表失败。重试前确认原
进程已结束并取得真实退出码，不能并行重复启动同一门禁。

`task check:full` 运行全仓 tests/coverage/staticcheck、frontend full gate/production bundle、供应链、
contracts、bindings、版本/Wails、AI 和 Rust。只在 CI、发布/打包、明确要求完整验收或变更会影响全局
门槛时使用，不作为普通修改默认收尾。

## Additional evidence by change

| Change | Additional verification |
| --- | --- |
| Go concurrency、lifecycle、shared state、durable store | 对受影响 package 运行 `go test -race`；再由 `task check` 收尾 |
| Vue 页面/组件/响应式交互 | Vitest/相关 test + CLI Playwright 对本地页面交互和截图；不要依赖内置浏览器 |
| Workflow 编辑器或 Wails WebView 集成 | `task webview:smoke`；涉及 floating launcher 时用 `task webview:smoke:full` |
| Win32 input/capture/window/recording/native hook | `task windows:smoke:automation`，独占真实桌面并串行运行 |
| Process/Wasm host、package runtime | `task windows:smoke:plugins`，验证真实 LPAC/AppContainer + Job isolation |
| root layout、Catalog/Run migration、backup/journal/recovery | `task smoke:storage-migration` |
| Android ADB adapter | 在已授权 exact serial/package 上运行 `scripts/android-adb-smoke.ps1` |
| Browser CDP adapter | 运行 `scripts/browser-cdp-smoke.ps1`，使用独立 profile 和空闲端口 |
| release candidate | `task release:smoke`；只验证 frozen staging，不修改 payload |

native automation smoke 会使用全局输入、前台窗口和 hook。不要与其它 UI smoke 并行，不要中途强杀；
如果当前目标完整性级别更高，应以与 production 一致的管理员权限运行，而不是重试刷绿。

## Generated contracts and bindings

- `frontend/bindings/` 由 Wails 生成且 gitignore；不能手改。`task dev`、`task build` 和正式 bindings 入口
  负责生成，`task check:bindings` 验证 tracked RPC contract。
- Workflow/Node/Data/Authoring schema 与内建 Catalog/Projection 由正式 generator 产生；修改所属合同后运行
  `task contracts:check`，需要接受新产物时使用仓库定义的 update 入口。
- 当前节点视图运行 `task nodes`、`task nodes:catalog`、`task nodes:authoring`；不要在文档或测试里硬编码
  某次节点数、RPC 数、测试数或 bundle bytes。

## Build, package and version

- 开发入口：`task dev`。
- 正式 Windows 构建：`task build`。它负责 bindings、frontend、resource/syso、GUI/CLI、worker、runner、
  capture DLL 与 ADB；不要用裸 `go build -o Yotta.exe` 代替。
- `task package` 要求 clean worktree，并运行 full gate、production build、staging、manifest/archive 和 frozen
  candidate smoke。未经用户授权不要为了满足它清理或提交工作区。
- 产品版本的唯一可编辑来源是根 `VERSION`。查看当前域用 `task version:show` / `task versions:inventory`；
  提升用 `task version:bump BUMP=<patch|minor|major|x.y.z>`；手工改 `VERSION` 后运行
  `task version:sync` 和 `task versions:check`。版本工具不 commit、不 tag。
- 签名只作用于 frozen payload；签名后 restage 并重复 candidate smoke，不能让 sign 步骤隐式 rebuild。

CI 还负责 Windows race、parser/package/MCP fuzz、Linux/macOS portable core 与三平台 GUI compile。Windows
cross-compile 只能作为编译证据，不能冒充 Linux/macOS 原生测试或 GUI 宿主验证。
