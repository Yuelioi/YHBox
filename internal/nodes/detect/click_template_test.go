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

func (r *recordingInput) KeyDown(vk string) error                  { return r.err }
func (r *recordingInput) KeyUp(vk string) error                    { return r.err }
func (r *recordingInput) Click(xRatio, yRatio float64, button string, durationMs int) error {
	r.calls = append(r.calls, fmt.Sprintf("Click:%.3f:%.3f:%s:%d", xRatio, yRatio, button, durationMs))
	return r.err
}
func (r *recordingInput) MouseMoveRel(dx, dy, durationMs int) error      { return r.err }
func (r *recordingInput) MoveTo(xRatio, yRatio float64) error            { return r.err }
func (r *recordingInput) CursorRatio() (float64, float64, error)         { return 0, 0, r.err }
func (r *recordingInput) Scroll(xRatio, yRatio float64, notches int) error { return r.err }
func (r *recordingInput) MouseDown(xRatio, yRatio float64, button string) error { return r.err }
func (r *recordingInput) MouseUp(button string) error                          { return r.err }

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
	if len(rec.calls) != 1 || rec.calls[0] != "Click:0.550:0.400:left:50" {
		t.Errorf("calls = %v, want [Click:0.550:0.400:left:50]", rec.calls)
	}
}

func TestClickTemplate_Capture_Done(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&ClickTemplate{})
	rn, _ := node.Get("ClickTemplate")

	pt := node.Point{X: 0.55, Y: 0.4}
	vision := &mockVision{point: &pt, conf: 0.93, hitOnCall: 1}
	rec := &recordingInput{}
	vars := newRecVars()
	b := node.StubServices()
	b.Vision = vision
	b.Input = rec
	b.Vars = vars
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{clkInTemplates: []string{"fishing.start_fish"}, clkInButton: "left",
			clkInTimeoutMs: 200, clkInThreshold: 0.85, clkCapFound: "f", clkCapPoint: "p"},
		nil, b, false)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != clkOutDone {
		t.Fatalf("exit = %q, want Done", r.ExitName)
	}
	if got, ok := vars.Get("f"); !ok || got != true {
		t.Errorf("capture f = %v (ok=%v), want true", got, ok)
	} else if _, isBool := got.(bool); !isBool {
		// CaptureType=bool: 断言写入值的 Go 类型是 bool.
		t.Errorf("capture f Go type = %T, want bool", got)
	}
	gp, ok := vars.Get("p")
	if !ok || gp != pt {
		t.Fatalf("capture p = %v (ok=%v), want %v", gp, ok, pt)
	}
	// CaptureType=point: 断言写入值的 Go 类型是 node.Point.
	if _, isPoint := gp.(node.Point); !isPoint {
		t.Errorf("capture p Go type = %T, want node.Point", gp)
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
