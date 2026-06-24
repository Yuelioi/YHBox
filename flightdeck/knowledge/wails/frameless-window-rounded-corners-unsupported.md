# ⚠ frameless 独立窗在 Win10 + wails3 alpha.91 做不出干净圆角 (别再调研)

SUMMARY: frameless 工具窗在 Win10 + wails3 alpha.91 加不了干净圆角, 是能力缺口非方法没试对, 别再调研
READ WHEN: 想给 frameless 独立窗 (录制/校准/截图/鼠标检测/悬浮窗 HUD) 加圆角 / 透明 / 毛玻璃前; 调研 wails3 窗口外观能力时

---

## 一句话

**这套 frameless 工具窗 (录制/校准/截图/鼠标 HUD/悬浮窗) 在 Windows 10 + wails3 v3.0.0-alpha.91 上加不了干净的圆角**。已逐处核 wails3 源码确认,不是"还没试对方法"——是这版 wails + Win10 的能力缺口。用户 2026-06-15 提的需求,调研结论 = 不做。

## 为什么 (全部核过 wails3 源码)

窗口在 `internal/services/tools/service.go` 全是 `Frameless:true` + 实心 `BackgroundColour: NewRGB(18,18,18)`。

1. **wails3 没有 Windows 原生圆角 API**。整个 v3 模块搜不到 `DWMWA_WINDOW_CORNER_PREFERENCE`/`DWMWCP`。跨平台的 `CornerRadius` 选项 (`webview_window_options.go:453`) **只在 macOS 实现** (`webview_window_darwin.go` 用了 `cornerRadius`),Windows 实现文件 `webview_window_windows.go` 压根没引用它。
2. **原生 DWM 圆角是 Win11 限定**。`DWMWCP_ROUND` 要 Windows 11 (build 22000+)。本机 Win10 19045,系统层就不支持。
3. **透明窗这条路被 click-through 堵死**。唯一能让 CSS `border-radius` 真显圆角的办法是透明窗 (四角透出)。但 wails3 对 `Frameless + BackgroundType=Transparent` **强制加 `WS_EX_TRANSPARENT` (鼠标穿透)** (`webview_window_windows.go:359-362`) → HUD 按钮全点不动。废。
4. **Translucent 岔路也不干净**。`BackgroundType=Translucent` 不穿透 (走 `WS_EX_NOREDIRECTIONBITMAP`),但会强制走毛玻璃 backdrop;`SupportsBackdropTypes()` 要 build **22621** (Win11 22H2),Win10 落到老式 `ACCENT_ENABLE_BLURBEHIND` (`setBackdropType` 的 `!Supports` 分支),**拖动出名地卡** + 关不掉,且它是模糊效果不是圆角。

直接加 `border-radius`: 窗口本体仍是实心硬矩形,CSS 只裁内层,四角露 `#121212` 方块 → 叠游戏/桌面上还是方角,白做。

## 怎么办

- **现状 (Win10)**: 保持方角。frameless 工具窗方角本就正常/专业。
- **未来 (若上 Win11)**: 可加一句 `DwmSetWindowAttribute(hwnd, 33 /*DWMWA_WINDOW_CORNER_PREFERENCE*/, &2 /*DWMWCP_ROUND*/)` 的 w32 syscall (约 10 行) → Win11 原生圆角 + 保留系统阴影 + 不用透明 + 不穿透;Win10 静默 no-op。**只在确认用户上 Win11 后做**,否则零可见收益。
- 想让这些窗"好看点"别卡在圆角:走 [standalone-window-style](../checklists/standalone-window-style.md) 的标题栏质感/内描边/状态面板配色统一。
