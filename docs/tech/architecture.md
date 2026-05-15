# YHBox 架构 & 跨 bot 通用机制

面向维护者 / 二次开发者，记**跨多个 bot 共享的底层机制**和**踩过的坑**。
单个 bot 的内部细节在对应 [fish.md](fish.md) / [rhythm.md](rhythm.md) / 等等。

---

## 1. 项目脉络

异环（UE5 + Slate UI 游戏）的桌面自动化工具集。核心约束：

- **真后台**：截图走 `PrintWindow(PW_RENDERFULLCONTENT)` 或 WGC、输入走 `WM_ACTIVATE + PostMessage`，全程不抢前台焦点、不动光标
- **单 exe 分发**：模板 PNG / yaml 配置通过 `embed` 进二进制，运行时只可选读 exe 同目录的 override 配置
- **多分辨率**：720p / 1080p 实测覆盖（精确匹配，无跨分辨率缩放兜底）
- **多 bot 统一接口**：fish / cook / piano / rhythm / battle 都走 `BotService` + `botRun` runner + `runctl.Control` (Pause/Resume/Stop)，main.go 一个 for 循环注册

## 2. 架构平面

```text
main.go             入口：EnsureAdmin → wails3 App + 服务注册
   │
   ├── cmd/yhbox/services/   wails3 服务层（暴露给前端 JS）
   │     ├── app.go              App 协调器
   │     ├── battle_service.go   战斗（热键驱动）
   │     ├── cook_service.go     店长
   │     ├── fish_service.go     钓鱼
   │     ├── piano_service.go    弹琴
   │     ├── rhythm_service.go   音游
   │     ├── settings_service.go 设置 (RFC7386 merge patch)
   │     ├── game_service.go     游戏窗口检测
   │     ├── registry.go         BotMeta 注册表
   │     ├── botrun.go           长跑 bot 的 ctx+state+log 公用 runner
   │     ├── hotkey.go           Win32 RegisterHotKey 封装
   │     └── log.go              LogSink: zerolog → wails3 events
   │
   ├── frontend/         Vue 3 + NuxtUI v4 + Tailwind 4 + vue-i18n
   │     ├── src/views/      每个 bot 一个 view + Settings/Help/About
   │     ├── src/stores/     pinia: settings / game / 各 bot 状态
   │     ├── src/i18n/       zh.ts / en.ts (i18n 口子,en 占位)
   │     └── src/lib/        backend 调用 / 事件订阅
   │
   ├── tools/
   │     ├── fish/         钓鱼状态机 + 检测器 + 耐力条分析 + flow 引擎
   │     ├── cook/         锤子连点
   │     ├── piano/        MIDI 解析 + 选轨 + 自动 transpose
   │     ├── rhythm/       4 轨亮度检测 + 异步按键
   │     └── battle/       一次性切队伍（热键触发）
   │
   ├── pkg/
   │     ├── capture/      截屏后端 (GDI / WGC / Mock 三选一)
   │     ├── input/        WM_ACTIVATE + PostMessage 键鼠
   │     ├── vision/       NCC 模板匹配
   │     ├── botcore/      bot 共享：Manifest + LocaleAlias + RuntimeAssets + LoadYAMLWithOverride
   │     ├── locale/       Zh/En 常量
   │     ├── screenshot/   异步带框 PNG 落盘 (调试用)
   │     ├── runctl/       Pause / Resume / Stop 控制接口
   │     ├── platform/     UAC 提权 + HighResTimer
   │     └── winutil/      游戏窗口枚举 (UnrealWindow class)
   │
   └── native/capture_wgc/  Rust cdylib：Windows Graphics Capture (WGC) 后端
```

## 3. 后台输入子系统（[pkg/input/input.go](../../pkg/input/input.go)）

踩坑最多的地方。

### 3.1 WM_ACTIVATE 翻 IsActive

UE 的 IMC 默认丢弃 `IsActive=false` 窗口的键盘消息。Trick：

```go
SendMessage(hwnd, WM_ACTIVATE, WA_ACTIVE, 0)
```

**只翻窗口内部 IsActive，不改 GetForegroundWindow 返回值**。之后必须 sleep 一段（`ActivateDelay`，默认 30ms）让 Slate 真处理完——SendMessage 同步返回不代表 Slate 处理完，它通常下一 UE tick 才生效。

ActivateDelay 太短 → KEYDOWN 在 IsActive 还是 false 时到达 → 被丢。

**rhythm 特殊**：连续高频按键场景，每次 Tap 后立刻 FakeActivate 会让 IsActive 在按下一帧又翻 false。rhythm 用"启动期 FakeActivate + goroutine 每 500ms keepalive"模式让 IsActive 长期保 true。

### 3.2 keyLParam 必须填 scancode

最隐蔽的坑。`WM_KEYDOWN/UP` 的 `lParam` bits 16-23 是 scancode，UE InputComponent 用 scancode 查 ProfileMap，缺了会零星丢键（首次正常、后面失败，没规律）。

```go
sc := MapVirtualKeyW(vk, MAPVK_VK_TO_VSC) & 0xFF
lp := uintptr(1) | (uintptr(sc) << 16)  // bit 0-15 repeat=1, bit 16-23 scancode
if keyUp {
    lp |= (1 << 30) | (1 << 31)
}
```

历史症状：fish 抛竿稳定，按 F 收线 / 按 ESC 偶发失败。

### 3.3 Click 时序：DOWN/UP 必须紧贴

UE Slate 按钮把 DOWN→长 sleep→UP 当成"按住"，**不触发 click 事件**。必须紧贴发，sleep 放在 UP 之后：

```text
FakeActivate → ActivateDelay(30ms)
  → SetCursorPos(屏幕坐标)
  → CursorSettleDelay(5-30ms)    ← 让 OS 发出自然 MOUSEMOVE
  → PostMessage(WM_LBUTTONDOWN)
  → PostMessage(WM_LBUTTONUP)     ← 紧贴 DOWN
  → sleep ClickHold(30ms)         ← 让游戏 tick 处理 click
  → SetCursorPos(原位置)
```

踩过的具体坑：

1. **加 `WM_MOUSEMOVE` PostMessage 反而让 cook 漏点更多**（30%→70%）。`SetCursorPos` 已让真光标进客户区，OS 会自然给游戏发 MOUSEMOVE，再 PostMessage 造成 hover 抖动
2. **`ClickHold` 100-150ms 把 DOWN-UP 间隔拉长 → cook 命中率只 60%**。改成紧贴 + sleep 在后面，命中率 95%+
3. **`CursorSettleDelay` 80-120ms 是过度防御**，5ms 就够。fish flow 引擎里所有 click 都用 `delayShort=30ms`（含 click hold / activate / cursor settle）

参考实现：[ok-script GenshinInteraction.do_click](https://github.com/ok-oldking/ok-script/blob/master/ok/device/intercation.py) — 同样 UE 游戏，模式一致。

### 3.4 ClickHold vs TapHold 语义不同

| 字段 | ClickHold                        | TapHold                              |
| ---- | -------------------------------- | ------------------------------------ |
| 作用 | DOWN+UP 之后等多久 (after-click) | KEYDOWN→KEYUP 之间间隔 (press hold) |
| 用途 | UI 按钮点击                      | F / ESC / A / D 等键盘               |
| 默认 | 30ms (`delayShort`)            | 150ms (`delayMid`)                 |

**鼠标**：DOWN/UP 紧贴 = "click"，长 hold = "按住"。click_hold 改成"after-click sleep"语义让游戏 tick 处理事件。
**键盘**：UE InputComponent tick 轮询采样，至少一帧（30Hz 下 33ms）才会被采样到，留 150ms 给低帧率/IME 余量。

**例外**：rhythm 音游用 `KeyHoldMs=5ms` 短按。这是 ok-nte 实测出来的，异环音游 D/F/J/K 接受 5ms 短按 + 异步队列不阻塞 detect 主循环。

### 3.5 检测到 UI 立刻按键 = 丢键

动画淡入期间，UI 第一帧出现时游戏的 InputAction 还没在 PlayerController 上注册。立刻 PostMessage 等于按到空气。

fish 抛竿走 `FStreak`（连续 3 帧/200ms 确认）天然延迟，所以稳。上钩之前一开始用 conf 突变那一帧立刻按 F → 经常丢。修法是抄 FStreak：触发后再确认 `hookStreakCount=2` 帧 textOK 才按 F。

**任何"检测到立刻按键"的逻辑，都先想想动画时序**。

## 4. 截屏（[pkg/capture/](../../pkg/capture/)）

三个 backend，启动期 `SetBackend()` 选一个：

| Backend        | 实现                                  | 优势                 | 劣势                                                       |
| -------------- | ------------------------------------- | -------------------- | ---------------------------------------------------------- |
| **gdi**  | `PrintWindow(PW_RENDERFULLCONTENT)` | 兼容性最好，无黄框   | D3D 游戏后台抓帧偶有冻结/黑帧                              |
| **wgc**  | Rust DLL Windows Graphics Capture     | 后台抓帧稳定         | Win10 强制显示黄色边框关不掉（Win11 build ≥20348 才能关） |
| **mock** | 读 `bin/mock-frames/*.png` 序列     | 离线调参，无需开游戏 | 显然 — 只用于调试                                         |

启动期 `Settings.Capture.Method = "auto"` 走 `AutoBackend()` 按 OS build 自动选：Win11 / Server 2022 选 wgc，其它选 gdi。

### PW_RENDERFULLCONTENT

```go
PrintWindow(hwnd, hdc, PW_RENDERFULLCONTENT)  // 0x02
```

UE 走 D3D 后端，**不带 flag 拿到的是黑屏**。这个 flag 让 PrintWindow 等 GPU 渲染完再 blit 到 DC。

### 客户区 vs 窗口区

`PrintWindow` 抓**整个窗口**（含标题栏 / 边框）。UE 渲染、所有 ROI 都按**客户区**算。GDI backend 内部计算 offset：

```go
GetClientRect → clientW, clientH
ClientToScreen(0,0) - winRect.Left/Top → offsetX/Y
```

读位图时 `srcY = y + offsetY`、`srcX = x + offsetX`。忘了 offset，1080p 窗口模式下整体偏 8-30 像素，模板匹配 conf 直接掉到 0.4 以下。

WGC 抓的就是客户区，没这层麻烦。

### Frame vs FrameROI vs NewCapturer

- `Frame(hwnd)` 转整客户区 BGRA→RGBA，1080p ~3MB
- `FrameROI(hwnd, x, y, w, h)` 只转矩形区域
- `NewCapturer(hwnd, rois)` 给高频抓多 ROI 用，复用资源 + 复用 Pix buffer

fish FISHING 阶段 30ms 一帧耐力条 → 必须 `FrameROI`。rhythm 120Hz tick + 4 ROI → 必须 `NewCapturer`。fish detect 多个 UI slot → 用 `Frame`（拿全帧给 vision.PickBest）。

## 5. 模板匹配（[pkg/vision/](../../pkg/vision/)）

`MatchFast`：NCC（CCOEFF_NORMED，与 OpenCV `TM_CCOEFF_NORMED` 等价）。积分图把 patch 的 `∑I` / `∑I²` 降到 O(1)，行级 goroutine 并行，2× 下采样粗找 + 原图 ±4 邻域精修。1080p 下约 8.9ms。

### 模板尺寸必须 < ROI

血泪教训：模板 43px 高 vs ROI 42px 高 → 搜索空间为 0 → 永远 conf=-1。**写检测代码前先验证 ROI 像素尺寸 ≥ 模板尺寸**。

低分辨率（720p）下尤其容易出。当前每个分辨率独立 embed 一份模板（hook_text_720.png / hook_text_1080.png），跑 NCC 时模板尺寸天然就是基准分辨率的尺寸。

### 半透明 UI 的 NCC 教训

历史踩坑：曾认为 NCC 对半透明背景敏感（conf 跨场景在 0.4-0.9 漂）→ 放弃文字检测改用 icon + 颜色校验。

**真相**：根因不是 NCC，是模板和实际渲染像素级对不上（字体抗锯齿、alpha 渲染差异）。模板从真实游戏截图抠出（带精确 bbox）时，NCC 跨场景 conf 都 ≥ 0.99。

新约定：模板元数据走 `templates.toml`（file/resolution/bbox/note），加载用 `vision.BuildNamedTemplate`，`MatchTextROI` 自动按 bbox + `roiPaddingPx (30)` 算 ROI。

### 多分辨率精确匹配，不缩放

`vision.PickBest` 按当前游戏帧 W×H 精确匹配 Resolution。**找不到就返回 nil**（detect 返回 conf=-1），不做"按 aspect/H 最近邻"或 ScaleTemplate 跨分辨率自动缩放——实测跨分辨率 NCC conf 损失明显，宁缺勿滥。

加新分辨率：截图 + 抠 bbox + 丢 `tools/<bot>/configs/<locale>/templates/` + 在该 bot 的 `templates.toml` 加 `[[<bot>.<slot>]]` 块 + 重编译。

## 6. 配置系统

每个 bot 自己的视觉模板 + 阈值放在：

```text
tools/<bot>/configs/<locale>/
  ├── manifest.yaml      该 locale 是否实装 + 可选 locale_alias 引用别的 locale
  ├── config.yaml        非视觉参数（fish bar_rois / rhythm ROIs）
  ├── templates.toml     视觉模板元数据
  └── templates/         模板 PNG 文件
```

跨 bot 共享的设置（locale / capture backend / 日志开关 / 战斗热键等）走 `bin/settings.json`，由 wails3 `SettingsService` 用 RFC7386 merge patch 维护。

### locale_alias

某些 bot 跨 locale 视觉无差异（如 rhythm 纯像素，不依赖文字 UI）。在 `configs/en/manifest.yaml` 写 `locale_alias: zh` 即可让 en 复用 zh 的配置，不必另起一份。详见 [development.md](development.md)。

### bot.LoadConfig 4-helper 拆分

```
LoadConfig(loc)
  ├─ loadManifest(loc)     → 看 manifest.implemented / 跟随 locale_alias
  ├─ loadConfig(loc)       → 反序列化 yaml + exe-dir override merge
  ├─ loadTemplates(loc)    → 加载 templates.toml + 模板 PNG fs.FS
  └─ Validate()            → fail-fast 检查
```

未实装 locale 返回 `ErrLocaleNotImplemented`，main.go 把它当成"bot unavailable"info 日志，不当错误。

## 7. wails3 / 前端 / 国际化

- **wails3 alpha 91**：bot service 通过 `application.NewService(svc)` 注册暴露给前端。frontend bindings 由 `wails3 generate` 自动生成进 `frontend/bindings/`
- **frameless window**：`Window.Frameless: true`，前端自己画 title bar（[AppTitleBar.vue](../../frontend/src/components/AppTitleBar.vue)），用 `--wails-draggable: drag` CSS attr 标可拖区
- **vue-i18n@9**：消息走 ts module 不是 yaml（yaml-plugin 输出的 AST 跟 runtime 不兼容会 SyntaxError，已踩过坑）
- **NuxtUI v4 dark theme**：`html.dark + #app.isolate.dark` 硬编码；toast 也 slots hardcode 走 zinc/emerald 避免 portal 里 fallback light-mode

## 8. 不要犯的错

- ❌ Tap 不传 scancode（→ 零星丢键）
- ❌ Click 的 DOWN/UP 间隔太长（→ Slate 当"按住"不触发 click，cook 漏 40%）
- ❌ Click 显式发 PostMessage WM_MOUSEMOVE（→ 跟 OS 自然 MOUSEMOVE 撞，hover 抖动）
- ❌ 检测到 UI 立刻按键（→ InputAction 还没注册，丢键）
- ❌ 模板尺寸 ≥ ROI 像素尺寸（→ 搜索空间 0，永远不命中）
- ❌ NCC 用对不上实际渲染的模板（→ conf 跨场景漂；解决：从真实游戏抠模板）
- ❌ 检测代码用绝对像素 buffer（→ 720p 爆炸；所有 buffer/radius/阈值按比例算）
- ❌ 忘了 PrintWindow 抓窗口区不是客户区（→ 整体偏移）
- ❌ 没有 `PW_RENDERFULLCONTENT` 抓 UE D3D 窗口（→ 黑屏）
- ❌ Click 时让用户同时动鼠标（→ `SetCursorPos` 跟用户操作打架，cook 漏 ~15%）。键盘路径不受影响
- ❌ 长内层循环不检 `runctl.Control`（→ 用户按[暂停]几十秒无响应；用 `m.shouldExit()` 兜底）
- ❌ 期待 vision.PickBest 跨分辨率缩放兜底（→ 精确匹配；补 embed 模板，不改 PickBest）
- ❌ 在 GUI 回调里直接操作 wails 跨线程状态（→ panic；bot goroutine 调 UI 一律走 service emit event）
- ❌ rhythm 主循环里同步 KeyDown→sleep→KeyUp（→ 阻塞 detect 漏其它轨音符；用异步 pressWorker queue）
- ❌ rhythm 用 HSV 颜色检测（→ 边界敏感 95%；用亮度 dark_ratio，见 [rhythm.md](rhythm.md)）
