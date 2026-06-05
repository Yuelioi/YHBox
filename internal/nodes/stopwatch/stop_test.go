package stopwatch

import (
	"context"
	"testing"
	"time"

	"yotta/internal/node"
)

func TestStop_HappyPath(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Stop{})
	rn, _ := node.Get("StopwatchStop")

	services := node.StubServices()
	services.Stopwatches.Start("k")
	time.Sleep(5 * time.Millisecond)

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{swStopInKey: "k"}, nil, services, false)
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != swStopOutOut {
		t.Errorf("exit = %q, want Out", r.ExitName)
	}

	// stopped: Read 应返固定值, 再调一次差值很小 (running=false)
	first := services.Stopwatches.Read("k")
	time.Sleep(10 * time.Millisecond)
	second := services.Stopwatches.Read("k")
	if first != second {
		t.Errorf("after Stop: Read drift first=%d second=%d (should be frozen)", first, second)
	}
}

func TestStop_MissingKeyIsNoop(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Stop{})
	rn, _ := node.Get("StopwatchStop")

	// 不 Start, 直接 Stop — 静默 no-op
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{swStopInKey: "never_started"}, nil, node.StubServices(), false)
	if r.Error != nil {
		t.Fatalf("Stop on missing key should be no-op, got: %v", r.Error)
	}
}
