//go:build windows

package capture

import (
	"errors"
	"fmt"
	"runtime"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"github.com/lxn/win"
)

var gdiFixtureSequence atomic.Uint64

func gdiFixtureWindowProc(hwnd win.HWND, message uint32, wParam, lParam uintptr) uintptr {
	switch message {
	case win.WM_CLOSE:
		win.DestroyWindow(hwnd)
		return 0
	case win.WM_DESTROY:
		win.PostQuitMessage(0)
		return 0
	default:
		return win.DefWindowProc(hwnd, message, wParam, lParam)
	}
}

func startGDIFixture(t testing.TB) win.HWND {
	t.Helper()
	type result struct {
		hwnd win.HWND
		err  error
	}
	ready := make(chan result, 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		className, err := syscall.UTF16PtrFromString(fmt.Sprintf("YottaGDICaptureFixture%d", gdiFixtureSequence.Add(1)))
		if err != nil {
			ready <- result{err: err}
			return
		}
		instance := win.GetModuleHandle(nil)
		windowClass := win.WNDCLASSEX{
			CbSize:        uint32(unsafe.Sizeof(win.WNDCLASSEX{})),
			LpfnWndProc:   syscall.NewCallback(gdiFixtureWindowProc),
			HInstance:     instance,
			HbrBackground: win.HBRUSH(win.COLOR_WINDOW + 1),
			LpszClassName: className,
		}
		if win.RegisterClassEx(&windowClass) == 0 {
			ready <- result{err: errors.New("register GDI fixture window class")}
			return
		}
		defer win.UnregisterClass(className)
		title, err := syscall.UTF16PtrFromString("Yotta GDI Capture Fixture")
		if err != nil {
			ready <- result{err: err}
			return
		}
		hwnd := win.CreateWindowEx(0, className, title, win.WS_OVERLAPPEDWINDOW|win.WS_VISIBLE,
			160, 160, 480, 320, 0, 0, instance, nil)
		if hwnd == 0 {
			ready <- result{err: errors.New("create GDI fixture window")}
			return
		}
		win.ShowWindow(hwnd, win.SW_SHOW)
		win.UpdateWindow(hwnd)
		ready <- result{hwnd: hwnd}

		var message win.MSG
		for win.GetMessage(&message, 0, 0, 0) > 0 {
			win.TranslateMessage(&message)
			win.DispatchMessage(&message)
		}
	}()

	select {
	case started := <-ready:
		if started.err != nil {
			t.Fatal(started.err)
		}
		t.Cleanup(func() {
			win.PostMessage(started.hwnd, win.WM_CLOSE, 0, 0)
			select {
			case <-done:
			case <-time.After(3 * time.Second):
				t.Error("GDI fixture did not close")
			}
		})
		return started.hwnd
	case <-time.After(3 * time.Second):
		t.Fatal("GDI fixture did not start")
		return 0
	}
}

func TestGDIBackendReusesSurfaceUntilWindowSizeChanges(t *testing.T) {
	hwnd := startGDIFixture(t)
	backend, err := newGDIBackend()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = backend.Close() })

	if _, err := backend.FrameROI(hwnd, 10, 10, 80, 60); err != nil {
		t.Fatalf("first FrameROI: %v", err)
	}
	firstDC, firstBitmap, firstBits := backend.surface.hdcDst, backend.surface.hbm, backend.surface.bits
	if firstDC == 0 || firstBitmap == 0 || firstBits == nil {
		t.Fatal("first capture did not initialize the cached GDI surface")
	}
	if _, err := backend.FrameROI(hwnd, 20, 20, 100, 70); err != nil {
		t.Fatalf("second FrameROI: %v", err)
	}
	if backend.surface.hdcDst != firstDC || backend.surface.hbm != firstBitmap || backend.surface.bits != firstBits {
		t.Fatal("same-size capture rebuilt the cached GDI surface")
	}

	oldW, oldH := backend.surface.winW, backend.surface.winH
	if !win.SetWindowPos(hwnd, 0, 160, 160, int32(oldW+80), int32(oldH+40), win.SWP_NOZORDER) {
		t.Fatal("resize GDI fixture window")
	}
	if _, err := backend.FrameROI(hwnd, 10, 10, 80, 60); err != nil {
		t.Fatalf("FrameROI after resize: %v", err)
	}
	if backend.surface.winW == oldW && backend.surface.winH == oldH {
		t.Fatal("window resize did not rebuild the cached GDI surface")
	}

	if err := backend.Close(); err != nil {
		t.Fatal(err)
	}
	if backend.surface.hdcDst != 0 || backend.surface.hbm != 0 || backend.surface.bits != nil {
		t.Fatal("Close did not release the cached GDI surface")
	}
	if _, err := backend.FrameROI(hwnd, 0, 0, 10, 10); err == nil {
		t.Fatal("FrameROI succeeded after backend Close")
	}
}

func TestGDIFrameROIMatchesFullFrameCrop(t *testing.T) {
	hwnd := startGDIFixture(t)
	backend, err := newGDIBackend()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = backend.Close() })

	if _, err := backend.Frame(hwnd); err != nil {
		t.Fatalf("warm-up Frame: %v", err)
	}
	time.Sleep(30 * time.Millisecond)
	full, err := backend.Frame(hwnd)
	if err != nil {
		t.Fatalf("Frame: %v", err)
	}
	const x, y, width, height = 37, 29, 160, 80
	roi, err := backend.FrameROI(hwnd, x, y, width, height)
	if err != nil {
		t.Fatalf("FrameROI: %v", err)
	}
	if roi.Bounds().Dx() != width || roi.Bounds().Dy() != height {
		t.Fatalf("FrameROI bounds = %v, want %dx%d", roi.Bounds(), width, height)
	}
	mismatches := 0
	firstMismatch := ""
	for pixelY := range height {
		for pixelX := range width {
			if got, want := roi.RGBAAt(pixelX, pixelY), full.RGBAAt(x+pixelX, y+pixelY); got != want {
				mismatches++
				if firstMismatch == "" {
					firstMismatch = fmt.Sprintf("pixel %d,%d = %#v, want %#v", pixelX, pixelY, got, want)
				}
			}
		}
	}
	if mismatches != 0 {
		t.Fatalf("FrameROI differs from full-frame crop at %d/%d pixels; first %s", mismatches, width*height, firstMismatch)
	}
}

func BenchmarkGDIBackendFrameROI(b *testing.B) {
	hwnd := startGDIFixture(b)
	for _, benchmark := range []struct {
		name          string
		rebuildBefore bool
	}{
		{name: "cached"},
		{name: "rebuild-each-frame", rebuildBefore: true},
	} {
		b.Run(benchmark.name, func(b *testing.B) {
			backend, err := newGDIBackend()
			if err != nil {
				b.Fatal(err)
			}
			b.Cleanup(func() { _ = backend.Close() })
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				if benchmark.rebuildBefore {
					backend.mu.Lock()
					backend.surface.close()
					backend.mu.Unlock()
				}
				if _, err := backend.FrameROI(hwnd, 40, 40, 160, 80); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
