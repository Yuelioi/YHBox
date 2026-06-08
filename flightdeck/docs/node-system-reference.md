---
status: active
last_updated: 2026-06-06
when_to_read: 查 pin 类型有哪些 / 查节点 Run 里能拿哪些 ctx 服务 / 不确定 pin 值取值优先级 / 想要节点 kind 全目录 / 用拟人化 jitter
applies_to: [node-system, pin-types, ctx-services, inputs, outputs, catalog, jitter, reference]
when_to_update: 增删 pin 类型 / 改 ctx 服务集 / pin 值取值优先级 / 新增或删除节点 kind / 改 jitter 模型时
---

# 节点系统参考

配套 [node-system-architecture.md](node-system-architecture.md) 的速查表。源码：`internal/node/`。

## 1. Pin 类型（12 内置）

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
| `Exec` | (framework) | exec-pin | 控制流连线，非数据 |

**域类型形状**（`types.go`）：`Point{X,Y}` / `Rect{X,Y,W,H}`（都是 ratio float）；`Geometry{Pct Rect, Overrides []GeoOverride}` —— 运行时解析：匹配当前帧分辨率的 override 优先，否则 `pct×帧尺寸`，`pct.W==0||H==0` 且无匹配 = 全帧。Geometry pin 值的存储形状坑见 incident [../incidents/2026-06-04-geometry-pin-value-pct-shape.md](../incidents/2026-06-04-geometry-pin-value-pct-shape.md)。

自定义类型用 `node.RegisterType(TypeSpec{...})`。

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

`Run`/`RunRegion`/`Evaluate` 期间框架注入 `Ctx`（`interfaces.go` + `ctx.go`）。基础：`Context()`（容器 cancel，长操作配合 Stop 瞬停）、`Now()`、`Out(exitName)`。

服务（**全 nullable**；有 service 的节点 wire 时已塞真 backend，节点直接用；测试 stub）：

| `ctx.X()` | 接口 | 给谁用 |
|---|---|---|
| `Vision()` | VisionService | 模板匹配 Match/WaitMatch、颜色 DetectColor/HSV、双色条 DualBarTrack、ROIColorScan、帧签名 GridSignature |
| `Input()` | InputService | KeyDown/Up、Click、MouseMoveRel/MoveTo、Scroll、MouseDown/Up（xRatio/yRatio 是 0-1 客户区比例） |
| `Vars()` | VarStore | SetVar/GetVar/IncVar；scope = auto/local/global |
| `Sys()` | SysStore | GetSys（read-only，path 形如 `lastTemplate.found`，schema 见 `services/container/sys/schema.go`） |
| `Params()` | ParamStore | GetParam（读当前 frame 的 subgraph 入参，read-only） |
| `Window()` | WindowService | BringForeground / HWND / ClientSize / SetActive |
| `Capture()` | CaptureService | Screenshot（Capture / CaptureROI，返 PNG 字节） |
| `Stopwatches()` | StopwatchStore | StopwatchStart/Stop/Read（per-key，跟 vars 独立命名空间） |
| `Clip()` | ClipPlayer | PlayClip（阻塞回放录制，ctx 取消即中断释放按键） |
| `Log()` | LogService | Debug/Info/Warn（接 zerolog） |

> 用 `ctx.Vision/Input/Capture/Window/Clip` 的节点 = `NeedsWindow: true`，否则无窗口容器里被静默 no-op。

## 4. 输出 — OutBuilder

节点不直接构造 `Outputs`，走 `ctx.Out(exitName).Set(field, value).Fire()`（`outputs.go` / `ctx.go`）。守卫：

- `Out(name)`：name 不在 `Spec.Outputs` → **立即 panic**（author bug，fail fast）。`DynamicOutputs` 节点放行任意 name。
- `Set` after `Fire` → panic；同一 Run 内**第二次 Fire**（任何 builder）→ panic（ctx 级 `markFired` 守卫）。
- exec 出口携带数据：`ctx.Out("Found").Set("Point", pt).Fire()` —— 下游 exec-data wire 收同名字段。出口能带哪些 Data 在 `Spec.Outputs[].Data` 声明。

## 5. 拟人化 jitter

`jitter.go`：`JitterInt(base, pct)` / `JitterDuration(d, pct)` —— 对值施加 **±pct% 近正态**抖动（取 5 个 uniform 样本求均值 → 中心极限，值聚在中点、极端罕见，比纯 uniform 拟人）。`pct<=0` → 原值不变。时间/移动类节点的 `JitterPct` 输入走这个。

## 6. 节点目录（69 kinds / 9 category）

> **AI / 调研节点必读**：要某节点的**全 pin / 全出口 + 出口携带数据 (Data)** 明细，**跑命令拿当前值，别翻源码、别信本页下面这张表的数字**（它只存结构层、会过时）。三个口子同一数据源、都带大白话 + 出口 Data：
> - `task nodes`（= `go run ./cmd/node-catalog export --md`）—— 人读 Markdown 速查表，扫一眼回答"哪些出口吐 Point/坐标"。
> - `go run ./cmd/node-catalog export` —— 同数据的 JSON。
> - MCP `list_nodes` —— 同数据，给 LLM 直接调。
>
> 另一个视图——**按 pin 名合并**（命名对齐用，不是逐节点）：`task nodes:pins`（= `go run ./cmd/node-catalog pins`）—— 全节点 pin 名归并 + 用量 + 「命名分裂告警」（揪 `Roi` vs `ROI` 这种）。加新节点选 pin 名时查它，配 [node-spec-style §9 Canonical 词汇表](../checklists/node-spec-style.md)。
>
> 数据源 `node.All()` → `catalog.BuildWithI18n()`（结构来自 `catalog.Build()`，i18n 经 `node-i18n.json`，`cd frontend && pnpm gen:node-i18n` 生成、catalog drift 测试守护）。出口携带的 Data 字段（如 `DetectColor.Found` 的 `Center(Point)`）在 `Spec.Outputs[].Data` 声明、由 catalog 序列化导出。

| Category | kinds |
|---|---|
| **Control** (8) | Break, Continue, If, Loop, Sleep, Start, Stop, Switch |
| **Detect** (11) | CheckTemplate, ClickTemplate, DetectColor, DetectColorBlobs, DetectColorHSV, DualColorBarTrack, ROIColorScan, Screenshot, WaitChange, WaitStable, WaitTemplate — **全 NeedsWindow** |
| **Event** (1) | EventTick |
| **Input** (10) | BringWindowForeground, ClickAt, KeyHoldStart, KeyHoldStop, KeyPress, MouseHoldStart, MouseHoldStop, MouseMoveRel, MouseMoveTo, Scroll — **全 NeedsWindow** |
| **IO** (2) | Log, PlayClip(NeedsWindow) |
| **PureFunc** (23) | Add, And, Concat, Contains, Div, Eq, Expr, Gt, GtEq, Length, Lt, LtEq, Mod, Mul, Neg, Not, NotEq, Or, Select, Sub, ToBool, ToNumber, ToString — **全 PureData (Evaluator)** |
| **Stopwatch** (3) | StopwatchRead, StopwatchStart, StopwatchStop |
| **System** (7) | CollapsedNode, CommentBox(VisualOnly), MouseCalibration, Subgraph(Region), Throw, Try(Region), WindowTarget |
| **Variable** (5) | GetParam(PureData), GetSys(PureData), GetVar(PureData), IncVar, SetVar |

能力小结：**PureData/Evaluator 26 个**（PureFunc 23 + Variable 的 3 个 Get*）；**RegionRunner 4 个**（Loop, Try, Subgraph, CollapsedNode）；**NeedsWindow 22 个**（Detect 11 + Input 10 + PlayClip）；其余 Runnable。

> 计数是 2026-06-08 实测（DetectColorBlobs 加入后）。加/删节点后数字会变 —— 要现值跑命令，别信这表过时数字。

**DetectColorBlobs**（2026-06-08 加）：颜色连通域定位 —— 给 Range(hsv/rgb 6 槽) + ROI → flood-fill(8-邻域) 找所有色块 → Found 出口带 `Blobs`(JSON: 每块归一化 centerX/centerY/x/y/w/h + area 像素数) + `BlobCount`(MinArea 过滤后总数, 不受 MaxBlobs) + `PrimaryCenter`/`PrimaryArea`(按 Sort 排序首项, **非必然最大**)。Sort: area_desc / dist_screen_center((0.5,0.5)) / dist_point(RefPoint 归一化, 未设默认(0,0))。TimeoutMs=0 单次扫描。坐标全帧归一化、质心=像素均值(非 bbox 中心)。**不做** 形态学合并(碎裂目标 v1 不保证)、精确血量%(走 DualColorBarTrack)、3D 导航。
