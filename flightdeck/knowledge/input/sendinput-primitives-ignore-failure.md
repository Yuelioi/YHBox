# ✅ 已解决：SendInput 原语统一验证注入数
**Date**: 2026-06-24 (Phase 3 InputText 终审定夺)

**Resolved**: 2026-07-18，3.1 R2。所有生产原语以实际注入数为成功信号；以下清单保留为历史审计。

`procSendInput.Call(1, ...)` 的正确语义: 第一返回值 = 成功注入的事件数; 第二/三返回是 syscall sentinel (恒非 nil, 必须忽略); **真信号是 `ret < 期望事件数`**。

当前状态 (2026-06-30 源码核验):

- `sendKeyEvent` 已检查注入数并返回 error; `KeyDown/KeyUp` 能向上报失败。
- `sendMouseBtnEvent` / `sendAbsMove` / `sendWheel` / `sendHWheel` / `sendInputMouseRel` 仍直接丢弃返回值。
- `TypeText` 逐 UTF-16 code unit 调 SendInput down/up, 仍直接丢弃返回值。
- `Drag` 仍调共享 `MouseDrag` PostMessage 原语, 另见 [sendinput-drag-uses-postmessage.md](sendinput-drag-uses-postmessage.md)。

⇒ `InputService.TypeText/Click/Drag/Scroll/MouseMoveRel` 的 `error` 返回在 sendinput 后端上仍可能假绿; 节点 (如 InputText/ClickAt/Scroll) 的 `node.Failf(CodeSendFailed, ...)` Fail 路径可能只在 mock 注入 error 时触发, 生产 SendInput 真失败 (UIPI: 目标进程提权高于脚本 / 锁屏桌面) 不一定走到。

**已采用修复**: 注入数不足立即返回 error，并沿 Backend → installed provider → node failure 上报。