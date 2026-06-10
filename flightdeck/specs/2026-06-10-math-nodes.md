---
status: active
summary: 数学补全节点 Clamp/Round/Floor/Ceil/Pow/Sqrt/Abs/Min/Max + Expr 补函数 (加节点路线图阶段3)
last_updated: 2026-06-10
related: [specs/2026-06-10-random-nodes.md]
---

# 数学补全节点（加节点路线图 · 阶段 3）

## 背景与定位

加节点路线图（[random-nodes spec](2026-06-10-random-nodes.md)）的**阶段 3**。补 PureFunc 缺的数学函数。**无框架改动、无新 palette 分类**——直接进现有 `internal/nodes/purefunc` 包、Category `PureFunc`，跟 Add/Sub/Mul 等 22 个并列，复用 `specBuilder`/`numIn` 模式。最轻的一批，只有后端节点 + i18n。

## 已验证的源码事实

- **Expr 的 `evalCall`（`services/expr/eval.go`）只有 `abs/min/max/now`**——`round/floor/ceil/clamp/pow/sqrt` 是真缺口。
- **项目故意让独立节点与 Expr 并存**（Add/Sub/Mul 都有节点，尽管 Expr 也能 `+ - *`）→ 加 Abs/Min/Max 独立节点不算冗余，与设计一致。
- **`purefunc.go::specBuilder(kind, inputs, resultType)`** 构造单 `Result` 出口的 `IsPureData` Spec；`numIn()` = A/B 两 Number 输入；`in.Float64` 宽松取数。`Neg` 用单 `X` 输入是现成范例。

## 节点清单（9 个，加进 `purefunc` 包，Category `PureFunc`）

> 全 `IsPureData:true` + Evaluator。底层 `math` 包。

| 节点 | 输入 | 输出 | 实现 |
|---|---|---|---|
| **Abs** | `X`(Number) | `Result`(Number) | `math.Abs` |
| **Min** | `A`(Number)、`B`(Number) | `Result`(Number) | `math.Min` |
| **Max** | `A`(Number)、`B`(Number) | `Result`(Number) | `math.Max` |
| **Floor** | `X`(Number) | `Result`(Number) | `math.Floor` |
| **Ceil** | `X`(Number) | `Result`(Number) | `math.Ceil` |
| **Round** | `X`(Number)、`Digits`(Integer, 默认0) | `Result`(Number) | `factor=10^Digits; math.Round(X*factor)/factor`（Digits=0→最近整数；负 Digits→十位/百位取整）|
| **Clamp** | `X`(Number)、`Min`(Number)、`Max`(Number) | `Result`(Number) | `Min>Max` 先交换；`X<Min→Min`、`X>Max→Max`、否则 X |
| **Pow** | `Base`(Number)、`Exp`(Number) | `Result`(Number) | `math.Pow` |
| **Sqrt** | `X`(Number) | `Result`(Number) | `math.Sqrt`（X<0 自然返 NaN，透传不特判）|

## D. Expr 补函数（顺手，零前端）

`services/expr/eval.go::evalCall` 加 case（每个 ~3 行，复用现有 `AsNumber`/arg 数校验范式，跟 `abs/min/max` 同写法）：

- `floor(x)` / `ceil(x)` / `round(x)` / `sqrt(x)` — 1 arg
- `pow(base, exp)` — 2 args
- `clamp(x, lo, hi)` — 3 args

让"写表达式"的人也能用，不用非接节点。Expr 测试（`services/expr/expr_test.go`）补对应用例。

## 设计判断（已拍，待复核）

- **进 purefunc 包、Category PureFunc**——不新建包/分类（数学就是纯函数，跟现有 22 个同族），零前端 Category 工作。
- **Round 带 Digits**（默认 0）——比只能取整覆盖更广（三号铁律：该暴露的选项暴露），成本仅一个输入。
- **NaN/Inf 透传**——Sqrt 负数、Pow 溢出等返 NaN/Inf，不特判（GIGO，与现有 Div 返 NaN 一致）。
- **Abs/Min/Max 与 Expr 重复但保留**——连线党要节点（与 Add/Sub 一致）。

## 非目标（YAGNI）

- 不做三角/对数/指数（sin/cos/log/exp/log2…）——无 demand，按需再加。
- 不做 Sign/Trunc/Hypot/Cbrt 等长尾——同上。
- 不做定点/大数/精度控制——float64 够用。

## 落地清单（按 add-node.md，要点）

1. **后端**：`purefunc.go` 加 9 个 node type + `init()` 注册列表追加（现有 `for _, n := range []node.Node{...}`）。无新包、无 blank-import 改动（purefunc 已被 10 处 import）。
2. **Expr**：`eval.go::evalCall` 加 6 个 case + `expr_test.go` 用例。
3. **i18n**：zh/en 加 `node.Abs/Min/Max/Floor/Ceil/Round/Clamp/Pow/Sqrt`（label/description/pin label），跑 `pnpm gen:node-i18n`。**无 nodeGroup 改动**（仍 PureFunc）。
4. **验证**：`go test ./internal/nodes/... ./internal/catalog/... ./internal/services/expr/... -count=1`；`pnpm gen:node-i18n && pnpm typecheck && pnpm i18n:check`；`task nodes` 见 9 节点在 PureFunc。
