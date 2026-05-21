// wire_templates.go 模板相关 wiring: 截屏 adapter.
package main

import (
	"bytes"
	"fmt"
	"image/png"

	"github.com/lxn/win"

	"yhbox/internal/services"
	"yhbox/pkg/capture"
)

// templateCaptureAdapter 实现 template.CaptureAdapter interface.
// 用户在 UI 截"模板"时, 抓当前游戏窗口一帧 → PNG bytes. game window 没就绪则 error.
type templateCaptureAdapter struct {
	app *services.App
}

// Capture 抓游戏窗口一帧, 返 PNG bytes. width/height 调用方从 PNG header 自己读.
func (t *templateCaptureAdapter) Capture() ([]byte, error) {
	g := t.app.Game()
	if g == nil || !g.OK {
		return nil, fmt.Errorf("游戏窗口未就绪, 无法截屏")
	}
	img, err := capture.Frame(win.HWND(g.HWND))
	if err != nil {
		return nil, fmt.Errorf("capture.Frame: %w", err)
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("png.Encode: %w", err)
	}
	return buf.Bytes(), nil
}
