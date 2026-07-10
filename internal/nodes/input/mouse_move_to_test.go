package input

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/node"
)

func TestMouseMoveTo_HappyPath(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&MouseMoveTo{})
	rn, _ := node.Get("MouseMoveTo")
	rec := &recordingInput{}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{mmtInPoint: node.Point{X: 0.4, Y: 0.6}},
		nil, withInput(rec), false)
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	// MoveMs=0 → 单帧瞬移 MoveTo
	if len(rec.calls) != 1 || rec.calls[0] != "MoveTo:0.400:0.600" {
		t.Errorf("calls=%v want MoveTo:0.400:0.600", rec.calls)
	}
}

func TestMouseMoveTo_PxPoint_ResolvesToRatio(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&MouseMoveTo{})
	rn, _ := node.Get("MouseMoveTo")
	rec := &recordingInput{}
	b := node.StubServices()
	b.Input = rec
	b.Window = sizeWindow{w: 800, h: 600}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{mmtInPoint: node.Point{X: 400, Y: 300, Unit: node.UnitPx}},
		nil, b, false)
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if len(rec.calls) != 1 || rec.calls[0] != "MoveTo:0.500:0.500" {
		t.Errorf("calls=%v want MoveTo:0.500:0.500", rec.calls)
	}
}

func TestMouseMoveTo_Instant(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&MouseMoveTo{})
	rn, _ := node.Get("MouseMoveTo")

	rec := &recordingInput{}
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{mmtInPoint: node.Point{X: 0.5, Y: 0.5}, mmtInMoveMs: 0},
		nil, withInput(rec), false)
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != mmtOutDone {
		t.Errorf("exit = %q, want Done", r.ExitName)
	}
	// MoveMs=0 → 单帧瞬移
	if len(rec.calls) != 1 || rec.calls[0] != "MoveTo:0.500:0.500" {
		t.Errorf("calls = %v, want [MoveTo:0.500:0.500]", rec.calls)
	}
}

func TestMouseMoveTo_Slide_FramesEndAtTarget(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&MouseMoveTo{})
	rn, _ := node.Get("MouseMoveTo")

	rec := &recordingInput{}
	// MoveMs=64 → 64/16 = 4 帧; spy 起点 (0,0); 直线插值, 末帧落终点 (1,1)
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{mmtInPoint: node.Point{X: 1.0, Y: 1.0}, mmtInMoveMs: 64},
		nil, withInput(rec), false)
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if len(rec.calls) != 4 {
		t.Fatalf("calls = %v, want 4 帧", rec.calls)
	}
	if rec.calls[3] != "MoveTo:1.000:1.000" {
		t.Errorf("末帧 = %q, want MoveTo:1.000:1.000 (终点)", rec.calls[3])
	}
}

func TestMouseMoveTo_CtxCancel_StopsMidSlide(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&MouseMoveTo{})
	rn, _ := node.Get("MouseMoveTo")

	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(20 * time.Millisecond); cancel() }()

	rec := &recordingInput{}
	start := time.Now()
	r := node.RunNode(ctx, rn, nil,
		map[string]any{mmtInPoint: node.Point{X: 1.0, Y: 1.0}, mmtInMoveMs: 10000},
		nil, withInput(rec), false)
	elapsed := time.Since(start)

	if elapsed > time.Second {
		t.Fatalf("elapsed %v — MouseMoveTo 没响应 ctx 取消, 仍滑满 10s", elapsed)
	}
	if r.Error == nil || !errors.Is(r.Error, context.Canceled) {
		t.Errorf("error = %v, want context.Canceled", r.Error)
	}
}
