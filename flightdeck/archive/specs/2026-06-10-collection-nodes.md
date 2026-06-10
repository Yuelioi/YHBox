---
status: done
summary: 数组/集合子系统第一批 - List pin 类型 + in.List getter + ForEach + 8 个列表节点 (加节点路线图阶段2) — A' 审计过 gate 后已实现 (含 LooseEqual 不可比防护修正)
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

1. **`List` pin 类型**：`RegisterType(TypeSpec{Tag:"List", GoType:"[]any", WidgetKind:"list-preview", Color:"#818cf8"})`。元素**不分类型**（`[]any`，无泛型，跟 `JSON=map[string]any` 一路子）。
   - **widget 锁定（不留开放点）**：**只读预览**（新 `list-preview` widget 或现有 json widget 配只读），**不做手填 literal 编辑器**。理由：简化前端 + 缩小 A' 审计面（无 literal 持久化分支）。**列表来源** = Split / 节点 / 连线；要"手造列表"用 `Split("a,b,c")`，要空列表用 `Split("","")`。
2. **`in.List(name) []any` getter**（`node/inputs.go` + `interfaces.go`）：容忍 `[]any`（原样）、`[]string`（转 `[]any`，一次性分配、不缓存）、nil 及**其它任意类型 → nil**（不 panic）。**不**把裸 string 当一元列表（与 StringList 区别）。
3. **`toExprValue` 显式加 `case []any: return x`**（当前靠 default 能跑但脆，显式化 + 注释；纯加固）。确认 `coerceToType`（`data_pull.go:183`）对 List 走透传。
4. **导出等值/格式化 helper**：把 `purefunc` 的 `equalAny`/`formatValue` 提升为 **`node.LooseEqual(a,b any) bool` / `node.FormatValue(v any) string`**（放 `internal/node` 包——只依赖 fmt/strconv，无新 import、不形成循环依赖；purefunc 反过来调）。
   - **改名 `LooseEqual`**（非 EqualAny）：强调"宽松等值"，区别于 `expr` 包的 `valueEqual`（基本类型/Point 比较）；注释写明"仅 Eq/ListContains 用，同类型直比、跨类型 FormatValue 串比"。
   - **零行为变更纯搬移**：签名/逻辑（含 nil 处理）原样搬，不趁机改；回归测试守 Eq/Concat/ToString 输出不变。
   - → ListContains 用 `node.LooseEqual`（**语义 == Eq 节点**）、Join 用 `node.FormatValue`，全项目一套，杜绝漂移。
   - **nil 语义（源码核 equalAny）**：`LooseEqual(nil,nil)`=true（sameType 都 nil → `a==b`）；`LooseEqual(nil,"")`=false（FormatValue(nil)=`"null"` ≠ `""`）。故 `ListContains([nil], nil)`=true、`ListContains([nil],"")`=false。

## A'. 框架审计（⛔ **独立 gate 步骤 0，结论未出前不动框架代码** — 头号铁律 consumer-audit）

只验了数据线透传（toExprValue）**不够**。List 类型一旦能进变量/连线，下列链路都要逐一确认 `[]any` 能稳定携带，否则"存进变量的列表读出来碎了"类 bug（参 incident [[2026-05-29-storage-convention-consumer-audit-gap]]）。**这是独立前置步骤**：若发现阻塞（如前端 pin 不认 List），框架改动要回炉，故必须先于 A 节任何代码。

- **SetVar/GetVar**：var store `map[string]expr.Value`（`expr.Value=any`）→ `[]any` 能存（**低风险**），验 GetVar 读回不被压扁。
- **TickSnapshot**：`CaptureSnapshot` 的 `maps.Copy` 浅拷贝 → `[]any` 按引用共享（只读快照，**低风险**）。
- **⚠ Expr 读列表变量**：用户可在 Expr 里 `GetVar("mylist")` 拿到 `[]any` → `expr` 求值器（`AsNumber`/比较/算术）**不认 `[]any`**，会报错或出垃圾。**结论二选一**：(a) 本阶段不处理 → i18n/文档明告"列表变量不能进 Expr"，且确认 Expr 拿到 `[]any` 是**干净报错**（非 panic）；(b) 纳入修复（超 scope，倾向 a）。
- **持久化/编辑链路**（A.1 已锁**无 List literal** → 大幅缩面，但仍验）：graph JSON 导入导出、undo/redo、节点 clipboard copy/paste、Subgraph/CollapsedNode 参数传递——确认这些只搬 literal+结构、不碰 wire 值；**List 不作变量默认值/静态值槽**（无 literal 编辑器，天然规避）。
- **类型兼容矩阵**：`*`/Any pin 接 List 是否放行；List↔JSON 是否误自动转；连线校验是否因新类型意外放宽/收紧；`coerceToType` 无 List 误转。
- **catalog/RPC**：GetAllTypes 把 List 推前端后，pin 渲染/连线校验/分组色认不认。
- **validator**：无对 pin 类型的 exhaustive switch 因未知 List 报错。

**产出**：plan/impl 前出一页"List 链路审计"结论；命中阻塞回炉。

### A' 审计结论（2026-06-10，三路并行审计完毕 — **gate 通过，零架构阻塞**）

**后端链路（全 PASS + caveats）**：var store（SetVar/GetVar 原样存取 `[]any`，runtime_context.go:120-125）、TickSnapshot 浅拷贝引用共享只读、`node.Capture` 无 coercion、`toExprValue`/`coerceToType` default 透传、库导出只搬结构、`dumpValue` 对 `[]any` 走 `json.Marshal` —— 全链路 OK。Caveats：
- **⛔→设计修正：LooseEqual 必须加不可比类型防护**。`equalAny` 的 `sameType → a == b` 对 slice/map 动态类型是 Go 运行时 panic（被 EvaluatePureData recover 成 error → 再被数据线路径吞成 nil = 静默错值）。提升时改为：同类型且**可比**→直比；同类型但不可比（slice/map）→ FormatValue 串比。这是对"零行为变更纯搬移"的唯一修正（原行为=panic，无保留价值），ListContains 遇嵌套元素必踩。
- IncVar 对 list 变量：`AsNumber`→0→静默改写为数字（GIGO，与框架"宽容非法输入"惯例一致，不修，文档化）。
- Expr 读 list：全运算路径**干净 error 无 panic**（eval.go 各 AsNumber gate）；`AsBool([]any)`=true（恒真）；error 被 `buildDataWireFor` 吞（dispatch_v5.go:58，预存行为）。按选项 (a)：文档化"列表不进 Expr"。
- `canonPinType` 靠 lowercase fallback 碰巧得 "list"——显式加 case 加固。
- CaptureType：ForEach 用 `"any"`（spec_capture_test 白名单已含，不扩词表）。
- 后端 `PinTypeCompat`：List→List 同型放行、List→Number 正确拒、List→`*` 放行 ✓。

**前端类型矩阵（2 个必改站点，纳入落地清单）**：
- **闭合词表三连**：`nodeRegistry/index.ts` 的 `PinType` 联合 + `TYPE_COLOR` 静态表（画布 pin 用它，**不是** RPC 色表）+ `adapter.ts::backendTypeToPinType`（现在 List→default `'any'` 静默降级灰色）——三处必须加 `'list'`（色 #818cf8）。`pinTypeCompat` 顺手扩（注：它是**死代码**，vue-flow 未接任何连线类型门禁——预存缺口，本批不接）。
- widget `list-preview` 未注册会回退成可编辑文本框（能手输垃圾 literal）→ 实现为**只读"由连线提供"占位**（PinInput/PinLiteral 对 list 型分支）。
- ExpressionInput `expectedType` 无 list——优雅降级，不改。

**持久化/编辑链（全 PASS + 2 个词表 caveat，均 YAGNI 文档化不扩）**：
- graph JSON / undo / clipboard / 库导出只搬结构+literal，wire 值不落盘 ✓。
- 子图参数类型词表无 "list"：List 进子图须把参数类型设 **any**（typed 参数 PIN_TYPE_MISMATCH 正确拒绝）。
- VarDecl/VarType 词表无 "list"：存列表的变量声明 **any** 型（SetVar 无类型门禁照存；VarRow JSON.stringify 显示正常）。

## B. ForEach（RegionRunner，放 `nodes/control` 包）

> 与 Loop 同为 RegionRunner、同包（`nodes/control`，遵循机制归类，不混进纯数据 collection 包）。Category 标 `List`（palette 与列表节点同组）。

- Kind `ForEach`，Category `List`。
- Inputs：`In`(Exec)、`List`(List)、`CaptureItem`(capture, CaptureType `any`)、`CaptureIndex`(capture, CaptureType `number`)。
- Outputs：`Body`(Exec)、`Done`(Exec)、`Fail`(Exec, Semantic error, Data: Error/Code)。
- `RunRegion`：`items := in.List("List")`（ForEach 自身 dispatch 内**入口取一次**——若上游是非确定节点如 RandomChoice，阶段1 per-dispatch 缓存保证它在 ForEach 求值内同值，故列表对整轮循环稳定）；`for i, el := range items { Capture(CaptureItem,el); Capture(CaptureIndex,i); body(ctx) }`；Break→Done、Continue→下一轮（照 Loop sentinel）。`in.List` 对非列表/nil → 空切片 → 0 轮 → 直接 Done（不算错、不 panic）。
  - **快照仅冻结切片头**：`items` 是 `[]any` 的引用头；元素若是 map/子 list，body 改其内容后续轮可见（照 Go for-range，**非深快照**）。
  - **dispatch 边界 / 缓存生命周期**：每轮 body 经 `runRegionBody` → 每个 body exec 节点走 `dispatchInRegion`、**各自新建 TickSnapshot+evalCache**（不跨轮共享）。ForEach 的 `in.List` 在 ForEach 自身那个 dispatch 解析；body 的缓存独立、逐节点 dispatch。
- **Fail**：复用框架 region 错误路由（见源码事实），ForEach 不自己 fire。
- 框架：`makeBodyFor` switch 加 `case "ForEach": return r.makeBodyForForEach(node, tok), nil`，实体 = `seeds := r.edges.next(node.ID+".Body", tok.LoopStack); r.runRegionBody(c.Context(), seeds)`（与 `makeBodyForLoop` 逐行同构）。
- **注**：`CaptureIndex` 是 `number`（float64，与 Loop 既有一致——保持一致不改）；下游 `in.Int` 宽松转回 int 安全；Expr 里做 index 运算得 float（既存现象，非本节点引入）。

## C. 列表节点（纯数据，新 `internal/nodes/collection` 包，Category `List`）

> 7 个，全 `IsPureData:true` + Evaluator。底层 `strings` + `node.LooseEqual`/`node.FormatValue`。

| 节点 | 输入 | 输出 | 语义 |
|---|---|---|---|
| **Split** | `Text`(String)、`Separator`(String,默认`,`) | `Result`(List) | 元素为 string。**边界（刻意，写进 i18n）**：`Text==""`→**空列表**（刻意偏离 Go `strings.Split("",sep)` 的 `[""]`，更直觉）；`Separator==""`→等价 `strings.Split(s,"")`（UTF-8 字符/rune 边界、每字符一元素，非字节非字素簇）；`Text=="" && Sep==""`→空列表；否则 `strings.Split` |
| **Join** | `List`(List)、`Separator`(String,默认`,`) | `Result`(String) | 各元素 `node.FormatValue` 后 `strings.Join`（复杂元素 map/list 的串化由 FormatValue 统一定，与 ToString/Concat 一致）|
| **ListLength** | `List`(List) | `Result`(Number) | `len` |
| **ListGet** | `List`(List)、`Index`(Integer,默认0) | `Result`(`*`) | 越界（含负）→ **nil**（下游按自身缺省/零值处理；不做负索引）。**nil 歧义（写进 i18n）**：返回 nil 可能是元素本身=nil 也可能越界——要区分先 ListLength 查长度 |
| **ListContains** | `List`(List)、`Value`(`*`) | `Result`(Bool) | 任一元素 `node.LooseEqual(el, Value)`==true（**与 Eq 节点完全同语义**：同类型直比、跨类型串比，`Contains([1],"1")`==true；含 nil 元素见 A.4 nil 语义）|
| **ListAppend** | `List`(List)、`Item`(`*`) | `Result`(List) | **返回新列表**：`out := append(append([]any{}, in...), Item)` —— **必 copy** 防 `append` 原地改写**上游 Evaluate 返回的切片**（底层数组别名）。**浅拷贝**（写进 i18n）：嵌套 map/子 list 与原列表共享引用，同 Python `list.copy()`（值语义，非 bug）|
| **ListSlice** | `List`(List)、`Start`(Integer,默认0)、`Count`(Integer,默认 **-1**) | `Result`(List) | **返回新列表**。`Count<0`（任意负数）→从 Start **取到末尾**（默认）；`Count==0`→**空**（可表达"取0个"）；`Count>0`→取 Count 个。**`Start≥len`→恒空（Count 忽略）**；Start clamp `[0,len]`、负 Start→0；end=`min(Start+Count,len)`。copy 同 Append。负数=取到末尾是产品决策，写进 i18n |

## D. RandomChoice（代码放 `random` 包，Category `Random`）

- Kind `RandomChoice`，`IsPureData:true + IsNonDeterministic:true`，底层 `math/rand/v2`。
- Input `List`(List)。Output `Result`(`*`)。
- 语义：`rand.IntN(len)` 均匀取一元素；**空列表→nil**（纯数据节点天然无 exec Fail 出口，无可选元素当正常值返 nil，与 ListGet 越界一致）。受阶段 1 per-dispatch 缓存覆盖（同求值多路径同值）。**nil 歧义同 ListGet**（选中元素本身=nil 与空列表都返 nil，写进 i18n：要区分先 ListLength 查）。

## 设计判断（已拍，待复核）

- **Append/Slice 返回新列表**（纯函数）：pull 模型无可变持久列表；要累积改列表用 SetVar 存 + 每轮重写（ListAppend 本身**不累积**，无状态）。copy 理由 = 防别名上游切片（**非**防缓存——确定性节点不进缓存）。
- **ListContains/Join 复用导出的 `node.LooseEqual`/`FormatValue`**（改名避开与 expr `valueEqual` 混淆），与 Eq/Concat/ToString 同一套，不另写。
- **元素不分类型**（`[]any`）；**ListGet/RandomChoice 越界/空→nil**（不做负索引、不做 error，YAGNI）。
- **ForEach 归 `nodes/control`（机制）+ Category `List`（palette）**；列表纯数据节点归 `collection` 包。
- **Slice/Substring 统一 Count/Length 约定**：负=取到末尾(默认)、0=空、正=N（[string spec](2026-06-10-string-nodes.md) 同步）。
- **新 `List` palette 分类**：前端注册全套同 [random-nodes spec](2026-06-10-random-nodes.md) Category 路径（GROUP_MAP/NodeGroup 联合类型/GROUP_LABEL/GROUP_I18N_KEY/visualRegistry + `nodeGroup.list`）。

## 非目标（YAGNI）

- 不做 **Filter/Map**（高阶，要谓词子区域/Expr 嵌入；ForEach+If+SetVar 能拼）。
- 不做 typed list（无泛型）；不做深拷贝元素；不做 List 进 Expr（`services/expr` 不认 `[]any`）。
- 不做 ListInsert/Remove/Sort/Reverse/Unique/IndexOf——第一批不铺满。

## 落地清单（按 add-node.md，要点）

0. **⛔ A' 框架审计（gate）**：先出 List 链路审计结论（尤其 Expr 读列表变量是否干净报错、前端 pin 认不认 List、类型兼容矩阵），命中阻塞回炉。**结论未出不开工框架代码。**
1. **框架**：types.go 注册 List（`list-preview` 只读 widget）；inputs.go+interfaces.go 加 `List()`（非列表→nil）；data_pull.go `toExprValue` 加 `[]any` case + 确认 coerceToType；**导出 `node.LooseEqual`/`FormatValue`**（**零行为变更纯搬移**，purefunc 改调用，回归测试守 Eq/Concat/ToString 输出逐字不变）；dispatch_v5.go `makeBodyFor` 加 ForEach。
2. **control 包**：ForEach（RegionRunner）+ 测试（遍历/空/非列表/break/continue/Fail 路由/入口快照仅冻结切片头）。
3. **collection 包**：`internal/nodes/collection/` 7 纯数据节点 + 测试（含 Split 边界、ListSlice Count 负/0/正 + Start≥len、Append 不改上游、ListContains==Eq、**含 nil 元素的 Contains/Get**）。blank-import **镜像 purefunc 的全部站点**（凡需注册 collection 节点的 binary/测试，按原则不写死数量）加 `_ ".../collection"`。
4. **random 包**：加 RandomChoice + 测试（空→nil、含 nil 元素、缓存稳定）。
5. **i18n**：zh/en 加 ForEach + 7 collection + RandomChoice + `nodeGroup.list`；Split/边界写进 description；跑 `pnpm gen:node-i18n`。
6. **前端 List 分类**：GROUP_MAP/NodeGroup 联合类型/GROUP_LABEL/GROUP_I18N_KEY/visualRegistry 加 `list`（图标如 `i-tabler-list`）。
7. **验证**：`go build ./... && go test ./internal/...`；`pnpm gen:node-i18n && pnpm typecheck && pnpm i18n:check`；`task nodes` 见全部新节点；真机 smoke：Split→ForEach→Log 跑通、List 分组三处可见、SetVar 存列表 GetVar 读回正常。
