# Target / Controller 升级指南

SUMMARY: YHFish 从 Window/容器级后端升级到多目标 Target/Controller 自动化内核的实施路线
READ WHEN: 准备破坏性升级底层自动化 / 支持 Android / 改输入或截图后端 / 设计节点兼容迁移 / 修 AE 弹窗、浏览器、模拟器点击差异时
RECHECK WHEN: 改 `pkg/input`、`pkg/capture`、`internal/services/container/runtime`、窗口节点、资产截图取点、Android/Browser controller 时

---

## 决策

YHFish 主运行时继续用 **Go**。不要整体迁移 Rust。

推荐形态：

```text
Go Runtime
  - node graph / container runtime
  - target registry
  - action router
  - trace/report
  - config/storage/Wails bridge
  - ADB/browser controller

Rust Native Controller
  - Win32 screenshot hot path
  - Win32 input hot path
  - WGC/DXGI/PrintWindow/GDI experiments
  - SendInput/PostMessage/RawInput/native fallback experiments
```

Rust 是 native controller 插件层，不是整个产品运行时。

## 为什么不是全量 Rust

- 当前主体是 Go/Wails，节点运行时、服务、配置、资产、调度都已经成形。
- 当前脆弱点主要是边界混乱，不是 Go 性能不足。
- Android ADB、浏览器 CDP、HTTP/WebSocket、文件 IO 用 Go 足够。
- 全量迁移会同时制造业务迁移风险和语言边界风险。
- Rust 的收益集中在 Win32 native、截图、输入、图像缓冲这些 hot path。

## 目标架构

### Target

`Target` 是自动化对象的身份，不等同于 Window。

建议 target kinds：

| Kind | 例子 | 说明 |
|---|---|---|
| `win32-window` | AE 主窗口 / AE 新建合成弹窗 / Notepad | HWND 目标 |
| `win32-screen` | 全屏桌面区域 | 无窗口或多窗口场景 |
| `android-adb` | 手机 / MuMu / 雷电 / 蓝叠 | ADB serial 目标 |
| `browser-cdp` | Chrome tab / WebView | CDP target |
| `debug-replay` | 录制回放 | 自测和回归 |
| `mock` | 固定图片 / 空输入 | 单测和 CI |

每个 target 至少应有：

```go
type Target struct {
    ID          string
    Kind        string
    DisplayName string
    Ref         TargetRef
    Bounds      Rect
    Resolution  Size
    DPI         DPIMeta
    Metadata    map[string]any
}
```

`Window` 未来是 `Target.Ref` 的一种，不是执行系统的根。

### Controller

Controller 是 target 的操作能力提供者。

建议接口按能力拆分，不要一个大接口要求所有后端都实现所有动作：

```go
type Controller interface {
    Target() Target
    Capabilities(ctx context.Context) CapabilitySet
    HealthCheck(ctx context.Context) HealthReport
}

type Screenshotter interface {
    Screenshot(ctx context.Context, req ScreenshotRequest) (Frame, error)
}

type PointerInput interface {
    Click(ctx context.Context, req ClickRequest) error
    Move(ctx context.Context, req MoveRequest) error
    Scroll(ctx context.Context, req ScrollRequest) error
}

type KeyboardInput interface {
    KeyChord(ctx context.Context, req KeyChordRequest) error
    KeyDown(ctx context.Context, req KeyRequest) error
    KeyUp(ctx context.Context, req KeyRequest) error
    Text(ctx context.Context, req TextRequest) error
}

type AppLifecycle interface {
    StartApp(ctx context.Context, req StartAppRequest) error
    StopApp(ctx context.Context, req StopAppRequest) error
}
```

节点只提交 action request，不直接调用 SendInput/PostMessage/ADB。

### Action Router

Action Router 负责把节点动作映射到实际 controller 方法。

输入：

- `ActionKind`：click/key/text/swipe/scroll/screenshot/start-app...
- `TargetID`
- `CoordinateSpace`
- `Policy`：foreground / background / no-steal-focus / best-effort / require-confirmed。
- `Fallbacks`：允许哪些后端降级。

输出：

- 选中的 controller。
- 选中的 backend。
- 坐标转换结果。
- trace detail。

AE 的 `Ctrl+N` 应表达为：

```text
KeyChord(Ctrl+N)
target = AE main window
policy = foreground-required
```

之后切到新建合成弹窗应表达为：

```text
ResolveTarget(selector = title/class/process/owner)
SetCurrentTarget(target = composition dialog)
Capture/Click(target = composition dialog)
```

### CoordinateSpace

裸 `x/y` 必须逐步淘汰。坐标参数需要携带空间：

```go
type Point struct {
    X float64
    Y float64
    Space CoordinateSpace
}

const (
    SpaceNormalized    = "normalized"
    SpaceScreen        = "screen"
    SpaceWindowClient  = "window-client"
    SpaceCaptureFrame  = "capture-frame"
    SpaceAndroidDevice = "android-device"
    SpaceBrowserView   = "browser-viewport"
)
```

坐标转换必须进 trace：

```text
normalized(0.5,0.5)
  -> capture-frame(640,360)
  -> window-client(640,360)
  -> screen(1920,840)
```

这样 AE 弹窗截图取点仍落在主窗口的问题会变成 trace 上一眼可见的 target mismatch。

### Trace / Report

每个节点执行至少记录：

- node id / kind / container id。
- target id / target kind / selector / resolved ref。
- screenshot before / after。
- recognition detail：ROI、box、confidence、template/OCR/model。
- action detail：action kind、backend、policy、raw params。
- coordinate transform chain。
- result：success / validation / runtime error / panic。
- fallback chain：尝试了哪些 backend，为什么失败或降级。

Trace 不是可选调试日志，而是后续批量 smoke 的基础设施。

## Android 接入路线

第一阶段不要接 MaaFramework，先做 YHFish 自己的 ADB Controller。

### Phase ADB-1：最小可用

- 设备发现：`adb devices` + serial。
- 截图：`adb exec-out screencap -p`。
- 点击：`adb shell input tap x y`。
- 滑动：`adb shell input swipe x1 y1 x2 y2 duration`。
- 按键：`adb shell input keyevent`。
- 文本：先支持 ASCII，复杂输入后续再做 IME agent。
- app lifecycle：`monkey -p package` / `am force-stop`。

### Phase ADB-2：模拟器优化

- MuMu / 雷电 / 蓝叠设备发现。
- 读取分辨率、方向、DPI。
- 针对模拟器做截图/输入性能基准。
- 评估 minitouch / maatouch / emulator extras。

### Phase ADB-3：高级能力

- 录制/回放。
- 多点触控。
- IME 文本输入。
- 截图流。
- 失败诊断：offline、unauthorized、resolution mismatch、orientation mismatch。

## Win32 接入路线

现有 Go Win32 能力保留，但外面套 controller。

### Phase WIN-1：包现有能力

- 把当前 `pkg/input`、`pkg/capture`、`pkg/winutil` 包成 `Win32Controller`。
- 所有现有节点仍可走兼容 adapter。
- controller 暴露 capabilities：sendinput、postmessage、foreground、gdi、wgc 等。

### Phase WIN-2：目标一致性

- `ScreenshotRequest.TargetID` 和 `ClickRequest.TargetID` 必须一致，除非 action 明确声明跨 target。
- `SetCurrentTarget` 不再只改窗口栈，还要影响截图、取点、点击、识别。
- 弹窗目标必须能独立 capture。

### Phase WIN-3：Rust native plugin

Rust 只承接 Win32 hot path：

- WGC / DXGI / PrintWindow / GDI 截图统一输出 frame。
- SendInput / PostMessage / RawInput 实验后端。
- 减少 Go syscall 和图像拷贝。

Go 侧只依赖稳定 ABI 或窄 RPC。

## 浏览器路线

浏览器不要伪装成 Win32 窗口点击优先。

优先级：

1. CDP：截图、点击、键盘、JS、DOM query。
2. Accessibility / selector。
3. 最后才是 Win32 鼠标键盘 fallback。

Browser target 应该是 tab/page，不是 Chrome 主窗口。

## 节点迁移策略

不要一次性重写全部节点。

### 兼容层

先提供：

```text
Window -> Target adapter
CurrentWindow -> CurrentTarget adapter
NeedsWindow -> NeedsTarget(win32-window)
```

旧容器 JSON 继续跑，新节点和新 UI 开始使用 Target。

### 新节点

优先新增：

- `GetTarget`
- `SetCurrentTarget`
- `TargetState`
- `TargetScreenshot`
- `TargetClick`
- `TargetKeyChord`
- `WaitTarget`
- `FindTarget`

旧 `WindowState/ClickAt/KeyPress` 后续逐步标记 legacy 或内部改走 Target。

## 实施阶段

### Phase 0：基线文档和红线

- 固化本调研文档和升级指南。
- 列出不变量：Target、Controller、CoordinateSpace、Trace。
- 明确 Go/Rust 分工。

### Phase 1：Controller interface

- 新增 controller 包和能力接口。
- 写 `Win32Controller` 包装现有实现。
- 不改节点行为，只把底层调用套进新接口。

### Phase 2：Trace

- 给每次节点执行生成 action trace。
- 先支持截图前后、target、backend、坐标转换。
- UI 先能打开最近一次 trace，不追求完整报告。

### Phase 3：Target resolver

- 从 `Window` 迁移到 `Target`。
- AE 主窗口/弹窗作为验收样例。
- 修复截图取点、点击、键盘 target 一致性。

### Phase 4：Action router

- 点击、按键、文本、截图统一走 router。
- 输入后端从容器配置迁移到 action policy + target profile。
- 支持 fallback 和失败原因。

### Phase 5：Android ADB POC

- 加 `android-adb` target。
- 最小节点：截图、点击、按键、启动 app。
- 用一个模拟器 smoke graph 验证。

### Phase 6：批量 smoke/report

- 加 Notepad、Chrome、AE、Android emulator 的 smoke case。
- 每次改 controller 都能跑最小回归。
- 报告落文件，UI 可查看失败节点。

## 不要做的事

- 不要把 Android 做成独立旁路，绕过节点 runtime。
- 不要把 Maa pipeline 替换 YHFish graph。
- 不要继续把 `Window` 当所有自动化目标的中心。
- 不要继续让裸 `x/y` 在节点、截图、点击之间自由传。
- 不要把输入后端做成容器级唯一选项。
- 不要全量迁移 Rust。

## 验收标准

这次大升级不是“代码改完”就算完成。至少要满足：

- AE `Ctrl+N` 能打开新建合成弹窗。
- 后续截图取点绑定到弹窗 target，而不是 AE 主窗口。
- 同一套点击/按键节点能明确显示使用了哪个 target 和 backend。
- Notepad、Chrome、AE、Android emulator 至少各有一个 smoke graph。
- 失败报告能解释：找不到 target、截图失败、坐标转换失败、输入后端不支持、动作执行失败。

