package input

import (
	"context"
	"testing"

	"yotta/internal/node"
)

func TestMouseHoldStart_HappyPath(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&MouseHoldStart{})
	rn, _ := node.Get("MouseHoldStart")

	rec := &recordingInput{}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{mhStartInXRatio: 0.25, mhStartInYRatio: 0.75, mhStartInButton: "right"},
		nil, withInput(rec))

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != mhStartOutOut {
		t.Errorf("exit = %q, want Out", r.ExitName)
	}
	if len(rec.calls) != 1 || rec.calls[0] != "MouseDown:0.250:0.750:right" {
		t.Errorf("calls = %v, want [MouseDown:0.250:0.750:right]", rec.calls)
	}
}

func TestMouseHoldStop_HappyPath(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&MouseHoldStop{})
	rn, _ := node.Get("MouseHoldStop")

	rec := &recordingInput{}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{mhStopInButton: "middle"},
		nil, withInput(rec))

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

func TestMouseHoldStart_InvalidButton_ValidationError(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&MouseHoldStart{})
	rn, _ := node.Get("MouseHoldStart")

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{mhStartInButton: "x1"},
		nil, withInput(&recordingInput{}))

	if len(r.Validation) != 1 || r.Validation[0].Code != "INVALID_MOUSE_BUTTON" {
		t.Errorf("validation = %v, want INVALID_MOUSE_BUTTON", r.Validation)
	}
}

func TestMouseHoldStop_InvalidButton_ValidationError(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&MouseHoldStop{})
	rn, _ := node.Get("MouseHoldStop")

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{mhStopInButton: "x2"},
		nil, withInput(&recordingInput{}))

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
		map[string]any{mhStartInXRatio: 0.5, mhStartInYRatio: 0.5, mhStartInButton: "left"},
		nil, bundle)
	if r1.Error != nil {
		t.Fatal(r1.Error)
	}

	rnStop, _ := node.Get("MouseHoldStop")
	r2 := node.RunNode(context.Background(), rnStop, nil,
		map[string]any{mhStopInButton: "left"}, nil, bundle)
	if r2.Error != nil {
		t.Fatal(r2.Error)
	}

	want := []string{"MouseDown:0.500:0.500:left", "MouseUp:left"}
	if len(rec.calls) != 2 || rec.calls[0] != want[0] || rec.calls[1] != want[1] {
		t.Errorf("calls = %v, want %v", rec.calls, want)
	}
}
