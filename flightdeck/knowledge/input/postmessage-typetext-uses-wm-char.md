---
kind: trap
summary: "PostMessageBackend.TypeText 旧实现直接调 pkg/input.TypeText (全局 SendInput KEYEVENTF_UNICODE), 注入到真实前台焦点窗口 — 但 postmessage 模式面向后台、目标窗口不持焦, 字符全进错窗口; 同窗口 KeyPress 走 targeted WM_KEYDOWN 却能打字. 已修: 改走 PostText (PostMessage WM_CHAR, targeted hwnd)"
activation: symptom
read_when: "改 InputText/TypeText / 排查「文本输入节点在后台窗口没生效但按键/点击生效」/ 给后端补文本输入 / review TypeText 走哪条路"
recheck_when: "TypeText 实现改动 / 默认后端从 postmessage 换走 / WM_CHAR 对某类窗口 (Chromium/Slate) 真机表现有新发现"
---
# ⚠ InputText 在 postmessage 后端旧实现走全局 SendInput, 后台目标窗口收不到
**Date**: 2026-06-25 (detect-click 真机 smoke 发现: 记事本/vscode InputText 不输入, 同窗口 KeyPress 能输入)

## 根因
`InputText` → `ctx.Services().Input.TypeText` → backend.TypeText。`PostMessageBackend.TypeText` **旧实现**直接 `return TypeText(hwnd, s)`, 而 pkg 级 `TypeText` (pkg/input/typetext_windows.go) 走**全局 SendInput KEYEVENTF_UNICODE**, `func TypeText(_ win.HWND, ...)` 把 hwnd 直接丢弃 —— SendInput 注入到**真实持有键盘焦点的前台窗口**。

但 postmessage 后端整个设计前提 = 目标窗口可在后台、不抢前台 (`BackgroundInput=true`)。跑流程时焦点在 Yotta GUI / 别处, 目标窗口 (记事本/vscode) 后台无焦 → SendInput 的字符全注入到错误窗口 → 目标一个字收不到、**且不报错**。2026-07-14 起容器默认已改为 `sendinput`；本条仍约束用户显式选择 PostMessage 的路径。

对照实证 (用户真机): **同一个 vscode 窗口**, `KeyPress` (按键) 能打字、`InputText` 不能。差异就在 KeyPress 走 `PostMessageBackend.KeyDown` → `postMessage(hwnd, WM_KEYDOWN)` (**targeted hwnd, 后台可用**), InputText 走全局 SendInput。

这是 [sendinput-drag-uses-postmessage.md](sendinput-drag-uses-postmessage.md) 的**镜像同类**: 那条是「选 sendinput 后端但 Drag 错走 PostMessage」, 这条是「选 postmessage 后端但 TypeText 错走全局 SendInput」—— 共因都是某个文本/拖拽操作没跟随所选后端的语义。backend.go 接口注释当年还把这个 bug 写成了设计 ("PostMessage 实现走相同 SendInput path")。

## 修复 (已做)
PostMessage **能**直接发 unicode: 用 `WM_CHAR (0x0102)`, wParam = UTF-16 code unit, lParam=0, targeted 投递到 hwnd。新增 `pkg/input.PostText` (逐 rune 拆 UTF-16 → 每 code unit 一条 WM_CHAR, BMP 外字符拆 surrogate pair)。`PostMessageBackend.TypeText` 改为 `ensureActivated(hwnd)` + `PostText(hwnd, s)`, 与本后端 KeyDown/Click 同语义。`sendInputBackend.TypeText` **保持**走全局 `TypeText` (它本就要前台, 语义正确)。

## 边界 (真机待确认 / 未来 demand)
- WM_CHAR 只投字符、不产 `WM_KEYDOWN/UP`。标准 Edit / RichEdit / Slate (OnKeyChar) / Chromium 处理 WM_CHAR 的控件都收得到; 极少数只认完整 keydown→char→keyup 序列的自绘输入框收不到 → 那类切 `sendinput` 后端 (前台全局)。
- Chromium (vscode/Electron) 对 PostMessage WM_CHAR 的接收靠真机 smoke 确认 —— 用户已验证它响应 PostMessage WM_KEYDOWN, WM_CHAR 大概率同样可收。
