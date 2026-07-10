//go:build windows

package calibration

import (
	"sync/atomic"
	"testing"
	"time"
	"unsafe"
)

const wmKeyUp = 0x0101

// TestHotkeyKeyboardProc_AutorepeatDebounce 直接驱动 LL-hook callback, 验证去抖:
// 命中键 down 只在「按住」跳变时 fire 一次 (autorepeat 高频重发 keydown 不重复),
// up 复位, 再 down 再 fire. 对照 ll-hook-keydown-coalesce incident (录制热键没去抖
// → 多个 goroutine → toast 互相覆盖). 命中 down/up 都返 1 (拦截不透传游戏).
func TestHotkeyKeyboardProc_AutorepeatDebounce(t *testing.T) {
	var fires int32
	cb := func() { atomic.AddInt32(&fires, 1) }

	hkVK.Store(0x77) // VK_F8
	hkKeyHeld.Store(false)
	hkCallback.Store(&cb)
	t.Cleanup(func() {
		hkVK.Store(0)
		hkCallback.Store(nil)
		hkKeyHeld.Store(false)
	})

	kbd := kbdllhookstruct{VkCode: 0x77}
	lp := uintptr(unsafe.Pointer(&kbd))

	if r := hotkeyKeyboardProc(hcAction, wmKeyDown, lp); r != 1 {
		t.Fatalf("命中 keydown 应返 1 (拦截), got %d", r)
	}
	// autorepeat: 按住时 OS 高频重发 keydown — 不该重复 fire.
	hotkeyKeyboardProc(hcAction, wmKeyDown, lp)
	hotkeyKeyboardProc(hcAction, wmKeyDown, lp)

	if r := hotkeyKeyboardProc(hcAction, wmKeyUp, lp); r != 1 {
		t.Fatalf("命中 keyup 应返 1 (拦截), got %d", r)
	}
	// 复位后再按 → 再 fire 一次.
	hotkeyKeyboardProc(hcAction, wmKeyDown, lp)

	// callback 是 go cb() 异步 — 等一会儿再断言.
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) && atomic.LoadInt32(&fires) < 2 {
		time.Sleep(5 * time.Millisecond)
	}
	if got := atomic.LoadInt32(&fires); got != 2 {
		t.Fatalf("两轮独立按下应 fire 2 次 (autorepeat 不算), got %d", got)
	}
}

// TestHotkeyKeyboardProc_IgnoresOtherKeys 非命中键不 fire callback.
// (透传走 CallNextHookEx — 这里只验 callback 不被误触发; return 值是真实 Win32
// CallNextHookEx 结果, 不断言.)
func TestHotkeyKeyboardProc_IgnoresOtherKeys(t *testing.T) {
	var fires int32
	cb := func() { atomic.AddInt32(&fires, 1) }
	hkVK.Store(0x77)
	hkKeyHeld.Store(false)
	hkCallback.Store(&cb)
	t.Cleanup(func() {
		hkVK.Store(0)
		hkCallback.Store(nil)
		hkKeyHeld.Store(false)
	})

	other := kbdllhookstruct{VkCode: 0x41} // 'A'
	hotkeyKeyboardProc(hcAction, wmKeyDown, uintptr(unsafe.Pointer(&other)))

	time.Sleep(30 * time.Millisecond)
	if got := atomic.LoadInt32(&fires); got != 0 {
		t.Fatalf("非命中键不该 fire, got %d", got)
	}
}
