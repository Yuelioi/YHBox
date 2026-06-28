# Window vs Target Boundary

SUMMARY: 明确 Window 只表示 Win32 HWND 窗口，Target 表示跨 Windows/Android/Browser 的自动化对象
READ WHEN: 改 Win32WindowTarget / AndroidTarget / BrowserTarget / NeedsTarget / NeedsWindow / TargetService / 窗口控制节点 / 输入和截图路由时
RECHECK WHEN: 新增目标类型、重命名节点 kind、改 validator 缺目标提示、改截图取点或前台策略时

---

## Current decision

`Window` 不再作为所有自动化对象的总称。

- `WindowService`、`WindowHandle`、`WindowInputSpec`、`NeedsWindow` 表示 Win32 HWND 窗口能力。
- `TargetService`、`target.Target`、controller factory、`NeedsTarget` 表示自动化目标能力。
- `AndroidTarget` 和 `BrowserTarget` 是 target selection node，不是 window node。
- `Win32WindowTarget` 是当前唯一的 Windows HWND target selection kind。
- 旧 `WindowTarget` kind 不再支持；项目未上线，不做 alias、不做旧容器 loader 兜底。

## Destructive naming rule

Phase 61 已执行破坏性重命名收敛，当前策略：

1. 运行时、validator、MCP schema、录制服务、前端事件/API 都只使用 `Win32WindowTarget`。
2. 用户可见名称写成“Windows 窗口目标 / Windows window target”，避免裸露 Win32 作为普通用户概念。
3. 错误码、事件名和工具方法包含 `WIN32_WINDOW_TARGET` / `win32windowtarget` / `Win32WindowTarget`，不保留旧 `WINDOW_TARGET` / `windowtarget` / `WindowTarget` contract。
4. Phase 63 已引入布尔 `NeedsTarget`；后续继续细化成 `NeedsTarget(kind=..., capabilities=...)`，不为旧 `WindowTarget` 留兼容层。

## Win32-only nodes

这些节点只应作用于 Windows HWND：

- `Win32WindowTarget`
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
- 输入、截图、视觉这类 target-aware 节点应声明 `NeedsTarget`，不是 `NeedsWindow`。
- 不要让 Browser target 伪装成 Chrome 主窗口优先点击；CDP/selector 是主路径。
- 不要让 Android target 伪装成模拟器窗口优先点击；ADB/minitouch/MAA-touch 风格 controller 是主路径。

## Naming rule

- Generic automation object: `目标` / `自动化目标` / `Target`
- Win32 HWND object: `Windows 窗口` / `Win32 window`
- Serialized kind: `Win32WindowTarget`
- Removed legacy kind: `WindowTarget`
