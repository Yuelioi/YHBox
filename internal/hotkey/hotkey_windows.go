//go:build windows

package hotkey

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"syscall"
	"time"
	"unsafe"

	"github.com/lxn/win"
)

const (
	winWM_HOTKEY = 0x0312
	winPM_REMOVE = 0x0001

	errHotkeyAlreadyRegistered = 1409
)

var (
	user32               = syscall.NewLazyDLL("user32.dll")
	procRegisterHotKey   = user32.NewProc("RegisterHotKey")
	procUnregisterHotKey = user32.NewProc("UnregisterHotKey")
	procPeekMessageW     = user32.NewProc("PeekMessageW")
)

func runHotkeyLoop(ctx context.Context, specs []HotkeySpec, handler func(int), ready chan<- error, done chan<- struct{}) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer close(done)

	registered := make([]int, 0, len(specs))
	defer func() {
		for _, id := range registered {
			procUnregisterHotKey.Call(0, uintptr(id))
		}
	}()

	for _, spec := range specs {
		result, _, callErr := procRegisterHotKey.Call(0, uintptr(spec.ID), uintptr(spec.Mods), uintptr(spec.VK))
		if result == 0 {
			var errno syscall.Errno
			if errors.As(callErr, &errno) && errno == errHotkeyAlreadyRegistered {
				ready <- fmt.Errorf("热键 %s 已被其它应用占用", spec.Name)
			} else {
				ready <- fmt.Errorf("注册热键 %s 失败: %v", spec.Name, callErr)
			}
			return
		}
		registered = append(registered, spec.ID)
	}
	ready <- nil

	var msg win.MSG
	tick := time.NewTicker(20 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
		result, _, _ := procPeekMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0, winPM_REMOVE)
		if result != 0 && msg.Message == winWM_HOTKEY && handler != nil {
			handler(int(msg.WParam))
		}
	}
}
