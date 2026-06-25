# Window 类型 + 窗口控制节点 — design

SUMMARY: 把窗口做成一等数据类型(仿 Image) + GetWindow/WindowTarget 两个产出节点 + 给 NeedsWindow 节点统一加可选 Window 输入(不连=最近活动窗口, 连了=派发期 per-node 覆盖) + 窗口控制节点群; 默认用法零改动。
READ WHEN: 实现/改 internal/nodes/window/* · 派发期窗口覆盖逻辑(dispatch 单节点 wrap) · runtime_context 的 ActiveHWND/WindowHandle/覆盖字段 · WindowService 控制方法 · winutil 窗口控制原语 时。

---

> 行号是「约」提示, **会漂**——实现前以函数/符号名 grep 核实(项目铁律: 不信行号信源码)。
> 评审纳入(2026-06-25, ds/gpt/claude 三家 + 用户裁定): 覆盖机制改 per-node 覆盖字段+defer; 无边框存原布局; CloseWindow 定为「发送关闭请求」; sendinput 覆盖补前台/hwnd 失效校验由 open 提为 decided; 控制节点 Done 透传 Window; 注入改显式 helper; GetWindow 进 v1; 兼容不考虑(未发布, 现存仅 1 个容器, 直接改)。

## 1. Background / 动机

现状窗口模型 = **单一粘性活动窗口**: `WindowTarget` 按 title/class/process 解析窗口 → `SetActive` 写进 `rt.window` → 之后所有 `NeedsWindow` 节点经 `rt.ActiveHWND()` 隐式读它。切窗口 = 执行线上再插一个 `WindowTarget`(最近跑过的决定当前窗口)。

真实场景: 打开一个 app, app 里有子窗口(对话框), 脚本要分别控制主窗口和子窗口。现状靠 re-target 能跑, 但缺: ① 让某节点显式作用在指定窗口(只能改全局粘性); ② 把窗口当可命名复用对象。

参考影刀 `xbot.win32.Window`: 【获取窗口对象】产出对象 → 存变量复用 → `activate/maximize/minimize/close`; 操作也能指定窗口对象。本设计取其内核, 但**窗口输入做成可选、默认走最近活动窗口**, 避免每节点被迫连线。

## 2. Goals / Non-goals

**Goals**

- 新增 `Window` 一等 pin 类型(仿 `Image`: 只产出/连线/存变量, 不可手填)。
- 两个产出节点: `GetWindow`(解析→Window, **不改活动窗口**) + `WindowTarget`(设活动窗口 + **附带**产出 Window)。
- 任一 `NeedsWindow` 节点可**可选**接 `Window` 输入; 不连=当前活动窗口(现状, 零改动); 连了=派发期作用在传入窗口、作用域限本节点。
- 窗口控制节点群: WindowState(最大化/最小化/还原/无边框全屏/退无边框) + MoveResizeWindow + CloseWindow。
- 收一个 "Window" palette 类别。

**Non-goals (YAGNI / 硬约束)**

- 不做「真·独占全屏」(DirectX, 外部强制不了)。
- 不推倒活动窗口模型; `WindowTarget` + 粘性活动窗口仍是默认上下文, `Window` 输入是显式覆盖。
- 不做并行/async 节点执行(当前单线程顺序派发; 真要并行是另一次大改, 与本设计无关)。
- 不为「未来扩展」留口子; **不写任何兼容 shim**(未发布、现存仅 1 个容器, 结构变更直接改那个容器)。

## 3. 现状锚点 (实现前 grep 核实, 别脑补; 行号会漂)

- 粘性活动窗口: `runtime_context.go` `WindowHandle()`/`ActiveHWND()`/`SetActiveWindow()`(约 :166/:173/:183)。`SetActiveWindow` = 整体替换 + 清该 hwnd 帧缓存, **不拉前台不闪**(拉前台是 sendinput 在 adapter `SetActive` 单独做, `node_services.go` 约 :434)。
- 窗口元数据结构: `winutil.WindowHandle{HWND,Title,Class,ProcessName,PID,ClientW,ClientH}`(`pkg/winutil/window.go`)。解析入口 `winutil.ResolveWindow`(GetWindow/WindowTarget 共用)。
- 只读产出类型范式: `node.Image{Format,Data}` + `RegisterType` + 内置类型表(`types.go`)。
- 窗口服务抽象: `node.WindowService`(`services.go` 约 :176, 方法 BringForeground/HWND/ClientSize/SetActive) + adapter(`node_services.go` 约 :389) + stub。
- 窗口控制底层原语已部分在用: `ShowWindow`/`SetForegroundWindow`/`IsIconic`(`window.go` 约 :28)。新增: `SetWindowPos`/`Get-SetWindowLongPtr`/`MonitorFromWindow`/`GetMonitorInfo`/`GetWindowPlacement`/`SetWindowPlacement`/`PostMessage`/`IsWindow`。
- 变量值是**运行期的、不进 workflow JSON**: 容器只存 `VarDecl` 声明(`container/model.go` 约 :19/:23, `Vars []VarDecl`); 值靠运行期 `SetScoped`(`dispatch_v5.go` 约 :165 `r.bundle.Vars.SetScoped`)。→ Window 值是 run 内瞬时对象, 不持久化(同 Image)。
- NeedsWindow 校验: `runner.go` `containerNeedsWindow`(约 :410) + validator 同判定(要求图里有 WindowTarget)。
- catalog 导出读各节点 `s.Inputs`(`catalog.go` 约 :58); runtime spec 来自 `node.All()→rn.Spec`(`registry.go` 约 :116)。
- 加节点全链路: `knowledge/nodes/add-node.md`; 数据流: `node-data-flow.md`; 类型表: `node-system-reference.md`。

## 4. 核心设计

### 4.1 `Window` 类型

- Go 域类型 `node.Window`(放 `types.go`), 字段镜像 `winutil.WindowHandle`(HWND uintptr + Title/Class/Process/PID/ClientW/ClientH)。
- `RegisterType(TypeSpec{Tag:"Window", GoType:"node.Window", WidgetKind:"preview", Color:"#22d3ee"})`——cyan(现有类型色未占用); 只读 preview, 不可手填, 只能连线/变量提供(同 Image/List)。FE preview 要给一句提示「窗口对象由 GetWindow/WindowTarget 产出, 不能直接填标题」, 免用户困惑。
- 取值 helper: `Inputs.Window(name) (node.Window, bool)`(`inputs.go`), 仿 `Point()`; nil/类型不符返 zero+false。
- **HWND 是 live 句柄, 元数据是解析时快照**: 操作一律走 live HWND(实时有效性见 4.3 入口校验); Title/ClientW/H 等是 GetWindow/WindowTarget 解析那一刻的值, 窗口改标题/尺寸后会过期——**别拿 Window 里的元数据当实时值用**(要实时尺寸用 `ctx.Window().ClientSize()`)。
- **生命周期 = 单次 run 内瞬时**: 不写进 workflow JSON(见 §3 VarDecl/SetScoped), 不跨 session 持久化; 「HWND 跨重启失效」不构成问题(值根本不持久化)。

### 4.2 产出节点: `GetWindow`(新) + `WindowTarget`(增产出)

两者**职责不重叠, 差别只在改不改活动窗口**, 共用 `winutil.ResolveWindow`:

- **`GetWindow`**(新, exec 节点 Runnable, Category Window, **非** NeedsWindow): `In` → `Done`(带 Data 字段 `Window`) / `Fail`(NotFound, 同 WindowTarget)。config 同 WindowTarget(Title/Class/ProcessName/TitleMatch)。**只解析、不调 SetActive**——拿窗口对象不污染活动窗口上下文。用户把 `Done.Window` 绑变量(config.capture) → 下游 GetVar 拉出来喂进 Window 输入。
- **`WindowTarget`**(改): `Done` 出口**加 Data 字段 `Window`**(现 Done 无任何 Data 字段, 纯附加、不冲突; 无「默认捕获」概念, 用户显式绑才捕获)。仍照旧 `SetActive` 设粘性活动窗口——产出 Window 是附加能力。

**用法分流**:
- 顺序切窗口(默认上下文): 用 WindowTarget, 后续不连输入的节点走当前窗口。
- 多窗口存变量(不污染上下文): 用 GetWindow 绑变量, 各节点按需连 Window 输入。
  ```
  GetWindow(主窗口 → 变量 main)   # 不改活动窗口
  GetWindow(子窗口 → 变量 dlg)    # 不改活动窗口
    → 最大化(Window ← GetVar main)
    → 截图  (Window ← GetVar dlg)
  ```

### 4.3 可选 `Window` 输入 + 派发期 per-node 覆盖

**输入声明 — 显式共享 helper(不在 Register 偷改 Spec)**: 提供 `node.WindowInputSpec()` 返 `InputSpec{Name:"Window", Type:"Window"}`(可选, 无 Default); 各 NeedsWindow 节点在 `Spec().Inputs` 里显式 spread 它(~22 处机械加一行)。**源码可见、SpecConsistency 守卫不用特判、`Window` 是约定保留 pin 名**(NeedsWindow 节点不得另用 `Window` 作别的语义)。放弃 `node.Register` 动态注入(会让 Spec 不再「声明即文档」、影响 catalog/前端/文档/测试定位)。

**覆盖机制 — per-node 覆盖字段 + defer, 不碰粘性窗口**:
- rt 加独立字段 `windowOverride *winutil.WindowHandle`(windowMu 守护, nil=无覆盖)。
- `ActiveHWND()` / `WindowHandle()`: 覆盖非 nil 返覆盖, 否则返粘性 `rt.window`。
- 框架在跑 NeedsWindow 节点前(其 `Inputs` 已 build 好), 若 `in.Has("Window")` 且取到合法 Window:
  1. `winutil.IsWindow(hwnd)` 校验失效 → 节点 Coded error `WINDOW_INVALID`(不静默);
  2. set `rt.windowOverride`, **`defer` 清**(panic/cancel/早返都不漏, 框架本有 `runWithRecover`);
  3. **sendinput 覆盖补前台**(正确性): 若 `Container.InputBackend=="sendinput"` 且本节点是输入类, 比照 adapter `SetActive` 拉一次 `BringToForeground`+150ms(不在前台 SendInput 会打到别的窗); 截图/postmessage 不需要;
  4. 跑节点(所有服务经 `ActiveHWND()` 自动读覆盖窗口);
  5. defer 清覆盖。
- **粘性窗口 `rt.window` 全程不碰** → 不清它缓存、不污染、无泄漏。帧缓存按 hwnd, 覆盖窗口正常缓存、不抖动其它窗口。
- 不连 `Window` → 整段跳过 → 行为与现状逐字节一致。
- 同步性: 节点 Run 同步(ShowWindow/SetWindowPos/截图/点击/Clip 回放都在 Run 内完成才返回), 无「Run 返回后仍跑的异步」, 故 defer-after-Run 清覆盖正确。

**NeedsWindow 校验放宽**(因 GetWindow 可不经 WindowTarget 供窗口): 校验「图里需有 WindowTarget」改为「**有 Window 输入连线的 NeedsWindow 节点不计入**WindowTarget 需求」——即仅当存在「无 Window 输入 且 上游无 WindowTarget」的 NeedsWindow 节点时才报缺 WindowTarget。挂载点 `runner.go containerNeedsWindow` + validator 同改。

挂载点(派发单节点 wrap)精确函数 plan 阶段定(`dispatch_v5.go` 调 `RunNode` 外层)。

### 4.4 窗口控制节点群 (新增, 全 NeedsWindow + 经 4.3 获得可选 Window 输入)

| Kind | 操作 | 输入(除 In + Window) | 出口 |
|---|---|---|---|
| `WindowState` | `State` 下拉 | State(见下 5 选) | Done(透传 `Window`) |
| `MoveResizeWindow` | 移动+改尺寸 | X/Y/Width/Height(Number) | Done(透传 `Window`) |
| `CloseWindow` | 发送关闭请求 | — | Done |

- **Done 透传 Window**: WindowState/MoveResize 的 Done 带它作用的那个 Window(= 覆盖窗口或当前活动窗口), 链式 `WindowState→MoveResize→…` 同一窗口免反复 GetVar。
- 单 `Done` 出口; 真出错(SetWindowPos 返 0 等)裸冒泡当节点失败, 不给死 Fail。

**WindowState 的 5 个 State(Win32 语义)**:

| State | 实现 |
|---|---|
| `maximize` | `ShowWindow(SW_MAXIMIZE)` |
| `minimize` | `ShowWindow(SW_MINIMIZE)` |
| `restore` | `ShowWindow(SW_RESTORE)`(从最大/最小还原) |
| `borderlessFullscreen` | **先存原状态**(GetWindowPlacement + GetWindowLong style 存进 rt per-run saved-state map[hwnd]) → 去 `WS_CAPTION\|WS_THICKFRAME` → `MonitorFromWindow`→`GetMonitorInfo` 拿窗口所在显示器 → `SetWindowPos` 铺满该显示器(盖任务栏) |
| `restoreBorders` | 从 rt saved-state 取回原 style + `SetWindowPlacement` **还原全屏前精确位置/大小**(无记录则退化 `WS_OVERLAPPEDWINDOW`+`SW_RESTORE`) |

> saved-state 放 rt(per-run, adapter 管, 同帧缓存的归属), winutil 仅出纯原语(GetWindowPlacement/SetWindowPlacement/Get-SetWindowLongPtr/MonitorFromWindow/GetMonitorInfo)。这样 restoreBorders 能回到全屏前布局, 不再「回默认大小」。

- **`MoveResizeWindow`**: `SetWindowPos(hwnd, 0, X, Y, Width, Height, SWP_NOZORDER)`。坐标 = **虚拟桌面物理像素**(SetWindowPos 原生口径)。⚠ per-monitor 混合 DPI(主屏 100% + 副屏 150%)下物理像素跨屏需测——plan/验证阶段覆盖。
- **`CloseWindow`**: `PostMessage(hwnd, WM_CLOSE, 0, 0)` = **发送关闭请求**, 立即 Done; **不等真关闭、不处理保存框、不保证关掉**(可能弹保存框/被应用拦截)。要确认关掉 → 接现成 `WaitWindowGone`。≠ 杀进程。

### 4.5 分层

```
internal/nodes/window/*.go   GetWindow + WindowState/MoveResizeWindow/CloseWindow。控制节点零 Win32, 调 ctx.Window().<op>()。
        ↓
node.WindowService  新增控制方法(Maximize/Minimize/Restore/BorderlessFullscreen/RestoreBorders/MoveResize/Close);
   ├─ windowAdapter:  经 ActiveHWND()(含 4.3 覆盖)拿 hwnd; borderless/restore 用 rt per-run saved-state; 调 winutil 纯函数。
   └─ stubWindowService: no-op 返 nil(测试)。
        ↓
pkg/winutil  纯原语: Maximize/Minimize/Restore/MoveResize/CloseWindow + GetWindowPlacement/SetWindowPlacement/
             Get-SetWindowLongPtr/MonitorFromWindow/GetMonitorInfo/IsWindow。复用已有 user32 LazyDLL 句柄。
```

GetWindow 与 WindowTarget 共用 `winutil.ResolveWindow`(WindowTarget 多一步 SetActive)。控制节点经 `ctx.Window()`(活动窗口, 含 4.3 覆盖)拿 hwnd, 与截图/点击同语义、一致。

### 4.6 "Window" palette 类别

新建类别 `Window`; 现有窗口节点改挂 `Category:"Window"`: `WindowTarget`/`WaitWindow`/`WaitWindowGone`(现 System) + `BringWindowForeground`(现 Input) + 新增四个(GetWindow + 三控制)。文件不挪(只改 Spec `Category`); 新节点放新 `internal/nodes/window/` 包(blank-import: `main.go` + `dispatch_v5_test.go`)。GROUP_MAP / nodeGroup.* i18n 按 `add-node.md §6`。**无兼容顾虑**(未发布, 无 muscle-memory); 类别变更若影响那唯一现存容器, 直接改它。

## 5. Data flow / 用法

- 常见(单窗口/顺序切换): 不连 Window 输入 → 全走当前活动窗口。零改动、零 spaghetti。
- 多窗口不污染上下文: `GetWindow` 绑窗口变量 → `GetVar` → 拉进需要的那个节点的 Window 输入。只有要作用在「非当前窗口」的那一两个节点才连线。
- Window 值进变量 = `winutil.WindowHandle` 元数据(`any` 存取, GetVar 原样取出, 同 Image/Point); 运行期瞬时、不序列化。
- 控制链: `GetWindow→WindowState(Done.Window)→MoveResize(Done.Window)→…` 透传同一窗口。

## 6. Open questions (仅余实现细节, 核心行为已 decided)

1. 派发单节点 wrap 的精确挂载函数(`dispatch_v5.go` 内)——plan 阶段定位, 不影响行为。
2. `WindowInputSpec()` 的 i18n: `input.Window.label` 在 ~22 节点重复 vs gen 兜共享 key——plan 定; 不影响行为。
3. per-monitor 混合 DPI 下 MoveResize 物理像素的跨屏正确性——验证阶段真机覆盖。
4. NeedsWindow 校验放宽(4.3 末)的 validator/runner 具体改法——plan 定。

(已 decided, 不再 open: 覆盖机制=per-node 字段+defer; sendinput 覆盖补前台=是; hwnd 失效=入口 IsWindow→WINDOW_INVALID; 注入=显式 helper; 无边框=存原布局可还原; CloseWindow=发送请求; 类型色=#22d3ee; GetWindow 进 v1。)

## 7. Testing / 验证

- `pkg/winutil` 纯原语靠真窗口难单测(同 `BringToFront` 现状); 重点测节点层 + 覆盖逻辑。
- 节点层(StubWindowService): WindowState 各 State 解析 + MoveResize 取 x/y/w/h + CloseWindow Done + Done 透传 Window; GetWindow 解析成功/NotFound 走 Done/Fail 且**不改活动窗口**; Spec 一致性守卫。
- 4.3 覆盖逻辑(fixture 注两窗口, 仿 `windowtarget_dispatch_test.go`): ① 连 Window 输入的节点作用在 B、之后无输入节点回到粘性 A(覆盖字段 scoping); ② Window 输入 hwnd 失效 → `WINDOW_INVALID`; ③ sendinput 后端覆盖期补前台(校验 BringToForeground 被调)。
- borderless: 进入存 placement、restoreBorders 还原到原 rect(用 stub/fake 验 saved-state 往返)。
- i18n: zh/en 加 `node.GetWindow/WindowState/MoveResizeWindow/CloseWindow` 块 + `input.Window.label` + State 5 option + Done/Window 出口字段 label; 跑 `pnpm gen:node-i18n`。
- 全绿门(`add-node.md §8`): `go build ./... && go test ./internal/nodes/... ./internal/node/... ./internal/catalog/... ./internal/services/container/...` + 前端 `typecheck`/`i18n:check` + `task build` + 真机 smoke(三处面板能加、单窗口默认不变、多窗口连线作用对窗口、无边框往返)。

## 8. 影刀对照

| 影刀 | 本设计 |
|---|---|
| 【获取窗口对象】(不改激活) | `GetWindow`(解析→Window, 不改活动窗口) |
| 捕获当前激活/句柄等多途径 | v1: 按 title/class/process(同 WindowTarget); 其它途径 YAGNI |
| 存窗口变量复用 | 绑变量(config.capture)+ GetVar |
| `activate()` | 现有 `BringWindowForeground` + 可选 Window 输入 |
| `maximize/minimize/restore/close` | `WindowState` 枚举 + `CloseWindow` |
| 操作指定窗口对象 | 任一 NeedsWindow 节点的可选 `Window` 输入(默认最近) |
