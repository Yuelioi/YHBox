package input

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"yotta/internal/node"
)

func TestClickAt_HappyPath(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&ClickAt{})
	rn, _ := node.Get("ClickAt")

	rec := &recordingInput{}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{caInXRatio: 0.3, caInYRatio: 0.7, caInButton: "right", caInDurationMs: 80},
		nil, withInput(rec), false)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != caOutDone {
		t.Errorf("exit = %q, want Done", r.ExitName)
	}
	if len(rec.calls) != 2 ||
		rec.calls[0] != "MoveTo:0.300:0.700" ||
		rec.calls[1] != "Click:0.300:0.700:right:80" {
		t.Errorf("calls = %v, want [MoveTo Click]", rec.calls)
	}
}

func TestClickAt_DefaultsApplied(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&ClickAt{})
	rn, _ := node.Get("ClickAt")

	rec := &recordingInput{}
	// 不传任何 config — 全走 Default (0.5, 0.5, left, 50)
	r := node.RunNode(context.Background(), rn, nil, nil, nil, withInput(rec), false)
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if len(rec.calls) != 2 ||
		rec.calls[0] != "MoveTo:0.500:0.500" ||
		rec.calls[1] != "Click:0.500:0.500:left:50" {
		t.Errorf("calls = %v, want [MoveTo Click]", rec.calls)
	}
}

// ClickAt 走 Click (内部 down→hold→up 原子, 不可中途取消) → 取消语义在「滑动阶段」:
// 长 MoveMs 滑动途中取消 → 还没按下就返回, 不会发 Click, 不会有按键残留.
func TestClickAt_CtxCancel_AbortsBeforeClick(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&ClickAt{})
	rn, _ := node.Get("ClickAt")

	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(20 * time.Millisecond); cancel() }()

	rec := &recordingInput{}
	start := time.Now()
	r := node.RunNode(ctx, rn, nil,
		map[string]any{caInXRatio: 1.0, caInYRatio: 1.0, caInButton: "left",
			caInMoveMs: 2000, caInDurationMs: 50},
		nil, withInput(rec), false)
	elapsed := time.Since(start)

	if elapsed > time.Second {
		t.Fatalf("elapsed %v — ClickAt 没在滑动途中响应 ctx 取消", elapsed)
	}
	if r.Error == nil || !errors.Is(r.Error, context.Canceled) {
		t.Errorf("error = %v, want context.Canceled", r.Error)
	}
	for _, c := range rec.calls {
		if strings.HasPrefix(c, "Click") {
			t.Errorf("取消应发生在按下前, 不该有 Click; calls = %v", rec.calls)
		}
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
		nil, withInput(rec), false)
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if len(rec.calls) != 5 {
		t.Fatalf("calls = %v, want 5 (4 滑动帧 + Click)", rec.calls)
	}
	if rec.calls[3] != "MoveTo:1.000:1.000" {
		t.Errorf("末滑帧 = %q, want MoveTo:1.000:1.000", rec.calls[3])
	}
	if rec.calls[4] != "Click:1.000:1.000:left:10" {
		t.Errorf("click = %q, want Click:1.000:1.000:left:10", rec.calls[4])
	}
}

func TestClickAt_InvalidButton_ValidationError(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&ClickAt{})
	rn, _ := node.Get("ClickAt")

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{caInButton: "side1"},
		nil, withInput(&recordingInput{}), false)

	if len(r.Validation) != 1 || r.Validation[0].Code != "INVALID_MOUSE_BUTTON" {
		t.Errorf("validation = %v, want INVALID_MOUSE_BUTTON", r.Validation)
	}
}
