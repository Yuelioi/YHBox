package calibration

import "runtime"

// 包装一层方便测试 mock；当前直接转发到 runtime。
func runtimeLockOSThread()   { runtime.LockOSThread() }
func runtimeUnlockOSThread() { runtime.UnlockOSThread() }
