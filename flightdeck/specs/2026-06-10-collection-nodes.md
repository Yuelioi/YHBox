---
status: active
summary: 数组/集合子系统第一批 - List pin 类型 + in.List getter + ForEach + 8 个列表节点 (加节点路线图阶段2)
last_updated: 2026-06-10
related: [specs/2026-06-10-random-nodes.md]
---

# 数组/集合节点（加节点路线图 · 阶段 2）

## 背景与定位

加节点路线图（[random-nodes spec](2026-06-10-random-nodes.md)）的**阶段 2**，用户决定「数组提前」。唯一动框架内核的一批：加 `List` pin 类型 + 遍历（ForEach RegionRunner）+ 一批列表节点。用户拍板 scope = MVP 7 + ListAppend + ListSlice。经一轮外部 AI 审（3 家）收紧如下。

**依赖**：RandomChoice 的**代码**放阶段 1 的 `internal/nodes/random` 包（就近），但它是**本阶段（阶段 2）的交付物**、依赖本阶段 List 类型，实现排在 [random-nodes](2026-06-10-random-nodes.md) 之后。包归属 ≠ 阶段归属。

## 已验证的源码事实（设计依据）

- **新 pin 类型靠 `node.RegisterType(TypeSpec{Tag,GoType,WidgetKind,Color})`**（`internal/node/types.go`），前端经 `GetAllTypes` RPC 自动拉颜色/widget。
- **`[]any` 能走数据线**：`runtime/data_pull.go::toExprValue` 未知类型走 `default: return v` → `[]any` 原样透传。
- **`in.StringList`**（`node/inputs.go:76`）已处理 `[]any`/`[]string` → 泛化 `in.List(name) []any`。
- **RegionRunner 模式**（`nodes/control/loop.go`）：`RunRegion(ctx, in, body func(Ctx) error)`；每轮 `node.Capture(ctx, in, "Capture<X>", 值)` 落 auto 变量；Break/Continue 走 sentinel error。`makeBodyForLoop` 实体 = `seeds := r.edges.next(node.ID+".Body", tok.LoopStack); r.runRegionBody(c.Context(), seeds)`。
- **Region 错误路由**（`dispatch_v5.go:218-249`，所有节点共用）：`RunRegion` 返 **Coded error** 且 `node.ID+".Fail"` 有连线 → `nextWithData(".Fail", {Error,Code})`；Break/Continue sentinel 漏给 RunRegion（不进失败路由）；非 Coded error 上抛容器。→ **ForEach 的 Fail 完全复用此机制，无需自己 fire**。
- **per-dispatch evalCache 只 gate `IsNonDeterministic`**（`data_pull.go::evalDataSource`）→ 确定性节点（ListAppend/Slice 等）**不进缓存、每 pull 重算**。
- **`Eq` 用 `equalAny`**（`purefunc.go`：同类型直比 `a==b`，跨类型 `formatValue` 串比）；`formatValue` 软转任意值为 string。两者**包内未导出** → 见 A.4 导出方案。
- **数据输入在 RunRegion 前已解析**：Loop `in.Int(Count)` 可用 → ForEach `in.List("List")` 在 RunRegion 内可读。

## A. 框架地基

1. **`List` pin 类型**：`RegisterType(TypeSpec{Tag:"List", GoType:"[]any", WidgetKind:<见下>, Color:"#818cf8"})`。元素**不分类型**（`[]any`，无泛型，跟 `JSON=map[string]any` 一路子）。
   - **widget（待前端验证）**：优先**只读预览**渲染连进来的列表值。先确认现有 `json` widget 是否可编辑：若只读则用它；若可编辑，则**顺势允许 JSON 数组 literal 作为列表来源之一**（用户可手敲 `["a","b"]`，是 Split 之外的额外来源），但需保证它解析为 `[]any`。impl 第一步定夺。
2. **`in.List(name) []any` getter**（`node/inputs.go` + `interfaces.go`）：容忍 `[]any`（原样）、`[]string`（转 `[]any`，一次性分配、不缓存）、nil（→nil）。**不**把裸 string 当一元列表（与 StringList 区别）。
3. **`toExprValue` 显式加 `case []any: return x`**（当前靠 default 能跑但脆，显式化 + 注释；纯加固）。确认 `coerceToType`（`data_pull.go:183`）对 List 走透传。
4. **导出等值/格式化 helper**：把 `purefunc` 的 `equalAny`/`formatValue` 提升为 **`node.EqualAny(a,b any) bool` / `node.FormatValue(v any) string`**（移到 `internal/node` 或新 `internal/nodes/valueutil`，purefunc 改调用）。→ ListContains 用 `node.EqualAny`（**语义 == Eq 节点**）、Join 用 `node.FormatValue`，**全项目一套等值/格式化**，杜绝两套实现漂移。

## A'. 框架审计（⛔ plan 前必做 — 头号铁律 consumer-audit）

只验了数据线透传（toExprValue）**不够**。List 类型一旦能进变量/持久化，下列链路都要逐一确认 `[]any` 能稳定携带，否则"存进变量的列表读出来碎了"类 bug（参 incident [[2026-05-29-storage-convention-consumer-audit-gap]]）：

- **SetVar/GetVar**：var store 是 `map[string]expr.Value`（`expr.Value=any`）→ `[]any` 作为值能存（**低风险，初判 OK**），但要验 GetVar 读回类型不被压扁。
- **TickSnapshot**：`CaptureSnapshot` 做 `maps.Copy` 浅拷贝 → `[]any` 值按引用共享（只读快照，**低风险 OK**）。
- **catalog 导出 / RPC schema / 前端图保存恢复**：List **literal**（若 A.1 允许 JSON 数组 literal）要能 JSON round-trip；wire 值不持久化（只 literal+结构持久化）→ 若不做 literal 则此项风险低。**要验**：GetAllTypes 把 List 类型推给前端后，前端 pin 渲染/连线校验认不认。
- **validator**：确认无对 pin 类型的 exhaustive switch 会因未知 List 报错。

**产出**：plan 前出一页"List 链路审计"结论；命中阻塞则回炉。

## B. ForEach（RegionRunner，放 `nodes/control` 包）

> 与 Loop 同为 RegionRunner、同包（`nodes/control`，遵循机制归类，不混进纯数据 collection 包）。Category 标 `List`（palette 与列表节点同组）。

- Kind `ForEach`，Category `List`。
- Inputs：`In`(Exec)、`List`(List)、`CaptureItem`(capture, CaptureType `any`)、`CaptureIndex`(capture, CaptureType `number`)。
- Outputs：`Body`(Exec)、`Done`(Exec)、`Fail`(Exec, Semantic error, Data: Error/Code)。
- `RunRegion`：`items := in.List("List")`（**入口取一次快照**，循环中上游变动不影响本次遍历，照 Go for-range）；`for i, el := range items { Capture(CaptureItem,el); Capture(CaptureIndex,i); body(ctx) }`；Break→Done、Continue→下一轮（照 Loop sentinel）。空/非列表 → 0 轮 → 直接 Done（不算错）。
- **Fail**：复用框架 region 错误路由（见源码事实），ForEach 不自己 fire。
- 框架：`makeBodyFor` switch 加 `case "ForEach": return r.makeBodyForForEach(node, tok), nil`，实体 = `seeds := r.edges.next(node.ID+".Body", tok.LoopStack); r.runRegionBody(c.Context(), seeds)`（与 `makeBodyForLoop` 逐行同构）。
- **注**：`CaptureIndex` 是 `number`（float64，与 Loop 既有一致）；下游 `in.Int` 宽松转回 int 安全；Expr 里做 index 运算得 float（既存现象，非本节点引入）。

## C. 列表节点（纯数据，新 `internal/nodes/collection` 包，Category `List`）

> 7 个，全 `IsPureData:true` + Evaluator。底层 `strings` + `node.EqualAny`/`node.FormatValue`。

| 节点 | 输入 | 输出 | 语义 |
|---|---|---|---|
| **Split** | `Text`(String)、`Separator`(String,默认`,`) | `Result`(List) | 元素为 string。**边界（刻意，写进 i18n）**：`Text==""`→**空列表**（刻意偏离 Go 的 `[""]`，更直觉）；`Separator==""`→按 **rune**（UTF-8 码点，非字节非字素簇）切；`Text=="" && Sep==""`→空列表；否则 `strings.Split` |
| **Join** | `List`(List)、`Separator`(String,默认`,`) | `Result`(String) | 各元素 `node.FormatValue` 后 `strings.Join`（复杂元素 map/list 的串化由 FormatValue 统一定，与 ToString/Concat 一致）|
| **ListLength** | `List`(List) | `Result`(Number) | `len` |
| **ListGet** | `List`(List)、`Index`(Integer,默认0) | `Result`(`*`) | 越界（含负）→ **nil**（下游按自身缺省/零值处理，与"未设可选输入"同；不做负索引从尾）|
| **ListContains** | `List`(List)、`Value`(`*`) | `Result`(Bool) | 任一元素 `node.EqualAny(el, Value)`==true（**与 Eq 节点完全同语义**：同类型直比、跨类型串比，故 `Contains([1],"1")`==`Eq(1,"1")`）|
| **ListAppend** | `List`(List)、`Item`(`*`) | `Result`(List) | **返回新列表**：`out := append(append([]any{}, in...), Item)` —— **必 copy** 防 `append` 原地改写**上游 Evaluate 返回的切片**（底层数组别名）。注：元素**浅拷贝**，嵌套 map/list 引用仍共享（值语义，非 bug）|
| **ListSlice** | `List`(List)、`Start`(Integer,默认0)、`Count`(Integer,默认 **-1**) | `Result`(List) | **返回新列表**。`Count<0`→从 Start **取到末尾**（默认）；`Count==0`→**空**（可表达"取0个"）；`Count>0`→取 Count 个。Start clamp `[0,len]`、负 Start clamp 到 0；end=`min(Start+Count,len)`。copy 同 Append |

## D. RandomChoice（代码放 `random` 包，Category `Random`）

- Kind `RandomChoice`，`IsPureData:true + IsNonDeterministic:true`，底层 `math/rand/v2`。
- Input `List`(List)。Output `Result`(`*`)。
- 语义：`rand.IntN(len)` 均匀取一元素；**空列表→nil**（纯数据节点天然无 exec Fail 出口，无可选元素当正常值返 nil，与 ListGet 越界一致）。受阶段 1 per-dispatch 缓存覆盖（同求值多路径同值）。

## 设计判断（已拍，待复核）

- **Append/Slice 返回新列表**（纯函数）：pull 模型无可变持久列表；要累积改列表用 SetVar 存 + 每轮重写（ListAppend 本身**不累积**，无状态）。copy 理由 = 防别名上游切片（**非**防缓存——确定性节点不进缓存）。
- **ListContains/Join 复用导出的 `node.EqualAny`/`FormatValue`**，与 Eq/Concat/ToString 同一套，不另写。
- **元素不分类型**（`[]any`）；**ListGet/RandomChoice 越界/空→nil**（不做负索引、不做 error，YAGNI）。
- **ForEach 归 `nodes/control`（机制）+ Category `List`（palette）**；列表纯数据节点归 `collection` 包。
- **Slice/Substring 统一 Count/Length 约定**：负=取到末尾(默认)、0=空、正=N（[string spec](2026-06-10-string-nodes.md) 同步）。
- **新 `List` palette 分类**：前端注册全套同 [random-nodes spec](2026-06-10-random-nodes.md) Category 路径（GROUP_MAP/NodeGroup 联合类型/GROUP_LABEL/GROUP_I18N_KEY/visualRegistry + `nodeGroup.list`）。

## 非目标（YAGNI）

- 不做 **Filter/Map**（高阶，要谓词子区域/Expr 嵌入；ForEach+If+SetVar 能拼）。
- 不做 typed list（无泛型）；不做深拷贝元素；不做 List 进 Expr（`services/expr` 不认 `[]any`）。
- 不做 ListInsert/Remove/Sort/Reverse/Unique/IndexOf——第一批不铺满。

## 落地清单（按 add-node.md，要点）

1. **框架**：types.go 注册 List；inputs.go+interfaces.go 加 `List()`；data_pull.go `toExprValue` 加 `[]any` case + 确认 coerceToType；**导出 `node.EqualAny`/`FormatValue`**（purefunc 改调用，回归测试守 Eq/Concat 不变）；dispatch_v5.go `makeBodyFor` 加 ForEach。**先做 A' 审计**。
2. **control 包**：ForEach（RegionRunner）+ 测试（遍历/空/break/continue/Fail 路由/入口快照）。
3. **collection 包**：`internal/nodes/collection/` 7 纯数据节点 + 测试（含 Split 边界、ListSlice Count 负/0/正、Append 不改上游、ListContains==Eq）。10 处 blank-import 镜像 purefunc 加 `_ ".../collection"`。
4. **random 包**：加 RandomChoice + 测试（空→nil、缓存稳定）。
5. **i18n**：zh/en 加 ForEach + 7 collection + RandomChoice + `nodeGroup.list`；Split/边界写进 description；跑 `pnpm gen:node-i18n`。
6. **前端 List 分类**：GROUP_MAP/NodeGroup 联合类型/GROUP_LABEL/GROUP_I18N_KEY/visualRegistry 加 `list`（图标如 `i-tabler-list`）。
7. **验证**：`go build ./... && go test ./internal/...`；`pnpm gen:node-i18n && pnpm typecheck && pnpm i18n:check`；`task nodes` 见全部新节点；真机 smoke：Split→ForEach→Log 跑通、List 分组三处可见、SetVar 存列表 GetVar 读回正常。
