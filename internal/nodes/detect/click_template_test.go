// internal/nodes/detect/click_template_test.go
package detect

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"yotta/internal/node"
)

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
	vision := &mockVision{point: &pt, conf: 0.93, hitOnCall: 1}
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
	vision := &mockVision{point: &pt, conf: 0.93, hitOnCall: 1}
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
	vision := &mockVision{point: &pt, conf: 0.93, hitOnCall: 1, missAfterCall: 2}
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
	vision := &mockVision{point: &pt, conf: 0.9, hitOnCall: 1}
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
	vision := &mockVision{point: &pt, conf: 0.9, hitOnCall: 1}
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
