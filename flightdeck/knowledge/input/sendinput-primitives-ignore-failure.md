# ⚠ pkg/input 的 SendInput 原语不查注入数, 失败上报不到节点层

SUMMARY: sendinput 后端所有原语 (sendKeyEvent / sendWheel / sendHWheel / sendMouseBtnEvent / sendAbsMove / sendInputMouseRel + TypeText) 都丢弃 procSendInput.Call 的返回值恒不报错 — InputService/Backend 方法的 error 返回在 sendinput 后端上实际恒 nil, 真 SendInput 失败 (UIPI 提权拦截 / 锁屏桌面) 到不了节点的 Fail 出口
READ WHEN: 给 pkg/input 后端方法加错误处理 / 排查「输入静默失败但节点走 Done」/ 评估 InputText 等输入节点 error 路径的生产可靠性

---

**Date**: 2026-06-24 (Phase 3 InputText 终审定夺)

`procSendInput.Call(1, ...)` 的正确语义: 第一返回值 = 成功注入的事件数; 第二/三返回是 syscall sentinel (恒非 nil, 必须忽略); **真信号是 `ret < 期望事件数`**。但 pkg/input 全家 (sendKeyEvent / sendAbsMove / sendWheel / sendHWheel / sendMouseBtnEvent / sendInputMouseRel / TypeText) 都不查 ret, 一律返回成功。

⇒ `InputService.TypeText/Click/Drag/Scroll` 的 `error` 返回在 sendinput 后端上**实际恒 nil**; 节点 (如 InputText) 的 `node.Failf(CodeSendFailed, ...)` Fail 路径只在 mock 注入 error 时触发, 生产 SendInput 真失败 (UIPI: 目标进程提权高于脚本 / 锁屏桌面) 走不到。

**为何当时不修 (终审裁决, opus)**: 这是**全包既有模式**、非 TypeText 独有。单修 TypeText = 全包唯一不一致的方法 (更糟, 违反「不打补丁」); 要修就一次性给全家加 `ret < want → error`, 是独立 refactor。失败模式低危 (UIPI/锁屏同样挡 PostMessage 路径, 节点层 hwnd/ensure guard 已部分兜住; 部分注入是质量问题非数据丢失)。按二号铁律 (一处修干净) + YAGNI, 无 demand 前整支不做。

**修复方向 (未做)**: 一个 pass 给所有 SendInput 原语加「注入数 < 期望 → 返回 error」, 错误沿 Backend → InputService → 节点 Fail 出口上报。
