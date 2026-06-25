# Window 类型 + 窗口控制节点 — design

SUMMARY: 把窗口做成一等数据类型(仿 Image) + 给 NeedsWindow 节点统一加可选 Window 输入(不连=最近活动窗口) + 新增窗口控制节点群; 多窗口靠存窗口变量再拉出来, 不改默认用法。
READ WHEN: 实现 / 复核 Window 类型与窗口控制节点 · 改活动窗口解析 / 派发期窗口覆盖时

---

## 1. Background / 动机

YHFish 现在的窗口模型是**单一粘性活动窗口**: `WindowTarget` 节点按 title/class/process 解析一个窗口, 经 `SetActive` 写进 `rt.window`(`runtime_context.go:183`); 之后所有 `NeedsWindow` 节点(截图/点击/检测/输入)经 `rt.ActiveHWND()` 隐式读它。要切窗口就在执行线上再插一个 `WindowTarget`——"最近跑过的 WindowTarget 决定当前窗口"。

真实场景: **打开一个 app, app 里有子窗口(对话框)**, 脚本要分别控制主窗口和子窗口。现状靠 re-target 能跑, 但缺两样:

1. 没有"让某个节点显式作用在指定窗口"的能力——只能改全局粘性窗口。
2. 没有把"窗口"当成可命名、可复用对象的手段。

参考: 商业 RPA **影刀** 把窗口做成一等对象 `xbot.win32.Window`——【获取窗口对象】产出窗口对象 → 存窗口变量 → 复用 → 在它身上 `activate()/maximize()/minimize()/close()`; 点击等操作也能指定窗口对象。本设计取其"窗口是可复用对象"的内核, 但**输入做成可选、默认走最近活动窗口**, 避免每个节点都被迫连线。

## 2. Goals / Non-goals

**Goals**

- 新增 `Window` 一等 pin 类型(仿 `Image`: 只产出/连线/存变量, 不可手填)。
- 任一 `NeedsWindow` 节点都能**可选**接一个 `Window` 输入; 不连 → 用当前活动窗口(现状行为, 绝大多数图零改动); 连了 → 作用在传入窗口, 作用域限本节点。
- 新增窗口控制节点群: 最大化 / 最小化 / 还原 / 无边框全屏 / 退无边框 / 移动改尺寸 / 关闭。
- 把散落的窗口节点收进一个新的 "Window" palette 类别。

**Non-goals (YAGNI / 硬约束)**

- 不做"真·独占全屏"(DirectX 那种, 外部强制不了)。
- 不把活动窗口模型推倒重来; `WindowTarget` + 粘性活动窗口仍是默认上下文, `Window` 输入是显式覆盖。
- v1 不单开 `GetWindow` 产出节点——复用 `WindowTarget`(它产出 `Window`)。等出现"要拿窗口对象但不想改活动窗口"的具体需求再加。
- 不做"激活窗口"新节点——`BringWindowForeground` 已是激活(置前台), 经统一 `Window` 输入即可激活指定窗口。

## 3. 现状锚点 (实现前先核, 别脑补)

- 粘性活动窗口: `runtime_context.go` `WindowHandle()`:166 / `ActiveHWND()`:173 / `SetActiveWindow()`:183(整体替换 + 清该 hwnd 帧缓存, **不拉前台不闪**——拉前台是 sendinput 那条在 adapter `SetActive` 单独做的, `node_services.go:434`)。
- 窗口元数据结构: `winutil.WindowHandle{HWND,Title,Class,ProcessName,PID,ClientW,ClientH}`(`pkg/winutil/window.go:86`)。
- 现成只读产出类型范式: `node.Image{Format,Data}`(`types.go:68`) + 类型注册 `RegisterType`(`types.go:81`) + 内置类型表(`types.go:103`)。
- 窗口服务抽象: `node.WindowService`(`services.go:176`, 方法 BringForeground/HWND/ClientSize/SetActive) + adapter(`node_services.go:389`) + stub。
- 窗口控制底层原语已在用: `ShowWindow`/`SetForegroundWindow`/`IsIconic`(`window.go:28`)。
- catalog 导出读各节点 `s.Inputs`(`catalog.go:58`); runtime spec 来自 `node.All()→rn.Spec`(`registry.go:116`)——**在 `node.Register` 处给 NeedsWindow 节点追加 `Window` 输入, 下游全见**。
- NeedsWindow 校验要求图里有 WindowTarget(`runner.go:410` containerNeedsWindow / validator 同判定)。
- 加节点全链路: `flightdeck/knowledge/nodes/add-node.md`; 数据流: `node-data-flow.md`; 类型表: `node-system-reference.md`。

## 4. 核心设计

### 4.1 `Window` 类型

- Go 域类型 `node.Window`(放 `types.go`), 字段镜像 `winutil.WindowHandle`(HWND uintptr + Title/Class/Process/PID/ClientW/ClientH)。值在进程内按引用流动, 跟 Image 一样视为不可变。
- `RegisterType(TypeSpec{Tag:"Window", GoType:"node.Window", WidgetKind:"preview", Color:"#22d3ee"})`——cyan, 现有类型色未占用; 只读 preview, 不可在 Inspector 手填, 只能由连线/变量提供(与 Image/List 同)。
- 取值 helper: `Inputs.Window(name) (node.Window, bool)`(`inputs.go`), 仿 `Point()`; 宽松处理 nil/类型不符返 zero+false。

### 4.2 `WindowTarget` 产出 `Window`

- 在 `WindowTarget` 的 `Done` 出口加一个 Data 字段 `Window`(Type "Window")(`window_target.go:45`)。`Run` 解析成功后 `ctx.Out(wtOutDone).Set("Window", win).Fire()`。
- 这样用户能用现成 **config.capture** 机制把它绑到一个窗口变量(`add-node.md §1b`), 下游 `GetVar` 拉出来。`WindowTarget` 仍照旧设粘性活动窗口——产出 `Window` 是**附加**能力, 不改原语义。
- 多窗口典型用法:
  ```
  WindowTarget(主窗口, 绑变量 main) → WindowTarget(子窗口, 绑变量 dlg)
     → 最大化(Window ← GetVar main)        # 作用主窗口
     → 截图  (Window ← GetVar dlg)         # 作用子窗口
     → 点击  (无 Window 输入)               # 作用当前活动窗口 = dlg(最近 target)
  ```

### 4.3 可选 `Window` 输入 + 派发期覆盖

**统一注入(推荐, 已核实可行)**: 在 `node.Register`(`registry.go`)里, 若 `spec.NeedsWindow` 且其 Inputs 未显式声明 `Window`, 追加 `InputSpec{Name:"Window", Type:"Window"}`(可选, 无 Default)。一处改, catalog / 校验 / `newInputs` / 派发全见, **22 个现有节点文件不动**。

> 退化方案(若集中注入有副作用): 每个 NeedsWindow 节点 Spec 手加一行(~22 处, 机械)。覆盖**逻辑**无论哪种都集中在派发层, 只是 pin **声明**位置不同。spec→plan 阶段二选一。

**派发期覆盖逻辑**(框架一处, 节点 Run 不动):
- 框架在跑一个 NeedsWindow 节点前, 其 `Inputs` 已构建好(`newInputs` 已按数据线/literal/default 填值)。
- 若 `in.Has("Window")` 且取到合法 `Window`:
  1. 存当前 `prev := rt.WindowHandle()`;
  2. `rt.SetActiveWindow(传入窗口转 winutil.WindowHandle)`;
  3. 跑节点(所有服务经 `rt.ActiveHWND()` 自动读到覆盖窗口);
  4. `rt.SetActiveWindow(prev)` 还原——**覆盖作用域限本节点, 不污染粘性活动窗口**。
- 不连 `Window` → 跳过整段 → 行为与现状逐字节一致。

挂载点: 派发单节点处(`dispatch_v5.go` 调 `RunNode` 的外层)统一 wrap; 具体函数 plan 阶段定。

### 4.4 窗口控制节点群 (新增)

按"配置形状"拆三个(纯状态无废框 / 带坐标 / 关闭各成节点), 全部 `NeedsWindow: true`, 经 4.3 自动获得可选 `Window` 输入:

| 节点 Kind | 操作 | 输入(除 In + 注入的 Window) | 出口 |
|---|---|---|---|
| `WindowState` | `State` 下拉枚举 | State: maximize/minimize/restore/borderlessFullscreen/restoreBorders | Done |
| `MoveResizeWindow` | 移动到 X/Y + 设宽高 | X/Y/Width/Height(Number, 屏幕像素) | Done |
| `CloseWindow` | 温和关闭 | — | Done |

单 `Done` 出口; 真出错(如 SetWindowPos 返 0)裸冒泡当节点失败, 不给死 Fail 引脚(同 WaitWindow 取舍)。

**各 State 的 Win32 语义**:

| State | 实现 |
|---|---|
| `maximize` | `ShowWindow(SW_MAXIMIZE)` — 带标题栏填满工作区 |
| `minimize` | `ShowWindow(SW_MINIMIZE)` |
| `restore` | `ShowWindow(SW_RESTORE)` — 从最大/最小还原 |
| `borderlessFullscreen` | 去 `WS_CAPTION\|WS_THICKFRAME` + `MonitorFromWindow`→`GetMonitorInfo` 拿窗口所在显示器 + `SetWindowPos` 铺满该显示器(盖任务栏) |
| `restoreBorders` | 加回 `WS_OVERLAPPEDWINDOW` + `SW_RESTORE` — 退无边框回普通窗口(**不记忆全屏前精确 rect**, 只回普通窗口) |

`MoveResizeWindow`: `SetWindowPos(hwnd, 0, X, Y, Width, Height, SWP_NOZORDER)`。
`CloseWindow`: `PostMessage(hwnd, WM_CLOSE, 0, 0)`——等于点 X, 可能弹保存框; **≠ 杀进程**(那是别的)。

### 4.5 分层

```
internal/nodes/window/*.go   新节点: 读 State/x/y/w/h, 调 ctx.Window().<op>()。零 Win32。
        ↓
node.WindowService  新增控制方法(Maximize/Minimize/Restore/BorderlessFullscreen/
   ├─ windowAdapter:  RestoreBorders/MoveResize/Close); 经 rt.ActiveHWND() 拿当前(可能已被
   │                  4.3 覆盖的)活动窗口 hwnd, 调 winutil 纯函数。
   └─ stubWindowService: no-op 返 nil(测试)。
        ↓
pkg/winutil  新增纯函数: Maximize/Minimize/Restore/BorderlessFullscreen/RestoreBorders/
             MoveResize/CloseWindow(hwnd)。复用已有 user32 LazyDLL 句柄, 补
             SetWindowPos / Get-SetWindowLongPtr / MonitorFromWindow / GetMonitorInfo / PostMessage。
```

控制节点经 `ctx.Window()`(当前活动窗口, 已含 4.3 覆盖)而非自己接 hwnd——这样控制节点和截图/点击走同一条"活动窗口(可被 Window 输入覆盖)"语义, 一致。

### 4.6 "Window" palette 类别 (组织收尾)

新建类别 `Window`; 现有窗口节点改挂 `Category:"Window"`: `WindowTarget` / `WaitWindow` / `WaitWindowGone`(现 System) + `BringWindowForeground`(现 Input) + 新增三个。文件不挪(只改 Spec `Category` 字段); 新节点放新 `internal/nodes/window/` 包(需 blank-import: `main.go` + `dispatch_v5_test.go`)。GROUP_MAP / nodeGroup.* i18n 按 `add-node.md §6` 加。

## 5. Data flow / 用法

- 常见(单窗口/顺序切换): 不连 `Window` 输入 → 全走当前活动窗口。零改动、零 spaghetti。
- 多窗口: `WindowTarget` 绑窗口变量 → `GetVar` → 拉进需要的那个节点的 `Window` 输入。只有要"作用在非当前窗口"的那一两个节点才连线。
- `Window` 值进变量 = `winutil.WindowHandle` 那套元数据(`any` 存取, `GetVar` 原样取出, 同 Image/Point)。

## 6. Open questions (spec→plan 阶段定)

1. **集中注入 vs 逐节点声明** `Window` 输入(4.3)——默认走集中注入; plan 阶段验证 `node.Register` 追加 input 对 spec-consistency 守卫(`TestSpecConsistency_*` / `TestNoPinNameSplit`)/ 校验管线 / 前端渲染无副作用, 有则退化逐节点。
2. **切到非当前窗口做 sendinput 点击要不要补拉前台**: 4.3 的覆盖只换 hwnd 不拉前台; sendinput 后端注入需目标前台。覆盖期若 InputBackend=sendinput, 是否在 set 覆盖窗口后比照 adapter `SetActive`(`node_services.go:434`)补一次 `BringToForeground`+150ms? 倾向: 是(对齐既有 sendinput 焦点契约), 但仅覆盖且 sendinput 时。postmessage 后端不需要。
3. **传入窗口句柄已失效**(变量里的窗口中途被关): `Window` 输入的 hwnd 已不存在 → 覆盖后 ShowWindow/SetWindowPos 失败。统一在覆盖入口校验 hwnd 有效(`IsWindow`)→ 失效则节点 Coded error(如 `WINDOW_INVALID`), 别静默 no-op。
4. **`Window` 类型 widget/颜色**: preview 只读(定); FE 经 `GetAllTypes` 自动拿颜色映射; 色定为 cyan `#22d3ee`(现有未占用)。
5. **控制节点 granularity 复核**: 三节点拆分(WindowState/MoveResizeWindow/CloseWindow)已是结论; 若 plan 阶段发现 borderless 的 restore 语义复杂, 不影响拆分。

## 7. Testing / 验证

- `pkg/winutil` 纯函数靠真窗口, 难单测(同 `BringToFront` 现状无单测); 重点测节点层 + 覆盖逻辑。
- 节点层用 `StubWindowService` 测: State 正确解析 / MoveResize 取 x/y/w/h / Done 触发; Spec 一致性守卫。
- 4.3 覆盖逻辑: 用 fixture 注入两个窗口, 测"连 Window 输入的节点作用在 B、之后无输入节点回到粘性 A"(单测 runtime 派发, 仿 `windowtarget_dispatch_test.go`)。
- i18n: zh/en 加 `node.WindowState/MoveResizeWindow/CloseWindow` 块 + 注入的 `input.Window.label` + State 5 个 option 翻译 + Done 出口 label; 跑 `pnpm gen:node-i18n`。
- 全绿门(`add-node.md §8`): `go build ./... && go test ./internal/nodes/... ./internal/node/... ./internal/catalog/... ./internal/services/container/...` + 前端 `typecheck`/`i18n:check` + `task build` + 真机 smoke(三处面板能加、单窗口默认行为不变、多窗口连线作用对窗口)。

## 8. 影刀对照 (参考, 非照搬)

| 影刀 | 本设计 |
|---|---|
| 【获取窗口对象】产出窗口对象 | `WindowTarget` Done 带 `Window` Data 字段 |
| 存窗口变量复用 | 绑变量(config.capture)+ GetVar |
| `activate()` | 现有 `BringWindowForeground` + 可选 Window 输入 |
| `maximize/minimize/restore/close` | `WindowState` 枚举 + `CloseWindow` |
| 操作指定窗口对象 | 任一 NeedsWindow 节点的可选 `Window` 输入(默认最近) |
