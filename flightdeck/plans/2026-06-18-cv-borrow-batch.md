---
status: active
summary: 实现 cv-borrow-batch spec 的 3 节点: FindColorSignature → DecodeQR → FindTemplateAll, 按风险递增 TDD 落地。
last_updated: 2026-06-18
implements: specs/2026-06-18-cv-borrow-batch.md
---

# CV 低依赖借鉴批 实现计划 (颜色签名 / QR / 模板全部命中)

> **For agentic workers:** REQUIRED SUB-SKILL: 用 superpowers:subagent-driven-development 或 superpowers:executing-plans
> 逐任务执行。步骤用 `- [ ]` 勾选。**规范**: 节点全链路照 [add-node.md](../checklists/add-node.md);pin 命名照
> [node-spec-style.md](../checklists/node-spec-style.md);数据流/捕获照 [node-data-flow](../checklists/2026-06-05-node-data-flow.md)。
> **本计划的设计契约 (类型/算法/边界) 全部冻结在 [spec](../specs/2026-06-18-cv-borrow-batch.md)** —— 撞细节回查 spec, 别脑补。

**Goal:** 给容器编辑器加 3 个纯 Go 视觉节点 (颜色签名 / QR 解码 / 模板全部命中), 零新增原生依赖。

**Architecture:** 每个能力走 `VisionService` 接口 (`internal/node/interfaces.go`) → `visionAdapter` 实现
(`internal/services/container/runtime/node_services.go`) → stub (`internal/node/services.go`); 纯算法落 `pkg/vision`
(不反向 import `internal/node`, 用本地基元类型); 节点落 `internal/nodes/detect/`。按风险递增: 颜色签名 (零依赖零核心改动)
→ QR (引 gozxing) → 模板全部命中 (动 matcher 接口 + 4 实现)。

**Tech Stack:** Go (纯, 无 CGO) · `golang.org/x/image` · `github.com/makiuchi-d/gozxing` (QR) · Vue3 i18n (前端文案) · Taskfile + vitest。

---

## 文件结构 (改/建 一览)

**Phase 1 — FindColorSignature**
- 建 `internal/node/color_signature.go` — `ColorSignature` / `ColorPoint` 类型 (跟 `Point`/`BlobEntry` 同处)
- 改 `internal/node/interfaces.go` — `VisionService` 加 `FindColorSignature`
- 改 `internal/node/services.go` — `stubVisionService` 加桩
- 建 `pkg/vision/color_signature.go` — 纯像素扫描 (本地 `ColorSigPoint` 类型)
- 建 `pkg/vision/color_signature_test.go`
- 改 `internal/services/container/runtime/node_services.go` — `visionAdapter.FindColorSignature`
- 建 `internal/nodes/detect/find_color_signature.go` + `_test.go`
- 改 `frontend/src/i18n/zh.ts` + `en.ts` — `node.FindColorSignature`

**Phase 2 — DecodeQR**
- 改 `go.mod` / `go.sum` — 加 gozxing (过 gate)
- 改 `internal/node/interfaces.go` — `VisionService` 加 `DecodeQR`; 建 `QRResult` 类型 (放 `color_signature.go` 或新 `qr.go`)
- 改 `internal/node/services.go` — 桩
- 建 `pkg/vision/qr.go` + `_test.go` — gozxing 封装 + 排序
- 改 `node_services.go` — `visionAdapter.DecodeQR`
- 建 `internal/nodes/detect/decode_qr.go` + `_test.go`
- 改 `zh.ts` + `en.ts` — `node.DecodeQR`

**Phase 3 — FindTemplateAll**
- 建 `pkg/vision/match_all.go` + `_test.go` — `MatchAll` (3×3 极大 + NMS)
- 改 `internal/node/interfaces.go` — `VisionService` 加 `MatchAll`; 建 `TemplateMatch` 类型
- 改 `internal/node/services.go` — 桩
- 改 `internal/services/container/runtime/interfaces.go` — `TemplateMatcher` 加 `DetectAll` + `NoopMatcher` 实现
- 改 `wire_container.go` — `templateMatcherAdapter.DetectAll` (生产)
- 改 `inspect_phase_test.go` (`mockMatcher`) + `node_services_test.go` (`stubMatcher`) — 加 `DetectAll`
- 改 `node_services.go` — `visionAdapter.MatchAll`
- 建 `internal/nodes/detect/find_template_all.go` + `_test.go`
- 改 `zh.ts` + `en.ts` — `node.FindTemplateAll`

---

## Phase 1 — FindColorSignature

### Task 1.1: 共享类型 `ColorSignature` / `ColorPoint`

**Files:** Create: `internal/node/color_signature.go`

- [ ] **Step 1: 写类型 (无逻辑, 直接建)**

```go
// internal/node/color_signature.go
// ColorSignature — FindColorSignature 用: 锚点 + N 偏移点的颜色签名。spec §节点1。
package node

// ColorPoint 签名里一个点。DX/DY 相对锚点的像素偏移 (锚点本身 DX=DY=0)。
// Tol *int 纯 nullable: nil → 用节点默认容差; 非 nil (含 0=严格) → 显式。R/G/B ∈ [0,255]。
type ColorPoint struct {
	DX, DY  int
	R, G, B int
	Tol     *int
}

// ColorSignature 点列表, Points[0] = 锚点。
type ColorSignature struct {
	Points []ColorPoint
}
```

- [ ] **Step 2: 编译**

Run: `go build ./internal/node/...`
Expected: PASS (无引用方, 仅确认类型合法)。

- [ ] **Step 3: Commit**

```bash
git add internal/node/color_signature.go
git commit -m "feat(node): add ColorSignature/ColorPoint types"
```

### Task 1.2: `pkg/vision` 纯像素签名扫描

**Files:** Create `pkg/vision/color_signature.go` · Test `pkg/vision/color_signature_test.go`

- [ ] **Step 1: 写失败测试**

```go
// pkg/vision/color_signature_test.go
package vision

import (
	"image"
	"testing"
)

// 造 20x20 RGBA: 在 (12,8) 放红 (200,30,30), 其右 (14,8) 放白 (255,255,255)。
func sigTestFrame() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 20, 20))
	for i := range img.Pix {
		img.Pix[i] = 0
	}
	set := func(x, y, r, g, b int) {
		o := img.PixOffset(x, y)
		img.Pix[o], img.Pix[o+1], img.Pix[o+2], img.Pix[o+3] = uint8(r), uint8(g), uint8(b), 255
	}
	set(12, 8, 200, 30, 30)
	set(14, 8, 255, 255, 255)
	return img
}

func TestFindColorSignature_HitWithOffset(t *testing.T) {
	f := sigTestFrame()
	sig := []ColorSigPoint{
		{DX: 0, DY: 0, R: 200, G: 30, B: 30, Tol: 8},
		{DX: 2, DY: 0, R: 255, G: 255, B: 255, Tol: 8},
	}
	found, ax, ay := FindColorSignature(f, 0, 0, 20, 20, sig)
	if !found || ax != 12 || ay != 8 {
		t.Fatalf("want (true,12,8), got (%v,%d,%d)", found, ax, ay)
	}
}

func TestFindColorSignature_OffsetOutOfFrameMiss(t *testing.T) {
	f := sigTestFrame()
	// 偏移点指向帧外 (dx=+100) → miss。
	sig := []ColorSigPoint{
		{DX: 0, DY: 0, R: 200, G: 30, B: 30, Tol: 8},
		{DX: 100, DY: 0, R: 255, G: 255, B: 255, Tol: 8},
	}
	if found, _, _ := FindColorSignature(f, 0, 0, 20, 20, sig); found {
		t.Fatal("offset out of frame should miss")
	}
}

func TestFindColorSignature_ROINonZeroOrigin(t *testing.T) {
	f := sigTestFrame()
	sig := []ColorSigPoint{{DX: 0, DY: 0, R: 200, G: 30, B: 30, Tol: 8}}
	// ROI 从 (10,5) 起, 仍应在 (12,8) 命中锚点。
	found, ax, ay := FindColorSignature(f, 10, 5, 8, 10, sig)
	if !found || ax != 12 || ay != 8 {
		t.Fatalf("want (true,12,8), got (%v,%d,%d)", found, ax, ay)
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `go test ./pkg/vision/ -run TestFindColorSignature -v`
Expected: FAIL `undefined: ColorSigPoint / FindColorSignature`。

- [ ] **Step 3: 写实现**

```go
// pkg/vision/color_signature.go
// FindColorSignature — 在 ROI 矩形内 (锚点搜索区) 行主序找首个完整签名命中。
// 锚点色命中的像素上验全部偏移点 (偏移点采样整帧, 越界=miss=整签名失败, 继续搜)。
// 纯基元类型, 不 import internal/node。spec §节点1。
package vision

import "image"

// ColorSigPoint 一个签名点, Tol 已被 caller 解析成具体值 (不再 nullable)。
type ColorSigPoint struct {
	DX, DY  int
	R, G, B int
	Tol     int
}

// chMatch 逐通道绝对差 ≤ tol (只比 RGB)。
func chMatch(r, g, b uint8, p ColorSigPoint) bool {
	return absI(int(r)-p.R) <= p.Tol && absI(int(g)-p.G) <= p.Tol && absI(int(b)-p.B) <= p.Tol
}

func absI(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// FindColorSignature: rx,ry,rw,rh = 锚点搜索区像素矩形 (半开 [rx,rx+rw)×[ry,ry+rh))。
// sig[0] = 锚点 (DX=DY=0)。返回锚点像素坐标 (全帧)。
func FindColorSignature(frame *image.RGBA, rx, ry, rw, rh int, sig []ColorSigPoint) (bool, int, int) {
	if len(sig) == 0 {
		return false, 0, 0
	}
	b := frame.Bounds()
	x0, y0 := maxI(rx, b.Min.X), maxI(ry, b.Min.Y)
	x1, y1 := minI(rx+rw, b.Max.X), minI(ry+rh, b.Max.Y)
	anchor := sig[0]
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			o := frame.PixOffset(x, y)
			if !chMatch(frame.Pix[o], frame.Pix[o+1], frame.Pix[o+2], anchor) {
				continue
			}
			if verifyOffsets(frame, x, y, sig) {
				return true, x, y
			}
		}
	}
	return false, 0, 0
}

// verifyOffsets 验 sig[1:] 各偏移点 (采样整帧; 越界即 false)。
func verifyOffsets(frame *image.RGBA, ax, ay int, sig []ColorSigPoint) bool {
	b := frame.Bounds()
	for _, p := range sig[1:] {
		px, py := ax+p.DX, ay+p.DY
		if px < b.Min.X || px >= b.Max.X || py < b.Min.Y || py >= b.Max.Y {
			return false
		}
		o := frame.PixOffset(px, py)
		if !chMatch(frame.Pix[o], frame.Pix[o+1], frame.Pix[o+2], p) {
			return false
		}
	}
	return true
}

func maxI(a, b int) int { if a > b { return a }; return b }
func minI(a, b int) int { if a < b { return a }; return b }
```

- [ ] **Step 4: 运行确认通过**

Run: `go test ./pkg/vision/ -run TestFindColorSignature -v`
Expected: PASS (3 个)。

- [ ] **Step 5: Commit**

```bash
git add pkg/vision/color_signature.go pkg/vision/color_signature_test.go
git commit -m "feat(vision): pure-Go color signature search"
```

### Task 1.3: `VisionService.FindColorSignature` 接口 + 桩 + adapter

**Files:** Modify `internal/node/interfaces.go` · `internal/node/services.go` · `internal/services/container/runtime/node_services.go`

- [ ] **Step 1: 接口加方法**

`internal/node/interfaces.go` 的 `VisionService` 接口尾部 (`GridSignature` 后) 加:

```go
	// FindColorSignature 在 roi (锚点搜索区) 找颜色签名首个完整命中, 偏移点采样整帧。
	// defaultTol 用于 ColorPoint.Tol==nil 的点。未命中 found=false。spec §节点1。
	FindColorSignature(roi Geometry, sig ColorSignature, defaultTol int) (found bool, pt Point, err error)
```

- [ ] **Step 2: 桩实现 (保持编译)**

`internal/node/services.go` 的 `stubVisionService` 加:

```go
func (stubVisionService) FindColorSignature(_ Geometry, _ ColorSignature, _ int) (bool, Point, error) {
	return false, Point{}, nil
}
```

- [ ] **Step 3: adapter 实现**

`node_services.go` 加 (照 `DualBarTrack`/`matchOnce` 的抓帧法: `ActiveHWND` + `CaptureFrameCached`; Geometry→子帧解析复用现有 `DetectColorBlobs` adapter 同款 helper — 读那个方法照抄解析行):

```go
func (a *visionAdapter) FindColorSignature(roi node.Geometry, sig node.ColorSignature, defaultTol int) (bool, node.Point, error) {
	if a.rt.Capture == nil {
		return false, node.Point{}, fmt.Errorf("capture backend not initialised")
	}
	h, err := a.rt.ActiveHWND()
	if err != nil {
		return false, node.Point{}, err
	}
	frame, err := a.rt.CaptureFrameCached(h)
	if err != nil {
		return false, node.Point{}, err
	}
	// roi 解析成像素矩形 rx,ry,rw,rh (复用 DetectColorBlobs adapter 同款 ResolveGeometry → pixel rect)。
	rx, ry, rw, rh := resolveGeometryPx(frame, roi) // ← 用 DetectColorBlobs adapter 里已有的解析法
	// ColorPoint.Tol *int → 具体 tol (nil 用 defaultTol)。
	pts := make([]vision.ColorSigPoint, len(sig.Points))
	for i, p := range sig.Points {
		tol := defaultTol
		if p.Tol != nil {
			tol = *p.Tol
		}
		pts[i] = vision.ColorSigPoint{DX: p.DX, DY: p.DY, R: p.R, G: p.G, B: p.B, Tol: tol}
	}
	found, ax, ay := vision.FindColorSignature(frame, rx, ry, rw, rh, pts)
	if !found {
		return false, node.Point{}, nil
	}
	fw, fh := frame.Bounds().Dx(), frame.Bounds().Dy()
	return true, node.Point{X: float64(ax) / float64(fw), Y: float64(ay) / float64(fh)}, nil
}
```

> 注: `resolveGeometryPx` 是占位名 —— 实现时打开 `node_services.go` 的 `DetectColorBlobs`/`DetectColorHSV` adapter,
> 把它解析 `node.Geometry` → 像素矩形的那几行原样复用 (项目已有 helper, 别新造)。

- [ ] **Step 4: 编译**

Run: `go build ./...`
Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add internal/node/interfaces.go internal/node/services.go internal/services/container/runtime/node_services.go
git commit -m "feat(vision): wire FindColorSignature through VisionService"
```

### Task 1.4: 节点 `FindColorSignature` (Spec + Run + Validate)

**Files:** Create `internal/nodes/detect/find_color_signature.go` · Test `internal/nodes/detect/find_color_signature_test.go`

- [ ] **Step 1: 写失败测试** (用 stub vision 验路由 + 解析 + 校验)

```go
// internal/nodes/detect/find_color_signature_test.go
package detect

import "testing"

func TestFindColorSignature_ParseValid(t *testing.T) {
	raw := []any{
		map[string]any{"dx": 0.0, "dy": 0.0, "r": 200.0, "g": 30.0, "b": 30.0},
		map[string]any{"dx": 2.0, "dy": 0.0, "r": 255.0, "g": 255.0, "b": 255.0, "tol": 8.0},
	}
	sig, err := parseColorSignature(raw)
	if err != nil || len(sig.Points) != 2 || sig.Points[1].Tol == nil || *sig.Points[1].Tol != 8 {
		t.Fatalf("parse: %v %+v", err, sig)
	}
	if sig.Points[0].Tol != nil {
		t.Fatal("anchor tol should be nil (default)")
	}
}

func TestFindColorSignature_ValidateAnchorNonZero(t *testing.T) {
	raw := []any{map[string]any{"dx": 1.0, "dy": 0.0, "r": 1.0, "g": 1.0, "b": 1.0}}
	if _, err := parseColorSignature(raw); err == nil {
		t.Fatal("anchor dx≠0 should error")
	}
}

func TestFindColorSignature_ValidatePointCount(t *testing.T) {
	var raw []any
	for i := 0; i < 65; i++ {
		raw = append(raw, map[string]any{"dx": 0.0, "dy": 0.0, "r": 1.0, "g": 1.0, "b": 1.0})
	}
	raw[0] = map[string]any{"dx": 0.0, "dy": 0.0, "r": 1.0, "g": 1.0, "b": 1.0}
	if _, err := parseColorSignature(raw); err == nil {
		t.Fatal("65 points should error")
	}
}
```

- [ ] **Step 2: 运行确认失败**

Run: `go test ./internal/nodes/detect/ -run TestFindColorSignature -v`
Expected: FAIL `undefined: parseColorSignature`。

- [ ] **Step 3: 写节点实现** (Spec/Run/Validate/parse; 照 `detect_color.go` 形态)

```go
// internal/nodes/detect/find_color_signature.go
// FindColorSignature — 颜色签名: 锚点 + N 偏移点的颜色组合, 区域搜索返回首个锚点位置。spec §节点1。
package detect

import (
	"encoding/json"
	"fmt"

	"yotta/internal/node"
)

func init() { node.Register(&FindColorSignature{}) }

type FindColorSignature struct{}

const (
	fcsInExec      = "In"
	fcsInROI       = "ROI"
	fcsInSignature = "Signature"
	fcsInTolerance = "Tolerance"
	fcsOutFound    = "Found"
	fcsOutNotFound = "NotFound"
	fcsDataPoint   = "Point"
	fcsMaxPoints   = 64
)

func (FindColorSignature) Spec() node.Spec {
	return node.Spec{
		Kind:        "FindColorSignature",
		Category:    "Detect",
		NeedsWindow: true,
		Inputs: []node.InputSpec{
			{Name: fcsInExec, Type: "Exec"},
			{Name: fcsInROI, Type: "Geometry", Schema: node.GeometrySchema()},
			{Name: fcsInSignature, Type: "JSON",
				Widget: node.WidgetSpec{Kind: "json", Props: node.MarshalProps(node.JSONProps{Rows: 4})}},
			{Name: fcsInTolerance, Type: "Number", Default: json.Number("16"),
				Widget: node.WidgetSpec{Kind: "number"}},
		},
		Outputs: []node.OutputSpec{
			{Name: fcsOutFound, Type: "Exec", Data: []node.DataField{{Name: fcsDataPoint, Type: "Point"}}},
			{Name: fcsOutNotFound, Type: "Exec"},
		},
	}
}

func (FindColorSignature) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	sig, err := parseColorSignature(in.Raw(fcsInSignature))
	if err != nil {
		return nil, fmt.Errorf("FindColorSignature signature: %w", err)
	}
	tol := in.Int(fcsInTolerance)
	if tol < 0 {
		tol = 0
	} else if tol > 255 {
		tol = 255
	}
	found, pt, err := ctx.Vision().FindColorSignature(in.Geometry(fcsInROI), sig, tol)
	if err != nil {
		return nil, node.Failf(node.CodeCaptureFailed, err, "FindColorSignature: %v", err)
	}
	if found {
		// 捕获由 Spec C config.capture 自动处理 (applyCaptures), 节点只 Set Data 字段, 无 node.Capture。
		return ctx.Out(fcsOutFound).Set(fcsDataPoint, pt).Fire(), nil
	}
	return ctx.Out(fcsOutNotFound).Fire(), nil
}

func (FindColorSignature) Validate(in node.Inputs) []node.ValidationError {
	if _, err := parseColorSignature(in.Raw(fcsInSignature)); err != nil {
		return []node.ValidationError{{Code: "INVALID_COLOR_SIGNATURE", Message: err.Error(), Field: fcsInSignature}}
	}
	return nil
}

// parseColorSignature 解析 Raw (JSON 数组) → ColorSignature。
// 校验: 非空、≤64 点、首项锚点 dx=dy=0、tol≥0。tol 缺省→nil (用默认), 给值→*int。
func parseColorSignature(raw any) (node.ColorSignature, error) {
	arr, ok := raw.([]any)
	if !ok || len(arr) == 0 {
		return node.ColorSignature{}, fmt.Errorf("signature must be non-empty array")
	}
	if len(arr) > fcsMaxPoints {
		return node.ColorSignature{}, fmt.Errorf("signature > %d points", fcsMaxPoints)
	}
	pts := make([]node.ColorPoint, len(arr))
	for i, e := range arr {
		m, ok := e.(map[string]any)
		if !ok {
			return node.ColorSignature{}, fmt.Errorf("point[%d] not object", i)
		}
		p := node.ColorPoint{
			DX: jnInt(m["dx"]), DY: jnInt(m["dy"]),
			R: jnInt(m["r"]), G: jnInt(m["g"]), B: jnInt(m["b"]),
		}
		if i == 0 && (p.DX != 0 || p.DY != 0) {
			return node.ColorSignature{}, fmt.Errorf("anchor (point[0]) must have dx=dy=0")
		}
		if tv, has := m["tol"]; has && tv != nil {
			t := jnInt(tv)
			if t < 0 {
				return node.ColorSignature{}, fmt.Errorf("point[%d] tol < 0", i)
			}
			p.Tol = &t
		}
		pts[i] = p
	}
	return node.ColorSignature{Points: pts}, nil
}

// jnInt 把 JSON 数字 (float64 / json.Number / int) 转 int, 非数字→0。
func jnInt(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case json.Number:
		if i, err := n.Int64(); err == nil {
			return int(i)
		}
	}
	return 0
}
```

- [ ] **Step 4: 运行确认通过**

Run: `go test ./internal/nodes/detect/ -run TestFindColorSignature -v`
Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add internal/nodes/detect/find_color_signature.go internal/nodes/detect/find_color_signature_test.go
git commit -m "feat(detect): FindColorSignature node"
```

### Task 1.5: i18n + 验证全绿

**Files:** Modify `frontend/src/i18n/zh.ts` · `frontend/src/i18n/en.ts`

- [ ] **Step 1: 加文案** (zh/en 对称, 照 `add-node.md §3` + `node-spec-style §10`)

`zh.ts` `node` 块加:
```ts
FindColorSignature: {
  label: '颜色签名',
  description: '在区域内搜索"锚点色+若干偏移点色"的组合，命中返回锚点位置',
  input: {
    ROI: { label: '搜索区域' },
    Signature: { label: '颜色签名' },
    Tolerance: { label: '默认容差' },
  },
},
```
`en.ts` 对称 (英文 sentence-case)。

- [ ] **Step 2: 重生成 catalog**

Run: `cd frontend && pnpm gen:node-i18n`
Expected: 更新 `internal/catalog/node-i18n.json`。

- [ ] **Step 3: 全套验证**

Run: `go build ./... && go test ./internal/nodes/... ./internal/node/... ./pkg/vision/... ./internal/catalog/... -count=1`
Expected: PASS (已知预存失败按 [build.md](../checklists/build.md) 判)。
Run: `cd frontend && pnpm typecheck && pnpm i18n:check`
Expected: PASS。

- [ ] **Step 4: Commit**

```bash
git add frontend/src/i18n/zh.ts frontend/src/i18n/en.ts internal/catalog/node-i18n.json
git commit -m "feat(detect): FindColorSignature i18n"
```

### Task 1.6: runtime 集成测试

**Files:** Create `internal/services/container/runtime/find_color_signature_test.go`

- [ ] **Step 1: 写集成测试** (合成全红帧 + 单锚点签名 → Found; 照 `TestDetectColorHSVTimeoutOnNoMatch` execNode 模式)

```go
// 100x100 全红帧, 单锚点签名 (200,0,0 tol 30) → Found 出口, Point 非零。
// 参照 detect_hsv_test.go 的 execNode 构造法 (合成帧注入 + 跑节点 + 断言出口)。
```

- [ ] **Step 2-4:** 运行 (`go test ./internal/services/container/runtime/ -run FindColorSignature -v` → PASS) → commit。

---

## Phase 2 — DecodeQR

### Task 2.1: 依赖 gate (gozxing 纯 Go 验证)

**Files:** Modify `go.mod` · `go.sum`

- [ ] **Step 1: 引库 + 验传递依赖无 CGO**

```bash
go get github.com/makiuchi-d/gozxing
go mod why -m github.com/makiuchi-d/gozxing
go mod download && go list -deps github.com/makiuchi-d/gozxing | xargs go list -f '{{.ImportPath}} {{.Standard}}'  # 确认无 CGO/原生
```
Expected: gozxing 传递依赖全纯 Go (无 `C` import)。**若发现 CGO → 回退** `go get github.com/liyue201/goqr` (纯 Go 仅 QR), 本 Phase 改用 goqr (Points 可能为空, 见 spec)。

- [ ] **Step 2: 验许可证** (含传递依赖): gozxing = Apache-2.0, 兼容 → 记入 commit message。

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "build: add gozxing (pure-Go QR, Apache-2.0, no CGO transitive)"
```

### Task 2.2: `QRResult` 类型 + `VisionService.DecodeQR` + 桩 + adapter

**Files:** Modify `internal/node/color_signature.go` (加 `QRResult`) · `interfaces.go` · `services.go` · `pkg/vision/qr.go` (建) · `node_services.go`

- [ ] **Step 1: 类型 + 接口 + 桩**

`internal/node/color_signature.go` 加:
```go
// QRResult 一个解码成功的二维码。Points = 解码器定位点 (点数/顺序以库为准), 全帧归一化。
type QRResult struct {
	Text   string  `json:"text"`
	Points []Point `json:"points"`
}
```
`interfaces.go` `VisionService` 加:
```go
	// DecodeQR 裁 roi 后纯 Go 解码全部 QR, 返回成功解码的结果。spec §节点3。
	DecodeQR(roi Geometry) ([]QRResult, error)
```
`services.go` 桩: `func (stubVisionService) DecodeQR(_ Geometry) ([]QRResult, error) { return nil, nil }`

- [ ] **Step 2: 写失败测试 (pkg/vision)** `pkg/vision/qr_test.go`: 用 gozxing encoder 生成一张 QR PNG → `DecodeQRImage` 解出原文; 多 QR 排序 (左上优先 + 同 y 用 x 决胜)。

- [ ] **Step 3: 写 pkg/vision 封装** `pkg/vision/qr.go`:

```go
// pkg/vision/qr.go — gozxing QR 解码封装 + 多 QR 排序。返回基元 (不 import internal/node)。
package vision

import (
	"image"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
)

// QRHit 解码结果 (像素坐标定位点)。
type QRHit struct {
	Text   string
	Points [][2]int // 像素坐标 (定位点; 点数以库为准)
}

// DecodeQRImage 解码 sub (已裁好的 ROI 子图) 内的 QR。
// gozxing QRCodeReader 单次返回一个最佳; 多 QR 需 QRCodeMultiReader (impl 时按库 API 核实)。
// 返回按定位点外接 bbox 左上角 (min-x,min-y) 升序 (y 再 x) 排序。
func DecodeQRImage(sub image.Image) ([]QRHit, error) {
	bmp, err := gozxing.NewBinaryBitmapFromImage(sub)
	if err != nil {
		return nil, err
	}
	reader := qrcode.NewQRCodeReader()
	res, err := reader.Decode(bmp, nil)
	if err != nil {
		return nil, nil // NotFound/解码失败 → 空 (节点走 NotFound), 非致命
	}
	hit := QRHit{Text: res.GetText()}
	for _, p := range res.GetResultPoints() {
		hit.Points = append(hit.Points, [2]int{int(p.GetX()), int(p.GetY())})
	}
	return []QRHit{hit}, nil // 多 QR: 升级 QRCodeMultiReader 后排序, 见 Step 注
}
```
> impl 注: 多 QR 用 `gozxing/multi` 的 `GenericMultipleBarcodeReader` 包 `qrcode.NewQRCodeReader()`; 拿到多个 `Result`
> 后按 `min(point.X)/min(point.Y)` 排序。**先核实 gozxing 的 `GetResultPoints` 实际点数/顺序**, 别假设四角。

- [ ] **Step 4: adapter** `node_services.go` `visionAdapter.DecodeQR`: 抓帧 (`ActiveHWND`+`CaptureFrameCached`) → 裁 ROI 子图 (复用 `vision.CropROIRGBA(frame, rx,ry,rw,rh)`, 已存在) → `vision.DecodeQRImage` → 定位点像素 +ROI 偏移 → 全帧归一化 `QRResult`。

- [ ] **Step 5:** `go build ./...` → PASS → commit。

### Task 2.3: 节点 `DecodeQR`

**Files:** Create `internal/nodes/detect/decode_qr.go` + `_test.go`

- [ ] **Step 1: 写失败测试** (stub vision 返 2 个 QR → 验 `Text`=首个 / `Count`=2; 空 → NotFound)。

- [ ] **Step 2: 写实现** (Spec: In/ROI; Found Data `Text`/`Count`/`Points`; NotFound Data `Count`。**无捕获框** —— 产出靠 Data 字段 + config.capture 自动捕获, 见 spec §贯穿约束):

```go
// 关键 Run 骨架 (只 Set Data, 不 node.Capture):
res, err := ctx.Vision().DecodeQR(in.Geometry(dqInROI))
if err != nil { return nil, node.Failf(node.CodeCaptureFailed, err, "DecodeQR: %v", err) }
if len(res) == 0 {
	return ctx.Out(dqOutNotFound).Set(dqDataCount, 0).Fire(), nil
}
first := res[0] // adapter 已按左上排序
return ctx.Out(dqOutFound).Set(dqDataText, first.Text).Set(dqDataCount, len(res)).Set(dqDataPoints, first.Points).Fire(), nil
```

- [ ] **Step 3-5:** 运行 PASS → commit。

### Task 2.4: i18n + 验证 + 集成测试

- [ ] zh/en 加 `node.DecodeQR` (label 二维码解码 / Text 内容→变量 等) → `pnpm gen:node-i18n`。
- [ ] `go build ./... && go test ./... -count=1` + 前端 `typecheck`/`i18n:check` → PASS。
- [ ] runtime 集成: 合成含 QR 的帧 → Found + Text 对。
- [ ] Commit。

---

## Phase 3 — FindTemplateAll

### Task 3.1: `pkg/vision.MatchAll` (3×3 极大 + NMS)

**Files:** Create `pkg/vision/match_all.go` + `_test.go`

- [ ] **Step 1: 写失败测试**: 合成灰度图平铺同模板 3 份 (间距 > 模板尺寸) + 噪声 → `MatchAll` 返 3 个、conf 降序、NMS 不重; 单目标对齐 `Match`; 低阈值造 >4096 候选验截断不 panic。

- [ ] **Step 2: 写实现** `pkg/vision/match_all.go`:

```go
// pkg/vision/match_all.go — Match 的"全部命中"变体: 3×3 局部极大 ≥threshold + NMS。spec §节点2 底层改动。
package vision

import (
	"math"
	"sort"
)

// MatchHit 一个命中 (ROI 内左上角像素 + conf)。
type MatchHit struct {
	X, Y int
	Conf float32
}

const matchAllCandidateCap = 4096

// MatchAll: 收集 conf≥threshold 且为 3×3 局部极大的候选, 做 NMS (minDist 像素; <=0 → 模板 min(W,H)/2)。
// 返回按 conf 降序 (并列 y,x) 的命中 (ROI 内坐标)。
func MatchAll(img []float32, iw, ih int, tpl *Template, parallel int, threshold float32, minDist int) []MatchHit {
	confMap, sw, sh := correlationMap(img, iw, ih, tpl, parallel) // 复用 Match 的 NCC 计算, 返回完整 conf 图
	if sw <= 0 || sh <= 0 {
		return nil
	}
	var cand []MatchHit
	for sy := 0; sy < sh; sy++ {
		for sx := 0; sx < sw; sx++ {
			c := confMap[sy*sw+sx]
			if c < threshold || !isLocalMax(confMap, sw, sh, sx, sy, c) {
				continue
			}
			cand = append(cand, MatchHit{X: sx, Y: sy, Conf: c})
		}
	}
	sort.Slice(cand, func(i, j int) bool {
		if cand[i].Conf != cand[j].Conf {
			return cand[i].Conf > cand[j].Conf
		}
		if cand[i].Y != cand[j].Y {
			return cand[i].Y < cand[j].Y
		}
		return cand[i].X < cand[j].X
	})
	if len(cand) > matchAllCandidateCap {
		// log 截断告知 (调用方 adapter 记 log); 这里仅裁剪。
		cand = cand[:matchAllCandidateCap]
	}
	if minDist <= 0 {
		minDist = minInt(tpl.W, tpl.H) / 2
	}
	return nms(cand, tpl.W, tpl.H, minDist)
}

// isLocalMax: c ≥ 全部存在的 8 邻域 (边缘缩减邻域); plateau (相等) 仅在 (y,x) 最小处算极大。
func isLocalMax(m []float32, sw, sh, x, y int, c float32) bool {
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			nx, ny := x+dx, y+dy
			if nx < 0 || nx >= sw || ny < 0 || ny >= sh {
				continue
			}
			nc := m[ny*sw+nx]
			if nc > c {
				return false
			}
			if nc == c && (ny < y || (ny == y && nx < x)) { // plateau: 让位给更靠前的
				return false
			}
		}
	}
	return true
}

// nms: 按 conf 降序 (cand 已排序) 贪心保留, dist(中心) < minDist 抑制。
func nms(cand []MatchHit, tw, th, minDist int) []MatchHit {
	var kept []MatchHit
	for _, c := range cand {
		cxC, cyC := float64(c.X+tw/2), float64(c.Y+th/2)
		ok := true
		for _, k := range kept {
			cxK, cyK := float64(k.X+tw/2), float64(k.Y+th/2)
			if math.Hypot(cxC-cxK, cyC-cyK) < float64(minDist) {
				ok = false
				break
			}
		}
		if ok {
			kept = append(kept, c)
		}
	}
	return kept
}

func minInt(a, b int) int { if a < b { return a }; return b }
```
> **实现关键**: `correlationMap` = 把现有 `Match` (template.go) 内层 NCC 计算抽出成"返回完整 conf 图"的版本
> (`Match` 现在丢弃非极大值; `MatchAll` 要全图)。重构 `Match` 调 `correlationMap` 取 argmax, 保证两者算法一致、
> 不重复维护 NCC。先写 `correlationMap` + 让 `Match` 复用它 (回归测试 `template_test.go` 必须仍绿)。

- [ ] **Step 3-5:** 运行 PASS (含 `go test ./pkg/vision/ -run 'TestMatch' ` 确认 `Match` 重构无回归) → commit。

### Task 3.2: `TemplateMatch` 类型 + `TemplateMatcher.DetectAll` (接口 + 4 实现)

**Files:** Modify `internal/node/color_signature.go` (加 `TemplateMatch`) · `interfaces.go` (Vision) · `runtime/interfaces.go` (`TemplateMatcher`+`NoopMatcher`) · `wire_container.go` · `inspect_phase_test.go` · `node_services_test.go`

- [ ] **Step 1: 类型** `internal/node` 加:
```go
type TemplateMatch struct {
	Point       Point      `json:"point"`       // 实例中心, 全帧归一化
	Conf        float64    `json:"conf"`
	BBox        [4]float64 `json:"bbox"`        // [x,y(左上),w,h] 全帧归一化; Point=BBox 中心
	TemplateKey string     `json:"templateKey"`
}
```

- [ ] **Step 2: `TemplateMatcher` 接口加 `DetectAll`** (`runtime/interfaces.go`):
```go
	// DetectAll 单帧返回某模板所有命中 (3×3 极大 + 单模板内 NMS)。坐标全帧归一化。spec §节点2。
	DetectAll(ctx context.Context, frame *image.RGBA, guid string, th float64, region []float64, scaleTolerance float64) ([]node.TemplateMatch, error)
```
> 注: `runtime/interfaces.go` 已 import `internal/node`? 若否则用其已有的 `expr.Point` 风格 —— **核实该文件 import**,
> `TemplateMatch` 放能被双方引用处 (`internal/node` 已被 runtime import, 安全)。

- [ ] **Step 3: 4 实现同步加** (二号铁律, 不留旧签名):
  - `NoopMatcher.DetectAll` (`runtime/interfaces.go`) → `return nil, nil`
  - `mockMatcher.DetectAll` (`inspect_phase_test.go`) → 返测试固定切片
  - `stubMatcher.DetectAll` (`node_services_test.go`) → 返固定切片
  - **`templateMatcherAdapter.DetectAll`** (`wire_container.go`, 生产): 镜像 `Detect` (PickVariant → scaleTolerance 门 →
    ScaleTemplate → ROI 裁切), 但: ① 单 ROI 路径调 `vision.MatchAll` (非 `Match`), 每命中算中心+bbox 全帧归一化、填
    `TemplateKey=guid`; ② 多槽 `variant.Regions` 路径调 `detectMultiRegionAll` (新, 复制 `detectMultiRegion` 但每 region
    调 `MatchAll` + **收全部命中**、逐一换算, 不取末位单个)。

- [ ] **Step 4:** `go build ./...` + `go test ./internal/services/container/runtime/ -run 'Template|Match' -count=1` (模板族无回归) → PASS。
- [ ] **Step 5: Commit**。

### Task 3.3: `VisionService.MatchAll` + 桩 + adapter

**Files:** Modify `interfaces.go` · `services.go` · `node_services.go`

- [ ] **Step 1: 接口 + 桩**:
```go
	// MatchAll 单帧多模板全部命中: 各模板 DetectAll (同一帧) → 合并 → 统一 NMS。spec §节点2。
	MatchAll(ctx context.Context, keys []string, threshold float64, minDistance int) ([]TemplateMatch, error)
```
桩返 `nil, nil`。

- [ ] **Step 2: adapter** `visionAdapter.MatchAll`: 抓**一次**帧 (`ActiveHWND`+`CaptureFrameCached`) → 各 guid 调
  `a.rt.Matcher.DetectAll(ctx, frame, guid, threshold, nil, a.scaleTolerance())` → 合并 → **统一 NMS** (按 conf 降序,
  `dist < min(rA,rB)`; auto 半径用各命中 BBox 短边/2 — 全帧归一化下按 BBox 算; 显式 minDistance>0 用归一化/像素一致单位)。
  > 统一 NMS 用 `internal/node` 侧的小 helper (跨模板, 含 TemplateKey), 不复用 pkg/vision 的 (那是单模板 ROI 内坐标)。

- [ ] **Step 3-4:** `go build ./...` → PASS → commit。

### Task 3.4: 节点 `FindTemplateAll`

**Files:** Create `internal/nodes/detect/find_template_all.go` + `_test.go`

- [ ] **Step 1: 写失败测试** (stub vision 返 3 命中 → 验 `Count`=3 / `PrimaryPoint`=Matches[0] / `Matches` JSON / 捕获; MaxResults=2 → 列表 2 条但 Count=3; 空 → NotFound Count=0)。

- [ ] **Step 2: 写实现** (Spec: In/Templates(GUID 列表, `templateDeps`)/ROI/Threshold(默认 .85)/MaxResults(0)/MinDistance(0);
  Found Data `Matches`/`Count`/`PrimaryPoint`/`PrimaryConf`; NotFound Data `Count`。**无捕获框** —— Data 字段靠 config.capture 自动捕获):

```go
// 关键 Run 骨架 (只 Set Data, 不 node.Capture):
th := clamp01(in.Float(ftaInThreshold))         // [0,1]
maxR := in.Int(ftaInMaxResults); if maxR < 0 { maxR = 0 }
minD := in.Int(ftaInMinDistance); if minD < 0 { minD = 0 }
matches, err := ctx.Vision().MatchAll(ctx.Context(), guids, th, minD)
if err != nil { return nil, node.Failf(node.CodeCaptureFailed, err, "FindTemplateAll: %v", err) }
count := len(matches)
if count == 0 { return ctx.Out(ftaOutNotFound).Set(ftaDataCount, 0).Fire(), nil }
out := matches
if maxR > 0 && len(out) > maxR { out = out[:maxR] } // 列表截断; Count 报总数
primary := matches[0]
return ctx.Out(ftaOutFound).
	Set(ftaDataMatches, out).Set(ftaDataCount, count).
	Set(ftaDataPrimaryPoint, primary.Point).Set(ftaDataPrimaryConf, primary.Conf).Fire(), nil
```
> `Templates` GUID 列表字段 + 依赖: 照 `template_common.go` `templateDeps` + 模板族节点 (`check_template.go`) 的
> `Templates` 字段声明法 (读它照抄字段类型/widget)。`Deps()` 方法返回 `templateDeps(guids)`。

- [ ] **Step 3-5:** 运行 PASS → commit。

### Task 3.5: i18n + 全套验证 + 回归

- [ ] zh/en 加 `node.FindTemplateAll` → `pnpm gen:node-i18n`。
- [ ] **全套** (照 [add-node.md §8](../checklists/add-node.md)):
  `go build ./... && go test ./internal/nodes/... ./internal/node/... ./pkg/vision/... ./internal/catalog/... ./internal/services/container/... -count=1`
  + `cd frontend && pnpm typecheck && pnpm i18n:check` + `task build`。
  Expected: PASS (预存失败按 build.md 判)。**重点确认模板族 (CheckTemplate/ClickTemplate/WaitTemplate) 测试仍绿** (DetectAll 接口改动无回归)。
- [ ] Commit。

---

## 最终验收 (真机 smoke)

- [ ] `task dev` 起 app, 在**侧边面板 + 右键菜单 + explorer 弹窗**找到 3 个新节点 (颜色签名 / 二维码解码 / 模板全部命中)。
- [ ] `FindColorSignature`: 拖入 + 填一个签名 (屏幕拾色) → 跑 → Found/NotFound 正确。
- [ ] `DecodeQR`: 屏幕摆一张 QR → 跑 → Text 读对; 多 QR → 取最左上 + Count 对。
- [ ] `FindTemplateAll`: 同模板屏上铺 N 份 → Count=N, Matches 列表全、Primary 对; 收窄 ROI 验性能。
- [ ] 文案 / 分组标签 (Detect) / 默认值 / 渲染都对。

## 自检 (spec 覆盖)

- §节点1 FindColorSignature → Task 1.1–1.6 ✓ (类型/扫描/接口/节点/i18n/集成)
- §节点2 FindTemplateAll → Task 3.1–3.5 ✓ (MatchAll/接口4实现/Vision/节点/验证)
- §节点3 DecodeQR → Task 2.1–2.4 ✓ (gate/封装/节点/验证)
- 贯穿约束 (纯 Go / 一帧快照 / 全帧归一化 / 捕获框 / 错误校验分工 / 参数 clamp) → 各节点 Run/adapter 内落实 ✓
- 落地顺序 (风险递增) = Phase 1→2→3 ✓
