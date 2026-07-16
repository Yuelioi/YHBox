package capture

import (
	"image"
	"sync"
)

// IBackend is a per-Run instance interface. 与包级 Backend enum + Frame/FrameROI 并存.
//
// impl 必须 IsWindow(hwnd) 前置 + defer recover(), invalid hwnd 返 (nil, error) 永不 panic.
// WGC impl 额外捕 WGC Session 错误 (Direct3D11CaptureFramePool 在窗口刚关闭瞬间会抛 Closed).
type IBackend interface {
	Name() string
	Frame(hwnd Handle) (*image.RGBA, error)
	FrameROI(hwnd Handle, x, y, w, h int) (*image.RGBA, error)
	ClientSize(hwnd Handle) (int, int, error)
	Close() error
}

var (
	mockInitOnce sync.Once
	mockInitErr  error
)

func ensureMockInit() error {
	mockInitOnce.Do(func() {
		mockInitErr = initMock()
	})
	return mockInitErr
}
