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
	if len(rec.calls) != 3 ||
		rec.calls[0] != "MoveTo:0.300:0.700" ||
		rec.calls[1] != "MouseDown:0.300:0.700:right" ||
		rec.calls[2] != "MouseUp:right" {
		t.Errorf("calls = %v, want [MoveTo MouseDown MouseUp]", rec.calls)
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
	if len(rec.calls) != 3 ||
		rec.calls[0] != "MoveTo:0.500:0.500" ||
		rec.calls[1] != "MouseDown:0.500:0.500:left" ||
		rec.calls[2] != "MouseUp:left" {
		t.Errorf("calls = %v, want [MoveTo MouseDown MouseUp]", rec.calls)
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
	if len(rec.calls) != 3 ||
		rec.calls[0] != "MoveTo:0.500:0.500" ||
		rec.calls[1] != "MouseDown:0.500:0.500:left" ||
		rec.calls[2] != "MouseUp:left" {
		t.Errorf("calls = %v, want [MoveTo MouseDown MouseUp]", rec.calls)
	}
}

func TestClickAt_MoveMs_SlidesBeforeDown(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&ClickAt{})
	rn, _ := node.Get("ClickAt")

	rec := &recordingInput{}
	// MoveMs=64 → 4 帧 MoveTo (起点 spy(0,0) → 终点 (1,1)), 再 MouseDown/Up
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{caInXRatio: 1.0, caInYRatio: 1.0, caInButton: "left",
			caInMoveMs: 64, caInDurationMs: 10},
		nil, withInput(rec))
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if len(rec.calls) != 6 {
		t.Fatalf("calls = %v, want 6 (4 滑动帧 + down + up)", rec.calls)
	}
	if rec.calls[3] != "MoveTo:1.000:1.000" {
		t.Errorf("末滑帧 = %q, want MoveTo:1.000:1.000", rec.calls[3])
	}
	if rec.calls[4] != "MouseDown:1.000:1.000:left" || rec.calls[5] != "MouseUp:left" {
		t.Errorf("down/up = %v, want [MouseDown:1.000:1.000:left MouseUp:left]", rec.calls[4:])
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
