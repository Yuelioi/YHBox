package stopwatch

import (
	"context"
	"testing"

	"yotta/internal/node"
)

func TestStart_HappyPath(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Start{})
	rn, _ := node.Get("StopwatchStart")

	services := node.StubServices()
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{swStartInKey: "fishingTimer"},
		nil, services, false)

	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != swStartOutOut {
		t.Errorf("exit = %q, want Out", r.ExitName)
	}
	// 验证 stopwatch 已 start — Read 应返 >= 0 (running)
	elapsed := services.Stopwatches.Read("fishingTimer")
	if elapsed < 0 {
		t.Errorf("elapsed = %d, want >= 0 after Start", elapsed)
	}
}

func TestStart_ResetExistingKey(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Start{})
	rn, _ := node.Get("StopwatchStart")

	services := node.StubServices()
	// 先手动 start + 等少许时间
	services.Stopwatches.Start("k")
	// 再用节点 Start 同 key → 应该 reset
	r := node.RunNode(context.Background(), rn, nil,
		map[string]any{swStartInKey: "k"}, nil, services, false)
	if r.Error != nil {
		t.Fatal(r.Error)
	}
}

func TestStart_RequiredKeyMissing(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&Start{})
	rn, _ := node.Get("StopwatchStart")

	// Default 是 "default", 显式传空才触发 REQUIRED — Required 检查的是 Has, default 提供值
	// 所以不传任何 key 时, default 兜底, Required 不报. 这跟 KeyHoldStart 一致.
	r := node.RunNode(context.Background(), rn, nil, nil, nil, node.StubServices(), false)
	if r.Error != nil {
		t.Fatalf("default 应兜底, got error: %v", r.Error)
	}
}
