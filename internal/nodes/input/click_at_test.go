package input

import (
	"context"
	"errors"
	"testing"
	"time"

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
	if len(rec.calls) != 2 || rec.calls[0] != "MouseDown:0.300:0.700:right" || rec.calls[1] != "MouseUp:right" {
		t.Errorf("calls = %v, want [MouseDown:0.300:0.700:right MouseUp:right]", rec.calls)
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
	if len(rec.calls) != 2 || rec.calls[0] != "MouseDown:0.500:0.500:left" || rec.calls[1] != "MouseUp:left" {
		t.Errorf("calls = %v, want [MouseDown:0.500:0.500:left MouseUp:left]", rec.calls)
	}
}

func TestClickAt_CtxCancel_ReleasesAndReturns(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&ClickAt{})
	rn, _ := node.Get("ClickAt")

	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(20 * time.Millisecond); cancel() }()

	rec := &recordingInput{}
	start := time.Now()
	r := node.RunNode(ctx, rn, nil,
		map[string]any{caInXRatio: 0.5, caInYRatio: 0.5, caInButton: "left", caInDurationMs: 10000},
		nil, withInput(rec))
	elapsed := time.Since(start)

	if elapsed > time.Second {
		t.Fatalf("elapsed %v — ClickAt 没响应 ctx 取消", elapsed)
	}
	if r.Error == nil || !errors.Is(r.Error, context.Canceled) {
		t.Errorf("error = %v, want context.Canceled", r.Error)
	}
	if len(rec.calls) != 2 || rec.calls[0] != "MouseDown:0.500:0.500:left" || rec.calls[1] != "MouseUp:left" {
		t.Errorf("calls = %v, want [MouseDown:0.500:0.500:left MouseUp:left]", rec.calls)
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
