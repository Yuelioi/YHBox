# scripts

这里存放仓库级自动化的底层实现。日常开发优先从仓库根目录运行 `task` 中的正式入口，不要绕过
Taskfile 复制一套并行命令；`task check` 是增量本地门禁，`task check:full` 是 CI/发布完整门禁。

脚本默认从仓库根目录调用。带写入、进程控制或真机操作的脚本会在下表明确标注，调用前先确认对应前置条件。

## 目录清单

| 文件 | 正式入口 | 用途与副作用 |
| --- | --- | --- |
| `assert-clean-worktree.ps1` | `task release:verify-clean` | 只读；拒绝包含 staged、unstaged 或 untracked 文件的发布工作区。 |
| `check-changed.mjs` / `check-go-changed.mjs` | `task check` | 只读；按 Git 变更选择相关门禁，Go 包包含反向依赖。 |
| `check-docs.mjs` | `task check:docs` / `task check` | 只读；验证稳定 Markdown 的本地链接、Task 名和已淘汰公开名称。 |
| `check-actions-pinned.ps1` | `task check:supply-chain:actions` / `task check:full` | 只读；检查第三方 GitHub Actions 是否固定到完整 commit SHA。 |
| `verify-toolchains.ps1` | `task check:supply-chain:toolchains` / `task check:full` | 只读；校验工具链清单、源码/CI pin、固定 runner 与本机工具版本。 |
| `verify-third-party-artifacts.ps1` | `task check:supply-chain:artifacts` / `task check:full` | 只读；校验随仓库分发的第三方二进制、来源元数据与 SHA-256。 |
| `generate-workflow-contracts.mjs` | `task contracts:check` / `task contracts:update` | 默认只比较生成结果；`contracts:update` 会重写 tracked Workflow/Node contract。 |
| `check-go-coverage.ps1` | `task check:go:full` | 只读；合并 Go coverage profile 并按 `go-coverage-budgets.json` 执行门槛。 |
| `go-coverage-budgets.json` | `task check:go:full` | `check-go-coverage.ps1` 的版本化预算数据，不是可执行脚本。 |
| `test-script-worker.ps1` | `task check:go:full` | 在 `.task/` 构建临时 ScriptWorker，并运行隔离 worker smoke。 |
| `verify-version.ps1` | 发布流程直接调用 | 只读校验根 `VERSION` 与发布标签期望版本，并复用 Wails/Windows 产品版本投影门禁。 |
| `verify-windows-binary-version.ps1` | `task versions:check:binary` | 只读校验已构建 EXE 的固定/字符串版本资源和 `WINDOWS_GUI` subsystem。 |
| `smoke-windows-desktop-startup.ps1` | `task smoke:desktop` | 把已构建 GUI 与 worker 复制到隔离目录，验证启动后不会立即退出。 |
| `smoke-storage-migration.ps1` | `task smoke:storage-migration` | 用冻结 layout 1 profile 触发 recovery，强停 GUI 后隔离阻塞记录、续接到当前 layout，并验证 production health/重启。 |
| `verify-wails-version.ps1` | `task wails:verify` | 只读；校验 Wails Go/CLI/runtime pin 与已安装 CLI。 |
| `bump-version.ps1` | `task version:bump BUMP=<alpha|patch|minor|major|x.y.z[-alpha.N]>` | 修改根 `VERSION` 并刷新投影；alpha.N 映射到 Windows 第四段 N；不 commit、不 tag。`-DryRun` 只报告。 |
| `stage-release.ps1` | `task release:stage` | **会重建** `artifacts/staging/Yotta` 并生成确定性 ZIP 与 artifact manifest。 |
| `smoke-release-candidate.ps1` | `task release:smoke` | 将冻结 payload 复制到 `.task/`，运行 worker/plugin/CLI/桌面启动 smoke，并控制测试进程。 |
| `write-release-checksums.ps1` | `.github/workflows/release.yml` | Release CI 专用；为指定发布产物写 `SHA256SUMS`。 |
| `smoke-workflow-editor.ps1` | `task webview:smoke` / `task webview:smoke:full` | 构建隔离 DEV host、启动 Vite/WebView2 CDP、执行 UI 旅程并在 `.task/` 保存截图和日志。 |
| `smoke-windows-automation.ps1` | `task windows:smoke:automation` | **占用真实 Windows 桌面输入**；验证窗口、截图、录制与 SendInput 路径，脚本以 `-p 1` 串行执行会争用全局桌面的 Go package。 |
| `template-match-diagnostic.ps1` | 手动诊断 | 按 Workflow Resource 和 configured desktop Target 抓取一帧，同时比较原始模板与目标分辨率派生模板；证据写入本地 `diagnostics/captures/`。 |
| `android-adb-smoke.ps1` | 手工真机 smoke | **操作已授权 Android 设备/模拟器**；验证 ADB provider 的完整 Workflow 纵向路径。参数和触发条件见 build knowledge。 |
| `browser-cdp-smoke.ps1` | 手工浏览器 smoke | 启动带隔离 profile 的 Chrome/Edge，验证 Browser CDP provider，并精确清理对应进程/profile。 |

真机、浏览器、发布签名和 WebView smoke 的环境要求及运行时机，以 [`flightdeck/knowledge/build/build.md`](../flightdeck/knowledge/build/build.md) 为准。

## 维护约定

- 能由 Taskfile 表达的入口只在 Taskfile 暴露；脚本负责实现，不在文档中维护另一套完整流水线。
- 需要支持直接运行的脚本必须从 `$PSScriptRoot` 或模块 URL 推导仓库路径；只由 Taskfile 调用的实现可以依赖 Task 的仓库根工作目录，并在表中保留唯一正式入口。
- 调用 `go`、`node`、`task`、CLI 或测试进程后必须检查退出码；`finally` 中的清理不能覆盖真实失败。
- 临时产物写入 `.task/`，发布候选写入 `artifacts/`；不要把用户数据、凭据或私有样本写进仓库。
- 需要删除脚本时，同时移除 Task/CI/knowledge 引用。只有正式入口已经替换、调用目标已经消失，且引用搜索与阶段门禁都证明无消费者时，才算可删除；“不常运行”不等于废弃。
