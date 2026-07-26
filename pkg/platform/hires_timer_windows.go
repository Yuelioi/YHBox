package platform

import (
	"fmt"
	"sync/atomic"
	"syscall"
)

// HighResTimer 包装 winmm.timeBeginPeriod(1) / timeEndPeriod(1)，
// 让 time.Sleep / time.NewTicker 精度从默认 15.6ms 提到 ~1ms。
//
// Begin 全进程生效（系统级精度），同一进程多个 HighResTimer 引用计数。
// End 必须配对，否则进程一直保持高精度（耗电）。
type HighResTimer struct {
	active atomic.Bool
}

func NewHighResTimer() *HighResTimer { return &HighResTimer{} }

var (
	winmm                  = syscall.NewLazyDLL("winmm.dll")
	procTimeBeginPeriod    = winmm.NewProc("timeBeginPeriod")
	procTimeEndPeriod      = winmm.NewProc("timeEndPeriod")
	timeBeginPeriodSuccess = uintptr(0) // TIMERR_NOERROR
)

// Begin 提升进程定时器精度到 1ms。已开则 no-op。
func (h *HighResTimer) Begin() error {
	if h.active.Swap(true) {
		return nil
	}
	r, _, _ := procTimeBeginPeriod.Call(1)
	if r != timeBeginPeriodSuccess {
		h.active.Store(false)
		return fmt.Errorf("timeBeginPeriod(1) failed: code=%d", r)
	}
	return nil
}

// End 释放高精度定时器。未 Begin 或已 End 时 no-op。
func (h *HighResTimer) End() {
	if !h.active.Swap(false) {
		return
	}
	procTimeEndPeriod.Call(1)
}
