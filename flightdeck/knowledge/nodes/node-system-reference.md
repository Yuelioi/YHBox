# 节点系统参考

SUMMARY: 节点系统速查表 —— pin 类型、pin 值取值优先级、Ctx 服务目录、节点目录查询命令、jitter
READ WHEN: 查 pin 类型有哪些 / 查节点 Run 里能拿哪些 ctx 服务 / 不确定 pin 值取值优先级 / 想要节点 kind 当前目录 / 用拟人化 jitter
RECHECK WHEN: 增删 pin 类型 / 改 ctx 服务集 / pin 值取值优先级 / 新增或删除节点 kind / 改 jitter 模型时

---

配套 [node-system-architecture.md](node-system-architecture.md) 的速查表。源码：`internal/node/`。

## 1. Pin 类型（14 内置）

`types.go init()` 注册，前端启动时经 `GetAllTypes` RPC 拉颜色/widget 映射：

| Type tag | Go 类型 | 默认 widget | 说明 |
|---|---|---|---|
| `String` | `string` | text | |
| `Number` | `float64` | number | Default 用 `json.Number("0")` |
| `Integer` | `int` | number | 同上 |
| `Bool` | `bool` | checkbox | |
| `Point` | `node.Point` | point-editor | ratio 坐标 (0-1) |
| `Rect` | `node.Rect` | rect-editor | ratio 矩形 |
| `Geometry` | `node.Geometry` | geometry | pct 自适应 + 可选 per-resolution 像素覆盖 |
| `Color` | `node.Color` | color-picker | HSV |
| `Image` | `*image.RGBA` | preview | |
| `Duration` | `time.Duration` | duration | 值是毫秒数字 |
| `JSON` | `map[string]any` | json | |
| `List` | `[]any` | list-preview | 异构列表；只读占位（不可在 Inspector 手输，必须由连线提供） |
| `Window` | `node.Window` | preview | Win32 HWND 窗口对象，运行期瞬时值，不序列化进 workflow JSON |
| `Exec` | (framework) | exec-pin | 控制流连线，非数据 |

**域类型形状**（`types.go`）：`Point{X,Y}` / `Rect{X,Y,W,H}`（都是 ratio float）；`Geometry{Pct Rect, Overrides []GeoOverride}` —— 运行时解析：匹配当前帧分辨率的 override 优先，否则 `pct×帧尺寸`，`pct.W==0||H==0` 且无匹配 = 全帧。Geometry pin 值的存储形状坑见 [geometry-pin-value-pct-shape.md](geometry-pin-value-pct-shape.md)。

自定义类型用 `node.RegisterType(TypeSpec{...})`。

列表值存变量时变量类型声明 **any**（`List` 型变量暂不支持，`IncVar` 对 `any` 型 list 变量会静默改写，属 GIGO）；列表不能进 `Expr` 运算（会干净报错，但错误被数据线吞成 nil，调试看日志）。

## 2. Pin 值解析优先级

`Inputs` 取值，高优先级覆盖低（`inputs.go::newInputs`，倒序填）：

```
1. data-wire    上游数据 pin 连进来的         [最高]
2. config       Inspector 配置（画布正源 = config.literal[pin]）
3. exec-data    上游 exec 出口的同名 Data 字段注入
4. Default      InputSpec.Default
5. Required + 缺 → framework 返 ValidationError（不是 panic）
6. Optional + 缺 → 零值，Has() = false
```

- 画布上 pin 字面值的**正源是 `config.literal[pin]`**（顶层 `config[pin]` 只是读取 fallback）。静态分析路径（library scanner / dependency）用 `node.NewInputsFromConfig(cfg)` 复刻这个 literal 优先级。
- `Inputs` 取值方法（`interfaces.go`）：`String/Float64/Int/Bool/StringList/Point/Rect/Geometry/Color/Duration/JSON/Raw/Has/Keys`。各方法做宽松类型转换（如 `Duration` 容忍 float/int/json.Number/string；`Float64` 容忍 json.Number/string）。
- `Keys()` 给 dynamic-input 节点（Expr）遍历所有 merged key 用。

## 3. Ctx 服务目录

`Run`/`RunRegion`/`Evaluate` 期间框架注入 `Ctx`（`interfaces.go` + `ctx.go`）。稳定核心只有 `Context()`（容器 cancel，长操作配合 Stop 瞬停）、`Now()`、`Out(exitName)`、`CaptureOutput(...)` 与 `Services()`。

`Services()` 按值返回本 dispatch 的 `ServiceBundle`。字段可以 nil，但 Spec 声明的 `RuntimeCapabilities` 会在节点代码前验证；真正可选的 port 必须 nil-safe：

| `ctx.Services().X` | 接口 | 给谁用 |
|---|---|---|
| `Vision` | VisionService | 模板匹配 Match/WaitMatch、颜色 DetectColor/HSV、双色条 DualBarTrack、ROIColorScan、帧签名 GridSignature |
| `Input` | InputService | KeyDown/Up、Click、MouseMoveRel/MoveTo、Scroll、MouseDown/Up（xRatio/yRatio 是 0-1 客户区比例） |
| `Vars` | VarStore | SetVar/GetVar/IncVar；scope = auto/local/global |
| `Params` | ParamStore | GetParam（读当前 frame 的 subgraph 入参，read-only） |
| `Window` | WindowService | BringForeground / HWND / ClientSize / SetActive |
| `Target` | TargetService | Android 等非窗口目标选择；Win32WindowTarget 仍走 WindowService |
| `App` | AppLifecycleService | AndroidStartApp / AndroidStopApp 等目标内应用生命周期 |
| `Capture` | CaptureService | Screenshot（Capture / CaptureROI，返 PNG 字节） |
| `Stopwatches` | StopwatchStore | StopwatchStart/Stop/Read（per-key，跟 vars 独立命名空间） |
| `Clip` | ClipPlayer | PlayClip（阻塞回放录制，ctx 取消即中断释放按键） |
| `Subgraphs` | SubgraphCaller | Script 绑定层同步调用当前容器子图 |
| `AI` | AIProviderService | AI 节点按 connectionID 获取 provider |
| `Log` | LogService | Debug/Info/Warn（接 zerolog） |

依赖标记不要按服务名旧经验乱填：

- `ctx.Services().Input` / `Capture` / `Vision`：通常是 target-aware 服务，节点应使用 `NeedsTarget: true`，声明 `RuntimeCapabilities` 与具体 `TargetCapabilities`。
- `ctx.Services().Window` / Win32 HWND 语义：使用 `NeedsWindow: true`，并声明 window runtime capability。
- `ctx.Services().App`：使用 `NeedsTarget: true`，声明 app runtime capability 与 app lifecycle target capability。
- `ctx.Services().Clip`：当前是 Windows 输入回放语义，按具体节点的 Window/Foreground contract 判断。

## 4. 输出 — OutBuilder

节点不直接构造 `Outputs`，走 `ctx.Out(exitName).Set(field, value).Fire()`（`outputs.go` / `ctx.go`）。守卫：

- `Out(name)`：name 不在 `Spec.Outputs` → **立即 panic**（author bug，fail fast）。声明 output role 的 `DynamicPorts` 节点由外部绑定完整出口集，旧 runtime 的 builder 因此放行动态 name；新 Program runtime 必须改为只接受 Compiler 冻结的 resolved ports。
- `Set` after `Fire` → panic；同一 Run 内**第二次 Fire**（任何 builder）→ panic（ctx 级 `markFired` 守卫）。
- exec 出口携带数据：`ctx.Out("Found").Set("Point", pt).Fire()` —— 下游 exec-data wire 收同名字段。出口能带哪些 Data 在 `Spec.Outputs[].Data` 声明。

## 5. 拟人化 jitter

`jitter.go`：`JitterInt(base, pct)` / `JitterDuration(d, pct)` —— 对值施加 **±pct% 近正态**抖动（取 5 个 uniform 样本求均值 → 中心极限，值聚在中点、极端罕见，比纯 uniform 拟人）。`pct<=0` → 原值不变。时间/移动类节点的 `JitterPct` 输入走这个。

## 6. 节点目录查询

> **AI / 调研节点必读**：要某节点的**全 pin / 全出口 + 出口携带数据 (Data)** 明细，**跑命令拿当前值，别翻源码、别信旧静态表**。三个口子同一数据源、都带大白话 + 出口 Data：
> - `task nodes`（= `go run ./cmd/node-catalog export --md`）—— 人读 Markdown 速查表，扫一眼回答"哪些出口吐 Point/坐标"。
> - `go run ./cmd/node-catalog export` —— 同数据的 JSON。
> - MCP `list_nodes` —— 同数据，给 LLM 直接调。
>
> 另一个视图——**按 pin 名合并**（命名对齐用，不是逐节点）：`task nodes:pins`（= `go run ./cmd/node-catalog pins`）—— 全节点 pin 名归并 + 用量 + 「命名分裂告警」（揪 `Roi` vs `ROI` 这种）。加新节点选 pin 名时查它，配 [node-spec-style §9 Canonical 词汇表](node-spec-style.md)。
>
> 数据源 `node.All()` → `catalog.BuildWithI18n()`（结构来自 `catalog.Build()`，i18n 经 `node-i18n.json`，`cd frontend && pnpm gen:node-i18n` 生成、catalog drift 测试守护）。出口携带的 Data 字段（如 `DetectColor.Found` 的 `Center(Point)`）在 `Spec.Outputs[].Data` 声明、由 catalog 序列化导出。

2026-06-30 核验的粗计数，仅作 sanity check，不作为开发依据：

- total kinds: 130
- categories: AI 1, Control 9, Detect 18, Event 1, Image 3, Input 11, IO 6, List 8, PureFunc 46, Random 4, Stopwatch 3, System 5, Target 2, Variable 6, Window 7
- PureData: 65
- NeedsTarget: 28
- NeedsWindow: 6
- Target selection nodes: AndroidTarget, Win32WindowTarget

加/删节点后数字会变。要现值跑命令，别信旧表。

**DetectColorBlobs**（2026-06-08 加）：颜色连通域定位 —— 给 Range(hsv/rgb 6 槽) + ROI → flood-fill(8-邻域) 找所有色块 → Found 出口带 `Blobs`(JSON: 每块归一化 centerX/centerY/x/y/w/h + area 像素数) + `BlobCount`(MinArea 过滤后总数, 不受 MaxBlobs) + `PrimaryCenter`/`PrimaryArea`(按 Sort 排序首项, **非必然最大**)。Sort: area_desc / dist_screen_center((0.5,0.5)) / dist_point(RefPoint 归一化, 未设默认(0,0))。TimeoutMs=0 单次扫描。坐标全帧归一化、质心=像素均值(非 bbox 中心)。**不做** 形态学合并(碎裂目标 v1 不保证)、精确血量%(走 DualColorBarTrack)、3D 导航。
