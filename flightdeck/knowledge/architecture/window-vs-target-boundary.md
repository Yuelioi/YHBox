# Window vs Target Boundary

SUMMARY: 明确 Window 只表示 Win32 HWND 窗口，Target 表示跨 Windows/Android/Browser 的自动化对象
READ WHEN: 改 WindowTarget / AndroidTarget / BrowserTarget / NeedsWindow / TargetService / 窗口控制节点 / 输入和截图路由时
RECHECK WHEN: 新增目标类型、重命名节点 kind、改 validator 缺目标提示、改截图取点或前台策略时

---

## Current decision

`Window` 不再作为所有自动化对象的总称。

- `WindowService`、`WindowHandle`、`WindowInputSpec`、`NeedsWindow` 表示 Win32 HWND 窗口能力。
- `TargetService`、`target.Target`、controller factory 表示自动化目标能力。
- `AndroidTarget` 和 `BrowserTarget` 是 target selection node，不是 window node。
- `WindowTarget` 暂时保留为兼容 kind，但用户可见名称应写成 Windows 窗口目标。

## Why keep `WindowTarget` for now

旧容器 JSON、validator、MCP authoring examples、录制服务、编辑期工具和很多测试都已经依赖 kind 字符串 `WindowTarget`。直接改名会让历史容器和外部调用同时断裂。

当前策略：

1. 先修 UI/文案：把通用“目标窗口”拆成“自动化目标”或“Windows 窗口”。
2. 保留 serialized kind：`WindowTarget` 继续可读可跑。
3. 后续加 alias：新增 display-facing `Win32WindowTarget` / `WindowsWindowTarget` 时，loader 把旧 `WindowTarget` 迁到同一内部语义。
4. 再迁移 contract：`NeedsWindow` 逐步变成 `NeedsTarget(kind=win32-window, capabilities=...)`，但兼容旧字段。

## Win32-only nodes

这些节点只应作用于 Windows HWND：

- `WindowTarget`
- `GetWindow`
- `WaitWindow`
- `WaitWindowGone`
- `BringWindowForeground`
- `WindowState`
- `MoveResizeWindow`
- `CloseWindow`

Browser tab 的“置前”以后应是 browser/page activation。Android 的“置前”以后应是 app lifecycle / wake device。它们不应复用 `BringWindowForeground`。

## Forbidden drift

- 不要把 Android/Browser target 加进 `WindowService`。
- 不要让“目标窗口”继续作为通用 UI 词。
- 不要在新节点里用 `NeedsWindow` 表示“需要任意自动化目标”。
- 不要让 Browser target 伪装成 Chrome 主窗口优先点击；CDP/selector 是主路径。
- 不要让 Android target 伪装成模拟器窗口优先点击；ADB/minitouch/MAA-touch 风格 controller 是主路径。

## Naming rule

- Generic automation object: `目标` / `自动化目标` / `Target`
- Win32 HWND object: `Windows 窗口` / `Win32 window`
- Legacy node display: `Windows 窗口目标` / `Windows window target`
- Serialized compatibility kind: keep `WindowTarget` until alias migration is complete.
