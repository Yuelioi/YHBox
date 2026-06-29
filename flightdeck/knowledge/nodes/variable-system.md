# 变量系统 (Variable System)

SUMMARY: 容器级命名值存储全貌 —— VarDecl 类型、scope（local/global/auto）、VarStore 读写、快照、变量节点、config.capture 存储与校验
READ WHEN: 加变量类型 / 加变量节点 (GetVar/SetVar/IncVar/VarLastChange/输出捕获) / 改变量引用存储或读写法 / 撞 'config 有 varName 但 widget 空 / runtime missing VarName' 类 bug 前
RECHECK WHEN: 加变量类型 / 加或改变量节点 (GetVar/SetVar/IncVar/VarLastChange/输出捕获) / 改变量引用存储或读写法 / 改 scope 规则时

---

## 一句话

变量 = **容器级的命名值存储**。用户在容器上声明若干变量（`VarDecl`），运行时由 `VarStore` 服务读写；节点（GetVar/SetVar/IncVar/VarLastChange/输出捕获）、表达式（`$hp`）、脚本（`$hp` / GetVar·SetVar 节点函数）三条路都走同一个 `VarStore`。源码：声明结构 `internal/services/container/model.go`、运行时存储+读写 `internal/services/container/runtime/`、变量节点 `internal/nodes/variable/`。

配套：表达式里怎么用 `$` → [expression-system.md](expression-system.md)；脚本里怎么用 → [script-system.md](script-system.md)；变量节点的 pin 全表跑 `task nodes`。

## 1. 声明 — VarDecl

容器存一张变量声明表（`Container.Vars []VarDecl`，`model.go`）：

```go
type VarDecl struct {
    Name    string // 变量名
    Type    string // 类型 tag (见下)
    Default any    // 初值, 已是对应 Go 形态
}
```

**6 种类型**（`Type` 合法值）+ 其 Default 形态：

| Type | Go 形态 | Default 例 | 备注 |
|---|---|---|---|
| `number` | `float64` | `0` / `3.14` | |
| `string` | `string` | `""` | |
| `bool` | `bool` | `false` | |
| `point` | `map[string]float64{x,y}` | `{x:0,y:0}` | ratio 坐标 |
| `list` | `[]any` | `[]` | 异构列表（2026-06-10 加）；前端走 JSON 数组输入框，非法 JSON / 非数组不落盘 |
| `any` | 任意 | `null` | 不限类型；编辑器是裸文本框（手填 `[1,2]` 会存成**字符串**，要 list 请选 `list` 类型） |

启动时 `RuntimeContext.initVars()` 把每个 `v.Default` 原样塞进 `rt.vars`（Default 已是正确 Go 形态，list 的 `[]any` 直接放，无需转换）。

## 2. 作用域 — scope（local / global / auto）

GetVar/SetVar/IncVar 都有一个 `Scope` 输入（缺省 `"auto"`）。三种值：

- **`global`** — 容器全局变量，存 `RuntimeContext.vars`（`map[string]expr.Value`）。
- **`local`** — 当前帧的局部变量，存 `ExecState.LocalVars`（per-frame 栈，子图 / Loop body 内部用）。frame-private，编辑期跳过全局存在性校验（运行期才解析）。
- **`auto`**（默认） — 按下面规则在 local 链和 global 间自动选。

**auto 解析规则**（`runtime/node_services.go` 的 `varStoreAdapter`）：

- **读 `GetScoped(name,"auto")`**：先查本帧及父帧的 LocalVars 链（`GetLocalVarChain`）→ 命中即用；都没有 → 兜底查 global（`rt.Vars()`）。
- **写 `SetScoped(name,"auto")`**：本帧 LocalVars 已有该名 → 改 local；否则 → 写 global（即"新建/改全局"）。

> 直觉：auto 让"在子图里读到的还是外层那个变量"，除非子图自己声明了同名 local。

## 3. 存储与读写 — VarStore

运行时三块存储：

- `RuntimeContext.vars` —— 全局 KV（`map[string]expr.Value`）。
- `ExecState.LocalVars` —— 局部栈（子图/Loop 帧私有）。
- `RuntimeContext.varTimestamps` —— 每个变量最后改写的 unix 毫秒（`SetVar` 时更新）。

`VarStore` 接口（节点经 `ctx.Vars()` 拿到，实现 = `varStoreAdapter`）：

| 方法 | 语义 |
|---|---|
| `Get(name)` / `GetScoped(name,scope)` | 读，返 `(value, ok)` |
| `Set(name,v)` / `SetScoped(name,scope,v)` | 写；副作用：刷新 `varTimestamps[name]` |
| `Inc(name,d)` / `IncScoped(name,scope,d)` | 自增，返新值；当前值非 number 视为 0 起步（GIGO） |
| `LastChange(name)` | 取该变量最后改写时间戳（毫秒，未设过返 0）；**live 语义，不受快照影响** |

### 快照 (Snapshot) —— PureData 读的是 tick 冻结视图

PureData 节点（GetVar 等 Evaluator）在 `EvaluatePureData` 入口被 wrap 进一份 **tick 快照**（`runtime/snapshot.go` 的 `TickSnapshot`，对 `rt.Vars()` 浅拷贝）。这保证同一次派发内多个 data 节点读到一致的变量状态。

- **读**走快照（一致）；**写**（SetVar）直接落 `rt.vars`，下一个 exec 节点的新快照才看得到。
- `LastChange` 例外 —— 直读 `varTimestamps`，是 live 值（Fishing v2 watchdog 靠它算"变量多久没变" = `Now − VarLastChange`）。
- 为什么用 wrap 而非给 Ctx 加方法，见 [framework-extension-dispatch-context.md](framework-extension-dispatch-context.md)。

## 4. 变量节点

| Kind | capability | 关键 pin |
|---|---|---|
| `GetVar` | **Evaluator**（pure-data） | in: `VarName`(req) / `Scope`(=auto) → out: `Value`(`*`) |
| `SetVar` | **Runnable**（exec） | in: `VarName`(req) / `Scope`(=auto) / `Value`(`*`,req) → out: `Done` |
| `IncVar` | **Runnable**（exec） | in: `VarName`(req) / `Scope`(=auto) / `Delta`(Number,=1) → out: `Done` |
| `VarLastChange` | **Evaluator**（pure-data） | in: `VarName`(req) → out: `Value`(Number, 毫秒) |

实现都在 `internal/nodes/variable/`，逻辑就是"读 pin → 调对应 `ctx.Vars()` 方法"。GetVar 读不到不报错、返 nil。

### 输出捕获 (config.capture) —— 产出型节点把出口值写进变量

很多节点（Capture / Script / 模板匹配类）的 exec 出口带 `OutputSpec.Data` 字段。用户可在 Inspector 的「输出」组把某个 Data 字段绑定到变量；节点 fire 时框架把本次出口实际带的字段写进该变量。

- 声明：节点只在 exec 出口声明 `OutputSpec.Data []DataField`；不再给每个字段加 `Capture<DataField>` 输入 pin。
- 存储：绑定关系写进节点 `config.capture`，形状是 `map[字段名]变量名`。
- 执行：`dispatch_v5.applyCaptures` 在 fire 时读取 `config.capture`，只写该出口实际 `.Set()` 过的字段；空变量名 no-op，非空则 `Vars().SetScoped(name,"auto",value)`（**scope 固定 auto**）。
- 校验：`validator_capture_refs.go` + `nodepkg.BindableFieldsForNode` 校验字段和变量引用；动态输出节点会把 `config.Outputs[]` 并入可绑字段。
- 直连：如果只是把产出喂给紧随或后续节点，优先用 held-output 数据线直连；需要命名复用 / 跨作用域时再捕获到变量。

## 5. 变量引用怎么存（config）+ 校验

GetVar/SetVar/IncVar 的 `VarName` / `Scope` 存在节点 config 里：

```json
{ "VarName": "hp", "Scope": "auto" }
```

- **取值优先级**：`PinValue(node,"VarName")` 先读 `config.literal["VarName"]`，fallback 顶层 `config["VarName"]`（`pin_value.go`）。**画布正源是 `config.literal[pin]`**，顶层只是读取兜底。
- **编辑期校验**（`validator_var_refs.go`，码 `INVALID_VAR_REF`）：Kind ∈ {GetVar,SetVar,IncVar} 且 Scope ∈ {auto, global, 空} 时，若 `VarName` 既不在 `Container.Vars[].Name` 也不在子图 `RequiredGlobals[].Name` → 红错。`Scope=="local"` 跳过（帧私有，只运行期查）。

**常见 bug 根因**：

- *"config 有 varName 但 widget 显示空"* —— 值落在顶层 `config` 而编辑器只读 `config.literal`（旧数据形态），或反之。正源永远是 `config.literal`。
- *"runtime missing VarName"* —— `VarName` pin 既没连线也没填 literal，framework 返 validation error。
- *"list 变量手填成字符串"* —— 用了 `any` 类型（裸文本框）手打 `[1,2,3]` 存成字符串；要真 list 选 `list` 类型（有专用 JSON 数组编辑框 + `parseListDraft` 校验）。

## 6. 表达式 / 脚本里的变量（连接点）

- **Expr**：`$名字` 直读容器变量，固定 **auto** scope，快照语义同 GetVar（lexer `tkVarRef` → eval 走 `env.Get("$"+name)` 前缀通道）。细节 [expression-system.md](expression-system.md)。
- **Script**：`$名字` 是 **live getter**（起跑时给每个已知变量注入 accessor，中途 set 后再读是新值）；动态新建的变量没 getter，用 `GetVar({VarName})` 节点函数读。细节 [script-system.md](script-system.md)。

## 7. List 变量类型 — 消费点审计（2026-06-10 落地）

加 `list` 类型时审计过"哪些地方消费变量类型"，当前都已覆盖（历史材料在 cold archive `2026-06-10-list-var-type`）：

| 消费点 | 位置 | List 处理 |
|---|---|---|
| `VarDecl.Type` 枚举 | `model.go` | `list` 在 6 种内 |
| 类型兼容 (Go) | `runtime/pin_types.go` `PinTypeCompat` | list→list 通、list→any 通、list→其它拒 |
| 类型兼容 (TS) | `nodeRegistry/index.ts` | 同上一致 |
| data-pin 类型校验 | `validator_pin_types.go` | List 变量 → List pin 自动放行 |
| 运行时初值 | `runtime_context.go::initVars` | `[]any` Default 原样塞 |
| 前端类型联合/零值/选项 | `lib/variableRef.ts` | `list` 在集合内，零值 `[]`，下拉有 option |
| 默认值编辑器 | `VarRow.vue` | list 走 JSON 数组输入框 + `parseListDraft` |
| `node.LooseEqual`/`FormatValue` | `value_helpers.go` | `[]any` 走 `fmt.Sprintf("%v")` 串比 |
| List pin 读取 | `inputs.go::in.List` | 容忍 `[]any`/`[]string` |
| 类型注册 | `types.go` | `{Tag:"List", GoType:"[]any", WidgetKind:"list-preview", Color:"#818cf8"}` |
| 拖拽自动连线 | `useNodeCreation.ts` | list 变量拖近 List pin 自动连 |
| 提升/新建变量 modal | `PromoteToVarModal` / `NewVarModal` | list option + 零值 `[]` |

**故意不支持**：`IncVar` 对 list 变量运行时当 0（类型不匹配，GIGO）。输出捕获没有旧 `CaptureType` 白名单，字段类型来自 `OutputSpec.Data` / 动态输出声明。
