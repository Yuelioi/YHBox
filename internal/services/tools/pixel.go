package tools

import (
	"fmt"

	"github.com/lxn/win"

	"yhbox/pkg/capture"
	"yhbox/pkg/vision"
)

// PixelInfo 当前光标下的颜色采样。client 坐标在游戏窗口客户区内。
type PixelInfo struct {
	OK      bool   `json:"ok"`
	ClientX int    `json:"clientX"`
	ClientY int    `json:"clientY"`
	R       int    `json:"r"`
	G       int    `json:"g"`
	B       int    `json:"b"`
	H       int    `json:"h"`
	S       int    `json:"s"`
	V       int    `json:"v"`
	Hex     string `json:"hex"`
}

// PixelAt 截当前帧，读光标位置的像素颜色。前端"取色"按钮按一次调一次。
// 太频繁会拖性能（每次都 capture）。
func (s *Service) PixelAt(containerID string) (PixelInfo, error) {
	sx, sy, ok := readCursor()
	if !ok {
		return PixelInfo{}, fmt.Errorf("GetCursorPos failed")
	}
	wh, hasGame := s.gameWindowFor(containerID)
	if !hasGame {
		return PixelInfo{}, fmt.Errorf("游戏窗口未就绪")
	}
	hwnd, cw, ch := win.HWND(wh.HWND), wh.ClientW, wh.ClientH
	cx, cy, ok2 := screenToClient(hwnd, sx, sy)
	if !ok2 || cx < 0 || cy < 0 || cx >= cw || cy >= ch {
		return PixelInfo{OK: false, ClientX: cx, ClientY: cy}, nil
	}
	frame, err := capture.Frame(hwnd)
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
