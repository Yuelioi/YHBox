package tools

import (
	"fmt"

	"github.com/yottaapp/yotta/pkg/capture"
	"github.com/yottaapp/yotta/pkg/vision"
)

// win32PixelAt 截当前帧，读光标位置的像素颜色。前端"取色"按钮按一次调一次。
// 太频繁会拖性能（每次都 capture）。
func (s *Service) win32PixelAt(targetSlot string) (PixelInfo, error) {
	sx, sy, err := readCursor()
	if err != nil {
		return PixelInfo{}, err
	}
	wh, hasGame := s.gameWindowFor(targetSlot)
	if !hasGame {
		return PixelInfo{}, fmt.Errorf("游戏窗口未就绪")
	}
	hwnd, cw, ch := wh.HWND, wh.ClientW, wh.ClientH
	cx, cy, err := screenToClient(hwnd, sx, sy)
	if err != nil {
		return PixelInfo{}, err
	}
	if cx < 0 || cy < 0 || cx >= cw || cy >= ch {
		return PixelInfo{OK: false, ClientX: cx, ClientY: cy}, nil
	}
	backendName, err := s.resolver.CaptureBackend(targetSlot)
	if err != nil {
		return PixelInfo{}, err
	}
	backend, _, err := capture.NewIBackend(backendName)
	if err != nil {
		return PixelInfo{}, fmt.Errorf("capture backend: %w", err)
	}
	defer backend.Close()
	frame, err := backend.Frame(capture.Handle(hwnd))
	if err != nil {
		return PixelInfo{}, fmt.Errorf("capture: %w", err)
	}
	bounds := frame.Bounds()
	if cx >= bounds.Dx() || cy >= bounds.Dy() {
		return PixelInfo{OK: false, ClientX: cx, ClientY: cy}, nil
	}
	i := cy*frame.Stride + cx*4
	r, g, b := frame.Pix[i], frame.Pix[i+1], frame.Pix[i+2]
	h, sat, v := vision.RGBToHSV(r, g, b)
	return PixelInfo{
		OK: true, ClientX: cx, ClientY: cy,
		R: int(r), G: int(g), B: int(b),
		H: h, S: sat, V: v,
		Hex: fmt.Sprintf("#%02X%02X%02X", r, g, b),
	}, nil
}
