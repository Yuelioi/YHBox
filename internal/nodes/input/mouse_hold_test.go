package input

import (
	"context"
	"testing"

	"github.com/yottaapp/yotta/internal/node"
)

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

func TestMouseHoldStop_HappyPath(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&MouseHoldStop{})
	rn, _ := node.Get("MouseHoldStop")

	rec := &recordingInput{}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{mhStopInButton: "middle"},
		nil, withInput(rec), false)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != mhStopOutOut {
		t.Errorf("exit = %q, want Out", r.ExitName)
	}
	if len(rec.calls) != 1 || rec.calls[0] != "MouseUp:middle" {
		t.Errorf("calls = %v, want [MouseUp:middle]", rec.calls)
	}
}

func TestMouseHoldStop_InvalidButton_ValidationError(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&MouseHoldStop{})
	rn, _ := node.Get("MouseHoldStop")

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{mhStopInButton: "x2"},
		nil, withInput(&recordingInput{}), false)

	if len(r.Validation) != 1 || r.Validation[0].Code != "INVALID_MOUSE_BUTTON" {
		t.Errorf("validation = %v, want INVALID_MOUSE_BUTTON", r.Validation)
	}
}

// Start/Stop 配对走完整 down→up.
func TestMouseHold_StartStopPair(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&MouseHoldStart{})
	node.Register(&MouseHoldStop{})

	rec := &recordingInput{}
	bundle := withInput(rec)

	rnStart, _ := node.Get("MouseHoldStart")
	r1 := node.RunNode(context.Background(), rnStart, nil,
		map[string]any{mhStartInPoint: node.Point{X: 0.5, Y: 0.5}, mhStartInButton: "left"},
		nil, bundle, false)
	if r1.Error != nil {
		t.Fatal(r1.Error)
	}

	rnStop, _ := node.Get("MouseHoldStop")
	r2 := node.RunNode(context.Background(), rnStop, nil,
		map[string]any{mhStopInButton: "left"}, nil, bundle, false)
	if r2.Error != nil {
		t.Fatal(r2.Error)
	}

	want := []string{"MouseDown:0.500:0.500:left", "MouseUp:left"}
	if len(rec.calls) != 2 || rec.calls[0] != want[0] || rec.calls[1] != want[1] {
		t.Errorf("calls = %v, want %v", rec.calls, want)
	}
}
