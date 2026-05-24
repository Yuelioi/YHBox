package input

import (
	"context"
	"testing"

	"yhbox/internal/node"
)

func TestClickAt_HappyPath(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&ClickAt{})
	rn, _ := node.Get("ClickAt")

	rec := &recordingInput{}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{caInXRatio: 0.3, caInYRatio: 0.7, caInButton: "right", caInDurationMs: 80},
		nil, withInput(rec))

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != caOutDone {
		t.Errorf("exit = %q, want Done", r.ExitName)
	}
	if len(rec.calls) != 1 || rec.calls[0] != "Click:0.300:0.700:right:80" {
		t.Errorf("calls = %v, want [Click:0.300:0.700:right:80]", rec.calls)
	}
}

func TestClickAt_DefaultsApplied(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&ClickAt{})
	rn, _ := node.Get("ClickAt")

	rec := &recordingInput{}
	// 不传任何 config — 全走 Default (0.5, 0.5, left, 50)
	r := node.RunNode(context.Background(), rn, nil, nil, nil, withInput(rec))
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if len(rec.calls) != 1 || rec.calls[0] != "Click:0.500:0.500:left:50" {
		t.Errorf("calls = %v, want [Click:0.500:0.500:left:50]", rec.calls)
	}
}

func TestClickAt_InvalidButton_ValidationError(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&ClickAt{})
	rn, _ := node.Get("ClickAt")

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{caInButton: "side1"},
		nil, withInput(&recordingInput{}))

	if len(r.Validation) != 1 || r.Validation[0].Code != "INVALID_MOUSE_BUTTON" {
		t.Errorf("validation = %v, want INVALID_MOUSE_BUTTON", r.Validation)
	}
}
