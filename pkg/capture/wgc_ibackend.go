package capture

import (
	"errors"
	"fmt"
	"image"

	"github.com/lxn/win"
)

type wgcBackend struct{}

func newWGCBackend() (*wgcBackend, error) {
	if WindowsBuild() < 18362 {
		return nil, errors.New("WGC requires Windows 10 1903+")
	}
	if err := ensureWGCInit(); err != nil {
		return nil, fmt.Errorf("WGC init: %w", err)
	}
	return &wgcBackend{}, nil
}

func (b *wgcBackend) Name() string { return "wgc" }

// Frame WGC 三层防御 (IsWindow 前置 + defer recover + WGC Session error 由底层 wgcFrame 处理).
// Direct3D11CaptureFramePool 在窗口刚关闭瞬间可能抛 Closed 异常 — recover 兜底.
func (b *wgcBackend) Frame(hwnd win.HWND) (img *image.RGBA, err error) {
	if !isWindow(hwnd) {
		return nil, errors.New("wgc.Frame: invalid hwnd")
	}
	defer func() {
		if r := recover(); r != nil {
			img, err = nil, fmt.Errorf("wgc.Frame panicked (likely Direct3D11CaptureFramePool closed): %v", r)
		}
	}()
	return wgcFrame(hwnd)
}

func (b *wgcBackend) FrameROI(hwnd win.HWND, x, y, w, h int) (img *image.RGBA, err error) {
	if !isWindow(hwnd) {
		return nil, errors.New("wgc.FrameROI: invalid hwnd")
	}
	defer func() {
		if r := recover(); r != nil {
			img, err = nil, fmt.Errorf("wgc.FrameROI panicked: %v", r)
		}
	}()
	return wgcFrameROI(hwnd, x, y, w, h)
}

func (b *wgcBackend) ClientSize(hwnd win.HWND) (int, int, error) {
	if !isWindow(hwnd) {
		return 0, 0, errors.New("wgc.ClientSize: invalid hwnd")
	}
	return ClientSize(hwnd)
}

func (b *wgcBackend) Close() error {
	// wgc backend 当前是 stateless wrapper, 无 Direct3D11CaptureFramePool / COM 资源要 Release, no-op.
	return nil
}
