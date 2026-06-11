---
status: active
last_updated: 2026-06-11
when_to_read: 第一次碰节点系统 / 设计新节点前想搞懂"节点怎么被定义·注册·派发" / 不确定一个节点该实现哪种 capability / 改 framework dispatch 或 validator 前
applies_to: [node-system, framework, spec, registry, capability, dispatch, runnable, regionrunner, evaluator, validation]
when_to_update: 改节点注册流程 / capability 分类 / dispatch 派发逻辑 / RegionRunner / Evaluator / validator 管线结构时
---

# 节点系统架构

YHFish 的节点系统 = **声明式 Spec + 运行时注册表 + 能力派发（capability dispatch）**。这篇讲"节点怎么被定义、注册、跑起来"的整体架构；具体类型表 / Ctx 服务 / 节点目录见 [node-system-reference.md](node-system-reference.md)；动手加节点的全链路步骤见 [../checklists/add-node.md](../checklists/add-node.md)；pin 命名/Default 约定见 [../checklists/node-spec-style.md](../checklists/node-spec-style.md)。

源码：`internal/node/`（框架核心）+ `internal/nodes/<category>/`（69 个节点实现）。

## 1. 心智模型

- **节点只声明"我是什么"，不写"我怎么被调度"**。`Node` 接口最小化 —— 只有一个 `Spec() Spec` 方法（`interfaces.go`）。执行语义靠**另外三个 capability 接口**（Runnable / RegionRunner / Evaluator）表达。
- **后端只出结构，前端负责渲染**。Spec 里只有 kind / pin name / type / widget / enum value 这些**结构**。所有**展示文本**（节点名、描述、pin label、hint、enum 选项名）单源在前端 i18n `frontend/src/i18n/zh.ts` 的 `node.<kind>.*`（`spec.go` 头注释）。画布上的普通节点由 `ContainerFlowNode` 按 Spec 自动渲染，**新增普通节点不用写 Vue 组件**。
- **Inspector-first**：节点的输入面板从 Spec 的 `Inputs[]` 自动派生（widget / 默认值 / 可见条件 `VisibleWhen` / 结构化 schema）。

## 2. 节点解剖 — Spec

`Spec`（`spec.go`）：

| 字段 | 含义 |
|---|---|
| `Kind` | 唯一标识（PascalCase）。**是 graph JSON 的序列化 key —— 改名 = 迁移所有已存容器**，不能随便动 |
| `Category` | 前端 palette 分组（Control / Detect / Input / …，见 reference 目录） |
| `Inputs []InputSpec` | 输入 pin（exec + data） |
| `Outputs []OutputSpec` | 输出 pin（exec 出口可带 `Data []DataField`，即"出口携带的数据"，如 `CheckTemplate.Found` 带 `Point`） |

**标志位**（都是 `Spec` 上的 bool，决定派发/校验行为）：

| 标志 | 作用 |
|---|---|
| `NeedsWindow` | 节点 Run 依赖目标窗口 hwnd（调了 `ctx.Input/Capture/Vision/Window/Clip`）。**图里有 NeedsWindow 节点才要求 `WindowTarget`**；纯窗口无关容器（Sleep/Log/Expr…）免。**漏置 → 该节点在无窗口容器里被 SafeBackend 静默 no-op**，必须老实置真 |
| `IsPureData` | 纯数据节点（无副作用、求一个值）。**必须实现 Evaluator**（Register 强制） |
| `IsVisualOnly` | 纯渲染节点（如 CommentBox）。允许零 capability |
| `IsGraphMarker` | 图结构标记节点。允许零 capability（框架为 SubgraphInput/Output 预留；**当前没有后端节点用它**，子图入口/出口标记是前端 virtual 的） |
| `DynamicOutputs` | 出口名运行时按 config 推导（Switch 的 named-by-value case 出口；Subgraph/CollapsedNode 的出口 = callee OutputPins decl ID）。置真时 `ctx.Out(name)` 放行任意 name |
| `DynamicInputs` | data-in pin 由 `config.Inputs[]` 运行时声明、不在静态 `Inputs` 枚举（Expr / Script）。dispatch / validator / 前端都按此标志统一走 `container.ParseDynamicInputDecls`，**无 kind 字符串特判**（`spec.go`） |
| `IsNonDeterministic` | Evaluate 非确定（随机 / now）。框架在 per-dispatch eval 缓存里**只记忆化带此标志的节点**（`runtime/data_pull.go::evalPureDataCached` 的 `caching` gate），保证同一次派发内多路径引用同一节点拿同一个值。仅对 `IsPureData` 节点有意义 |

`InputSpec` 关键字段：`Type`（类型 tag，见 reference）、`Required`、`Advanced`、`Default`（**Number 类用 `json.Number` 保精度**）、`Widget`（UI 控件，跟 Type 解耦）、`VisibleWhen`（条件显隐）、`Schema`（结构化输入递归 schema，非 nil → 前端 StructuredInput，如 Geometry/HSV）。

## 3. 三条能力路线（the routes）—— 节点必须**恰好实现一种**

这是"节点路线"的核心：一个节点的执行语义由它实现哪个 capability 接口决定。Register 时框架**用 type assertion 探测一次**并缓存成函数指针（`registry.go`），引擎派发只走缓存指针、**绝不再 runtime assert**。

| 路线 | 接口（`interfaces.go`） | framework 入口（`engine.go`） | 用于 | 例子 |
|---|---|---|---|---|
| **Runnable** | `Run(ctx, in) (Outputs, error)` | `RunNode` | 普通 exec 节点，框架同步调一次 | KeyPress、Sleep、SetVar、CheckTemplate、Script（内嵌 JS） |
| **RegionRunner** | `RunRegion(ctx, in, body func(Ctx) (string, error)) (Outputs, error)` | `RunNodeAsRegion` | 控制流 / 包子图 body —— body 回调由 dispatch 构造，**节点自己决定调几次**；body 第一返回值 = region 内部到达的出口 | **恰好 4 个**：Loop（调 N 次/forever）、ForEach（遍历 List 逐项调）、Subgraph、CollapsedNode |
| **Evaluator** | `Evaluate(ctx, in) (any, error)` | `EvaluatePureData` | 纯数据求值，返一个标量给 data-edge 下游，**无 exec 出口** | 41 个 PureFunc（Add/Eq/Select…）+ Expr、List 类（Join/ListGet…）、Random×4、GetVar/GetParam/Now/VarLastChange（PureData 全集计数见 [reference §6](node-system-reference.md)，别在这写死数字） |

**两个例外**（允许零 capability）：`IsVisualOnly`（CommentBox）、`IsGraphMarker`（结构标记）。

### RegionRunner 的 body 回调

`body func(Ctx) (string, error)` 是"执行 region 内部下游"的回调，第一返回值 = **region 内部到达的出口**（Subgraph/CollapsedNode：callee OutputPins 的 decl ID；Loop/ForEach 单轮迭代无出口语义，忽略它）。Subgraph/CollapsedNode 把 body 回报的出口原样 `ctx.Out(exit)` fire（Spec 是 `DynamicOutputs`，父图边 pin = decl ID）；`""` = 没到达任何出口 → 不 fire。节点对返回的 error 做语义翻译：Loop/ForEach 截获 `errBreakRequested`（跳完成出口）/ `errContinueRequested`（下一轮），其余 error 直接 propagate，由该 RegionRunner 的 **`Fail` 出口**（`Semantic: error`；四个 RegionRunner 都声明了）接住（`loop.go` / `foreach.go`）。`Throw` 节点（Runnable，`internal/nodes/system/throw.go`）就是借这条 propagate 通道把错误抛给最近一层 RegionRunner 的 Fail。这就是 Break/Continue/Throw 这些 sentinel 节点能工作的机制 —— **没有 Try 节点**（旧文档误记，截获是靠 RegionRunner 的 Fail 出口，不是一个专门节点）。

### 子图调用的出口路由（2026-06-11 起）

子图调用（Subgraph / CollapsedNode 节点，或脚本 `Subgraph()` 函数）统一走 runtime 共享核心 `runSubgraphCall`（`runtime/subgraph_call.go`）：push frame + seed params + 切 dispatch 表到 callee + 从 entry 播种跑到出口 marker。**出口 key 单一来源 = callee OutputPins 的 decl ID**（父图边 / 前端 handle / runtime fire 三层一致；decl rename 只改 Name 不动 ID，父图无感）。出口裁决：到达 marker → 该 decl；跑干没到 marker → 单出口视同唯一出口，多出口 → `subgraph_no_exit` Coded 错误（可被 Fail 接）。嵌套深度上限 32（`subgraph_recursion`，防脚本动态递归）。脚本侧拿到的是 decl **Name**（人读名），图层路由用 decl **ID**。

### Evaluator 看到的是 tick-frozen 快照

PureData 节点 Evaluate 内看到的 `Vars`/`Sys` 是**当前 tick 冻结的快照**，不是 live state（`engine.go::EvaluatePureData` 入口 `services.Snapshot` wrap）。这保证同一 tick 内多个 data 节点读到一致的状态。为什么用 wrap 而不是给 Ctx 加方法，见 [framework-extension-dispatch-context.md](framework-extension-dispatch-context.md)。

> ⚠️ `interfaces.go:44-48` 有句**陈旧注释**说 GetVar/GetParam 不实现 Evaluator、dispatch 走 fallback —— 已过时。实测（grep）这俩 + 全部 PureFunc 都实现了 Evaluator + `IsPureData`。以源码为准。（注：**没有 GetSys 节点** —— `GetSys` 是 `ctx.Sys()` 服务的读方法，read-only 系统信息，不是一个可放画布的节点。）

### 选哪条路线（决策树）

1. 节点要**产出一个值给别的节点的 data 入口**、无副作用 → **Evaluator**（+ `IsPureData: true`）。
2. 节点要**包住一段子流程、控制它跑几次/截获其结果** → **RegionRunner**。
3. 其余有副作用的"做一件事然后往下走" → **Runnable**。
4. 只为画布展示、不参与执行 → `IsVisualOnly`，零 capability。

## 4. 节点要求（注册契约）

一个节点能被系统认到，必须：

1. **`func init() { node.Register(&X{}) }`**（节点文件里）。
2. **包被 blank-import**：新 `internal/nodes/<category>` 包要在 `main.go` + `internal/services/container/runtime/dispatch_v5_test.go` 里有 `_ "yotta/internal/nodes/<category>"`，否则 `init` 不跑、节点不存在（已有 category 包加节点则无需动）。
3. **满足 capability invariant**（`registry.go` Register 时校验，违反直接 panic、init-time 立刻暴露）：
   - 非 marker/visual 节点 **恰好一种** capability（0 个 → panic "zero capabilities"；>1 个 → panic "multiple capabilities"）。
   - `IsPureData: true` 必须实现 Evaluator，否则 panic。
4. 注册表在 Wails OnStartup 末尾 `Freeze()`，之后再 Register 会 panic（注册只能在 init 期）。

可选扩展接口（type assertion 探测，实现即生效）：`Validator`（节点自身静态校验）、`Dependencer`（子图分享/library import 时 BFS 抽外部资产引用）。

## 5. 校验双管线（最容易踩的坑）

节点校验有**两条独立管线**，写错地方会"加了校验但编辑期不报 / 重复报"（incident [../incidents/2026-06-04-node-validation-pipeline-bifurcation.md](../incidents/2026-06-04-node-validation-pipeline-bifurcation.md)）：

| 管线 | 在哪 | 何时跑 | 写什么 |
|---|---|---|---|
| **编辑期** | `internal/services/container/validator.go` 的 `checkGraphPerKind` → `validateXxx(n)` | NodeInspector 实时（编辑器红错） | 图级/容器级 per-kind 静态校验（断边、缺 WindowTarget、必填 pin 缺失 `MISSING_REQUIRED_PIN`、未知 literal pin、RegexMatch/RegexExtract 的 `INVALID_REGEX_PATTERN` 等） |
| **运行期** | 框架 `prepareExec`：`validateRequired`（`REQUIRED_FIELD_MISSING`）+ 节点的 `Validate()`（Validator 接口） | engine 真跑该节点时 | 节点自身输入合法性（HSV min>max 之类） |

**编辑期校验写在 `checkGraphPerKind`，不是节点的 `Validate()` 方法**（后者只在 runtime 跑，编辑器看不到）。静态必填校验为什么要镜像 `PinValue` 的两级回退（literal + 顶层 config），见 incident [../incidents/2026-06-02-pin-presence-check-must-mirror-pinvalue.md](../incidents/2026-06-02-pin-presence-check-must-mirror-pinvalue.md)。

## 6. 错误分类（RunResult）

框架 `RunNode` 把单节点结果分三类（`engine.go` 头注释）：

- **Validation**：user graph 写错（Required 缺值 / Validator 返错）→ 节点变红，**不是 panic**。
- **Error**：runtime fail（Run 返 error）→ 节点变红。
- **Panic**：framework invariant 被破（double Fire / `Out(unknown)` / 不可能状态）→ `runWithRecover` recover + stack 进 log；panic 路径会清空 ExitName/OutputData 防止 half-baked result 被当合法路由。

services 内任何字段都可能 nil；节点拿到 nil service 调方法 → panic → 被 framework recover 报回 `RunResult.Panic`。

## 7. 值与类型 helper（框架级，常被新节点漏用）

- **`node.FormatValue(v any) string`**（`value_helpers.go`）：任意值软转字符串（nil→`"null"`、bool→`"true"/"false"`、数字/字符串直转、其余 `fmt.Sprintf("%v")`）。日志 / 比较 / 字符串节点统一走它，别各写一份。
- **`node.LooseEqual(a, b any) bool`**（同文件）：宽松等值 —— 同类型可比较的直比，否则 `FormatValue` 串比（slice 这类不可比较的也安全降级到串比）。Eq/NotEq/Contains 等 PureFunc 跨类型比较的统一口径。
- **`List` pin 类型 + `in.List(name) []any`**（`inputs.go`）：List 型 pin（异构列表）取值 helper，容忍 `[]any` 原样 / `[]string` 转换，nil 及其它类型 → nil 不 panic；**不把裸 string 当一元列表**（与 `StringList` 区别）。List 类型表/颜色见 [reference §1](node-system-reference.md)。
- **字符串位置语义按 rune**：`Length` 用 `utf8.RuneCountInString`（CJK 一字算 1，非 byte），与 `Substring`/`IndexOf` 的位置语义统一。

## 8. 相关

- 加节点全链路 checklist：[../checklists/add-node.md](../checklists/add-node.md)
- pin 命名 / Default / exec exit 约定：[../checklists/node-spec-style.md](../checklists/node-spec-style.md)
- 节点间数据怎么连（pin wiring / GetSys / exec 出口 Data）：[../checklists/2026-06-05-node-data-flow.md](../checklists/2026-06-05-node-data-flow.md)
- 类型表 / Ctx 服务 / pin 值解析 / 节点目录：[node-system-reference.md](node-system-reference.md)
- framework 扩展（行为随 dispatch context 变时该怎么扩）：[framework-extension-dispatch-context.md](framework-extension-dispatch-context.md)
