package detect

import (
	"context"
	"testing"

	"yotta/internal/node"
)

// sig1 构造单格签名 (1×1 grid → 3 字节).
func sig1(r, g, b uint8) []uint8 { return []uint8{r, g, b} }

// ---- WaitStable ----

func TestWaitStable_BecomesStable(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&WaitStable{})
	rn, _ := node.Get("WaitStable")

	a := sig1(100, 100, 100)
	// 首帧 prev=a, 之后 3 个 poll 全 a → 连续 3 帧不变 → Stable.
	vision := &mockVision{gridSigs: [][]uint8{a, a, a, a}}
	cfg := map[string]any{
		wfGridSize:        1,
		wsStableThreshold: 0.01,
		wsStableFrames:    3,
		wfPollIntervalMs:  5,
		wfTimeoutMs:       2000,
	}
	r := node.RunNode(context.Background(), rn, nil, cfg, nil, withVision(vision), false)
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != wsOutStable {
		t.Errorf("exit = %q, want Stable", r.ExitName)
	}
}

func TestWaitStable_TimeoutWhenChurning(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&WaitStable{})
	rn, _ := node.Get("WaitStable")

	a, b := sig1(0, 0, 0), sig1(255, 255, 255)
	// 交替 → 每帧都变 → stableCount 永远归零 → timeout.
	vision := &mockVision{gridSigs: [][]uint8{a, b, a, b, a, b, a, b}}
	cfg := map[string]any{
		wfGridSize:        1,
		wsStableThreshold: 0.01,
		wsStableFrames:    3,
		wfPollIntervalMs:  10,
		wfTimeoutMs:       30,
	}
	r := node.RunNode(context.Background(), rn, nil, cfg, nil, withVision(vision), false)
	if r.ExitName != wsOutTimeout {
		t.Errorf("exit = %q, want Timeout", r.ExitName)
	}
}

// ---- WaitChange ----

func TestWaitChange_DetectsChange(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&WaitChange{})
	rn, _ := node.Get("WaitChange")

	base, changed := sig1(10, 10, 10), sig1(250, 250, 250)
	// baseline=base, poll1=base (无变化), poll2=changed → Changed.
	vision := &mockVision{gridSigs: [][]uint8{base, base, changed}}
	cfg := map[string]any{
		wfGridSize:        1,
		wcChangeThreshold: 0.5,
		wfPollIntervalMs:  5,
		wfTimeoutMs:       2000,
	}
	r := node.RunNode(context.Background(), rn, nil, cfg, nil, withVision(vision), false)
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	if r.ExitName != wcOutChanged {
		t.Errorf("exit = %q, want Changed", r.ExitName)
	}
	if v, ok := r.OutputData[wfDataValue].(float64); !ok || v < 0.5 {
		t.Errorf("Value = %v (ok=%v), want >=0.5", r.OutputData[wfDataValue], ok)
	}
}

func TestWaitChange_TimeoutWhenStatic(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&WaitChange{})
	rn, _ := node.Get("WaitChange")

	a := sig1(10, 10, 10)
	vision := &mockVision{gridSigs: [][]uint8{a}} // mock 耗尽返最后一个 → 恒 a → 永不变.
	cfg := map[string]any{
		wfGridSize:        1,
		wcChangeThreshold: 0.5,
		wfPollIntervalMs:  10,
		wfTimeoutMs:       30,
	}
	r := node.RunNode(context.Background(), rn, nil, cfg, nil, withVision(vision), false)
	if r.ExitName != wcOutTimeout {
		t.Errorf("exit = %q, want Timeout", r.ExitName)
	}
}

func TestWaitChange_MeanDiffMetric(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&WaitChange{})
	rn, _ := node.Get("WaitChange")

	base, changed := sig1(0, 0, 0), sig1(255, 255, 255) // mean_diff = 1.0
	vision := &mockVision{gridSigs: [][]uint8{base, changed}}
	cfg := map[string]any{
		wfGridSize:        1,
		wfMetric:          metricMeanDiff,
		wcChangeThreshold: 0.5,
		wfPollIntervalMs:  5,
		wfTimeoutMs:       2000,
	}
	r := node.RunNode(context.Background(), rn, nil, cfg, nil, withVision(vision), false)
	if r.ExitName != wcOutChanged {
		t.Errorf("exit = %q, want Changed", r.ExitName)
	}
}

func TestWaitChange_UnknownMetricErrors(t *testing.T) {
	node.ResetRegistryForTest()
	node.Register(&WaitChange{})
	rn, _ := node.Get("WaitChange")

	vision := &mockVision{gridSigs: [][]uint8{sig1(0, 0, 0)}}
	cfg := map[string]any{
		wfGridSize:        1,
		wfMetric:          "bogus",
		wcChangeThreshold: 0.5,
		wfTimeoutMs:       100,
	}
	r := node.RunNode(context.Background(), rn, nil, cfg, nil, withVision(vision), false)
	if r.Error == nil {
		t.Error("expected error on unknown metric")
	}
}
