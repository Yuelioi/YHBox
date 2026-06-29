# Phase 2:ClickTemplate 全家桶 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: 用 superpowers:subagent-driven-development 逐 task 实现。步骤用 `- [ ]` 复选框跟踪。依赖 Phase 1(已落地:`Match/WaitMatch` 返回 `node.MatchHit{Found,Point,BBox,Conf}`、带 roi 参数;`node.TemplateMatch{Point,Conf,BBox,TemplateKey}`、`MatchAll(ctx,keys,threshold,minDistance,roi)`)。

**Goal:** 给 ClickTemplate 加 锚点+偏移(①)、order_by+index 多命中选择(⑤)、ROI 搜索区(⑧)、组合键+多击(③④),全部默认 = 旧行为(零回归)。

**Architecture:** 抽一个统一 `locateOnce`(默认走单帧 WaitMatch、order_by/index 非默认走 MatchAll+pickMatch),ClickTemplate.Run 围绕它做「获取→settle 重定位→点击→MaxAttempts 重点」;点击点由 `anchorPoint(bbox, anchor, offset)` 算;点击经 `clickWithMods`(修饰键 + 连点)。

**Tech Stack:** Go;`internal/node`(ResolveScalar helper)、`internal/nodes/detect/click_template.go`。

## Global Constraints

- **不要兼容**:不留 deprecated/shim(spec §1)。
- **新增 pin 默认严格 = 旧行为(零回归)**:`Anchor=center, OffsetX=0, OffsetY=0, OrderBy=score, Index=0, ROI=空, Keys="", ClickCount=1` ⇒ 行为与 Phase 1 后的 ClickTemplate 完全一致。
- **单位约定(OffsetX/Y)**:`|v|≤1`=客户区比例(`1`=100%)、`|v|>1`=像素(经 `ctx.Window().ClientSize()` 换算);负值保留符号。唯一权威实现 = `node.ResolveScalar`。
- **TDD**:每个行为变更先写测试再实现。
- 构建:`go build ./...` + `go test ./internal/node/... ./internal/nodes/detect/...`;改 pin 后 `cd frontend && pnpm gen:node-i18n` 再 `go test ./internal/catalog/...`。
- **预存失败基线**(判红排除):见 `flightdeck/knowledge/build/build.md`。
- **本机 Write 故障**:Write 写文件可能尾部混入 `</content>` 标签,写完检查清掉;改 Go 优先用 Edit。

---

### Task 2.1: `node.ResolveScalar` 单位解析 helper

**Files:**
- Create: `internal/node/scalar.go`
- Test: `internal/node/scalar_test.go`

**Interfaces:**
- Produces: `func ResolveScalar(v float64, fullPx int) float64`(返回客户区比例 ratio)

- [ ] **Step 1:写失败测试**

`internal/node/scalar_test.go`:
```go
package node

import "testing"

func TestResolveScalar(t *testing.T) {
	cases := []struct {
		name   string
		v      float64
		fullPx int
		want   float64
	}{
		{"比例-半", 0.5, 1920, 0.5},
		{"比例-满幅1=100%", 1, 1920, 1},
		{"比例-负", -1, 1080, -1},
		{"像素-正", 12, 1920, 12.0 / 1920.0},
		{"像素-负", -30, 1080, -30.0 / 1080.0},
		{"像素-边界刚过1", 1.5, 100, 0.015},
		{"零", 0, 1920, 0},
		{"fullPx<=0退回原值", 12, 0, 12},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ResolveScalar(c.v, c.fullPx); got != c.want {
				t.Fatalf("ResolveScalar(%v,%d)=%v want %v", c.v, c.fullPx, got, c.want)
			}
		})
	}
}
```

- [ ] **Step 2:跑测试确认失败**

Run: `go test ./internal/node/ -run TestResolveScalar -v`
Expected: FAIL(`undefined: ResolveScalar`)

- [ ] **Step 3:实现**

`internal/node/scalar.go`:
```go
package node

// ResolveScalar 把"一值两单位"标量解析成客户区比例。
// 约定: |v|<=1 → 比例 (1=100%, 直接用); |v|>1 → 像素 (v/fullPx)。负值保留符号。
// fullPx<=0 无法换算像素 → 退回原值。
func ResolveScalar(v float64, fullPx int) float64 {
	if v >= -1 && v <= 1 {
		return v
	}
	if fullPx <= 0 {
		return v
	}
	return v / float64(fullPx)
}
```

- [ ] **Step 4:跑测试确认通过**

Run: `go test ./internal/node/ -run TestResolveScalar -v`
Expected: PASS

- [ ] **Step 5:提交**

```bash
git add internal/node/scalar.go internal/node/scalar_test.go
git commit -m "feat(node): ResolveScalar — 一值两单位 (|v|<=1 比例/>1 像素) 解析 helper"
```

---

### Task 2.2: ClickTemplate ROI + OrderBy + Index(多命中选择)

重构 Run 围绕统一 `locateOnce`:默认(score/0)走单帧 WaitMatch,非默认走 MatchAll+pickMatch。加 ROI/OrderBy/Index 三个 pin。

**Files:**
- Modify: `internal/nodes/detect/click_template.go`
- Test: `internal/nodes/detect/click_template_test.go`

**Interfaces:**
- Consumes(Phase 1):`ctx.Vision().WaitMatch(ctx, keys, threshold, roi, timeout) (node.MatchHit, error)`、`ctx.Vision().MatchAll(ctx, keys, threshold, minDistance, roi) ([]node.TemplateMatch, error)`、`node.TemplateMatch{Point,Conf,BBox,TemplateKey}`
- Produces:`detect.locateOnce(ctx, keys, threshold, roi, orderBy, index) (node.MatchHit, error)`、`detect.pickMatch(matches, orderBy, index) (node.TemplateMatch, bool)`

- [ ] **Step 1:写失败测试 — pickMatch 排序 + index**

加到 `click_template_test.go`:
```go
func TestPickMatch(t *testing.T) {
	ms := []node.TemplateMatch{
		{Point: node.Point{X: 0.8, Y: 0.1}, Conf: 0.90, BBox: [4]float64{0.8, 0.1, 0.05, 0.05}}, // 右上, 小
		{Point: node.Point{X: 0.2, Y: 0.9}, Conf: 0.99, BBox: [4]float64{0.2, 0.9, 0.20, 0.20}}, // 左下, 大, 最高分
		{Point: node.Point{X: 0.5, Y: 0.5}, Conf: 0.85, BBox: [4]float64{0.5, 0.5, 0.10, 0.10}}, // 中
	}
	// horizontal: 按 BBox.x 升序 → [0.2,0.5,0.8]; index0 = x=0.2
	if hit, ok := pickMatch(ms, "horizontal", 0); !ok || hit.BBox[0] != 0.2 {
		t.Fatalf("horizontal idx0 → x=%v ok=%v", hit.BBox[0], ok)
	}
	// vertical: 按 BBox.y 升序 → [0.1,0.5,0.9]; index0 = y=0.1
	if hit, ok := pickMatch(ms, "vertical", 0); !ok || hit.BBox[1] != 0.1 {
		t.Fatalf("vertical idx0 → y=%v", hit.BBox[1])
	}
	// area: 面积降序 → 0.04(0.2),0.01(0.5),0.0025(0.8); index0 = 大块 x=0.2
	if hit, ok := pickMatch(ms, "area", 0); !ok || hit.BBox[0] != 0.2 {
		t.Fatalf("area idx0 → x=%v", hit.BBox[0])
	}
	// score(默认): 已按 conf 降序传入 → index0 = 传入首项 x=0.8
	if hit, ok := pickMatch(ms, "score", 0); !ok || hit.BBox[0] != 0.8 {
		t.Fatalf("score idx0 → x=%v", hit.BBox[0])
	}
	// index 越界 → ok=false
	if _, ok := pickMatch(ms, "score", 5); ok {
		t.Fatalf("index 越界应 ok=false")
	}
}
```
> 注:`score` 不重排(MatchAll 已 conf 降序);`random` 不在单测里断言顺序(非确定),只在实现里洗牌。

- [ ] **Step 2:跑测试确认失败**

Run: `go test ./internal/nodes/detect/ -run TestPickMatch -v`
Expected: FAIL(`undefined: pickMatch`)

- [ ] **Step 3:实现 pickMatch + locateOnce + 新 pin + 重构 Run**

加 pin 常量与 Spec(在 `clkInThreshold` 后插 ROI;在末尾加 OrderBy/Index):
```go
	clkInROI        = "ROI"
	clkInOrderBy    = "OrderBy"
	clkInIndex      = "Index"
```
Spec.Inputs 加(ROI 放 Threshold 后,OrderBy/Index 放 RetryIntervalMs 后,均 Advanced 除 ROI):
```go
	{Name: clkInROI, Type: "Geometry", Schema: node.GeometrySchema()},
	// ... 末尾:
	{Name: clkInOrderBy, Type: "String", Default: "score", Advanced: true,
		Widget: node.WidgetSpec{Kind: "dropdown",
			Props: node.MarshalProps(node.DropdownProps{
				Options: []node.EnumOption{
					{Value: "score"}, {Value: "horizontal"}, {Value: "vertical"}, {Value: "area"}, {Value: "random"},
				}})}},
	{Name: clkInIndex, Type: "Number", Default: json.Number("0"), Advanced: true,
		Widget: node.WidgetSpec{Kind: "number"}},
```
加 helper(import `sort`、`math/rand`):
```go
// pickMatch 多命中按 orderBy 排序后取 index。score 不重排 (MatchAll 已 conf 降序)。
func pickMatch(matches []node.TemplateMatch, orderBy string, index int) (node.TemplateMatch, bool) {
	if index < 0 || index >= len(matches) {
		return node.TemplateMatch{}, false
	}
	sorted := make([]node.TemplateMatch, len(matches))
	copy(sorted, matches)
	switch orderBy {
	case "horizontal":
		sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].BBox[0] < sorted[j].BBox[0] })
	case "vertical":
		sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].BBox[1] < sorted[j].BBox[1] })
	case "area":
		sort.SliceStable(sorted, func(i, j int) bool {
			return sorted[i].BBox[2]*sorted[i].BBox[3] > sorted[j].BBox[2]*sorted[j].BBox[3]
		})
	case "random":
		rand.Shuffle(len(sorted), func(i, j int) { sorted[i], sorted[j] = sorted[j], sorted[i] })
	default: // score: 保持 MatchAll 的 conf 降序
	}
	return sorted[index], true
}

// locateOnce 单帧定位选中命中。默认 (score/0) 走单帧 WaitMatch (保留 variant.BBox 快速定位);
// 非默认走 MatchAll + pickMatch。Found=false 表本帧没选到。
func locateOnce(ctx node.Ctx, keys []string, threshold float64, roi node.Geometry, orderBy string, index int) (node.MatchHit, error) {
	if (orderBy == "" || orderBy == "score") && index == 0 {
		return ctx.Vision().WaitMatch(ctx.Context(), keys, threshold, roi, 0)
	}
	matches, err := ctx.Vision().MatchAll(ctx.Context(), keys, threshold, 0, roi)
	if err != nil {
		return node.MatchHit{}, err
	}
	tm, ok := pickMatch(matches, orderBy, index)
	if !ok {
		return node.MatchHit{}, nil
	}
	return node.MatchHit{Found: true, Point: tm.Point, BBox: tm.BBox, Conf: tm.Conf}, nil
}
```
重构 Run:把开头的 `WaitMatch(...timeout)` 一发命中改成「轮询 locateOnce 直到命中或 timeout」,settle/MaxAttempts 的 `matchOnce` 也换成 `locateOnce`(带 roi/orderBy/index)。新 Run:
```go
func (ClickTemplate) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	keys := in.StringList(clkInTemplates)
	threshold := in.Float64(clkInThreshold)
	roi := in.Geometry(clkInROI)
	orderBy := in.String(clkInOrderBy)
	index := in.Int(clkInIndex)
	timeout := time.Duration(in.Int(clkInTimeoutMs)) * time.Millisecond
	settle := time.Duration(in.Int(clkInSettleMs)) * time.Millisecond
	maxAttempts := in.Int(clkInMaxAttempts)
	retryInterval := time.Duration(in.Int(clkInRetryIntervalMs)) * time.Millisecond
	btn := in.String(clkInButton)
	if btn == "" {
		btn = "left"
	}

	// 获取: 轮询 locateOnce 到命中或超时 (单帧 locateOnce 替代 WaitMatch 内轮询, 统一两路径)。
	hit, err := acquire(ctx, keys, threshold, roi, orderBy, index, timeout)
	if err != nil {
		return nil, node.Failf(node.CodeCaptureFailed, err, "ClickTemplate wait %s: %v", strings.Join(keys, "+"), err)
	}
	if !hit.Found {
		return ctx.Out(clkOutTimeout).Set(clkDataConf, hit.Conf).Set(clkDataMatched, false).Fire(), nil
	}

	// settle: 等稳 + 重定位一次。
	if settle > 0 {
		if err := waitOrCancel(ctx, settle); err != nil {
			return nil, err
		}
		if h2, err := locateOnce(ctx, keys, threshold, roi, orderBy, index); err == nil && h2.Found {
			hit = h2
		}
	}

	if err := clickAt(ctx, keys, hit.Point, btn); err != nil {
		return nil, err
	}
	if maxAttempts <= 1 {
		return ctx.Out(clkOutDone).Set(clkDataPoint, hit.Point).Set(clkDataConf, hit.Conf).Set(clkDataMatched, true).Fire(), nil
	}
	clicks := 1
	for {
		if err := waitOrCancel(ctx, retryInterval); err != nil {
			return nil, err
		}
		h2, err := locateOnce(ctx, keys, threshold, roi, orderBy, index)
		if err != nil {
			return nil, node.Failf(node.CodeCaptureFailed, err, "ClickTemplate recheck %s: %v", strings.Join(keys, "+"), err)
		}
		if !h2.Found {
			return ctx.Out(clkOutDone).Set(clkDataPoint, hit.Point).Set(clkDataConf, hit.Conf).Set(clkDataMatched, true).Fire(), nil
		}
		if clicks >= maxAttempts {
			return ctx.Out(clkOutTimeout).Set(clkDataConf, h2.Conf).Set(clkDataMatched, true).Fire(), nil
		}
		hit = h2
		if err := clickAt(ctx, keys, hit.Point, btn); err != nil {
			return nil, err
		}
		clicks++
	}
}

// acquire 轮询 locateOnce 直到命中或 timeout。timeout<=0 单帧。记录见过的最高 conf 供 Timeout 诊断。
func acquire(ctx node.Ctx, keys []string, threshold float64, roi node.Geometry, orderBy string, index int, timeout time.Duration) (node.MatchHit, error) {
	hit, err := locateOnce(ctx, keys, threshold, roi, orderBy, index)
	if err != nil {
		return node.MatchHit{}, err
	}
	if hit.Found || timeout <= 0 {
		return hit, nil
	}
	deadline := ctx.Now().Add(timeout)
	bestConf := hit.Conf
	for {
		if err := ctx.Context().Err(); err != nil {
			return node.MatchHit{}, err
		}
		if err := waitOrCancel(ctx, visionPollInterval); err != nil {
			return node.MatchHit{}, err
		}
		hit, err = locateOnce(ctx, keys, threshold, roi, orderBy, index)
		if err != nil {
			return node.MatchHit{}, err
		}
		if hit.Conf > bestConf {
			bestConf = hit.Conf
		}
		if hit.Found {
			return hit, nil
		}
		if ctx.Now().After(deadline) {
			return node.MatchHit{Conf: bestConf}, nil
		}
	}
}
```
> `visionPollInterval` = 复用一个 100ms 常量(若 detect 包内已有同义常量直接用;否则在本文件加 `const visionPollInterval = 100 * time.Millisecond`)。`ctx.Now()` 见 node Ctx 基础方法。

- [ ] **Step 4:跑测试确认通过**

Run: `go test ./internal/nodes/detect/ -run TestPickMatch -v`
Expected: PASS

- [ ] **Step 5:补行为测试 — order_by 端到端 + 默认零回归**

加测试:mock vision(`mockVision` 扩展支持 `MatchAll` 返回多命中)→ ClickTemplate(OrderBy=vertical,Index=0)点中最上面那个(断言传给 Input.Click 的坐标 = 最上 match 的中心);ClickTemplate 默认(score/0)单命中行为与 Phase 1 一致(走 WaitMatch 路径,点中心)。

- [ ] **Step 6:跑测试 + i18n**

Run: `go test ./internal/nodes/detect/`;再 `cd frontend && pnpm gen:node-i18n` + `go test ./internal/catalog/...`
Expected: PASS

- [ ] **Step 7:提交**

```bash
git add internal/nodes/detect/click_template.go internal/nodes/detect/click_template_test.go internal/catalog/
git commit -m "feat(ClickTemplate): ROI 搜索区 + order_by/index 多命中选择 (默认 score/0 零回归)"
```

---

### Task 2.3: ClickTemplate Anchor + OffsetX/OffsetY

在已选中命中的 bbox 上按九宫格锚点 + 偏移算落点。默认 center/0/0 = 命中框中心(= 旧行为)。

**Files:**
- Modify: `internal/nodes/detect/click_template.go`
- Test: `internal/nodes/detect/click_template_test.go`

**Interfaces:**
- Consumes: `node.ResolveScalar`(Task 2.1)、`ctx.Window().ClientSize() (w,h int, err error)`、`hit.BBox`
- Produces: `detect.anchorPoint(bbox [4]float64, anchor string, offX, offY float64) node.Point`

- [ ] **Step 1:写失败测试 — anchorPoint**

```go
func TestAnchorPoint(t *testing.T) {
	bb := [4]float64{0.2, 0.4, 0.10, 0.20} // x,y,w,h → 中心 (0.25,0.5)
	if p := anchorPoint(bb, "center", 0, 0); p.X != 0.25 || p.Y != 0.5 {
		t.Fatalf("center=%v want (0.25,0.5)", p)
	}
	if p := anchorPoint(bb, "topLeft", 0, 0); p.X != 0.2 || p.Y != 0.4 {
		t.Fatalf("topLeft=%v want (0.2,0.4)", p)
	}
	if p := anchorPoint(bb, "botRight", 0, 0); p.X != 0.3 || p.Y != 0.6 {
		t.Fatalf("botRight=%v want (0.3,0.6)", p)
	}
	// 偏移 (已是 ratio): topRight + (0.05,-0.05)
	if p := anchorPoint(bb, "topRight", 0.05, -0.05); p.X != 0.35 || p.Y != 0.35 {
		t.Fatalf("topRight+off=%v want (0.35,0.35)", p)
	}
	// clamp: 越界裁到 [0,1]
	if p := anchorPoint(bb, "botRight", 1.0, 0); p.X != 1 {
		t.Fatalf("clamp X=%v want 1", p.X)
	}
}
```

- [ ] **Step 2:跑测试确认失败**

Run: `go test ./internal/nodes/detect/ -run TestAnchorPoint -v`
Expected: FAIL(`undefined: anchorPoint`)

- [ ] **Step 3:实现 anchorPoint + 接进 Run**

加 pin 常量 + Spec(ROI 后插 Anchor/OffsetX/OffsetY,Advanced):
```go
	clkInAnchor  = "Anchor"
	clkInOffsetX = "OffsetX"
	clkInOffsetY = "OffsetY"
```
```go
	{Name: clkInAnchor, Type: "String", Default: "center", Advanced: true,
		Widget: node.WidgetSpec{Kind: "dropdown",
			Props: node.MarshalProps(node.DropdownProps{
				Options: []node.EnumOption{
					{Value: "topLeft"}, {Value: "topCenter"}, {Value: "topRight"},
					{Value: "midLeft"}, {Value: "center"}, {Value: "midRight"},
					{Value: "botLeft"}, {Value: "botCenter"}, {Value: "botRight"},
				}})}},
	{Name: clkInOffsetX, Type: "Number", Default: json.Number("0"), Advanced: true,
		Widget: node.WidgetSpec{Kind: "number"}},
	{Name: clkInOffsetY, Type: "Number", Default: json.Number("0"), Advanced: true,
		Widget: node.WidgetSpec{Kind: "number"}},
```
helper:
```go
func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// anchorPoint 按九宫格锚点 + ratio 偏移算落点 (归一化, clamp [0,1])。
// bbox=[x,y(左上),w,h]; offX/offY 已是 ratio (调用方经 ResolveScalar 换算)。
func anchorPoint(bbox [4]float64, anchor string, offX, offY float64) node.Point {
	var fx, fy float64
	switch anchor {
	case "topLeft":
		fx, fy = 0, 0
	case "topCenter":
		fx, fy = 0.5, 0
	case "topRight":
		fx, fy = 1, 0
	case "midLeft":
		fx, fy = 0, 0.5
	case "midRight":
		fx, fy = 1, 0.5
	case "botLeft":
		fx, fy = 0, 1
	case "botCenter":
		fx, fy = 0.5, 1
	case "botRight":
		fx, fy = 1, 1
	default: // center
		fx, fy = 0.5, 0.5
	}
	return node.Point{
		X: clamp01(bbox[0] + bbox[2]*fx + offX),
		Y: clamp01(bbox[1] + bbox[3]*fy + offY),
	}
}
```
在 Run 里:读 anchor/offsetX/offsetY;在每次点击前把 `hit.Point` 换成 `anchorPoint(hit.BBox, anchor, offRatioX, offRatioY)`。偏移换算放一个小闭包(需 ClientSize):
```go
	anchor := in.String(clkInAnchor)
	offXRaw := in.Float64(clkInOffsetX)
	offYRaw := in.Float64(clkInOffsetY)
	// 解析偏移单位 (像素需客户区尺寸)
	offX, offY := offXRaw, offYRaw
	if (offXRaw < -1 || offXRaw > 1) || (offYRaw < -1 || offYRaw > 1) {
		if w, h, err := ctx.Window().ClientSize(); err == nil {
			offX = node.ResolveScalar(offXRaw, w)
			offY = node.ResolveScalar(offYRaw, h)
		}
	}
	clickPt := func(hit node.MatchHit) node.Point { return anchorPoint(hit.BBox, anchor, offX, offY) }
```
把 Run 里两处 `clickAt(ctx, keys, hit.Point, btn)` 改成 `clickAt(ctx, keys, clickPt(hit), btn)`;Done 出口的 `clkDataPoint` 改吐 `clickPt(hit)`(最终落点)。
> 默认 center/0/0 时 `clickPt(hit)` == 命中框中心 == `hit.Point`(零回归,单测钉死)。

- [ ] **Step 4:跑测试确认通过**

Run: `go test ./internal/nodes/detect/ -run TestAnchorPoint -v`
Expected: PASS

- [ ] **Step 5:补端到端 + 零回归测试**

mock vision 返 bbox + mock window ClientSize + mock input 记录点击坐标 → ClickTemplate(Anchor=topRight,OffsetX=0)点在命中框右上;默认(center/0/0)点在中心(== Phase 2 Task 2.2 后的中心行为,零回归)。

- [ ] **Step 6:跑测试 + i18n**

Run: `go test ./internal/nodes/detect/`;`cd frontend && pnpm gen:node-i18n` + `go test ./internal/catalog/...`
Expected: PASS

- [ ] **Step 7:提交**

```bash
git add internal/nodes/detect/click_template.go internal/nodes/detect/click_template_test.go internal/catalog/
git commit -m "feat(ClickTemplate): 九宫格锚点 + 偏移落点 (默认 center/0 = 命中框中心, 零回归)"
```

---

### Task 2.4: ClickTemplate Keys(组合键)+ ClickCount(多击)

抽公共 `clickWithMods` helper(Phase 3 ClickAt 也会用),给 ClickTemplate 加 Keys/ClickCount。

**Files:**
- Modify: `internal/nodes/detect/click_template.go`(改 `clickAt` 走 helper)
- Create: `internal/nodes/detect/click_common.go`(放 `clickWithMods` + 修饰键解析/校验)
- Test: `internal/nodes/detect/click_common_test.go`

**Interfaces:**
- Consumes:`ctx.Input().KeyDown(vk)`、`ctx.Input().KeyUp(vk)`、`ctx.Input().Click(x,y,btn,durationMs)`
- Produces:`detect.clickWithMods(ctx, pt node.Point, btn string, keys string, count int) error`、`detect.parseMods(keys string) ([]string, bool)`(bool=全合法)

- [ ] **Step 1:写失败测试 — parseMods 校验 + clickWithMods 序列**

```go
func TestParseMods(t *testing.T) {
	if mods, ok := parseMods("ctrl+shift"); !ok || len(mods) != 2 || mods[0] != "ctrl" || mods[1] != "shift" {
		t.Fatalf("ctrl+shift → %v ok=%v", mods, ok)
	}
	if mods, ok := parseMods(""); !ok || len(mods) != 0 {
		t.Fatalf("空 → %v ok=%v", mods, ok)
	}
	if _, ok := parseMods("ctrl+foo"); ok {
		t.Fatalf("非法修饰键应 ok=false")
	}
}

func TestClickWithMods_Sequence(t *testing.T) {
	rec := &recInput{} // 记录 KeyDown/Click/KeyUp 调用序列
	ctx := newTestCtxWithInput(rec) // 复用本包测试 ctx 构造
	if err := clickWithMods(ctx, node.Point{X: 0.5, Y: 0.5}, "left", "ctrl+shift", 2); err != nil {
		t.Fatal(err)
	}
	// 期望: KeyDown(ctrl) KeyDown(shift) Click Click KeyUp(shift) KeyUp(ctrl)
	want := []string{"KeyDown:ctrl", "KeyDown:shift", "Click", "Click", "KeyUp:shift", "KeyUp:ctrl"}
	if !rec.matches(want) {
		t.Fatalf("序列=%v want %v", rec.seq, want)
	}
}
```
> `recInput`/`newTestCtxWithInput`/`recInput.matches` 按本包既有 mock 风格实现(实现者参照 `click_template_test.go` 里 mock input 的构造方式;若没有现成 mock input,在本测试文件加一个记录序列的 InputService stub)。

- [ ] **Step 2:跑测试确认失败**

Run: `go test ./internal/nodes/detect/ -run 'TestParseMods|TestClickWithMods' -v`
Expected: FAIL(undefined)

- [ ] **Step 3:实现 click_common.go**

```go
package detect

import (
	"strings"
	"time"

	"yotta/internal/node"
)

const interClickGapMs = 60 // 连点间隔, < 系统双击时限 (500ms)

var validMods = map[string]bool{"ctrl": true, "shift": true, "alt": true, "win": true}

// parseMods 解析 "ctrl+shift" → ["ctrl","shift"]; 全合法返 true。空串 → (nil,true)。
func parseMods(keys string) ([]string, bool) {
	keys = strings.TrimSpace(keys)
	if keys == "" {
		return nil, true
	}
	var mods []string
	for _, p := range strings.Split(keys, "+") {
		m := strings.ToLower(strings.TrimSpace(p))
		if !validMods[m] {
			return nil, false
		}
		mods = append(mods, m)
	}
	return mods, true
}

// clickWithMods 按住修饰键 → 连点 count 次 → 逆序松开。count<=1 单击。
func clickWithMods(ctx node.Ctx, pt node.Point, btn string, keys string, count int) error {
	mods, _ := parseMods(keys) // 合法性由节点 Validate 保证
	for _, m := range mods {
		if err := ctx.Input().KeyDown(m); err != nil {
			return err
		}
	}
	if count < 1 {
		count = 1
	}
	var clickErr error
	for i := 0; i < count; i++ {
		if clickErr = ctx.Input().Click(pt.X, pt.Y, btn, 50); clickErr != nil {
			break
		}
		if i < count-1 {
			if err := waitOrCancel(ctx, interClickGapMs*time.Millisecond); err != nil {
				clickErr = err
				break
			}
		}
	}
	// 无论点击成败, 逆序松开修饰键 (别卡着 ctrl)
	for i := len(mods) - 1; i >= 0; i-- {
		_ = ctx.Input().KeyUp(mods[i])
	}
	return clickErr
}
```

- [ ] **Step 4:ClickTemplate 接入 Keys/ClickCount**

加 pin 常量 + Spec(Advanced):
```go
	clkInKeys       = "Keys"
	clkInClickCount = "ClickCount"
```
```go
	{Name: clkInKeys, Type: "String", Default: "", Advanced: true,
		Widget: node.WidgetSpec{Kind: "text"}},
	{Name: clkInClickCount, Type: "Number", Default: json.Number("1"), Advanced: true,
		Widget: node.WidgetSpec{Kind: "number"}},
```
把 `clickAt(ctx, keys, pt, btn)` 调用改为 `clickWithMods(ctx, clickPt(hit), btn, modKeys, clickCount)`(读 `modKeys := in.String(clkInKeys)`、`clickCount := in.Int(clkInClickCount)`);删掉旧的 `clickAt` 函数(已被 clickWithMods 取代)或保留为薄封装——优先删,改所有调用点。
Validate 加:
```go
	if _, ok := parseMods(in.String(clkInKeys)); !ok {
		errs = append(errs, node.ValidationError{Code: "INVALID_MODIFIER_KEY",
			Message: "Keys 含非法修饰键 (仅 ctrl/shift/alt/win)", Field: clkInKeys})
	}
	if in.Int(clkInClickCount) < 1 {
		errs = append(errs, node.ValidationError{Code: "INVALID_CLICK_COUNT",
			Message: "ClickCount 必须 >= 1", Field: clkInClickCount})
	}
```

- [ ] **Step 5:跑测试确认通过**

Run: `go test ./internal/nodes/detect/ -v`
Expected: PASS(含新 + 旧;默认 Keys=""/ClickCount=1 = 单击无修饰,零回归)

- [ ] **Step 6:i18n + 全量**

Run: `cd frontend && pnpm gen:node-i18n` + `go test ./internal/node/... ./internal/nodes/detect/... ./internal/catalog/...`
Expected: PASS

- [ ] **Step 7:提交**

```bash
git add internal/nodes/detect/ internal/catalog/
git commit -m "feat(ClickTemplate): 组合键 Keys + 多击 ClickCount (抽 clickWithMods 公共 helper)"
```

---

## 后续

- **Phase 3**:新节点群(WaitTemplateGone / Swipe / InputText / StopApp / Scroll 横向)+ ClickAt 接入 `clickWithMods`(复用本 Phase Task 2.4 的 helper)。含 InputService/pkg.input 后端新增。

## Self-Review(已过)

- **spec 覆盖**:本 Phase 对应 spec ①(Task 2.3)、⑤(Task 2.2)、⑧(Task 2.2 ROI)、③④(Task 2.4),单位约定(Task 2.1)。WaitTemplateGone/Swipe/InputText/StopApp/Scroll 横向归 Phase 3。
- **占位符**:测试 mock 复用本包既有风格(参照 click_template_test.go),非占位;`visionPollInterval` 给了落点。
- **类型一致**:`locateOnce`/`pickMatch`/`anchorPoint`/`clickWithMods`/`parseMods` 签名贯穿一致;`node.MatchHit`/`node.TemplateMatch` 用 Phase 1 定义。
- **零回归**:每 task 默认值都钉死 = 旧行为,且各有回归测试。
