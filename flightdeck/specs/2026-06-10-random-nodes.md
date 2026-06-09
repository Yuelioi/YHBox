---
status: active
summary: 随机数节点 RandomInt/RandomFloat/RandomBool + per-dispatch 求值稳定 (加节点路线图阶段1)
last_updated: 2026-06-10
---

# 随机数节点（加节点路线图 · 阶段 1）

## 背景与定位

用户要"再加几个节点"，方向定为**补通用框架 building block 缺口**（非绑具体场景）。从 69 节点目录看，最扎眼的通用缺口是 IO、字符串、集合、随机。用户拍板分阶段做，本 spec 是**阶段 1：随机数**。

经一轮外部 AI 审核（3 家，未看完整项目，仅参考），phase 1 范围确定包含**一处框架小改动**（per-dispatch 求值稳定），故本 spec 分两部分：A 框架机制、B 随机节点。

### 加节点路线图（用户排序，2026-06-10）

> 排序约束：① 实际需求 ② 成本 ③ 依赖（数组要先有 List 类型，是框架改动）。用户决定「数组提前」。

1. **阶段 1 — 随机数 + per-dispatch 求值稳定**（本 spec）。
2. **阶段 2 — 数组/集合**（提前）：先给框架加 `List` pin 类型 + 遍历（ForEach region runner），再做 Split/Join/ListGet/ListLength/ListAppend/ListContains/RandomChoice/Filter/Map。
3. **阶段 3 — 数学补全**：Clamp/Round/Floor/Ceil/Pow/Sqrt/Abs/Min/Max。长尾函数顺手塞 Expr 的 `evalCall`。
4. **阶段 4 — 字符串函数**：Replace/Substring/Trim/ToUpper/ToLower/IndexOf/StartsWith/EndsWith/RegexMatch/RegexExtract。

每阶段独立 spec→plan→impl。

## 已验证的源码事实（设计依据）

- **纯数据求值无结果缓存**：`runtime/data_pull.go::evalDataSource` 每次 pin pull 都重调 `EvaluatePureData`；`node/engine.go::EvaluatePureData` 内部直接跑 `rn.Evaluate`、无记忆化。
- **框架已有「Determinism contract」**：`runtime/snapshot.go` 的 `TickSnapshot` 在 `dispatchInRegion` 入口（`dispatch_v5.go:328` `withTickSnapshot(ctx, CaptureSnapshot(...))`）冻结 Vars，挂在 ctx（`tickCtxKey`）上，保证同一 exec tick 内 GetVar 读到一致值。`TickSnapshot` 是 **per-消费节点-dispatch** 的可变对象（每个 exec 节点 dispatch 时新建一个）。
- **pure-data 不走 `dispatchInRegion`**：纯数据节点经 `evalDataSource`→`EvaluatePureData` 求值，**不**各自换快照 → 一个消费 exec 节点的整棵 pure-data 拉取树共享它那一个 `TickSnapshot`，且单 goroutine 顺序拉取（并发分支各有独立 ctx chain）。
- **Expr 已有 `abs/min/max/now`、无 `random`**（`services/expr/eval.go::evalCall`）。注：`now()` 用 `time.Now()`，本身非确定 —— 见本 spec「非目标」对它的处置。
- **jitter 用 `math/rand/v2`**（`node/jitter.go`），`jitterSamples=5` CLT 求均值。`normalJitterFactor(pct)` 返回**有符号 `[-p,+p]`、中心 0** 的因子（base±pct% 形态），**非区间形态**，不能复用于区间随机。
- **dropdown widget 存在**：`WidgetSpec{Kind:"dropdown", Props: node.MarshalProps(node.DropdownProps{Options:[]node.EnumOption{{Value:"..."}}})}`（`detect/check_template.go:43` MatchMode、`variable/get_var.go` scope）。
- **纯函数节点模式**：`nodes/purefunc/purefunc.go::specBuilder(kind, inputs, resultType)` 构造单 `Result` 出口的 `IsPureData` Spec + `Evaluate(ctx,in)(any,error)`。**全 22 个纯函数节点输出 pin 名硬编码为 `Result`** —— `Result` 是既定约定。`Integer` 类型 + `in.Int(name)` getter 可用。
- **新 Category 的前端注册点**（`checklists/add-node.md §6`）：`Category` 字符串 → `NodeGroup` 经 `adapter.ts::GROUP_MAP`（`{Control:'control', Detect:'detect', ...}`）；分组标签 i18n 经 `NodePalette.vue::GROUP_LABEL` + `useNodeGroupColor.ts::GROUP_I18N_KEY`，都要指向真实存在的 `nodeGroup.<key>`；分组视觉（图标+色）从 `visualRegistry` 派生。**面板无白名单** —— 三处选择器（NodePalette/NodeExplorerModal/InlineContextMenu）只按 `excludeFromPalette` 过滤，普通节点自动显示。
- **无 `gen:node-types` 脚本**（`frontend/package.json` 只有 `gen:node-i18n`）—— 前端按后端 Spec 经 RPC 动态渲染（`add-node.md §4`），无 TS 类型生成环节。

## A. 框架机制 — per-dispatch 求值稳定

### 问题

随机节点 `Evaluate` 非确定。框架每次 pin pull 都重算（无缓存）→ **一个消费 exec 节点经多条路径拉到同一随机节点，各路径拿到不同值**（reviewer 3 家共同 flag 的高频陷阱；也违背框架既有 Determinism contract）。

### 方案（用户拍：方案 1，最小 blast radius）

给非确定节点纳入 Determinism contract：**同一 dispatch 内对同一非确定节点只求值一次，结果记忆化**。

1. **`node.Spec` 加字段 `IsNonDeterministic bool`**（默认 false）。随机三节点置 `true`。确定性节点（Add/Expr/GetVar/...）不动，行为可证明 100% 不变。
2. **`TickSnapshot` 加 per-dispatch 求值缓存**（`runtime/snapshot.go`）：
   ```go
   type TickSnapshot struct {
       Vars      map[string]expr.Value
       evalCache map[string]evalEntry // lazy; key = srcNodeID+"\x00"+srcPin
   }
   type evalEntry struct { v expr.Value; err error }
   ```
3. **`evalDataSource`（`data_pull.go`）gate 缓存**：求值前，若 `rn.Spec.IsNonDeterministic` 且 `tick := tickFromCtx(ctx)` 非 nil → 查 `tick.evalCache[key]`，命中即返；未命中正常 `EvaluatePureData` 后写入。确定性节点完全跳过缓存路径。
   - **cold path**（listener config init 传 `context.Background()`，无 tick）→ `tick==nil` → 不缓存，正常求值。无碍（随机节点不会出现在 listener config 静态求值里；即便出现也只是退化为每次新值）。
4. **并发安全**：`TickSnapshot` 是 per-dispatch、per-goroutine ctx 作用域（`snapshot.go` 注释明确），单 dispatch 的 pure-data 拉取树顺序执行 → 普通 map 无需 mutex。impl 时复核此不变量（若未来出现并发拉取再加锁）。

### 语义结果

- 同一 exec 节点求值内，同一随机节点多路径引用 → **同值**（治本陷阱）。
- 不同 exec 节点 dispatch（不同时刻 fire）→ 各自新快照 → 新随机值（命令式 exec 流下这是正确语义，非 bug）。

## B. 随机节点（3 个）

### 包与归类

- 新建 `internal/nodes/random/` 包（语义独立；后续 RandomChoice 归此）。**新 `Category: "Random"`**（用户拍）。机制上 `IsPureData: true` + `IsNonDeterministic: true` + 实现 `Evaluator`。
- 底层 `math/rand/v2`（与 jitter.go 一致，自带 seed、并发安全）。

### 节点清单

| 节点 | 输入 | 输出 | 语义 |
|---|---|---|---|
| **RandomInt** | `Min`(Integer,0)、`Max`(Integer,100)、`Distribution`(dropdown: `uniform`默认/`centered`) | `Result`(Integer) | **闭区间** `[Min,Max]`（两端含）。`Min>Max` 自动交换；`Min==Max` 返回 Min |
| **RandomFloat** | `Min`(Number,0)、`Max`(Number,1)、`Distribution`(dropdown: `uniform`/`centered`) | `Result`(Number) | **半开** `[Min,Max)`；但 Max 概率测度为 0、实际不可达，用户可把两者都当 `[Min,Max]` 理解。`Min>Max` 自动交换；`Min==Max` 返回 Min |
| **RandomBool** | `Prob`(Number,0.5) | `Result`(Bool) | true 概率=Prob；`Prob≤0` 恒 false、`≥1` 恒 true（夹紧）。**无 Distribution**（二值无意义）|

### Distribution（分布选项）

> ⚠ **不叫 "bell"**：实现是 Bates 分布（n 个 uniform 求均值），近钟形但**非高斯**、值域有界、不可调标准差。命名 `centered`（聚中），i18n 描述写明"向区间中点聚集，非正态，不可调集中度"，避免用户按高斯预期误用。

- `uniform`：`rand.Float64()` 铺满区间。
  - RandomFloat：`Min + rand.Float64()*(Max-Min)`
  - RandomInt：`Min + rand.IntN(Max-Min+1)`（`IntN` 等宽桶、无偏）
- `centered`：helper `bellUnit() float64` —— 取 `randomSamples=5` 个 `rand.Float64()` 求均值（Bates，向 0.5 聚集），缩放到区间。
  - RandomFloat：`Min + bellUnit()*(Max-Min)`
  - RandomInt：`clamp(Min + int(bellUnit()*float64(Max-Min+1)), Min, Max)`
    - **clamp 是纯防御**：`rand/v2 Float64()∈[0,1)` → 均值 `<1` 严格成立 → `int(bellUnit()*(N))` 落 `[0,N-1]`、等宽桶无偏，clamp 实际永不触发。显式写**双端** clamp，注释说明"防越界纯防御，勿当冗余删除"（防实现者乱优化，也防未来 rand 实现变动）。
  - **不复用** `normalJitterFactor`（base±pct%，非区间）；`bellUnit` 内 `randomSamples` 注释交叉引用 `node/jitter.go` 的 `jitterSamples`（同为 5，独立维护，避免跨包耦合一个常量）。

### 边界与数值

- `Min>Max`：交换后照常（宽容，不报错）。
- 内部用 `int64` 算 `Max-Min+1`，避免普通 int32 满区间溢出。极端满 int64 区间出 scope（YAGNI：自动化区间都很小）。
- `NaN/Inf` 输入（如上游 Div 除零返 `NaN` 流入）：透传（GIGO，与 Div 自身返 NaN 一致），不做特殊校验（YAGNI）。
- `Prob` 越界自动夹紧 `[0,1]`（`rand.Float64()<Prob` 自然覆盖）。

### 多下游语义（核心，非脚注）

见「A. 框架机制」。**节点 i18n 描述必须写明**该节点已纳入 per-dispatch 稳定：同一求值内多处引用同值；跨不同 exec 节点是不同值。（不再作为可选脚注。）

## 全链路落地清单（按 checklists/add-node.md）

1. **框架**：`node.Spec` 加 `IsNonDeterministic`；`runtime/snapshot.go` 加 `evalCache`；`data_pull.go::evalDataSource` 加 gate；单测：同 dispatch 内同随机节点两路径取值相等、跨 dispatch 不等、确定性节点不受影响。
2. **后端节点**：`internal/nodes/random/{random.go, random_test.go}` —— 3 节点 + `init()` 注册 + `bellUnit`。`main.go` + `dispatch_v5_test.go` 加 `_ "yotta/internal/nodes/random"` blank import。
3. **新 Category 注册**：`adapter.ts::GROUP_MAP` 加 `Random:'random'`；`NodeGroup` 联合类型加 `'random'`；`NodePalette.vue::GROUP_LABEL` + `useNodeGroupColor.ts::GROUP_I18N_KEY` 加 `random` → 真实 `nodeGroup.random`；`visualRegistry` 给 `random` 配图标+色；zh.ts/en.ts 加 `nodeGroup.random`。
4. **i18n**：zh.ts/en.ts 加 `node.RandomInt/RandomFloat/RandomBool`（label/description/各 pin label + `input.Distribution.option.uniform|centered`），zh/en 对称。跑 `cd frontend && pnpm gen:node-i18n`。
5. **验证（全绿）**：
   - `go build ./... && go test ./internal/nodes/... ./internal/node/... ./internal/catalog/... ./internal/services/container/... -count=1`
   - `cd frontend && pnpm gen:node-i18n && pnpm typecheck && pnpm i18n:check`
   - `task build`
   - 真机 smoke：侧边面板 + 右键菜单 + explorer 三处都能找到 Random 分组 + 3 节点，分组标签/配色/默认值/文案对。

## 非目标（YAGNI）

- 不做 Seed/可复现（增状态复杂度，无 demand）。
- 不做 RandomChoice（需 List 类型，归阶段 2）。
- 不做正态/泊松等更多分布（uniform + centered 够用）。
- **不碰 `now()` / 不做全 pure-data 记忆化**：`IsNonDeterministic` 只标随机三节点；Expr 含 `now()` 的非确定行为保持现状（不在本 spec 范围；若要同样纳入 Determinism contract，另开 spec）。
- 不做整体 pure-data 求值缓存的性能优化（方案 2 已否，避免改框架不变量 + 波及确定性测试）。
