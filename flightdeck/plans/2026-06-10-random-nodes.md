---
status: active
summary: 随机节点 RandomInt/Float/Bool + per-dispatch 求值稳定的实现计划 (TDD 分任务)
last_updated: 2026-06-10
implements: specs/2026-06-10-random-nodes.md
---

# 随机数节点 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 加 RandomInt / RandomFloat / RandomBool 三个随机节点（含 uniform/centered 分布），并给框架加 per-dispatch 求值稳定，让随机在同一次节点求值内多路径引用拿到同值。

**Architecture:** 后端新建 `internal/nodes/random` 包（`IsPureData + IsNonDeterministic`，底层 `math/rand/v2`）；框架在 `node.Spec` 加 `IsNonDeterministic` 标志 + 在每个 exec 节点 dispatch 作用域挂一个 `dispatchEvalCache`（ctx 携带，与既有 `TickSnapshot` 并列），`evalDataSource` 对非确定节点的成功结果按 `{nodeID,pin}` 记忆化。前端新建 `Random` palette 分类（6 处注册点）。

**Tech Stack:** Go（`math/rand/v2`）、Wails、Vue 3 + Nuxt UI、vue-i18n、Taskfile。

**实现依据见 spec：** `flightdeck/specs/2026-06-10-random-nodes.md`。配套全链路规范 `flightdeck/checklists/add-node.md`。

---

## File Structure

**新建：**
- `internal/nodes/random/random.go` — 3 节点 + `bellUnit` helper + `init()` 注册。
- `internal/nodes/random/random_test.go` — 节点单测（属性/边界）。

**修改（后端）：**
- `internal/node/spec.go` — `Spec` 加 `IsNonDeterministic` 字段。
- `internal/services/container/runtime/snapshot.go` — 加 `evalKey` / `dispatchEvalCache` / `withEvalCache` / `evalCacheFromCtx`。
- `internal/services/container/runtime/dispatch_v5.go:328` — `withTickSnapshot` 旁并列 `withEvalCache`。
- `internal/services/container/runtime/data_pull.go::evalDataSource` — 加缓存 gate。
- `internal/services/container/runtime/eval_cache_test.go`（新建）— 框架缓存测试 + 测试用计数节点。
- 10 处 blank-import 站点加 `_ "yotta/internal/nodes/random"`（见 Task 5）。

**修改（前端 / i18n）：**
- `frontend/src/i18n/zh.ts` + `en.ts` — `node.RandomInt/RandomFloat/RandomBool` 块 + `nodeGroup.random`。
- `frontend/src/components/containers/nodeRegistry/index.ts` — `NodeGroup` 联合类型加 `'random'`。
- `frontend/src/components/containers/nodeRegistry/adapter.ts` — `GROUP_MAP` 加 `Random:'random'`。
- `frontend/src/components/containers/NodePalette.vue` — `GROUP_LABEL` 加 `random`。
- `frontend/src/composables/editor/useNodeGroupColor.ts` — `GROUP_I18N_KEY` 加 `random`。
- `frontend/src/components/containers/visualRegistry.ts` — `GROUP_VISUAL` 加 `random`。

---

## Task 1: Spec 加 `IsNonDeterministic` 字段

**Files:**
- Modify: `internal/node/spec.go:19`

- [ ] **Step 1: 加字段**

在 `internal/node/spec.go` 的 `Spec` 结构体里，`IsPureData` 字段下方加：

```go
	IsPureData   bool `json:"isPureData,omitempty"`
	// IsNonDeterministic — 节点 Evaluate 非确定 (如随机). 框架在 per-dispatch eval cache
	// 里记忆化其成功结果, 让同一 dispatch 内多路径引用拿同值 (守住 Determinism contract).
	// 仅在 IsPureData=true 节点上有意义 — 记忆化发生在 pure-data 拉取路径; exec 节点不读此字段.
	IsNonDeterministic bool `json:"isNonDeterministic,omitempty"`
	IsVisualOnly bool `json:"isVisualOnly,omitempty"`
```

- [ ] **Step 2: 编译验证**

Run: `go build ./internal/node/...`
Expected: 无报错。

- [ ] **Step 3: Commit**

```bash
git add internal/node/spec.go
git commit -m "feat(node): Spec 加 IsNonDeterministic 标志 (随机节点用)"
```

---

## Task 2: random 包 — `bellUnit` + RandomInt

**Files:**
- Create: `internal/nodes/random/random.go`
- Test: `internal/nodes/random/random_test.go`

- [ ] **Step 1: 写失败测试**

`internal/nodes/random/random_test.go`：

```go
package random

import (
	"testing"

	"yotta/internal/node"
)

// evalInt 用 config.literal 构造 Inputs 跑 Evaluate, 返回 int 结果.
func evalInt(t *testing.T, n node.Evaluator, cfg map[string]any) int {
	t.Helper()
	in := node.NewInputsFromConfig(map[string]any{"literal": cfg})
	v, err := n.Evaluate(nil, in)
	if err != nil {
		t.Fatalf("Evaluate err: %v", err)
	}
	got, ok := v.(int)
	if !ok {
		t.Fatalf("want int, got %T (%v)", v, v)
	}
	return got
}

func TestRandomInt_Spec_Flags(t *testing.T) {
	s := RandomInt{}.Spec()
	if !s.IsPureData || !s.IsNonDeterministic {
		t.Fatalf("RandomInt must be IsPureData+IsNonDeterministic, got %+v", s)
	}
	if s.Category != "Random" {
		t.Fatalf("Category = %q, want Random", s.Category)
	}
}

func TestRandomInt_Uniform_InClosedRange(t *testing.T) {
	for i := 0; i < 2000; i++ {
		got := evalInt(t, RandomInt{}, map[string]any{"Min": 3, "Max": 7})
		if got < 3 || got > 7 {
			t.Fatalf("uniform out of [3,7]: %d", got)
		}
	}
}

func TestRandomInt_Centered_InClosedRange(t *testing.T) {
	for i := 0; i < 2000; i++ {
		got := evalInt(t, RandomInt{}, map[string]any{"Min": 0, "Max": 10, "Distribution": "centered"})
		if got < 0 || got > 10 {
			t.Fatalf("centered out of [0,10]: %d", got)
		}
	}
}

func TestRandomInt_MinGtMax_Swaps(t *testing.T) {
	for i := 0; i < 500; i++ {
		got := evalInt(t, RandomInt{}, map[string]any{"Min": 9, "Max": 2})
		if got < 2 || got > 9 {
			t.Fatalf("swap out of [2,9]: %d", got)
		}
	}
}

func TestRandomInt_MinEqMax_ReturnsMin(t *testing.T) {
	got := evalInt(t, RandomInt{}, map[string]any{"Min": 5, "Max": 5})
	if got != 5 {
		t.Fatalf("Min==Max: want 5, got %d", got)
	}
}

func TestRandomInt_LargeRange_NoOverflow(t *testing.T) {
	// 接近 int32 边界的大区间, 证明 int64 内部运算常规安全.
	got := evalInt(t, RandomInt{}, map[string]any{"Min": -2000000000, "Max": 2000000000})
	if got < -2000000000 || got > 2000000000 {
		t.Fatalf("large range out of bounds: %d", got)
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `go test ./internal/nodes/random/ -run TestRandomInt -v`
Expected: 编译失败（`RandomInt` 未定义）。

- [ ] **Step 3: 写实现**

`internal/nodes/random/random.go`：

```go
// Package random — 非确定随机数节点 (RandomInt / RandomFloat / RandomBool).
// 全 IsPureData=true + IsNonDeterministic=true: 框架在 per-dispatch eval cache 里记忆化成功
// 结果, 同一求值内多路径引用拿同值 (见 specs/2026-06-10-random-nodes.md A 节). 底层 math/rand/v2.
package random

import (
	"encoding/json"
	"math/rand/v2"

	"yotta/internal/node"
)

func init() {
	for _, n := range []node.Node{&RandomInt{}, &RandomFloat{}, &RandomBool{}} {
		node.Register(n)
	}
}

// randomSamples — centered 分布的 Bates 样本数. 与 internal/node/jitter.go::jitterSamples
// 同为 5、同 CLT 思路 (独立维护, 不为一个 int 跨包耦合).
const randomSamples = 5

// bellUnit 返回 [0,1) 内向 0.5 聚集的样本 (Bates: randomSamples 个 uniform 求均值).
func bellUnit() float64 {
	var sum float64
	for i := 0; i < randomSamples; i++ {
		sum += rand.Float64()
	}
	return sum / randomSamples
}

const (
	distUniform  = "uniform"
	distCentered = "centered"
)

// distributionInput — uniform/centered 下拉 (Advanced).
func distributionInput() node.InputSpec {
	return node.InputSpec{
		Name: "Distribution", Type: "String", Default: distUniform, Advanced: true,
		Widget: node.WidgetSpec{Kind: "dropdown",
			Props: node.MarshalProps(node.DropdownProps{
				Options: []node.EnumOption{{Value: distUniform}, {Value: distCentered}}})},
	}
}

// ===== RandomInt =====

type RandomInt struct{}

func (RandomInt) Spec() node.Spec {
	return node.Spec{
		Kind: "RandomInt", Category: "Random",
		Inputs: []node.InputSpec{
			{Name: "Min", Type: "Integer", Default: json.Number("0"), Widget: node.WidgetSpec{Kind: "number"}},
			{Name: "Max", Type: "Integer", Default: json.Number("100"), Widget: node.WidgetSpec{Kind: "number"}},
			distributionInput(),
		},
		Outputs:            []node.OutputSpec{{Name: "Result", Type: "Integer"}},
		IsPureData:         true,
		IsNonDeterministic: true,
	}
}

func (RandomInt) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	lo, hi := int64(in.Int("Min")), int64(in.Int("Max"))
	if lo > hi {
		lo, hi = hi, lo
	}
	if lo == hi {
		return int(lo), nil
	}
	span := hi - lo + 1 // int64 防普通 int32 满区间溢出
	if in.String("Distribution") == distCentered {
		n := lo + int64(bellUnit()*float64(span))
		if n < lo { // 纯防御: Float64()∈[0,1) → 均值<1 → 理论不触发; 勿当冗余删
			n = lo
		}
		if n > hi {
			n = hi
		}
		return int(n), nil
	}
	return int(lo + rand.Int64N(span)), nil
}
```

- [ ] **Step 4: 运行确认通过**

Run: `go test ./internal/nodes/random/ -run TestRandomInt -v`
Expected: PASS（全部）。

- [ ] **Step 5: Commit**

```bash
git add internal/nodes/random/random.go internal/nodes/random/random_test.go
git commit -m "feat(random): RandomInt 节点 + bellUnit (uniform/centered)"
```

---

## Task 3: RandomFloat

**Files:**
- Modify: `internal/nodes/random/random.go`
- Test: `internal/nodes/random/random_test.go`

- [ ] **Step 1: 写失败测试**

追加到 `random_test.go`：

```go
func evalFloat(t *testing.T, n node.Evaluator, cfg map[string]any) float64 {
	t.Helper()
	in := node.NewInputsFromConfig(map[string]any{"literal": cfg})
	v, err := n.Evaluate(nil, in)
	if err != nil {
		t.Fatalf("Evaluate err: %v", err)
	}
	got, ok := v.(float64)
	if !ok {
		t.Fatalf("want float64, got %T (%v)", v, v)
	}
	return got
}

func TestRandomFloat_Spec_Flags(t *testing.T) {
	s := RandomFloat{}.Spec()
	if !s.IsPureData || !s.IsNonDeterministic || s.Category != "Random" {
		t.Fatalf("bad spec: %+v", s)
	}
}

func TestRandomFloat_Uniform_InHalfOpenRange(t *testing.T) {
	for i := 0; i < 2000; i++ {
		got := evalFloat(t, RandomFloat{}, map[string]any{"Min": 2.0, "Max": 5.0})
		if got < 2.0 || got >= 5.0 {
			t.Fatalf("out of [2,5): %v", got)
		}
	}
}

func TestRandomFloat_Centered_InRange(t *testing.T) {
	for i := 0; i < 2000; i++ {
		got := evalFloat(t, RandomFloat{}, map[string]any{"Min": 0.0, "Max": 1.0, "Distribution": "centered"})
		if got < 0.0 || got >= 1.0 {
			t.Fatalf("centered out of [0,1): %v", got)
		}
	}
}

func TestRandomFloat_MinEqMax_ReturnsMin(t *testing.T) {
	got := evalFloat(t, RandomFloat{}, map[string]any{"Min": 3.5, "Max": 3.5})
	if got != 3.5 {
		t.Fatalf("Min==Max: want 3.5, got %v", got)
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `go test ./internal/nodes/random/ -run TestRandomFloat -v`
Expected: 编译失败（`RandomFloat` 未定义）。

- [ ] **Step 3: 写实现**

追加到 `random.go`：

```go
// ===== RandomFloat =====

type RandomFloat struct{}

func (RandomFloat) Spec() node.Spec {
	return node.Spec{
		Kind: "RandomFloat", Category: "Random",
		Inputs: []node.InputSpec{
			{Name: "Min", Type: "Number", Default: json.Number("0"), Widget: node.WidgetSpec{Kind: "number"}},
			{Name: "Max", Type: "Number", Default: json.Number("1"), Widget: node.WidgetSpec{Kind: "number"}},
			distributionInput(),
		},
		Outputs:            []node.OutputSpec{{Name: "Result", Type: "Number"}},
		IsPureData:         true,
		IsNonDeterministic: true,
	}
}

func (RandomFloat) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	lo, hi := in.Float64("Min"), in.Float64("Max")
	if lo > hi {
		lo, hi = hi, lo
	}
	if lo == hi {
		return lo, nil
	}
	u := rand.Float64()
	if in.String("Distribution") == distCentered {
		u = bellUnit()
	}
	return lo + u*(hi-lo), nil
}
```

- [ ] **Step 4: 运行确认通过**

Run: `go test ./internal/nodes/random/ -run TestRandomFloat -v`
Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add internal/nodes/random/random.go internal/nodes/random/random_test.go
git commit -m "feat(random): RandomFloat 节点"
```

---

## Task 4: RandomBool

**Files:**
- Modify: `internal/nodes/random/random.go`
- Test: `internal/nodes/random/random_test.go`

- [ ] **Step 1: 写失败测试**

追加到 `random_test.go`：

```go
func evalBool(t *testing.T, n node.Evaluator, cfg map[string]any) bool {
	t.Helper()
	in := node.NewInputsFromConfig(map[string]any{"literal": cfg})
	v, err := n.Evaluate(nil, in)
	if err != nil {
		t.Fatalf("Evaluate err: %v", err)
	}
	got, ok := v.(bool)
	if !ok {
		t.Fatalf("want bool, got %T (%v)", v, v)
	}
	return got
}

func TestRandomBool_Spec_Flags(t *testing.T) {
	s := RandomBool{}.Spec()
	if !s.IsPureData || !s.IsNonDeterministic || s.Category != "Random" {
		t.Fatalf("bad spec: %+v", s)
	}
}

func TestRandomBool_ProbZero_AlwaysFalse(t *testing.T) {
	for i := 0; i < 500; i++ {
		if evalBool(t, RandomBool{}, map[string]any{"Prob": 0.0}) {
			t.Fatal("Prob=0 should always be false")
		}
	}
}

func TestRandomBool_ProbOne_AlwaysTrue(t *testing.T) {
	for i := 0; i < 500; i++ {
		if !evalBool(t, RandomBool{}, map[string]any{"Prob": 1.0}) {
			t.Fatal("Prob=1 should always be true")
		}
	}
}

func TestRandomBool_ProbHalf_BothSeen(t *testing.T) {
	var sawT, sawF bool
	for i := 0; i < 2000 && !(sawT && sawF); i++ {
		if evalBool(t, RandomBool{}, map[string]any{"Prob": 0.5}) {
			sawT = true
		} else {
			sawF = true
		}
	}
	if !sawT || !sawF {
		t.Fatalf("Prob=0.5 should yield both; sawT=%v sawF=%v", sawT, sawF)
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `go test ./internal/nodes/random/ -run TestRandomBool -v`
Expected: 编译失败（`RandomBool` 未定义）。

- [ ] **Step 3: 写实现**

追加到 `random.go`：

```go
// ===== RandomBool =====

type RandomBool struct{}

func (RandomBool) Spec() node.Spec {
	return node.Spec{
		Kind: "RandomBool", Category: "Random",
		Inputs: []node.InputSpec{
			{Name: "Prob", Type: "Number", Default: json.Number("0.5"), Widget: node.WidgetSpec{Kind: "number"}},
		},
		Outputs:            []node.OutputSpec{{Name: "Result", Type: "Bool"}},
		IsPureData:         true,
		IsNonDeterministic: true,
	}
}

// Evaluate — Prob 越界自然夹紧: Float64()∈[0,1) → Prob<=0 恒 false, Prob>=1 恒 true.
func (RandomBool) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	return rand.Float64() < in.Float64("Prob"), nil
}
```

- [ ] **Step 4: 运行确认通过**

Run: `go test ./internal/nodes/random/ -v`
Expected: PASS（全部 random 测试）。

- [ ] **Step 5: Commit**

```bash
git add internal/nodes/random/random.go internal/nodes/random/random_test.go
git commit -m "feat(random): RandomBool 节点"
```

---

## Task 5: blank-import 注册 + i18n + catalog 绿

> ⚠ 注册靠 blank import 触发 `init()`。`purefunc` 包在 **10 处**被 blank-import，random 必须**逐一镜像**，漏一处 → 对应 binary/测试看不到 random 节点。

**Files:**
- Modify（各加一行 `_ "yotta/internal/nodes/random"`，紧挨该文件已有的 `_ "yotta/internal/nodes/purefunc"`）：
  - `main.go`
  - `cmd/node-catalog/main.go`
  - `cmd/validate-fishing-v2/main.go`
  - `cmd/yotta-mcp/main.go`
  - `internal/catalog/catalog_test.go`
  - `internal/catalog/markdown_test.go`
  - `internal/node/spec_capture_test.go`
  - `internal/node/spec_consistency_test.go`
  - `internal/services/container/runtime/dispatch_v5_test.go`
  - `internal/services/container/setup_test.go`
- Modify: `frontend/src/i18n/zh.ts`、`frontend/src/i18n/en.ts`

- [ ] **Step 1: 加 10 处 blank import**

每个文件在已有 `_ "yotta/internal/nodes/purefunc"` 行下方加：

```go
	_ "yotta/internal/nodes/random" // RandomInt/RandomFloat/RandomBool
```

- [ ] **Step 2: zh.ts 加节点块**

在 `frontend/src/i18n/zh.ts` 的 `node:` 大块内（与 `RandomInt` 字母序相近处或紧跟 purefunc 节点后）加：

```ts
    RandomInt: {
      label: '随机整数', description: '在 Min~Max 之间随机取一个整数（含两端）。同一次求值里多处引用拿到同一个值；跨不同节点是不同值。',
      input: {
        Min: { label: '最小值' },
        Max: { label: '最大值' },
        Distribution: { label: '分布', hint: '均匀=各整数等概率；聚中=向区间中点聚集（固定 5 样本钟形，集中度不可调）', option: { uniform: '均匀', centered: '聚中' } },
      },
      output: { Result: { label: '结果' } },
    },
    RandomFloat: {
      label: '随机小数', description: '在 Min~Max 之间随机取一个小数（含 Min、不含 Max）。同一次求值里多处引用拿到同一个值；跨不同节点是不同值。',
      input: {
        Min: { label: '最小值' },
        Max: { label: '最大值' },
        Distribution: { label: '分布', hint: '均匀=区间内等概率；聚中=向区间中点聚集（固定 5 样本钟形，集中度不可调）', option: { uniform: '均匀', centered: '聚中' } },
      },
      output: { Result: { label: '结果' } },
    },
    RandomBool: {
      label: '随机真假', description: '按概率随机给真/假。概率 0.5 = 一半一半；≤0 恒假、≥1 恒真。',
      input: { Prob: { label: '为真概率' } },
      output: { Result: { label: '结果' } },
    },
```

并在 `nodeGroup:` 块加（`event` 行后）：

```ts
    random: '随机',
```

- [ ] **Step 3: en.ts 加镜像块**

在 `frontend/src/i18n/en.ts` 对应位置加：

```ts
    RandomInt: {
      label: 'Random Int', description: 'Random integer in [Min, Max] (both ends included). Multiple references within one evaluation get the same value; different nodes get different values.',
      input: {
        Min: { label: 'Min' },
        Max: { label: 'Max' },
        Distribution: { label: 'Distribution', hint: 'uniform = each integer equally likely; centered = clusters toward the midpoint (fixed 5-sample bell, spread not adjustable)', option: { uniform: 'Uniform', centered: 'Centered' } },
      },
      output: { Result: { label: 'Result' } },
    },
    RandomFloat: {
      label: 'Random Float', description: 'Random float in [Min, Max) (Min included, Max excluded). Multiple references within one evaluation get the same value; different nodes get different values.',
      input: {
        Min: { label: 'Min' },
        Max: { label: 'Max' },
        Distribution: { label: 'Distribution', hint: 'uniform = equal probability across the range; centered = clusters toward the midpoint (fixed 5-sample bell, spread not adjustable)', option: { uniform: 'Uniform', centered: 'Centered' } },
      },
      output: { Result: { label: 'Result' } },
    },
    RandomBool: {
      label: 'Random Bool', description: 'Random true/false by probability. 0.5 = fifty-fifty; <=0 always false, >=1 always true.',
      input: { Prob: { label: 'True probability' } },
      output: { Result: { label: 'Result' } },
    },
```

并在 `nodeGroup:` 块加：

```ts
    random: 'Random',
```

- [ ] **Step 4: 重新生成 catalog i18n**

Run: `cd frontend && pnpm gen:node-i18n`
Expected: 无报错，`internal/catalog/node-i18n.json` 被更新（含 RandomInt/RandomFloat/RandomBool）。

- [ ] **Step 5: i18n 校验 + catalog drift 绿**

Run: `cd frontend && pnpm i18n:check`
Expected: PASS（zh/en parity + compile 段无报错）。

Run: `go test ./internal/catalog/... ./internal/node/... -count=1`
Expected: PASS（drift 守卫认得 random 节点 + 有对应 i18n；`TestNoPinNameSplit` 不报新分裂）。

- [ ] **Step 6: Commit**

```bash
git add main.go cmd/ internal/catalog/ internal/node/ internal/services/container/ frontend/src/i18n/ internal/catalog/node-i18n.json
git commit -m "feat(random): 注册 random 包 (10 处 blank import) + zh/en i18n + nodeGroup.random"
```

---

## Task 6: 框架 per-dispatch 求值稳定

**Files:**
- Modify: `internal/services/container/runtime/snapshot.go`
- Modify: `internal/services/container/runtime/dispatch_v5.go:328`
- Modify: `internal/services/container/runtime/data_pull.go::evalDataSource`
- Test: `internal/services/container/runtime/eval_cache_test.go`（新建）

- [ ] **Step 1: 写失败测试**

`internal/services/container/runtime/eval_cache_test.go`：

```go
package runtime

import (
	"context"
	"sync/atomic"
	"testing"

	"yotta/internal/node"
	"yotta/internal/services/container"
	"yotta/internal/services/expr"
)

// 测试用计数源节点: 每次 Evaluate 自增. 用于确定性地探测记忆化是否生效.
var testCtrN atomic.Int64
var testCtrDetN atomic.Int64

type testCounter struct{}

func (testCounter) Spec() node.Spec {
	return node.Spec{
		Kind: "__TestCounter", Category: "PureFunc",
		Outputs:            []node.OutputSpec{{Name: "Result", Type: "Number"}},
		IsPureData:         true,
		IsNonDeterministic: true,
	}
}
func (testCounter) Evaluate(_ node.Ctx, _ node.Inputs) (any, error) {
	return float64(testCtrN.Add(1)), nil
}

type testCounterDet struct{}

func (testCounterDet) Spec() node.Spec {
	return node.Spec{
		Kind: "__TestCounterDet", Category: "PureFunc",
		Outputs:    []node.OutputSpec{{Name: "Result", Type: "Number"}},
		IsPureData: true, // 注意: 故意 IsNonDeterministic=false
	}
}
func (testCounterDet) Evaluate(_ node.Ctx, _ node.Inputs) (any, error) {
	return float64(testCtrDetN.Add(1)), nil
}

func init() {
	node.Register(&testCounter{})
	node.Register(&testCounterDet{})
}

// wireCounter 建一条 src.Result → sleep.Duration 的数据边.
func wireCounter(t *testing.T, srcKind string) *ContainerRunner {
	t.Helper()
	_, r := newTestRunner(t)
	src := &container.GraphNode{ID: "ctr", Kind: srcKind}
	dst := &container.GraphNode{ID: "sleep", Kind: "Sleep", Config: map[string]any{}}
	r.nodesByID = map[string]*container.GraphNode{"ctr": src, "sleep": dst}
	r.dataEdges = buildDataEdgeIndex(container.Graph{
		Nodes: []container.GraphNode{*src, *dst},
		Edges: []container.GraphEdge{{From: "ctr.Result", To: "sleep.Duration"}},
	})
	return r
}

func dispatchCtx() context.Context {
	ctx := withTickSnapshot(context.Background(), NewTickSnapshot())
	return withEvalCache(ctx, newDispatchEvalCache())
}

// 非确定节点: 同一 dispatch 内多次拉取 → 记忆化同值.
func TestEvalCache_NonDeterministic_MemoizedWithinDispatch(t *testing.T) {
	r := wireCounter(t, "__TestCounter")
	ctx := dispatchCtx()
	v1, _ := r.pullDataPin(ctx, "sleep", "Duration")
	v2, _ := r.pullDataPin(ctx, "sleep", "Duration")
	n1, _ := expr.AsNumber(v1)
	n2, _ := expr.AsNumber(v2)
	if n1 != n2 {
		t.Fatalf("same dispatch: want memoized equal, got %v vs %v", n1, n2)
	}
}

// 非确定节点: 跨 dispatch → 重新求值 (断言重算发生, 不断言"值不等").
func TestEvalCache_NonDeterministic_FreshAcrossDispatch(t *testing.T) {
	r := wireCounter(t, "__TestCounter")
	before := testCtrN.Load()
	v1, _ := r.pullDataPin(dispatchCtx(), "sleep", "Duration")
	v2, _ := r.pullDataPin(dispatchCtx(), "sleep", "Duration")
	_ = v1
	_ = v2
	if got := testCtrN.Load() - before; got != 2 {
		t.Fatalf("two dispatches should re-eval twice, got %d evals", got)
	}
}

// 确定性节点 (IsNonDeterministic=false): 同一 dispatch 内不走缓存 → 每次重算.
func TestEvalCache_Deterministic_NotCached(t *testing.T) {
	r := wireCounter(t, "__TestCounterDet")
	ctx := dispatchCtx()
	before := testCtrDetN.Load()
	r.pullDataPin(ctx, "sleep", "Duration")
	r.pullDataPin(ctx, "sleep", "Duration")
	if got := testCtrDetN.Load() - before; got != 2 {
		t.Fatalf("deterministic node must not be cached; want 2 evals, got %d", got)
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `go test ./internal/services/container/runtime/ -run TestEvalCache -v`
Expected: 编译失败（`withEvalCache` / `newDispatchEvalCache` 未定义）。

- [ ] **Step 3: snapshot.go 加缓存类型 + ctx helper**

在 `internal/services/container/runtime/snapshot.go` 末尾加：

```go
// evalKey — per-dispatch 求值缓存 key. 结构体 key: 零碰撞、零字符串拼接分配.
type evalKey struct{ nodeID, pin string }

// dispatchEvalCache — 单个 exec 节点 dispatch 作用域内的 pure-data 求值缓存.
// 只缓存 IsNonDeterministic 节点的成功结果 (见 evalDataSource), 让随机在同一求值内多路径稳定,
// 把随机纳入框架既有 Determinism contract.
//
// 并发: 一个实例只属一个 dispatch 的单 goroutine — 每个 dispatchInRegion 入口新建 (与
// TickSnapshot 同), Parallel/Race 是 exec 层并发、各节点各自新建, 故不跨 goroutine 共享.
// pure-data 拉取树同步执行. 普通 map 无需 mutex; 不变量靠 TestEvalCache_* 守护.
type dispatchEvalCache struct{ m map[evalKey]expr.Value }

func newDispatchEvalCache() *dispatchEvalCache {
	return &dispatchEvalCache{m: map[evalKey]expr.Value{}}
}

type evalCacheKeyT struct{}

var evalCacheKey = evalCacheKeyT{}

// withEvalCache 把 cache 挂到 ctx. dispatchInRegion 入口与 withTickSnapshot 并列调.
func withEvalCache(ctx context.Context, c *dispatchEvalCache) context.Context {
	return context.WithValue(ctx, evalCacheKey, c)
}

// evalCacheFromCtx 读 ctx 上的 cache. 没挂 (cold path) → nil.
func evalCacheFromCtx(ctx context.Context) *dispatchEvalCache {
	if ctx == nil {
		return nil
	}
	c, _ := ctx.Value(evalCacheKey).(*dispatchEvalCache)
	return c
}
```

- [ ] **Step 4: dispatch_v5.go 入口并列挂 cache**

在 `internal/services/container/runtime/dispatch_v5.go:328`，`withTickSnapshot` 那行下方加一行：

```go
	ctx = withTickSnapshot(ctx, CaptureSnapshot(r.rt.Vars()))
	ctx = withEvalCache(ctx, newDispatchEvalCache())
```

- [ ] **Step 5: evalDataSource 加缓存 gate**

在 `internal/services/container/runtime/data_pull.go::evalDataSource`，把末尾构造+求值那段：

```go
	srcDataWire := r.buildDataWireFor(ctx, n, rn)
	srcConfig := r.buildConfigFor(n)
	v, err := nodepkg.EvaluatePureData(ctx, rn, srcDataWire, srcConfig, r.bundle)
	return toExprValue(v), err
```

改为：

```go
	// per-dispatch 记忆化: 非确定节点 (随机) 同一 dispatch 内只求值一次, 守住 Determinism contract.
	// 确定性节点完全跳过此路径 (零影响). cold path (无 cache) → 退化为每 pull 重算.
	cache := evalCacheFromCtx(ctx)
	var key evalKey
	if rn.Spec.IsNonDeterministic && cache != nil {
		key = evalKey{nodeID: srcNodeID, pin: srcPin}
		if cached, ok := cache.m[key]; ok {
			return cached, nil
		}
	}
	srcDataWire := r.buildDataWireFor(ctx, n, rn)
	srcConfig := r.buildConfigFor(n)
	v, err := nodepkg.EvaluatePureData(ctx, rn, srcDataWire, srcConfig, r.bundle)
	ev := toExprValue(v)
	if err == nil && rn.Spec.IsNonDeterministic && cache != nil {
		cache.m[key] = ev // 只缓存成功值 — 不把"第一次失败"钉死
	}
	return ev, err
```

- [ ] **Step 6: 运行确认通过**

Run: `go test ./internal/services/container/runtime/ -run TestEvalCache -v`
Expected: PASS（3 个）。

- [ ] **Step 7: 跑全 runtime 包确认无回归**

Run: `go test ./internal/services/container/... -count=1`
Expected: PASS（既有确定性/快照测试不受影响）。

- [ ] **Step 8: Commit**

```bash
git add internal/services/container/runtime/snapshot.go internal/services/container/runtime/dispatch_v5.go internal/services/container/runtime/data_pull.go internal/services/container/runtime/eval_cache_test.go
git commit -m "feat(runtime): per-dispatch 求值缓存 — 非确定节点同 dispatch 稳定"
```

---

## Task 7: 前端 Random palette 分类

**Files:**
- Modify: `frontend/src/components/containers/nodeRegistry/index.ts:84`
- Modify: `frontend/src/components/containers/nodeRegistry/adapter.ts:27`
- Modify: `frontend/src/components/containers/NodePalette.vue:229`
- Modify: `frontend/src/composables/editor/useNodeGroupColor.ts:47`
- Modify: `frontend/src/components/containers/visualRegistry.ts:54`

- [ ] **Step 1: NodeGroup 联合类型加 'random'**

`index.ts` 的 `export type NodeGroup =` 列表末尾加：

```ts
  | 'event'
  | 'random'
```

- [ ] **Step 2: GROUP_MAP 加映射**

`adapter.ts` 的 `GROUP_MAP` 加：

```ts
  Event: 'event',
  Random: 'random',
}
```

- [ ] **Step 3: GROUP_LABEL 加分组标题**

`NodePalette.vue` 的 `GROUP_LABEL` 加：

```ts
  event: t('nodeGroup.event'),
  random: t('nodeGroup.random'),
}))
```

- [ ] **Step 4: GROUP_I18N_KEY 加映射**

`useNodeGroupColor.ts` 的 `GROUP_I18N_KEY` 加：

```ts
  event: 'nodeGroup.event',
  random: 'nodeGroup.random',
  misc: 'nodeGroup.other',
}
```

- [ ] **Step 5: GROUP_VISUAL 加图标+色**

`visualRegistry.ts` 的 `GROUP_VISUAL` 加（`teal` 是现有分组未占用色，`i-tabler-dice` 语义贴近随机）：

```ts
  event: { color: 'rose', icon: 'i-tabler-player-play' },
  random: { color: 'teal', icon: 'i-tabler-dice' },
}
```

- [ ] **Step 6: typecheck**

Run: `cd frontend && pnpm typecheck`
Expected: PASS（`Record<NodeGroup,...>` 的 exhaustive 已补齐，无 TS 报错）。

- [ ] **Step 7: Commit**

```bash
git add frontend/src/components/containers/ frontend/src/composables/editor/useNodeGroupColor.ts
git commit -m "feat(random): 前端 Random palette 分类 (GROUP_MAP/LABEL/I18N_KEY/visual)"
```

---

## Task 8: 全量验证 + 真机 smoke

**Files:** 无（仅验证）

- [ ] **Step 1: 后端全绿**

Run: `go build ./... && go test ./internal/nodes/... ./internal/node/... ./internal/catalog/... ./internal/services/container/... -count=1`
Expected: PASS。

- [ ] **Step 2: 前端全绿**

Run: `cd frontend && pnpm gen:node-i18n && pnpm typecheck && pnpm i18n:check`
Expected: 全 PASS（gen 后无新 drift）。

- [ ] **Step 3: catalog 自查节点在册**

Run: `task nodes` （或 `go run ./cmd/node-catalog export --md`）
Expected: 输出含 `RandomInt` / `RandomFloat` / `RandomBool`，归 `Random` category，pin/出口/默认值正确。

- [ ] **Step 4: 构建**

Run: `task build`
Expected: 成功产出 artifact（注: 若撞已知 runtime fish fixture 预存失败，见 `flightdeck/checklists/build.md`，非本次回归）。

- [ ] **Step 5: 真机 smoke（人工）**

启动 app → 打开容器编辑器：
- **看什么**：侧边 NodePalette、画布右键菜单、explorer 弹窗三处都出现 **「随机」分组**，含 RandomInt/RandomFloat/RandomBool 三个节点，分组色/图标正常。
- **怎么验**：拖一个 RandomInt 进画布 → Min=1/Max=6 → Distribution 下拉能切 均匀/聚中 → 连到一个 Log 的输入 → 跑一次容器 → Log 输出落在 1~6。
- **什么算过**：三处都能找到、能加、默认值/文案/分组标签/下拉选项都对、运行输出在区间内。

- [ ] **Step 6: Commit（如有 smoke 修正）**

```bash
git add -A
git commit -m "chore(random): smoke 修正"
```

---

## Self-Review（写完计划的自查结论）

- **Spec 覆盖**：A 框架机制 → Task 1+6；B 三节点（含 centered/边界/IsNonDeterministic）→ Task 2/3/4；Random 分类全注册 → Task 5(i18n)+Task 7(前端)；blank-import 10 处 → Task 5；验证命令 → Task 8。全覆盖。
- **类型一致**：`IsNonDeterministic`（spec.go）↔ 三节点 Spec ↔ evalDataSource gate 读 `rn.Spec.IsNonDeterministic`；`dispatchEvalCache`/`withEvalCache`/`newDispatchEvalCache`/`evalCacheFromCtx`/`evalKey` 在 Task 6 定义并在测试/gate 一致使用；`distUniform`/`distCentered` 常量贯穿。
- **无占位符**：所有代码步骤含完整 Go/TS 代码与确切命令、预期。
- **已知非回归**：`task build` 的 fish fixture 预存失败见 build.md，不阻断本计划。
