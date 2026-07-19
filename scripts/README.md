# scripts

这里存放仓库级自动化的底层实现。日常开发优先从仓库根目录运行 `task` 中的正式入口，不要绕过 Taskfile 复制一套并行命令；完整本地门禁始终只有 `task check`。

脚本默认从仓库根目录调用。带写入、进程控制或真机操作的脚本会在下表明确标注，调用前先确认对应前置条件。

## 目录清单

| 文件 | 正式入口 | 用途与副作用 |
| --- | --- | --- |
| `assert-clean-worktree.ps1` | `task release:verify-clean` | 只读；拒绝包含 staged、unstaged 或 untracked 文件的发布工作区。 |
| `check-actions-pinned.ps1` | `task check:supply-chain` | 只读；检查第三方 GitHub Actions 是否固定到完整 commit SHA。 |
| `verify-toolchains.ps1` | `task check:supply-chain` | 只读；校验工具链清单、源码/CI pin、固定 runner 与本机工具版本。 |
| `verify-third-party-artifacts.ps1` | `task check:supply-chain` | 只读；校验随仓库分发的第三方二进制、来源元数据与 SHA-256。 |
| `generate-workflow-contracts.mjs` | `task contracts:check` / `task contracts:update` | 默认只比较生成结果；`contracts:update` 会重写 tracked Workflow/Node contract。 |
| `check-go-coverage.ps1` | `task check:go` | 只读；合并 Go coverage profile 并按 `go-coverage-budgets.json` 执行门槛。 |
| `go-coverage-budgets.json` | `task check:go` | `check-go-coverage.ps1` 的版本化预算数据，不是可执行脚本。 |
| `test-script-worker.ps1` | `task check:go` | 在 `.task/` 构建临时 ScriptWorker，并运行隔离 worker smoke。 |
| `verify-version.ps1` | `task version:verify` | 只读；校验 Go、前端、Wails/Windows 资源中的产品版本一致。 |
| `verify-wails-version.ps1` | `task wails:verify` | 只读；校验 Wails Go/CLI/runtime pin 与已安装 CLI。 |
| `bump-version.ps1` | `task version:bump VERSION=<x.y.z>` | **会修改并提交**版本文件，然后创建 Git tag；要求干净工作区。`-DryRun` 只报告。 |
| `stage-release.ps1` | `task release:stage` | **会重建** `artifacts/staging/Yotta` 并生成确定性 ZIP 与 artifact manifest。 |
| `smoke-release-candidate.ps1` | `task release:smoke` | 将冻结 payload 复制到 `.task/`，运行 worker/plugin/CLI/桌面启动 smoke，并控制测试进程。 |
| `write-release-checksums.ps1` | `.github/workflows/release.yml` | Release CI 专用；为指定发布产物写 `SHA256SUMS`。 |
| `smoke-workflow-editor.ps1` | `task webview:smoke` / `task webview:smoke:full` | 构建隔离 DEV host、启动 Vite/WebView2 CDP、执行 UI 旅程并在 `.task/` 保存截图和日志。 |
| `smoke-windows-automation.ps1` | `task windows:smoke:automation` | **占用真实 Windows 桌面输入**；验证窗口、截图、录制与 SendInput 路径，应在阶段末串行运行。 |
| `android-adb-smoke.ps1` | 手工真机 smoke | **操作已授权 Android 设备/模拟器**；验证 ADB provider 的完整 Workflow 纵向路径。参数和触发条件见 build knowledge。 |
| `browser-cdp-smoke.ps1` | 手工浏览器 smoke | 启动带隔离 profile 的 Chrome/Edge，验证 Browser CDP provider，并精确清理对应进程/profile。 |

真机、浏览器、发布签名和 WebView smoke 的环境要求及运行时机，以 [`flightdeck/knowledge/build/build.md`](../flightdeck/knowledge/build/build.md) 为准。

## 维护约定

- 能由 Taskfile 表达的入口只在 Taskfile 暴露；脚本负责实现，不在文档中维护另一套完整流水线。
- 需要支持直接运行的脚本必须从 `$PSScriptRoot` 或模块 URL 推导仓库路径；只由 Taskfile 调用的实现可以依赖 Task 的仓库根工作目录，并在表中保留唯一正式入口。
- 调用 `go`、`node`、`task`、CLI 或测试进程后必须检查退出码；`finally` 中的清理不能覆盖真实失败。
- 临时产物写入 `.task/`，发布候选写入 `artifacts/`；不要把用户数据、凭据或私有样本写进仓库。
- 需要删除脚本时，同时移除 Task/CI/knowledge 引用。只有正式入口已经替换、调用目标已经消失，且引用搜索与阶段门禁都证明无消费者时，才算可删除；“不常运行”不等于废弃。

## 2026-07-19 审计结果

本次审计覆盖新增 README 前的全部 18 个工具文件：16 个 PowerShell 文件均可被 PowerShell parser 解析，MJS 通过 `node --check`，JSON 可解析；所有脚本引用的 Go package 与 test function 仍存在。调用方分布为 14 个 Taskfile 入口、2 个有明确触发条件的手工 smoke、1 个 Release CI 入口和 1 个配套预算文件。

未发现可以安全删除的失效脚本，因此本次没有误删仍承担真机、发布或门禁职责的文件。后续若入口发生替换，按上面的删除判据整链清理。
