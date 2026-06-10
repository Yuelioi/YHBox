---
status: done
summary: List 类型 + in.List + LooseEqual 提升 + ForEach + 7 列表节点 + RandomChoice 的实现计划 (TDD 分任务, A' 审计已过) — 已实现 (d61e1d5..7425bca), 终审 SHIP
last_updated: 2026-06-10
implements: specs/2026-06-10-collection-nodes.md
---

# 数组/集合节点 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 给框架加 `List` pin 类型（`[]any`）+ `in.List` getter + ForEach 遍历 + Split/Join/ListLength/ListGet/ListContains/ListAppend/ListSlice 七个列表节点 + RandomChoice。

**Architecture:** A' 审计已过（结论见 spec），零架构阻塞。框架面：types.go 注册 List、inputs 加 getter、`toExprValue`/`canonPinType` 显式化、`equalAny`/`formatValue` 提升为 `node.LooseEqual`/`node.FormatValue`（唯一行为修正：不可比类型防护，直比 slice/map 是 panic）。ForEach 进 `nodes/control`（RegionRunner，复用 `makeBodyForLoop` —— 其实体只是 seed `node.ID+".Body"`，与 Loop 零差异，不按 spec 原文再抄一份）。列表节点进新 `internal/nodes/collection` 包。前端：palette `list` 分组 6 站点 + **类型词表三连**（PinType/TYPE_COLOR/adapter，审计抓出的必改）+ List pin 只读"由连线提供"占位。

**Tech Stack:** Go、Wails、Vue 3 + Nuxt UI、vue-i18n。

**实现依据：** spec `flightdeck/specs/2026-06-10-collection-nodes.md`（含 A' 审计结论）。配套 `flightdeck/checklists/add-node.md`。

---

## File Structure

**新建（后端）：**
- `internal/node/value_helpers.go` + `value_helpers_test.go` — LooseEqual/FormatValue（从 purefunc 提升）。
- `internal/nodes/collection/collection.go` + `collection_test.go` — 7 列表节点。
- `internal/nodes/control/foreach.go` + `foreach_test.go` — ForEach RegionRunner。

**修改（后端）：**
- `internal/node/types.go` — init 内置类型表加 List。
- `internal/node/interfaces.go:64` + `internal/node/inputs.go` — `List(name) []any`。
- `internal/services/container/runtime/data_pull.go::toExprValue` — 显式 `case []any`。
- `internal/services/container/validate.go::canonPinType` — 显式 "List" case。
- `internal/services/container/runtime/dispatch_v5.go::makeBodyFor` — `case "Loop", "ForEach"`。
- `internal/nodes/purefunc/purefunc.go` — 删 `formatValue`/`equalAny`/`sameType`/`typeNameOf`，调用点切 `node.FormatValue`/`node.LooseEqual`。
- `internal/nodes/random/random.go` — RandomChoice。
- 10 处 blank-import 站点加 `_ "yotta/internal/nodes/collection"`。

**修改（前端 / i18n / docs）：**
- `frontend/src/components/containers/nodeRegistry/index.ts` — `PinType` 联合 + `TYPE_COLOR` 加 `list`；`NodeGroup` 联合加 `'list'`。
- `frontend/src/components/containers/nodeRegistry/adapter.ts` — `backendTypeToPinType` 加 list case；`GROUP_MAP` 加 `List:'list'`。
- `frontend/src/components/containers/NodePalette.vue` — GROUP_LABEL + KINDS_BY_GROUP 加 list。
- `frontend/src/composables/editor/useNodeGroupColor.ts` — GROUP_I18N_KEY + ALL_NODE_GROUPS 加 list。
- `frontend/src/components/containers/visualRegistry.ts` — GROUP_VISUAL 加 list。
- `frontend/src/components/containers/inline/PinLiteral.vue` + `PinInput.vue`（NodeInspector 用）— list 型只读占位。
- `frontend/src/i18n/zh.ts` + `en.ts` — 9 节点块 + `nodeGroup.list` + 占位文案。
- `flightdeck/docs/node-system-reference.md` — List 分类行 + Random 行 +RandomChoice。

**已核源码事实（撞不一致停下报告）：**
- types.go init 内置类型表 :84-99（直接加条目，**不要**在别处调 RegisterType——内置类型归 init 表）。
- `Inputs` 接口 interfaces.go:59-75（`StringList` 在 :64，List 加其后）；实现 inputs.go（StringList :76-94 是参考形态）。
- Loop 全文 internal/nodes/control/loop.go；sentinel `errBreakRequested`/`errContinueRequested`（sentinels.go:17-21，同包可直用）；Loop 测试范式 loop_test.go（`node.RunNodeAsRegion(ctx, rn, dataWire, config, execData, services, false, body)` + `recVars` 记录式 VarStore）。
- `makeBodyFor` dispatch_v5.go:414-422；`makeBodyForLoop` :426-432 —— 实体仅 `seeds := r.edges.next(node.ID+".Body", parentLoopStack); r.runRegionBody(...)`，无 Loop 特有逻辑 → **ForEach 直接复用**（对 spec"逐行同构"的 DRY 修正，已批）。
- `equalAny`/`sameType`/`typeNameOf` purefunc.go:265-285；`formatValue` :106-125；purefunc 内调用点：Concat/Contains/Length(`formatValue`)、Eq/NotEq(`equalAny`)。
- `toExprValue` data_pull.go:166-191（`[]any` 现走 default）；`canonPinType` validate.go:155-170（unknown→lowercase，碰巧得 "list"——显式化）。
- 前端：`PinType`/`TYPE_COLOR` nodeRegistry/index.ts:7-16；`pinTypeCompat` :25-30（同型放行已覆盖 list→list，**无需改**；且它是死代码未接 vue-flow——预存缺口不接）；`backendTypeToPinType` adapter.ts:84-100（default→'any'）；palette 6 站点同 random 阶段（NodeGroup 联合/GROUP_MAP/GROUP_LABEL/KINDS_BY_GROUP/GROUP_I18N_KEY+ALL_NODE_GROUPS/GROUP_VISUAL）。
- GROUP_VISUAL 已用色：blue/cyan/lime/violet/orange/zinc/sky/amber/pink/rose/teal → **list 用 `indigo`**（贴 #818cf8）+ `i-tabler-list`。
- CaptureType 白名单（spec_capture_test.go:22-27）已含 "any" —— ForEach 的 CaptureItem 用 "any"，不扩白名单。
- A' 审计 caveats（文档化即可，不写代码）：IncVar 对 list 变量静默改写（GIGO）；列表进 Expr 干净报错但被数据线吞；存列表的变量/子图参数声明 **any** 型。

---

## Task 1: List 类型 + in.List getter + 透传显式化

**Files:**
- Modify: `internal/node/types.go`（init 表）
- Modify: `internal/node/interfaces.go:64`、`internal/node/inputs.go`
- Modify: `internal/services/container/runtime/data_pull.go::toExprValue`
- Modify: `internal/services/container/validate.go::canonPinType`
- Test: `internal/node/inputs_test.go`（若无此文件则新建；先 grep 现有 inputs 测试所在文件，追加为先）

- [ ] **Step 1: 写失败测试**

在 `internal/node` 包的 inputs 测试文件追加（文件名以实际为准）：

```go
func TestInputs_List(t *testing.T) {
	cases := []struct {
		name string
		val  any
		want []any
	}{
		{"any_slice", []any{1.0, "a"}, []any{1.0, "a"}},
		{"string_slice", []string{"a", "b"}, []any{"a", "b"}},
		{"nil", nil, nil},
		{"bare_string_not_list", "a,b", nil}, // 与 StringList 区别: 不把裸 string 当一元列表
		{"number_not_list", 3.14, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := NewInputsFromConfig(map[string]any{"literal": map[string]any{"X": tc.val}})
			got := in.List("X")
			if len(got) != len(tc.want) {
				t.Fatalf("len = %d, want %d (%v)", len(got), len(tc.want), got)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("[%d] = %v, want %v", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestListTypeRegistered(t *testing.T) {
	for _, ts := range AllTypes() {
		if ts.Tag == "List" {
			if ts.GoType != "[]any" || ts.Color != "#818cf8" || ts.WidgetKind != "list-preview" {
				t.Fatalf("List TypeSpec wrong: %+v", ts)
			}
			return
		}
	}
	t.Fatal("List type not registered")
}
```

- [ ] **Step 2: 运行确认失败**

Run: `go test ./internal/node/ -run "TestInputs_List|TestListTypeRegistered" -v`
Expected: 编译失败（接口无 `List` 方法）或 FAIL（类型未注册）。

- [ ] **Step 3: 实现**

a. `types.go` init 表（`JSON` 行后）加：

```go
		{Tag: "List", GoType: "[]any", WidgetKind: "list-preview", Color: "#818cf8"},
```

b. `interfaces.go` `Inputs` 接口 `StringList` 行后加：

```go
	// List 读 List 型 pin ([]any). 非列表/nil → nil. 不把裸 string 当一元列表 (与 StringList 区别).
	List(name string) []any
```

c. `inputs.go`（`StringList` 实现后）加：

```go
// List 读 List 型 pin. 容忍 []any (原样) / []string (转 []any, 一次性分配不缓存);
// nil 及其它任意类型 → nil (不 panic). 不把裸 string 当一元列表 (与 StringList 区别).
func (i *inputsImpl) List(name string) []any {
	switch v := i.merged[name].(type) {
	case []any:
		return v
	case []string:
		out := make([]any, len(v))
		for idx, s := range v {
			out[idx] = s
		}
		return out
	}
	return nil
}
```

（若有其它 `Inputs` 实现体编译报缺方法，按同语义补齐并报告。）

d. `data_pull.go::toExprValue` 在 `case map[string]any:` 之前加：

```go
	case []any:
		// List pin 值 — 原样透传 (之前靠 default 碰巧透传, 显式化防回归).
		return x
```

e. `validate.go::canonPinType` 加显式 case（按该函数现有 case 风格，在已知类型组里加）：

```go
	case "List":
		return "list"
```

- [ ] **Step 4: 运行确认通过**

Run: `go test ./internal/node/ ./internal/services/container/... -count=1`
Expected: PASS（fish-fixture 预存除外）。

- [ ] **Step 5: Commit**

```bash
git add internal/node/ internal/services/container/
git commit -m "feat(framework): List pin 类型 + in.List getter + toExprValue/canonPinType 显式化"
```

---

## Task 2: LooseEqual / FormatValue 提升（含不可比类型防护）

**Files:**
- Create: `internal/node/value_helpers.go`、`internal/node/value_helpers_test.go`
- Modify: `internal/nodes/purefunc/purefunc.go`（删 4 个 helper、切调用点）

- [ ] **Step 1: 写失败测试**

`internal/node/value_helpers_test.go`：

```go
package node

import "testing"

func TestFormatValue(t *testing.T) {
	cases := []struct {
		in   any
		want string
	}{
		{nil, "null"},
		{true, "true"},
		{false, "false"},
		{3.14, "3.14"},
		{42, "42"},
		{int64(7), "7"},
		{"s", "s"},
	}
	for _, tc := range cases {
		if got := FormatValue(tc.in); got != tc.want {
			t.Errorf("FormatValue(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestLooseEqual(t *testing.T) {
	if !LooseEqual(nil, nil) {
		t.Error("nil,nil want true")
	}
	if LooseEqual(nil, "") {
		t.Error(`nil,"" want false (FormatValue(nil)="null")`)
	}
	if !LooseEqual(1.0, 1.0) || LooseEqual(1.0, 2.0) {
		t.Error("same-type compare broken")
	}
	if !LooseEqual(1.0, "1") {
		t.Error(`cross-type 1.0 vs "1" want true (串比)`)
	}
}

// 防护: slice/map 同类型直比是 Go 运行时 panic — LooseEqual 必须退 FormatValue 串比.
func TestLooseEqual_UncomparableNoPanic(t *testing.T) {
	if !LooseEqual([]any{1, "a"}, []any{1, "a"}) {
		t.Error("equal slices (string-repr) want true")
	}
	if LooseEqual([]any{1}, []any{2}) {
		t.Error("different slices want false")
	}
	if !LooseEqual(map[string]any{"k": 1}, map[string]any{"k": 1}) {
		t.Error("equal maps (string-repr) want true")
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `go test ./internal/node/ -run "TestFormatValue|TestLooseEqual" -v`
Expected: 编译失败（`FormatValue`/`LooseEqual` 未定义）。

- [ ] **Step 3: 写实现**

`internal/node/value_helpers.go`：

```go
// value_helpers.go — 跨包共享的宽松值语义 (Eq/NotEq/ListContains/Join/Concat/ToString 共用).
// 从 purefunc 提升 (specs/2026-06-10-collection-nodes.md A.4). 行为唯一修正: 不可比类型防护
// (A' 审计: 同类型 slice/map 直比是运行时 panic, 被 EvaluatePureData recover 吞成静默 nil).
package node

import (
	"fmt"
	"reflect"
	"strconv"
)

// FormatValue 软转任意值为 string. 与 expr 包数值/Point 比较是两套语义, 勿混.
func FormatValue(v any) string {
	switch x := v.(type) {
	case nil:
		return "null"
	case bool:
		if x {
			return "true"
		}
		return "false"
	case float64:
		return strconv.FormatFloat(x, 'g', -1, 64)
	case int:
		return strconv.Itoa(x)
	case int64:
		return strconv.FormatInt(x, 10)
	case string:
		return x
	}
	return fmt.Sprintf("%v", v)
}

// LooseEqual 宽松等值 — 同类型直比, 跨类型 FormatValue 串比.
// 同类型但不可比 (slice/map) → FormatValue 串比 (直比 panic, 防护性退化).
// nil 语义: LooseEqual(nil,nil)=true; LooseEqual(nil,"")=false (FormatValue(nil)="null").
func LooseEqual(a, b any) bool {
	if looseSameType(a, b) {
		if a == nil || reflect.TypeOf(a).Comparable() {
			return a == b
		}
		return FormatValue(a) == FormatValue(b)
	}
	return FormatValue(a) == FormatValue(b)
}

func looseSameType(a, b any) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return fmt.Sprintf("%T", a) == fmt.Sprintf("%T", b)
}
```

- [ ] **Step 4: purefunc 切换调用点（一次性切干净, 不留 shim）**

`purefunc.go`：
1. 删除 `formatValue`（:106-125）、`equalAny`/`sameType`/`typeNameOf`（:265-285）。
2. 调用点替换：`formatValue(` → `node.FormatValue(`（Concat/Contains/Length 共 4 处）；`equalAny(` → `node.LooseEqual(`（Eq/NotEq 2 处）。
3. 若 `strconv` 因此不再被 purefunc.go 使用，从 import 删（`asNumber`/ToNumber 可能还用——以编译为准）。

- [ ] **Step 5: 回归 + 新测试通过**

Run: `go test ./internal/node/ ./internal/nodes/purefunc/ -count=1`
Expected: PASS——既有 Eq/Concat/ToString/Length 测试输出逐字不变（搬移零行为变更；防护只影响原本 panic 的输入）。

- [ ] **Step 6: Commit**

```bash
git add internal/node/value_helpers.go internal/node/value_helpers_test.go internal/nodes/purefunc/purefunc.go
git commit -m "refactor(node): equalAny/formatValue 提升为 node.LooseEqual/FormatValue (加不可比类型防护)"
```

---

## Task 3: collection 包 — Split/Join/ListLength/ListGet

**Files:**
- Create: `internal/nodes/collection/collection.go`、`internal/nodes/collection/collection_test.go`

- [ ] **Step 1: 写失败测试**

`collection_test.go`：

```go
package collection

import (
	"context"
	"testing"

	"yotta/internal/node"
)

// evalNode — EvaluatePureData 路径 (同 purefunc math_test 范式).
func evalNode(t *testing.T, n node.Node, dataWire map[string]any) any {
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

func wantList(t *testing.T, got any, want []any) {
	t.Helper()
	l, ok := got.([]any)
	if !ok {
		t.Fatalf("want []any, got %T(%v)", got, got)
	}
	if len(l) != len(want) {
		t.Fatalf("len = %d (%v), want %d (%v)", len(l), l, len(want), want)
	}
	for i := range l {
		if l[i] != want[i] {
			t.Fatalf("[%d] = %v, want %v", i, l[i], want[i])
		}
	}
}

func TestSplit(t *testing.T) {
	wantList(t, evalNode(t, &Split{}, map[string]any{"Text": "a,b,c", "Separator": ","}), []any{"a", "b", "c"})
	// Text="" → 空列表 (刻意偏离 Go Split("",sep)=[""])
	wantList(t, evalNode(t, &Split{}, map[string]any{"Text": "", "Separator": ","}), []any{})
	// Separator="" → 按 UTF-8 字符逐个拆 (CJK 安全)
	wantList(t, evalNode(t, &Split{}, map[string]any{"Text": "中a", "Separator": ""}), []any{"中", "a"})
	// 默认分隔符 ","
	wantList(t, evalNode(t, &Split{}, map[string]any{"Text": "x,y"}), []any{"x", "y"})
}

func TestJoin(t *testing.T) {
	got := evalNode(t, &Join{}, map[string]any{"List": []any{"a", 1.5, true, nil}, "Separator": "-"})
	if got != "a-1.5-true-null" {
		t.Fatalf("Join = %v, want a-1.5-true-null", got)
	}
	if got := evalNode(t, &Join{}, map[string]any{"List": []any{}}); got != "" {
		t.Fatalf("empty list join = %v, want \"\"", got)
	}
}

func TestListLength(t *testing.T) {
	if got := evalNode(t, &ListLength{}, map[string]any{"List": []any{1, 2, 3}}); got != 3.0 {
		t.Fatalf("len = %v, want 3", got)
	}
	// 非列表 → in.List 返 nil → 0
	if got := evalNode(t, &ListLength{}, map[string]any{"List": "not a list"}); got != 0.0 {
		t.Fatalf("non-list len = %v, want 0", got)
	}
}

func TestListGet(t *testing.T) {
	lst := []any{"a", nil, "c"}
	if got := evalNode(t, &ListGet{}, map[string]any{"List": lst, "Index": 0}); got != "a" {
		t.Fatalf("get[0] = %v", got)
	}
	// 元素本身是 nil → nil (与越界同输出, spec 已声明歧义)
	if got := evalNode(t, &ListGet{}, map[string]any{"List": lst, "Index": 1}); got != nil {
		t.Fatalf("get[1] = %v, want nil", got)
	}
	// 越界 / 负索引 → nil
	if got := evalNode(t, &ListGet{}, map[string]any{"List": lst, "Index": 99}); got != nil {
		t.Fatalf("get[99] = %v, want nil", got)
	}
	if got := evalNode(t, &ListGet{}, map[string]any{"List": lst, "Index": -1}); got != nil {
		t.Fatalf("get[-1] = %v, want nil (不做负索引)", got)
	}
}

func TestCollection_SpecShape(t *testing.T) {
	for _, n := range []node.Node{&Split{}, &Join{}, &ListLength{}, &ListGet{}} {
		s := n.Spec()
		if !s.IsPureData || s.Category != "List" {
			t.Fatalf("%s: must be IsPureData + Category List, got %+v", s.Kind, s)
		}
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `go test ./internal/nodes/collection/ -v`
Expected: 编译失败（`Split` 未定义）。

- [ ] **Step 3: 写实现**

`collection.go`：

```go
// Package collection 列表纯数据节点 (7 个): Split/Join/ListLength/ListGet/ListContains/
// ListAppend/ListSlice. 见 specs/2026-06-10-collection-nodes.md C 节. Category "List".
// 元素不分类型 ([]any); 越界/非列表一律安全值 (nil/空/0), 从不 error.
package collection

import (
	"encoding/json"
	"strings"

	"yotta/internal/node"
)

func init() {
	for _, n := range []node.Node{
		&Split{}, &Join{}, &ListLength{}, &ListGet{},
		&ListContains{}, &ListAppend{}, &ListSlice{},
	} {
		node.Register(n)
	}
}

// listSpec 单 Result 出口的 List 分类 pure-data Spec (同 purefunc.specBuilder 思路, 包内自持).
func listSpec(kind string, inputs []node.InputSpec, resultType string) node.Spec {
	return node.Spec{
		Kind: kind, Category: "List",
		Inputs:     inputs,
		Outputs:    []node.OutputSpec{{Name: "Result", Type: resultType}},
		IsPureData: true,
	}
}

func listIn() node.InputSpec {
	return node.InputSpec{Name: "List", Type: "List"}
}

func sepIn() node.InputSpec {
	return node.InputSpec{Name: "Separator", Type: "String", Default: ",", Widget: node.WidgetSpec{Kind: "text"}}
}

// ===== Split =====

type Split struct{}

func (Split) Spec() node.Spec {
	return listSpec("Split", []node.InputSpec{
		{Name: "Text", Type: "String", Default: "", Widget: node.WidgetSpec{Kind: "text"}},
		sepIn(),
	}, "List")
}

// Evaluate — Text=="" → 空列表 (刻意偏离 Go Split("",sep) 的 [""], 更直觉);
// Separator=="" → 按 UTF-8 字符逐个拆 (Go strings.Split 语义, rune 边界).
func (Split) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	text := in.String("Text")
	if text == "" {
		return []any{}, nil
	}
	parts := strings.Split(text, in.String("Separator"))
	out := make([]any, len(parts))
	for i, p := range parts {
		out[i] = p
	}
	return out, nil
}

// ===== Join =====

type Join struct{}

func (Join) Spec() node.Spec {
	return listSpec("Join", []node.InputSpec{listIn(), sepIn()}, "String")
}
func (Join) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	items := in.List("List")
	parts := make([]string, len(items))
	for i, el := range items {
		parts[i] = node.FormatValue(el)
	}
	return strings.Join(parts, in.String("Separator")), nil
}

// ===== ListLength =====

type ListLength struct{}

func (ListLength) Spec() node.Spec {
	return listSpec("ListLength", []node.InputSpec{listIn()}, "Number")
}
func (ListLength) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	return float64(len(in.List("List"))), nil
}

// ===== ListGet =====

type ListGet struct{}

func (ListGet) Spec() node.Spec {
	return listSpec("ListGet", []node.InputSpec{
		listIn(),
		{Name: "Index", Type: "Integer", Default: json.Number("0"), Widget: node.WidgetSpec{Kind: "number"}},
	}, "*")
}

// Evaluate — 越界 (含负) → nil (不做负索引). nil 歧义 (元素=nil vs 越界) spec 已声明: 要区分先 ListLength.
func (ListGet) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	items := in.List("List")
	idx := in.Int("Index")
	if idx < 0 || idx >= len(items) {
		return nil, nil
	}
	return items[idx], nil
}
```

- [ ] **Step 4: 运行确认通过**

Run: `go test ./internal/nodes/collection/ -count=1`
Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add internal/nodes/collection/
git commit -m "feat(collection): Split/Join/ListLength/ListGet 节点"
```

---

## Task 4: collection 包 — ListContains/ListAppend/ListSlice

**Files:**
- Modify: `internal/nodes/collection/collection.go`
- Test: `internal/nodes/collection/collection_test.go`

- [ ] **Step 1: 写失败测试**

追加：

```go
func TestListContains(t *testing.T) {
	lst := []any{1.0, "b", nil}
	cases := []struct {
		name  string
		value any
		want  bool
	}{
		{"same_type", "b", true},
		{"cross_type_strcmp", "1", true}, // 与 Eq 节点同语义: 跨类型串比
		{"nil_element", nil, true},       // LooseEqual(nil,nil)=true
		{"absent", "x", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := evalNode(t, &ListContains{}, map[string]any{"List": lst, "Value": tc.value})
			if got != tc.want {
				t.Fatalf("Contains(%v) = %v, want %v", tc.value, got, tc.want)
			}
		})
	}
	// 嵌套子列表元素 — 防护回归 (直比 panic 路径), 串比语义
	nested := []any{[]any{1, 2}}
	if got := evalNode(t, &ListContains{}, map[string]any{"List": nested, "Value": []any{1, 2}}); got != true {
		t.Fatalf("nested contains = %v, want true (串比)", got)
	}
	// nil vs "" 不等 (FormatValue(nil)="null")
	if got := evalNode(t, &ListContains{}, map[string]any{"List": []any{nil}, "Value": ""}); got != false {
		t.Fatalf(`Contains([nil], "") = %v, want false`, got)
	}
}

func TestListAppend(t *testing.T) {
	orig := []any{"a"}
	got := evalNode(t, &ListAppend{}, map[string]any{"List": orig, "Item": "b"})
	wantList(t, got, []any{"a", "b"})
	// 必 copy: 原列表不被改 (防 append 别名上游切片)
	if len(orig) != 1 || orig[0] != "a" {
		t.Fatalf("orig mutated: %v", orig)
	}
	// 空/非列表 → 单元素新列表
	wantList(t, evalNode(t, &ListAppend{}, map[string]any{"List": nil, "Item": "x"}), []any{"x"})
}

func TestListSlice(t *testing.T) {
	lst := []any{"a", "b", "c", "d"}
	// Count 默认 -1 → 取到末尾
	wantList(t, evalNode(t, &ListSlice{}, map[string]any{"List": lst, "Start": 1}), []any{"b", "c", "d"})
	// Count=0 → 空
	wantList(t, evalNode(t, &ListSlice{}, map[string]any{"List": lst, "Start": 1, "Count": 0}), []any{})
	// Count>0 → N 个, 超尾截断
	wantList(t, evalNode(t, &ListSlice{}, map[string]any{"List": lst, "Start": 2, "Count": 99}), []any{"c", "d"})
	// Start>=len → 恒空 (Count 忽略)
	wantList(t, evalNode(t, &ListSlice{}, map[string]any{"List": lst, "Start": 9, "Count": 2}), []any{})
	// 负 Start → 0
	wantList(t, evalNode(t, &ListSlice{}, map[string]any{"List": lst, "Start": -3, "Count": 2}), []any{"a", "b"})
	// copy: 改结果不影响原列表 — 用长度断言间接验 (返回的是新底层数组)
	got := evalNode(t, &ListSlice{}, map[string]any{"List": lst, "Start": 0, "Count": 2}).([]any)
	got[0] = "Z"
	if lst[0] != "a" {
		t.Fatalf("slice aliased original: %v", lst)
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `go test ./internal/nodes/collection/ -run "TestListContains|TestListAppend|TestListSlice" -v`
Expected: 编译失败。

- [ ] **Step 3: 写实现**

追加到 `collection.go`：

```go
// ===== ListContains =====

type ListContains struct{}

func (ListContains) Spec() node.Spec {
	return listSpec("ListContains", []node.InputSpec{
		listIn(),
		{Name: "Value", Type: "*"},
	}, "Bool")
}

// Evaluate — 与 Eq 节点完全同语义 (node.LooseEqual): 同类型直比、跨类型串比.
func (ListContains) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	val := in.Raw("Value")
	for _, el := range in.List("List") {
		if node.LooseEqual(el, val) {
			return true, nil
		}
	}
	return false, nil
}

// ===== ListAppend =====

type ListAppend struct{}

func (ListAppend) Spec() node.Spec {
	return listSpec("ListAppend", []node.InputSpec{
		listIn(),
		{Name: "Item", Type: "*"},
	}, "List")
}

// Evaluate — 返回新列表, 必 copy: 防 append 原地改写上游 Evaluate 返回的切片 (底层数组别名).
// 浅拷贝 — 嵌套 map/子 list 与原列表共享引用 (值语义同 Python list.copy(), 非 bug, i18n 写明).
func (ListAppend) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	items := in.List("List")
	out := make([]any, 0, len(items)+1)
	out = append(out, items...)
	return append(out, in.Raw("Item")), nil
}

// ===== ListSlice =====

type ListSlice struct{}

func (ListSlice) Spec() node.Spec {
	return listSpec("ListSlice", []node.InputSpec{
		listIn(),
		{Name: "Start", Type: "Integer", Default: json.Number("0"), Widget: node.WidgetSpec{Kind: "number"}},
		// Count 默认 -1 = 取到末尾 (与 Substring.Length 同约定: 负=到末尾/0=空/正=N).
		{Name: "Count", Type: "Integer", Default: json.Number("-1"), Widget: node.WidgetSpec{Kind: "number"}},
	}, "List")
}

// Evaluate — 返回新列表 (copy 防别名). Start clamp [0,len], Start>=len → 恒空.
func (ListSlice) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	items := in.List("List")
	start := in.Int("Start")
	if start < 0 {
		start = 0
	}
	if start >= len(items) {
		return []any{}, nil
	}
	count := in.Int("Count")
	if count == 0 {
		return []any{}, nil
	}
	end := len(items)
	if count > 0 {
		end = start + count
		if end > len(items) {
			end = len(items)
		}
	}
	out := make([]any, end-start)
	copy(out, items[start:end])
	return out, nil
}
```

- [ ] **Step 4: 运行确认通过**

Run: `go test ./internal/nodes/collection/ -count=1`
Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add internal/nodes/collection/
git commit -m "feat(collection): ListContains/ListAppend/ListSlice 节点 (LooseEqual/必 copy)"
```

---

## Task 5: ForEach（control 包 RegionRunner）+ makeBodyFor 接线

**Files:**
- Create: `internal/nodes/control/foreach.go`、`internal/nodes/control/foreach_test.go`
- Modify: `internal/services/container/runtime/dispatch_v5.go::makeBodyFor`（:414-422）

- [ ] **Step 1: 写失败测试**

`foreach_test.go`（范式照 loop_test.go，`recVars` 已在 loop_test.go 同包可复用）：

```go
package control

import (
	"context"
	"errors"
	"testing"

	"yotta/internal/node"
)

func TestForEach_IteratesAllItems(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&ForEach{})
	rn, _ := node.Get("ForEach")

	vars := newRecVars()
	services := node.StubServices()
	services.Vars = vars

	var items, indices []any
	body := func(_ node.Ctx) error {
		v, _ := vars.Get("item")
		i, _ := vars.Get("idx")
		items = append(items, v)
		indices = append(indices, i)
		return nil
	}

	r := node.RunNodeAsRegion(context.Background(), rn,
		map[string]any{feInList: []any{"a", "b", "c"}},
		map[string]any{feCapItem: "item", feCapIndex: "idx"},
		nil, services, false, body)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if len(items) != 3 || items[0] != "a" || items[2] != "c" {
		t.Errorf("items = %v, want [a b c]", items)
	}
	if len(indices) != 3 || indices[0] != 0 || indices[2] != 2 {
		t.Errorf("indices = %v, want [0 1 2]", indices)
	}
	if r.ExitName != feOutDone {
		t.Errorf("exit = %q, want %q", r.ExitName, feOutDone)
	}
}

func TestForEach_EmptyOrNonList_ZeroIterationsDone(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&ForEach{})
	rn, _ := node.Get("ForEach")

	for _, listVal := range []any{[]any{}, nil, "not a list", 42} {
		iterations := 0
		r := node.RunNodeAsRegion(context.Background(), rn,
			map[string]any{feInList: listVal}, nil, nil,
			node.StubServices(), false, func(_ node.Ctx) error { iterations++; return nil })
		if r.Error != nil {
			t.Fatalf("listVal=%v: %v", listVal, r.Error)
		}
		if iterations != 0 {
			t.Errorf("listVal=%v: iterations = %d, want 0", listVal, iterations)
		}
		if r.ExitName != feOutDone {
			t.Errorf("listVal=%v: exit = %q, want Done", listVal, r.ExitName)
		}
	}
}

func TestForEach_BreakSentinel(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&ForEach{})
	rn, _ := node.Get("ForEach")

	iterations := 0
	body := func(_ node.Ctx) error {
		iterations++
		if iterations == 2 {
			return errBreakRequested
		}
		return nil
	}
	r := node.RunNodeAsRegion(context.Background(), rn,
		map[string]any{feInList: []any{1, 2, 3, 4}}, nil, nil,
		node.StubServices(), false, body)
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if iterations != 2 || r.ExitName != feOutDone {
		t.Errorf("iterations=%d exit=%q, want 2/Done", iterations, r.ExitName)
	}
}

func TestForEach_ContinueSentinel(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&ForEach{})
	rn, _ := node.Get("ForEach")

	iterations := 0
	r := node.RunNodeAsRegion(context.Background(), rn,
		map[string]any{feInList: []any{1, 2, 3}}, nil, nil,
		node.StubServices(), false, func(_ node.Ctx) error { iterations++; return errContinueRequested })
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if iterations != 3 || r.ExitName != feOutDone {
		t.Errorf("iterations=%d exit=%q, want 3/Done", iterations, r.ExitName)
	}
}

func TestForEach_BodyErrorPropagates(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&ForEach{})
	rn, _ := node.Get("ForEach")

	boom := errors.New("boom")
	r := node.RunNodeAsRegion(context.Background(), rn,
		map[string]any{feInList: []any{1, 2}}, nil, nil,
		node.StubServices(), false, func(_ node.Ctx) error { return boom })
	if !errors.Is(r.Error, boom) {
		t.Errorf("error = %v, want boom", r.Error)
	}
	if r.ExitName != "" {
		t.Errorf("exit = %q, want empty", r.ExitName)
	}
}
```

注意：`RunNodeAsRegion(ctx, rn, dataWire, config, execData, services, logEnabled, body)` —— List 走 **dataWire**（第 3 参，运行时连线值），capture 变量名走 **config**（第 4 参，literal）。与 loop_test 的用法（config 里塞 Mode/Count）差异是刻意的：List 在真实运行里来自连线。若签名/参数序与此不符，以 loop_test.go 实际为准适配并报告。

- [ ] **Step 2: 运行确认失败**

Run: `go test ./internal/nodes/control/ -run TestForEach -v`
Expected: 编译失败（`ForEach` 未定义）。

- [ ] **Step 3: 写实现**

`foreach.go`：

```go
// internal/nodes/control/foreach.go
// ForEach — RegionRunner 节点. 遍历 List, 每轮 Capture 元素+下标跑 Body.
// Break/Continue sentinel 同 Loop. 非列表/空列表 → 0 轮直接 Done (不算错).
// Category "List" (palette 与列表节点同组; 机制归 control 包).
package control

import (
	"errors"

	"yotta/internal/node"
)

func init() { node.Register(&ForEach{}) }

type ForEach struct{}

const (
	feInExec   = "In"
	feInList   = "List"
	feCapItem  = "CaptureItem"
	feCapIndex = "CaptureIndex"

	feOutBody = "Body"
	feOutDone = "Done"
)

func (ForEach) Spec() node.Spec {
	return node.Spec{
		Kind:     "ForEach",
		Category: "List",
		Inputs: []node.InputSpec{
			{Name: feInExec, Type: "Exec"},
			{Name: feInList, Type: "List"},
			{Name: feCapItem, Type: "String", Advanced: true, Semantic: "capture",
				CaptureType: "any", Widget: node.WidgetSpec{Kind: "text"}},
			{Name: feCapIndex, Type: "String", Advanced: true, Semantic: "capture",
				CaptureType: "number", Widget: node.WidgetSpec{Kind: "text"}},
		},
		Outputs: []node.OutputSpec{
			{Name: feOutBody, Type: "Exec"},
			{Name: feOutDone, Type: "Exec"},
			{Name: "Fail", Type: "Exec", Semantic: "error",
				Data: []node.DataField{
					{Name: "Error", Type: "String"},
					{Name: "Code", Type: "String"},
				}},
		},
	}
}

// RunRegion — items 在 ForEach 自身 dispatch 入口取一次 (上游非确定节点由 per-dispatch
// 缓存保证同值 → 列表对整轮循环稳定). 快照仅冻结切片头: 元素是引用, body 改其内容后续轮可见.
func (ForEach) RunRegion(ctx node.Ctx, in node.Inputs, body func(node.Ctx) error) (node.Outputs, error) {
	items := in.List(feInList)
	for i, el := range items {
		node.Capture(ctx, in, feCapItem, el)
		node.Capture(ctx, in, feCapIndex, i)
		if err := body(ctx); err != nil {
			if errors.Is(err, errBreakRequested) {
				return ctx.Out(feOutDone).Fire(), nil
			}
			if errors.Is(err, errContinueRequested) {
				continue
			}
			return nil, err
		}
	}
	return ctx.Out(feOutDone).Fire(), nil
}
```

- [ ] **Step 4: makeBodyFor 接线**

`dispatch_v5.go::makeBodyFor`（:415-417）：

```go
	switch node.Kind {
	case "Loop", "ForEach":
		// ForEach body 与 Loop 完全同构 (seed node.ID+".Body") — 共用 builder.
		return r.makeBodyForLoop(node, tok), nil
```

并把 `makeBodyForLoop` 的 doc 注释（:424-425）首行改为：

```go
// makeBodyForLoop body callback 每次调跑一轮 Loop/ForEach body (从 node.Body 出口下游 seed 到 queue 空).
```

- [ ] **Step 5: 运行确认通过**

Run: `go test ./internal/nodes/control/ ./internal/services/container/... -count=1`
Expected: PASS（fish-fixture 预存除外）。Fail 出口路由复用框架 region 错误路由（dispatch_v5.go:218-249, 所有 RegionRunner 共用）——既有 Loop Fail 测试即覆盖该机制, ForEach 不需重复集成测试（unit 层 BodyErrorPropagates 已钉 error 上抛）。

- [ ] **Step 6: Commit**

```bash
git add internal/nodes/control/ internal/services/container/runtime/dispatch_v5.go
git commit -m "feat(control): ForEach RegionRunner (List 遍历, Break/Continue 同 Loop)"
```

---

## Task 6: RandomChoice（random 包）

**Files:**
- Modify: `internal/nodes/random/random.go`
- Test: `internal/nodes/random/random_test.go`

- [ ] **Step 1: 写失败测试**

追加到 `random_test.go`：

```go
func TestRandomChoice_Spec_Flags(t *testing.T) {
	s := RandomChoice{}.Spec()
	if !s.IsPureData || !s.IsNonDeterministic || s.Category != "Random" {
		t.Fatalf("bad spec: %+v", s)
	}
	if s.Outputs[0].Type != "*" {
		t.Fatalf("output type = %q, want *", s.Outputs[0].Type)
	}
}

func TestRandomChoice_PicksFromList(t *testing.T) {
	in := node.NewInputsFromConfig(map[string]any{"literal": map[string]any{"List": []any{"a", "b", "c"}}})
	seen := map[any]bool{}
	for i := 0; i < 500; i++ {
		v, err := (RandomChoice{}).Evaluate(nil, in)
		if err != nil {
			t.Fatal(err)
		}
		if v != "a" && v != "b" && v != "c" {
			t.Fatalf("picked %v, not in list", v)
		}
		seen[v] = true
	}
	if len(seen) < 2 {
		t.Fatalf("500 picks saw only %v — not random", seen)
	}
}

func TestRandomChoice_EmptyOrNonList_Nil(t *testing.T) {
	for _, lv := range []any{[]any{}, nil, "not a list"} {
		in := node.NewInputsFromConfig(map[string]any{"literal": map[string]any{"List": lv}})
		v, err := (RandomChoice{}).Evaluate(nil, in)
		if err != nil || v != nil {
			t.Fatalf("List=%v: got (%v, %v), want (nil, nil)", lv, v, err)
		}
	}
}

func TestRandomChoice_NilElementPickable(t *testing.T) {
	in := node.NewInputsFromConfig(map[string]any{"literal": map[string]any{"List": []any{nil}}})
	v, err := (RandomChoice{}).Evaluate(nil, in)
	if err != nil || v != nil {
		t.Fatalf("got (%v, %v), want (nil, nil) — 元素本身是 nil", v, err)
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `go test ./internal/nodes/random/ -run TestRandomChoice -v`
Expected: 编译失败。

- [ ] **Step 3: 写实现**

`random.go`：init 注册列表加 `&RandomChoice{}`；末尾追加：

```go
// ===== RandomChoice =====

type RandomChoice struct{}

func (RandomChoice) Spec() node.Spec {
	return node.Spec{
		Kind: "RandomChoice", Category: "Random",
		Inputs:             []node.InputSpec{{Name: "List", Type: "List"}},
		Outputs:            []node.OutputSpec{{Name: "Result", Type: "*"}},
		IsPureData:         true,
		IsNonDeterministic: true,
	}
}

// Evaluate — 均匀取一元素. 空/非列表 → nil (与 ListGet 越界一致; nil 歧义 i18n 写明).
// 受 per-dispatch 缓存覆盖: 同一求值内多路径引用同值.
func (RandomChoice) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	items := in.List("List")
	if len(items) == 0 {
		return nil, nil
	}
	return items[rand.IntN(len(items))], nil
}
```

- [ ] **Step 4: 运行确认通过**

Run: `go test ./internal/nodes/random/ -count=1`
Expected: PASS（全部, 现 18 个）。

- [ ] **Step 5: Commit**

```bash
git add internal/nodes/random/
git commit -m "feat(random): RandomChoice 节点 (List 均匀取一, 空→nil)"
```

---

## Task 7: blank-import + i18n + catalog 绿

**Files:**
- Modify：10 处 blank-import 站点（各加一行，紧挨已有 `_ "yotta/internal/nodes/random"`）：main.go、cmd/node-catalog/main.go、cmd/validate-fishing-v2/main.go、cmd/yotta-mcp/main.go、internal/catalog/catalog_test.go、internal/catalog/markdown_test.go、internal/node/spec_capture_test.go、internal/node/spec_consistency_test.go、internal/services/container/runtime/dispatch_v5_test.go、internal/services/container/setup_test.go
- Modify: `frontend/src/i18n/zh.ts`、`en.ts`

- [ ] **Step 1: 10 处 blank import**

```go
	_ "yotta/internal/nodes/collection" // Split/Join/List* 列表节点
```

（control/random 包已被全部站点 import，ForEach/RandomChoice 不需要新 import。）

- [ ] **Step 2: zh.ts 加 9 节点块 + nodeGroup**

`node:` 大块内新建 `// list` 区（`// string` 区后）：

```ts
    ForEach: {
      label: '遍历列表', description: '把列表里的元素一个个拿出来，每个都跑一遍「循环体」。把元素和序号存进变量（变量类型选 any），循环体里用「读变量」取。列表为空或不是列表就直接走「完成」。',
      input: { List: { label: '列表' }, CaptureItem: { label: '元素存入变量' }, CaptureIndex: { label: '序号存入变量' } },
      output: { Body: { label: '循环体' }, Done: { label: '完成' }, Fail: { label: '失败' } },
    },
    Split: {
      label: '拆分文本', description: '把文本按分隔符拆成列表。文本留空得空列表；分隔符留空则一个字一个字拆（中文安全）。',
      input: { Text: { label: '文本' }, Separator: { label: '分隔符' } },
      output: { Result: { label: '列表' } },
    },
    Join: {
      label: '拼接列表', description: '把列表里的元素用分隔符连成一段文本。非文本元素自动转文字。',
      input: { List: { label: '列表' }, Separator: { label: '分隔符' } },
      output: { Result: { label: '结果' } },
    },
    ListLength: {
      label: '列表长度', description: '数列表里有几个元素。不是列表算 0 个。',
      input: { List: { label: '列表' } },
      output: { Result: { label: '个数' } },
    },
    ListGet: {
      label: '取列表元素', description: '取第 Index 个元素（从 0 数）。序号超出范围得 null——元素本身也可能是 null，要区分先用「列表长度」查。',
      input: { List: { label: '列表' }, Index: { label: '序号' } },
      output: { Result: { label: '元素' } },
    },
    ListContains: {
      label: '列表包含', description: '判断列表里有没有等于 Value 的元素。比较规则与「等于」节点相同：类型相同直接比，类型不同转文字比。',
      input: { List: { label: '列表' }, Value: { label: '找什么' } },
      output: { Result: { label: '结果' } },
    },
    ListAppend: {
      label: '追加元素', description: '在列表末尾加一个元素，得到一个新列表（原列表不变；要累积请配合「写变量」存回去，变量类型选 any）。',
      input: { List: { label: '列表' }, Item: { label: '元素' } },
      output: { Result: { label: '新列表' } },
    },
    ListSlice: {
      label: '截取列表', description: '从第 Start 个元素开始取 Count 个，得到新列表。Count 填 -1（默认）取到末尾，0 得空列表。起点超出范围得空列表。',
      input: { List: { label: '列表' }, Start: { label: '起点' }, Count: { label: '个数' } },
      output: { Result: { label: '新列表' } },
    },
```

`// random` 区追加 RandomChoice：

```ts
    RandomChoice: {
      label: '随机取一个', description: '从列表里随机挑一个元素。同一次求值里多处引用拿到同一个。列表为空得 null（元素本身也可能是 null，要区分先用「列表长度」查）。',
      input: { List: { label: '列表' } },
      output: { Result: { label: '元素' } },
    },
```

`nodeGroup:` 块（`random:` 行后）加：

```ts
    list: '列表',
```

另在 zh.ts 找到容器编辑器通用文案块（与 pin/inspector 相关处，grep `expression.error` 附近的块归属），加 List pin 占位文案（key 路径以 Task 8 实现为准，先备文案）：`'由连线提供'`。

- [ ] **Step 3: en.ts 镜像**

```ts
    ForEach: {
      label: 'For Each', description: 'Take items from the list one by one and run the Body for each. The item and index are stored into variables (declare them with type "any"); read them inside the body with Get Variable. An empty or non-list input goes straight to Done.',
      input: { List: { label: 'List' }, CaptureItem: { label: 'Store item in' }, CaptureIndex: { label: 'Store index in' } },
      output: { Body: { label: 'Body' }, Done: { label: 'Done' }, Fail: { label: 'Fail' } },
    },
    Split: {
      label: 'Split', description: 'Split text into a list by a separator. Empty text gives an empty list; an empty separator splits into individual characters (CJK-safe).',
      input: { Text: { label: 'Text' }, Separator: { label: 'Separator' } },
      output: { Result: { label: 'List' } },
    },
    Join: {
      label: 'Join', description: 'Join list items into one piece of text with a separator. Non-text items are converted automatically.',
      input: { List: { label: 'List' }, Separator: { label: 'Separator' } },
      output: { Result: { label: 'Result' } },
    },
    ListLength: {
      label: 'List Length', description: 'Count the items in a list. A non-list counts as 0.',
      input: { List: { label: 'List' } },
      output: { Result: { label: 'Count' } },
    },
    ListGet: {
      label: 'List Get', description: 'Take the item at Index (counting from 0). Out-of-range gives null — an item can itself be null, so check List Length first to tell them apart.',
      input: { List: { label: 'List' }, Index: { label: 'Index' } },
      output: { Result: { label: 'Item' } },
    },
    ListContains: {
      label: 'List Contains', description: 'Whether the list has an item equal to Value. Same rules as the Equals node: same types compare directly, different types compare as text.',
      input: { List: { label: 'List' }, Value: { label: 'Find' } },
      output: { Result: { label: 'Result' } },
    },
    ListAppend: {
      label: 'List Append', description: 'Add one item to the end, producing a NEW list (the original is unchanged; to accumulate, store it back with Set Variable using type "any").',
      input: { List: { label: 'List' }, Item: { label: 'Item' } },
      output: { Result: { label: 'New list' } },
    },
    ListSlice: {
      label: 'List Slice', description: 'Take Count items starting at Start, as a new list. Count -1 (default) takes to the end, 0 gives an empty list. Out-of-range Start gives an empty list.',
      input: { List: { label: 'List' }, Start: { label: 'Start' }, Count: { label: 'Count' } },
      output: { Result: { label: 'New list' } },
    },
    RandomChoice: {
      label: 'Random Choice', description: 'Pick one random item from the list. Multiple references within one evaluation get the same pick. An empty list gives null (an item can itself be null — check List Length to tell them apart).',
      input: { List: { label: 'List' } },
      output: { Result: { label: 'Item' } },
    },
```

`nodeGroup:` 加 `list: 'List',`；占位文案备 `'Provided by wire'`。

- [ ] **Step 4: 生成 + 校验**

Run: `cd frontend && pnpm gen:node-i18n` → 期望 105 节点（96+9）。
Run: `cd frontend && pnpm i18n:check` → parity + compile OK（residue 28 已知）。
Run: `go test ./internal/... -count=1` → 无新失败（fish-fixture 预存除外；`TestNoPinNameSplit` 注意 `List`/`Index`/`Item`/`Value`/`Count`/`Start`/`Text`/`Separator` pin——`Text`/`Start` 已有同型先例（String/Integer），`Count` 撞 Loop 的 `Count`(Integer) **同型不分裂**，其余为新名或 `*` 豁免；若仍报分裂，停下报告，不要自行改 pin 名）。

- [ ] **Step 5: Commit**

```bash
git add main.go cmd/ internal/ frontend/src/i18n/
git commit -m "feat(collection): 注册 collection 包 (10 处 blank import) + 9 节点 zh/en i18n + nodeGroup.list"
```

---

## Task 8: 前端 — 类型词表三连 + List palette 分组 + 只读占位

**Files:**
- Modify: `frontend/src/components/containers/nodeRegistry/index.ts`（PinType + TYPE_COLOR + NodeGroup）
- Modify: `frontend/src/components/containers/nodeRegistry/adapter.ts`（backendTypeToPinType + GROUP_MAP）
- Modify: `frontend/src/components/containers/NodePalette.vue`（GROUP_LABEL + KINDS_BY_GROUP）
- Modify: `frontend/src/composables/editor/useNodeGroupColor.ts`（GROUP_I18N_KEY + ALL_NODE_GROUPS）
- Modify: `frontend/src/components/containers/visualRegistry.ts`（GROUP_VISUAL）
- Modify: `frontend/src/components/containers/inline/PinLiteral.vue` + NodeInspector 用的 `PinInput.vue`（list 只读占位）
- Modify: `frontend/src/i18n/zh.ts`/`en.ts`（占位文案 key 落位）

- [ ] **Step 1: 类型词表三连（A' 审计必改）**

a. `nodeRegistry/index.ts:7` `PinType` 联合加 `| 'list'`；`TYPE_COLOR` 表加 `list: '#818cf8',`。
b. `adapter.ts::backendTypeToPinType` 加 case（按该函数现有 case 风格，匹配大小写规范化后的 `'list'`）→ `return 'list'`。
c. `pinTypeCompat` **不用改**（`from === to` 已覆盖 list↔list；它当前未接 vue-flow，预存缺口不在本批接）。

- [ ] **Step 2: palette 分组 6 站点**（与 random 阶段同款手法）

`NodeGroup` 联合加 `| 'list'`；`GROUP_MAP` 加 `List: 'list',`；`GROUP_LABEL` 加 `list: t('nodeGroup.list'),`；`KINDS_BY_GROUP` 初始化器加 `list: { label: labels.list, items: [] },`；`GROUP_I18N_KEY` 加 `list: 'nodeGroup.list',`；`ALL_NODE_GROUPS` 加 `'list'`；`GROUP_VISUAL` 加 `list: { color: 'indigo', icon: 'i-tabler-list' },`。

- [ ] **Step 3: List pin 只读占位**

`PinInput.vue` 与 `PinLiteral.vue`：在现有 dispatch 链（dropdown → number → bool → 默认 UInput）最前加 list 分支——pin type 为 `'list'` 时渲染只读占位（不渲染可编辑 input，防手输垃圾 literal）：

```vue
<span v-if="type === 'list'" class="text-xs text-dimmed italic">{{ t('containers.listPinWireOnly') }}</span>
```

（具体 prop 名/结构以两文件现状为准——它们结构相同；i18n key 放 zh/en 的 `containers` 块：zh `listPinWireOnly: '由连线提供'`、en `listPinWireOnly: 'Provided by wire'`。若该文件没有现成 `t()` 引入按文件惯例补。）

- [ ] **Step 4: typecheck + i18n**

Run: `cd frontend && pnpm typecheck && pnpm i18n:check`
Expected: typecheck 零错（exhaustive Record 全补齐）；parity/compile OK。若 TS 报其它 `Record<NodeGroup,...>`/`Record<PinType,...>` 穷举站点缺 `list`，最小补齐并在报告列出。

- [ ] **Step 5: Commit**

```bash
git add frontend/src/
git commit -m "feat(collection): 前端 List 类型词表三连 + list palette 分组 + List pin 只读占位"
```

---

## Task 9: docs 同步 + 全量验证

**Files:**
- Modify: `flightdeck/docs/node-system-reference.md`

- [ ] **Step 1: 节点目录更新**

目录表加 List 行（按字母序插在 IO 与 PureFunc 之间）：

```markdown
| **List** (8) | ForEach(Region), Join, ListAppend, ListContains, ListGet, ListLength, ListSlice, Split — ForEach 为 exec RegionRunner, 其余全 PureData |
```

Random 行 (3)→(4) 加 RandomChoice（若该表尚无 Random 行——阶段1 落地时漏加——一并补：`| **Random** (4) | RandomBool, RandomChoice, RandomFloat, RandomInt — 全 PureData+NonDeterministic |`）。

另检查该 doc 的 pin 类型清单段（若有）加 List；变量/Expr 相关段加一句"列表存变量用 any 型；列表不能进 Expr 运算（干净报错）"。

- [ ] **Step 2: 后端全绿**

Run: `go build ./... && go test ./internal/... -count=1`
Expected: PASS（fish-fixture 预存除外）。

- [ ] **Step 3: catalog 自查**

Run: `task nodes`
Expected: List 分类含 ForEach + 7 节点；Random 含 RandomChoice；pin/默认值对（ListSlice.Count 默认 -1）。

- [ ] **Step 4: 构建**

Run: `task build` → 成功。

- [ ] **Step 5: 真机 smoke（人工，留给用户）**

- **看什么**：面板出现「列表」分组（8 节点）+「随机」组多了「随机取一个」；List pin 是靛蓝色；未连线的 List pin 显示"由连线提供"而非输入框。
- **怎么验**：① 拆分文本(`a,b,c`) → 遍历列表（元素存 `item`，变量类型 any）→ 循环体里 读变量(item) → Log → 跑容器 → 日志依次出 a、b、c。② 拆分文本 → 随机取一个 → Log → 跑 → 输出是 a/b/c 之一。③ 写变量把列表存进 any 型变量 → 读变量 → 列表长度 → Log → 输出 3。
- **什么算过**：三条全对。

- [ ] **Step 6: Commit**

```bash
git add flightdeck/docs/node-system-reference.md
git commit -m "docs(nodes): node-system-reference 加 List 分类 (8 节点) + Random 补 RandomChoice"
```

---

## Self-Review（写完计划的自查结论）

- **Spec 覆盖**：A.1 List 类型(widget 锁只读)→Task 1+8；A.2 in.List→Task 1；A.3 toExprValue 显式化→Task 1；A.4 LooseEqual/FormatValue 提升(含 A' 审计的不可比防护修正)→Task 2；A' gate→已完成(结论在 spec)；B ForEach(含空/非列表 0 轮、sentinel、Fail 复用框架路由、Capture any/number)→Task 5；C 7 节点全边界(Split 空串/空分隔、Join FormatValue、ListGet nil 歧义、Contains==Eq+嵌套防护、Append/Slice 必 copy、Slice Start≥len 恒空)→Task 3+4；D RandomChoice(空→nil、缓存覆盖)→Task 6；blank-import 镜像→Task 7；i18n(全部边界写进 description)→Task 7；前端 List 分类+类型词表(审计补的三连)+只读占位→Task 8；验证+smoke→Task 9。全覆盖。
- **对 spec 的两处已批修正**：① makeBodyForForEach 不另写, 复用 makeBodyForLoop(实体零差异, DRY)；② LooseEqual 加不可比类型防护(原行为=panic 被吞, A' 审计结论)。
- **类型一致**：`listSpec`/`listIn`/`sepIn` Task 3 定义 Task 4 复用；`fe*` 常量 Task 5 内一致；`node.LooseEqual`/`FormatValue` Task 2 定义、Task 4(Contains/Join) 与 purefunc 调用一致；ListSlice 与 Substring 同 Count 约定。
- **无占位符**：所有代码完整；唯前端 Step 3 的组件内部结构标注"以两文件现状为准"（dispatch 链分支形态差异），意图（只读占位、不渲染可编辑框）固定。
- **已知非回归**：fish-fixture、residue 28。
