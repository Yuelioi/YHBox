# YHBox 内部实现笔记

面向维护者/二次开发者，重点记**不显然的细节**和**踩过的坑**。用户向说明在 [README.md](README.md)。

---

## 1. 项目脉络

YHBox 是异环（UE5 + Slate UI 游戏）的后台自动化工具集。核心约束：

- **真后台**：截图走 `PrintWindow`（被遮挡也能抓）、输入走 `WM_ACTIVATE + PostMessage`，全程不抢前台焦点、不动光标。
- **单 exe 分发**：所有模板 PNG 通过 `embed` 进二进制，无 cgo、无 OpenCV。这是 Go 移植的核心动机，任何技术选型都先回到这条目标。
- **多分辨率**：720p / 1080p 实测覆盖（精确匹配，无跨分辨率缩放兜底）。其它分辨率需用户提交模板给作者补 embed。

工具集目前两个：`fish`（自动钓鱼，9 个 state）、`cook`（锤子连点）。

---

## 2. 架构平面

```text
cmd/yhbox/main.go   入口：EnsureAdmin → gui.Run(version)
   │
   ├── cmd/yhbox/gui     GUI 层（lxn/walk）— 唯一面向用户的界面
   │     ├── app.go             MainWindow + TabWidget + 共享 logSink/settings/botOwner
   │     ├── tab_fish.go        自动钓鱼 tab：游戏状态/AutoSell/统计区/开始-暂停-停止按钮
   │     ├── tab_cook.go        锤子连点 tab：游戏状态/Interval/开始-暂停-停止按钮
   │     ├── tab_settings.go    日志开关 / 自动滚动 / 显示统计区
   │     ├── tab_help.go        帮助文本
   │     ├── tab_about.go       关于（版本号、链接）
   │     ├── richlogedit.go     RichEdit50W 包装（彩色日志、行距、文本左右内边距、剥 client edge）
   │     ├── control.go         guiControl 实现 runctl.Control（sync.Cond 阻塞 WaitUnpause）
   │     ├── logsink.go         LogSink (ring buffer + 增量 onNewLines 回调)
   │     ├── settings.go        Settings JSON 读写（fish/cook/ui 三段）
   │     └── resolution.go      DetectGameWindow → GameStatus（winutil 找窗口 + 抓帧拿尺寸）
   │
   ├── tools/fish        状态机：IDLE / SETUP / WAITING / FISHING / RESULT / RECOVERING
   │     │                       + SHOPSELL / BUYBAIT / CHANGEBAIT（声明式 flow）
   │     ├── machine.go         struct + run() 主循环 + Stats + StatsHook
   │     ├── states_fishing.go  IDLE/SETUP/WAITING handler + tryHookF
   │     ├── states_fight.go    FISHING/RESULT handler + recordOutcome + classifyFish + pressEscUntilClear
   │     ├── states_recover.go  RECOVERING handler + enterRecover
   │     ├── states_shop.go     3 个声明式 flow{} 字面量 + 1 行 dispatcher
   │     ├── flow.go            Step / flow / runFlow 引擎 + 原语 (tap/waitDur/clickIfSeen/...)
   │     ├── phase.go           UIPhase + phaseTable 表驱动 inspectPhaseFrame / routePhase
   │     ├── detect.go          模板加载（embed only）+ 各 slot detect 方法 + PickBarROI
   │     ├── bar.go             耐力条颜色分析（findCursor / findTarget）
   │     ├── control.go         fishingControl{controlDir} + chooseDirection + applyDirection
   │     ├── state.go           State enum + DebugXXX 位掩码常量（保留备用，runtime 未驱动）
   │     ├── constants.go       所有延迟 / 置信度 / 超时 / 重试包级常量 + roiFishingBars
   │     └── config.go          Config{AutoSell atomic.Bool} — 当前唯一用户偏好字段
   │
   ├── tools/cook        单循环锤子检测 → 点击；仍支持 templates/cook.toml 用户外挂
   │
   └── pkg/
         ├── capture/      PrintWindow 抓帧（Frame / FrameROI）
         ├── input/        WM_ACTIVATE + PostMessage 键鼠
         ├── vision/       CCOEFF_NORMED 模板匹配，2× 下采样金字塔，PickBest 精确匹配
         ├── runctl/       Control 接口（Pause/Resume/Stop/WaitUnpause/IsPaused/IsBack）
         ├── log/          多 sink Logger（每条消息广播到 file + GUI LogSink）
         ├── platform/     UAC 提权
         └── winutil/      游戏窗口枚举（按 class 名 `UnrealWindow`）
```

### 钓鱼状态机

钓鱼模块的完整状态机 / 时序 / 检测链 / 兜底机制 → 见 **[fish.md](fish.md)**。

简化总览：

```text
IDLE ─StartFish→ SETUP ─click→ IDLE
IDLE ─HookIconDim→ 抛竿F → 0.3s 探 NeedBait/WarehouseFull/StartFish → WAITING
WAITING ─HookText (≥2 帧)→ tryHookF (60s 闭环) → FISHING
WAITING ─[75s 超时]→ inspectPhase → 路由 / RECOVERING
FISHING ─控制 A/D / 监 bar→ Result/FishEscape → IDLE
FISHING ─[40s 超时]→ RECOVERING

NeedBait      → BUYBAIT  → shop flow → CHANGEBAIT → IDLE
WarehouseFull → SHOPSELL → shop flow → IDLE
```

本节后续仅讲跨模块通用机制（输入、截图、模板匹配等），不重复 fish.md。

---

## 3. 后台输入子系统（[pkg/input/input.go](pkg/input/input.go)）

这是踩坑最多的地方，按出现频率列：

### 3.1 WM_ACTIVATE 翻 IsActive

UE 的 IMC（Input Mapping Context）默认丢弃 `IsActive=false` 窗口的键盘消息。关键 trick：

```go
SendMessage(hwnd, WM_ACTIVATE, WA_ACTIVE, 0)
```

**只翻窗口内部的 IsActive，不改变 GetForegroundWindow 返回值。** 之后必须 sleep 一段（`ActivateDelay`，默认 30ms）让 Slate 把 IsActive 真正翻过来——SendMessage 同步返回不代表 Slate 处理完，它通常下一 UE tick 才生效。

如果 ActivateDelay 太短，PostMessage 的 KEYDOWN 可能在 IsActive 还是 false 时到达，被丢。

### 3.2 keyLParam 必须填 scancode

**最隐蔽的坑**。`WM_KEYDOWN/UP` 的 `lParam` bits 16-23 是 scancode，UE InputComponent 用 scancode 查 ProfileMap，缺了会零星丢键（首次正常、后面失败，没规律）。

```go
sc := MapVirtualKeyW(vk, MAPVK_VK_TO_VSC) & 0xFF
lp := uintptr(1) | (uintptr(sc) << 16)   // bit 0-15 repeat=1, bit 16-23 scancode
if keyUp {
    lp |= (1 << 30) | (1 << 31)
}
```

历史症状：抛竿稳定，按 F 收线/按 ESC 偶发失败。

### 3.3 Click 时序——DOWN/UP 必须紧贴

UE Slate 按钮把 DOWN→长 sleep→UP 当成"**按住**"，不触发 click 事件。必须 DOWN 和 UP **紧贴发**，sleep 放在 UP 之后。完整顺序：

```text
FakeActivate → ActivateDelay(30ms)
  → SetCursorPos(屏幕坐标)
  → CursorSettleDelay(5-30ms)    ← 让 OS 发出自然 MOUSEMOVE
  → PostMessage(WM_LBUTTONDOWN)
  → PostMessage(WM_LBUTTONUP)     ← 紧贴 DOWN
  → sleep ClickHold(30ms)         ← 让游戏 tick 处理 click 事件
  → SetCursorPos(原位置)          ← 还原原位置，用户感知是光标闪一下回来
```

**踩过的坑**：

1. **加 `WM_MOUSEMOVE` PostMessage 反而会让 cook 漏点更多**（30%→稳定漏 70%）。`SetCursorPos` 已经让真光标进了客户区，OS 会自然给游戏发 MOUSEMOVE，再 PostMessage 一次造成 hover 状态抖动。
2. **`ClickHold` 100-150ms 把 DOWN-UP 间隔拉长**，cook 实测命中率只有 60%。改成 DOWN-UP 紧贴 + sleep 放后面，命中率到 95%+。
3. **`CursorSettleDelay` 80-120ms 是过度防御**，实测 5ms 就够 OS 发出 MOUSEMOVE 给 Slate。fish 这边的 flow 引擎里所有 click 都用 `delayShort=30ms`（含 click hold / activate / cursor settle）。

参考实现：[ok-script GenshinInteraction.do_click](https://github.com/ok-oldking/ok-script/blob/master/ok/device/intercation.py)（同样是 UE 游戏，模式一致）。

### 3.4 ClickHold vs TapHold（语义不同）

| 字段 | ClickHold | TapHold |
| --- | --- | --- |
| 作用 | DOWN+UP **之后**等多久（after-click） | KEYDOWN→KEYUP **之间**间隔（press hold） |
| 用途 | UI 按钮点击 | F / ESC / A / D 等键盘按键 |
| 默认 | 30ms (`delayShort`) | 150ms (`delayMid`) |

**鼠标**：DOWN/UP 紧贴 = "click"，长 hold = "按住"。所以 click_hold 改成"after-click sleep"语义，让游戏 tick 处理事件。

**键盘**：UE InputComponent tick 轮询采样，必须保持至少一帧（30Hz 下 33ms）才会被采样到，留 150ms 给低帧率/IME 余量。

### 3.5 检测到 UI 立刻按键 = 丢键

**动画淡入期间，UI 元素第一帧出现时游戏的 InputAction 还没在 PlayerController 上注册**。立刻 PostMessage 等于按到空气。

抛竿走 `FStreak`（连续 3 帧/200ms 确认）天然延迟，所以稳。上钩一开始用 conf 突变那一帧立刻按 F → 经常丢。修法是抄 FStreak：触发后再确认 `hookStreakCount`（=2）帧 textOK 才按 F。

任何"检测到立刻按键"的逻辑，都先想想动画时序。

---

## 4. 后台截图（[pkg/capture/capture.go](pkg/capture/capture.go)）

### 4.1 PW_RENDERFULLCONTENT

```go
PrintWindow(hwnd, hdc, PW_RENDERFULLCONTENT)  // 0x02
```

UE 走 D3D 后端，不带 `PW_RENDERFULLCONTENT` 拿到的是黑屏。这个 flag 让 PrintWindow 等 GPU 渲染完再 blit 到 DC。

### 4.2 客户区 vs 窗口区

`PrintWindow` 抓的是**整个窗口**（含标题栏 / 边框）。但 UE 渲染、所有 ROI 比例都是按**客户区**算的。需要：

```go
GetClientRect → clientW, clientH                      // 渲染区尺寸
ClientToScreen(0,0) - winRect.Left/Top → offsetX/Y    // 客户区在窗口位图中的偏移
```

读位图时 `srcY = y + offsetY`、`srcX = x + offsetX`。如果忘了 offset，1080p 窗口模式下整体偏 8-30 像素，模板匹配 conf 直接掉到 0.4 以下。

### 4.3 Frame vs FrameROI

`Frame` 转换整个客户区 BGRA→RGBA，1080p 下 ~3MB；`FrameROI` 只转换指定矩形。

WAITING 阶段每 250ms 抓一次（hook text 区域，~400×40），FISHING 阶段每 30ms (`loopInterval`) 抓一次耐力条（约 700×12）——这些都用 `FrameROI`。如果用 `Frame`，FISHING 单循环耗时从 ~1ms 涨到 ~15ms，控制器跟不上目标移动。

PrintWindow 本身仍然抓全窗口（API 限制），优化是省 BGRA→RGBA 像素转换。

---

## 5. 检测管线（[tools/fish/detect.go](tools/fish/detect.go)）

### 5.1 模板匹配

[pkg/vision/template.go](pkg/vision/template.go) 实现 CCOEFF_NORMED（与 OpenCV `TM_CCOEFF_NORMED` 等价）：

- 积分图把 patch 的 `∑I` / `∑I²` 降到 O(1)
- 行级 goroutine 并行（`runtime.NumCPU`）
- `MatchFast`：2× 下采样粗找 + 原图 ±4 邻域精修，1080p 下约 8.9ms
- 灰度化用 BT.601（0.299R + 0.587G + 0.114B），与 OpenCV 默认一致

### 5.2 模板尺寸必须 < ROI

血泪教训：曾有过模板 43px 高 vs ROI 42px 高 → 搜索空间为 0 → 永远 conf=-1。**写检测代码前先验证 ROI 像素尺寸 ≥ 模板尺寸。**

低分辨率（720p）下尤其容易出。当前实现的对策是 **每个分辨率独立 embed 一份模板**（hook_text_720.png / hook_text_1080.png），跑 NCC 时模板尺寸天然就是基准分辨率的尺寸，不再做"按 h/1080 等比缩放"那种跨分辨率缩放。

### 5.3 半透明 UI 的 NCC 教训：模板要从真实游戏抠

历史踩坑：曾经认为 NCC 对半透明背景敏感（conf 跨场景在 0.4-0.9 漂）→ 放弃文字检测改用 icon + cyan 颜色校验。

**真相**：根因不是 NCC，是模板和实际渲染像素级对不上（字体抗锯齿、alpha 渲染差异）。当模板从真实游戏截图抠出（带精确 bbox）时，NCC 在 day/night 跨场景 conf 都 ≥ 0.99。

新约定：模板元数据走 `templates.toml`（file/resolution/bbox/note），加载用 `vision.BuildNamedTemplate`，`MatchTextROI` 自动按 bbox + `roiPaddingPx (30)` 算 ROI。完整规范见 [fish.md §9](fish.md)。

**算法对比工具**：[pkg/vision/text_match_test.go](pkg/vision/text_match_test.go) 跑 4 种灰度预处理（plain BT.601 / V channel / Otsu 二值化 / Sobel 边缘）的 NCC 对比，输入是 `debug/hook@full@*.png` 全帧截图。下次再怀疑某算法对半透明文字更稳时，dump 一张失败帧丢 debug/ 重跑 `go test -v ./pkg/vision/ -run TestHookTextAlgoCompare` 直接看 conf 表，不用瞎改 detect.go。当前数据下 plain 已 ≥0.99，结论维持。

### 5.4 模板配置表（assets/templates.toml）

模板元数据集中在 [assets/templates.toml](assets/templates.toml)，PNG 文件名只是引用 key。
TOML 结构 `[[<tool>.<slot>]]` grouped，slot 由父级 key 推断（不重复写）。

每一项包含 `file` / `resolution=[W,H]` / `bbox=[x1,y1,x2,y2]` / `note`。
完整 schema 见 [pkg/vision/spec.go](pkg/vision/spec.go) `TemplateSpec`。

**Detector 行为**：启动时读 templates.toml → 按 `[tool][slot]` 分组 → 检测时
`vision.PickBest` 按当前游戏帧 W×H **精确匹配** Resolution。**找不到就返回 nil**（detect 返回 conf=-1），不再做"按 aspect/H 最近邻"或 `ScaleTemplate` 跨分辨率自动缩放——v2.2.0 实测跨分辨率 NCC conf 损失明显，宁缺勿滥。

**fish 不再支持用户外挂模板**：v2.2.0 起删除了 `templates/fish.toml` 加载路径。需要新分辨率请把截图 + bbox 提交给作者，由作者审过加 embed。理由：跨分辨率 NCC 不稳 + 单 exe 分发原则 + 早期阶段品控需要。

> **对比**：cook 模块仍保留 `templates/cook.toml` 用户外挂层（[tools/cook/cook.go](tools/cook/cook.go) 里调 `MergeUserOverrides`）。vision 包的 `SourceUser` / `MergeUserOverrides` API 也保留着，fish 这边主动选择不调而已。

**耐力条 ROI** 单独走 `constants.go` 里的 `roiFishingBars []BarROI`，不在 templates.toml 里——因为它不是模板匹配，而是颜色分析的固定 ROI。`Detector.PickBarROI(clientW, clientH)` 也是**精确匹配**：找不到当前分辨率就 `fmt.Fprintf(os.Stderr, ...FATAL)` + 返回 `(0,0,0,0)`，调用方靠 `if w<=0 || h<=0 return` 守卫跳过本次检测。

加新分辨率不外挂：截图 + 命名 + 丢 `assets/<tool>/` + 在 `assets/templates.toml`
加 `[[<tool>.<slot>]]` 块 + （耐力条还要在 `roiFishingBars` 追加一条 `BarROI`）+ 重编译。

CI 阶段 [tools/fish/config_test.go](tools/fish/config_test.go) 验证 templates.toml
里所有 file 都真的 embed、所有 `requiredSlots` 都有候选。

---

## 6. 耐力条分析（[tools/fish/bar.go](tools/fish/bar.go)）

游标是黄色长条，目标区是绿色横条。颜色阈值：

```go
// 黄色游标
r >= 230 && g >= 220 && b <= 190
// 绿色目标
g >= 200 && b >= 175 && g > r
```

**不用 HSV** 因为 RGB 阈值在异环这套配色下区分度足够、且省一次空间转换。`rgbToHSV` 函数留着但未使用（曾试过、效果差不多）。

### 6.1 bandHalf 的分辨率坑

```go
bandHalf := max(int(float64(h)*0.30), int(float64(cursorH)*0.85))
```

之前是 `max(6, ...)` 固定 6 像素。在 720p 下 ROI 高 ~16px，bandHalf=6 占 75%，把 bar 上下边框拉进绿色检测区 → TargetW 假阳性 → deadzone 爆炸 → 控制器躺平。

**所有 buffer / radius / 阈值都必须按比例算**，固定像素值在多分辨率下相对意义完全不同。

### 6.2 控制器是 bang-bang

[control.go](tools/fish/control.go) 三态：err > deadzone 按 D，err < -deadzone 按 A，否则松键。dir 没变就不重发 KeyDown/KeyUp（避免 PostMessage spam）。

`deadzonePx = max(2, TargetW * deadzoneRatio)`，`deadzoneRatio = 0.08`（包级常量）。

> **`fishingControl` 现在只有 `controlDir` 一个字段**。原本的 `missingStart` 时间戳已剥到 `machine.fishingBarMissingStart`——missing 计时是状态机时序逻辑，不属控制器关心的事。

---

## 7. 多分辨率（推荐 1080p）

ROI 全部用百分比（`ROI{X,Y,W,H float64}`），主流程会乘以 `clientW/H` 转像素。**模板不再按 `clientH/1080` 等比缩放**——v2.2.0 改为每个分辨率独立 embed 模板（720p 用 720p 模板，1080p 用 1080p 模板，精确匹配），跨分辨率运行时不做缩放。

但**逻辑里的固定像素值仍然是分辨率敏感的**：

- bar.go 的 bandHalf（已修）
- 任何 `max(N, ...)` / `min(N, ...)` 形式的常量
- isCyanish 的采样 radius（[detect.go](tools/fish/detect.go) 里基于 `tpl.W/12` 按比例）

新增检测代码时，所有像素常量都按 `h` 或 `w` 缩放，少用绝对值。

---

## 8. 状态机注意事项

### 8.1 文件拆分

`machine.go` 在 v2.2.0 重构前 873 行单文件，现在拆成三块（合计 873 行依然不变，只是组织清晰）：

- `machine.go` (223 行): `struct machine` + `newMachine` + `run` 主循环 + `sleep` / `shouldExit` / `debugEnabled` / `logState` utils
- `states_fishing.go` (418 行): `handleIdle` / `handleSetup` / `handleWaiting` / `tryHookF` / `handleFishing` / `recordOutcome` / `handleResult` / `pressEscUntilClear` / `enterRecover` / `handleRecovering`
- `states_shop.go` (~70 行): `shopSellFlow` / `buyBaitFlow` / `changeBaitFlow` 三个声明式 flow{} + 3 个 1 行 dispatcher

### 8.2 兜底机制（已大幅瘦身）

v2.2.0 实测后删掉了 3 个"卡死自动恢复"兜底：

- ❌ IDLE 30s 卡死自动 ESC
- ❌ WAITING 75s 探不出就回 IDLE 重抛
- ❌ RECOVERING 8s 超时自动回 IDLE

现在原则更保守：

- **检测到任何已知画面就路由**（IDLE 看到结算就关、WAITING 看到 fish_escape 就记账失败）
- **WAITING/FISHING 超时 → RECOVERING**，再探一次：能路由就路由；探不出且未发 ESC → 发一次 ESC；都不行就 500ms 一轮循环**等用户介入**（在 GUI 点[暂停]或[停止]）
- 商店流程任一步检测失败 → MsgBox + 暂停等用户处理（`missFailPause` 策略）

哲学：**能确认安全的自动做，否则停下来叫人**——比悄悄把游戏洗到错误状态强得多。

各阶段超时阈值见 [fish.md §8](fish.md)；超时常量在 [tools/fish/constants.go](tools/fish/constants.go)。

### 8.3 RECOVERING

线性流程：每帧 `inspectPhaseFrame`；探出 phase 就 `routePhase`；探不出且 `recoveryEscDone=false` 时发一次 ESC、置 `recoveryEscDone=true`；之后探不出 phase 就 500ms 一轮循环等用户介入（**无超时**）。

`enterRecover(reason, pressEsc bool)`：`pressEsc=false` 时初始就把 `recoveryEscDone=true`，跳过 ESC 步骤——用于"已经动过手"的场景（如 F 收线已发但耐力条没出来，再补 ESC 会把游戏 UI 戳飞）。

### 8.4 鱼饵不足 → 自动购买 → 自动更换

NeedBait 检测在两处：handleSetup 点完"开始钓鱼"后单次 + handleIdle 按 F 后 300ms 单次（鱼饵警告 ~0.25s 即时弹出）。命中 → 切 BUYBAIT 状态 → `handleBuyBait` 一行 dispatcher `runFlow(ctx, m, buyBaitFlow)` 跑声明式 Step 序列：

1. `waitLong()` (~2s) 等"鱼饵不足"警告消散
2. `tap("r")` 开鱼饵商店 + `waitLong()`
3. `multiSlotClick("万能鱼饵", (*Detector).BaitInShop)` — **6 个候选槽位扫描**，按阅读顺序末位返回（金币款在前、代币款在后，要代币款）。6 个 `[[fish.bait_product]]` 块共用同一张 PNG，只是 bbox 不同。这是唯一一个不能走 `vision.PickBest` 的 detector
4. `clickIfSeen("拉满", ..., missFailPause)` → `clickIfSeen("购买", ...)` → `clickIfSeen("二次确认", ..., missSkip)`（钱够时不弹，跳过）→ `clickIfSeen("成功 toast", ...)` → `tap("esc")` → `retryIfStillSeen("商店未关", BuyButton, esc again)`
5. **接 CHANGEBAIT**：买回来的鱼饵还要装备才能用，`OnDone: CHANGEBAIT`

CHANGEBAIT (`changeBaitFlow`)：`tap("e")` 进更换 UI → `waitLong()` → `clickIfSeen("更换", ChangeBaitConfirm, missFailPause)` → 1s 自动返回钓鱼界面 → `OnDone: IDLE`。

### 8.5 鱼仓已满 → 一键出售

`shopSellFlow`：`tap("q")` 开商店 → 鱼仓 tab → 一键出售 → 确认 → ESC 退出。任一 `clickIfSeen` 失败走 `missFailPause` 弹窗暂停。和 BUYBAIT/CHANGEBAIT 共用 `delayLong (2s)` 作为 UI 加载等待。

> **删除的 detector**：`BaitShopButton` / `ChangeBaitButton`（曾经用模板找"鱼饵商店"和"更换鱼饵"两个 UI 入口按钮）—— v2.2.0 删，因为这两个按钮在不同界面位置不同、模板维护成本高，改成直接 `tap("r")` / `tap("e")` 快捷键开 UI，可靠性反而上去了。

### 8.6 KeepAlive

主循环每 500ms (`keepAliveEvery`) 调一次 `FakeActivate` 维持 IsActive。某些场景下游戏会自己把 IsActive 翻 false（比如弹出不可见 widget），不维持的话下一个 PostMessage 会丢。

### 8.7 phaseTable 表驱动

`inspectPhaseFrame` 原本是 6 个 if 链，重构后改成 `[]phaseEntry` 表驱动：

```go
var phaseTable = []phaseEntry{
    {PhaseSettleWin,     "WIN",   "result",         (*Detector).Result},
    {PhaseSettleFail,    "FAIL",  "fish_escape",    (*Detector).FishEscape},
    {PhaseWarehouseFull, "FULL",  "warehouse_full", (*Detector).WarehouseFull},
    {PhaseNeedBait,      "BAIT",  "need_bait",      (*Detector).NeedBait},
    {PhaseSetup,         "SETUP", "start_fish",     (*Detector).StartFish},
    {PhaseReady,         "READY", "hook_icon",      (*Detector).HookIconDim},
}
```

`PhaseFighting`（耐力条）单独放最后，因为它需要抓 ROI 子帧而不是全帧检测。加新 detector 入口现在是表里加一行，不用动 6-if 链。

---

## 9. 配置系统（[cmd/yhbox/gui/settings.go](cmd/yhbox/gui/settings.go)）

### 9.1 settings.json — GUI 持久化层

YHBox 已无 CLI flag，也不再用 TOML。唯一的运行时配置文件是 `<exe目录>/settings.json`：

```json
{
  "fish": { "auto_sell": true },
  "cook": { "interval_ms": 1500 },
  "ui":   { "show_log": true }
}
```

只有真正属于"用户偏好"的字段才会出现在这里。延迟 / 置信度 / 超时 / ROI 全部写死成 [tools/fish/constants.go](tools/fish/constants.go) 包级常量（fish 这一边）或 [tools/cook/config.go](tools/cook/config.go) DefaultConfig 默认值（cook 这一边），不暴露给 UI。

**Settings struct（GUI 私有）和 fish.Config / cook.Config 解耦**：

```go
// cmd/yhbox/gui/settings.go
type Settings struct {
    Fish FishSettings `json:"fish"`
    Cook CookSettings `json:"cook"`
    UI   UISettings   `json:"ui"`
}
```

- 启动时 `LoadSettings(path)` 读 JSON（文件缺失 / 解析失败 → `defaultSettings()`，不报错）
- tab 点[开始]时把 settings 字段写入新建的 `fish.Config{AutoSell: ...}` / `cook.Config{Interval: ...}`，bot 不知道 GUI 存在
- 用户改 checkbox / NumberEdit 立刻 `SaveSettings(path, s)` 全量写回 JSON

### 9.2 rationale

- **fish 没有可调参数**：作者实测最优解，暴露给用户当 footgun；调参成本是"提 issue + 作者验"
- **cook 暴露 interval_ms**：用户场景不同（点装备 vs 点对话框），间隔是有意义的偏好
- **AutoSell**：鱼仓满时自动开商店卖 vs 弹窗等手动——这是真正的策略偏好
- **show_log**：纯 UI 偏好

新增 settings 字段流程：

1. 在 `Settings` struct 加 JSON tag 字段
2. 在 `defaultSettings()` 补默认值
3. 对应 tab 文件加控件 + `On*Changed` handler 调 `app.SaveSettings()`
4. tab `onStart` 时把字段写入 `fish.Config` / `cook.Config`

JSON 反序列化用"先填默认值再 unmarshal"模式，老 settings.json 缺新字段时保留默认（见 `LoadSettings` 实现）。

---

## 10. 调试

### 10.1 日志

运行时输出走多 sink 日志器（[pkg/log/log.go](pkg/log/log.go) `Logger`）：

- **文件 sink**：`<exe目录>/logs/yh_<tool>_YYYYMMDD_HHMMSS.log`（tab `onStart` 时新建）
- **GUI sink**：主窗口下方 TextEdit，可通过底部"显示日志"checkbox 隐藏。底层 `LogSink` 是一个 io.Writer：按 `\n` 切行 → 维护 ring buffer（容量 `logRingCapacity=500`）→ `onChange(snapshot)` 通过 `walk.Synchronize` 推到 UI 线程的 TextEdit

`Logger.Log(scope, fmt, args)` 把每条消息广播到所有 sink，sink 之间无序无依赖。`Scope` 还在（SYSTEM/READY/SETUP/CLICK/...），颜色字段保留但不渲染——GUI TextEdit 不解 ANSI，控制台已不存在。

### 10.2 State 详细日志（开发者）

`state.go` 里的 `DebugIDLE`/`DebugSETUP`/.../`DebugALL` 位掩码常量保留备用，但 **runtime 无 mask 驱动**——`machine.debugEnabled()` 当前永远返回 false（[machine.go](tools/fish/machine.go)）。需要看某 state 内部细节时，直接改这个函数（按 `m.state` 加条件 / 直接 return true），重编译。未来若给 GUI 加 debug 开关，把这套 mask 拉回来用即可。

### 10.3 ROI / 模板出问题

最快定位办法：

1. 在 [debug.go](tools/fish/debug.go) `saveDebugFrame(name, img)` 写一帧到 `logs/`（用 `SetDebugDir` 可改路径）
2. 打开 PNG 看：ROI 是否覆盖目标？目标在 ROI 里位置对不对？
3. 模板匹配失败八成是 ROI 错位、模板尺寸 ≥ ROI 尺寸、或当前分辨率没 embed 模板（看日志 WARN / stderr "FATAL 缺 WxH"）

### 10.4 单像素分析

bar.go 这类颜色检测出问题，比盲调阈值快得多的办法：

1. dump 一帧 ROI 到 PNG
2. 写个小脚本扫每一列，统计满足阈值的像素数
3. 直接看哪些列被错判（比如游标右边其实也有黄色装饰像素）

### 10.5 流程级测试

不再有 `-flow <name>` CLI dry-run（v2.4.0 删了 RunFlow / runFlowTable / FlowNames，CLI flag 整段去掉了）。开发者验证单流程改走 Go 测试：

```powershell
go test ./tools/fish/ -run TestXxx -v
```

`tools/fish/flow_test.go` 已有 flow 引擎单测覆盖；状态机级 + 商店流程级用例需要时按 Go 测试惯例添加（mock runctl.Control、mock Detector 等）。

---

## 11. 构建

```powershell
go build -o YHBox.exe ./cmd/yhbox
```

需要 Go 1.21+ 和 Windows（用了 syscall + lxn/win，不能交叉编译）。**不要用 `go build ./...`**——它会把 cmd 子目录的 main package 编到 GOBIN 而不是项目根，也不会取到 `cmd/yhbox/rsrc_windows_*.syso` 资源。

`scripts/` 目录有 UPX 打包脚本，发布时跑一下能从 ~3.5MB 压到 ~1.5MB。

---

## 12. 关键文件速查

| 关心什么 | 看哪 |
| --- | --- |
| 钓鱼完整流程 / 时序 / 状态机 | **[fish.md](fish.md)** |
| 状态机主循环 | [tools/fish/machine.go](tools/fish/machine.go) `run()` |
| 钓鱼 state handler | [tools/fish/states_fishing.go](tools/fish/states_fishing.go) |
| 商店三流程（声明式 flow） | [tools/fish/states_shop.go](tools/fish/states_shop.go) |
| Step / runFlow 引擎 + 原语 | [tools/fish/flow.go](tools/fish/flow.go) |
| 阶段探测 / 路由（表驱动） | [tools/fish/phase.go](tools/fish/phase.go) `phaseTable` / `inspectPhaseFrame` / `routePhase` |
| 检测器（命名模板） | [tools/fish/detect.go](tools/fish/detect.go) + [pkg/vision/named.go](pkg/vision/named.go) |
| 后台输入怎么工作 | [pkg/input/input.go](pkg/input/input.go) 顶部注释 |
| 包级常量（所有时序/置信度/超时） | [tools/fish/constants.go](tools/fish/constants.go) |
| 耐力条 ROI 表 | [tools/fish/constants.go](tools/fish/constants.go) `roiFishingBars` + [tools/fish/detect.go](tools/fish/detect.go) `PickBarROI` |
| 模板匹配实现 | [pkg/vision/template.go](pkg/vision/template.go) `MatchFast` |
| 命名模板 + ROI 匹配 | [pkg/vision/named.go](pkg/vision/named.go) `PickBest` / `MatchTextROI` |
| 耐力条颜色阈值 | [tools/fish/bar.go](tools/fish/bar.go) `findCursor` / `findTarget` |
| GUI 入口 + tab 装配 | [cmd/yhbox/gui/app.go](cmd/yhbox/gui/app.go) `Run()` |
| 用户偏好 JSON | [cmd/yhbox/gui/settings.go](cmd/yhbox/gui/settings.go) `Settings` |
| Pause/Stop 控制接口 | [pkg/runctl/runctl.go](pkg/runctl/runctl.go) `Control` |
| GUI 日志 sink | [cmd/yhbox/gui/logsink.go](cmd/yhbox/gui/logsink.go) `LogSink` |

---

## 13. 不要犯的错

- ❌ Tap 不传 scancode（→ 零星丢键）
- ❌ Click 的 DOWN/UP 间隔太长（→ Slate 当成"按住"不触发 click，cook 实测漏 40%）
- ❌ Click 显式发 PostMessage WM_MOUSEMOVE（→ 跟 OS 自然 MOUSEMOVE 撞车，hover 抖动）
- ❌ 检测到 UI 立刻按键（→ InputAction 还没注册，丢键）
- ❌ 模板尺寸 ≥ ROI 像素尺寸（→ 搜索空间 0，永远不命中）
- ❌ NCC 用了对不上实际渲染的模板（→ conf 跨场景漂；解决：模板从真实游戏截图抠）
- ❌ 检测代码用绝对像素 buffer（→ 720p 爆炸）
- ❌ 忘了 PrintWindow 抓的是窗口区不是客户区（→ 整体偏移）
- ❌ 没有 `PW_RENDERFULLCONTENT` 抓 UE D3D 窗口（→ 黑屏）
- ❌ 给 fish 加可调字段（→ 早期阶段方向错；想加先问"用户改了能带来什么"，多半答案是"什么都不能"；放 [constants.go](tools/fish/constants.go) 包级常量）
- ❌ 新增 GUI 偏好字段时忘了同步 `defaultSettings()` 默认值（→ 老 settings.json 加载后字段是零值，不是合理默认）
- ❌ 用 `Frame` 抓 FISHING 阶段每 30ms 一帧（→ 控制器跟不上）
- ❌ 期待 vision.PickBest 跨分辨率缩放兜底（→ 现在精确匹配，找不到就 nil；补 embed 模板而不是改 PickBest）
- ❌ 期待 fish 用户外挂 `templates/fish.toml`（→ v2.2.0 删了；让用户提交模板给作者）
- ❌ Click 时让用户同时动鼠标（→ `SetCursorPos` 跟用户操作打架，click 落错位置或被丢；cook 实测漏点 ~15%）。键盘路径不受影响。
- ❌ 长内层循环（如 60s 重试）不检 `runctl.Control`（→ 用户在 GUI 按[暂停]/[停止]几十秒无响应；用 `m.shouldExit()` 兜底）
- ❌ 在 fish 状态机里加"30s 没动静自动 ESC"这种激进兜底（→ 历史教训，反而把好状态洗坏；现在原则是停下来 MsgBox 叫人）
- ❌ 在 GUI 回调里直接操作 walk widget（→ 跨线程 panic；bot goroutine 调 UI 一律走 `mw.Synchronize(fn)`，LogSink/control.go 已经包好）
