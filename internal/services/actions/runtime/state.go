package runtime

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/lxn/win"
)

type RunnerState int

const (
	StateIdle RunnerState = iota
	StateRunning
	StateStopping
	StateRecording // 给 recorder 占位；runner.go 不用
)

// StopMode 当前仅实现 StopImmediate；StopGraceful 占位，行为同 StopImmediate。
type StopMode int

const (
	StopImmediate StopMode = iota
	StopGraceful  // 占位；当前等同 StopImmediate
)

type Runner struct {
	mu        sync.Mutex
	state     RunnerState
	current   *RuntimeContext // 一次 RunOnce 的 mutable runtime state；idle 时 nil
	driver    InputDriver
	hwnd      win.HWND
	nextRunID atomic.Uint64
	nextSeq   atomic.Uint64          // event Seq 单调递增
	emit      func(EventActionState) // wails Emit 注入；可为 nil（单测）
	// localMouseCounts360Getter 取当前机器的 360° HID counts 基准。
	// 调用即得，避免 Runner 跟 settings 同步。返 0 = 未校准；回放不缩放。
	localMouseCounts360Getter func() int
}

func NewRunner(driver InputDriver, emit func(EventActionState)) *Runner {
	return &Runner{driver: driver, emit: emit}
}

// SetLocalMouseCounts360Getter main.go 启动时注入；getter 每次 MouseMoveRel
// 都被调用一次，反映 settings 当前值（用户改完不用重启）。
func (r *Runner) SetLocalMouseCounts360Getter(f func() int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.localMouseCounts360Getter = f
}

// scaleRelMouse MouseMoveRel 的跨 DPI 缩放：localCounts/sourceCounts。
// 双方都 >0 且不等才缩放，否则原值返回。
func (r *Runner) scaleRelMouse(dx, dy int) (int, int) {
	r.mu.Lock()
	getter := r.localMouseCounts360Getter
	cur := r.current
	r.mu.Unlock()
	if getter == nil {
		return dx, dy
	}
	local := int64(getter())
	if local <= 0 {
		return dx, dy
	}
	if cur == nil || cur.action == nil {
		return dx, dy
	}
	src := int64(cur.action.RecordingContext.MouseCounts360)
	if src <= 0 || src == local {
		return dx, dy
	}
	fx := float64(dx) * float64(local) / float64(src)
	fy := float64(dy) * float64(local) / float64(src)
	if fx < 0 {
		fx -= 0.5
	} else {
		fx += 0.5
	}
	if fy < 0 {
		fy -= 0.5
	} else {
		fy += 0.5
	}
	return int(fx), int(fy)
}

// SetHWND 由 main.go 在 game detect 之后调，绑定目标游戏窗口。
func (r *Runner) SetHWND(h win.HWND) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.hwnd = h
}

// State 给 service 层 / 测试查询。
func (r *Runner) State() RunnerState {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.state
}

// EventActionState 上报给前端的事件 payload。Seq 单调递增，前端凭它判 dropped/reorder。
type EventActionState struct {
	Seq       uint64    `json:"seq"`
	Timestamp time.Time `json:"timestamp"`
	ActionID  string    `json:"actionId"`
	RunID     uint64    `json:"runId"`
	Status    string    `json:"status"` // "running" | "step" | "idle" | "error"
	Error     string    `json:"error,omitempty"`

	// 进度字段（status=="step" 时填）—— 前端用来画进度条 / 高亮当前 step。
	StepIndex int    `json:"stepIndex,omitempty"`
	StepTotal int    `json:"stepTotal,omitempty"`
	StepID    string `json:"stepId,omitempty"`
	// LoopIter Action 内部不再有循环（搬到 Container），保留字段兼容 emit 调用。
	LoopIter int `json:"loopIter,omitempty"`
}
