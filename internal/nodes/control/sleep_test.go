package control

import (
	"context"
	"errors"
	"testing"
	"time"

	"yhbox/internal/node"
)

func TestSleep_HappyPath(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Sleep{})
	rn, _ := node.Get("Sleep")

	start := time.Now()
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{sleepInDuration: 50 * time.Millisecond},
		nil, node.StubServices())
	elapsed := time.Since(start)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != sleepOutDone {
		t.Errorf("exit = %q, want Done", r.ExitName)
	}
	if elapsed < 40*time.Millisecond {
		t.Errorf("elapsed %v < 40ms — Sleep 没真等", elapsed)
	}
}

func TestSleep_CtxCancel_InterruptsEarly(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Sleep{})
	rn, _ := node.Get("Sleep")

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	r := node.RunNode(ctx, rn, nil,
		map[string]any{sleepInDuration: 10 * time.Second},
		nil, node.StubServices())
	elapsed := time.Since(start)

	if elapsed > time.Second {
		t.Fatalf("elapsed %v — Sleep 没响应 ctx 取消, 仍等满 10s", elapsed)
	}
	if r.Error == nil || !errors.Is(r.Error, context.Canceled) {
		t.Errorf("error = %v, want context.Canceled", r.Error)
	}
	if r.ExitName != "" {
		t.Errorf("exit = %q, 取消时不该 Fire Done", r.ExitName)
	}
}

func TestSleep_ZeroDuration_Error(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Sleep{})
	rn, _ := node.Get("Sleep")

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{sleepInDuration: time.Duration(0)},
		nil, node.StubServices())

	if r.Error == nil {
		t.Error("expected error on zero Duration")
	}
}

func TestSleep_RequiredMissing(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Sleep{})
	rn, _ := node.Get("Sleep")

	r := node.RunNode(context.Background(), rn, nil, nil, nil, node.StubServices())
	if len(r.Validation) == 0 {
		t.Error("expected REQUIRED_FIELD_MISSING for Duration")
	}
}
