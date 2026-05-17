package recording

import "testing"

// 真测 LL hook / IsPointInsideGameWindow（命中分支）需要真 hwnd 和真鼠标输入，
// 集成阶段手测。单测只能验证编译 + 基础结构 + 不会 panic。

func TestPackageCompiles(t *testing.T) {
	// gameHwnd=0 时 IsPointInsideGameWindow 走 short-circuit 返 false。
	if IsPointInsideGameWindow(0, 0, 0) {
		t.Error("gameHwnd=0 应返 false")
	}
	if IsPointInsideGameWindow(0, 100, 100) {
		t.Error("gameHwnd=0 应返 false")
	}
	// 负坐标 / 大坐标都不应 panic
	if IsPointInsideGameWindow(0, -1000, -1000) {
		t.Error("gameHwnd=0 应返 false")
	}
}

func TestGetCurrentThreadID(t *testing.T) {
	tid := GetCurrentThreadID()
	if tid == 0 {
		t.Error("线程 id 不应为 0")
	}
	// 同一 goroutine 立刻再调，因为 test 主 goroutine 没 LockOSThread，
	// 两次未必相同 —— 只验证非零即可。
}

func TestPostQuitToInvalidThreadDoesNotPanic(t *testing.T) {
	// 给一个不存在的 thread id 发 WM_QUIT，PostThreadMessage 会失败返 0，
	// 不应 panic。
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("PostQuitToThread 不应 panic, 但 panic 了: %v", r)
		}
	}()
	PostQuitToThread(0xFFFFFFFF) // 大概率不存在的 tid
}
