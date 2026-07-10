//go:build windows

package capture

import (
	"errors"
	"fmt"
	"image"

	"github.com/lxn/win"
)

type gdiBackend struct{}

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
	return gdiFrame(hwnd)
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
	return gdiFrameROI(hwnd, x, y, w, h)
}

func (b *gdiBackend) ClientSize(hwnd win.HWND) (int, int, error) {
	if !isWindow(hwnd) {
		return 0, 0, errors.New("gdi.ClientSize: invalid hwnd")
	}
	return winClientSize(hwnd)
}

func (b *gdiBackend) Close() error { return nil }
