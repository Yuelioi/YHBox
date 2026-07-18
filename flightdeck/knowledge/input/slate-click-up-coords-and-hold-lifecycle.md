---
kind: trap
summary: "UE Slate 点击坐标/松键/hover-settle 前案；3.1 held input 已改为 Run-owned lease，不再要求用 Sleep 让图保活。"
activation: symptom
read_when: "改鼠标点击/按住路径 (ClickAt / MouseHoldStart·Stop / PostMessageBackend.Mouse* / sendinput 坐标); UE 里点击点歪或点不到或「按住变单击」; 给 ClickAt 加可取消长按; review MouseDown/MouseUp 落点坐标或 hover 时序; 排查「held 按键莫名被松」类问题"
---
# ⚠ UE Slate 点击落点三根因 + hold 生命周期留尾
**Date**: 2026-06-08（用户报 ClickAt 在 UE 里点歪/点不到 → 一路查到 MouseHold「按住变单击」）

相关前案: 历史 timed-input 修法（已退役）（#4 把 timed-input 拆成节点层 down/up 的那次；本次 ClickAt 把它**拆回** Click，方向相反，见留尾 1）。

## 根因 + 修法（commit `9e7d6ac` / `0dda3a3`，均验证通过）

1. **点歪** — `sendInputBackend` 把窗口客户区 ratio 直接喂 `sendAbsMove`，但 SendInput 绝对坐标基准是**整屏**。修: `clientRatioToScreenRatio` = 客户区 ratio →(×GetClientRect)客户区像素 →(ClientToScreen)屏幕像素 →(÷SM_CXSCREEN)全屏 ratio。`CursorRatio` 同步走逆变换。2026-07-14 起默认后端已改为 `sendinput`，因此这条现在属于默认生效路径。

2. **点不到（松键落点）** — `PostMessageBackend.MouseUp` 旧实现 `MouseBtnUp(hwnd, 0, 0, btn)`，WM_xBUTTONUP 的 lParam 是 (0,0)。**UE Slate 在 BUTTONUP 判 click 且看坐标** → 按在 (x,y)、松在 (0,0) = 控件外松手 → 不触发。修: `heldBtns` 从 `map[...]struct{}` 改存按下坐标 `map[...]point`，松键回按下坐标（ReleaseAll 同改）。**这条同时修好了 `MouseHoldStop`。**

3. **点不到（按下落点）** — 旧 ClickAt 走节点层 down/up 拆分 → `PostMessageBackend.MouseDown` → 裸 `MouseBtnDown`，**缺** `ClickButton` 那套 `setCursorPos + WM_MOUSEMOVE(hover) + cursorSettle`（input.go:287-298 注释: Slate 需一个 tick 处理 MOUSEMOVE 更新 hover 元素再收 DOWN）。修: ClickAt 改走 `Click`(→ClickButton)；并给 `PostMessageBackend.MouseDown` 也补上同序 hover-settle（`MoveToClient` + `defaultCursorSettle`），让拆分按下路径跟点击路径一致。

## 现在没事、以后可能咬人的三处留尾

### 留尾 1: ClickAt 拆回 Click，反转了 #4 的可取消拆分
历史 timed-input 修法（已退役） 里 #4 特意把 ClickAt 拆成 down→`select{ctx.Done/After}`→up，为的是**长按 DurationMs 能中途强停**。本次为了 Slate 落点把它**改回** `Click`，而 `Click` 内部是 down→裸`time.Sleep(hold)`→up，**hold 期间不可取消**。
- 快速点击（几十 ms）: 无影响。
- 故意长按（DurationMs 设几秒）: 停容器时会等 hold 走完才松键，重蹈 #4 想治的「长按停不下」。
- 谁要「可取消长按」: 用 `MouseHoldStart` + Wait + `MouseHoldStop`，别给 ClickAt 加长 hold。

### 留尾 2: MouseDown 的 hover-settle 没隔离验证，且会动真光标
给 `PostMessageBackend.MouseDown` 补的 hover-settle 是「跟已验证的 ClickButton 同序」的**保守一致性**改动，**不是隔离证明过必需**——ClickAt 那次修复同时带了「松键坐标」和「hover-settle」两件事，无法证明单独 hover-settle 起了作用，也可能光松键坐标就够。后果两点:
- 每次 MouseDown 多 `setCursorPos`（**动真 OS 光标**）+ `defaultCursorSettle`(30ms)。PostMessage 后端「后台不动用户光标」的预期被打破（ClickButton 本来也动但**会复位**，MouseDown 是 hold 语义**不复位**，光标停在按下点）。
- 哪天想抠最小改动 / 怀疑这步多余: 单独回退 `0dda3a3` 对着 UE 测一次即可证伪。

### 留尾 3: hold 类操作，图必须保活，否则 teardown 的 ReleaseAll 把按住塌成单击
排查 detour: 用户只摆一个 `MouseHoldStart` → 图跑完立刻结束 → `runner.go:375 teardownRuntime defer ReleaseAll` 发 MouseUp → 按钮被激活进菜单 → 看起来「按住变单击」。**`ReleaseAll` 是容器 stop/panic/cancel 的防卡键兜底，不是每 tick 调**——但「图跑完 = stop」，所以单节点 hold 必塌。
- 判据: 凡 `MouseHoldStart`（或任何 down-不-up 的 held 操作），**Start 和松手之间图必须保活**（垫 Sleep/Wait/后续节点）到你想松手为止。
- 调试教训: 我一度因为「ReleaseAll 只在 stop 调」就把它从嫌疑里排除——漏了「单节点图的 stop 是立即的」。held 按键莫名被松，第一个怀疑对象就是 teardown 的 ReleaseAll。


## 3.1 R2 lifecycle update

坐标、松键落点和 hover-settle 仍是排障知识；“图靠 Sleep 保活”不再是 3.1 contract。Hold Keys / Hold Pointer Button 返回 durable HandleRef，由 Run owner 持有；显式 Release Held Input 或 Run cancel/failure/teardown 释放。提前消失应检查 HandleRef owner/close path，不能指导用户添加无业务意义的等待节点。