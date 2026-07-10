package stopwatch

import (
	"context"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/node"
)

func TestRead_RunningReturnsPositive(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Read{})
	rn, _ := node.Get("StopwatchRead")

	services := node.StubServices()
	services.Stopwatches.Start("k")
	time.Sleep(10 * time.Millisecond)

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{swReadInKey: "k"}, nil, services, false)
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != swReadOutOut {
		t.Errorf("exit = %q, want Out", r.ExitName)
	}
	elapsed, _ := r.OutputData[swReadDataElapsedMs].(int64)
	if elapsed <= 0 {
		t.Errorf("ElapsedMs = %d, want > 0 after 10ms sleep", elapsed)
	}
}

func TestRead_MissingKeyReturnsZero(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Read{})
	rn, _ := node.Get("StopwatchRead")

	services := node.StubServices()
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{swReadInKey: "never_started"}, nil, services, false)
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	elapsed, _ := r.OutputData[swReadDataElapsedMs].(int64)
	if elapsed != 0 {
		t.Errorf("ElapsedMs = %d, want 0 for missing key", elapsed)
	}
}

func TestRead_StoppedReturnsFrozen(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Read{})
	rn, _ := node.Get("StopwatchRead")

	services := node.StubServices()
	services.Stopwatches.Start("k")
	time.Sleep(10 * time.Millisecond)
	services.Stopwatches.Stop("k")

	r1 := node.RunNode(context.Background(), rn, nil,
		map[string]any{swReadInKey: "k"}, nil, services, false)
	first, _ := r1.OutputData[swReadDataElapsedMs].(int64)

	time.Sleep(15 * time.Millisecond)

	r2 := node.RunNode(context.Background(), rn, nil,
		map[string]any{swReadInKey: "k"}, nil, services, false)
	second, _ := r2.OutputData[swReadDataElapsedMs].(int64)

	if first != second {
		t.Errorf("stopped Read drift first=%d second=%d (should be frozen)", first, second)
	}
}
