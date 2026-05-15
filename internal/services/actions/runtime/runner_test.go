package runtime

import (
	"testing"
	"time"

	"yhbox/internal/services/actions"
)

// waitIdle 等 runner 回 idle，超时 fail
func waitIdle(t *testing.T, r *Runner, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if r.State() == StateIdle {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("runner 没在 %v 内回 idle (state=%d)", timeout, r.State())
}

func TestRunner_RunsKeyStep(t *testing.T) {
	drv := &MockDriver{}
	r := NewRunner(drv, nil)
	a := &actions.Action{
		Steps: []actions.Step{{Kind: actions.StepKey, Vk: "W", DurationMs: 10}},
	}
	actions.NormalizeAction(a)

	if err := r.Start(a); err != nil {
		t.Fatal(err)
	}
	waitIdle(t, r, time.Second)

	if len(drv.Calls) != 2 {
		t.Fatalf("应有 KeyDown+KeyUp 两次调用，got %+v", drv.Calls)
	}
	if drv.Calls[0].Op != "key_down" || drv.Calls[1].Op != "key_up" {
		t.Errorf("调用顺序错: %+v", drv.Calls)
	}
}

func TestRunner_StopReleasesPressedKey(t *testing.T) {
	drv := &MockDriver{}
	r := NewRunner(drv, nil)
	a := &actions.Action{
		Steps: []actions.Step{
			{Kind: actions.StepKeyDown, Vk: "W"},
			{Kind: actions.StepSleep, DurationMs: 5000},
		},
	}
	actions.NormalizeAction(a)
	if err := r.Start(a); err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond) // 让 sleep 5s 开始
	_ = r.Stop(StopImmediate)
	waitIdle(t, r, time.Second)

	// 必须有 KeyUp("W") release —— 否则 cleanup contract 失效
	sawUp := false
	for _, c := range drv.Calls {
		if c.Op == "key_up" && c.Vk == "W" {
			sawUp = true
		}
	}
	if !sawUp {
		t.Errorf("releaseAllKeys 应释放 W，got %+v", drv.Calls)
	}
}

func TestRunner_BusyRejected(t *testing.T) {
	drv := &MockDriver{}
	r := NewRunner(drv, nil)
	a := &actions.Action{
		Steps: []actions.Step{{Kind: actions.StepSleep, DurationMs: 200}},
	}
	actions.NormalizeAction(a)
	if err := r.Start(a); err != nil {
		t.Fatal(err)
	}
	if err := r.Start(a); err == nil {
		t.Error("第二次 Start 应返 errBusy")
	}
	_ = r.Stop(StopImmediate)
	waitIdle(t, r, time.Second)
}

func TestRunner_ClickStepRatioToPixel(t *testing.T) {
	t.Skip("需要真 hwnd 才能 capture.ClientSize；后续集成测时跑")
}
