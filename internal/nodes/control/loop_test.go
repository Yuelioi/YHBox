package control

import (
	"context"
	"errors"
	"testing"

	"yotta/internal/node"
)

func TestLoop_CountMode_BodyInvokedN(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Loop{})
	rn, _ := node.Get("Loop")

	iterations := 0
	body := func(_ node.Ctx) error {
		iterations++
		return nil
	}

	r := node.RunNodeAsRegion(context.Background(), rn, nil,
		map[string]any{loopInMode: "count", loopInCount: 5},
		nil, node.StubServices(), false, body)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.Panic != nil {
		t.Fatalf("panic: %v", r.Panic)
	}
	if iterations != 5 {
		t.Errorf("iterations = %d, want 5", iterations)
	}
	if r.ExitName != loopOutDone {
		t.Errorf("exit = %q, want %q", r.ExitName, loopOutDone)
	}
}

func TestLoop_BreakSentinel(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Loop{})
	rn, _ := node.Get("Loop")

	iterations := 0
	body := func(_ node.Ctx) error {
		iterations++
		if iterations == 3 {
			return errBreakRequested
		}
		return nil
	}

	r := node.RunNodeAsRegion(context.Background(), rn, nil,
		map[string]any{loopInMode: "count", loopInCount: 100},
		nil, node.StubServices(), false, body)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if iterations != 3 {
		t.Errorf("iterations = %d, want 3 (break at 3rd)", iterations)
	}
	if r.ExitName != loopOutDone {
		t.Errorf("exit = %q, want %q", r.ExitName, loopOutDone)
	}
}

func TestLoop_ContinueSentinel(t *testing.T) {
	// Continue 不 break — count=3 → body 调 3 次, 最后走 done.
	node.ResetRegistryForTest()
	node.Register(&Loop{})
	rn, _ := node.Get("Loop")

	iterations := 0
	body := func(_ node.Ctx) error {
		iterations++
		return errContinueRequested
	}

	r := node.RunNodeAsRegion(context.Background(), rn, nil,
		map[string]any{loopInMode: "count", loopInCount: 3},
		nil, node.StubServices(), false, body)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if iterations != 3 {
		t.Errorf("iterations = %d, want 3 (continue 不消耗轮次)", iterations)
	}
	if r.ExitName != loopOutDone {
		t.Errorf("exit = %q, want %q", r.ExitName, loopOutDone)
	}
}

func TestLoop_BodyErrorPropagates(t *testing.T) {
	// 非 sentinel error → 节点返该 error, ExitName 空.
	node.ResetRegistryForTest()
	node.Register(&Loop{})
	rn, _ := node.Get("Loop")

	boom := errors.New("boom")
	iterations := 0
	body := func(_ node.Ctx) error {
		iterations++
		if iterations == 2 {
			return boom
		}
		return nil
	}

	r := node.RunNodeAsRegion(context.Background(), rn, nil,
		map[string]any{loopInMode: "count", loopInCount: 10},
		nil, node.StubServices(), false, body)

	if !errors.Is(r.Error, boom) {
		t.Errorf("error = %v, want boom", r.Error)
	}
	if iterations != 2 {
		t.Errorf("iterations = %d, want 2", iterations)
	}
	if r.ExitName != "" {
		t.Errorf("exit = %q, want empty (error path)", r.ExitName)
	}
}

func TestLoop_ForeverMode_BreakRequired(t *testing.T) {
	// mode=forever → body 必须 Break 才停, 否则死循环. 防御: 用 break sentinel 在第 7 次跳.
	node.ResetRegistryForTest()
	node.Register(&Loop{})
	rn, _ := node.Get("Loop")

	iterations := 0
	body := func(_ node.Ctx) error {
		iterations++
		if iterations == 7 {
			return errBreakRequested
		}
		return nil
	}

	r := node.RunNodeAsRegion(context.Background(), rn, nil,
		map[string]any{loopInMode: "forever"},
		nil, node.StubServices(), false, body)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if iterations != 7 {
		t.Errorf("iterations = %d, want 7", iterations)
	}
	if r.ExitName != loopOutDone {
		t.Errorf("exit = %q, want %q", r.ExitName, loopOutDone)
	}
}

func TestLoop_UnknownMode_Error(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Loop{})
	rn, _ := node.Get("Loop")

	r := node.RunNodeAsRegion(context.Background(), rn, nil,
		map[string]any{loopInMode: "while"}, // while was old runtime, 砍掉了
		nil, node.StubServices(), false, func(_ node.Ctx) error { return nil })

	if r.Error == nil {
		t.Errorf("expected error on unknown mode, got nil")
	}
}

func TestLoop_NoRegionRunner_NotARegionError(t *testing.T) {
	// Sanity: RunNodeAsRegion on a non-RegionRunner node (Sleep) 应返 "not a RegionRunner".
	node.ResetRegistryForTest()
	node.Register(&Sleep{})
	rn, _ := node.Get("Sleep")

	r := node.RunNodeAsRegion(context.Background(), rn, nil, nil, nil,
		node.StubServices(), false, func(_ node.Ctx) error { return nil })

	if r.Error == nil {
		t.Errorf("expected error for non-RegionRunner, got nil")
	}
}
