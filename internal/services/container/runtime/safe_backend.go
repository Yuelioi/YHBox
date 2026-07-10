package runtime

import (
	"image"
	"sync"

	pkgcapture "github.com/yottaapp/yotta/pkg/capture"
	pkginput "github.com/yottaapp/yotta/pkg/input"
)

// SafeInputBackend wraps pkginput.Backend, 集中 invalid-hwnd warn-once + emit.
// 新节点直接调 rt.Input.Click(...) 无需知道 SafeBackend 存在, warn 模板也不重复.
type SafeInputBackend struct {
	inner  pkginput.Backend
	rt     *RuntimeContext
	mu     sync.Mutex
	warned bool
}

func NewSafeInputBackend(inner pkginput.Backend, rt *RuntimeContext) *SafeInputBackend {
	return &SafeInputBackend{inner: inner, rt: rt}
}

func (s *SafeInputBackend) warnOnce(action string, hwnd pkginput.Handle) {
	s.mu.Lock()
	if s.warned {
		s.mu.Unlock()
		return
	}
	s.warned = true
	s.mu.Unlock()
	if s.rt != nil && s.rt.Emit != nil {
		s.rt.Emit("container:warning", map[string]any{
			"message": "目标窗口 hwnd=" + uintToHex(uintptr(hwnd)) + " 已失效, 后续 input 操作将失败. action=" + action,
		})
	}
}

func uintToHex(u uintptr) string {
	const hex = "0123456789abcdef"
	if u == 0 {
		return "0x0"
	}
	buf := [20]byte{}
	i := len(buf) - 1
	for u > 0 {
		buf[i] = hex[u&0xf]
		u >>= 4
		i--
	}
	return "0x" + string(buf[i+1:])
}

func (s *SafeInputBackend) Name() string                        { return s.inner.Name() }
func (s *SafeInputBackend) Capabilities() pkginput.Capabilities { return s.inner.Capabilities() }

func (s *SafeInputBackend) Click(hwnd pkginput.Handle, xr, yr float64, btn string, durMs int) error {
	if !isValidHwnd(uintptr(hwnd)) {
		s.warnOnce("Click", hwnd)
		return nil
	}
	return s.inner.Click(hwnd, xr, yr, btn, durMs)
}
func (s *SafeInputBackend) KeyDown(hwnd pkginput.Handle, vk string) error {
	if !isValidHwnd(uintptr(hwnd)) {
		s.warnOnce("KeyDown", hwnd)
		return nil
	}
	return s.inner.KeyDown(hwnd, vk)
}
func (s *SafeInputBackend) KeyUp(hwnd pkginput.Handle, vk string) error {
	// KeyUp 即使 hwnd invalid 也尝试 (释放 backend 内部 state)
	return s.inner.KeyUp(hwnd, vk)
}
func (s *SafeInputBackend) MouseDown(hwnd pkginput.Handle, xr, yr float64, btn string) error {
	if !isValidHwnd(uintptr(hwnd)) {
		s.warnOnce("MouseDown", hwnd)
		return nil
	}
	return s.inner.MouseDown(hwnd, xr, yr, btn)
}
func (s *SafeInputBackend) MouseUp(hwnd pkginput.Handle, btn string) error {
	return s.inner.MouseUp(hwnd, btn)
}
func (s *SafeInputBackend) MouseMoveRel(hwnd pkginput.Handle, dx, dy, durMs int) error {
	if !isValidHwnd(uintptr(hwnd)) {
		s.warnOnce("MouseMoveRel", hwnd)
		return nil
	}
	return s.inner.MouseMoveRel(hwnd, dx, dy, durMs)
}
func (s *SafeInputBackend) MoveTo(hwnd pkginput.Handle, xr, yr float64) error {
	if !isValidHwnd(uintptr(hwnd)) {
		s.warnOnce("MoveTo", hwnd)
		return nil
	}
	return s.inner.MoveTo(hwnd, xr, yr)
}
func (s *SafeInputBackend) CursorRatio(hwnd pkginput.Handle) (float64, float64, error) {
	if !isValidHwnd(uintptr(hwnd)) {
		s.warnOnce("CursorRatio", hwnd)
		return 0, 0, nil
	}
	return s.inner.CursorRatio(hwnd)
}
func (s *SafeInputBackend) Scroll(hwnd pkginput.Handle, xr, yr float64, notches int, horizontal bool) error {
	if !isValidHwnd(uintptr(hwnd)) {
		s.warnOnce("Scroll", hwnd)
		return nil
	}
	return s.inner.Scroll(hwnd, xr, yr, notches, horizontal)
}
func (s *SafeInputBackend) Drag(hwnd pkginput.Handle, x1, y1, x2, y2 float64, button string, durationMs int) error {
	if !isValidHwnd(uintptr(hwnd)) {
		s.warnOnce("Drag", hwnd)
		return nil
	}
	return s.inner.Drag(hwnd, x1, y1, x2, y2, button, durationMs)
}
func (s *SafeInputBackend) TypeText(hwnd pkginput.Handle, text string) error {
	if !isValidHwnd(uintptr(hwnd)) {
		s.warnOnce("TypeText", hwnd)
		return nil
	}
	return s.inner.TypeText(hwnd, text)
}
func (s *SafeInputBackend) ReleaseAll() error { return s.inner.ReleaseAll() }
func (s *SafeInputBackend) Close() error      { return s.inner.Close() }

// SafeCaptureBackend 同模式包 pkgcapture.IBackend.
type SafeCaptureBackend struct {
	inner  pkgcapture.IBackend
	rt     *RuntimeContext
	mu     sync.Mutex
	warned bool
}

func NewSafeCaptureBackend(inner pkgcapture.IBackend, rt *RuntimeContext) *SafeCaptureBackend {
	return &SafeCaptureBackend{inner: inner, rt: rt}
}

func (s *SafeCaptureBackend) warnOnce(action string, hwnd pkgcapture.Handle) {
	s.mu.Lock()
	if s.warned {
		s.mu.Unlock()
		return
	}
	s.warned = true
	s.mu.Unlock()
	if s.rt != nil && s.rt.Emit != nil {
		s.rt.Emit("container:warning", map[string]any{
			"message": "目标窗口 hwnd=" + uintToHex(uintptr(hwnd)) + " 已失效, 后续 capture 操作将失败. action=" + action,
		})
	}
}

func (s *SafeCaptureBackend) Name() string { return s.inner.Name() }

func (s *SafeCaptureBackend) Frame(hwnd pkgcapture.Handle) (*image.RGBA, error) {
	if !isValidHwnd(uintptr(hwnd)) {
		s.warnOnce("Frame", hwnd)
		return nil, nil
	}
	return s.inner.Frame(hwnd)
}
func (s *SafeCaptureBackend) FrameROI(hwnd pkgcapture.Handle, x, y, w, h int) (*image.RGBA, error) {
	if !isValidHwnd(uintptr(hwnd)) {
		s.warnOnce("FrameROI", hwnd)
		return nil, nil
	}
	return s.inner.FrameROI(hwnd, x, y, w, h)
}
func (s *SafeCaptureBackend) ClientSize(hwnd pkgcapture.Handle) (int, int, error) {
	if !isValidHwnd(uintptr(hwnd)) {
		s.warnOnce("ClientSize", hwnd)
		return 0, 0, nil
	}
	return s.inner.ClientSize(hwnd)
}
func (s *SafeCaptureBackend) Close() error { return s.inner.Close() }
