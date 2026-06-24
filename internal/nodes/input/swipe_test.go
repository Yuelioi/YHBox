package input

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"yotta/internal/node"
)

// Drag 记录格式: "Drag:x1:y1:x2:y2:button:durationMs"
func (r *recordingInput) Drag(x1, y1, x2, y2 float64, button string, durationMs int) error {
	r.calls = append(r.calls, fmt.Sprintf("Drag:%.3f:%.3f:%.3f:%.3f:%s:%d", x1, y1, x2, y2, button, durationMs))
	return r.err
}

func TestSwipe_HappyPath(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Swipe{})
	rn, _ := node.Get("Swipe")

	rec := &recordingInput{}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{swInBegin: node.Point{X: 0.1, Y: 0.2}, swInEnd: node.Point{X: 0.8, Y: 0.9},
			swInButton: "left", swInDurationMs: 300},
		nil, withInput(rec), false)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != swOutDone {
		t.Errorf("exit = %q, want Done", r.ExitName)
	}
	if len(rec.calls) != 1 || rec.calls[0] != "Drag:0.100:0.200:0.800:0.900:left:300" {
		t.Errorf("calls = %v, want [Drag:0.100:0.200:0.800:0.900:left:300]", rec.calls)
	}
}

func TestSwipe_DefaultsApplied(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Swipe{})
	rn, _ := node.Get("Swipe")

	rec := &recordingInput{}
	// 不传任何 config — Point pin 无 Default，Begin/End → zero Point{} = (0,0); button=left, durationMs=200
	r := node.RunNode(context.Background(), rn, nil, nil, nil, withInput(rec), false)
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	// Begin/End 均为 zero Point (0,0), button=left, durationMs=200
	if len(rec.calls) != 1 || rec.calls[0] != "Drag:0.000:0.000:0.000:0.000:left:200" {
		t.Errorf("calls = %v, want [Drag:0.000:0.000:0.000:0.000:left:200]", rec.calls)
	}
}

func TestSwipe_RightButton(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Swipe{})
	rn, _ := node.Get("Swipe")

	rec := &recordingInput{}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{swInBegin: node.Point{X: 0.0, Y: 0.0}, swInEnd: node.Point{X: 1.0, Y: 1.0},
			swInButton: "right", swInDurationMs: 100},
		nil, withInput(rec), false)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if len(rec.calls) != 1 || rec.calls[0] != "Drag:0.000:0.000:1.000:1.000:right:100" {
		t.Errorf("calls = %v", rec.calls)
	}
}

func TestSwipe_BackendError_Propagates(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Swipe{})
	rn, _ := node.Get("Swipe")

	rec := &recordingInput{err: errors.New("hwnd closed")}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{swInBegin: node.Point{X: 0.1, Y: 0.2}, swInEnd: node.Point{X: 0.8, Y: 0.9}},
		nil, withInput(rec), false)

	if r.Error == nil {
		t.Fatal("expected backend error to propagate")
	}
}

func TestSwipe_InvalidButton_ValidationError(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Swipe{})
	rn, _ := node.Get("Swipe")

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{swInButton: "side1"},
		nil, withInput(&recordingInput{}), false)

	if len(r.Validation) != 1 || r.Validation[0].Code != "INVALID_MOUSE_BUTTON" {
		t.Errorf("validation = %v, want INVALID_MOUSE_BUTTON", r.Validation)
	}
}
