//go:build windows

package runtime

import (
	"syscall"
	"unsafe"
)

var (
	kernel32                      = syscall.NewLazyDLL("kernel32.dll")
	procQueryPerformanceCounter   = kernel32.NewProc("QueryPerformanceCounter")
	procQueryPerformanceFrequency = kernel32.NewProc("QueryPerformanceFrequency")
	procTimeBeginPeriod           = syscall.NewLazyDLL("winmm.dll").NewProc("timeBeginPeriod")
	procTimeEndPeriod             = syscall.NewLazyDLL("winmm.dll").NewProc("timeEndPeriod")
)

var qpcFreq int64

func init() {
	var f int64
	procQueryPerformanceFrequency.Call(uintptr(unsafe.Pointer(&f)))
	qpcFreq = f
}

// QPCMicros 当前 QPC 微秒计数. monotonic high-res.
func QPCMicros() uint64 {
	var t int64
	procQueryPerformanceCounter.Call(uintptr(unsafe.Pointer(&t)))
	if qpcFreq == 0 {
		return 0
	}
	return uint64(t) * 1_000_000 / uint64(qpcFreq)
}

// TimerSetPrecision1ms 调 timeBeginPeriod(1). 回放前调一次, 后调 TimerResetPrecision.
func TimerSetPrecision1ms() {
	procTimeBeginPeriod.Call(1)
}

func TimerResetPrecision() {
	procTimeEndPeriod.Call(1)
}
