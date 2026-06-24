# Point %/px 单位 + 截图取点 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 给 `node.Point` 加显式单位(%/px),4 个坐标节点(ClickAt/Scroll/MouseMoveTo/Swipe)统一到单个 Point pin,PointWidget 自带单位开关 + 截图取点,新增 MakePoint 构造节点恢复 X/Y 分别连线。

**Architecture:** px→ratio 换算只在 `node.ResolvePoint(ctx,p)` 一处发生(复用既有 `ctx.Window().ClientSize()`,不动 InputService 接口);坐标节点 Run 开头 resolve 一次,downstream(moveCursor/ClickWithMods/Scroll/Drag)+ backend + 检测节点全保持纯比例不动。FE PointWidget 经 StructuredInput(`schema.widget==='point'`)渲染,自持截图 picker(镜像 GeometryWidget)。

**Tech Stack:** Go(节点框架)+ Vue3/TS(Nuxt UI)+ Wails(RPC/event)+ vitest/vue-tsc。

设计 spec:`flightdeck/work/point-px-unit/design.md`(实现时对照)。

## Global Constraints

- **不要兼容 / 无 shim**:项目未发布,删字段→改所有调用点→一次切干净,不留 deprecated;测试该改就改。
- **有源码必读源码**:用任何 API 先核实名/键/返回值。
- **`""`=比例默认是有意为之**:检测节点输出 Point 无单位 = 比例,`omitempty` 让其零改动。
- **不复用 `node.ResolveScalar`**:它带 `|v|≤1⇒比例` 魔法启发,对显式 px 小值会误判;显式单位直接除。
- **预存红基线**(判红排除,非回归):runtime 子包 `TestApplyDirection_*` / `TestWatchdog_*` / `TestFishingV2Main_StateCycleSmoke`(fixture 缺失);i18n residue / `pnpm lint` 既有项。详见 `flightdeck/checklists/build.md`。
- **验证命令**(以 build.md 为准):Go `go build ./...` + `go test ./pkg/input/... ./internal/...`;FE 单测 `cd frontend && ./node_modules/.bin/vitest run <路径>`(**别用 `pnpm -C frontend test`**);类型 `cd frontend && pnpm vue-tsc --noEmit`;改 catalog `cd frontend && pnpm gen:node-i18n` 后 `go test ./internal/catalog/...`;出 exe `task build`。
- **Git**:commit 直推当前分支 `migrate/flightdeck-new-form`,永不 push,不跳 hook。

---

### Task 1: `PointUnit` 类型 + `Point.Unit` 字段 + 坐标 coercion 带单位

**Files:**
- Modify: `internal/node/types.go:6-10`(Point + 新增 PointUnit)
- Modify: `internal/services/container/runtime/data_pull.go:244-260`(asNodePoint 读 unit)
- Test: `internal/services/container/runtime/data_pull_test.go`

**Interfaces:**
- Produces: `node.PointUnit`(`UnitRatio=""` / `UnitPx="px"`);`node.Point{X,Y float64, Unit PointUnit}`。后续所有任务消费。

- [ ] **Step 1: 写失败测试**(asNodePoint 带 unit)

加到 `internal/services/container/runtime/data_pull_test.go`(无则建,`package runtime`):

```go
func TestAsNodePoint_Unit(t *testing.T) {
	// map 带 unit:"px" → Unit=UnitPx
	p, ok := asNodePoint(map[string]any{"x": 960.0, "y": 540.0, "unit": "px"})
	if !ok || p.X != 960 || p.Y != 540 || p.Unit != nodepkg.UnitPx {
		t.Fatalf("got %+v ok=%v, want {960 540 px}", p, ok)
	}
	// map 无 unit → Unit 空 (比例默认)
	p2, _ := asNodePoint(map[string]any{"x": 0.5, "y": 0.5})
	if p2.Unit != nodepkg.UnitRatio {
		t.Fatalf("got unit %q, want empty", p2.Unit)
	}
	// 已是 node.Point 原样透传 unit
	p3, _ := asNodePoint(nodepkg.Point{X: 1, Y: 2, Unit: nodepkg.UnitPx})
	if p3.Unit != nodepkg.UnitPx {
		t.Fatalf("passthrough lost unit: %+v", p3)
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `go test ./internal/services/container/runtime/ -run TestAsNodePoint_Unit`
Expected: 编译失败(`UnitPx` 未定义 / Point 无 Unit 字段)。

- [ ] **Step 3: 加 PointUnit + Unit 字段**(`internal/node/types.go`)

```go
// PointUnit — Point 坐标单位. "" = 比例(0-1), "px" = 客户区像素.
type PointUnit string

const (
	UnitRatio PointUnit = ""   // 百分比/比例 (X/Y 0-1) — 默认
	UnitPx    PointUnit = "px" // 客户区像素 (X/Y 像素值)
)

// Point 域类型 — 坐标 + 单位.
type Point struct {
	X    float64   `json:"x"`
	Y    float64   `json:"y"`
	Unit PointUnit `json:"unit,omitempty"`
}
```

- [ ] **Step 4: asNodePoint 读 unit**(`internal/services/container/runtime/data_pull.go`)

map 分支改为带 unit;`expr.Point` 分支无单位保持空:

```go
case map[string]any:
	if _, ok := t["x"]; ok {
		return nodepkg.Point{
			X:    asFloat(t["x"]),
			Y:    asFloat(t["y"]),
			Unit: nodepkg.PointUnit(asString(t["unit"])),
		}, true
	}
```

> 若 `asString` helper 不存在,用内联:`u, _ := t["unit"].(string)` 然后 `Unit: nodepkg.PointUnit(u)`。先 grep `func asString` 核实(data_pull.go 已有 `asFloat`)。

- [ ] **Step 5: 跑测试确认通过 + 全包回归**

Run: `go test ./internal/services/container/runtime/ -run TestAsNodePoint_Unit && go build ./...`
Expected: PASS;`go build ./...` 绿(加字段不破坏既有 `Point{X,Y}` 构造,零值 Unit="")。

- [ ] **Step 6: Commit**

```bash
git add internal/node/types.go internal/services/container/runtime/data_pull.go internal/services/container/runtime/data_pull_test.go
git commit -m "feat(node): Point 加 PointUnit 单位字段 + coercion 带 unit"
```

---

### Task 2: `node.ResolvePoint(ctx, p)` helper

**Files:**
- Create: `internal/node/resolve.go`
- Test: `internal/node/resolve_test.go`

**Interfaces:**
- Consumes: `Point`/`PointUnit`(Task 1)、`Ctx.Window().ClientSize()`(既有 `WindowService`,interfaces.go:296)。
- Produces: `func ResolvePoint(ctx Ctx, p Point) (xRatio, yRatio float64, err error)`。Task 3-6 消费。

- [ ] **Step 1: 写失败测试**(`internal/node/resolve_test.go`,`package node`)

```go
package node

import (
	"context"
	"errors"
	"testing"
)

type sizeWin struct {
	w, h int
	err  error
}

func (s sizeWin) BringForeground() error { return nil }
func (s sizeWin) HWND() uintptr          { return 0 }
func (s sizeWin) ClientSize() (int, int, error) { return s.w, s.h, s.err }
func (s sizeWin) SetActive(context.Context, string, string, string, string) error { return nil }

func resolveCtx(w WindowService) Ctx {
	svc := StubServices()
	svc.Window = w
	return newCtx(context.Background(), svc, &Spec{Kind: "_test"}, nil)
}

func TestResolvePoint_Ratio_NoWindow(t *testing.T) {
	// UnitRatio: 原样返回, 不碰 Window (即使 ClientSize 会报错也不该调).
	ctx := resolveCtx(sizeWin{err: errors.New("should not be called")})
	x, y, err := ResolvePoint(ctx, Point{X: 0.25, Y: 0.75})
	if err != nil || x != 0.25 || y != 0.75 {
		t.Fatalf("got %v,%v,%v want 0.25,0.75,nil", x, y, err)
	}
}

func TestResolvePoint_Px_DividesByClientSize(t *testing.T) {
	ctx := resolveCtx(sizeWin{w: 1920, h: 1080})
	x, y, err := ResolvePoint(ctx, Point{X: 960, Y: 540, Unit: UnitPx})
	if err != nil || x != 0.5 || y != 0.5 {
		t.Fatalf("got %v,%v,%v want 0.5,0.5,nil", x, y, err)
	}
}

func TestResolvePoint_Px_SmallValueNotHeuristic(t *testing.T) {
	// 显式 px=1 必须除 (不走 |v|<=1 启发) → 1/1920.
	ctx := resolveCtx(sizeWin{w: 1920, h: 1080})
	x, _, _ := ResolvePoint(ctx, Point{X: 1, Y: 1, Unit: UnitPx})
	if x != 1.0/1920.0 {
		t.Fatalf("got %v want %v", x, 1.0/1920.0)
	}
}

func TestResolvePoint_Px_ClientSizeErr(t *testing.T) {
	ctx := resolveCtx(sizeWin{err: errors.New("no window")})
	if _, _, err := ResolvePoint(ctx, Point{X: 1, Y: 1, Unit: UnitPx}); err == nil {
		t.Fatal("want error when ClientSize fails")
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `go test ./internal/node/ -run TestResolvePoint`
Expected: 编译失败(`ResolvePoint` 未定义)。

- [ ] **Step 3: 实现 helper**(`internal/node/resolve.go`)

```go
package node

import "fmt"

// ResolvePoint 把 Point 按 Unit 归一成 0-1 客户区比例.
// UnitRatio → 原样(不碰 Window);UnitPx → 取 ClientSize 后 px÷宽高.
// 不复用 ResolveScalar — 那带 |v|<=1⇒比例 启发, 对显式 px 小值会误判.
func ResolvePoint(ctx Ctx, p Point) (xRatio, yRatio float64, err error) {
	if p.Unit != UnitPx {
		return p.X, p.Y, nil
	}
	w, h, err := ctx.Window().ClientSize()
	if err != nil {
		return 0, 0, err
	}
	if w <= 0 || h <= 0 {
		return 0, 0, fmt.Errorf("client size %dx%d invalid for px point", w, h)
	}
	return p.X / float64(w), p.Y / float64(h), nil
}
```

- [ ] **Step 4: 跑测试确认通过**

Run: `go test ./internal/node/ -run TestResolvePoint`
Expected: 4 PASS。

- [ ] **Step 5: Commit**

```bash
git add internal/node/resolve.go internal/node/resolve_test.go
git commit -m "feat(node): ResolvePoint helper — px→ratio 单点换算 (复用 ClientSize)"
```

---

### Task 3: ClickAt → 单个 Point pin + resolve + i18n

**Files:**
- Modify: `internal/nodes/input/click_at.go`
- Test: `internal/nodes/input/click_at_test.go`
- Modify: `frontend/src/i18n/zh.ts`(ClickAt.input)、`frontend/src/i18n/en.ts`

**Interfaces:**
- Consumes: `node.ResolvePoint`(Task 2)、`node.PointSchema()`、`in.Point()`。
- Produces: ClickAt 用 `Point` pin(常量 `caInPoint = "Point"`),替代 `caInXRatio`/`caInYRatio`。

- [ ] **Step 1: 改测试为 Point pin + 加 px 测试**(`click_at_test.go`)

把 config 里 `caInXRatio: x, caInYRatio: y` 全改成 `caInPoint: node.Point{X: x, Y: y}`。逐个改:
- `TestClickAt_HappyPath`:`map[string]any{caInPoint: node.Point{X:0.3,Y:0.7}, caInButton:"right", caInDurationMs:80}`,断言不变(`MoveTo:0.300:0.700` / `Click:0.300:0.700:right:80`)。
- `TestClickAt_DefaultsApplied`:不传 config,断言 `MoveTo:0.500:0.500`(Default 中心)。
- `TestClickAt_CtxCancel_AbortsBeforeClick`:`caInPoint: node.Point{X:1,Y:1}`。
- `TestClickAt_MoveMs_SlidesBeforeDown`:`caInPoint: node.Point{X:1,Y:1}`。
- `TestClickAt_Keys_ClickCount` / `_DefaultRegression` / `_DurationMs_Preserved`:同样换 caInPoint。
- validation 测试(button/keys/count)不涉坐标,只删 caInXRatio/YRatio(若有)。

新增 px 测试(需 Window stub + Input stub 同一 bundle):

```go
// stub window with fixed client size for px tests.
type sizeWindow struct{ w, h int }

func (s sizeWindow) BringForeground() error           { return nil }
func (s sizeWindow) HWND() uintptr                    { return 0 }
func (s sizeWindow) ClientSize() (int, int, error)    { return s.w, s.h, nil }
func (s sizeWindow) SetActive(ctx context.Context, a, b, c, d string) error { return nil }

func TestClickAt_PxPoint_ResolvesToRatio(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&ClickAt{})
	rn, _ := node.Get("ClickAt")

	rec := &recordingInput{}
	b := node.StubServices()
	b.Input = rec
	b.Window = sizeWindow{w: 1920, h: 1080}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{caInPoint: node.Point{X: 960, Y: 540, Unit: node.UnitPx}, caInDurationMs: 50},
		nil, b, false)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	// 960/1920=0.5, 540/1080=0.5
	if len(rec.calls) != 2 || rec.calls[1] != "Click:0.500:0.500:left:50" {
		t.Errorf("calls=%v want Click:0.500:0.500", rec.calls)
	}
}
```

> `sizeWindow` 与 bring_foreground_test 的 `recordingWindow`(ClientSize 返 0,0)不同 — 放 click_at_test.go 本地;若与他处重名,改名 `clickSizeWin`。

- [ ] **Step 2: 跑测试确认失败**

Run: `go test ./internal/nodes/input/ -run TestClickAt`
Expected: 编译失败(`caInPoint` 未定义 / Spec 仍是 XRatio)。

- [ ] **Step 3: 改 ClickAt spec + Run**(`click_at.go`)

常量:删 `caInXRatio`/`caInYRatio`,加 `caInPoint = "Point"`。Spec.Inputs 把两个 slider Number 换成:

```go
{Name: caInPoint, Type: "Point", Default: node.Point{X: 0.5, Y: 0.5},
	Schema: node.PointSchema()},
```

Run 开头:

```go
func (ClickAt) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	pt := in.Point(caInPoint)
	x, y, err := node.ResolvePoint(ctx, pt)
	if err != nil {
		return nil, node.Failf(node.CodeSendFailed, err, "ClickAt resolve point: %v", err)
	}
	btn := in.String(caInButton)
	// ... (btn/dur/jitter 不变) ...
	if err := moveCursor(ctx, x, y, in.Int(caInMoveMs)); err != nil {
		return nil, err
	}
	modKeys := in.String(caInKeys)
	clickCount := in.Int(caInClickCount)
	if clickCount < 1 {
		clickCount = 1
	}
	if err := node.ClickWithMods(ctx, node.Point{X: x, Y: y}, btn, modKeys, clickCount, dur); err != nil {
		return nil, node.Failf(node.CodeSendFailed, err, "ClickAt (%.3f,%.3f) %s: %v", x, y, btn, err)
	}
	return ctx.Out(caOutDone).Fire(), nil
}
```

Validate 不变(button/keys/count)。

- [ ] **Step 4: 跑测试确认通过**

Run: `go test ./internal/nodes/input/ -run TestClickAt`
Expected: 全 PASS(含 px)。

- [ ] **Step 5: 更新 i18n(ClickAt.input)+ 生成 + catalog 测试**

`frontend/src/i18n/zh.ts` ClickAt 块:删 `XRatio`/`YRatio`,加 `Point: { label: '坐标' }`;`description` 改为不再说"X、Y 都填 0-1"(改述:在窗口里点一下,位置用坐标控件填,可切百分比/像素,可截图取点)。`en.ts` 对称改。

Run: `cd frontend && pnpm gen:node-i18n && cd .. && go test ./internal/catalog/...`
Expected: catalog 测试 PASS(ClickAt 所有 pin 有 i18n 键,无 orphan)。

- [ ] **Step 6: Commit**

```bash
git add internal/nodes/input/click_at.go internal/nodes/input/click_at_test.go frontend/src/i18n/zh.ts frontend/src/i18n/en.ts frontend/src/catalog* internal/catalog/*
git commit -m "refactor(input): ClickAt XRatio/YRatio → 单个 Point pin + px resolve"
```
> `git add` 把 gen:node-i18n 产物一并提交(具体生成文件路径以 `git status` 为准)。

---

### Task 4: Scroll → 单个 Point pin + resolve + i18n

**Files:**
- Modify: `internal/nodes/input/scroll.go`
- Test: `internal/nodes/input/scroll_test.go`(无则建)
- Modify: `frontend/src/i18n/zh.ts`(Scroll.input)、`en.ts`

**Interfaces:**
- Produces: Scroll 用 `Point` pin(`scInPoint = "Point"`),替代 `scInXRatio`/`scInYRatio`。

- [ ] **Step 1: 写/改测试**(`scroll_test.go`)

若已有 Scroll 测试:把 `scInXRatio/scInYRatio` 换 `scInPoint: node.Point{...}`。加 happy + px:

```go
func TestScroll_HappyPath(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Scroll{})
	rn, _ := node.Get("Scroll")
	rec := &recordingInput{}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{scInPoint: node.Point{X: 0.5, Y: 0.5}, scInDelta: 3, scInAxis: "horizontal"},
		nil, withInput(rec), false)
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if len(rec.calls) != 1 || rec.calls[0] != "Scroll:0.500:0.500:3:true" {
		t.Errorf("calls=%v want Scroll:0.500:0.500:3:true", rec.calls)
	}
}

func TestScroll_PxPoint_ResolvesToRatio(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Scroll{})
	rn, _ := node.Get("Scroll")
	rec := &recordingInput{}
	b := node.StubServices()
	b.Input = rec
	b.Window = sizeWindow{w: 1000, h: 500}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{scInPoint: node.Point{X: 250, Y: 250, Unit: node.UnitPx}, scInDelta: 1},
		nil, b, false)
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	// 250/1000=0.25, 250/500=0.5
	if len(rec.calls) != 1 || rec.calls[0] != "Scroll:0.250:0.500:1:false" {
		t.Errorf("calls=%v want Scroll:0.250:0.500:1:false", rec.calls)
	}
}
```

> `sizeWindow` 在 Task 3 的 click_at_test.go 里已定义(同 package input,可直接用)。

- [ ] **Step 2: 跑测试确认失败**

Run: `go test ./internal/nodes/input/ -run TestScroll`
Expected: 编译失败(`scInPoint` 未定义)。

- [ ] **Step 3: 改 Scroll spec + Run**(`scroll.go`)

常量删 `scInXRatio`/`scInYRatio`,加 `scInPoint = "Point"`。Spec 两 slider → 一 Point pin:

```go
{Name: scInPoint, Type: "Point", Default: node.Point{X: 0.5, Y: 0.5},
	Schema: node.PointSchema()},
```

Run:

```go
func (Scroll) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	x, y, err := node.ResolvePoint(ctx, in.Point(scInPoint))
	if err != nil {
		return nil, node.Failf(node.CodeSendFailed, err, "Scroll resolve point: %v", err)
	}
	delta := in.Int(scInDelta)
	delta = node.JitterInt(delta, in.Int(scInJitterPct))
	horizontal := in.String(scInAxis) == "horizontal"
	if err := ctx.Input().Scroll(x, y, delta, horizontal); err != nil {
		return nil, node.Failf(node.CodeSendFailed, err, "Scroll (%.3f,%.3f) Δ=%d horizontal=%v: %v", x, y, delta, horizontal, err)
	}
	return ctx.Out(scOutDone).Fire(), nil
}
```

- [ ] **Step 4: 跑测试确认通过**

Run: `go test ./internal/nodes/input/ -run TestScroll`
Expected: PASS。

- [ ] **Step 5: i18n + gen + catalog**

`zh.ts`/`en.ts` Scroll:删 XRatio/YRatio,加 `Point: { label: '坐标' }`,description 同步。
Run: `cd frontend && pnpm gen:node-i18n && cd .. && go test ./internal/catalog/...` → PASS。

- [ ] **Step 6: Commit**

```bash
git add internal/nodes/input/scroll.go internal/nodes/input/scroll_test.go frontend/src/i18n/zh.ts frontend/src/i18n/en.ts frontend/src/catalog* internal/catalog/*
git commit -m "refactor(input): Scroll XRatio/YRatio → 单个 Point pin + px resolve"
```

---

### Task 5: MouseMoveTo → 单个 Point pin + resolve + i18n

**Files:**
- Modify: `internal/nodes/input/mouse_move_to.go`
- Test: `internal/nodes/input/mouse_move_to_test.go`(无则建)
- Modify: `frontend/src/i18n/zh.ts`(MouseMoveTo.input)、`en.ts`

**Interfaces:**
- Produces: MouseMoveTo 用 `Point` pin(`mmtInPoint = "Point"`),替代 `mmtInXRatio`/`mmtInYRatio`。

- [ ] **Step 1: 写/改测试**

```go
func TestMouseMoveTo_HappyPath(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&MouseMoveTo{})
	rn, _ := node.Get("MouseMoveTo")
	rec := &recordingInput{}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{mmtInPoint: node.Point{X: 0.4, Y: 0.6}},
		nil, withInput(rec), false)
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	// MoveMs=0 → 单帧瞬移 MoveTo
	if len(rec.calls) != 1 || rec.calls[0] != "MoveTo:0.400:0.600" {
		t.Errorf("calls=%v want MoveTo:0.400:0.600", rec.calls)
	}
}

func TestMouseMoveTo_PxPoint_ResolvesToRatio(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&MouseMoveTo{})
	rn, _ := node.Get("MouseMoveTo")
	rec := &recordingInput{}
	b := node.StubServices()
	b.Input = rec
	b.Window = sizeWindow{w: 800, h: 600}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{mmtInPoint: node.Point{X: 400, Y: 300, Unit: node.UnitPx}},
		nil, b, false)
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if len(rec.calls) != 1 || rec.calls[0] != "MoveTo:0.500:0.500" {
		t.Errorf("calls=%v want MoveTo:0.500:0.500", rec.calls)
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `go test ./internal/nodes/input/ -run TestMouseMoveTo`
Expected: 编译失败(`mmtInPoint` 未定义)。

- [ ] **Step 3: 改 spec + Run**(`mouse_move_to.go`)

常量删 `mmtInXRatio`/`mmtInYRatio`,加 `mmtInPoint = "Point"`。Spec 两 slider → 一 Point pin(同 Task3/4 写法,Default 中心 + Schema)。Run:

```go
func (MouseMoveTo) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	x, y, err := node.ResolvePoint(ctx, in.Point(mmtInPoint))
	if err != nil {
		return nil, node.Failf(node.CodeSendFailed, err, "MouseMoveTo resolve point: %v", err)
	}
	moveMs := node.JitterInt(in.Int(mmtInMoveMs), in.Int(mmtInJitterPct))
	if err := moveCursor(ctx, x, y, moveMs); err != nil {
		return nil, err
	}
	return ctx.Out(mmtOutDone).Fire(), nil
}
```

- [ ] **Step 4: 跑测试确认通过**

Run: `go test ./internal/nodes/input/ -run TestMouseMoveTo`
Expected: PASS。

- [ ] **Step 5: i18n + gen + catalog**

`zh.ts`/`en.ts` MouseMoveTo:删 XRatio/YRatio,加 `Point: { label: '坐标' }`,description 同步。
Run: `cd frontend && pnpm gen:node-i18n && cd .. && go test ./internal/catalog/...` → PASS。

- [ ] **Step 6: Commit**

```bash
git add internal/nodes/input/mouse_move_to.go internal/nodes/input/mouse_move_to_test.go frontend/src/i18n/zh.ts frontend/src/i18n/en.ts frontend/src/catalog* internal/catalog/*
git commit -m "refactor(input): MouseMoveTo XRatio/YRatio → 单个 Point pin + px resolve"
```

---

### Task 5b: MouseHoldStart → 单个 Point pin + resolve + i18n(执行期补漏)

**Files:**
- Modify: `internal/nodes/input/mouse_hold.go`(MouseHoldStart Spec + Run;MouseHoldStop 不动)
- Test: `internal/nodes/input/mouse_hold_test.go`(无则建)
- Modify: `frontend/src/i18n/zh.ts`(MouseHoldStart.input)、`en.ts`

**Interfaces:** 与 Task 3-5 同款。MouseHoldStart 用 `Point` pin(`mhStartInPoint = "Point"`),替代 `mhStartInXRatio`/`mhStartInYRatio`。`MouseDown` 接口不变。

- [ ] **Step 1: 写/改测试**(`mouse_hold_test.go`)。MouseHoldStop 测试(若有)不动。

```go
func TestMouseHoldStart_HappyPath(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&MouseHoldStart{})
	rn, _ := node.Get("MouseHoldStart")
	rec := &recordingInput{}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{mhStartInPoint: node.Point{X: 0.4, Y: 0.6}, mhStartInButton: "left"},
		nil, withInput(rec), false)
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	// recordingInput.MouseDown 记录格式 "MouseDown:%.3f:%.3f:%s"
	if len(rec.calls) != 1 || rec.calls[0] != "MouseDown:0.400:0.600:left" {
		t.Errorf("calls=%v want MouseDown:0.400:0.600:left", rec.calls)
	}
}

func TestMouseHoldStart_PxPoint_ResolvesToRatio(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&MouseHoldStart{})
	rn, _ := node.Get("MouseHoldStart")
	rec := &recordingInput{}
	b := node.StubServices()
	b.Input = rec
	b.Window = sizeWindow{w: 1000, h: 1000}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{mhStartInPoint: node.Point{X: 300, Y: 700, Unit: node.UnitPx}, mhStartInButton: "right"},
		nil, b, false)
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if len(rec.calls) != 1 || rec.calls[0] != "MouseDown:0.300:0.700:right" {
		t.Errorf("calls=%v want MouseDown:0.300:0.700:right", rec.calls)
	}
}

func TestMouseHoldStart_InvalidButton_ValidationError(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&MouseHoldStart{})
	rn, _ := node.Get("MouseHoldStart")
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{mhStartInButton: "side1"},
		nil, withInput(&recordingInput{}), false)
	if len(r.Validation) != 1 || r.Validation[0].Code != "INVALID_MOUSE_BUTTON" {
		t.Errorf("validation=%v want INVALID_MOUSE_BUTTON", r.Validation)
	}
}
```

> `sizeWindow` 已在 click_at_test.go 定义(同 package input,直接用)。`recordingInput.MouseDown` 记录格式见 key_press_test.go(`"MouseDown:%.3f:%.3f:%s"`)。

- [ ] **Step 2: 跑测试确认失败** — `go test ./internal/nodes/input/ -run TestMouseHoldStart`,期望编译失败(`mhStartInPoint` 未定义)。

- [ ] **Step 3: 改 MouseHoldStart spec + Run**(`mouse_hold.go`)

常量删 `mhStartInXRatio`/`mhStartInYRatio`,加 `mhStartInPoint = "Point"`。Spec 两 slider → 一 Point pin:

```go
{Name: mhStartInPoint, Type: "Point", Default: node.Point{X: 0.5, Y: 0.5},
	Schema: node.PointSchema()},
```

Run 把 `x := in.Float64(mhStartInXRatio)` / `y := ...YRatio` 两行换成:

```go
x, y, err := node.ResolvePoint(ctx, in.Point(mhStartInPoint))
if err != nil {
	return nil, node.Failf(node.CodeSendFailed, err, "MouseHoldStart resolve point: %v", err)
}
```

button 校验 + `ctx.Input().MouseDown(x, y, btn)` 不变。Validate 不变。MouseHoldStop 整段不动。

- [ ] **Step 4: 跑测试确认通过** — `go test ./internal/nodes/input/ -run TestMouseHold`(Start + Stop 都绿)。

- [ ] **Step 5: i18n + gen + catalog** — `zh.ts`/`en.ts` `MouseHoldStart` 块:删 XRatio/YRatio,加 `Point: { label: '坐标' }`,description 同步(MouseHoldStop 不动)。`cd frontend && pnpm gen:node-i18n && cd .. && go test ./internal/catalog/...` → PASS。

- [ ] **Step 6: Commit**

```bash
git add internal/nodes/input/mouse_hold.go internal/nodes/input/mouse_hold_test.go frontend/src/i18n/zh.ts frontend/src/i18n/en.ts frontend/src/catalog* internal/catalog/*
git commit -m "refactor(input): MouseHoldStart XRatio/YRatio → 单个 Point pin + px resolve"
```

---

### Task 6: Swipe → resolve Begin/End(+ px 测试)

**Files:**
- Modify: `internal/nodes/input/swipe.go:57-66`(Run)
- Test: `internal/nodes/input/swipe_test.go`

**Interfaces:**
- Consumes: `node.ResolvePoint`(Task 2)。Swipe 结构不变(Begin/End 已是 Point pin),仅 Run 加 resolve。

> Swipe 的 i18n / pin 名不变 → 无 catalog 改动。Begin/End 的单位 UI 由 Task 8/9 的 PointWidget 自动获得。

- [ ] **Step 1: 加 px 测试**(`swipe_test.go`)

现有 ratio 测试(HappyPath 等)不变(Unit 空 → ResolvePoint early-return,行为零变)。加:

```go
func TestSwipe_PxPoints_ResolveToRatio(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Swipe{})
	rn, _ := node.Get("Swipe")
	rec := &recordingInput{}
	b := node.StubServices()
	b.Input = rec
	b.Window = sizeWindow{w: 1000, h: 1000}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{
			swInBegin:      node.Point{X: 100, Y: 200, Unit: node.UnitPx},
			swInEnd:        node.Point{X: 800, Y: 900, Unit: node.UnitPx},
			swInButton:     "left",
			swInDurationMs: 300,
		},
		nil, b, false)
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	// 100/1000=0.1 ... 900/1000=0.9
	if len(rec.calls) != 1 || rec.calls[0] != "Drag:0.100:0.200:0.800:0.900:left:300" {
		t.Errorf("calls=%v want Drag:0.100:0.200:0.800:0.900:left:300", rec.calls)
	}
}
```

> `sizeWindow` 已在 click_at_test.go 定义(同 package)。

- [ ] **Step 2: 跑测试确认失败**

Run: `go test ./internal/nodes/input/ -run TestSwipe_PxPoints`
Expected: FAIL(当前 Run 直接用 begin.X 像素值,得 `Drag:100.000:200.000:...` 不等)。

- [ ] **Step 3: 改 Swipe Run**(`swipe.go`)

```go
func (Swipe) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	bx, by, err := node.ResolvePoint(ctx, in.Point(swInBegin))
	if err != nil {
		return nil, node.Failf(node.CodeSendFailed, err, "Swipe resolve begin: %v", err)
	}
	ex, ey, err := node.ResolvePoint(ctx, in.Point(swInEnd))
	if err != nil {
		return nil, node.Failf(node.CodeSendFailed, err, "Swipe resolve end: %v", err)
	}
	btn := in.String(swInButton)
	if btn == "" {
		btn = "left"
	}
	dur := in.Int(swInDurationMs)
	if dur <= 0 {
		dur = 200
	}
	if err := ctx.Input().Drag(bx, by, ex, ey, btn, dur); err != nil {
		return nil, node.Failf(node.CodeSendFailed, err, "Swipe drag: %v", err)
	}
	return ctx.Out(swOutDone).Fire(), nil
}
```

> 对照现有 swipe.go 保留 button/duration 的既有默认逻辑(若与上不同以现有为准,只插入两行 ResolvePoint + 把 begin/end 换成 bx,by,ex,ey)。

- [ ] **Step 4: 跑测试确认通过(含既有 ratio 回归)**

Run: `go test ./internal/nodes/input/ -run TestSwipe`
Expected: 全 PASS(ratio 测试零回归 + 新 px PASS)。

- [ ] **Step 5: Commit**

```bash
git add internal/nodes/input/swipe.go internal/nodes/input/swipe_test.go
git commit -m "feat(input): Swipe Begin/End 走 ResolvePoint 支持 px 单位"
```

---

### Task 7: MakePoint 构造节点 + i18n

**Files:**
- Modify: `internal/nodes/purefunc/purefunc.go`(加 MakePoint type + init 注册)
- Test: `internal/nodes/purefunc/purefunc_test.go`(或同包新 `make_point_test.go`)
- Modify: `frontend/src/i18n/zh.ts`、`en.ts`(MakePoint 节点条目)

**Interfaces:**
- Consumes: `node.Point`/`node.UnitPx`(Task 1)。
- Produces: `MakePoint` 节点 — 输入 `X`(Number)/`Y`(Number)/`Unit`(dropdown percent|px),输出 `Result`(Point)。

- [ ] **Step 1: 写失败测试**(`purefunc_test.go` 加,`package purefunc`)

```go
func TestMakePoint_Percent(t *testing.T) {
	v, err := MakePoint{}.Evaluate(nil, node.NewInputsFromConfig(
		map[string]any{"X": 0.5, "Y": 0.25, "Unit": "percent"}))
	if err != nil {
		t.Fatal(err)
	}
	p, ok := v.(node.Point)
	if !ok || p.X != 0.5 || p.Y != 0.25 || p.Unit != node.UnitRatio {
		t.Fatalf("got %+v ok=%v want {0.5 0.25 ratio}", v, ok)
	}
}

func TestMakePoint_Px(t *testing.T) {
	v, _ := MakePoint{}.Evaluate(nil, node.NewInputsFromConfig(
		map[string]any{"X": 960.0, "Y": 540.0, "Unit": "px"}))
	p := v.(node.Point)
	if p.X != 960 || p.Y != 540 || p.Unit != node.UnitPx {
		t.Fatalf("got %+v want {960 540 px}", p)
	}
}

func TestMakePoint_Spec(t *testing.T) {
	s := MakePoint{}.Spec()
	if s.Kind != "MakePoint" || !s.IsPureData {
		t.Fatalf("spec wrong: %+v", s)
	}
	if len(s.Outputs) != 1 || s.Outputs[0].Type != "Point" {
		t.Fatalf("want single Point output, got %+v", s.Outputs)
	}
}
```

> `Evaluate(nil, ...)` 传 nil Ctx 安全 — MakePoint 是纯函数不碰 ctx(对照 purefunc 既有 `Evaluate(_ node.Ctx, ...)`)。

- [ ] **Step 2: 跑测试确认失败**

Run: `go test ./internal/nodes/purefunc/ -run TestMakePoint`
Expected: 编译失败(`MakePoint` 未定义)。

- [ ] **Step 3: 实现 MakePoint**(`purefunc.go`)

加 type + Spec + Evaluate(用 `json` 已 import;放"构造"小节):

```go
// ===== 构造 (1) =====

type MakePoint struct{}

func (MakePoint) Spec() node.Spec {
	return node.Spec{
		Kind:     "MakePoint",
		Category: "PureFunc",
		Inputs: []node.InputSpec{
			{Name: "X", Type: "Number", Default: json.Number("0"), Widget: node.WidgetSpec{Kind: "number"}},
			{Name: "Y", Type: "Number", Default: json.Number("0"), Widget: node.WidgetSpec{Kind: "number"}},
			{Name: "Unit", Type: "String", Default: "percent",
				Widget: node.WidgetSpec{Kind: "dropdown",
					Props: node.MarshalProps(node.DropdownProps{
						Options: []node.EnumOption{
							{Value: "percent"},
							{Value: "px"},
						}})}},
		},
		Outputs:    []node.OutputSpec{{Name: "Result", Type: "Point"}},
		IsPureData: true,
	}
}

func (MakePoint) Evaluate(_ node.Ctx, in node.Inputs) (any, error) {
	u := node.UnitRatio
	if in.String("Unit") == "px" {
		u = node.UnitPx
	}
	return node.Point{X: in.Float64("X"), Y: in.Float64("Y"), Unit: u}, nil
}
```

在 `init()` 的 register 列表末尾加 `&MakePoint{}`(并在注释计数处补一行 `// 构造 (1)`)。

- [ ] **Step 4: 跑测试确认通过**

Run: `go test ./internal/nodes/purefunc/ -run TestMakePoint`
Expected: 3 PASS。

- [ ] **Step 5: i18n + gen + catalog**

`zh.ts`/`en.ts` 加 MakePoint 条目(对照既有 PureFunc 节点如 Add 的结构):

```ts
MakePoint: {
  label: '组装坐标',
  description: '把两个数字(X、Y)拼成一个「坐标」值,接到点击/滑动等需要坐标的节点上。可选单位:百分比(0-1,跟窗口大小无关)或像素。需要用算出来的数字当坐标时用它。',
  example: '把检测到的血量比例当 X、固定 0.9 当 Y,组装成坐标喂给点击节点。',
  input: {
    X: { label: 'X' },
    Y: { label: 'Y' },
    Unit: { label: '单位', option: { percent: '百分比', px: '像素' } },
  },
  output: { Result: { label: '坐标' } },
},
```
`en.ts` 对称。
Run: `cd frontend && pnpm gen:node-i18n && cd .. && go test ./internal/catalog/...` → PASS。

- [ ] **Step 6: Commit**

```bash
git add internal/nodes/purefunc/purefunc.go internal/nodes/purefunc/*_test.go frontend/src/i18n/zh.ts frontend/src/i18n/en.ts frontend/src/catalog* internal/catalog/*
git commit -m "feat(purefunc): MakePoint 节点 — 2 Number + Unit → Point"
```

---

### Task 8: PointValue.unit + PointWidget 单位开关(%/px 显示+存储)

**Files:**
- Modify: `frontend/src/components/containers/nodeRegistry/index.ts:44-47`(PointValue)
- Modify: `frontend/src/components/containers/inline/PointWidget.vue`
- Test: `frontend/src/components/containers/inline/PointWidget.test.ts`
- Modify: `frontend/src/i18n/zh.ts`、`en.ts`(point widget 单位文案)

**Interfaces:**
- Produces: `PointValue { x; y; unit?: 'px' }`;PointWidget 渲染单位开关,px 模式存原值/百分比模式存比例。Task 9 在其上加截图按钮。

- [ ] **Step 1: 写失败测试**(`PointWidget.test.ts`,对照现有测试 mount 方式)

```ts
it('px 模式: 显示原始像素值, 不×100', async () => {
  const wrapper = mountPointWidget({ x: 960, y: 540, unit: 'px' })
  // px 模式两个数字框直接显示 960 / 540
  const inputs = wrapper.findAllComponents({ name: 'UInputNumber' })
  expect(inputs[0].props('modelValue')).toBe(960)
  expect(inputs[1].props('modelValue')).toBe(540)
})

it('% 模式: 显示 ×100', () => {
  const wrapper = mountPointWidget({ x: 0.5, y: 0.25 })
  const inputs = wrapper.findAllComponents({ name: 'UInputNumber' })
  expect(inputs[0].props('modelValue')).toBe(50)
  expect(inputs[1].props('modelValue')).toBe(25)
})

it('切到 px 保留数字不换算 (50 → 50)', async () => {
  const wrapper = mountPointWidget({ x: 0.5, y: 0.5 })
  await clickUnitToggle(wrapper, 'px')
  const emitted = lastEmit(wrapper) // { x, y, unit }
  expect(emitted.unit).toBe('px')
  expect(emitted.x).toBe(50) // 框里数字 50 原样进 x (不换算)
})
```

> `mountPointWidget` / `clickUnitToggle` / `lastEmit` 按 PointWidget.test.ts 现有 helper 写;若无,参照现有 `mount(PointWidget, { props: { modelValue, fieldPath: 'P' } })`。单位开关用 `data-testid="point-unit-toggle"` 定位。

- [ ] **Step 2: 跑测试确认失败**

Run: `cd frontend && ./node_modules/.bin/vitest run src/components/containers/inline/PointWidget.test.ts`
Expected: FAIL(无单位开关 / px 显示逻辑)。

- [ ] **Step 3: PointValue 加 unit**(`index.ts`)

```ts
/** 坐标点存储值 (x/y; unit 空=比例 0-1, 'px'=像素). */
export interface PointValue {
  x: number
  y: number
  unit?: 'px'
}
```

- [ ] **Step 4: PointWidget 加单位开关 + 显示逻辑**(`PointWidget.vue`)

template 顶部加单位开关(USelect 或 UButton group,`data-testid="point-unit-toggle"`,选项 百分比/像素);两个 UInputNumber 的 `:model-value`/`@update` 改为按 unit 分流:

```vue
<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { PointValue } from '@/components/containers/nodeRegistry/index'

const { t } = useI18n()
const props = defineProps<{ modelValue: PointValue | null; fieldPath: string }>()
const emit = defineEmits<{ (e: 'update:modelValue', v: PointValue): void }>()

function round4(n: number): number { return Math.round(n * 1e4) / 1e4 }

const safeValue = computed<PointValue>(() => {
  const v = props.modelValue
  if (!v || typeof v.x !== 'number' || typeof v.y !== 'number') return { x: 0, y: 0 }
  return { x: v.x, y: v.y, unit: v.unit }
})

const isPx = computed(() => safeValue.value.unit === 'px')

// 显示: px 原值; % ×100
const displayX = computed(() => (isPx.value ? safeValue.value.x : round4(safeValue.value.x * 100)))
const displayY = computed(() => (isPx.value ? safeValue.value.y : round4(safeValue.value.y * 100)))

function emitVal(next: PointValue) { emit('update:modelValue', next) }

function onChange(field: 'x' | 'y', displayVal: number) {
  const next: PointValue = { ...safeValue.value }
  next[field] = isPx.value ? displayVal : round4(displayVal / 100)
  emitVal(next)
}

// 切单位: 保留框里数字不换算 → 数据层 x/y 随之改 (% 0.5 ↔ px 50 是不同数值, 但显示数字相同)
function setUnit(u: 'percent' | 'px') {
  const next: PointValue = { ...safeValue.value }
  const curDisplayX = displayX.value
  const curDisplayY = displayY.value
  if (u === 'px') {
    next.unit = 'px'
    next.x = curDisplayX // 框里数字原样进 px
    next.y = curDisplayY
  } else {
    delete next.unit
    next.x = round4(curDisplayX / 100) // 框里数字回比例
    next.y = round4(curDisplayY / 100)
  }
  emitVal(next)
}
</script>
```

UInputNumber 的 `:min`/`:max`/`:step` 按 `isPx` 切:px → `:min="0" :step="1"` 无 max;% → `:min="0" :max="100" :step="0.1"`。label 文案 `X {{ unitLabel }}` / `Y {{ unitLabel }}`(unitLabel = isPx ? 'px' : '%')。

- [ ] **Step 5: i18n 单位文案**

`zh.ts`/`en.ts` 加 point widget 单位 key(放 `point_widget` 或复用既有结构化区,如 `structured_input`):`point_widget: { unit_percent: '百分比', unit_px: '像素' }`。en 对称。PointWidget 用 `t('point_widget.unit_percent')` 等。

- [ ] **Step 6: 跑测试 + 类型**

Run: `cd frontend && ./node_modules/.bin/vitest run src/components/containers/inline/PointWidget.test.ts && pnpm vue-tsc --noEmit`
Expected: PASS + 类型绿。

- [ ] **Step 7: Commit**

```bash
git add frontend/src/components/containers/nodeRegistry/index.ts frontend/src/components/containers/inline/PointWidget.vue frontend/src/components/containers/inline/PointWidget.test.ts frontend/src/i18n/zh.ts frontend/src/i18n/en.ts
git commit -m "feat(fe): PointWidget %/px 单位开关 + PointValue.unit"
```

---

### Task 9: PointWidget 截图取点按钮(自持 picker)

**Files:**
- Modify: `frontend/src/components/containers/inline/PointWidget.vue`
- Test: `frontend/src/components/containers/inline/PointWidget.test.ts`
- Modify: `frontend/src/i18n/zh.ts`、`en.ts`(取点按钮文案)

**Interfaces:**
- Consumes: `backend.tools.openScreenPicker('point', id, containerId)`、wails event `tools:picker-result`(payload `{xRatio,yRatio,screenW,screenH}`,view 已 emit screenW/H)。镜像 GeometryWidget.vue:332-354。

- [ ] **Step 1: 写失败测试**

```ts
it('截图取点 % 模式: 存比例', async () => {
  mockPicker({ xRatio: 0.3, yRatio: 0.7, screenW: 1920, screenH: 1080 })
  const wrapper = mountPointWidget({ x: 0, y: 0 })
  await clickPickButton(wrapper) // data-testid="point-pick-btn"
  const e = lastEmit(wrapper)
  expect(e).toEqual({ x: 0.3, y: 0.7 })
})

it('截图取点 px 模式: 存像素 (ratio×screen)', async () => {
  mockPicker({ xRatio: 0.5, yRatio: 0.5, screenW: 1920, screenH: 1080 })
  const wrapper = mountPointWidget({ x: 0, y: 0, unit: 'px' })
  await clickPickButton(wrapper)
  const e = lastEmit(wrapper)
  expect(e).toEqual({ x: 960, y: 540, unit: 'px' })
})
```

> `mockPicker` mock `backend.tools.openScreenPicker` + emit `tools:picker-result`(参照 GeometryWidget 测试或 useScreenPick 测试既有 mock 方式)。

- [ ] **Step 2: 跑测试确认失败**

Run: `cd frontend && ./node_modules/.bin/vitest run src/components/containers/inline/PointWidget.test.ts`
Expected: FAIL(无取点按钮)。

- [ ] **Step 3: PointWidget 加截图按钮**(`PointWidget.vue`,镜像 GeometryWidget)

template 加按钮:

```vue
<UButton
  size="xs" variant="soft" color="primary" icon="i-tabler-pointer"
  data-testid="point-pick-btn" :loading="picking"
  @click="onPickPoint"
>
  {{ t('point_widget.pick_point') }}
</UButton>
```

script 加(import `backend`、`awaitWailsEvent`、`useTemplatesStore`,`ref`):

```ts
import { ref } from 'vue'
import { backend } from '@/lib/backend'
import { awaitWailsEvent } from '@/composables/useWailsEvent'
import { useTemplatesStore } from '@/stores/templates'

type PointPayload = { xRatio: number; yRatio: number; screenW?: number; screenH?: number; cancelled?: boolean }
const tplStore = useTemplatesStore()
const picking = ref(false)

function genID(): string { return 'pick-pt-' + Math.random().toString(36).slice(2, 10) + '-' + Date.now() }

async function onPickPoint() {
  if (picking.value) return
  const id = genID()
  picking.value = true
  try {
    const waiter = awaitWailsEvent<{ id: string; payload: PointPayload }>('tools:picker-result', (p) => p?.id === id)
    const r = await backend.tools.openScreenPicker('point', id, tplStore.containerId)
    if (r === undefined) return
    const res = await waiter
    const p = res.payload
    if (!p || p.cancelled) return
    const next: PointValue = { ...safeValue.value }
    if (isPx.value && p.screenW && p.screenH) {
      next.unit = 'px'
      next.x = Math.round(p.xRatio * p.screenW)
      next.y = Math.round(p.yRatio * p.screenH)
    } else {
      delete next.unit
      next.x = round4(p.xRatio)
      next.y = round4(p.yRatio)
    }
    emitVal(next)
  } finally {
    picking.value = false
  }
}
```

- [ ] **Step 4: 跑测试 + 类型**

Run: `cd frontend && ./node_modules/.bin/vitest run src/components/containers/inline/PointWidget.test.ts && pnpm vue-tsc --noEmit`
Expected: PASS + 类型绿。

- [ ] **Step 5: i18n 按钮文案**

`zh.ts`/`en.ts` `point_widget` 加 `pick_point: '截图取点'`(en: `'Pick on screen'`)。

- [ ] **Step 6: Commit**

```bash
git add frontend/src/components/containers/inline/PointWidget.vue frontend/src/components/containers/inline/PointWidget.test.ts frontend/src/i18n/zh.ts frontend/src/i18n/en.ts
git commit -m "feat(fe): PointWidget 自持截图取点 (%/px 分流)"
```

---

### Task 10: 删旧 point 取点路径(useScreenPick + NodeInspector)

**Files:**
- Modify: `frontend/src/composables/containerEditor/useScreenPick.ts`
- Modify: `frontend/src/components/containers/NodeInspector.vue:128-159,1075-1077`
- Test: `frontend/src/composables/containerEditor/useScreenPick.test.ts`(若有,删 point 相关用例)

**Interfaces:**
- 删除:`useScreenPick` 的 `canPickPoint`/`onPickPoint`/`applyPoint`/`PointPayload`;NodeInspector 的 point 取点按钮 + `applyPoint` 配置。rect/color 路径保留不动。

- [ ] **Step 1: 删 useScreenPick point 路径**(`useScreenPick.ts`)

- 删 `type PointPayload`(行 11)。
- 删 opts 里的 `applyPoint`(行 33)。
- 删 `canPickPoint` computed(行 55-58)。
- 删 `onPickPoint`(行 82-85)。
- return 里删 `canPickPoint, onPickPoint`(行 105)。
- 保留 `canPickRect`/`onPickRect`/`onPickColor`/`onOpenHUD`/`picking`。

- [ ] **Step 2: 删 NodeInspector point 按钮 + applyPoint**(`NodeInspector.vue`)

- `<section v-if="canPickPoint || canPickRect">` 改为 `v-if="canPickRect"`(行 130)。
- 删整个 `<UButton v-if="canPickPoint" ... @click="onPickPoint">…</UButton>`(行 138-148)。
- useScreenPick 解构(行 1075)删 `canPickPoint, onPickPoint`。
- 删 `applyPoint: (x, y) => setLiteralBatch({ XRatio: round4(x), YRatio: round4(y) }),`(行 1077)。
- 若 `setLiteralBatch`/`round4` 因此变 unused,一并清理(grep 确认无其它引用再删)。

- [ ] **Step 3: 类型 + 相关单测**

Run: `cd frontend && pnpm vue-tsc --noEmit`
Expected: 无 `canPickPoint`/`onPickPoint`/`applyPoint` 未定义或 unused 报错。

若 `useScreenPick.test.ts` 存在且测 point:删那些用例;rect/color 用例保留。
Run: `cd frontend && ./node_modules/.bin/vitest run src/composables/containerEditor/`
Expected: PASS。

- [ ] **Step 4: Commit**

```bash
git add frontend/src/composables/containerEditor/useScreenPick.ts frontend/src/components/containers/NodeInspector.vue frontend/src/composables/containerEditor/useScreenPick.test.ts
git commit -m "refactor(fe): 删 NodeInspector point 取点路径 (搬进 PointWidget)"
```

---

### Task 11: 全量验证 + 真机 smoke 清单

**Files:** 无(纯验证)

- [ ] **Step 1: Go 全量**

Run: `go build ./... && go test ./pkg/input/... ./internal/...`
Expected: 绿,除预存红基线(`TestApplyDirection_*` / `TestWatchdog_*` / `TestFishingV2Main_StateCycleSmoke`)。逐条核对失败项确属基线、非本次改动引入。

- [ ] **Step 2: catalog 一致性**

Run: `cd frontend && pnpm gen:node-i18n && cd .. && go test ./internal/catalog/...`
Expected: PASS(ClickAt/Scroll/MouseMoveTo/MakePoint 的 pin 与 i18n 全对齐)。

- [ ] **Step 3: FE 单测 + 类型**

Run: `cd frontend && ./node_modules/.bin/vitest run src/components/containers/inline/ src/composables/containerEditor/ && pnpm vue-tsc --noEmit`
Expected: PASS + 类型绿。

- [ ] **Step 4: 出 exe**

Run: `task build`
Expected: 成功产 exe(对照 build.md 判红基线)。

- [ ] **Step 5: 真机 smoke 清单(交用户验)**

记入 cockpit 待验,逐条:
1. ClickAt:% 模式填 90/90 点右下;切 px 填具体像素点同处;截图取点(% 与 px 各一次)落点正确。
2. Scroll / MouseMoveTo:同样 %/px/取点 各验一次。
3. Swipe:Begin/End 用 PointWidget 手填(% 与 px)+ 截图取点;拖拽轨迹对。
4. MakePoint:连两个 Number(或 Add 输出)→ Point → 喂 ClickAt,% 与 px(Unit 下拉)各验落点。
5. Point 连线:检测节点 Point 输出 → ClickAt Point 输入,点中。

全过 → 移 `work/point-px-unit` 到 cold store。

- [ ] **Step 6: Commit(若验证产生 gen 产物变更)**

```bash
git add -A && git commit -m "chore: point-px-unit 全量验证 + catalog 产物"
```

---

## Self-Review(规划期自查)

**Spec 覆盖**:§1 PointUnit→T1;§2 ResolvePoint→T2,4 节点 resolve→T3-6;§3 PointWidget→T8(单位)+T9(取点);§4 截图尺寸(view 已就绪)→T9 消费;§5 删旧路径→T10;§6 i18n→折进 T3-9;§7 MakePoint→T7;§8 取舍(MakePoint 抹平)→T7。全覆盖。
**Placeholder 扫描**:每步含真实 test/impl 代码 + 确切命令。无 TBD。
**类型一致**:`ResolvePoint(ctx,p)(x,y,err)` T2 定义、T3-6 一致调用;`PointUnit`/`UnitPx`/`UnitRatio` 全程统一;`MakePoint` Unit 用 `percent`/`px` dropdown、Evaluate 映射到 `UnitPx`/`UnitRatio` 一致;`PointValue{x,y,unit?}` T8 定义、T9 消费一致。
**已知不确定(实现时核)**:gen:node-i18n 产物的确切文件路径(以 `git status` 为准);PointWidget.test.ts 既有 helper 名;`asString` helper 是否存在(否则内联)。
