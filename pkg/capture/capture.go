// GDI backend: 用 PrintWindow(PW_RENDERFULLCONTENT) 抓客户区帧。
// 调用方走 capture.Frame / FrameROI / NewCapturer 包级入口（backend.go），
// active == BackendGDI 时分发到本文件里的 gdiFrame / gdiFrameROI / newGDICapturer。
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

func captureRaw(hwnd win.HWND) (*rawCapture, func(), error) {
	var winRect win.RECT
	if !win.GetWindowRect(hwnd, &winRect) {
		return nil, nil, fmt.Errorf("GetWindowRect 失败")
	}
	winW, winH := int(winRect.Right-winRect.Left), int(winRect.Bottom-winRect.Top)
	if winW <= 0 || winH <= 0 {
		return nil, nil, fmt.Errorf("窗口尺寸无效 %dx%d", winW, winH)
	}

	var clientRect win.RECT
	if !win.GetClientRect(hwnd, &clientRect) {
		return nil, nil, fmt.Errorf("GetClientRect 失败")
	}
	clientW, clientH := int(clientRect.Right-clientRect.Left), int(clientRect.Bottom-clientRect.Top)

	var clientOrigin point
	procClientToScreen.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&clientOrigin)))
	offsetX := int(clientOrigin.X) - int(winRect.Left)
	offsetY := int(clientOrigin.Y) - int(winRect.Top)

	hdcSrc := win.GetDC(hwnd)
	if hdcSrc == 0 {
		return nil, nil, fmt.Errorf("GetDC 失败")
	}
	hdcDst := win.CreateCompatibleDC(hdcSrc)
	if hdcDst == 0 {
		win.ReleaseDC(hwnd, hdcSrc)
		return nil, nil, fmt.Errorf("CreateCompatibleDC 失败")
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
	if hbm == 0 || bits == nil {
		win.DeleteDC(hdcDst)
		win.ReleaseDC(hwnd, hdcSrc)
		return nil, nil, fmt.Errorf("CreateDIBSection 失败")
	}
	old := win.SelectObject(hdcDst, win.HGDIOBJ(hbm))

	r, _, lastErr := procPrintWindow.Call(uintptr(hwnd), uintptr(hdcDst), uintptr(PW_RENDERFULLCONTENT))
	if r == 0 {
		win.SelectObject(hdcDst, old)
		win.DeleteObject(win.HGDIOBJ(hbm))
		win.DeleteDC(hdcDst)
		win.ReleaseDC(hwnd, hdcSrc)
		return nil, nil, fmt.Errorf("PrintWindow 返回 false (lastErr=%v)", lastErr)
	}

	cleanup := func() {
		win.SelectObject(hdcDst, old)
		win.DeleteObject(win.HGDIOBJ(hbm))
		win.DeleteDC(hdcDst)
		win.ReleaseDC(hwnd, hdcSrc)
	}

	rc := &rawCapture{
		src:     unsafe.Slice((*byte)(bits), winW*winH*4),
		winW:    winW,
		winH:    winH,
		clientW: clientW,
		clientH: clientH,
		offsetX: offsetX,
		offsetY: offsetY,
	}
	return rc, cleanup, nil
}

// gdiFrame 抓一帧，返回完整客户区 RGBA。
func gdiFrame(hwnd win.HWND) (*image.RGBA, error) {
	rc, cleanup, err := captureRaw(hwnd)
	if err != nil {
		return nil, err
	}
	defer cleanup()

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
func gdiFrameROI(hwnd win.HWND, roiX, roiY, roiW, roiH int) (*image.RGBA, error) {
	rc, cleanup, err := captureRaw(hwnd)
	if err != nil {
		return nil, err
	}
	defer cleanup()

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

// ClientSize 返回客户区像素尺寸。mock backend 下走 mock 帧的尺寸，省得用户开游戏。
func ClientSize(hwnd win.HWND) (int, int, error) {
	if active == BackendMock {
		return mockClientSize(hwnd)
	}
	var r win.RECT
	if !win.GetClientRect(hwnd, &r) {
		return 0, 0, fmt.Errorf("GetClientRect 失败")
	}
	return int(r.Right - r.Left), int(r.Bottom - r.Top), nil
}
