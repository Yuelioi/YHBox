---
status: active
when_to_read: 设计让节点消费/产出数据的节点前; 连线节点间数据前; 撞 INVALID_PIN 'out pin X 不存在' 或数据连不上时
applies_to: [node-data-flow, data-edge, exec-edge, capture, output-capture, GetVar, Now, VarLastChange, pin-wiring, validator, node-design]
when_to_update: 改节点数据线/exec 线规则 / pin-wiring / output-capture / validator 对 pin 存在性的判定时
last_updated: 2026-06-07
---

# 节点数据流与连线

设计"消费/产出别的节点的数据"的节点、或连线节点间数据**之前**先读这份。脑补"输出口直接连过去"会撞 `INVALID_PIN`，或写出根本连不上的节点。

> 配套：加节点 kind 的全链路机械步骤看 [add-node.md](add-node.md)（含"产出节点必须加捕获框"硬约束）；pin 命名看 [node-spec-style.md](node-spec-style.md)；校验该写哪条管线看 incident [[2026-06-04-node-validation-pipeline-bifurcation]]。
>
> ⚠️ 历史变更（2026-06-07）：旧的全局 `$sys` 快照 + `GetSys` 节点 + `PathSchema` **已整套删除**。产出值改成"节点显式捕获到用户命名变量"，引擎 live 值改成 `Now` / `VarLastChange` 节点。下面是新模型；任何还提 `$sys`/`GetSys` 的记忆都过时了。

## 心智模型（一句话）

**控制流走 exec 线；数据只有"纯数据节点的输出"能直接连。exec 节点的产出不能直连——在该节点上把产出"捕获"进一个你命名的变量，下游用 `GetVar` 读出来。**

## 两种连线，由 (源节点 kind, 源 pin) 推导

- **exec 线**：源是 exec 出口（`OutputSpec.Type == "Exec"`，如 `命中`/`Done`/`Found`/`已对准`）。驱动"先跑谁后跑谁"。
- **data 线**：源是 **data-out pin** = 顶层 `OutputSpec` 且 `Type` 是数据类型（非 Exec）。**只有这种能当数据线的源。**

判定在 `internal/services/container/validate.go` 的 `dataOutPinTypeForKind` / `IsDataOutPin`：只遍历**顶层 `Spec.Outputs`**。运行时 `runtime/data_pull.go evalDataSource` 再加一道闸：**源节点必须 `IsPureData`**，否则报错。

## 谁能当数据线的源（data-out pin）

**只有纯数据节点（`IsPureData: true`）**：`GetVar` / `GetParam` / `Now` / `VarLastChange` / `Expr` + 全部 PureFunc（`Add`/`Eq`/`Select`/`ToNumber`… ）。它们各有一个非-exec 输出（多为 `Value`/`Result`），可被求值、可连数据线。

## ⛔ exec 节点的 Data 字段不是 data-out pin

exec 节点常在某个 exec 出口上**携带数据**：`OutputSpec.Data []DataField`（如 `DetectColor.Found` 带 `Count`/`Center`；`CheckTemplate.Found` 带 `Point`/`Conf`）。这些 Data 字段：

- **不能直接连数据线**（`IsDataOutPin` 只认顶层输出，不认 Data 字段 → validator 报 `INVALID_PIN out pin X 不存在`）。
- 真正用途两个：① 节点 `Display()` 打日志；② 经 **exec-data wire** 注入下游 exec 节点的**同名数据输入 pin**（顺着同一条 exec 线往下，下游有同名 input 才注入）。

**想让 exec 节点的产出被任意下游消费 → 用"输出捕获到变量"（下一节），不要假设 Data 字段能直连。**

## ✅ 标准做法：消费某 exec 节点产出的数据 —— 捕获到变量 + GetVar

产出型节点（视觉/模板/截图/秒表/Loop/Try…）在 Spec 里带一批**可选捕获框**：`Capture<字段>` String 输入（`Advanced:true, Semantic:"capture"`），`Run()` 跑完调 `node.Capture(ctx, in, "Capture<字段>", 值)` 把值写进用户命名变量（`ctx.Vars().SetScoped(name,"auto",值)`）。

以"让 `转向目标` 吃 `颜色检测` 的命中中心点"为例：

1. **exec 线**：`颜色检测.命中 → 转向目标.In`
2. 在 `颜色检测` 节点上，把 **`中心点→变量`（CaptureCenter）填 `c`**。
3. **data 线**：放一个 `GetVar`，`VarName` 填 `c` → `GetVar.Value → 转向目标.目标`。

时序：捕获写发生在产出节点 `Run()`；下游消费节点是后一个 exec step、各自入口抓新快照（`dispatchInRegion` 每个子节点各抓一次），读得到刚写的值。跟节点跑的先后一致。

**捕获语义要点（设计/排查都按这个）：**
- 捕获框 **trim 后非空** 才写；空/纯空白 = 没配 = 不写。
- 只写该 exit **实际带的**字段：如 `DetectColor` 未命中走 `NotFound`（不带 Center）→ `CaptureCenter` 该轮不写、变量留旧值。要区分"未命中"就同时捕获 `found`/`Count` 来 gate。
- 节点 `Run()` 返 error → 一个都不写（保留旧值）。
- 多个节点捕获到同名变量会互相覆盖（用户自己命名、显式可见，跟两次 SetVar 同名一样，预期行为）。
- 写进变量的是该产出**原本的 typed 值**（point=`node.Point`、clusters=`[]ClusterEntry` any…），GetVar 原样取出。

## 引擎 live 值：Now / VarLastChange（取代旧 $sys.now_ms / varLastChange.X）

| 要什么 | 用哪个纯数据节点 | 输出 |
|---|---|---|
| 当前 unix 毫秒（墙钟，live） | **Now**（无输入） | number |
| 某变量上次 Set/Inc 的 unix-ms 时间戳（没设过返 0） | **VarLastChange**（输入 VarName） | number |

"某状态多久没变" = `Now - VarLastChange(该变量)`（旧 watchdog `$sys.now_ms - $sys.varLastChange.X` 的等价写法）。循环序号 = 在 `Loop` 节点上勾 `CaptureIndex→变量` 后 `GetVar` 读。

## 设计新节点时

- **消费方**（要吃别的节点的数据）：开一个普通**数据输入 pin**（如 `Target` Point）。由图作者用 `GetVar` 桥接喂进来——**不要假设能从某个 exec 节点的 Data 字段直连**。
- **产出方·纯转换** → 做成 `IsPureData` 节点（输出可直连数据线）。
- **产出方·exec 节点** → 给每个有意义的产出加 `Capture<字段>` 捕获框（`Advanced:true, Semantic:"capture"`）+ `Run()` 里 `node.Capture(...)`。**这是硬约束**：新增产出型节点 / 给节点加 OutputData 时必须同步决定是否可捕获、默认可捕获（防"有产出但用户拿不到"，详见 [add-node.md](add-node.md) 捕获节。捕获字段只能对应 `OutputSpec.Data`，不许另立隐藏字段）。

## 数据输入 pin 怎么取值（优先级）

`data_pull.go pullDataPin`：① 有数据线 → 递归求值源（仅纯数据源）；② 否则 config `literal[pin]`（画布手填/录制/屏幕拾取）；③ 否则同名 exec-data 注入；④ 否则 nil → 消费节点用自己的默认。

## 常见错误

| 症状 | 真相 / 对策 |
|---|---|
| 连 `颜色检测.Center → 目标` 报 `out pin center 不存在` | Center 是 exec 出口携带的 Data 字段，不是 data-out pin。改：在颜色检测上勾 `CaptureCenter→c`，下游 `GetVar(c)`。 |
| 想读"上次模板命中点 / bar 位置 / 循环序号 / 当前时间" | 分别：在检测节点勾对应 Capture 框 + GetVar；Loop 勾 CaptureIndex + GetVar；`Now` 节点。**没有 `$sys`/`GetSys` 了**。 |
| 数据线连到了 exec 节点的输出 | 只有 `IsPureData` 节点的输出能当数据源；exec 节点产出走"捕获到变量 + GetVar"。 |
| 给 exec 节点加了 Data 出口，以为别人能连 | 默认不能。要让人消费 → 加 `Capture<字段>` 捕获框（见 add-node.md 硬约束）。 |
| **编辑器 footgun**：把 exec 出口的 Data 字段画成了可连输出口 | 画得出、连不上（validator 拒）。认准"纯数据节点输出 / GetVar"才是数据源。（已知 UX bug，见 [editor-footgun-backlog](../specs/editor-footgun-backlog.md)。） |

## 源码锚点（撞问题先核这些，别脑补）

- 捕获助手：`internal/node/capture.go` `Capture`
- 各节点捕获框：`grep "Semantic: \"capture\""` `internal/nodes/**`（DetectColor/模板族/HSV/ROI/DualColorBarTrack/StopwatchRead/Screenshot/Loop/Try）
- 引擎 live 节点：`internal/nodes/variable/now.go`、`var_last_change.go`；`VarStore.LastChange`（`interfaces.go`）→ `runtime_context.go VarLastChange`/`varTimestamps`
- 边类型推导 / data-out 判定：`internal/services/container/validate.go` `dataOutPinTypeForKind` `IsDataOutPin`
- 数据线只认纯数据源：`internal/services/container/runtime/data_pull.go` `evalDataSource` `pullDataPin`
- per-tick 快照（变量一致读 + 跨-step 可见）：`runtime/snapshot.go` `CaptureSnapshot`、`dispatch_v5.go dispatchInRegion`
