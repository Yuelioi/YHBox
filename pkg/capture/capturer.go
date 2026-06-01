package capture

import (
	"fmt"
	"image"
	"sync"
	"unsafe"

	"github.com/lxn/win"
)

// Capturer 是高频抓多 ROI 的会话句柄。GDI/WGC 各自实现。
//
// 调用约定：
//   - Frame() 必须串行调用（同一 capturer 内不并发）
//   - Close() 可从任意 goroutine 调用，重复调用安全
//   - 返回的 imgs[i].Pix 是 capturer 内部复用 buffer，下次 Frame 会被覆盖
type Capturer interface {
	Frame() ([]*image.RGBA, error)
	Close() error
}

// gdiCapturer 是 BackendGDI 下的 Capturer 实现：持有可复用的 GDI 资源
// (full-window hbm) 和 N 个 ROI-size *image.RGBA buffer，给 rhythm 这种
// 120Hz 高频抓多 ROI 的场景用，稳态运行不再 alloc 全窗口 RGBA。
//
// ROI 用客户区坐标。内部 ClientToScreen 算偏移翻成 hbm 全窗口索引。
//
// 调用方走 newGDICapturer(hwnd, rois)，拿到的是 *gdiCapturer (满足 Capturer interface)；
// 这个 struct 不导出。
type gdiCapturer struct {
	hwnd win.HWND
	rois []image.Rectangle

	mu      sync.Mutex
	hdcSrc  win.HDC
	hdcDst  win.HDC
	hbm     win.HBITMAP
	hbmOld  win.HGDIOBJ
	bits    unsafe.Pointer
	pixHBM  []byte // 全窗口 hbm 的 BGRA 原始内存视图
	winW    int    // 构造时记录的窗口尺寸；Frame 时校验不变
	winH    int
	clientW int // 客户区尺寸；Frame 时校验不变
	clientH int
	offsetX int // 客户区原点相对窗口 rect 的偏移（client → window 坐标加这个）
	offsetY int
	imgs    []*image.RGBA // 每个 ROI 一个，Pix 复用
	closed  bool
}

// newGDICapturer 创建并分配 GDI 资源。
// rois 必须非空；每个 ROI 必须落在窗口客户区内，否则返回 error。
func newGDICapturer(hwnd win.HWND, rois []image.Rectangle) (*gdiCapturer, error) {
	if len(rois) == 0 {
		return nil, fmt.Errorf("rois 不能为空")
	}
	c := &gdiCapturer{hwnd: hwnd, rois: append([]image.Rectangle{}, rois...)}
	if err := c.init(); err != nil {
		return nil, err
	}
	return c, nil
}

// init 抓窗口尺寸 + 客户区偏移、分配 hbm、为每个 ROI 分配独立 RGBA buffer。
func (c *gdiCapturer) init() error {
	var winRect win.RECT
	if !win.GetWindowRect(c.hwnd, &winRect) {
		return fmt.Errorf("GetWindowRect 失败")
	}
	w := int(winRect.Right - winRect.Left)
	h := int(winRect.Bottom - winRect.Top)
	if w <= 0 || h <= 0 {
		return fmt.Errorf("窗口尺寸无效 %dx%d", w, h)
	}

	var clientRect win.RECT
	if !win.GetClientRect(c.hwnd, &clientRect) {
		return fmt.Errorf("GetClientRect 失败")
	}
	cw := int(clientRect.Right - clientRect.Left)
	ch := int(clientRect.Bottom - clientRect.Top)
	if cw <= 0 || ch <= 0 {
		return fmt.Errorf("客户区尺寸无效 %dx%d", cw, ch)
	}

	var clientOrigin point
	procClientToScreen.Call(uintptr(c.hwnd), uintptr(unsafe.Pointer(&clientOrigin)))
	offsetX := int(clientOrigin.X) - int(winRect.Left)
	offsetY := int(clientOrigin.Y) - int(winRect.Top)

	// ROI 边界校验（客户区坐标）
	for i, r := range c.rois {
		if r.Min.X < 0 || r.Min.Y < 0 || r.Max.X > cw || r.Max.Y > ch ||
			r.Dx() <= 0 || r.Dy() <= 0 {
			return fmt.Errorf("rois[%d]=%v 越出客户区 %dx%d 或尺寸无效", i, r, cw, ch)
		}
	}

	c.hdcSrc = win.GetDC(c.hwnd)
	if c.hdcSrc == 0 {
		return fmt.Errorf("GetDC 失败")
	}
	c.hdcDst = win.CreateCompatibleDC(c.hdcSrc)
	if c.hdcDst == 0 {
		win.ReleaseDC(c.hwnd, c.hdcSrc)
		c.hdcSrc = 0
		return fmt.Errorf("CreateCompatibleDC 失败")
	}

	bmi := win.BITMAPINFO{
		BmiHeader: win.BITMAPINFOHEADER{
			BiSize:        uint32(unsafe.Sizeof(win.BITMAPINFOHEADER{})),
			BiWidth:       int32(w),
			BiHeight:      -int32(h), // top-down
			BiPlanes:      1,
			BiBitCount:    32,
			BiCompression: win.BI_RGB,
		},
	}
	var bits unsafe.Pointer
	c.hbm = win.CreateDIBSection(c.hdcDst, &bmi.BmiHeader, win.DIB_RGB_COLORS, &bits, 0, 0)
	if c.hbm == 0 || bits == nil {
		win.DeleteDC(c.hdcDst)
		c.hdcDst = 0
		win.ReleaseDC(c.hwnd, c.hdcSrc)
		c.hdcSrc = 0
		return fmt.Errorf("CreateDIBSection 失败")
	}
	c.hbmOld = win.SelectObject(c.hdcDst, win.HGDIOBJ(c.hbm))
	c.bits = bits
	c.pixHBM = unsafe.Slice((*byte)(bits), w*h*4)
	c.winW, c.winH = w, h
	c.clientW, c.clientH = cw, ch
	c.offsetX, c.offsetY = offsetX, offsetY

	c.imgs = make([]*image.RGBA, len(c.rois))
	for i, r := range c.rois {
		c.imgs[i] = &image.RGBA{
			Pix:    make([]byte, r.Dx()*r.Dy()*4),
			Stride: r.Dx() * 4,
			Rect:   r,
		}
	}
	return nil
}

// Close 释放 GDI 资源。多次调用安全。
func (c *gdiCapturer) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true
	c.releaseLocked()
	return nil
}

func (c *gdiCapturer) releaseLocked() {
	if c.hdcDst != 0 && c.hbmOld != 0 {
		win.SelectObject(c.hdcDst, c.hbmOld)
		c.hbmOld = 0
	}
	if c.hbm != 0 {
		win.DeleteObject(win.HGDIOBJ(c.hbm))
		c.hbm = 0
	}
	if c.hdcDst != 0 {
		win.DeleteDC(c.hdcDst)
		c.hdcDst = 0
	}
	if c.hdcSrc != 0 {
		win.ReleaseDC(c.hwnd, c.hdcSrc)
		c.hdcSrc = 0
	}
	c.bits = nil
	c.pixHBM = nil
}

// Frame PrintWindow 抓一帧到 hbm，然后把 N 个 ROI 区域的 BGRA 翻成 RGBA 写到对应 imgs[i].Pix。
// 返回的 []*image.RGBA 跟前几次返回的是同一组对象，Pix 复用。
//
// 窗口尺寸变化 → 返回错误。caller 决定是否重建 Capturer。
func (c *gdiCapturer) Frame() ([]*image.RGBA, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil, fmt.Errorf("capturer closed")
	}

	// 尺寸校验（窗口 + 客户区都校验，任一变化都重建）
	var winRect win.RECT
	if !win.GetWindowRect(c.hwnd, &winRect) {
		return nil, fmt.Errorf("GetWindowRect 失败")
	}
	w := int(winRect.Right - winRect.Left)
	h := int(winRect.Bottom - winRect.Top)
	if w != c.winW || h != c.winH {
		return nil, fmt.Errorf("窗口尺寸变化: 构造时 %dx%d 现在 %dx%d", c.winW, c.winH, w, h)
	}
	var clientRect win.RECT
	if !win.GetClientRect(c.hwnd, &clientRect) {
		return nil, fmt.Errorf("GetClientRect 失败")
	}
	if int(clientRect.Right-clientRect.Left) != c.clientW ||
		int(clientRect.Bottom-clientRect.Top) != c.clientH {
		return nil, fmt.Errorf("客户区尺寸变化: 构造时 %dx%d 现在 %dx%d",
			c.clientW, c.clientH,
			int(clientRect.Right-clientRect.Left), int(clientRect.Bottom-clientRect.Top))
	}

	r, _, lastErr := procPrintWindow.Call(uintptr(c.hwnd), uintptr(c.hdcDst), uintptr(PW_RENDERFULLCONTENT))
	if r == 0 {
		return nil, fmt.Errorf("PrintWindow 失败 (lastErr=%v)", lastErr)
	}

	// 每个 ROI 把对应区域 BGRA → RGBA 拷到 imgs[i].Pix。
	// ROI 是客户区坐标 → 加 offsetX/Y 翻成 hbm 索引（hbm 是全窗口）。
	for i, roi := range c.rois {
		img := c.imgs[i]
		dst := img.Pix
		dstStride := img.Stride
		roiW := roi.Dx()
		for j := 0; j < roi.Dy(); j++ {
			srcY := roi.Min.Y + j + c.offsetY
			srcRowOff := srcY*c.winW*4 + (roi.Min.X+c.offsetX)*4
			dstRowOff := j * dstStride
			src := c.pixHBM[srcRowOff:]
			for x := 0; x < roiW; x++ {
				dst[dstRowOff+x*4+0] = src[x*4+2] // R = B'
				dst[dstRowOff+x*4+1] = src[x*4+1] // G
				dst[dstRowOff+x*4+2] = src[x*4+0] // B = R'
				dst[dstRowOff+x*4+3] = 255
			}
		}
	}
	return c.imgs, nil
}
