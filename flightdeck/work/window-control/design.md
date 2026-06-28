# Window 类型 + 窗口控制节点 — design

SUMMARY: 把窗口做成一等数据类型(仿 Image) + GetWindow/Win32WindowTarget 两个产出节点 + 给 NeedsWindow 节点统一加可选 Window 输入(不连=最近活动窗口, 连了=派发期 per-node 覆盖栈) + 窗口控制节点群; 默认用法零改动。
READ WHEN: 实现/改 internal/nodes/window/* · 派发期窗口覆盖逻辑(dispatch 单节点 wrap) · runtime_context 的 ActiveHWND/WindowHandle/覆盖栈 · WindowService 控制方法 · winutil 窗口解析/控制原语 · Spec NeedsForeground 标志 时。

---

> 行号是「约」提示, **会漂**——实现前以函数/符号名 grep 核实(铁律: 不信行号信源码)。
> 两轮 AI 评审(2026-06-25, ds/gpt/claude)+ 用户裁定均已纳入。第二轮纳入要点见各节内联标注; 三家均判「大坑已平、余为细化」。

## 1. Background / 动机

现状 = **单一粘性活动窗口**: `Win32WindowTarget` 按 title/class/process 解析 → `SetActive` 写 `rt.window` → 所有 `NeedsWindow` 节点经 `rt.ActiveHWND()` 隐式读它。切窗口 = 线上再插 `Win32WindowTarget`(最近跑过的决定当前窗口)。

真实场景: app 有子窗口(对话框), 要分别控制主/子窗口。现状靠 re-target 能跑, 但缺: ① 让某节点显式作用在指定窗口; ② 把窗口当可命名复用对象。参考影刀 `xbot.win32.Window`(获取对象→存变量→`activate/maximize/...`)。本设计取其内核, 但**窗口输入可选、默认走最近活动窗口**。

## 2. Goals / Non-goals

**Goals**: `Window` 一等类型(仿 Image, 不可手填); 两个产出节点 `GetWindow`(解析→Window, **不改活动窗口**)+`Win32WindowTarget`(设活动窗口 + 附带产出 Window); 任一 NeedsWindow 节点**可选** `Window` 输入(不连=当前活动窗口零改动, 连了=派发期覆盖、限本节点); 控制节点 WindowState/MoveResizeWindow/CloseWindow; 收 "Window" palette 类别。

**Non-goals (YAGNI/硬约束)**: 不做真·独占全屏(DirectX); 不推倒活动窗口模型(它仍是默认上下文, Window 输入是显式覆盖); 不做并行/async 节点执行(当前单线程顺序派发); **不写任何兼容 shim**(未发布、现存仅 1 个容器, 结构变更直接改它)。

## 3. 现状锚点 (实现前 grep 核实; 行号会漂)

- 粘性活动窗口: `runtime_context.go` `WindowHandle()`/`ActiveHWND()`/`SetActiveWindow()`(约 :166/:173/:183), **均 windowMu 守护**(有 UI/emit 并发读者, 故用 RWMutex——见 4.3 加锁理由)。`SetActiveWindow`=整体替换 + 清该 hwnd 帧缓存, 不拉前台(拉前台是 sendinput 在 adapter `SetActive` 单独做, `node_services.go` 约 :434)。
- 窗口元数据 + 解析: `winutil.WindowHandle{HWND,Title,Class,ProcessName,PID,ClientW,ClientH}`; `winutil.ResolveWindow`(GetWindow/Win32WindowTarget 共用)——用 EnumWindows + GetWindowText/GetClassName, **对背景窗口有效、不需前台**; 多命中按 **Z-order 取最前**(`window.go` 约 :180, MSDN「前台最上为先」)。
- 只读产出类型范式: `node.Image` + `RegisterType` + 内置类型表(`types.go`)。
- 窗口服务: `node.WindowService`(`services.go` 约 :176) + adapter(`node_services.go` 约 :389) + stub。
- 控制原语: 已有 `ShowWindow`/`SetForegroundWindow`/`IsIconic`(`window.go` 约 :28); 新增 `SetWindowPos`/`Get-SetWindowLongPtr`/`MonitorFromWindow`/`GetMonitorInfo`/`GetWindowPlacement`/`SetWindowPlacement`/`PostMessage`/`IsWindow`。
- 变量值**运行期、不进 workflow JSON**: 容器只存 `VarDecl` 声明(`container/model.go` 约 :19/:23), 值靠运行期 `SetScoped`(`dispatch_v5.go` 约 :165)。→ Window 值 run 内瞬时, 不持久化(同 Image)。
- NeedsWindow 校验: `runner.go containerNeedsWindow`(约 :410) + validator 同判定。
- 注册校验范式(capability invariant, init 期 panic): `registry.go` Register。catalog 读 `s.Inputs`(`catalog.go` 约 :58)。
- 加节点: `knowledge/nodes/add-node.md`; 数据流: `node-data-flow.md`; 类型表: `node-system-reference.md`。

## 4. 核心设计

### 4.1 `Window` 类型

- `node.Window`(放 `types.go`) 镜像 `winutil.WindowHandle`。`RegisterType(TypeSpec{Tag:"Window", GoType:"node.Window", WidgetKind:"preview", Color:"#22d3ee"})`——只读 preview, 不可手填, 只能连线/变量提供(同 Image/List)。FE preview 提示一句「窗口对象由 GetWindow/Win32WindowTarget 产出, 不能直接填标题」。
- helper `Inputs.Window(name) (node.Window, bool)`(仿 `Point()`)。
- **HWND live, 元数据是解析时快照**: 操作走 live HWND; Title/ClientW/H 等会过期, 别当实时(要实时尺寸用 `ctx.Window().ClientSize()`)。
- **生命周期 = 单 run 内瞬时**, 不序列化(见 §3); 「HWND 跨重启失效」不成立(值根本不持久化)。

### 4.2 产出节点: `GetWindow`(新) + `Win32WindowTarget`(增产出)

职责**不重叠, 差别只在改不改活动窗口**; 共用解析:

- **共享选择器**(防漂移, 评审2/gpt9): 抽 `windowSelectorInputs()`(Title/Class/ProcessName/TitleMatch 四个 InputSpec) + `matchSpecFrom(in) winutil.MatchSpec`, GetWindow/Win32WindowTarget 都用。
- **`GetWindow`**(新, exec Runnable, Category Window, **非** NeedsWindow): `In` → `Done`(带 Data `Window`) / `Fail`(带 `Error`/`Code`, **不带 Window**——解析失败无窗口; NotFound 走 Fail 不 panic, 同 Win32WindowTarget)。**只解析不 SetActive**——拿对象不污染活动窗口。背景窗口可解析(ResolveWindow 不需前台, 见 §3)。多命中按 Z-order 取最前(要精确用更严匹配)。
- **`Win32WindowTarget`**(改): `Done` 加 Data `Window`(现 Done 无任何 Data, 纯附加、不冲突; 无「默认捕获」, 用户显式绑才捕获)。仍 `SetActive` 设粘性活动窗口。
- 用法分流: 顺序切窗口→Win32WindowTarget(后续不连输入的节点走当前窗口); 多窗口存变量不污染→GetWindow 绑变量, 各节点按需连 Window 输入。

### 4.3 可选 `Window` 输入 + 派发期 per-node 覆盖

**输入声明 — 显式共享 helper + Register 强校验**(评审2/ds4·gpt4): `node.WindowInputSpec()` 返 `InputSpec{Name:"Window", Type:"Window"}`(可选无 Default), 各 NeedsWindow 节点显式 spread。**Register 时强制不变式**: `spec.NeedsWindow==true ⟹ Inputs 含 Window 输入`, 否则 init 期 panic(同 registry 现有 capability 校验路子)——杜绝「有窗口能力却漏 Window 输入」。`Window` 是 NeedsWindow 节点的保留 pin 名。放弃 Register 动态注入(会让 Spec 不「声明即文档」)。

**覆盖机制 — per-node 覆盖栈 + defer**:
- rt 加 `windowOverride []winutil.WindowHandle`(**栈**, 非单值, 抗嵌套 dispatch; 评审2/gpt1), windowMu 守护。**加锁理由**: `rt.window` 现就是 windowMu RLock 守护(UI/emit 等并发读者), 覆盖栈同锁是一致——**不是**因为节点派发并行(派发仍单线程顺序)。
- `ActiveHWND()`/`WindowHandle()`: 栈非空返**栈顶**, 否则返粘性 `rt.window`。
- 框架跑 NeedsWindow 节点前(Inputs 已 build): 若 `in.Has("Window")`——
  1. 取 Window; **取不到合法值(nil / IsWindow 失败) → 节点 Coded error `WINDOW_INVALID`, 不静默回落活动窗口**(评审2/ds2·gpt5);
  2. push 栈, **`defer` pop**(panic/cancel/早返不漏, 框架有 `runWithRecover`);
  3. **补前台靠 Spec 标志, 非 kind switch**(评审2/gpt6): 新增 `Spec.NeedsForeground`(输入节点 Click/KeyPress/… 置真)。若 `Container.InputBackend=="sendinput"` 且本节点 `NeedsForeground`, 比照 adapter `SetActive` 拉一次 `BringToForeground`+150ms(不在前台 SendInput 会打错窗); 截图/postmessage 不需。
  4. 跑节点(服务经 `ActiveHWND()` 读栈顶); defer pop。
- **粘性 `rt.window` 全程不碰** → 不清它缓存、不污染、无泄漏。
- 不连 `Window` → 整段跳过 → 与现状逐字节一致。Run 同步(ShowWindow/SetWindowPos/截图/点击/Clip 都在 Run 内完成才返回), 无「Run 后仍跑的异步」, defer-after-Run pop 正确。

**NeedsWindow 校验放宽**(因 GetWindow 可不经 Win32WindowTarget 供窗口): 判定取**图级**(全图有无任一 Win32WindowTarget; 非拓扑可达性, 评审2/claude)——仅当存在「无 Window 输入连线」的 NeedsWindow 节点**且**全图无 Win32WindowTarget 时才报缺。残余(连了 Window 但运行期取不到, 如条件分支跳过 GetWindow)由步骤 1 运行期 `WINDOW_INVALID` 兜, 不静默。接受**静态校验能力比以前弱**(评审2/gpt5)。挂载点 `runner.go containerNeedsWindow` + validator 同改。

挂载点(派发单节点 wrap)精确函数 plan 阶段定位(`dispatch_v5.go` 调 `RunNode` 外层)。

### 4.4 窗口控制节点群 (新增, 全 NeedsWindow + 经 4.3 获可选 Window 输入)

| Kind | 操作 | 输入(除 In + Window) | Done 透传 |
|---|---|---|---|
| `WindowState` | `State` 下拉 | State(5 选) | Window(见下) |
| `MoveResizeWindow` | 移动+改尺寸 | X/Y/Width/Height(Number) | Window(见下) |
| `CloseWindow` | 发送关闭请求 | — | 无 |

- **Done 透传 Window = 操作后按 live HWND 重读元数据构造**(评审2/gpt3·claude2): maximize/move 改了 ClientW/H/位置, 透传必须反映新值——不透传旧入参(否则下游拿过期尺寸)。链式 `WindowState→MoveResize→…` 同窗口免反复 GetVar。
- **CloseWindow 不透传 Window**(驳 gpt8): 窗口正被销毁, 传将失效句柄会害下游 `WINDOW_INVALID`; 关后对同窗口再操作本就逻辑错。
- 单 `Done` 出口; 真出错(SetWindowPos 返 0 等)裸冒泡当节点失败, 不给死 Fail。

**WindowState 5 个 State**:

| State | 实现 |
|---|---|
| `maximize` | `ShowWindow(SW_MAXIMIZE)` |
| `minimize` | `ShowWindow(SW_MINIMIZE)` |
| `restore` | `ShowWindow(SW_RESTORE)`(从最大/最小还原) |
| `borderlessFullscreen` | **进入时捕获原状态**(GetWindowPlacement + style, 存 rt per-run saved-state[hwnd], entry 带 **PID/Title/Class**) → 去 `WS_CAPTION\|WS_THICKFRAME` → `MonitorFromWindow`→`GetMonitorInfo` → `SetWindowPos` 铺满该显示器 |
| `restoreBorders` | 从 saved-state 取回——**先校验当前 hwnd 的 PID 匹配**(防 HWND 复用, 评审2/gpt2), 匹配则还原 style + `SetWindowPlacement` 到**进入全屏前**布局并**删 entry**; 不匹配/无记录则退化 `WS_OVERLAPPEDWINDOW`+`SW_RESTORE` |

> **状态交互**(评审2/gpt7): saved-state 在**进入 borderless 时**捕获; `borderless→MoveResize→restoreBorders` 还原到**进入前**布局(忽略全屏期间的 move); restoreBorders **删 entry**; 重新进入 borderless 重新捕获当前状态。saved-state 放 rt(per-run, adapter 管, 同帧缓存归属), winutil 仅出纯原语。

- **`MoveResizeWindow`**: `SetWindowPos(hwnd, 0, X, Y, Width, Height, SWP_NOZORDER)`。坐标 = **虚拟桌面物理像素**(SetWindowPos 原生口径)。⚠ per-monitor 混合 DPI 跨屏需真机验(验证阶段)。
- **`CloseWindow`**: `PostMessage(hwnd, WM_CLOSE, 0, 0)`。**节点描述首句**写明:「发送关闭请求后立即继续, **Done 不代表窗口已关**(可能弹保存框/被拦截); 要确认关掉接 `WaitWindowGone`」。≠ 杀进程。

### 4.5 分层

```
internal/nodes/window/*.go   GetWindow + WindowState/MoveResizeWindow/CloseWindow。控制节点零 Win32, 调 ctx.Window().<op>()。
        ↓
node.WindowService  控制方法(Maximize/Minimize/Restore/BorderlessFullscreen/RestoreBorders/MoveResize/Close);
   ├─ windowAdapter:  经 ActiveHWND()(含 4.3 覆盖栈)拿 hwnd; borderless/restore 用 rt per-run saved-state(带 PID 校验); 调 winutil 纯函数; Done.Window 重读元数据。
   └─ stubWindowService: no-op 返 nil(测试)。
        ↓
pkg/winutil  纯原语(见 §3 列表)。复用已有 user32 LazyDLL。
```

GetWindow/Win32WindowTarget 共用 `winutil.ResolveWindow` + `windowSelectorInputs()`/`matchSpecFrom`。控制节点经 `ctx.Window()`(含覆盖)拿 hwnd, 与截图/点击同语义。

### 4.6 "Window" palette 类别

新类别 `Window`; 现有窗口节点改挂 `Category:"Window"`: `Win32WindowTarget`/`WaitWindow`/`WaitWindowGone`(现 System)+`BringWindowForeground`(现 Input)+新四个(GetWindow+三控制)。文件不挪(只改 Category); 新节点放新 `internal/nodes/window/` 包(blank-import: `main.go`+`dispatch_v5_test.go`)。GROUP_MAP/nodeGroup.* 按 `add-node.md §6`。新 window 包加 `doc.go` 列「全部窗口类别节点位置(含不在本包的)」便于定位(评审2/ds)。**无兼容顾虑**; 影响那唯一容器就直接改它。

## 5. Data flow / 用法

- 常见(单窗口/顺序): 不连 Window 输入 → 全走当前活动窗口。零改动、零 spaghetti。
- 多窗口不污染上下文: `GetWindow` 绑窗口变量 → `GetVar` → 拉进那个节点的 Window 输入。只有作用在「非当前窗口」的节点才连线。
- Window 值进变量 = WindowHandle 元数据(`any` 存取, 运行期瞬时不序列化)。
- 控制链: `GetWindow→WindowState(Done.Window 重读)→MoveResize(Done.Window 重读)→…` 透传刷新后的同窗口。

## 6. Open questions (仅余实现细节/验证, 核心行为全 decided)

1. 派发单节点 wrap 精确挂载函数(`dispatch_v5.go` 内)——plan 定位。
2. `WindowInputSpec()` 的 i18n `input.Window.label` 在多节点的共享方式——plan 定。
3. per-monitor 混合 DPI 下 MoveResize 物理像素跨屏正确性——验证阶段真机覆盖。
4. NeedsWindow 校验放宽的 validator/runner 具体改法(图级判定)——plan 定。

> 两轮评审后核心行为已全 decided(覆盖栈+defer / NeedsForeground 标志 / sendinput 补前台 / hwnd 失效→WINDOW_INVALID / 连了取不到→报错不回落 / 无边框存原布局可还原+PID 防复用 / Done.Window 重读 / CloseWindow=发送请求 / GetWindow 进 v1 / 显式 helper+Register 强校验 / 无兼容)。**再加轮评审已边际递减, 建议进 writing-plans。**

## 7. Testing / 验证

- `pkg/winutil` 纯原语靠真窗口难单测(同 `BringToFront`); 重点测节点层 + 覆盖逻辑。
- **守卫**: `TestAllNeedsWindowNodesHaveWindowInput`(遍历 `node.All()`, NeedsWindow⟹含 Window 输入); Register 不变式亦兜(init panic)。
- 节点层(StubWindowService): WindowState 各 State 解析 + MoveResize 取 x/y/w/h + CloseWindow Done + Done.Window 重读(反映操作后尺寸/位置) + GetWindow 解析/NotFound 走 Done/Fail 且**不改活动窗口**; Spec 一致性守卫。
- 4.3 覆盖(fixture 注两窗口, 仿 `win32windowtarget_dispatch_test.go`): ① 连 Window 的节点作用 B、之后无输入节点回粘性 A(栈 scoping); ② **嵌套 push/pop** 回栈正确; ③ Window 输入失效 → `WINDOW_INVALID`; ④ 连了但取不到合法值 → 报错不回落; ⑤ sendinput + NeedsForeground 覆盖期补前台(校验 BringToForeground 被调)。
- borderless: 进入存 placement、restoreBorders 还原原 rect + **PID 不匹配退化**; `borderless→MoveResize→restoreBorders` 回进入前布局。
- i18n: zh/en 加 `node.GetWindow/WindowState/MoveResizeWindow/CloseWindow` 块 + `input.Window.label` + State 5 option + Done/Window 字段 label; 跑 `pnpm gen:node-i18n`。
- 全绿门(`add-node.md §8`): `go build ./... && go test ./internal/nodes/... ./internal/node/... ./internal/catalog/... ./internal/services/container/...` + 前端 `typecheck`/`i18n:check` + `task build` + 真机 smoke(三处面板能加、单窗口默认不变、多窗口连线作用对窗口、无边框往返回原布局)。

## 8. 影刀对照

| 影刀 | 本设计 |
|---|---|
| 【获取窗口对象】(不改激活) | `GetWindow`(解析→Window, 不改活动窗口) |
| 捕获当前激活/句柄等多途径 | v1: 按 title/class/process; 其它 YAGNI |
| 存窗口变量复用 | 绑变量(config.capture)+ GetVar |
| `activate()` | 现有 `BringWindowForeground` + 可选 Window 输入 |
| `maximize/minimize/restore/close` | `WindowState` 枚举 + `CloseWindow` |
| 操作指定窗口对象 | 任一 NeedsWindow 节点的可选 `Window` 输入(默认最近) |
