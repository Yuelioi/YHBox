---
status: active
summary: 随机数节点 RandomInt/RandomFloat/RandomBool（加节点路线图阶段1，含 uniform/bell 分布选项）
last_updated: 2026-06-10
---

# 随机数节点（加节点路线图 · 阶段 1）

## 背景与定位

用户要"再加几个节点"，方向定为**补通用框架 building block 缺口**（非绑具体场景）。从 69 节点目录看，最扎眼的通用缺口是 IO、字符串、集合、随机。用户拍板分阶段做，本 spec 是**阶段 1：随机数**。

### 加节点路线图（用户排序，2026-06-10）

> 排序约束：① 实际需求 ② 成本 ③ 依赖（数组要先有 List 类型，是框架改动）。用户决定「数组提前」。

1. **阶段 1 — 随机数**（本 spec）：RandomInt / RandomFloat / RandomBool。无框架改动，最高频，先落地。
2. **阶段 2 — 数组/集合**（提前）：需先给框架加 `List` pin 类型 + 遍历能力（ForEach region runner），再做 Split/Join/ListGet/ListLength/ListAppend/ListContains/RandomChoice/Filter/Map。唯一动框架内核。
3. **阶段 3 — 数学补全**：Clamp/Round/Floor/Ceil/Pow/Sqrt/Abs/Min/Max。无框架改动；长尾函数顺手塞进 Expr 的 `evalCall`。
4. **阶段 4 — 字符串函数**：Replace/Substring/Trim/ToUpper/ToLower/IndexOf/StartsWith/EndsWith/RegexMatch/RegexExtract。本框架低频，补完整性。

每阶段独立 spec→plan→impl。

## 已验证的源码事实（设计依据）

- **纯数据求值无结果缓存**：`runtime/data_pull.go::evalDataSource` 每次 pin pull 都重调 `nodepkg.EvaluatePureData`；`node/engine.go::EvaluatePureData` 内部直接跑 `rn.Evaluate`、无记忆化。→ 随机节点语义成立，每次读取得新值。
- **Expr 已有 `abs/min/max/now` 但没有 `random`**（`services/expr/eval.go::evalCall`）→ 随机无任何现有替代路径。
- **jitter 用 `math/rand/v2`**（`node/jitter.go`），`jitterSamples=5` 的 CLT 求均值法。但 `normalJitterFactor(pct)` 是「base±pct%」形态，**不是区间形态**，无法直接复用于区间随机。
- **dropdown widget 存在**：`WidgetSpec{Kind:"dropdown", Props: node.MarshalProps(node.DropdownProps{Options:[]node.EnumOption{{Value:"..."}}})}`（见 `detect/check_template.go:43` MatchMode、`variable/get_var.go` scope）。
- **纯函数节点模式**：`nodes/purefunc/purefunc.go` 用 `specBuilder(kind, inputs, resultType)` 构造单 `Result` 出口的 `IsPureData` Spec + 实现 `Evaluate(ctx, in) (any, error)`。`Integer` 类型 + `in.Int(name)` getter 可用。

## 设计

### 包与归类

- 新建 `internal/nodes/random/` 包（随机非纯函数、语义独立；后续 RandomChoice 也归此）。
- `Category: "Random"`（新 palette 分类）。理由：包已独立，Category 标 `PureFunc` 名不副实；新分类提升可发现性。机制上仍 `IsPureData: true` + 实现 `Evaluator`。
- 底层用 `math/rand/v2`（与 jitter.go 一致，自带 seed、并发安全）。

### 节点清单（3 个）

| 节点 | 输入 | 输出 | 语义 |
|---|---|---|---|
| **RandomInt** | `Min`(Integer, 默认0)、`Max`(Integer, 默认100)、`Distribution`(dropdown: `uniform`默认 / `bell`) | `Result`(Integer) | **闭区间** `[Min,Max]`（两端含）。`Min>Max` 自动交换；`Min==Max` 返回 Min |
| **RandomFloat** | `Min`(Number, 默认0)、`Max`(Number, 默认1)、`Distribution`(dropdown: `uniform` / `bell`) | `Result`(Number) | **半开** `[Min,Max)`。`Min>Max` 自动交换 |
| **RandomBool** | `Prob`(Number, 默认0.5) | `Result`(Bool) | 返回 true 的概率 = Prob。`Prob≤0` 恒 false，`≥1` 恒 true。**无 Distribution**（二值，分布无意义）|

### Distribution（分布选项）

- `uniform`：直接 `rand.Float64()` 铺满区间。
  - RandomFloat：`Min + rand.Float64()*(Max-Min)`
  - RandomInt：`Min + rand.IntN(Max-Min+1)`
- `bell`：新写 helper `bellUnit() float64` —— 取 `jitterSamples`(=5，与 jitter.go 同常量) 个 `rand.Float64()` 求均值（Bates 分布，向 0.5 聚集），再缩放到区间。
  - RandomFloat：`Min + bellUnit()*(Max-Min)`
  - RandomInt：`Min + int(bellUnit()*float64(Max-Min+1))`，结果 clamp 到 `[Min,Max]`（防 `bellUnit()` 取到接近 1 时越界）。
  - **不复用** `normalJitterFactor`（那是 base±pct%，非区间）；仅沿用其 5-样本 CLT 手法。

### 边界与不变量

- `Min>Max`：交换后照常（不报错，宽容）。
- RandomBool `Prob` 越界：`<0` 视作 0、`>1` 视作 1（`rand.Float64()<Prob` 自然覆盖）。
- 输入读取走 `in.Int`/`in.Float64`，沿用 framework 宽松类型转换。

### 已知脚注（写进节点 i18n 描述或 doc）

无 per-tick 记忆化 → **一个随机节点的输出连到两个下游 pin，两边各自重算、拿到不同随机值**。单连单（绝大多数用法）无碍；想"一处随机、多处复用同值"需先经 SetVar 落值再 GetVar 读。

## 全链路落地清单（按 checklists/add-node.md）

1. 后端：`internal/nodes/random/{random.go, random_test.go}` —— 3 个 node type + `init()` 注册 + `bellUnit` helper。
2. 节点目录 i18n：`internal/catalog/node-i18n.json` 加 `node.RandomInt.*` / `RandomFloat` / `RandomBool`（名、描述、各 pin 名/描述、Distribution 枚举项文案），跑 `cd frontend && pnpm gen:node-i18n`。
3. 前端 palette：把 `Random` 分类 + 3 节点挂上 NodePalette（确认新 Category 的分类注册点）。
4. 验证：单测覆盖区间边界 / Min>Max 交换 / bell 落在区间内 / Prob 边界；catalog drift 测试绿；`task nodes` 能看到 3 节点。

## 非目标（YAGNI）

- 不做 Seed/可复现（增状态管理复杂度，无 demand）。
- 不做 RandomChoice（需 List 类型，归阶段 2）。
- 不做正态/泊松等更多分布（uniform + bell 够覆盖，按需再加）。
- 不碰框架、不改现有节点。
