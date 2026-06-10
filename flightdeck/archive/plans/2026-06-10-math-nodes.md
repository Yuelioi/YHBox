---
status: done
summary: 数学补全 9 节点 (purefunc 包) + Expr 补 6 函数的实现计划 (TDD 分任务) — 已实现 (fea0d87..6402c4b), 终审 SHIP
last_updated: 2026-06-10
implements: specs/2026-06-10-math-nodes.md
---

# 数学补全节点 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 给 purefunc 包补 Abs/Min/Max/Floor/Ceil/Round/Clamp/Pow/Sqrt 九个数学节点，并给 Expr 补 floor/ceil/sqrt/round/pow/clamp 六个函数。

**Architecture:** 零框架改动。后端节点进现有 `internal/nodes/purefunc` 包（新文件 `math.go`，复用 `specBuilder`，Category `PureFunc`，无新 palette 分类、无 blank-import 改动）；Expr 函数加进 `services/expr/eval.go::evalCall`（沿用 abs/min/max 的 arg 校验范式）。前端只有 i18n + `ExpressionInput.vue` 的函数自动补全清单。

**Tech Stack:** Go（`math` 包）、vue-i18n、Taskfile。

**实现依据见 spec：** `flightdeck/specs/2026-06-10-math-nodes.md`。配套全链路规范 `flightdeck/checklists/add-node.md`。

---

## File Structure

**新建：**
- `internal/nodes/purefunc/math.go` — 9 个数学节点（不含 init，注册在 purefunc.go 既有 init 列表追加）。
- `internal/nodes/purefunc/math_test.go` — 节点单测（常规值 + NaN/Inf/边界特殊值）。

**修改（后端）：**
- `internal/nodes/purefunc/purefunc.go` — `init()` 注册列表追加 9 个 + 包 doc 数量 22→31。
- `internal/services/expr/eval.go::evalCall` — 加 6 个 case。
- `internal/services/expr/expr_test.go` — 补 6 函数用例。

**修改（前端 / i18n / docs）：**
- `frontend/src/i18n/zh.ts` + `en.ts` — 9 个 `node.<Kind>` 块；`node.Expr.description` 函数清单加 6 函数。
- `frontend/src/components/expressions/ExpressionInput.vue:102` — `FN_NAMES` 加 6 函数（自动补全）。
- `flightdeck/docs/node-system-reference.md:104` — PureFunc 行 23→32、加 9 节点名。

**已核源码事实（实现者不必重查，但撞不一致时停下报告）：**
- `specBuilder(kind, inputs, resultType)` 构造单 `Result` 出口 PureFunc Spec（purefunc.go:21）；`numIn()` = A/B 两 Number 输入（:34）；Neg 是单 `X`(Number) 输入范例（:200）。
- 命名分裂守卫 `DetectNameSplits`（catalog.go:138）：类型集含 `*` 的 pin 名整体豁免——`X`（ToString 等用 `*`）、`A`/`B`（Eq 用 `*`）已豁免；`Clamp` 的 `Min`/`Max` 必须用 **Number**（与 RandomInt/RandomFloat 的 Number 一致，用 Integer 会触发分裂 FAIL）；`Digits`/`Base`/`Exp` 是全新 pin 名，无碰撞。
- `evalCall`（expr/eval.go:205）现有 abs/min/max/now，范式：`len(args)!=N → fmt.Errorf("expr: f() expects N args at col %d", n.Pos)`，`AsNumber` 失败 → `needs number`。
- 既有测试范式：`purefunc_test.go::TestEvaluate_22PureFuncs` — `node.ResetRegistryForTest()` + 逐个 `node.Register` + 表驱动 `node.EvaluatePureData(context.Background(), rn, dataWire, nil, node.StubServices())`。
- `ExpressionInput.vue:102` `FN_NAMES = ['abs', 'min', 'max', 'now']` 只驱动自动补全（非校验），漏加不报错但体验缺失。
- `node.Expr.description`（zh.ts:1057 / en.ts:1037）正文列了"内置函数 abs、min、max、now"——必须同步加 6 函数。

---

## Task 1: math.go — 7 个一行实现节点 (Abs/Min/Max/Floor/Ceil/Pow/Sqrt)

**Files:**
- Create: `internal/nodes/purefunc/math.go`
- Create: `internal/nodes/purefunc/math_test.go`

- [ ] **Step 1: 写失败测试**

`internal/nodes/purefunc/math_test.go`：

```go
package purefunc

import (
	"context"
	"math"
	"testing"

	"yotta/internal/node"
)

// evalMathNode 直接走 framework EvaluatePureData (与 TestEvaluate_22PureFuncs 同范式).
func evalMathNode(t *testing.T, n node.Node, dataWire map[string]any) any {
	t.Helper()
	node.ResetRegistryForTest()
	node.Register(n)
	rn, ok := node.Get(n.Spec().Kind)
	if !ok {
		t.Fatalf("kind %q not registered", n.Spec().Kind)
	}
	got, err := node.EvaluatePureData(context.Background(), rn, dataWire, nil, node.StubServices())
	if err != nil {
		t.Fatalf("EvaluatePureData err: %v", err)
	}
	return got
}

func wantNum(t *testing.T, got any, want float64) {
	t.Helper()
	f, ok := got.(float64)
	if !ok {
		t.Fatalf("want float64, got %T (%v)", got, got)
	}
	if math.Abs(f-want) > 1e-9 {
		t.Fatalf("want %v, got %v", want, f)
	}
}

func wantNaN(t *testing.T, got any) {
	t.Helper()
	f, ok := got.(float64)
	if !ok || !math.IsNaN(f) {
		t.Fatalf("want NaN, got %T (%v)", got, got)
	}
}

func TestMath_SimpleSeven(t *testing.T) {
	cases := []struct {
		name     string
		n        node.Node
		dataWire map[string]any
		want     float64
	}{
		{"Abs_neg", &Abs{}, map[string]any{"X": -5.5}, 5.5},
		{"Abs_pos", &Abs{}, map[string]any{"X": 3.0}, 3.0},
		{"Min", &Min{}, map[string]any{"A": 3.0, "B": 7.0}, 3.0},
		{"Max", &Max{}, map[string]any{"A": 3.0, "B": 7.0}, 7.0},
		{"Floor_pos", &Floor{}, map[string]any{"X": 3.7}, 3.0},
		{"Floor_neg", &Floor{}, map[string]any{"X": -3.2}, -4.0},
		{"Ceil_pos", &Ceil{}, map[string]any{"X": 3.2}, 4.0},
		{"Ceil_neg", &Ceil{}, map[string]any{"X": -3.7}, -3.0},
		{"Pow", &Pow{}, map[string]any{"Base": 10.0, "Exp": 2.0}, 100.0},
		{"Pow_zero_zero", &Pow{}, map[string]any{"Base": 0.0, "Exp": 0.0}, 1.0},
		{"Sqrt", &Sqrt{}, map[string]any{"X": 9.0}, 3.0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wantNum(t, evalMathNode(t, tc.n, tc.dataWire), tc.want)
		})
	}
}

func TestMath_SpecialValues(t *testing.T) {
	// Sqrt 负数 → NaN 透传 (GIGO, 与 Div 除零返 NaN 一致).
	wantNaN(t, evalMathNode(t, &Sqrt{}, map[string]any{"X": -1.0}))
	// Pow 负底数+分数指数 → NaN.
	wantNaN(t, evalMathNode(t, &Pow{}, map[string]any{"Base": -2.0, "Exp": 0.5}))
	// Pow 0^负 → +Inf.
	if f := evalMathNode(t, &Pow{}, map[string]any{"Base": 0.0, "Exp": -1.0}).(float64); !math.IsInf(f, 1) {
		t.Fatalf("0^-1 want +Inf, got %v", f)
	}
	// Abs(NaN) → NaN 透传.
	wantNaN(t, evalMathNode(t, &Abs{}, map[string]any{"X": math.NaN()}))
}

func TestMath_SpecShape(t *testing.T) {
	for _, n := range []node.Node{&Abs{}, &Min{}, &Max{}, &Floor{}, &Ceil{}, &Pow{}, &Sqrt{}} {
		s := n.Spec()
		if !s.IsPureData || s.Category != "PureFunc" {
			t.Fatalf("%s: must be IsPureData + Category PureFunc, got %+v", s.Kind, s)
		}
		if len(s.Outputs) != 1 || s.Outputs[0].Name != "Result" || s.Outputs[0].Type != "Number" {
			t.Fatalf("%s: want single Result(Number) output, got %+v", s.Kind, s.Outputs)
		}
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `go test ./internal/nodes/purefunc/ -run TestMath -v`
Expected: 编译失败（`Abs` 等未定义）。

- [ ] **Step 3: 写实现**

`internal/nodes/purefunc/math.go`：

```go
// 数学补全节点 (9 个): Abs/Min/Max/Floor/Ceil/Round/Clamp/Pow/Sqrt.
// 见 specs/2026-06-10-math-nodes.md. 注册在 purefunc.go::init() "数学 (9)" 组.
// NaN/Inf 一律透传 (GIGO, 与 Div 除零返 NaN 同惯例), 不特判.
package purefunc

import (
	"math"

	"yotta/internal/node"
)

// numXIn 单 X Number 输入 (Abs/Floor/Ceil/Sqrt/Round/Clamp 共用形态, 同 Neg).
func numXIn() node.InputSpec {
	return node.InputSpec{Name: "X", Type: "Number", Default: json.Number("0"), Widget: node.WidgetSpec{Kind: "number"}}
}

// ===== 单输入族 =====

type Abs struct{}

func (Abs) Spec() node.Spec {
	return specBuilder("Abs", []node.InputSpec{numXIn()}, "Number")
}
func (Abs) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	return math.Abs(in.Float64("X")), nil
}

type Floor struct{}

func (Floor) Spec() node.Spec {
	return specBuilder("Floor", []node.InputSpec{numXIn()}, "Number")
}
func (Floor) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	return math.Floor(in.Float64("X")), nil
}

type Ceil struct{}

func (Ceil) Spec() node.Spec {
	return specBuilder("Ceil", []node.InputSpec{numXIn()}, "Number")
}
func (Ceil) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	return math.Ceil(in.Float64("X")), nil
}

type Sqrt struct{}

func (Sqrt) Spec() node.Spec {
	return specBuilder("Sqrt", []node.InputSpec{numXIn()}, "Number")
}

// Evaluate — X<0 → NaN 透传不特判 (i18n 已提示).
func (Sqrt) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	return math.Sqrt(in.Float64("X")), nil
}

// ===== 双输入族 =====

type Min struct{}

func (Min) Spec() node.Spec {
	return specBuilder("Min", numIn(), "Number")
}
func (Min) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	return math.Min(in.Float64("A"), in.Float64("B")), nil
}

type Max struct{}

func (Max) Spec() node.Spec {
	return specBuilder("Max", numIn(), "Number")
}
func (Max) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	return math.Max(in.Float64("A"), in.Float64("B")), nil
}

type Pow struct{}

func (Pow) Spec() node.Spec {
	return specBuilder("Pow", []node.InputSpec{
		{Name: "Base", Type: "Number", Default: json.Number("0"), Widget: node.WidgetSpec{Kind: "number"}},
		{Name: "Exp", Type: "Number", Default: json.Number("1"), Widget: node.WidgetSpec{Kind: "number"}},
	}, "Number")
}

// Evaluate — 特殊值走 Go math.Pow 语义透传: 负底数+分数指数→NaN, 0^负→Inf, 0^0→1.
func (Pow) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	return math.Pow(in.Float64("Base"), in.Float64("Exp")), nil
}
```

注意：`json.Number` 需要 `encoding/json` import——purefunc.go 已在包内 import 不等于 math.go 自动可用，**math.go 自己的 import 块要加 `encoding/json`**（Step 4 编译时 goimports/手动补上）。

- [ ] **Step 4: 运行确认通过**

Run: `go test ./internal/nodes/purefunc/ -run TestMath -v`
Expected: PASS（全部）。

- [ ] **Step 5: 既有 purefunc 测试无回归**

Run: `go test ./internal/nodes/purefunc/ -count=1`
Expected: PASS。

- [ ] **Step 6: Commit**

```bash
git add internal/nodes/purefunc/math.go internal/nodes/purefunc/math_test.go
git commit -m "feat(purefunc): 数学节点 Abs/Min/Max/Floor/Ceil/Pow/Sqrt"
```

---

## Task 2: Round + Clamp

**Files:**
- Modify: `internal/nodes/purefunc/math.go`
- Test: `internal/nodes/purefunc/math_test.go`

- [ ] **Step 1: 写失败测试**

追加到 `math_test.go`：

```go
func TestRound_SQLConvention(t *testing.T) {
	cases := []struct {
		name string
		x    float64
		d    int
		want float64
	}{
		{"d0_half_up", 2.5, 0, 3.0},
		{"d0_down", 2.4, 0, 2.0},
		{"d2", 3.14159, 2, 3.14},
		{"neg_d_hundreds", 12345, -2, 12300},
		{"neg_d_tens", 149, -1, 150},
		{"d_overclamp_hi", 1.23456, 99, 1.23456},   // Digits clamp 到 15, 精度内原样
		{"d_overclamp_lo", 12345, -99, 0},          // clamp 到 -15 → 10^15 量级取整 → 0
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := evalMathNode(t, &Round{}, map[string]any{"X": tc.x, "Digits": tc.d})
			wantNum(t, got, tc.want)
		})
	}
}

func TestRound_DefaultDigitsZero(t *testing.T) {
	// 不传 Digits → 默认 0 → 取整.
	wantNum(t, evalMathNode(t, &Round{}, map[string]any{"X": 7.6}), 8.0)
}

func TestClamp_Basic(t *testing.T) {
	cases := []struct {
		name          string
		x, lo, hi, want float64
	}{
		{"below", -5, 0, 10, 0},
		{"above", 15, 0, 10, 10},
		{"inside", 5, 0, 10, 5},
		{"swap_bounds", 5, 10, 0, 5},   // Min>Max 先交换 (与 RandomInt 同惯例)
		{"swap_below", -1, 10, 0, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := evalMathNode(t, &Clamp{}, map[string]any{"X": tc.x, "Min": tc.lo, "Max": tc.hi})
			wantNum(t, got, tc.want)
		})
	}
}

func TestClamp_NaNPassthrough(t *testing.T) {
	// NaN 任何比较为 false → 不触发任何边界分支 → 原样透传.
	wantNaN(t, evalMathNode(t, &Clamp{}, map[string]any{"X": math.NaN(), "Min": 0.0, "Max": 10.0}))
}

func TestRoundClamp_SpecShape(t *testing.T) {
	for _, n := range []node.Node{&Round{}, &Clamp{}} {
		s := n.Spec()
		if !s.IsPureData || s.Category != "PureFunc" {
			t.Fatalf("%s: bad spec %+v", s.Kind, s)
		}
	}
	// Clamp 的 Min/Max 必须是 Number — 命名分裂守卫要求与 RandomInt/RandomFloat 一致.
	for _, in := range (Clamp{}).Spec().Inputs {
		if (in.Name == "Min" || in.Name == "Max") && in.Type != "Number" {
			t.Fatalf("Clamp.%s must be Number (DetectNameSplits), got %s", in.Name, in.Type)
		}
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `go test ./internal/nodes/purefunc/ -run "TestRound|TestClamp" -v`
Expected: 编译失败（`Round` / `Clamp` 未定义）。

- [ ] **Step 3: 写实现**

追加到 `math.go`：

```go
// ===== Round / Clamp =====

type Round struct{}

func (Round) Spec() node.Spec {
	return specBuilder("Round", []node.InputSpec{
		numXIn(),
		{Name: "Digits", Type: "Integer", Default: json.Number("0"), Widget: node.WidgetSpec{Kind: "number"}},
	}, "Number")
}

// Evaluate — SQL ROUND 约定: Digits=0→最近整数, 2→两位小数, -2→取整到百位.
// Digits clamp 到 [-15,15]: 防 10^Digits 上溢 Inf/下溢 0 出垃圾 (float64 有效位本就 ~15-17).
func (Round) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	x := in.Float64("X")
	d := in.Int("Digits")
	if d > 15 {
		d = 15
	}
	if d < -15 {
		d = -15
	}
	factor := math.Pow(10, float64(d))
	return math.Round(x*factor) / factor, nil
}

type Clamp struct{}

func (Clamp) Spec() node.Spec {
	return specBuilder("Clamp", []node.InputSpec{
		numXIn(),
		{Name: "Min", Type: "Number", Default: json.Number("0"), Widget: node.WidgetSpec{Kind: "number"}},
		{Name: "Max", Type: "Number", Default: json.Number("100"), Widget: node.WidgetSpec{Kind: "number"}},
	}, "Number")
}

// Evaluate — Min>Max 先交换 (与 RandomInt 同惯例). NaN 比较恒 false → 原样透传.
func (Clamp) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	x, lo, hi := in.Float64("X"), in.Float64("Min"), in.Float64("Max")
	if lo > hi {
		lo, hi = hi, lo
	}
	switch {
	case x < lo:
		return lo, nil
	case x > hi:
		return hi, nil
	}
	return x, nil
}
```

- [ ] **Step 4: 运行确认通过**

Run: `go test ./internal/nodes/purefunc/ -count=1`
Expected: PASS（全部, 含既有）。

- [ ] **Step 5: Commit**

```bash
git add internal/nodes/purefunc/math.go internal/nodes/purefunc/math_test.go
git commit -m "feat(purefunc): Round (SQL Digits 约定) + Clamp (越界交换) 节点"
```

---

## Task 3: init 注册 + 包 doc + i18n + catalog 绿

**Files:**
- Modify: `internal/nodes/purefunc/purefunc.go`（init 列表 + 包 doc）
- Modify: `frontend/src/i18n/zh.ts`、`frontend/src/i18n/en.ts`

- [ ] **Step 1: init() 注册列表追加**

`purefunc.go::init()` 的列表里（`// 三元 (1)` 行后）加：

```go
		// 数学 (9)
		&Abs{}, &Min{}, &Max{}, &Floor{}, &Ceil{}, &Round{}, &Clamp{}, &Pow{}, &Sqrt{},
```

- [ ] **Step 2: 包 doc 数量更新**

`purefunc.go` 第 1 行包注释 `// Package purefunc 22 个纯函数节点 (Add/Sub/.../Select) + Expr.` 改为：

```go
// Package purefunc 31 个纯函数节点 (Add/Sub/.../Select + 数学 Abs/.../Sqrt) + Expr.
```

（math.go 的包注释与此并存没问题——Go 多文件包注释只认其一，math.go 顶部那段保留为文件级说明即可；若 `go vet` 报 duplicate package comment，把 math.go 顶部注释的 `// Package purefunc ...` 措辞改成普通行注释 `// math.go — ...`。）

- [ ] **Step 3: zh.ts 加 9 节点块**

`frontend/src/i18n/zh.ts` 的 `node:` 大块内，紧跟现有 purefunc 节点（看现有排序习惯就近插入）：

```ts
    Abs: {
      label: '绝对值', description: '取 X 的绝对值（负变正）。',
      input: { X: { label: '数值' } },
      output: { Result: { label: '结果' } },
    },
    Min: {
      label: '取较小', description: '两个数里取较小的那个。',
      input: { A: { label: '甲' }, B: { label: '乙' } },
      output: { Result: { label: '结果' } },
    },
    Max: {
      label: '取较大', description: '两个数里取较大的那个。',
      input: { A: { label: '甲' }, B: { label: '乙' } },
      output: { Result: { label: '结果' } },
    },
    Floor: {
      label: '向下取整', description: '把 X 往小的方向取整：3.7 得 3，-3.2 得 -4。',
      input: { X: { label: '数值' } },
      output: { Result: { label: '结果' } },
    },
    Ceil: {
      label: '向上取整', description: '把 X 往大的方向取整：3.2 得 4，-3.7 得 -3。',
      input: { X: { label: '数值' } },
      output: { Result: { label: '结果' } },
    },
    Round: {
      label: '四舍五入', description: '把 X 四舍五入。位数=0 取到整数；位数=2 保留 2 位小数；位数=-2 取整到百位（12345 得 12300）。位数最多 ±15（再多超出小数精度，按 ±15 算）。',
      input: { X: { label: '数值' }, Digits: { label: '位数' } },
      output: { Result: { label: '结果' } },
    },
    Clamp: {
      label: '限制范围', description: '把 X 限制在 Min~Max 里：小于 Min 出 Min，大于 Max 出 Max，否则原样出。Min 比 Max 大时自动交换。',
      input: { X: { label: '数值' }, Min: { label: '下限' }, Max: { label: '上限' } },
      output: { Result: { label: '结果' } },
    },
    Pow: {
      label: '乘方', description: '算 Base 的 Exp 次方。特殊情况按数学惯例：负数开分数次方得 NaN、0 的负次方得 Infinity、0 的 0 次方得 1。',
      input: { Base: { label: '底数' }, Exp: { label: '指数' } },
      output: { Result: { label: '结果' } },
    },
    Sqrt: {
      label: '开平方', description: '算 X 的平方根。X 是负数时得 NaN（需要时先接"绝对值"节点）。',
      input: { X: { label: '数值' } },
      output: { Result: { label: '结果' } },
    },
```

- [ ] **Step 4: en.ts 加镜像块**

```ts
    Abs: {
      label: 'Abs', description: 'Absolute value of X (negatives become positive).',
      input: { X: { label: 'X' } },
      output: { Result: { label: 'Result' } },
    },
    Min: {
      label: 'Min', description: 'The smaller of two numbers.',
      input: { A: { label: 'A' }, B: { label: 'B' } },
      output: { Result: { label: 'Result' } },
    },
    Max: {
      label: 'Max', description: 'The larger of two numbers.',
      input: { A: { label: 'A' }, B: { label: 'B' } },
      output: { Result: { label: 'Result' } },
    },
    Floor: {
      label: 'Floor', description: 'Round X down: 3.7 gives 3, -3.2 gives -4.',
      input: { X: { label: 'X' } },
      output: { Result: { label: 'Result' } },
    },
    Ceil: {
      label: 'Ceil', description: 'Round X up: 3.2 gives 4, -3.7 gives -3.',
      input: { X: { label: 'X' } },
      output: { Result: { label: 'Result' } },
    },
    Round: {
      label: 'Round', description: 'Round X. Digits=0 rounds to integer; 2 keeps two decimals; -2 rounds to hundreds (12345 gives 12300). Digits is capped at +/-15 (beyond float precision).',
      input: { X: { label: 'X' }, Digits: { label: 'Digits' } },
      output: { Result: { label: 'Result' } },
    },
    Clamp: {
      label: 'Clamp', description: 'Limit X to the Min~Max range: below Min gives Min, above Max gives Max, otherwise X. Min/Max swap automatically if reversed.',
      input: { X: { label: 'X' }, Min: { label: 'Min' }, Max: { label: 'Max' } },
      output: { Result: { label: 'Result' } },
    },
    Pow: {
      label: 'Pow', description: 'Base raised to Exp. Math conventions apply: negative base with fractional exponent gives NaN, 0 to a negative power gives Infinity, 0^0 gives 1.',
      input: { Base: { label: 'Base' }, Exp: { label: 'Exp' } },
      output: { Result: { label: 'Result' } },
    },
    Sqrt: {
      label: 'Sqrt', description: 'Square root of X. Negative X gives NaN (wire an Abs node first if needed).',
      input: { X: { label: 'X' } },
      output: { Result: { label: 'Result' } },
    },
```

⚠ i18n trap（项目 checklist `vue-i18n-message-compiler-traps`）：message 里 `{` `}` `|` `@` `$` 是特殊字符。上面块已避开（注意 `±` 在 en 里写成 `+/-`、`~` 安全）。照抄，别改写。

- [ ] **Step 5: 重新生成 catalog i18n + 校验**

Run: `cd frontend && pnpm gen:node-i18n`
Expected: 86 个节点（77+9），`internal/catalog/node-i18n.json` 更新。

Run: `cd frontend && pnpm i18n:check`
Expected: parity OK + compile OK（residue FAIL 28 为已知预存，与本次无关）。

Run: `go test ./internal/nodes/purefunc/ ./internal/catalog/... ./internal/node/... -count=1`
Expected: PASS（drift 守卫认得 9 新节点；`TestNoPinNameSplit` 无新分裂）。

- [ ] **Step 6: Commit**

```bash
git add internal/nodes/purefunc/purefunc.go frontend/src/i18n/ internal/catalog/node-i18n.json
git commit -m "feat(purefunc): 注册 9 数学节点 + zh/en i18n"
```

---

## Task 4: Expr 补 6 函数 + 前端函数清单同步

**Files:**
- Modify: `internal/services/expr/eval.go::evalCall`（now case 之后、`return nil, fmt.Errorf("expr: unknown function ...")` 之前）
- Test: `internal/services/expr/expr_test.go`
- Modify: `frontend/src/components/expressions/ExpressionInput.vue:102`（FN_NAMES）
- Modify: `frontend/src/i18n/zh.ts` + `en.ts`（`node.Expr.description` 函数清单）

- [ ] **Step 1: 写失败测试**

追加到 `internal/services/expr/expr_test.go`（找到现有 abs/min 用例所在测试函数，紧挨着加新函数，或新建一个测试函数——沿用该文件 `eval(t, "...", env)` + `AsNumber` 既有 helper 范式；下面以新建函数为例）：

```go
func TestEvalCall_MathFunctions(t *testing.T) {
	env := Env{}

	// floor / ceil / sqrt — 1 arg
	if got, _ := AsNumber(eval(t, "floor(3.7)", env)); got != 3 {
		t.Errorf("floor(3.7) = %v, want 3", got)
	}
	if got, _ := AsNumber(eval(t, "ceil(3.2)", env)); got != 4 {
		t.Errorf("ceil(3.2) = %v, want 4", got)
	}
	if got, _ := AsNumber(eval(t, "sqrt(9)", env)); got != 3 {
		t.Errorf("sqrt(9) = %v, want 3", got)
	}

	// round — 1 或 2 args (与 Round 节点对齐)
	if got, _ := AsNumber(eval(t, "round(2.5)", env)); got != 3 {
		t.Errorf("round(2.5) = %v, want 3", got)
	}
	if got, _ := AsNumber(eval(t, "round(3.14159, 2)", env)); got != 3.14 {
		t.Errorf("round(3.14159, 2) = %v, want 3.14", got)
	}
	if got, _ := AsNumber(eval(t, "round(12345, -2)", env)); got != 12300 {
		t.Errorf("round(12345, -2) = %v, want 12300", got)
	}

	// pow — 2 args
	if got, _ := AsNumber(eval(t, "pow(10, 2)", env)); math.Abs(got-100) > 1e-9 {
		t.Errorf("pow(10,2) = %v, want 100", got)
	}

	// clamp — 3 args, lo>hi 交换
	if got, _ := AsNumber(eval(t, "clamp(15, 0, 10)", env)); got != 10 {
		t.Errorf("clamp(15,0,10) = %v, want 10", got)
	}
	if got, _ := AsNumber(eval(t, "clamp(5, 10, 0)", env)); got != 5 {
		t.Errorf("clamp(5,10,0) = %v, want 5 (swap)", got)
	}

	// 特殊值: sqrt 负数 → NaN
	if got, _ := AsNumber(eval(t, "sqrt(-1)", env)); !math.IsNaN(got) {
		t.Errorf("sqrt(-1) = %v, want NaN", got)
	}
}

func TestEvalCall_MathFunctions_ArgErrors(t *testing.T) {
	env := Env{}
	for _, expr := range []string{"floor()", "ceil(1, 2)", "sqrt()", "round()", "round(1, 2, 3)", "pow(1)", "clamp(1, 2)"} {
		if _, err := evalErr(t, expr, env); err == nil {
			t.Errorf("%s: want arg-count error, got nil", expr)
		}
	}
}
```

⚠ 适配点：`eval` / `evalErr` helper 名、`Env` 构造按 expr_test.go 实际范式来（先读该文件——若没有 `evalErr` helper，仿 `eval` 写一个返回 err 的，或直接 `Parse`+`Eval` 两步）。`math` import 若缺则补。意图不变：正常值 + 交换 + NaN + arg 数错全覆盖。

- [ ] **Step 2: 运行确认失败**

Run: `go test ./internal/services/expr/ -run TestEvalCall_Math -v`
Expected: FAIL（`expr: unknown function "floor"` 类错误）。

- [ ] **Step 3: evalCall 加 6 个 case**

`internal/services/expr/eval.go::evalCall`，`case "now":` 块之后加：

```go
	case "floor":
		if len(args) != 1 {
			return nil, fmt.Errorf("expr: floor() expects 1 arg at col %d", n.Pos)
		}
		x, ok := AsNumber(args[0])
		if !ok {
			return nil, fmt.Errorf("expr: floor() needs number at col %d", n.Pos)
		}
		return math.Floor(x), nil
	case "ceil":
		if len(args) != 1 {
			return nil, fmt.Errorf("expr: ceil() expects 1 arg at col %d", n.Pos)
		}
		x, ok := AsNumber(args[0])
		if !ok {
			return nil, fmt.Errorf("expr: ceil() needs number at col %d", n.Pos)
		}
		return math.Ceil(x), nil
	case "sqrt":
		if len(args) != 1 {
			return nil, fmt.Errorf("expr: sqrt() expects 1 arg at col %d", n.Pos)
		}
		x, ok := AsNumber(args[0])
		if !ok {
			return nil, fmt.Errorf("expr: sqrt() needs number at col %d", n.Pos)
		}
		return math.Sqrt(x), nil
	case "round":
		// round(x) 取整 / round(x, digits) 带位数 — 与 Round 节点对齐, digits 同 clamp [-15,15].
		if len(args) != 1 && len(args) != 2 {
			return nil, fmt.Errorf("expr: round() expects 1 or 2 args at col %d", n.Pos)
		}
		x, ok := AsNumber(args[0])
		if !ok {
			return nil, fmt.Errorf("expr: round() needs number at col %d", n.Pos)
		}
		if len(args) == 1 {
			return math.Round(x), nil
		}
		d, ok := AsNumber(args[1])
		if !ok {
			return nil, fmt.Errorf("expr: round() digits needs number at col %d", n.Pos)
		}
		d = math.Trunc(d)
		if d > 15 {
			d = 15
		}
		if d < -15 {
			d = -15
		}
		factor := math.Pow(10, d)
		return math.Round(x*factor) / factor, nil
	case "pow":
		if len(args) != 2 {
			return nil, fmt.Errorf("expr: pow() expects 2 args at col %d", n.Pos)
		}
		a, aok := AsNumber(args[0])
		b, bok := AsNumber(args[1])
		if !aok || !bok {
			return nil, fmt.Errorf("expr: pow() needs numbers at col %d", n.Pos)
		}
		return math.Pow(a, b), nil
	case "clamp":
		// clamp(x, lo, hi) — lo>hi 先交换, 与 Clamp 节点一致.
		if len(args) != 3 {
			return nil, fmt.Errorf("expr: clamp() expects 3 args at col %d", n.Pos)
		}
		x, xok := AsNumber(args[0])
		lo, lok := AsNumber(args[1])
		hi, hok := AsNumber(args[2])
		if !xok || !lok || !hok {
			return nil, fmt.Errorf("expr: clamp() needs numbers at col %d", n.Pos)
		}
		if lo > hi {
			lo, hi = hi, lo
		}
		switch {
		case x < lo:
			return lo, nil
		case x > hi:
			return hi, nil
		}
		return x, nil
```

- [ ] **Step 4: 运行确认通过**

Run: `go test ./internal/services/expr/ -count=1`
Expected: PASS（全部, 含既有）。

- [ ] **Step 5: FN_NAMES 同步**

`frontend/src/components/expressions/ExpressionInput.vue:102`：

```ts
const FN_NAMES = ['abs', 'ceil', 'clamp', 'floor', 'max', 'min', 'now', 'pow', 'round', 'sqrt']
```

- [ ] **Step 6: Expr 节点 i18n description 同步**

`frontend/src/i18n/zh.ts`（~:1057）`node.Expr.description` 里 `内置函数 abs、min、max、now` 改为：

```
内置函数 abs、min、max、now、floor、ceil、round（可带位数）、clamp、pow、sqrt
```

`frontend/src/i18n/en.ts`（~:1037）`the built-in functions abs, min, max, now` 改为：

```
the built-in functions abs, min, max, now, floor, ceil, round (optionally with digits), clamp, pow, sqrt
```

（只动函数清单那几个词，其余 description 文字不动。）

- [ ] **Step 7: 前端校验**

Run: `cd frontend && pnpm gen:node-i18n && pnpm typecheck && pnpm i18n:check`
Expected: 全 PASS（residue 28 已知）。

- [ ] **Step 8: Commit**

```bash
git add internal/services/expr/ frontend/src/components/expressions/ExpressionInput.vue frontend/src/i18n/ internal/catalog/node-i18n.json
git commit -m "feat(expr): 补 floor/ceil/sqrt/round/pow/clamp 六函数 + 前端补全清单/描述同步"
```

---

## Task 5: docs 同步 + 全量验证

**Files:**
- Modify: `flightdeck/docs/node-system-reference.md:104`（PureFunc 行）

- [ ] **Step 1: node-system-reference.md 节点目录更新**

第 104 行 `| **PureFunc** (23) | Add, And, Concat, ... |` 改为（23→32，按字母序插入 9 个新名）：

```markdown
| **PureFunc** (32) | Abs, Add, And, Ceil, Clamp, Concat, Contains, Div, Eq, Expr, Floor, Gt, GtEq, Length, Lt, LtEq, Max, Min, Mod, Mul, Neg, Not, NotEq, Or, Pow, Round, Select, Sqrt, Sub, ToBool, ToNumber, ToString — **全 PureData (Evaluator)** |
```

- [ ] **Step 2: 后端全绿**

Run: `go build ./... && go test ./internal/nodes/... ./internal/node/... ./internal/catalog/... ./internal/services/expr/... -count=1`
Expected: PASS。

- [ ] **Step 3: catalog 自查**

Run: `task nodes`
Expected: 输出含 Abs/Min/Max/Floor/Ceil/Round/Clamp/Pow/Sqrt，归 PureFunc，pin/默认值正确。

- [ ] **Step 4: 构建**

Run: `task build`
Expected: 成功（runtime fish fixture 预存失败非本次回归，见 `flightdeck/checklists/build.md`）。

- [ ] **Step 5: 真机 smoke（人工，留给用户）**

启动 app → 容器编辑器：
- **看什么**：「纯函数」分组里出现 9 个新节点，文案/默认值对。
- **怎么验**：拖 Round → X=3.14159、位数=2 → 连 Log → 跑一次 → 输出 3.14。Expr 节点写 `clamp(15, 0, 10)` → 输出 10。
- **什么算过**：两条都对。

- [ ] **Step 6: Commit**

```bash
git add flightdeck/docs/node-system-reference.md
git commit -m "docs(nodes): node-system-reference PureFunc 目录 23→32 (数学 9 节点)"
```

---

## Self-Review（写完计划的自查结论）

- **Spec 覆盖**：9 节点表 → Task 1+2（输入/输出/实现逐一对应，Round Digits clamp ±15 + SQL 约定、Clamp 交换、Pow/Sqrt 特殊值透传都有测试）；Expr 6 函数（round 1/2 参 + clamp 交换 + arg 错报错）→ Task 4；i18n（Round 示例+精度、Clamp 交换、Pow/Sqrt 特殊值说明）→ Task 3；Expr 函数清单文档（spec 落地清单第 4 条）→ 实查结果是 node-system-reference **没有** Expr 函数清单、真正的清单在 `node.Expr.description`（zh/en）和 `ExpressionInput.vue::FN_NAMES`——Task 4 覆盖，spec 漏的这两处已补进。验证命令 → Task 5。全覆盖。
- **类型一致**：`numXIn()` 在 Task 1 定义、Task 2 复用；`Round.Digits` 用 Integer（新 pin 名无分裂）、`Clamp.Min/Max` 用 Number（守卫要求，含专门测试钉住）；expr 侧 round/clamp 语义与节点侧一字一致（clamp [-15,15]、交换）。
- **无占位符**：所有步骤含完整代码/命令/预期。
- **已知非回归**：fish fixture 预存失败、i18n residue 28 预存失败，均已在步骤中标注。
