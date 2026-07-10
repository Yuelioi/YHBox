package input

import (
	"context"
	"testing"

	"github.com/yottaapp/yotta/internal/node"
)

func TestKeyHoldStart_HappyPath(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&KeyHoldStart{})
	rn, _ := node.Get("KeyHoldStart")

	rec := &recordingInput{}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{khStartInVK: "D"},
		nil, withInput(rec), false)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != khStartOutOut {
		t.Errorf("exit = %q, want Out", r.ExitName)
	}
	if len(rec.calls) != 1 || rec.calls[0] != "KeyDown:D" {
		t.Errorf("calls = %v, want [KeyDown:D]", rec.calls)
	}
}

func TestKeyHoldStop_HappyPath(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&KeyHoldStop{})
	rn, _ := node.Get("KeyHoldStop")

	rec := &recordingInput{}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{khStopInVK: "D"},
		nil, withInput(rec), false)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != khStopOutOut {
		t.Errorf("exit = %q, want Out", r.ExitName)
	}
	if len(rec.calls) != 1 || rec.calls[0] != "KeyUp:D" {
		t.Errorf("calls = %v, want [KeyUp:D]", rec.calls)
	}
}

func TestKeyHoldStart_UnknownVK_ValidationError(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&KeyHoldStart{})
	rn, _ := node.Get("KeyHoldStart")

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{khStartInVK: "no-such-key"},
		nil, withInput(&recordingInput{}), false)

	if len(r.Validation) != 1 || r.Validation[0].Code != "INVALID_VK" {
		t.Errorf("validation = %v, want INVALID_VK", r.Validation)
	}
}

func TestKeyHoldStop_EmptyVK_ValidationError(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&KeyHoldStop{})
	rn, _ := node.Get("KeyHoldStop")

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{khStopInVK: ""},
		nil, withInput(&recordingInput{}), false)

	if len(r.Validation) != 1 || r.Validation[0].Code != "INVALID_VK" {
		t.Errorf("validation = %v, want INVALID_VK", r.Validation)
	}
}

// 配对 Start/Stop 走完整 down→up 序列, 验 stateful backend 路径.
func TestKeyHold_StartStopPair(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&KeyHoldStart{})
	node.Register(&KeyHoldStop{})

	rec := &recordingInput{}
	bundle := withInput(rec)

	rnStart, _ := node.Get("KeyHoldStart")
	r1 := node.RunNode(context.Background(), rnStart, nil,
		map[string]any{khStartInVK: "right"}, nil, bundle, false)
	if r1.Error != nil {
		t.Fatal(r1.Error)
	}

	rnStop, _ := node.Get("KeyHoldStop")
	r2 := node.RunNode(context.Background(), rnStop, nil,
		map[string]any{khStopInVK: "right"}, nil, bundle, false)
	if r2.Error != nil {
		t.Fatal(r2.Error)
	}

	want := []string{"KeyDown:right", "KeyUp:right"}
	if len(rec.calls) != 2 || rec.calls[0] != want[0] || rec.calls[1] != want[1] {
		t.Errorf("calls = %v, want %v", rec.calls, want)
	}
}
