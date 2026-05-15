// Package winutil 找异环游戏窗口。lxn/win 这版缺 EnumWindows / GetWindowTextW，
// 用 LazyDLL 自包。匹配规则：标题以"异环"开头 且 类名 = UnrealWindow（两条都要满足，
// 防止浏览器标签等"异环 ... - Chrome"被误匹配）。
package winutil

import (
	"syscall"
	"unsafe"

	"github.com/lxn/win"
)

var (
	user32                  = syscall.NewLazyDLL("user32.dll")
	procEnumWindows         = user32.NewProc("EnumWindows")
	procGetWindowTextW      = user32.NewProc("GetWindowTextW")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	procShowWindow          = user32.NewProc("ShowWindow")
	procIsIconic            = user32.NewProc("IsIconic")
	procAttachThreadInput   = user32.NewProc("AttachThreadInput")
	procGetWindowThreadProcId = user32.NewProc("GetWindowThreadProcessId")
	procGetForegroundWindow = user32.NewProc("GetForegroundWindow")
	kernel32                = syscall.NewLazyDLL("kernel32.dll")
	procGetCurrentThreadId  = kernel32.NewProc("GetCurrentThreadId")
)

const (
	swRestore = 9
)

type Target struct {
	HWND  win.HWND
	Title string
	Class string
}

func utf16ToString(buf []uint16) string {
	n := 0
	for n < len(buf) && buf[n] != 0 {
		n++
	}
	return syscall.UTF16ToString(buf[:n])
}

// BringToFront 把目标窗口拉到前台并恢复（如最小化）。Windows 限制：当前进程
// 不持有 fg 锁时直接 SetForegroundWindow 会被忽略，这里用 AttachThreadInput 借
// 当前前台线程的输入队列把限制绕过。失败返 false 不抛错——非关键路径。
func BringToFront(hwnd win.HWND) bool {
	if hwnd == 0 {
		return false
	}
	// 最小化的话先 restore
	iconic, _, _ := procIsIconic.Call(uintptr(hwnd))
	if iconic != 0 {
		procShowWindow.Call(uintptr(hwnd), swRestore)
	}
	curTid, _, _ := procGetCurrentThreadId.Call()
	fgHwnd, _, _ := procGetForegroundWindow.Call()
	fgTid, _, _ := procGetWindowThreadProcId.Call(fgHwnd, 0)
	attached := false
	if fgTid != 0 && fgTid != curTid {
		ret, _, _ := procAttachThreadInput.Call(curTid, fgTid, 1)
		attached = ret != 0
	}
	defer func() {
		if attached {
			procAttachThreadInput.Call(curTid, fgTid, 0)
		}
	}()
	ret, _, _ := procSetForegroundWindow.Call(uintptr(hwnd))
	return ret != 0
}

// FindGame 枚举所有可见顶层窗口，返回所有匹配的（通常只有一个）。
func FindGame() []Target {
	var found []Target
	cb := syscall.NewCallback(func(hwnd win.HWND, _ uintptr) uintptr {
		if !win.IsWindowVisible(hwnd) {
			return 1
		}
		var tBuf [256]uint16
		var cBuf [256]uint16
		procGetWindowTextW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&tBuf[0])), uintptr(len(tBuf)))
		win.GetClassName(hwnd, &cBuf[0], len(cBuf))
		t := utf16ToString(tBuf[:])
		c := utf16ToString(cBuf[:])
		titleMatch := len(t) >= len("异环") && t[:len("异环")] == "异环"
		classMatch := c == "UnrealWindow"
		if titleMatch && classMatch {
			found = append(found, Target{hwnd, t, c})
		}
		return 1
	})
	procEnumWindows.Call(cb, 0)
	return found
}
