//go:build windows

// GDI backend: 用 PrintWindow(PW_RENDERFULLCONTENT) 抓客户区帧。
// gdi/wgc IBackend.ClientSize 走 winClientSize (纯 GetClientRect, 不经全局分发).
package capture

import (
	"fmt"
	"image"
	"syscall"
	"unsafe"

	"github.com/lxn/win"
)

const PW_RENDERFULLCONTENT = 0x00000002

var (
	user32             = syscall.NewLazyDLL("user32.dll")
	procPrintWindow    = user32.NewProc("PrintWindow")
	procClientToScreen = user32.NewProc("ClientToScreen")
	procIsWindow       = user32.NewProc("IsWindow")
)

// isWindow 走 user32.IsWindow — lxn/win 没暴露这个 API,
// IBackend impls 需要它做 hwnd 前置校验. 比 IsWindowVisible 更宽 (隐藏窗口也算合法).
func isWindow(hwnd win.HWND) bool {
	r, _, _ := procIsWindow.Call(uintptr(hwnd))
	return r != 0
}

type point struct {
	X, Y int32
}

type rawCapture struct {
	src              []byte
	winW, winH       int
	clientW, clientH int
	offsetX, offsetY int
}

// gdiCaptureSurface owns the memory DC and DIB used by PrintWindow. A capture
// backend is scoped to one Run, so keeping this surface alive avoids allocating
// and destroying full-window GDI resources on every polling frame. The surface
// is rebuilt only when the target window size changes.
type gdiCaptureSurface struct {
	hdcDst win.HDC
	hbm    win.HBITMAP
	old    win.HGDIOBJ
	bits   unsafe.Pointer
	winW   int
	winH   int
}

func (s *gdiCaptureSurface) close() {
	if s.hdcDst != 0 {
		if s.old != 0 {
			win.SelectObject(s.hdcDst, s.old)
		}
		if s.hbm != 0 {
			win.DeleteObject(win.HGDIOBJ(s.hbm))
		}
		win.DeleteDC(s.hdcDst)
	}
	*s = gdiCaptureSurface{}
}

func (s *gdiCaptureSurface) ensure(hwnd win.HWND, winW, winH int) error {
	if s.hdcDst != 0 && s.bits != nil && s.winW == winW && s.winH == winH {
		return nil
	}

	hdcSrc := win.GetDC(hwnd)
	if hdcSrc == 0 {
		return fmt.Errorf("GetDC 失败")
	}
	hdcDst := win.CreateCompatibleDC(hdcSrc)
	if hdcDst == 0 {
		win.ReleaseDC(hwnd, hdcSrc)
		return fmt.Errorf("CreateCompatibleDC 失败")
	}

	bmi := win.BITMAPINFO{
		BmiHeader: win.BITMAPINFOHEADER{
			BiSize:        uint32(unsafe.Sizeof(win.BITMAPINFOHEADER{})),
			BiWidth:       int32(winW),
			BiHeight:      -int32(winH),
			BiPlanes:      1,
			BiBitCount:    32,
			BiCompression: win.BI_RGB,
		},
	}
	var bits unsafe.Pointer
	hbm := win.CreateDIBSection(hdcDst, &bmi.BmiHeader, win.DIB_RGB_COLORS, &bits, 0, 0)
	win.ReleaseDC(hwnd, hdcSrc)
	if hbm == 0 || bits == nil {
		win.DeleteDC(hdcDst)
		return fmt.Errorf("CreateDIBSection 失败")
	}
	old := win.SelectObject(hdcDst, win.HGDIOBJ(hbm))

	// Only discard the previous, still-usable surface after its replacement has
	// been created successfully.
	s.close()
	s.hdcDst = hdcDst
	s.hbm = hbm
	s.old = old
	s.bits = bits
	s.winW = winW
	s.winH = winH
	return nil
}

func (s *gdiCaptureSurface) captureRaw(hwnd win.HWND) (*rawCapture, error) {
	var winRect win.RECT
	if !win.GetWindowRect(hwnd, &winRect) {
		return nil, fmt.Errorf("GetWindowRect 失败")
	}
	winW, winH := int(winRect.Right-winRect.Left), int(winRect.Bottom-winRect.Top)
	if winW <= 0 || winH <= 0 {
		return nil, fmt.Errorf("窗口尺寸无效 %dx%d", winW, winH)
	}

	var clientRect win.RECT
	if !win.GetClientRect(hwnd, &clientRect) {
		return nil, fmt.Errorf("GetClientRect 失败")
	}
	clientW, clientH := int(clientRect.Right-clientRect.Left), int(clientRect.Bottom-clientRect.Top)

	var clientOrigin point
	procClientToScreen.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&clientOrigin)))
	offsetX := int(clientOrigin.X) - int(winRect.Left)
	offsetY := int(clientOrigin.Y) - int(winRect.Top)

	if err := s.ensure(hwnd, winW, winH); err != nil {
		return nil, err
	}
	r, _, lastErr := procPrintWindow.Call(uintptr(hwnd), uintptr(s.hdcDst), uintptr(PW_RENDERFULLCONTENT))
	if r == 0 {
		return nil, fmt.Errorf("PrintWindow 返回 false (lastErr=%v)", lastErr)
	}

	rc := &rawCapture{
		src:     unsafe.Slice((*byte)(s.bits), winW*winH*4),
		winW:    winW,
		winH:    winH,
		clientW: clientW,
		clientH: clientH,
		offsetX: offsetX,
		offsetY: offsetY,
	}
	return rc, nil
}

// gdiFrame 抓一帧，返回完整客户区 RGBA。
func gdiFrame(surface *gdiCaptureSurface, hwnd win.HWND) (*image.RGBA, error) {
	rc, err := surface.captureRaw(hwnd)
	if err != nil {
		return nil, err
	}

	img := image.NewRGBA(image.Rect(0, 0, rc.clientW, rc.clientH))
	for y := range rc.clientH {
		srcY := y + rc.offsetY
		if srcY < 0 || srcY >= rc.winH {
			continue
		}
		srcRowOff := srcY * rc.winW * 4
		dstRowOff := y * rc.clientW * 4
		for x := range rc.clientW {
			srcX := x + rc.offsetX
			if srcX < 0 || srcX >= rc.winW {
				continue
			}
			si := srcRowOff + srcX*4
			di := dstRowOff + x*4
			img.Pix[di+0] = rc.src[si+2]
			img.Pix[di+1] = rc.src[si+1]
			img.Pix[di+2] = rc.src[si+0]
			img.Pix[di+3] = 255
		}
	}
	return img, nil
}

// gdiFrameROI 抓一帧，只转换客户区内指定矩形区域的 BGRA→RGBA。
// 参数为客户区像素坐标。返回图的 (0,0) 对应 (roiX, roiY)。
func gdiFrameROI(surface *gdiCaptureSurface, hwnd win.HWND, roiX, roiY, roiW, roiH int) (*image.RGBA, error) {
	rc, err := surface.captureRaw(hwnd)
	if err != nil {
		return nil, err
	}

	// 裁剪到客户区边界
	if roiX < 0 {
		roiW += roiX
		roiX = 0
	}
	if roiY < 0 {
		roiH += roiY
		roiY = 0
	}
	if roiX+roiW > rc.clientW {
		roiW = rc.clientW - roiX
	}
	if roiY+roiH > rc.clientH {
		roiH = rc.clientH - roiY
	}
	if roiW <= 0 || roiH <= 0 {
		return nil, fmt.Errorf("ROI 无效 %d,%d %dx%d", roiX, roiY, roiW, roiH)
	}

	img := image.NewRGBA(image.Rect(0, 0, roiW, roiH))
	for y := range roiH {
		srcY := y + roiY + rc.offsetY
		if srcY < 0 || srcY >= rc.winH {
			continue
		}
		srcRowOff := srcY * rc.winW * 4
		dstRowOff := y * roiW * 4
		for x := range roiW {
			srcX := x + roiX + rc.offsetX
			if srcX < 0 || srcX >= rc.winW {
				continue
			}
			si := srcRowOff + srcX*4
			di := dstRowOff + x*4
			img.Pix[di+0] = rc.src[si+2]
			img.Pix[di+1] = rc.src[si+1]
			img.Pix[di+2] = rc.src[si+0]
			img.Pix[di+3] = 255
		}
	}
	return img, nil
}

// winClientSize 纯 GetClientRect 客户区像素尺寸 (真窗口). gdi/wgc IBackend.ClientSize 用.
func winClientSize(hwnd win.HWND) (int, int, error) {
	var r win.RECT
	if !win.GetClientRect(hwnd, &r) {
		return 0, 0, fmt.Errorf("GetClientRect 失败")
	}
	return int(r.Right - r.Left), int(r.Bottom - r.Top), nil
}
