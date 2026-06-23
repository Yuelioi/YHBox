---
when_to_read: 把 backend 带时长输入操作拆成节点层 down/up / 改 PostMessage 激活逻辑 / 失焦窗口「第一次」按键或点击在游戏里没生效 / review 输入节点 ctx 化改动是否丢了激活时序
applies_to: [input, postmessage, fakeactivate, slate, node-refactor, ctx-cancellation, verification-gap]
last_updated: 2026-05-31
status: active
---

# 拆 timed-input 到节点层时丢了 backend 的激活时序

**Date**: 2026-05-31（input-editor-optimizations #4 实施中发现）

## 背景
#4 要让「长按 999999ms」能被停止/强停打断。改法（D4.1，用户拍板）= 在**节点层**把 `KeyPress`/`ClickAt` 的 Run 拆成 `KeyDown/MouseDown → select{ctx.Done / time.After} → KeyUp/MouseUp`，复刻 `Sleep` 节点的可取消范式。

改完节点单测全绿、用户强停 smoke 也过。**但**：

## 教训 (2 条)

### 1. 拆 down/up 等价 ≠ 行为等价 —— 丢了 backend 函数自带的激活时序
旧路径 `KeyPress` → `backend.KeyPress` → `Tap()`：内部 **每次** 都 `FakeActivate(hwnd)` + `sleep(activateDelay=30ms)` + down + sleep(hold) + up。`ClickAt` → `Click()`/`ClickButton()` 同理（还多 cursorSettle）。

新路径 `KeyPress.Run` → `backend.KeyDown`（`PostMessageBackend.KeyDown` 只 `ensureActivated`(once per hwnd, **无 delay**) + postMessage WM_KEYDOWN）→ ctx-sleep → `KeyUp`。

丢了两样：**(a) per-call FakeActivate**（`ensureActivated` 只首次 hwnd 翻一次），**(b) FakeActivate 后的 30ms `activateDelay`**。

后果（按 `pkg/input/input.go` 头注释的原理）：`FakeActivate` 是 `SendMessage(WM_ACTIVATE)`，同步返回**不代表** Slate 已翻 `IsActive=true`——它通常下一 UE tick 才翻。紧接着 post 的**首个** WM_KEYDOWN 在 `IsActive` 仍 false 时被异环 IMC **静默丢弃** → 失焦/后台窗口的第一次 cast 不生效。强停 smoke 测的是「按住能不能断」，测不到「首击丢不丢」。

**修法**：激活+settle 下沉到 backend 的 `ensureActivated`——首次激活后补 `time.Sleep(defaultActivateDelay)`（commit `32791d7`）。仅 per-hwnd 一次，所以分帧 `MoveTo`(60fps) 不会每帧 +30ms 卡死。
**遗留 edge**：跑中被外部 refocus → OS 给游戏发 WA_INACTIVE → `IsActive` 翻 false，`ensureActivated`(once) 不会 re-activate → 后续输入又开始丢。当前靠 `BringForeground` 节点兜；要根治得 per-call 激活（但 per-call settle 会卡死分帧 MoveTo，需分场景）。

### 2. 节点 ctx 化改动的验证必须跑「集成层」套件，不只节点包单测
#4 plan 的验证只跑 `./internal/nodes/... ./internal/node/`，**没跑** `./internal/services/container/runtime/`。后者的 fishing-v2 状态机集成测试断言 cast 发 `press:f`（单事件），节点改 down/up 后变 `down:f`/`up:f` → 6+ 测试早就红了，下个 batch 才撞见。
**判据**：凡改「节点 Run 怎么调 InputService / 发什么底层事件」，验证范围要含**跑真实容器**的 runtime 套件，不能只验节点包。
