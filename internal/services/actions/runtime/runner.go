package runtime

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/lxn/win"

	"yhbox/internal/services/actions"
	"yhbox/pkg/capture"
	"yhbox/pkg/input"
)

var errBusy = errors.New("runner is busy")

// ctxSleep timer-based，可被 ctx cancel 打断。
// **必须**用 timer / NewTimer + select，不能用 time.Sleep —— 后者不可打断，
// 10 分钟 sleep 期间 Stop 完全没反应。这条 spec 写死。
func ctxSleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// Start 把 action 跑起来。state==Idle 才允许进入。
// 调用者必须先 NormalizeAction + Validate；Runner 不重复校验。
func (r *Runner) Start(a *actions.Action) error {
	runID := r.nextRunID.Add(1)
	r.mu.Lock()
	if r.state != StateIdle {
		r.mu.Unlock()
		return errBusy
	}
	ctx, cancel := context.WithCancel(context.Background())
	r.current = newRuntimeContext(a, runID, cancel)
	r.state = StateRunning
	hwnd := r.hwnd
	r.mu.Unlock()

	r.emitState(a.ID, runID, "running", nil)

	go func() {
		// 顺序：先 release 再 transition idle —— transition 后 r.current == nil，
		// release 拿不到 pressedKeys 就漏 release。Go defer LIFO，写在下面的先执行。
		defer r.transitionIdle(runID)
		defer r.releaseAllKeys(runID)
		if err := r.run(ctx, hwnd, a); err != nil && !errors.Is(err, context.Canceled) {
			r.emitState(a.ID, runID, "error", err)
		} else {
			r.emitState(a.ID, runID, "idle", nil)
		}
	}()
	return nil
}

// Stop 当前 v1 等同 StopImmediate：ctx cancel + releaseAllKeys。
// StopGraceful 留 v2 实现（当前 step 跑完才停）。
func (r *Runner) Stop(mode StopMode) error {
	_ = mode
	r.mu.Lock()
	if r.state != StateRunning {
		r.mu.Unlock()
		return nil
	}
	r.state = StateStopping
	cancel := r.current.Cancel
	r.mu.Unlock()
	cancel()
	return nil
}

// run Action v2: 单次线性执行 steps；循环/计数交给 Container 层。
func (r *Runner) run(ctx context.Context, hwnd win.HWND, a *actions.Action) error {
	for i := range a.Steps {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		r.emitStep(a.ID, i, len(a.Steps), a.Steps[i].ID, 0)
		if err := r.runStep(ctx, hwnd, &a.Steps[i]); err != nil {
			return err
		}
	}
	return nil
}

// emitStep 每跑 step 前调一次，让前端高亮 + 进度。
// 用独立 status="step"，不动现有 running/idle/error 语义。
func (r *Runner) emitStep(actionID string, stepIdx, stepTotal int, stepID string, loopIter int) {
	if r.emit == nil {
		return
	}
	r.mu.Lock()
	runID := uint64(0)
	if r.current != nil {
		runID = r.current.RunID
	}
	r.mu.Unlock()
	r.emit(EventActionState{
		Seq:       r.nextSeq.Add(1),
		Timestamp: time.Now(),
		ActionID:  actionID,
		RunID:     runID,
		Status:    "step",
		StepIndex: stepIdx,
		StepTotal: stepTotal,
		StepID:    stepID,
		LoopIter:  loopIter,
	})
}

// stepHooks 在执行单 step 时给 caller 插入"按键账面/dx 缩放"逻辑。
// runStep 用 runnerHooks 维护 pressedKeys + scaleRelMouse；ExecuteStep（容器
// InvokeAction 路径）用 noopHooks，因为容器不持有 action runner 状态。
type stepHooks interface {
	beforeKeyDown(vk string)
	afterKeyDown(vk string, err error)
	afterKeyUp(vk string, err error)
	scaleRel(dx, dy int) (int, int)
}

type noopHooks struct{}

func (noopHooks) beforeKeyDown(string)            {}
func (noopHooks) afterKeyDown(string, error)      {}
func (noopHooks) afterKeyUp(string, error)        {}
func (noopHooks) scaleRel(dx, dy int) (int, int) { return dx, dy }

// executeStepCore 单 step 执行的统一实现。caller 提供 driver/hwnd/hooks。
// runStep（Runner.run 主路径）和 ExecuteStep（container InvokeAction）都走这里。
//
// 注意：StepKey 路径里 ctxSleep 失败时仍要 KeyUp，避免按键泄漏；hooks 簿记顺序
// 依赖 caller —— runnerHooks 在 KeyDown 失败时回滚账面、KeyUp 成功时减计数。
func executeStepCore(ctx context.Context, driver InputDriver, hwnd win.HWND, s *actions.Step, hooks stepHooks) error {
	switch s.Kind {
	case actions.StepClickLeft, actions.StepClickMiddle, actions.StepClickRight:
		w, h, err := capture.ClientSize(hwnd)
		if err != nil {
			return err
		}
		x := int(float32(w) * s.XRatio)
		y := int(float32(h) * s.YRatio)
		return driver.Click(hwnd, x, y, buttonOf(s.Kind), s.DurationMs)

	case actions.StepKey:
		hooks.beforeKeyDown(s.Vk)
		if err := driver.KeyDown(hwnd, s.Vk); err != nil {
			hooks.afterKeyDown(s.Vk, err)
			return err
		}
		hooks.afterKeyDown(s.Vk, nil)
		_ = ctxSleep(ctx, time.Duration(s.DurationMs)*time.Millisecond)
		err := driver.KeyUp(hwnd, s.Vk)
		hooks.afterKeyUp(s.Vk, err)
		return err

	case actions.StepKeyDown:
		hooks.beforeKeyDown(s.Vk)
		err := driver.KeyDown(hwnd, s.Vk)
		hooks.afterKeyDown(s.Vk, err)
		return err

	case actions.StepKeyUp:
		err := driver.KeyUp(hwnd, s.Vk)
		hooks.afterKeyUp(s.Vk, err)
		return err

	case actions.StepSleep:
		return ctxSleep(ctx, time.Duration(s.DurationMs)*time.Millisecond)

	case actions.StepMouseMove:
		w, h, err := capture.ClientSize(hwnd)
		if err != nil {
			return err
		}
		x := int(float32(w) * s.XRatio)
		y := int(float32(h) * s.YRatio)
		return driver.MouseMove(hwnd, x, y)

	case actions.StepMouseDrag:
		w, h, err := capture.ClientSize(hwnd)
		if err != nil {
			return err
		}
		x1 := int(float32(w) * s.XRatio)
		y1 := int(float32(h) * s.YRatio)
		x2 := int(float32(w) * s.X2Ratio)
		y2 := int(float32(h) * s.Y2Ratio)
		return driver.MouseDrag(hwnd, x1, y1, x2, y2, mouseButtonOf(s.MouseButton), s.DurationMs)

	case actions.StepScroll:
		return driver.Scroll(hwnd, s.ScrollDelta)

	case actions.StepMouseMoveRel:
		dx, dy := hooks.scaleRel(s.Dx, s.Dy)
		return driver.MouseMoveRel(hwnd, dx, dy, s.DurationMs)
	}
	return fmt.Errorf("unknown step kind: %s", s.Kind)
}

// ExecuteStep 给 container.runtime InvokeAction 节点用：单步执行（无 emit、
// 无 pressedKeys 簿记 / 无 dx 缩放）。container 跑 InvokeAction 时按 step 串行
// 调一次，每步外套 InputBus.Lock/Unlock 保证输入互斥。ctx 取消立即返。
//
// 不维护 pressedKeys，因此 KeyDown / KeyUp 必须由 step 自身配对，否则 container
// 中断后会按键泄漏。Action 录制产物总是配对的；手编 Action 用户自负。
func ExecuteStep(ctx context.Context, driver InputDriver, hwnd win.HWND, s *actions.Step) error {
	return executeStepCore(ctx, driver, hwnd, s, noopHooks{})
}

// IsInputStep 判一步是否需要走 InputBus 锁（敲键/点鼠都要，sleep 不要）。
func IsInputStep(k actions.StepKind) bool {
	switch k {
	case actions.StepSleep:
		return false
	}
	return true
}

// runnerHooks 把 Runner 的 pressedKeys 簿记 + DPI 缩放接入 executeStepCore。
//
// 账面规则：
//   - StepKey: 先 KeyDown 成功 → markPressed，KeyUp 成功 → unmark；KeyUp 失败保留账面让
//     cleanup releaseAllKeys 补一刀。
//   - StepKeyDown: 先尝试 KeyDown，成功 → markPressed；失败不动账面。
//   - StepKeyUp: 仅 KeyUp 成功才 unmark，失败保留让 cleanup 处理。
type runnerHooks struct{ r *Runner }

func (h runnerHooks) beforeKeyDown(string) {}

func (h runnerHooks) afterKeyDown(vk string, err error) {
	if err == nil {
		h.r.markPressed(vk)
	}
}

func (h runnerHooks) afterKeyUp(vk string, err error) {
	if err == nil {
		h.r.unmarkPressed(vk)
	}
}

func (h runnerHooks) scaleRel(dx, dy int) (int, int) {
	return h.r.scaleRelMouse(dx, dy)
}

func (r *Runner) runStep(ctx context.Context, hwnd win.HWND, s *actions.Step) error {
	return executeStepCore(ctx, r.driver, hwnd, s, runnerHooks{r: r})
}

// mouseButtonOf 字符串 → MouseButton。空 / 未知 → MouseLeft。
func mouseButtonOf(s string) input.MouseButton {
	switch s {
	case "middle":
		return input.MouseMiddle
	case "right":
		return input.MouseRight
	default:
		return input.MouseLeft
	}
}

func buttonOf(k actions.StepKind) input.MouseButton {
	switch k {
	case actions.StepClickMiddle:
		return input.MouseMiddle
	case actions.StepClickRight:
		return input.MouseRight
	default:
		return input.MouseLeft
	}
}

func (r *Runner) markPressed(vk string) {
	r.mu.Lock()
	if r.current != nil {
		r.current.pressedKeys[vk]++
	}
	r.mu.Unlock()
}

func (r *Runner) unmarkPressed(vk string) {
	r.mu.Lock()
	if r.current != nil {
		r.current.pressedKeys[vk]--
		if r.current.pressedKeys[vk] <= 0 {
			delete(r.current.pressedKeys, vk)
		}
	}
	r.mu.Unlock()
}

// releaseAllKeys cleanup contract。带 runID 防 stale defer 覆盖新 run。
// 防 r.current == nil panic + runID 不匹配（不是这次 run 的 cleanup）。
func (r *Runner) releaseAllKeys(runID uint64) {
	r.mu.Lock()
	running := r.current
	if running == nil || running.RunID != runID {
		r.mu.Unlock()
		return
	}
	keys := slices.Collect(maps.Keys(running.pressedKeys))
	hwnd := r.hwnd
	r.mu.Unlock()
	for _, vk := range keys {
		_ = r.driver.KeyUp(hwnd, vk)
	}
}

// transitionIdle 带 runID 防 stale defer 覆盖新 run。
func (r *Runner) transitionIdle(runID uint64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.current == nil || r.current.RunID != runID {
		return
	}
	r.current = nil
	r.state = StateIdle
}

func (r *Runner) emitState(actionID string, runID uint64, status string, err error) {
	if r.emit == nil {
		return
	}
	ev := EventActionState{
		Seq:       r.nextSeq.Add(1),
		Timestamp: time.Now(),
		ActionID:  actionID,
		RunID:     runID,
		Status:    status,
	}
	if err != nil {
		ev.Error = err.Error()
	}
	r.emit(ev)
}
