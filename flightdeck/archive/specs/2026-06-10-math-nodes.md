---
status: done
summary: 数学补全节点 Clamp/Round/Floor/Ceil/Pow/Sqrt/Abs/Min/Max + Expr 补函数 (加节点路线图阶段3) — 已实现 (含 spec 漏列的 FN_NAMES/Expr-description 两站点)
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
| **Round** | `X`(Number)、`Digits`(Integer, 默认0) | `Result`(Number) | `Digits` **先 clamp 到 [-15,15]**（防 `10^Digits` 上溢 Inf/下溢 0 出垃圾；float64 有效位本就 ~15-17）；`factor=math.Pow(10,Digits); math.Round(X*factor)/factor`。SQL ROUND 约定：`Digits=0`→最近整数、`Digits=2`→保留 2 位小数、`Digits=-2`→取整到百位（如 `Round(12345,-2)=12300`、`Round(149,-1)=150`）。**i18n 写示例 + 大 Digits 精度警示** |
| **Clamp** | `X`(Number)、`Min`(Number)、`Max`(Number) | `Result`(Number) | `Min>Max` **先交换**（与阶段1 RandomInt 同惯例，i18n 写明）；`X<Min→Min`、`X>Max→Max`、否则 X |
| **Pow** | `Base`(Number)、`Exp`(Number) | `Result`(Number) | `math.Pow`。**特殊值透传**（i18n 简述）：负底数+分数指数→NaN、`0^负`→Inf、`0^0`→1（Go 语义）|
| **Sqrt** | `X`(Number) | `Result`(Number) | `math.Sqrt`（X<0→NaN 透传不特判；i18n 提示负数得 NaN）|

## D. Expr 补函数（顺手，零前端）

`services/expr/eval.go::evalCall` 加 case（复用现有 `AsNumber`/arg 数校验范式，跟 `abs/min/max` 同写法）：

- `floor(x)` / `ceil(x)` / `sqrt(x)` — 1 arg
- `round(x)` 或 `round(x, digits)` — **1 或 2 args**（与 Round 节点能力对齐：单参=取整、双参=带 Digits，同 clamp [-15,15]），消除"节点有 Digits、Expr 没有"的困惑
- `pow(base, exp)` — 2 args
- `clamp(x, lo, hi)` — 3 args（`lo>hi` **同样先交换**，与 Clamp 节点一致）

- **arg 数错**（如 `sqrt()`、`pow(1)`）→ 返 error，沿用现有 `abs/min/max` 的 `len(args)!=N → fmt.Errorf` 范式（不返 NaN/不静默）。
- **`abs/min/max` 已在 Expr，本阶段不动**（节点侧新增 Abs/Min/Max 与之并存，等价：Expr min/max 源码即 `math.Min/Max`）。
- **NaN/Inf 透传**同节点（Expr 算术/比较不特判 NaN，原样流）。
- Expr 测试（`services/expr/expr_test.go`）补 6 函数用例（含特殊值/arg 错）。

## 设计判断（已拍，待复核）

- **进 purefunc 包、Category PureFunc**——不新建包/分类（数学就是纯函数，跟现有 22 个同族），零前端 Category 工作。
- **Round 带 Digits**（默认 0）——比只能取整覆盖更广（三号铁律：该暴露的选项暴露），成本仅一个输入。
- **NaN/Inf 透传**——Sqrt 负数、Pow 特殊值等返 NaN/Inf，不特判（GIGO，与现有 Div 返 NaN 一致）。不做 ±0 规范化（YAGNI，保持 math 包行为）。
- **Abs/Min/Max 与 Expr 重复但保留**——连线党要节点（与 Add/Sub 一致）。**已核无命名冲突**：现有 PureFunc 23 节点无 Abs/Min/Max/Floor/Ceil/Round/Clamp/Pow/Sqrt。
- **Clamp/Round 的 Min>Max 交换 / Digits clamp 各自包内实现**，不抽共享 helper（各 2 行，跨包耦合不值）；若将来语义要改（如改报错）需同步 RandomInt——记此处。
- **PureFunc 面板会变长**（23→32，叠加阶段4 字符串→42）：本阶段接受（换取零新分类成本）；若日后过长，再把 PureFunc 拆 Math/String/Logic 子组（YAGNI，现在不做）。
- **输出仍 Number**：Floor/Ceil/Round 产整数值但走 float64 流（与现有 Length 等一致，下游 in.Int 宽松转）。

## 非目标（YAGNI）

- 不做三角/对数/指数（sin/cos/log/exp/log2…）——无 demand，按需再加。
- 不做 Sign/Trunc/Hypot/Cbrt 等长尾——同上。
- 不做定点/大数/精度控制——float64 够用。

## 落地清单（按 add-node.md，要点）

1. **后端**：`purefunc.go` 加 9 个 node type + `init()` 注册列表追加。无新包、无 blank-import 改动（purefunc 已被既有站点 import）。测试**必含特殊值**：NaN/±Inf 透传、Clamp 交换边界、Round 负 Digits（`12345,-2→12300`）与极端 Digits clamp、Pow/Sqrt 边界（`Pow(10,2)` 用 `abs(r-100)<1e-9` 容差、`Sqrt(-1)→NaN`、`0^0→1`）。
2. **Expr**：`eval.go::evalCall` 加 6 函数（round 1/2 参）+ `expr_test.go` 用例（含 arg 错、clamp 交换、round 双参）。**不动已有 abs/min/max**。
3. **i18n**：zh/en 加 `node.Abs/Min/Max/Floor/Ceil/Round/Clamp/Pow/Sqrt`（label/description/pin label）；description 写 Round Digits 示例+精度警示、Clamp 交换、Pow/Sqrt 特殊值。跑 `pnpm gen:node-i18n`。**无 nodeGroup 改动**。
4. **Expr 函数清单文档**：若 `docs/node-system-reference.md`（§5 拟人化 jitter 邻近）或其它有"Expr 支持函数"清单，同步加 6 函数（核一下 `node-system-reference` 是否列了 abs/min/max/now）。
5. **验证**：`go test ./internal/nodes/... ./internal/catalog/... ./internal/services/expr/... -count=1`；`pnpm gen:node-i18n && pnpm typecheck && pnpm i18n:check`；`task nodes` 见 9 节点在 PureFunc。
