# ✅ 已解决：SendInput drag 不再走 PostMessage
**Date**: 2026-06-24 (Phase 3 detect/click 整支终审发现)

**Resolved**: 2026-07-18，3.1 R2。现在全程原生 SendInput，native recorder hook 证明 down/up；以下保留为历史根因。

sendinput 后端存在的全部理由 = 读 RawInput/DirectInput 的游戏收不到 PostMessage 的窗口消息, 所以它的 Click/MouseDown 走 SendInput 真实注入 (sendAbsMove / sendMouseBtnEvent)。

但 `sendInputBackend.Drag` (pkg/input/sendinput_windows.go) 直接调共享原语 `pkg/input.MouseDrag` (pkg/input/input_windows.go), 而 MouseDrag 完全是 `FakeActivate + setCursorPos + postMessage(WM_MOUSEMOVE / WM_xBUTTONDOWN / WM_xBUTTONUP)` 实现。⇒ 选 sendinput 后端跑 Swipe, 拖拽实际走窗口消息, 这类游戏看不见; 无报错、单测也过 (拖拽测试全用 mock / PostMessage 路径)。

**注意**: MouseDrag 是 Phase 3 之前就有的共享原语, 两个后端 (postmessage / sendinput) 都这么调, **非 Phase 3 回归** — Phase 3 只是把它经 Swipe 暴露出来。

**已采用修复**: `sendAbsMove` → button down → 分帧插值 → button up，全程 SendInput；失败沿 backend/provider/node 返回。