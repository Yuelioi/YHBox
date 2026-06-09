---
status: active
summary: 数组/集合子系统第一批 - List pin 类型 + in.List getter + ForEach + 9 个列表节点 (加节点路线图阶段2)
last_updated: 2026-06-10
related: [specs/2026-06-10-random-nodes.md]
---

# 数组/集合节点（加节点路线图 · 阶段 2）

## 背景与定位

加节点路线图（[random-nodes spec](2026-06-10-random-nodes.md)）的**阶段 2**，用户决定「数组提前」。这是唯一动框架内核的一批：先加 `List` pin 类型 + 遍历能力（ForEach RegionRunner），再做一批列表节点。用户拍板第一批 scope = **MVP 7 + ListAppend + ListSlice = 9 节点**。

**依赖**：RandomChoice 归阶段 1 的 `internal/nodes/random` 包（阶段 1 欠的），依赖本阶段的 List 类型；故本 spec 实现顺序上排在 [random-nodes](2026-06-10-random-nodes.md) 之后。

## 已验证的源码事实（设计依据）

- **新 pin 类型靠 `node.RegisterType(TypeSpec{Tag,GoType,WidgetKind,Color})`**（`internal/node/types.go`），前端启动经 `GetAllTypes` RPC 自动拉颜色/widget，无需前端逐类型登记。12 内置类型里无 List/数组。
- **`[]any` 能走数据线**：`runtime/data_pull.go::toExprValue` 对未知类型走 `default: return v`（line 176）→ `[]any` 原样透传，不被拦。
- **现有 `in.StringList(name)`**（`node/inputs.go:76`）已处理 `[]any`/`[]string`/裸 string → 泛化出 `in.List(name) []any` 即可（不做 string 强转）。
- **RegionRunner 模式**（`nodes/control/loop.go`）：`RunRegion(ctx, in, body func(Ctx) error)`；每轮 `node.Capture(ctx, in, "Capture<X>", 值)` 把迭代值落成 auto 变量供 body 的 GetVar 读；Break/Continue 走 sentinel error。
- **新 RegionRunner 要在 dispatch 登记**：`runtime/dispatch_v5.go::makeBodyFor` 的 switch 现只认 `Loop`/`Subgraph`/`CollapsedNode`，default 报 "region runner not yet supported"。加 ForEach 要加一个 `makeBodyForForEach`（镜像 `makeBodyForLoop`：seed `node.ID+".Body"` 下游 + `runRegionBody`）。
- **纯函数节点模式**（`nodes/purefunc/purefunc.go`）：`Evaluate(ctx,in)(any,error)`，`formatValue`/`equalAny` 等 helper 在 purefunc 包内**未导出**（跨包不能直接用，需在新包重写等值/格式化小helper）。
- **数据输入在 RunRegion 前已解析**：Loop `in.Int(Count)` 可用 → ForEach `in.List("List")` 在 RunRegion 内可直接读。

## A. 框架地基

1. **`List` pin 类型**：`RegisterType(TypeSpec{Tag:"List", GoType:"[]any", WidgetKind:"json", Color:"#818cf8"})`（indigo，与现有 12 类型色不撞；如撞再调）。
   - 元素**不分类型**（泛型 `[]any`，跟 `JSON=map[string]any` 一个路子；框架无 Go 泛型）。
   - **无手填 literal 编辑器**——List pin 由 Split/节点/连线驱动；`WidgetKind:"json"` 复用现有 JSON 渲染做只读预览（连进来的值可视化），不期望手敲。
2. **`in.List(name) []any` getter**（`node/inputs.go` + `interfaces.go` 接口加方法）：容忍 `[]any`（原样）、`[]string`（转 `[]any`）、nil（→ nil）。**不**把裸 string 当一元列表（与 StringList 区别：List 是真列表语义）。
3. **`toExprValue` 显式加 `[]any` case**（当前靠 default 透传，能跑但脆）：`case []any: return x`，注释标明 List 值透传。同时确认 `coerceToType`（`data_pull.go:183`）对 List 走透传（无需物化）。

## B. ForEach（RegionRunner）

- Kind `ForEach`，Category `List`。
- Inputs：`In`(Exec)、`List`(List)、`CaptureItem`(capture, CaptureType `any`)、`CaptureIndex`(capture, CaptureType `number`)。
- Outputs：`Body`(Exec)、`Done`(Exec)、`Fail`(Exec, Semantic error, Data: Error/Code)。
- `RunRegion`：遍历 `in.List("List")`，每轮 `node.Capture(ctx,in,"CaptureItem",el)` + `node.Capture(ctx,in,"CaptureIndex",i)` → `body(ctx)`；Break sentinel→走 Done、Continue→下一轮（照 Loop）。空列表 → 直接 Done。
- 框架：`makeBodyFor` switch 加 `case "ForEach": return r.makeBodyForForEach(node, tok), nil`，实现镜像 `makeBodyForLoop`（seed `node.ID+".Body"`）。

## C. 列表节点（纯数据，新 `internal/nodes/collection` 包，Category `List`）

> 全 `IsPureData:true` + Evaluator。等值/格式化用包内小 helper（purefunc 的未导出）。

| 节点 | 输入 | 输出 | 语义 |
|---|---|---|---|
| **Split** | `Text`(String)、`Separator`(String, 默认`,`) | `Result`(List) | `Text==""`→空列表；否则 `strings.Split`（元素为 string）；`Separator==""`→按字符切（Go 语义） |
| **Join** | `List`(List)、`Separator`(String, 默认`,`) | `Result`(String) | 各元素 `formatValue` 后 `strings.Join` |
| **ListLength** | `List`(List) | `Result`(Number) | `len(list)` |
| **ListGet** | `List`(List)、`Index`(Integer, 默认0) | `Result`(`*`) | 越界（含负）→ nil（Optional 语义，Has=false）；不做负索引从尾 |
| **ListContains** | `List`(List)、`Value`(`*`) | `Result`(Bool) | 任一元素与 Value 宽松相等（formatValue 比较，照 Eq 跨类型 fallback） |
| **ListAppend** | `List`(List)、`Item`(`*`) | `Result`(List) | **返回新列表**（copy 输入再 append，**绝不**原地改——输入切片可能被缓存/共享，别名会出 bug） |
| **ListSlice** | `List`(List)、`Start`(Integer, 默认0)、`Count`(Integer, 默认0) | `Result`(List) | 从 Start 取 Count 个，**返回新列表**。`Count<=0`→取到末尾；Start/Count 越界 clamp 到 `[0,len]`；Start≥len→空 |

## D. RandomChoice（归阶段 1 `random` 包）

- Kind `RandomChoice`，Category `Random`（跟阶段 1 随机组同 palette 分类），`IsPureData:true + IsNonDeterministic:true`。
- Input `List`(List)。Output `Result`(`*`)。
- 语义：均匀随机取一个元素（`rand.IntN(len)`）；空列表→nil。底层 `math/rand/v2`。受阶段 1 per-dispatch 缓存覆盖（同一求值多路径引用同值）。
- **依赖**：阶段 1 的 random 包 + 本阶段 List 类型 + `in.List`。

## 设计判断（已拍，待你复核）

- **Append/Slice 一律返回新列表**（纯函数式）：pull-based 纯数据模型里没有可变持久列表可改；要"累积改列表"用 SetVar 存 + 每轮 ListAppend 重写。
- **元素不分类型**（`[]any`）：无泛型，元素混类型由各 getter 宽松转换兜（与现有节点一致）。
- **ListGet 越界→nil**，不做 Python 式负索引（YAGNI）。
- **新 `List` palette 分类**（8 个列表节点）+ RandomChoice 进 Random 分类——前端注册路径同 [random-nodes spec](2026-06-10-random-nodes.md) 的 Category 全套（GROUP_MAP/LABEL/I18N_KEY/visualRegistry/nodeGroup i18n + NodeGroup 联合类型）。

## 非目标（YAGNI）

- 不做 **Filter/Map**（高阶，要谓词/变换的子区域或 Expr 嵌入，是另一块框架设计；ForEach+If+ListAppend 能拼出大部分）——往后按需另开 spec。
- 不做 typed list（List<String> 等）——无泛型，YAGNI。
- 不做 List literal 手填编辑器——列表从节点/连线来。
- 不做 ListInsert/ListRemove/ListSort/ListReverse/Unique 等——第一批不铺满，按 demand 再加。
- 不做 List 进 Expr 表达式（`services/expr` 不认 `[]any`）——本批列表操作走节点，不走 Expr。

## 落地清单（按 checklists/add-node.md，要点）

1. **框架**：`types.go` 注册 List；`inputs.go`+`interfaces.go` 加 `List()` getter；`data_pull.go` `toExprValue` 加 `[]any` case + 确认 `coerceToType` 透传；`dispatch_v5.go::makeBodyFor` 加 ForEach 分支。单测：List getter 各形态、toExprValue 透传、ForEach 遍历/空/break/continue、makeBodyForForEach seed。
2. **collection 包**：`internal/nodes/collection/{collection.go, collection_test.go}` —— 8 节点（含 ForEach）+ 包内 equal/format helper。10 处 blank-import 镜像 purefunc（见 random-nodes plan Task 5 同款清单）加 `_ "yotta/internal/nodes/collection"`。
3. **random 包**：加 RandomChoice + 测试 + 已有 blank-import 站点已覆盖。
4. **i18n**：zh/en 加 8 collection 节点 + RandomChoice + `nodeGroup.list`；跑 `pnpm gen:node-i18n`。
5. **前端 List 分类**：GROUP_MAP/NodeGroup 联合类型/GROUP_LABEL/GROUP_I18N_KEY/visualRegistry 加 `list`（图标如 `i-tabler-list`）。
6. **验证**：`go build ./... && go test ./internal/...`；`pnpm gen:node-i18n && pnpm typecheck && pnpm i18n:check`；`task nodes` 见全部新节点；真机 smoke：Split→ForEach→Log 跑通、List 分组三处可见。
