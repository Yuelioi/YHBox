# 钓鱼模块完整流程

面向二次开发者 / 想理解逻辑的高级用户。游戏端时序数据来自实测。

---

## 1. 游戏侧时序（基线）

| 阶段 | 典型时长 | 极值 | 备注 |
| --- | --- | --- | --- |
| 抛竿动画 | 1s | — | 按 F 后到能检测上钩为止 |
| 等待上钩 | 5s | 0~10s | 总会有鱼咬钩 |
| **上钩文字提示** | **1s** | — | "鱼上钩了，快点击按钮上鱼!" 一闪即逝 |
| 漏检后静默 | 60s | — | 上钩文字消失到"鱼儿溜走了"出现 |
| 溜鱼 | 5s | 5~33s | 垃圾鱼 5s / 好鱼 8s / 再好 13s / 牛鱼 30s（1% 概率） |
| 结算 | 2s | 1~7s | 牛鱼有过场动画 3-5s |
| 鱼饵不足提示 | 0.25s 内 | — | 按 F 后立即弹出（如果没鱼饵） |

**单轮总时长**：
- 成功路径：cast 1s + wait 11s + hook 1s + fight 33s + settle 7s ≈ **53s**
- 漏检路径：cast 1s + wait 11s + 静默 60s + 处理 fish_escape 5s ≈ **77s**

---

## 2. 完整钓鱼流程（用户视角）

```
┌─────────────────────────────────────────────────────────┐
│                                                         │
│  T=0                                                    │
│   │                                                     │
│   │  在钓鱼准备界面                                     │
│   │  ↓ click "开始钓鱼" 按钮                            │
│   │                                                     │
│   ▼                                                     │
│  钓鱼点提示出现（角色站到水边，右下角鱼钩 icon 显示）   │
│   │                                                     │
│   │  ↓ 按 F 抛竿                                        │
│   │                                                     │
│   ▼                                                     │
│  T=0.3s 检测：                                          │
│   ┌── 鱼饵不足 → 进 BUYBAIT 自动购买 → CHANGEBAIT 装备  │
│   ├── 鱼仓已满 → 进 SHOPSELL 一键出售                   │
│   └── 回到准备界面 → SETUP 重新点开始                   │
│                                                         │
│  T=0~1s  抛竿动画                                       │
│   │                                                     │
│   ▼                                                     │
│  T=1~11s  等待鱼咬钩（典型 5s）                         │
│   │                                                     │
│   ▼                                                     │
│  T=2~12s  ★ "鱼上钩了..." 文字闪现 1s ★                 │
│   ├─→ 1s 内按 F：进入溜鱼 ✓                             │
│   │                                                     │
│   └─→ 没按 F：文字消失                                  │
│         │                                               │
│         ▼                                               │
│        T+60s  "鱼儿溜走了" 出现                         │
│         │                                               │
│         ▼                                               │
│        按 F → 重新进入"钓鱼点提示"循环                  │
│                                                         │
│  溜鱼（按 F 之后）                                      │
│   │  控制 A/D 让黄光标追绿目标区域                      │
│   │  鱼挣脱失败：耐力条消失 → 进结算                    │
│   │  鱼挣脱成功："鱼儿溜走了" → 失败                    │
│   │                                                     │
│   ▼                                                     │
│  结算界面                                               │
│   │  "点击空白区域关闭" 文字提示                        │
│   │  ↓ 按 ESC                                           │
│   │                                                     │
│   ▼                                                     │
│  回到钓鱼点提示，下一轮                                 │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

---

## 3. 程序状态机

```
                ┌──────────────────────────────────┐
                │              IDLE                │
                │                                  │
                │  扫描全帧：                      │
                │  ├ StartFish text → SETUP        │
                │  ├ HookIconDim    → 抛竿流程     │
                │  └ Result text    → ESC + IDLE   │
                │                                  │
                │  无超时兜底——只观察、不强制干预  │
                └────┬─────────────┬───────────────┘
                     │             │
              start_fish     hook_icon dim
                     │             │
                     ▼             │
              ┌──────────┐         │
              │  SETUP   │         │
              │ click 按钮│        │
              │ 检 NeedBait → BUYBAIT │
              │   → IDLE │         │
              └──────────┘         │
                                   ▼
                            ┌────────────────────────┐
                            │ 按 F 抛竿               │
                            │ 等 300ms 单次检：       │
                            │ ├ NeedBait    → BUYBAIT │
                            │ ├ WarehouseFull→SHOPSELL│
                            │ ├ StartFish   → SETUP   │
                            │ └ 否则 cast 1s          │
                            └────┬───────────────────┘
                                 │
                                 ▼
                ┌────────────────────────────────────────┐
                │              WAITING                   │
                │                                        │
                │  每 250ms 检测 hook text：             │
                │  ├ 命中 ≥2 帧 → tryHookF (60s) → FISHING│
                │  └ 75s 超时 → inspectPhase             │
                │      ├ 能路由 → 对应状态               │
                │      └ 探不出 → RECOVERING（无超时）   │
                │                                        │
                │  inspectPhase 表驱动，优先级：         │
                │  WIN > FAIL > FULL > BAIT              │
                │      > SETUP > READY > FIGHT(bar)      │
                └─────────────────┬──────────────────────┘
                                  │
                                  ▼
                ┌────────────────────────────────────────┐
                │              FISHING                   │
                │                                        │
                │  每 30ms 抓耐力条 ROI                  │
                │  bar 可见：                            │
                │   └ chooseDirection(err, deadzone)     │
                │   └ applyDirection (A/D 长按)          │
                │                                        │
                │  bar 消失（前 4s 不判定）：            │
                │   ├ 检 FishEscape → 失败 → IDLE        │
                │   ├ 检 Result    → 成功 → ESC → IDLE   │
                │   └ 10s 仍无信号 → RECOVERING          │
                │                                        │
                │  40s 总超时 → RECOVERING               │
                └─────────────────┬──────────────────────┘
                                  │
                       超时 / 异常 │
                                  ▼
                ┌────────────────────────────────────────┐
                │            RECOVERING                  │
                │                                        │
                │  每帧探+路由 (inspectPhaseFrame)       │
                │  探不出 + 没发过 ESC → 发 1 次 ESC     │
                │  探不出 + ESC 已发 → 500ms 循环等待    │
                │  无超时；用户按 ESC/空格 介入          │
                └────────────────────────────────────────┘
```

外加三个独立分支状态（不参与上面的钓鱼主循环）：

```
SHOPSELL    ── states_shop.go shopSellFlow   按 Q → 鱼仓 tab → 一键出售 → ESC → IDLE
BUYBAIT     ── states_shop.go buyBaitFlow    按 R → 多槽位扫万能鱼饵 → 拉满 → 购买 → CHANGEBAIT
CHANGEBAIT  ── states_shop.go changeBaitFlow 按 E → 点更换 → IDLE
```

三者用同一套 Step + flow 抽象（见 §6），不再各写一份命令式 handler。

---

## 4. 关键时序对应表

| 程序点 | 时间 | 实现位置 |
| --- | --- | --- |
| 按 F 抛竿 | T=0 | [states_fishing.go](tools/fish/states_fishing.go) `handleIdle` 鱼钩 icon 命中分支 |
| 鱼饵 / 鱼仓 / SETUP 兜底检测 | T=0.3s | 同上，单次扫 NeedBait + WarehouseFull + StartFish |
| Cast animation 完成 | T=1s | `delayLong = 2s`，减去 baitProbeDelay(300ms) ≈ 1.7s 剩余 |
| 进 WAITING | T≈2s | 同 |
| 开始检上钩文字 | T+0.5s | `minIconLatency = 500ms`（从 WAITING 进入算起） |
| 上钩文字命中确认 | 连续 ≥2 帧 × 250ms = 500ms | `hookStreakCount=2`、`hookStreakWindow=100ms` |
| F 收线闭环重试 | 每 2s 一次，最多 30 次（60s 预算）| `hookFMaxRetries=30`、`hookFRetryDelay=2s` |
| WAITING 兜底超时 | T=75s（从 WAITING 进入）| `baitWarningTimeout=75s` |
| 溜鱼超时 | T=40s（从 FISHING 进入）| `fishingTimeout=40s` |
| 耐力条消失判定缓冲 | 4s | states_fishing.go `handleFishing` 硬编码 |
| 结算检测频率 | 500ms 一次 | 同上 |
| 耐力条消失后等结算超时 | 10s | `barMissingTimeout=10s` |
| RESULT 状态超时 | 9s | `resultDetectTimeout=9s`（基本不进 RESULT，FISHING 直接处理） |
| RECOVERING | 无超时 | 探不出则 500ms 一轮循环等用户介入 |

上面这些常量全部写死在 [constants.go](tools/fish/constants.go)，**不再通过 config.toml 暴露**——早期阶段无用户，实测值即最优；改值需重编译。

---

## 5. 检测模板速查

模板元数据在 [assets/templates.toml](assets/templates.toml)（embed 进 exe），文件名只是引用 key：

| Slot | 用途 | conf 阈值 | 当前模板 |
| --- | --- | --- | --- |
| `hook_icon` | IDLE 找钓鱼点 | 0.85 | hook_icon_1080.png / 720.png |
| `hook_text` | WAITING 检上钩 | 0.85 | hook_text_1080.png / 720.png |
| `start_fish` | "开始钓鱼" 按钮 | 0.75 | start_fish_1080.png / 720.png |
| `fish_escape` | "鱼儿溜走了" | 0.75 | fish_escape_1080.png / 720.png |
| `need_bait` | "需要装备鱼饵..." | 0.75 | need_bait_1080.png / 720.png |
| `result` | "点击空白区域关闭" | 0.75 | result_1080.png / 720.png |
| `warehouse_full` | "鱼袋已满" | 0.75 | warehouse_full_720.png（缺 1080p） |
| `shop_bag_tab` / `shop_sell_all` / `shop_confirm_sell` | SHOPSELL 流程 | 0.75 | 720p only |
| `bait_product` | BUYBAIT 6 槽位扫描 | 0.75 | bait_product_{720,1080}.png × 6 个 bbox |
| `buy_max` / `buy_button` / `buy_confirm` / `buy_success` | BUYBAIT 流程 | 0.75 | 720p + 1080p |
| `change_bait_confirm` | CHANGEBAIT 流程 | 0.75 | 720p + 1080p |

阈值分档：`hook_*` 用 `confHigh=0.85`（误检代价大），其余用 `confNormal=0.75`，bar 颜色分析单独用 `confBar=0.50`。阈值见 [constants.go](tools/fish/constants.go)，slot 对应表见 [detect.go](tools/fish/detect.go) `slotConfTier`。

**精确分辨率匹配**：`Detector` 启动时读 templates.toml → 加载所有 PNG → 按 `slot` 分组；运行时 `vision.PickBest` 按当前游戏帧 W×H **精确匹配** Resolution，找不到就返回 nil（不再做"按 aspect/H 最近邻 + ScaleTemplate 跨分辨率"兜底——跨分辨率缩放 conf 损失明显，早期阶段宁缺勿滥）。

匹配实测在 day/night 跨场景 conf 都 ≥ 0.99（NCC + ROI 限定 + bbox 元数据）。

> **已知 limitation**：1080p 当前缺 `warehouse_full` / `shop_bag_tab` / `shop_sell_all` / `shop_confirm_sell` 四个 slot 的模板（只有 720p），1080p 下走到对应 flow 时 detector 会 PickBest=nil → 各 step 报"未识别"弹窗暂停。补 1080p 模板需用户提交截图给作者添加并重编译。

---

## 6. 关键代码路径

钓鱼相关 state handler 在 [states_fishing.go](tools/fish/states_fishing.go)；商店三个 flow 在 [states_shop.go](tools/fish/states_shop.go)；状态机主循环 + utils 在 [machine.go](tools/fish/machine.go)（仅 223 行）。

### handleIdle 抛竿
[states_fishing.go](tools/fish/states_fishing.go)
```
1. m.det.StartFish(frame)        → SETUP（看到准备界面）
2. m.det.HookIconDim(frame)      → 抛竿
   - input.Tap("f")
   - sleep 300ms (baitProbeDelay)
   - 单次检 NeedBait        → BUYBAIT
   - 单次检 WarehouseFull   → SHOPSELL
   - 单次检 StartFish       → SETUP
   - sleep 剩余 cast animation (delayLong - 300ms)
   - state = WAITING
3. m.det.Result(frame)           → pressEscUntilClear（清残留）
4. 都没有：原地继续扫描（不再有 IDLE 卡死 30s 兜底）
```

### handleWaiting 等上钩
[states_fishing.go](tools/fish/states_fishing.go)
```
elapsed = now - waitingStart
if elapsed >= minIconLatency (500ms):
    抓全帧 → m.det.HookText(frame)
    if textOK:
        hookStreak++
        if streak >= 2 && elapsed_since_streak_start >= 100ms:
            tryHookF(ctx)              # 30 × 2s = 60s 预算
                每按一次 → 300ms 轮询耐力条 → 见 bar 即 FISHING
            返回 false → enterRecover(pressEsc=false)
    else:
        hookStreak = 0
    检 StartFish → SETUP（误进 WAITING）

if elapsed > baitWarningTimeout (75s):
    inspectPhase → 能路由就路由 / 否则 enterRecover(pressEsc=false)
    （不再静默回 IDLE 重抛）
```

### handleFishing 溜鱼
[states_fishing.go](tools/fish/states_fishing.go)
```
elapsed = now - fishingStart
if elapsed > fishingTimeout (40s): RECOVERING

barX,Y,W,H = det.PickBarROI(clientW, clientH)
if w<=0 || h<=0: return    # 缺该分辨率精确 ROI，跳过本次
抓 FrameROI(barX,Y,W,H) → FishingBarDirect → BarResult
if barVisible (cursor + target + conf >= confBar):
    err = target_x - cursor_x
    deadzone = max(2, target_w * deadzoneRatio)
    dir = chooseDirection(err, deadzone)  # -1/0/+1
    fc.applyDirection(dir)                # 不同 dir 才重发 KeyDown/KeyUp
else:
    if elapsed < 4s: return     # 抛竿动画缓冲
    if m.fishingBarMissingStart.IsZero: 标记起点
    每 500ms 检 FishEscape / Result
    if since(fishingBarMissingStart) > barMissingTimeout (10s):
        RECOVERING
```

> **`fishingBarMissingStart` 字段在 machine struct**（不在 `fishingControl`）。`fishingControl` 现在只剩 `controlDir` 一个字段；missing 计时是状态机时序，不属控制器。

### inspectPhaseFrame 画面探测
[phase.go](tools/fish/phase.go) — `phaseTable` 表驱动，按数组顺序优先级降序
```
1. Result        → PhaseSettleWin   (WIN)
2. FishEscape    → PhaseSettleFail  (FAIL)
3. WarehouseFull → PhaseWarehouseFull (FULL)
4. NeedBait      → PhaseNeedBait    (BAIT)
5. StartFish     → PhaseSetup       (SETUP)
6. HookIconDim   → PhaseReady       (READY)
7. FishingBar    → PhaseFighting    (FIGHT，特殊：抓 ROI 子帧)
8. 都没有        → PhaseUnknown
```

注：上钩文字（HookText）**不在** phaseTable 里，因为它只显示 1s 左右，到探测时早已消失。

### routePhase 阶段路由
[phase.go](tools/fish/phase.go)
```
PhaseSettleWin     → recordOutcome(true) + pressEscUntilClear + IDLE
PhaseSettleFail    → recordOutcome(false) + sleep delayLong + IDLE
PhaseSetup         → SETUP
PhaseWarehouseFull → ReleaseAll + SHOPSELL
PhaseNeedBait      → ReleaseAll + BUYBAIT
PhaseReady         → ReleaseAll + IDLE（重抛）
PhaseFighting      → fc.reset() + FISHING
PhaseUnknown       → 返回 false（调用方决定下一步）
```

### Step + flow 抽象（SHOPSELL / BUYBAIT / CHANGEBAIT）
[flow.go](tools/fish/flow.go) + [states_shop.go](tools/fish/states_shop.go)

三个商店流程原本是 ~200 行命令式 handler（抓帧 → 检测 → 点击 → 等 → 抓帧 → 检测 → ...），重构后改成**声明式 Step 序列**：

```go
var shopSellFlow = flow{
    Name: "商店出售",
    Steps: []Step{
        logf("鱼仓已满 → 按 Q 开商店"),
        tap("q"),
        waitLong(),
        clickIfSeen("鱼仓 tab",   (*Detector).ShopBagTab,    missFailPause),
        waitLong(),
        clickIfSeen("一键出售",   (*Detector).ShopSellAll,   missFailPause),
        waitLong(),
        clickIfSeen("确认出售",   (*Detector).ShopConfirmSell, missFailPause),
        waitLong(),
        tap("esc"),
        waitDur(800*time.Millisecond),
    },
    OnDone: IDLE,
}
```

`machine.handleShopSell / handleBuyBait / handleChangeBait` 各自只剩一行 dispatcher `runFlow(ctx, m, xxxFlow)`。

**原语**（[flow.go](tools/fish/flow.go)）：
- `tap(key)` — `input.Tap` 一个键
- `waitDur(d)` / `waitLong()` — 等指定时长 / `delayLong (2s)`，ctx 取消立刻返 `errFlowAbort`
- `logf(fmt, args...)` — 写日志（test 模式 m.log=nil 时静默）
- `clickIfSeen(label, detectFn, MissPolicy)` — 抓帧 → detect → 命中点击；未命中按策略处理
- `multiSlotClick(label, detectFn)` — 语义 alias，强调"这是多槽位扫描"
- `retryIfStillSeen(label, detectFn, ...then)` — 检到 X 还在就跑 then 序列（ESC 再补一刀）

**MissPolicy**（[flow.go](tools/fish/flow.go)）：
- `missFailPause`（默认）— MsgBox + 暂停等用户介入（真异常）
- `missSkip` — 跳过本步骤，继续下一步（条件分支，如二次确认弹窗钱够时不出现）
- `missStop` — 终止流程（不弹窗）

新加流程的工作量基本是"写一组 Step 字面量 + 加个 flow 变量 + machine handler 一行 dispatcher"。`runFlow` 在 ctx 取消或用户按 ESC/空格时整体中止；无论中止还是跑完都把 `m.state` 切到 `OnDone`，统一收口。

---

## 7. 输入子系统

### Tap (按 F)

[pkg/input/input.go](pkg/input/input.go) `Tap()`:

```text
1. SendMessage(WM_ACTIVATE, WA_ACTIVE)   # 翻 IsActive=true，不抢前台
2. sleep ActivateDelay (delayShort=30ms) # 等 Slate tick 真正生效
3. PostMessage(WM_KEYDOWN, vk, lParam)   # lParam 必须含 scancode（MapVirtualKeyW）
4. sleep TapHold (delayMid=150ms)        # UE InputComponent 必须采到 ≥1 帧
5. PostMessage(WM_KEYUP, vk, lParam)
```

### Click (按"开始钓鱼"按钮)

[pkg/input/input.go](pkg/input/input.go) `Click()`:

```text
1. SendMessage(WM_ACTIVATE, WA_ACTIVE)
2. sleep ActivateDelay (delayShort=30ms)
3. SetCursorPos(屏幕坐标)                 # 真光标进客户区，OS 自动发 MOUSEMOVE
4. sleep CursorSettleDelay (delayShort)   # 让 OS MOUSEMOVE 到达 Slate
5. PostMessage(WM_LBUTTONDOWN)
6. PostMessage(WM_LBUTTONUP)              # 紧贴 DOWN 发，否则 Slate 当成"按住"
7. sleep ClickHold (delayShort=30ms)      # 让游戏 tick 处理 click 事件
8. SetCursorPos(原位置)                   # 还原，用户感知是光标闪一下回来
```

**关键时序差异**：键盘 hold 是 down→up 间隔（150ms 让 InputComponent 采到），鼠标 hold 是 up 之后的等待（DOWN/UP 必须紧贴才被 Slate 识别为 click）。详见 [INTERNALS.md §3.3-3.4](INTERNALS.md)。

---

## 8. 兜底机制设计哲学

整套设计放弃"每个 state 都加超时自动恢复"思路——v2.2.0 实测后删掉了 3 个兜底：

- ❌ IDLE 30s 卡死自动 ESC（误触概率比卡死还高）
- ❌ WAITING 75s 探不出回 IDLE 强行重抛（漏检鱼时容易掩盖问题）
- ❌ RECOVERING 8s 超时回 IDLE（用户来不及介入就被自动洗回钓鱼循环）

现状：

| 阶段 | 超时 | 触发后行为 |
| --- | --- | --- |
| IDLE | 无 | 持续扫描，等待 SETUP / 钓鱼点 icon |
| SETUP | 无 | 单次：找按钮 → click → IDLE；找不到也回 IDLE |
| WAITING | 75s | inspectPhase 能路由就路由；探不出 → RECOVERING |
| FISHING | 40s 总 / 10s 失耐力条 | RECOVERING |
| RECOVERING | 无 | 每帧探+路由；探不到且 ESC 已发 → 500ms 一轮循环等用户介入 |
| SHOPSELL / BUYBAIT / CHANGEBAIT | 每 step 无 timeout | 命中 → 进行；missFailPause → MsgBox 等用户处理 |

**双层兜底** for 上钩漏检（保留）：
1. **第一层** — hook text 命中后：F 收线闭环 60s 预算（足够覆盖 60s 静默窗口）
2. **第二层** — hook text 漏检：等 fish_escape 文字出现（T+71s）→ 正常记账失败

设计原则：**自动化能确认安全的就自动做，否则停下来叫人**——MsgBox 比悄无声息地把游戏洗到错误状态强得多。

---

## 9. 模板系统（assets/templates.toml）

模板元数据集中在 [assets/templates.toml](assets/templates.toml) 里，
TOML grouped 格式，PNG 文件路径相对 `assets/`。

**templates.toml 一项的结构**：

```toml
[[fish.hook_text]]                  # [[<工具>.<slot>]]，slot 自动推断
file = "fish/hook_text_720.png"    # 路径相对 assets/
resolution = [1280, 720]           # [W, H] 截图原始分辨率
bbox = [519, 165, 772, 186]        # [x1, y1, x2, y2] 模板位置
note = "720p 原生（避免 1080→720 缩放损失）"
```

**字段说明**：

- `file`：PNG 在 assets/ 下的路径
- `resolution`：截图原始分辨率（必须精确匹配运行时帧尺寸，没有缩放兜底）
- `bbox`：模板在 resolution 坐标系下的位置，runtime 跑 `MatchTextROI` 时直接用，不做尺寸缩放
- `note`：可选，给后人/自己看

**Detector 行为**：

- 启动时读 templates.toml → 按 `[tool][slot]` 分组 → 加载所有 PNG（全部走 embed）
- 检测时 `vision.PickBest` 按当前游戏帧 W×H 精确匹配 Resolution；**找不到就返回 nil**，本次 detect 返回 conf=-1
- 挑中的模板用 `MatchTextROI` 在 bbox 周围 ±30px (`roiPaddingPx`) 范围跑 NCC

**加新分辨率（唯一路径）**：

1. 截图游戏全屏 → 量出 bbox → 抠图保存为 `assets/<tool>/<slot>_<H>.png`
2. 在 `assets/templates.toml` 加 `[[<tool>.<slot>]]` 块
3. `go build -o YHBox.exe ./cmd/yhbox`

**作者优先策略**：早期阶段不再支持用户外挂 `templates/fish.toml`（v2.2.0 前的机制已删）。需要新分辨率请把截图 + bbox + note 提交给作者，由作者审过加 embed。理由：1）跨分辨率 NCC conf 损失大，模板质量需要把控；2）embed 才能拼成单 exe 分发；3）作为对照，cook 模块仍保留 `templates/cook.toml` 外挂层（迁移参考）。

`tools/fish/config_test.go` 在 CI 阶段验证：embed 模板都符合规范（spec validate 通过 + PNG 文件确实在 embed FS）、所有 `requiredSlots` 都有候选。

---

## 10. 常见排错

| 现象 | 原因 | 排查 |
| --- | --- | --- |
| 上钩漏检 | hook text conf < 0.85 / streak 没确认 | -debug 4 看 conf 数值 |
| F 按了没反应 | 后台输入失败 | 见 [INTERNALS.md §3](INTERNALS.md) PostMessage 踩坑 |
| 溜鱼跑偏 | deadzone 太大 / 太小 | `deadzoneRatio` 包级常量；想改要重编译 |
| WAITING 75s 后进 RECOVERING | inspectPhase 探不出画面 | 按 ESC/空格手动介入；不再自动洗回 IDLE |
| FISHING 立刻进 RECOVERING | 缺当前分辨率耐力条 ROI | 看 stderr "FATAL 缺 WxH 耐力条 ROI"，补 `roiFishingBars` |
| 模板对不上 | 当前分辨率没 embed | 检查 stderr WARN；按 §9 流程补模板提交作者 |
| 1080p 跑 SHOPSELL 卡住 | warehouse/shop 4 个 slot 缺 1080p 模板 | 见 §5 limitation |

---

## 11. 调试

```powershell
.\YHBox.exe -mode fish -debug 4    # WAITING 阶段详细日志 + 截图
.\YHBox.exe -mode fish -debug 8    # FISHING 阶段
.\YHBox.exe -mode fish -debug 12   # 两者
.\YHBox.exe -mode fish -debug 511  # 全部 9 个 state (1+2+4+...+256)
```

debug 模式下 ROI 截图保存到 `debug/` 目录，文件名带 conf 后缀方便排查。

单流程 dry-run 见 `-flow <name>`，详细列表见 [INTERNALS.md §10.4](INTERNALS.md)。
