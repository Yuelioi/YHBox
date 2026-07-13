# 节点数据流与连线 checklist

SUMMARY: 节点间数据怎么连 —— 控制流走 exec 线、纯数据节点输出可直连、exec 出口 Data 字段走 held output 直连；需要命名复用时再捕获到变量 + GetVar
READ WHEN: 设计让节点消费/产出数据的节点前; 连线节点间数据前; 撞 INVALID_PIN 'out pin X 不存在' 或数据连不上时
RECHECK WHEN: 改节点数据线/exec 线规则 / pin-wiring / output-capture / validator 对 pin 存在性的判定时

---

设计"消费/产出别的节点的数据"的节点、或连线节点间数据**之前**先读这份。先分清连的是 exec 流、纯数据输出，还是 exec 出口携带的 Data 字段；三者的校验和运行时取值路径不同。

> 配套：加节点 kind 的全链路机械步骤看 [add-node.md](add-node.md)（含产出节点 `config.capture` 模型——节点只声明 Data 字段，无捕获框）；pin 命名看 [node-spec-style.md](node-spec-style.md)；校验该写哪条管线看 [node-validation-pipeline-bifurcation.md](node-validation-pipeline-bifurcation.md)。
>
> ⚠️ 历史变更（2026-06-07）：旧的全局 `$sys` 快照 + `GetSys` 节点 + `PathSchema` **已整套删除**。产出值改成"节点显式捕获到用户命名变量"，引擎 live 值改成 `Now` / `VarLastChange` 节点。下面是新模型；任何还提 `$sys`/`GetSys` 的记忆都过时了。

## 心智模型（一句话）

**控制流走 exec 线；纯数据节点输出可按需求值；exec 节点出口携带的 Data 字段可直接拉数据线，运行时读 held output 缓存。需要给产出命名、跨图复用或跨作用域时，再把 Data 字段捕获到变量并用 `GetVar` 读。**

## 两种连线，由 (源节点 kind, 源 pin) 推导

- **exec 线**：源是 exec 出口（`OutputSpec.Type == "Exec"`，如 `命中`/`Done`/`Found`/`已对准`）。驱动"先跑谁后跑谁"。
- **data 线**：源是 data-out pin。它有两类：① 顶层数据输出（纯数据节点的 `Value`/`Result` 等）；② exec 出口下声明的 `DataField`（如 `Found.Center`、`Fail.Code`、AI 动态输出字段）。

判定在 `internal/services/container/validate.go`：
- 顶层数据输出：`dataOutPinTypeForKind` / `IsDataOutPinNode`。
- exec 出口 Data 字段：`IsExecOutputDataField` / `IsExecOutputDataFieldNode`（config-aware，覆盖 `DynamicPorts` outputData descriptor 指向的 `config.Outputs[]`）。

## 谁能当数据线的源（data-out pin）

顶层数据输出只有纯数据节点（`IsPureData: true`）能安全作为按需求值源：`GetVar` / `GetParam` / `Now` / `VarLastChange` / `Expr` + 全部 PureFunc（`Add`/`Eq`/`Select`/`ToNumber`… ）。它们各有一个非-exec 输出（多为 `Value`/`Result`）。

运行时 `runtime/data_pull.go evalDataSource` 对这类源再加一道闸：源节点必须 `IsPureData` 且实现 Evaluator，否则报错。exec 节点不会因为 data pull 被重跑。

## exec 节点的 Data 字段：held output 直连（任意距离）

exec 节点常在某个 exec 出口上**携带数据**：`OutputSpec.Data []DataField`（如 `DetectColor.Found` 带 `Count`/`Center`；`CheckTemplate.Found` 带 `Point`/`Conf`；`Fail` 带 `Error`/`Code`），以及 `DynamicPorts` outputData descriptor 指向的 config 声明字段（如 AI 结构化输出 `red`/`white`）。这些 Data 字段：

- **是 data-out pin、可直连数据线**：`IsDataOutPin`/`IsDataOutPinNode`（config-aware）认 exec 出口的 Data 字段 → validator 放行连线。
- **值经 held output 缓存任意距离直连下游 data-in**：源 fire 时存进 `ContainerRunner.execOutputs`，下游 `pullDataPin` 直读，**免 GetVar、免紧邻约束**。完整机制 → [held-exec-outputs.md](held-exec-outputs.md)。
- 另两个用途：① 节点 `Display()` 打日志；② 经 `RunNode` 的 `execData` 参数走 `in.X("k")` **原始 key-match**（消费 pin 名 == 源字段名时，无需数据线）。

**想让 exec 节点的产出被任意下游消费 → 直接拉数据线（默认、零样板）；要显式命名变量 / 跨子图作用域才用「输出捕获到变量」（下一节）。**

## 需要命名复用时：config.capture 绑变量 + GetVar

> ⚠️ **2026-06-15 (Spec C T4, `33fa43f`)**：旧的 `Capture<字段>` 输入框 + `node.Capture` 助手 + `Semantic:"capture"` **整套删除**。任何还提"加捕获框 / 调 node.Capture"的记忆都过时了——节点**零捕获代码**。

产出型节点（视觉/模板/截图/秒表/Loop/AI…）**只在 exec 出口声明 `OutputSpec.Data` 字段**，`Run()` 里 `ctx.Out(exit).Set(field,值)…Fire()`。捕获由框架做：用户在 Inspector 把某 Data 字段**绑到一个变量**（存进 `node.config.capture`：`map[字段]→变量`），fire 时 `dispatch_v5.applyCaptures` 自动 `Vars().SetScoped(varName,"auto",值)`。

以"把某个检测结果命名成变量，后面多处复用"为例：

1. **exec 线**：`颜色检测.Found → 后续节点.In`
2. 在 `颜色检测` 节点的「输出」组里，把 **`Center` 这个 Data 字段绑到变量 `c`**（写进 config.capture）。
3. 后续多处放 `GetVar`，`VarName` 填 `c` → `GetVar.Value` 连到需要 Point 的输入。

时序：捕获写发生在产出节点 fire 时（`applyCaptures`）；下游消费节点是后一个 exec step、各自入口抓新快照（`dispatchInRegion` 每个子节点各抓一次），读得到刚写的值。跟节点跑的先后一致。

**捕获语义要点（设计/排查都按这个）：**
- 只写该 exit **实际带的**字段（`data` 稀疏，只含节点 `.Set()` 过的）：如 `DetectColor` 未命中走 `NotFound`（不带 Center）→ 绑了 Center 的那条该轮不写、变量留旧值。要区分"未命中"就同时绑 `Count` 来 gate。
- 节点 `Run()` 返 error → 不 fire → 一个都不写（保留旧值）。
- 多个节点绑同名变量会互相覆盖（用户自己命名、显式可见，跟两次 SetVar 同名一样，预期行为）。
- 写进变量的是该产出**原本的 typed 值**（point=`node.Point`、clusters=`[]ClusterEntry` any…），GetVar 原样取出。
- 可绑字段 = `nodepkg.BindableFields(spec)`（exec 出口 Data 字段）；悬空绑定（删字段/删变量后残留）由 `validator_capture_refs.go` 暴露。

## 引擎 live 值：Now / VarLastChange（取代旧 $sys.now_ms / varLastChange.X）

| 要什么 | 用哪个纯数据节点 | 输出 |
|---|---|---|
| 当前 unix 毫秒（墙钟，live） | **Now**（无输入） | number |
| 某变量上次 Set/Inc 的 unix-ms 时间戳（没设过返 0） | **VarLastChange**（输入 VarName） | number |

"某状态多久没变" = `Now - VarLastChange(该变量)`（旧 watchdog `$sys.now_ms - $sys.varLastChange.X` 的等价写法）。循环序号优先直接连 `Loop` Body 出口的 `Index` Data 字段；要命名复用时再把 `Index` 通过 `config.capture` 绑变量后 `GetVar` 读。

## 设计新节点时

- **消费方**（要吃别的节点的数据）：开一个普通**数据输入 pin**（如 `Target` Point）。图作者可以直接从 exec 出口 Data 字段连线，也可以用 `GetVar` 喂命名变量；节点侧不用区分。
- **产出方·纯转换** → 做成 `IsPureData` 节点（输出可直连数据线）。
- **产出方·exec 节点** → 在 exec 出口声明 `OutputSpec.Data` 字段 + `Run()` `.Set(field,值)`，**就这样**。捕获是框架的事（用户绑 `config.capture`，`applyCaptures` 自动写）——**不加捕获框、不调 node.Capture**（已删，见上方 ⚠ + [add-node.md §1b](add-node.md)）。

## 数据输入 pin 怎么取值（优先级）

`data_pull.go pullDataPin`：① 有数据线且源是 exec 出口 Data 字段 → 读 held output 缓存 `execOutputs`（见 [held-exec-outputs.md](held-exec-outputs.md)）；② 有数据线且源是纯数据 → 递归求值源；③ 否则 config `literal[pin]`（画布手填/录制/屏幕拾取）；④ 否则 nil → 消费节点用自己的默认（节点 Run 时 `execData` 参数另走 `in.X("k")` 同名 key-match）。

## 常见错误

| 症状 | 真相 / 对策 |
|---|---|
| 连 `颜色检测.Center → 目标` 报 `out pin center 不存在` | 先确认 pin 名大小写。Data 字段名按 Spec 大小写，通常是 `Center`；小写 `center` 是无效 pin。 |
| 想读"上次模板命中点 / bar 位置 / 循环序号 / 当前时间" | 命中点 / bar 位置优先直接连对应 exec 出口 Data 字段；要多处复用再 capture + GetVar。循环序号可用 Loop Body 的 `Index` Data 字段或 capture + GetVar；当前时间用 `Now`。**没有 `$sys`/`GetSys` 了**。 |
| 数据线连到了普通 exec 出口（Done/Found 本身） | exec 出口本身是控制流，不是数据。要连它下面的 Data 字段（如 `Center` / `Code` / `Window`），或连纯数据节点输出。 |
| 给 exec 节点加了 Data 字段但前端/validator 认不出 | 检查 `OutputSpec.Data` 字段名、动态字段 `config.Outputs[]`、`IsExecOutputDataFieldNode`，以及 node-i18n/catalog 是否已生成。 |
| 想把产出跨子图边界复用 | 直接 held output 只在本次 runner 的节点 ID / data edge 内解决。跨子图公开契约或用户可读复用时，用 capture + GetVar / RequiredGlobals。 |

## 源码锚点（撞问题先核这些，别脑补）

- 捕获写入：`internal/services/container/runtime/dispatch_v5.go` `applyCaptures`（fire 时按 `node.config.capture` 写变量）
- 捕获绑定校验 / 可绑字段：`internal/services/container/validator_capture_refs.go` · `nodepkg.BindableFields`
- 产出节点范式（只声明 Data + Set，无捕获框）：`internal/nodes/detect/detect_color_blobs.go`
- 引擎 live 节点：`internal/nodes/variable/now.go`、`var_last_change.go`；`VarStore.LastChange`（`interfaces.go`）→ `runtime_context.go VarLastChange`/`varTimestamps`
- 边类型推导 / data-out 判定：`internal/services/container/validate.go` `dataOutPinTypeForKind` `IsDataOutPin`
- 数据线只认纯数据源：`internal/services/container/runtime/data_pull.go` `evalDataSource` `pullDataPin`
- per-tick 快照（变量一致读 + 跨-step 可见）：`runtime/snapshot.go` `CaptureSnapshot`、`dispatch_v5.go dispatchInRegion`
