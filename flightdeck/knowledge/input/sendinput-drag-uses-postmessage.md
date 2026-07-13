---
kind: trap
summary: "sendInputBackend.Drag 调共享原语 pkg/input.MouseDrag, 而 MouseDrag 是 PostMessage(WM_MOUSEMOVE/BUTTONDOWN/UP) 实现 — 选 sendinput 后端跑 Swipe 时走的是窗口消息而非 SendInput 全局注入, 读 RawInput/DirectInput 的游戏收不到拖拽, 且无报错"
activation: symptom
read_when: "改 Swipe/拖拽 / 选 sendinput 后端跑拖拽 / 排查「拖拽在某游戏不生效但点击生效」/ 给 sendinput 后端补原生 Drag"
---
# ⚠ Swipe 在 sendinput 后端走 PostMessage, Raw-Input 游戏收不到
**Date**: 2026-06-24 (Phase 3 detect/click 整支终审发现)

sendinput 后端存在的全部理由 = 读 RawInput/DirectInput 的游戏收不到 PostMessage 的窗口消息, 所以它的 Click/MouseDown 走 SendInput 真实注入 (sendAbsMove / sendMouseBtnEvent)。

但 `sendInputBackend.Drag` (pkg/input/sendinput_windows.go) 直接调共享原语 `pkg/input.MouseDrag` (pkg/input/input_windows.go), 而 MouseDrag 完全是 `FakeActivate + setCursorPos + postMessage(WM_MOUSEMOVE / WM_xBUTTONDOWN / WM_xBUTTONUP)` 实现。⇒ 选 sendinput 后端跑 Swipe, 拖拽实际走窗口消息, 这类游戏看不见; 无报错、单测也过 (拖拽测试全用 mock / PostMessage 路径)。

**注意**: MouseDrag 是 Phase 3 之前就有的共享原语, 两个后端 (postmessage / sendinput) 都这么调, **非 Phase 3 回归** — Phase 3 只是把它经 Swipe 暴露出来。

**修复方向 (未做, 等 demand)**: 给 sendinput 加原生 Drag — `sendAbsMove` 到起点 → `sendMouseBtnEvent` 按住 → 分帧 `sendAbsMove` 插值 → `sendMouseBtnEvent` 松开, 全程 SendInput。需要在「读 RawInput 的游戏里拖拽」时再做 (二号铁律 YAGNI: 无 user demand 不预先加)。
