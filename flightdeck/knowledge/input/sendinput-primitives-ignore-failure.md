---
kind: trap
summary: "sendinput 后端 keyboard path 已检查 `procSendInput.Call` 注入数, 但鼠标按钮/绝对移动/滚轮/相对移动/TypeText 仍丢弃返回值, 真 SendInput 失败可能不到节点 Fail 出口"
activation: symptom
read_when: "给 pkg/input 后端方法加错误处理 / 排查「输入静默失败但节点走 Done」/ 评估 InputText 等输入节点 error 路径的生产可靠性"
---
# ⚠ pkg/input 部分 SendInput 原语不查注入数, 失败上报不到节点层
**Date**: 2026-06-24 (Phase 3 InputText 终审定夺)

`procSendInput.Call(1, ...)` 的正确语义: 第一返回值 = 成功注入的事件数; 第二/三返回是 syscall sentinel (恒非 nil, 必须忽略); **真信号是 `ret < 期望事件数`**。

当前状态 (2026-06-30 源码核验):

- `sendKeyEvent` 已检查注入数并返回 error; `KeyDown/KeyUp` 能向上报失败。
- `sendMouseBtnEvent` / `sendAbsMove` / `sendWheel` / `sendHWheel` / `sendInputMouseRel` 仍直接丢弃返回值。
- `TypeText` 逐 UTF-16 code unit 调 SendInput down/up, 仍直接丢弃返回值。
- `Drag` 仍调共享 `MouseDrag` PostMessage 原语, 另见 [sendinput-drag-uses-postmessage.md](sendinput-drag-uses-postmessage.md)。

⇒ `InputService.TypeText/Click/Drag/Scroll/MouseMoveRel` 的 `error` 返回在 sendinput 后端上仍可能假绿; 节点 (如 InputText/ClickAt/Scroll) 的 `node.Failf(CodeSendFailed, ...)` Fail 路径可能只在 mock 注入 error 时触发, 生产 SendInput 真失败 (UIPI: 目标进程提权高于脚本 / 锁屏桌面) 不一定走到。

**修复方向 (未做完)**: 一个 pass 给所有剩余 SendInput 原语加「注入数 < 期望 → 返回 error」, 错误沿 Backend → InputService → 节点 Fail 出口上报。
