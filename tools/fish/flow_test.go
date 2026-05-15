package fish

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

// TestRunFlow_EmptySteps 空 Steps 列表也能跑完，且把 state 切到 OnDone。
func TestRunFlow_EmptySteps(t *testing.T) {
	m := newTestMachine(t)
	f := flow{Name: "empty", Steps: nil, OnDone: IDLE}
	m.state = SHOPSELL
	runFlow(context.Background(), m, f)
	if m.state != IDLE {
		t.Errorf("state = %v, want IDLE", m.state)
	}
}

// TestRunFlow_AllStepsRun 所有 step 顺序执行。
func TestRunFlow_AllStepsRun(t *testing.T) {
	m := newTestMachine(t)
	var calls []int
	mkStep := func(n int) Step {
		return func(_ context.Context, _ *machine) error {
			calls = append(calls, n)
			return nil
		}
	}
	f := flow{
		Name:   "seq",
		Steps:  []Step{mkStep(1), mkStep(2), mkStep(3)},
		OnDone: IDLE,
	}
	runFlow(context.Background(), m, f)
	if len(calls) != 3 || calls[0] != 1 || calls[1] != 2 || calls[2] != 3 {
		t.Errorf("calls = %v, want [1 2 3]", calls)
	}
}

// TestRunFlow_StepErrorAborts 任一 step 返回 error 后续 step 不执行，state 仍切 OnDone。
func TestRunFlow_StepErrorAborts(t *testing.T) {
	m := newTestMachine(t)
	var calls []int
	f := flow{
		Name: "abort",
		Steps: []Step{
			func(_ context.Context, _ *machine) error { calls = append(calls, 1); return nil },
			func(_ context.Context, _ *machine) error { calls = append(calls, 2); return errors.New("boom") },
			func(_ context.Context, _ *machine) error { calls = append(calls, 3); return nil },
		},
		OnDone: IDLE,
	}
	m.state = SHOPSELL
	runFlow(context.Background(), m, f)
	if len(calls) != 2 {
		t.Errorf("calls = %v, want first 2 only", calls)
	}
	if m.state != IDLE {
		t.Errorf("state = %v, want IDLE", m.state)
	}
}

// TestWaitDur_RespectsCtxCancel ctx 取消后 waitDur 立即返回 errFlowAbort。
func TestWaitDur_RespectsCtxCancel(t *testing.T) {
	m := newTestMachine(t)
	ctx, cancel := context.WithCancel(context.Background())
	step := waitDur(10 * time.Second)

	done := make(chan error, 1)
	go func() { done <- step(ctx, m) }()
	time.Sleep(50 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, errFlowAbort) {
			t.Errorf("err = %v, want errFlowAbort", err)
		}
	case <-time.After(time.Second):
		t.Fatal("waitDur 没在 ctx 取消后立即返回")
	}
}

// newTestMachine 构造一个最小可用的 machine — 不真实 hwnd / ctrl / detector / logger，
// 只测试 runFlow 引擎和 waitDur 这种纯逻辑原语。log 用 zerolog.Nop() 静默丢弃。
func newTestMachine(t *testing.T) *machine {
	t.Helper()
	return &machine{
		cfg:  &Config{},
		log:  zerolog.Nop(),
		ctrl: nil,
	}
}

// clickUntilGone / clickIfSeen 没有单元测试 — 内部 capture.Frame(m.hwnd) 要求真 HWND，
// hwnd=0 的测试 machine 抓帧直接失败，没法走到 detect/click 路径。
// 这两个函数依赖真游戏窗口的截屏 + 检测器，单测无法覆盖；靠手测验证。
