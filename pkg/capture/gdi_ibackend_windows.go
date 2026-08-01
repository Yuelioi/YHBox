//go:build windows

package capture

import (
	"errors"
	"fmt"
	"image"
	"sync"

	"github.com/lxn/win"
)

type gdiBackend struct {
	mu      sync.Mutex
	surface gdiCaptureSurface
	closed  bool
}

func newGDIBackend() (*gdiBackend, error) {
	return &gdiBackend{}, nil
}

func (b *gdiBackend) Name() string { return "gdi" }

func (b *gdiBackend) Frame(hwnd win.HWND) (img *image.RGBA, err error) {
	if !isWindow(hwnd) {
		return nil, errors.New("gdi.Frame: invalid hwnd")
	}
	defer func() {
		if r := recover(); r != nil {
			img, err = nil, fmt.Errorf("gdi.Frame panicked: %v", r)
		}
	}()
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return nil, errors.New("gdi.Frame: backend closed")
	}
	return gdiFrame(&b.surface, hwnd)
}

func (b *gdiBackend) FrameROI(hwnd win.HWND, x, y, w, h int) (img *image.RGBA, err error) {
	if !isWindow(hwnd) {
		return nil, errors.New("gdi.FrameROI: invalid hwnd")
	}
	defer func() {
		if r := recover(); r != nil {
			img, err = nil, fmt.Errorf("gdi.FrameROI panicked: %v", r)
		}
	}()
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return nil, errors.New("gdi.FrameROI: backend closed")
	}
	return gdiFrameROI(&b.surface, hwnd, x, y, w, h)
}

func (b *gdiBackend) ClientSize(hwnd win.HWND) (int, int, error) {
	if !isWindow(hwnd) {
		return 0, 0, errors.New("gdi.ClientSize: invalid hwnd")
	}
	return winClientSize(hwnd)
}

func (b *gdiBackend) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return nil
	}
	b.closed = true
	b.surface.close()
	return nil
}
