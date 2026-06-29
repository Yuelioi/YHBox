# Window vs Target Boundary

SUMMARY: 明确 Window 只表示 Win32 HWND 窗口，Target 表示跨 Windows/Android 的用户可见自动化对象；Browser CDP 仅剩内部后端代码
READ WHEN: 改 Win32WindowTarget / AndroidTarget / Browser CDP controller / NeedsTarget / NeedsWindow / TargetService / 窗口控制节点 / 输入和截图路由时
RECHECK WHEN: 新增目标类型、重命名节点 kind、改 validator 缺目标提示、改截图取点或前台策略时

---

## Current decision

`Window` 不再作为所有自动化对象的总称。

- `WindowService`、`WindowHandle`、`WindowInputSpec`、`NeedsWindow` 表示 Win32 HWND 窗口能力。
- `TargetService`、`target.Target`、controller factory、`NeedsTarget` 表示自动化目标能力。
- `AndroidTarget` 是 target selection node，不是 window node。
- `BrowserTarget` 用户节点已删除；Browser CDP controller/client 暂留内部代码，不作为普通节点入口。
- `Win32WindowTarget` 是当前唯一的 Windows HWND target selection kind。
- 旧 `WindowTarget` kind 不再支持；项目未上线，不做 alias、不做旧容器 loader 兜底。

## Destructive naming rule

Phase 61 已执行破坏性重命名收敛，当前策略：

1. 运行时、validator、MCP schema、录制服务、前端事件/API 都只使用 `Win32WindowTarget`。
2. 用户可见名称写成“Windows 窗口目标 / Windows window target”，避免裸露 Win32 作为普通用户概念。
3. 错误码、事件名和工具方法包含 `WIN32_WINDOW_TARGET` / `win32windowtarget` / `Win32WindowTarget`，不保留旧 `WINDOW_TARGET` / `windowtarget` / `WindowTarget` contract。
4. Phase 63 已引入布尔 `NeedsTarget`；Phase 64 已新增 `TargetCapabilities` 细化目标能力，不为旧 `WindowTarget` 留兼容层。

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

- 不要把 Android target 或内部 Browser CDP target 加进 `WindowService`。
- 不要让“目标窗口”继续作为通用 UI 词。
- 不要在新节点里用 `NeedsWindow` 表示“需要任意自动化目标”。
- 输入、截图、视觉这类 target-aware 节点应声明 `NeedsTarget`，不是 `NeedsWindow`。
- `NeedsTarget` 节点必须声明 `TargetCapabilities`；不要只靠布尔标记表达跨平台能力。
- 不要恢复旧 `BrowserTarget` 普通节点；真要做浏览器自动化，先设计 `打开网页/等待元素/填表单/点击文本/提取文本` 这类用户向节点，底层再选 Playwright/CDP。
- 不要让 Android target 伪装成模拟器窗口优先点击；ADB/minitouch/MAA-touch 风格 controller 是主路径。

## Naming rule

- Generic automation object: `目标` / `自动化目标` / `Target`
- Win32 HWND object: `Windows 窗口` / `Win32 window`
- Serialized kind: `Win32WindowTarget`
- Removed legacy kind: `WindowTarget`
