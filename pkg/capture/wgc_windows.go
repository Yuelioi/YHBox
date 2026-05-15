// WGC backend: syscall 调 capture_wgc.dll（Rust 编出来的 cdylib）。
//
// DLL 暴露 4 个 C 函数（详见 native/capture_wgc/src/lib.rs）：
//
//	uint64_t wgc_session_open(hwnd, *out_w, *out_h)
//	int32_t  wgc_session_grab(sid, buf, buf_len, *out_w, *out_h)
//	void     wgc_session_close(sid)
//	const char* wgc_last_error()
//
// 每个 wgcCapturer 自己开一个 session（HashMap 在 DLL 内部）。多 bot 同时跑
// 各持各的 sid，互不干扰。
//
// 文件名带 _windows.go 因为 syscall.LazyDLL 是 Windows 专属。本项目本来就只跑
// Windows，保留这个 suffix 让 go vet / cross-compile 不误报。

package capture

import (
	"fmt"
	"image"
	"sync"
	"syscall"
	"unsafe"

	"github.com/lxn/win"
)

// DLL 错误码常量，跟 Rust 端对齐（参见 native/capture_wgc/src/lib.rs）
const (
	wgcOK             = 0
	wgcErrBufTooSmall = -1
	wgcErrNotReady    = -2
	// < -10 是真错误，统一当 fatal 处理
)

var (
	wgcDLL          *syscall.LazyDLL
	wgcSessionOpen  *syscall.LazyProc
	wgcSessionGrab  *syscall.LazyProc
	wgcSessionClose *syscall.LazyProc
	wgcLastError    *syscall.LazyProc
	wgcLoadOnce     sync.Once
	wgcLoadErr      error
)

// initWGC 加载 DLL 并解析符号；多次调用幂等。
// 加载失败原因通常是 DLL 不在 exe 同目录（task build 走了但 dll 没拷到 bin/）。
func initWGC() error {
	wgcLoadOnce.Do(func() {
		// LazyDLL 按 Windows 搜索路径找 capture_wgc.dll：
		//   1. exe 同目录
		//   2. PATH
		// 在 dev mode (wails3 dev) 下 cwd 通常是项目根，那时 dll 也在 bin/，
		// 需要 PATH 或者手动加 bin/。但 dev mode 一般用不到 WGC，先不管。
		wgcDLL = syscall.NewLazyDLL("capture_wgc.dll")
		if err := wgcDLL.Load(); err != nil {
			wgcLoadErr = fmt.Errorf("加载 capture_wgc.dll 失败: %w (确认 bin/capture_wgc.dll 存在，且 cwd 跟 exe 同目录)", err)
			return
		}
		wgcSessionOpen = wgcDLL.NewProc("wgc_session_open")
		wgcSessionGrab = wgcDLL.NewProc("wgc_session_grab")
		wgcSessionClose = wgcDLL.NewProc("wgc_session_close")
		wgcLastError = wgcDLL.NewProc("wgc_last_error")
		// LazyProc.Find 主动解析符号，没找到的话第一次 Call 才报错
		for _, p := range []*syscall.LazyProc{wgcSessionOpen, wgcSessionGrab, wgcSessionClose, wgcLastError} {
			if err := p.Find(); err != nil {
				wgcLoadErr = fmt.Errorf("找不到导出 %s: %w", p.Name, err)
				return
			}
		}
	})
	return wgcLoadErr
}

// dllLastError 读 wgc_last_error() 返回的 C 字符串。线程本地，Go-syscall 走的
// 是当前 OS 线程，多数情况能拿到对应调用的错误（除非中间 goroutine 切换了线程，
// 但 syscall 期间 runtime 不会切线程，所以稳）。
func dllLastError() string {
	if wgcLastError == nil {
		return "wgc dll not loaded"
	}
	r, _, _ := wgcLastError.Call()
	if r == 0 {
		return ""
	}
	return cstrToGo(unsafe.Pointer(r))
}

func cstrToGo(p unsafe.Pointer) string {
	if p == nil {
		return ""
	}
	b := (*[1 << 20]byte)(p)
	n := 0
	for n < len(b) && b[n] != 0 {
		n++
	}
	return string(b[:n])
}

// wgcSession 是单个 HWND 的捕获会话。
type wgcSession struct {
	sid    uint64
	hwnd   win.HWND
	width  int32 // 上一次 grab 看到的尺寸；可能变（窗口 resize）
	height int32
	buf    []byte // BGRA 缓冲；需要时按 width*height*4 扩
}

func openWGCSession(hwnd win.HWND) (*wgcSession, error) {
	if err := initWGC(); err != nil {
		return nil, err
	}
	var w, h int32
	r, _, _ := wgcSessionOpen.Call(
		uintptr(hwnd),
		uintptr(unsafe.Pointer(&w)),
		uintptr(unsafe.Pointer(&h)),
	)
	if r == 0 {
		return nil, fmt.Errorf("wgc_session_open: %s", dllLastError())
	}
	// session_open 即使主流程成功，SetIsBorderRequired / SetIsCursorCaptureEnabled
	// 可能在 Win10 老版本上返回 Err 被 rust 端记到 last_err。这里读出来挂到包级
	// LastInitWarning，main.go 启动期检查并通过 zerolog 写到日志面板，方便诊断
	// "黄框为何关不掉"。
	if extra := dllLastError(); extra != "" && LastInitWarning == "" {
		LastInitWarning = extra
	}
	return &wgcSession{
		sid:    uint64(r),
		hwnd:   hwnd,
		width:  w,
		height: h,
	}, nil
}

// grab 抓最新帧到 s.buf；返回当前帧 W/H。
// 处理 ERR_BUF_TOO_SMALL / ERR_NOT_READY 自动重试一次。
func (s *wgcSession) grab() (w, h int, _ error) {
	for attempt := 0; attempt < 2; attempt++ {
		if cap(s.buf) < int(s.width)*int(s.height)*4 {
			s.buf = make([]byte, int(s.width)*int(s.height)*4)
		}
		var bufPtr *byte
		if len(s.buf) > 0 {
			bufPtr = &s.buf[0]
		}
		var outW, outH int32
		r, _, _ := wgcSessionGrab.Call(
			uintptr(s.sid),
			uintptr(unsafe.Pointer(bufPtr)),
			uintptr(uint32(cap(s.buf))),
			uintptr(unsafe.Pointer(&outW)),
			uintptr(unsafe.Pointer(&outH)),
		)
		ret := int32(r)
		switch ret {
		case wgcOK:
			s.width, s.height = outW, outH
			return int(outW), int(outH), nil
		case wgcErrBufTooSmall:
			s.width, s.height = outW, outH
			s.buf = make([]byte, int(outW)*int(outH)*4)
			continue // 重试
		case wgcErrNotReady:
			return 0, 0, errWGCNotReady
		default:
			return 0, 0, fmt.Errorf("wgc_session_grab=%d: %s", ret, dllLastError())
		}
	}
	return 0, 0, fmt.Errorf("wgc_session_grab: buf 重分配后仍不够")
}

func (s *wgcSession) close() {
	if s.sid != 0 {
		wgcSessionClose.Call(uintptr(s.sid))
		s.sid = 0
	}
}

// errWGCNotReady 表示 WGC frame pool 当前没有新帧。caller 可短暂 sleep 后重试。
var errWGCNotReady = fmt.Errorf("wgc: 暂无新帧（重试）")

// wgcBGRAtoRGBA 把 BGRA 行连续字节流转成 *image.RGBA（顺便从全帧裁出 ROI）。
// src 行无 padding（Rust 端已经 pitch→stride 拷贝过了）。
func wgcBGRAtoRGBA(src []byte, srcW, srcH, roiX, roiY, roiW, roiH int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, roiW, roiH))
	for y := 0; y < roiH; y++ {
		sy := y + roiY
		if sy < 0 || sy >= srcH {
			continue
		}
		srcRow := sy * srcW * 4
		dstRow := y * roiW * 4
		for x := 0; x < roiW; x++ {
			sx := x + roiX
			if sx < 0 || sx >= srcW {
				continue
			}
			si := srcRow + sx*4
			di := dstRow + x*4
			img.Pix[di+0] = src[si+2]
			img.Pix[di+1] = src[si+1]
			img.Pix[di+2] = src[si+0]
			img.Pix[di+3] = 255
		}
	}
	return img
}

// wgcFrame: 一次性抓帧，包级 Frame() 走这里。
// 为了简单 / 跟 GDI 行为对齐，每次调用都开-抓-关一个 session。
// 高频路径（rhythm 等）走 newWGCCapturer 复用 session。
func wgcFrame(hwnd win.HWND) (*image.RGBA, error) {
	s, err := openWGCSession(hwnd)
	if err != nil {
		return nil, err
	}
	defer s.close()
	w, h, err := s.grabWithRetry()
	if err != nil {
		return nil, err
	}
	return wgcBGRAtoRGBA(s.buf, w, h, 0, 0, w, h), nil
}

// wgcFrameROI 同 gdiFrameROI 语义：抓全帧后裁 ROI。
// 没用 WGC 的 dirty region 之类的，因为 windows-rs 没暴露简单 API。
func wgcFrameROI(hwnd win.HWND, roiX, roiY, roiW, roiH int) (*image.RGBA, error) {
	s, err := openWGCSession(hwnd)
	if err != nil {
		return nil, err
	}
	defer s.close()
	w, h, err := s.grabWithRetry()
	if err != nil {
		return nil, err
	}
	// 裁剪 ROI 到帧内
	if roiX < 0 {
		roiW += roiX
		roiX = 0
	}
	if roiY < 0 {
		roiH += roiY
		roiY = 0
	}
	if roiX+roiW > w {
		roiW = w - roiX
	}
	if roiY+roiH > h {
		roiH = h - roiY
	}
	if roiW <= 0 || roiH <= 0 {
		return nil, fmt.Errorf("ROI 无效")
	}
	return wgcBGRAtoRGBA(s.buf, w, h, roiX, roiY, roiW, roiH), nil
}

// grabWithRetry: 处理 ERR_NOT_READY 的轮询重试。WGC 启动后第一帧可能要等
// 一两次 vsync（游戏 60fps 时 ~17ms 一帧），bot 第一次抓帧偶尔要更久。
// 200 次 × 2ms = 400ms 给足缓冲；正常运行时 TryGetNextFrame 几乎瞬时成功。
func (s *wgcSession) grabWithRetry() (w, h int, _ error) {
	for i := 0; i < 200; i++ {
		w, h, err := s.grab()
		if err == nil {
			return w, h, nil
		}
		if err != errWGCNotReady {
			return 0, 0, err
		}
		sleepMs(2)
	}
	return 0, 0, fmt.Errorf("wgc: 等帧超时 400ms（窗口最小化或未渲染？）")
}

// sleepMs 给 grabWithRetry 用。不引 time 避免循环 import 的潜在风险（捕获是低层）。
var procSleep = syscall.NewLazyDLL("kernel32.dll").NewProc("Sleep")

func sleepMs(ms uint32) { procSleep.Call(uintptr(ms)) }

// wgcCapturer: BackendWGC 下的 Capturer 实现。
// keep session open + ROI 列表，每次 Frame() 抓全帧后裁 N 个 ROI。
//
// 注意 ROI 坐标语义：跟 gdi 一样用客户区坐标。WGC 抓到的就是客户区帧（不含
// 非客户区如标题栏），所以不需要 offsetX/Y 翻译。
type wgcCapturer struct {
	mu      sync.Mutex
	session *wgcSession
	rois    []image.Rectangle
	imgs    []*image.RGBA
	closed  bool
}

func newWGCCapturer(hwnd win.HWND, rois []image.Rectangle) (*wgcCapturer, error) {
	if len(rois) == 0 {
		return nil, fmt.Errorf("rois 不能为空")
	}
	s, err := openWGCSession(hwnd)
	if err != nil {
		return nil, err
	}
	// ROI 边界校验（WGC session 的 width/height 是开 session 时的客户区尺寸）
	cw, ch := int(s.width), int(s.height)
	for i, r := range rois {
		if r.Min.X < 0 || r.Min.Y < 0 || r.Max.X > cw || r.Max.Y > ch ||
			r.Dx() <= 0 || r.Dy() <= 0 {
			s.close()
			return nil, fmt.Errorf("rois[%d]=%v 越出客户区 %dx%d 或尺寸无效", i, r, cw, ch)
		}
	}
	c := &wgcCapturer{
		session: s,
		rois:    append([]image.Rectangle{}, rois...),
		imgs:    make([]*image.RGBA, len(rois)),
	}
	for i, r := range rois {
		c.imgs[i] = &image.RGBA{
			Pix:    make([]byte, r.Dx()*r.Dy()*4),
			Stride: r.Dx() * 4,
			Rect:   r,
		}
	}
	return c, nil
}

func (c *wgcCapturer) Frame() ([]*image.RGBA, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil, fmt.Errorf("capturer closed")
	}
	w, h, err := c.session.grabWithRetry()
	if err != nil {
		return nil, err
	}
	for i, roi := range c.rois {
		img := c.imgs[i]
		dst := img.Pix
		dstStride := img.Stride
		roiW := roi.Dx()
		roiH := roi.Dy()
		for y := 0; y < roiH; y++ {
			sy := roi.Min.Y + y
			if sy < 0 || sy >= h {
				continue
			}
			srcRow := sy*w*4 + roi.Min.X*4
			dstRow := y * dstStride
			src := c.session.buf[srcRow:]
			for x := 0; x < roiW; x++ {
				dst[dstRow+x*4+0] = src[x*4+2]
				dst[dstRow+x*4+1] = src[x*4+1]
				dst[dstRow+x*4+2] = src[x*4+0]
				dst[dstRow+x*4+3] = 255
			}
		}
	}
	return c.imgs, nil
}

func (c *wgcCapturer) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true
	c.session.close()
	return nil
}
