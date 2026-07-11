package control

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/yottaapp/yotta/internal/node"
)

func TestSleep_HappyPath(t *testing.T) {
	registry := node.NewRegistry()
	registry.Register(&Sleep{})
	rn, _ := registry.Get("Sleep")

	start := time.Now()
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{sleepInDuration: 50 * time.Millisecond},
		nil, node.StubServices(), false)
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
	registry := node.NewRegistry()
	registry.Register(&Sleep{})
	rn, _ := registry.Get("Sleep")

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	r := node.RunNode(ctx, rn, nil,
		map[string]any{sleepInDuration: 10 * time.Second},
		nil, node.StubServices(), false)
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
	registry := node.NewRegistry()
	registry.Register(&Sleep{})
	rn, _ := registry.Get("Sleep")

	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{sleepInDuration: time.Duration(0)},
		nil, node.StubServices(), false)

	if r.Error == nil {
		t.Error("expected error on zero Duration")
	}
}

func TestSleep_DefaultDuration_1s(t *testing.T) {
	registry := node.NewRegistry()
	registry.Register(&Sleep{})
	rn, _ := registry.Get("Sleep")

	// Spec 默认 = 1000ms (1s): 拖出来即可用, 不必手填。
	var dur *node.InputSpec
	for i := range rn.Spec.Inputs {
		if rn.Spec.Inputs[i].Name == sleepInDuration {
			dur = &rn.Spec.Inputs[i]
		}
	}
	if dur == nil {
		t.Fatal("Sleep 缺 Duration input")
	}
	if dur.Required {
		t.Error("Duration 有默认值后不该再标 Required (默认总能满足 Has, Required 成死标)")
	}
	if fmt.Sprintf("%v", dur.Default) != "1000" {
		t.Errorf("Duration 默认 = %v, want 1000 (1s)", dur.Default)
	}

	// 空 config 不再报 REQUIRED_FIELD_MISSING —— 默认值已填充 (用已取消 ctx 立即返回, 不真等 1s)。
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r := node.RunNode(ctx, rn, nil, nil, nil, node.StubServices(), false)
	if len(r.Validation) != 0 {
		t.Errorf("有默认值后空 config 不该报 required-missing, got %v", r.Validation)
	}
}
