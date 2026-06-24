// internal/nodes/detect/click_template_test.go
package detect

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"yotta/internal/node"
)

// ─── Task 2.2: pickMatch 单元测试 ────────────────────────────────────────────

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

// recordingInput 实现 InputService — ClickTemplate 测试用. detect 包内仅 ClickTemplate
// 需要 InputService, 跟 input 包的同名 helper 同款但本地一份避免跨包依赖.
type recordingInput struct {
	calls []string
	err   error
}

func (r *recordingInput) KeyDown(vk string) error { return r.err }
func (r *recordingInput) KeyUp(vk string) error   { return r.err }
func (r *recordingInput) Click(xRatio, yRatio float64, button string, durationMs int) error {
	r.calls = append(r.calls, fmt.Sprintf("Click:%.3f:%.3f:%s:%d", xRatio, yRatio, button, durationMs))
	return r.err
}
func (r *recordingInput) MouseMoveRel(dx, dy, durationMs int) error             { return r.err }
func (r *recordingInput) MoveTo(xRatio, yRatio float64) error                   { return r.err }
func (r *recordingInput) CursorRatio() (float64, float64, error)                { return 0, 0, r.err }
func (r *recordingInput) Scroll(xRatio, yRatio float64, notches int) error      { return r.err }
func (r *recordingInput) MouseDown(xRatio, yRatio float64, button string) error { return r.err }
func (r *recordingInput) MouseUp(button string) error                           { return r.err }

func withVisionAndInput(v node.VisionService, in node.InputService) node.ServiceBundle {
	b := node.StubServices()
	b.Vision = v
	b.Input = in
	return b
}

func TestClickTemplate_Done(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&ClickTemplate{})
	rn, _ := node.Get("ClickTemplate")

	pt := node.Point{X: 0.55, Y: 0.4}
	// bbox center = pt: bbox=[pt.X, pt.Y, 0, 0] → anchorPoint center = (pt.X, pt.Y)
	vision := &mockVision{point: &pt, bbox: [4]float64{0.55, 0.4, 0, 0}, conf: 0.93, hitOnCall: 1}
	rec := &recordingInput{}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{clkInTemplates: []string{"fishing.start_fish"}, clkInButton: "left",
			clkInTimeoutMs: 200, clkInThreshold: 0.85},
		nil, withVisionAndInput(vision, rec), false)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != clkOutDone {
		t.Errorf("exit = %q, want Done", r.ExitName)
	}
	if r.OutputData[clkDataMatched] != true {
		t.Errorf("Matched = %v, want true", r.OutputData[clkDataMatched])
	}
	if len(rec.calls) != 1 || rec.calls[0] != "Click:0.550:0.400:left:50" {
		t.Errorf("calls = %v, want [Click:0.550:0.400:left:50]", rec.calls)
	}
}

func TestClickTemplate_SettleMs_RedetectThenClick(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&ClickTemplate{})
	rn, _ := node.Get("ClickTemplate")

	pt := node.Point{X: 0.55, Y: 0.4}
	vision := &mockVision{point: &pt, bbox: [4]float64{0.55, 0.4, 0, 0}, conf: 0.93, hitOnCall: 1}
	rec := &recordingInput{}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{clkInTemplates: []string{"fishing.start_fish"}, clkInButton: "left",
			clkInTimeoutMs: 200, clkInThreshold: 0.85, clkInSettleMs: 5},
		nil, withVisionAndInput(vision, rec), false)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != clkOutDone {
		t.Errorf("exit = %q, want Done", r.ExitName)
	}
	// SettleMs>0 → 命中后等一下再 re-detect 一次定位 → WaitMatch 共调 2 次 (初次 + 重定位)。
	if vision.callCount != 2 {
		t.Errorf("WaitMatch callCount = %d, want 2 (initial + re-detect)", vision.callCount)
	}
	// 仍只点 1 次, 落在 (重定位后的) 命中点。
	if len(rec.calls) != 1 || rec.calls[0] != "Click:0.550:0.400:left:50" {
		t.Errorf("calls = %v, want one click at re-detected point", rec.calls)
	}
}

func TestClickTemplate_RetryUntilGone_Done(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&ClickTemplate{})
	rn, _ := node.Get("ClickTemplate")

	pt := node.Point{X: 0.55, Y: 0.4}
	// missAfterCall=2: call1 初次命中 + call2 点完重查仍在 → 点 2 下; call3 模板已消失 → 走 Done。
	vision := &mockVision{point: &pt, bbox: [4]float64{0.55, 0.4, 0, 0}, conf: 0.93, hitOnCall: 1, missAfterCall: 2}
	rec := &recordingInput{}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{clkInTemplates: []string{"fishing.start_fish"}, clkInButton: "left",
			clkInTimeoutMs: 200, clkInThreshold: 0.85, clkInMaxAttempts: 5, clkInRetryIntervalMs: 1},
		nil, withVisionAndInput(vision, rec), false)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != clkOutDone {
		t.Errorf("exit = %q, want Done", r.ExitName)
	}
	if r.OutputData[clkDataMatched] != true {
		t.Errorf("Matched = %v, want true", r.OutputData[clkDataMatched])
	}
	// 第一下没点掉 → 重试再点一下 → 模板消失。共 2 次 click, 没用满 MaxAttempts。
	if len(rec.calls) != 2 {
		t.Errorf("clicks = %d, want 2 (first + one retry, then template gone)", len(rec.calls))
	}
}

func TestClickTemplate_RetryExhausted_Timeout(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&ClickTemplate{})
	rn, _ := node.Get("ClickTemplate")

	pt := node.Point{X: 0.5, Y: 0.5}
	// missAfterCall=0 (禁用) → 模板一直在 → 点满 MaxAttempts 仍没消失 → Timeout(Matched=true)。
	vision := &mockVision{point: &pt, bbox: [4]float64{0.5, 0.5, 0, 0}, conf: 0.9, hitOnCall: 1}
	rec := &recordingInput{}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{clkInTemplates: []string{"fishing.start_fish"}, clkInButton: "left",
			clkInTimeoutMs: 200, clkInThreshold: 0.85, clkInMaxAttempts: 3, clkInRetryIntervalMs: 1},
		nil, withVisionAndInput(vision, rec), false)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != clkOutTimeout {
		t.Errorf("exit = %q, want Timeout", r.ExitName)
	}
	// 出现了但点不掉 → Matched=true (跟"压根没出现"的 Matched=false 区分)。
	if r.OutputData[clkDataMatched] != true {
		t.Errorf("Matched = %v, want true (appeared but never clicked away)", r.OutputData[clkDataMatched])
	}
	if len(rec.calls) != 3 {
		t.Errorf("clicks = %d, want 3 (= MaxAttempts)", len(rec.calls))
	}
}

func TestClickTemplate_Timeout_NoClick(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&ClickTemplate{})
	rn, _ := node.Get("ClickTemplate")

	vision := &mockVision{hitOnCall: -1, conf: 0.3}
	rec := &recordingInput{}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{clkInTemplates: []string{"fishing.start_fish"}, clkInTimeoutMs: 30},
		nil, withVisionAndInput(vision, rec), false)

	if r.ExitName != clkOutTimeout {
		t.Errorf("exit = %q, want Timeout", r.ExitName)
	}
	if r.OutputData[clkDataMatched] != false {
		t.Errorf("Matched = %v, want false", r.OutputData[clkDataMatched])
	}
	if len(rec.calls) != 0 {
		t.Errorf("Timeout 路径不该 click, got %v", rec.calls)
	}
}

func TestClickTemplate_BackendError(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&ClickTemplate{})
	rn, _ := node.Get("ClickTemplate")

	pt := node.Point{X: 0.5, Y: 0.5}
	vision := &mockVision{point: &pt, bbox: [4]float64{0.5, 0.5, 0, 0}, conf: 0.9, hitOnCall: 1}
	rec := &recordingInput{err: errors.New("hwnd closed")}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{clkInTemplates: []string{"fishing.start_fish"}},
		nil, withVisionAndInput(vision, rec), false)

	if r.Error == nil {
		t.Error("expected Click backend error propagation")
	}
}

func TestClickTemplate_InvalidButton_ValidationError(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&ClickTemplate{})
	rn, _ := node.Get("ClickTemplate")

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{clkInTemplates: []string{"fishing.start_fish"}, clkInButton: "side1"},
		nil, withVisionAndInput(&mockVision{}, &recordingInput{}), false)

	if len(r.Validation) == 0 {
		t.Fatal("expected validation error")
	}
	found := false
	for _, e := range r.Validation {
		if e.Code == "INVALID_MOUSE_BUTTON" {
			found = true
		}
	}
	if !found {
		t.Errorf("validation = %v, want INVALID_MOUSE_BUTTON", r.Validation)
	}
}

// ─── Task 2.2: order_by 端到端 + 默认零回归 ─────────────────────────────────

// TestClickTemplate_OrderBy_Vertical: OrderBy=vertical, Index=0 → 点最上面那个 (BBox.y 最小)。
// 传三个命中, vertical 排序后最上是 y=0.1 的那个 (Point.X=0.8, Point.Y=0.1)。
func TestClickTemplate_OrderBy_Vertical(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&ClickTemplate{})
	rn, _ := node.Get("ClickTemplate")

	// MatchAll 返回三个命中 (conf 降序, 与 MatchAll 约定一致)
	matches := []node.TemplateMatch{
		{Point: node.Point{X: 0.2, Y: 0.9}, Conf: 0.99, BBox: [4]float64{0.2, 0.9, 0.10, 0.10}},
		{Point: node.Point{X: 0.8, Y: 0.1}, Conf: 0.90, BBox: [4]float64{0.8, 0.1, 0.05, 0.05}},
		{Point: node.Point{X: 0.5, Y: 0.5}, Conf: 0.85, BBox: [4]float64{0.5, 0.5, 0.08, 0.08}},
	}
	vision := &mockVision{matchAllResults: matches}
	rec := &recordingInput{}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{
			clkInTemplates: []string{"tpl.x"},
			clkInTimeoutMs: 200,
			clkInThreshold: 0.80,
			clkInOrderBy:   "vertical",
			clkInIndex:     0,
		},
		nil, withVisionAndInput(vision, rec), false)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != clkOutDone {
		t.Fatalf("exit=%q want Done", r.ExitName)
	}
	// 最上面那个 BBox=[0.8,0.1,0.05,0.05] → center = anchorPoint = (0.825, 0.125)
	if len(rec.calls) != 1 {
		t.Fatalf("want 1 click, got %v", rec.calls)
	}
	want := fmt.Sprintf("Click:%.3f:%.3f:left:50", 0.825, 0.125)
	if rec.calls[0] != want {
		t.Errorf("click=%q want %q", rec.calls[0], want)
	}
}

// TestClickTemplate_DefaultScore_Regression: 默认 OrderBy=score/Index=0 走 WaitMatch 路径,
// 行为与 Phase 1 完全一致 (钉死零回归)。
func TestClickTemplate_DefaultScore_Regression(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&ClickTemplate{})
	rn, _ := node.Get("ClickTemplate")

	pt := node.Point{X: 0.55, Y: 0.4}
	vision := &mockVision{point: &pt, bbox: [4]float64{0.55, 0.4, 0, 0}, conf: 0.93, hitOnCall: 1}
	rec := &recordingInput{}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{
			clkInTemplates: []string{"fishing.start_fish"},
			clkInButton:    "left",
			clkInTimeoutMs: 200,
			clkInThreshold: 0.85,
			// OrderBy/Index 不传 → 用默认值 score/0 → 走 WaitMatch 路径
		},
		nil, withVisionAndInput(vision, rec), false)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != clkOutDone {
		t.Errorf("exit=%q want Done", r.ExitName)
	}
	if r.OutputData[clkDataMatched] != true {
		t.Errorf("Matched=%v want true", r.OutputData[clkDataMatched])
	}
	// WaitMatch 路径: MatchAll 不应被调用 (matchAllResults=nil, 走 WaitMatch)
	if len(rec.calls) != 1 || rec.calls[0] != "Click:0.550:0.400:left:50" {
		t.Errorf("calls=%v want [Click:0.550:0.400:left:50]", rec.calls)
	}
}

// ─── Task 2.3: anchorPoint 单元测试 ──────────────────────────────────────────

// approxEq 浮点近似相等 (容差 1e-9).
func approxEq(a, b float64) bool {
	d := a - b
	return d >= -1e-9 && d <= 1e-9
}

func TestAnchorPoint(t *testing.T) {
	bb := [4]float64{0.2, 0.4, 0.10, 0.20} // x,y,w,h → 中心 (0.25,0.5)
	if p := anchorPoint(bb, "center", 0, 0); !approxEq(p.X, 0.25) || !approxEq(p.Y, 0.5) {
		t.Fatalf("center=%v want (0.25,0.5)", p)
	}
	if p := anchorPoint(bb, "topLeft", 0, 0); !approxEq(p.X, 0.2) || !approxEq(p.Y, 0.4) {
		t.Fatalf("topLeft=%v want (0.2,0.4)", p)
	}
	if p := anchorPoint(bb, "botRight", 0, 0); !approxEq(p.X, 0.3) || !approxEq(p.Y, 0.6) {
		t.Fatalf("botRight=%v want (0.3,0.6)", p)
	}
	// 偏移 (已是 ratio): topRight + (0.05,-0.05)
	if p := anchorPoint(bb, "topRight", 0.05, -0.05); !approxEq(p.X, 0.35) || !approxEq(p.Y, 0.35) {
		t.Fatalf("topRight+off=%v want (0.35,0.35)", p)
	}
	// clamp: 越界裁到 [0,1]
	if p := anchorPoint(bb, "botRight", 1.0, 0); p.X != 1 {
		t.Fatalf("clamp X=%v want 1", p.X)
	}
}

// withVisionInputAndWindow 同时注入 Vision + Input + Window (Task 2.3 像素偏移测试用)。
func withVisionInputAndWindow(v node.VisionService, in node.InputService, w node.WindowService) node.ServiceBundle {
	b := node.StubServices()
	b.Vision = v
	b.Input = in
	b.Window = w
	return b
}

// TestClickTemplate_Anchor_TopRight: Anchor=topRight/Offset=0 → 点命中框右上角。
func TestClickTemplate_Anchor_TopRight(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&ClickTemplate{})
	rn, _ := node.Get("ClickTemplate")

	// BBox=[0.2,0.4,0.10,0.20] → topRight = (0.2+0.10, 0.4) = (0.30, 0.40)
	pt := node.Point{X: 0.25, Y: 0.5} // center (被 anchor 覆盖)
	bb := [4]float64{0.2, 0.4, 0.10, 0.20}
	vision := &mockVision{point: &pt, bbox: bb, conf: 0.93, hitOnCall: 1}
	rec := &recordingInput{}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{
			clkInTemplates: []string{"tpl.anchor"},
			clkInTimeoutMs: 200,
			clkInThreshold: 0.85,
			clkInAnchor:    "topRight",
			clkInOffsetX:   0.0,
			clkInOffsetY:   0.0,
		},
		nil, withVisionAndInput(vision, rec), false)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != clkOutDone {
		t.Fatalf("exit=%q want Done", r.ExitName)
	}
	// topRight = bbox[0]+bbox[2] , bbox[1] = (0.30, 0.40)
	want := fmt.Sprintf("Click:%.3f:%.3f:left:50", 0.30, 0.40)
	if len(rec.calls) != 1 || rec.calls[0] != want {
		t.Errorf("click=%q want %q", rec.calls, want)
	}
}

// TestClickTemplate_Anchor_Center_Regression: 默认 center/0/0 → 点命中框中心 = hit.Point (零回归)。
// mock window 不需要 ClientSize (offset=0 不触发像素换算)。
func TestClickTemplate_Anchor_Center_Regression(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&ClickTemplate{})
	rn, _ := node.Get("ClickTemplate")

	// BBox=[0.1, 0.3, 0.30, 0.40] → center = (0.1+0.15, 0.3+0.20) = (0.25, 0.50)
	// hit.Point 也设成中心以证明两者一致
	pt := node.Point{X: 0.25, Y: 0.50}
	bb := [4]float64{0.1, 0.3, 0.30, 0.40}
	vision := &mockVision{point: &pt, bbox: bb, conf: 0.90, hitOnCall: 1}
	rec := &recordingInput{}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{
			clkInTemplates: []string{"tpl.regression"},
			clkInTimeoutMs: 200,
			clkInThreshold: 0.85,
			// Anchor/OffsetX/OffsetY 不传 → 默认 center/0/0
		},
		nil, withVisionAndInput(vision, rec), false)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != clkOutDone {
		t.Fatalf("exit=%q want Done", r.ExitName)
	}
	// center of BBox = (0.25, 0.50) == hit.Point → 零回归
	want := fmt.Sprintf("Click:%.3f:%.3f:left:50", 0.25, 0.50)
	if len(rec.calls) != 1 || rec.calls[0] != want {
		t.Errorf("click=%q want %q", rec.calls, want)
	}
}

// TestClickTemplate_OffsetPx_UsesClientSize: OffsetX=96(像素) 触发 ClientSize 换算 → 96/1920=0.05。
func TestClickTemplate_OffsetPx_UsesClientSize(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&ClickTemplate{})
	rn, _ := node.Get("ClickTemplate")

	// BBox=[0.2,0.4,0.10,0.20] → topLeft=(0.2,0.4); 加 OffsetX=96px→0.05, OffsetY=0
	// 期望落点: (0.2+0.05, 0.4) = (0.25, 0.40)
	pt := node.Point{X: 0.25, Y: 0.50}
	bb := [4]float64{0.2, 0.4, 0.10, 0.20}
	vision := &mockVision{point: &pt, bbox: bb, conf: 0.92, hitOnCall: 1}
	rec := &recordingInput{}
	win := stubWindow{w: 1920, h: 1080}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{
			clkInTemplates: []string{"tpl.offset"},
			clkInTimeoutMs: 200,
			clkInThreshold: 0.85,
			clkInAnchor:    "topLeft",
			clkInOffsetX:   96.0, // 像素 > 1
			clkInOffsetY:   0.0,
		},
		nil, withVisionInputAndWindow(vision, rec, win), false)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != clkOutDone {
		t.Fatalf("exit=%q want Done", r.ExitName)
	}
	// topLeft + (96/1920, 0) = (0.2+0.05, 0.4+0) = (0.25, 0.40)
	want := fmt.Sprintf("Click:%.3f:%.3f:left:50", 0.25, 0.40)
	if len(rec.calls) != 1 || rec.calls[0] != want {
		t.Errorf("click=%q want %q", rec.calls, want)
	}
}
