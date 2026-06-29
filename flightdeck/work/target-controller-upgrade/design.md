# Target / Controller Upgrade Design

## 背景

YHFish 现在已经有大量节点、资产、窗口控制、截图、输入、子图和前端配置逻辑。近期 After Effects `Ctrl+N`、新建合成弹窗、截图取点仍落到 AE 主窗口的问题说明：当前底层把“当前窗口”“截图来源”“输入后端”“坐标系”“节点目标”混在一起了。

这不是单个 SendInput 或 PostMessage 的问题。市面上成熟方案也没有一个万能输入后端：

- MaaFramework 把 Win32、ADB、macOS、Linux、Gamepad、Replay 都做成 Controller，并明确 Win32 下不同程序需要不同输入/截图方式。
- Airtest 用 `Device` 抽象统一 Android、Windows、iOS 的截图、点击、键盘、文本和报告。
- ok-script 用 DeviceManager 管理 Windows、ADB、浏览器目标，并把 capture 与 interaction 分离。
- 商业 RPA 的强项是录制、组件库、日志审计、异常恢复和调度，不是游戏/模拟器/复杂 native UI 的底层输入。

因此本次升级的核心不是换一个按键实现，而是给 YHFish 补一个稳定的多目标自动化内核边界。

参考调研：

- `flightdeck/knowledge/architecture/automation-framework-survey.md`
- `flightdeck/knowledge/architecture/target-controller-upgrade-guide.md`

## 目标

1. 用 `Target` 替代 `Window` 成为自动化对象的核心抽象。
2. 用 `Controller` 表示目标的截图、输入、应用生命周期等能力。
3. 让节点只提交动作请求，不直接依赖 SendInput、PostMessage、ADB、CDP 等具体后端。
4. 显式建模坐标系，避免截图取点和点击目标错位。
5. 为每次节点执行记录可解释 trace，后续支持批量 smoke/report。
6. 保留 Go 作为主运行时，Rust 只下沉 Win32/native hot path。
7. Android 作为一等 Target 接入，而不是旁路系统。

## 非目标

1. 不全量重写成 Rust。
2. 不把 MaaFramework pipeline 替换 YHFish 节点图。
3. 不在第一阶段重写全部节点。
4. 不把 Android 做成绕过 runtime 的独立自动化脚本。
5. 不一次性引入 MaaFramework、Airtest Python runtime 或商业 RPA 模型。
6. 不做旧容器 JSON 的破坏性迁移，除非后续明确进入迁移阶段。

## 设计总览

目标架构：

```text
Graph Node
  -> Action Request
  -> Target Resolver
  -> Controller Capability Probe
  -> Action Router
  -> Screenshot/Input Backend
  -> Trace Report
```

当前阶段先引入边界，不追求一次性完成所有迁移。

## 核心概念

### Target

`Target` 是自动化对象身份，不等同于 Win32 窗口。

目标类型包括：

- `win32-window`：AE 主窗口、AE 新建合成弹窗、记事本。
- `win32-screen`：桌面区域或无明确 HWND 的屏幕目标。
- `android-adb`：手机、MuMu、雷电、蓝叠等 ADB 设备。
- `browser-cdp`：Chrome tab、WebView、浏览器页面。
- `debug-replay`：录制回放目标。
- `mock`：单测和 CI 固定图像目标。

`Window` 未来只是 `TargetKind=win32-window` 的一种引用，不再是所有自动化能力的根。

### Controller

Controller 是某个 Target 的能力提供者。它不应该是一个巨大的万能接口，而应按能力拆分：

- `Screenshotter`
- `PointerInput`
- `KeyboardInput`
- `AppLifecycle`
- 后续可扩展 `BrowserDOM`、`AccessibilityTree`、`ReplayControl`

每个 Controller 必须能返回：

- 当前 Target。
- 能力集合。
- 健康检查结果。

### Action Router

节点提交动作请求，例如 `Click`、`KeyChord`、`Text`、`Screenshot`。Action Router 根据：

- 目标类型。
- Controller 能力。
- 动作策略。
- 允许 fallback。
- 当前容器/节点上下文。

选择实际后端。

例子：

```text
KeyChord(Ctrl+N)
target = AE main window
policy = foreground-required
backend = sendinput
```

```text
Click(OK)
target = AE composition dialog
policy = targeted
backend = postmessage 或 foreground click
```

### CoordinateSpace

裸 `x/y` 不能继续跨层传递。坐标必须带空间：

- `normalized`
- `screen`
- `window-client`
- `capture-frame`
- `android-device`
- `browser-viewport`

坐标转换链必须可追踪。例如：

```text
normalized(0.5, 0.5)
  -> capture-frame(960, 540)
  -> window-client(960, 540)
  -> screen(2100, 870)
```

### Trace

Trace 是自动化系统的核心产物，不是普通日志。

每个节点执行应记录：

- node id / kind / container id。
- target id / kind / resolved ref。
- 执行前截图。
- 识别结果、ROI、bbox、confidence。
- 动作类型、策略、后端、实际参数。
- 坐标转换链。
- fallback 尝试链。
- 执行后截图。
- 成功或失败原因。

后续“节点很多不想一个个验证”的答案就是 smoke graph + trace/report。

## Go / Rust 决策

主运行时继续使用 Go：

- 节点图执行。
- 容器调度。
- 配置、资产、知识库。
- Wails 后端。
- ADB、CDP、HTTP/WebSocket。
- Controller interface 和 Action Router。

Rust 只用于 Win32/native controller hot path：

- WGC / DXGI / PrintWindow / GDI 截图实验。
- SendInput / PostMessage / RawInput 输入实验。
- 图像缓冲和低拷贝路径。

Rust 作为 Controller plugin，不作为全项目重写目标。

## Android 接入策略

第一版 Android 不接 MaaFramework，先做自有 ADB Controller：

1. 设备发现：`adb devices`。
2. 截图：`adb exec-out screencap -p`。
3. 点击：`adb shell input tap x y`。
4. 滑动：`adb shell input swipe ...`。
5. 按键：`adb shell input keyevent`。
6. 启动/停止 app：`monkey -p` / `am force-stop`。

后续再评估：

- minitouch。
- maatouch。
- 模拟器私有通道。
- MaaFramework optional provider。

## 迁移策略

### Phase 1：抽象层

新增 `internal/automation/target` 与 `internal/automation/controller`。

包住现有 Win32 能力，不改变节点行为。现有节点仍通过 runtime service 工作。

对应计划：

- `flightdeck/work/target-controller-upgrade/plan.md`

### Phase 2：Trace 基础

给 controller 调用记录最小 trace：

- target。
- backend。
- action request。
- 坐标输入和输出。
- 成功/失败。

先落文件或内存查看，不做复杂 UI。

### Phase 3：Target Resolver

把当前 Win32WindowTarget 扩展成 Target Resolver：

- 当前活动 target。
- per-node target override。
- 弹窗 target。
- target selector。

After Effects 主窗口到新建合成弹窗是第一验收场景。

### Phase 4：Action Router

点击、按键、文本、截图统一走 Action Router。

容器级 input backend 逐步降级为默认 profile，不再是唯一选择。

### Phase 5：Android ADB POC

新增 `android-adb` target 和最小 controller。

先支持截图、点击、按键、启动 app。

### Phase 6：Smoke / Report

建立最小回归集：

- Notepad。
- Chrome。
- After Effects。
- Android emulator。

每个 controller 改动后都能跑 smoke graph 并生成报告。

## 验收标准

整体升级最终要满足：

1. AE `Ctrl+N` 能稳定打开新建合成弹窗。
2. 后续截图取点绑定到合成弹窗 target，而不是 AE 主窗口。
3. 点击、键盘、截图的 target 在 trace 中一致可查。
4. 不同输入后端的选择和 fallback 有明确原因。
5. Android emulator 能作为 target 被发现、截图、点击。
6. Notepad、Chrome、AE、Android emulator 都有 smoke graph。
7. 用户不需要逐节点人工验证，而是能看报告定位失败。

## 风险

1. 过早迁移全部节点会扩大风险。解决：先 adapter，后逐节点迁移。
2. Controller 接口过大会拖慢 Android/browser 接入。解决：按 capability 拆分。
3. 坐标系如果只写文档不进类型，问题会复发。解决：Point 必须带 `CoordinateSpace`。
4. Trace 如果做成 UI 大工程会拖慢内核升级。解决：先最小结构和文件/内存查看。
5. Rust 过早进入会分散主线。解决：Win32 adapter 稳定后再替换 hot path。

## 推荐执行方式

推荐使用 **Subagent-Driven** 执行 Phase 1：

- 每个 task 改动边界小。
- Target/controller 抽象需要 review 防止接口一次性膨胀。
- 当前工作区已有前序未提交改动，分任务执行更容易隔离。

如果上下文丢失，恢复顺序：

1. 读本 spec。
2. 读 `flightdeck/knowledge/architecture/automation-framework-survey.md`。
3. 读 `flightdeck/knowledge/architecture/target-controller-upgrade-guide.md`。
4. 执行 `flightdeck/work/target-controller-upgrade/plan.md`。
