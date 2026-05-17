package capture

import (
	"errors"
	"fmt"
	"image"

	"github.com/lxn/win"
)

type mockBackend struct{}

func newMockBackend() (*mockBackend, error) {
	if err := ensureMockInit(); err != nil {
		return nil, fmt.Errorf("mock init: %w", err)
	}
	return &mockBackend{}, nil
}

func (b *mockBackend) Name() string { return "mock" }

// Frame mock 模式可能 hwnd=0 (没真窗口), IsWindow 跳过校验.
func (b *mockBackend) Frame(hwnd win.HWND) (img *image.RGBA, err error) {
	defer func() {
		if r := recover(); r != nil {
			img, err = nil, fmt.Errorf("mock.Frame panicked: %v", r)
		}
	}()
	return mockFrame(hwnd)
}

func (b *mockBackend) FrameROI(hwnd win.HWND, x, y, w, h int) (img *image.RGBA, err error) {
	defer func() {
		if r := recover(); r != nil {
			img, err = nil, fmt.Errorf("mock.FrameROI panicked: %v", r)
		}
	}()
	return mockFrameROI(hwnd, x, y, w, h)
}

func (b *mockBackend) ClientSize(hwnd win.HWND) (int, int, error) {
	if hwnd == 0 {
		// mock 没真窗口时返固定假尺寸 (跟 mock 文件预设对齐, 1920x1080 是默认录制基准)
		return 1920, 1080, nil
	}
	if !isWindow(hwnd) {
		return 0, 0, errors.New("mock.ClientSize: invalid hwnd")
	}
	return ClientSize(hwnd)
}

func (b *mockBackend) Close() error { return nil }
