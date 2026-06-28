# Target Controller Upgrade — Phase 63 Notes

SUMMARY: 新增 NeedsTarget，把跨平台自动化目标需求从 Win32 HWND 窗口需求里拆出来
READ WHEN: 新增输入/截图/视觉节点、调整 validator 缺目标逻辑、改 runtime setup、改 MCP run_node runnable gate 时
RECHECK WHEN: 引入 controller capability 级 spec、改 target selection 节点、改 Window 输入 override 或 Win32 sendinput 前台策略时

---

## Completed

- `node.Spec` 增加 `NeedsTarget`。
- Catalog/Markdown 输出 `needsTarget` / `NeedsTarget` 标记。
- 输入节点、检测节点、`Capture` 从 `NeedsWindow` 迁到 `NeedsTarget`。
- `NeedsWindow` 保留给直接 Win32 HWND / `WindowService` 操作：
  - `BringWindowForeground`
  - `WindowState`
  - `MoveResizeWindow`
  - `CloseWindow`
  - `PlayClip` / `Script` 暂保守保留 Win32 窗口需求
- Validator 行为：
  - Android/Browser/Win32 任一 target selection 可满足 `NeedsTarget`。
  - 直接 `NeedsWindow` 仍要求 `Win32WindowTarget`，除非该节点的 `Window` 输入已连线。
  - 没有任何 target selection 的 `NeedsTarget` 图仍报 `MISSING_WIN32_WINDOW_TARGET`，保留 Windows 默认自动修复入口。
- Runtime setup 行为：
  - direct `NeedsWindow` 图会初始化 Win32 input/capture backend。
  - `NeedsTarget + Win32WindowTarget` 或 `NeedsTarget + 无显式 target` 会初始化 Win32 backend。
  - `NeedsTarget + AndroidTarget/BrowserTarget only` 不初始化 Win32 backend，运行时走 controller factory。
- MCP `run_node` gate 从 `NeedsWindow` 改为 `NeedsTarget || NeedsWindow`。

## Boundary

`NeedsTarget` 现在只是布尔能力闸，不含 capability 列表。具体点击、键盘、截图、相对移动等能力仍由 runtime controller capability check 在执行时 fail closed。

下一步如果要继续收敛，应把 `NeedsTarget` 细化成 capability set，例如：

- `pointer-input`
- `keyboard-input`
- `screenshot`
- `mouse-button`
- `move-relative`

这比继续复用 `NeedsWindow` 更可维护，也能让 Android/Browser 的错误在编辑期更早暴露。

## Verification

- `go test ./internal/services/container -run "TestValidate_(AndroidTargetWithInput_NoMissingWin32WindowTarget|UnwiredNeedsWindow_ReportsMissingWin32WindowTarget)" -count=1`
- `go test ./internal/services/container/runtime -run "TestSetupRuntime_(AndroidTargetDoesNotBuildWin32Backends|BuildsBackendsWithoutResolvingWindow)" -count=1`
- `go test ./internal/nodes/all ./internal/services/container ./internal/services/container/runtime ./internal/catalog`
- `go test ./...`
- `cd frontend && pnpm gen:node-i18n && pnpm i18n:check`
- `git diff --check`
