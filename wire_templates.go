// wire_templates.go 模板相关 wiring：截屏 provider。
package main

import (
	"bytes"
	"fmt"
	"image/png"

	"github.com/lxn/win"

	"yhbox/internal/services"
	"yhbox/pkg/capture"
)

// ---- CaptureProvider: capture.Frame(gameHWND) → template.CaptureProvider ----
//
// 用户在 UI 截"模板"时，抓当前游戏窗口一帧 → PNG bytes 给前端预览 + 落盘到
// data/assets/templates/。game window 没就绪则 error。

type templateCaptureAdapter struct {
	app *services.App
}

func (t *templateCaptureAdapter) CapturePNG() ([]byte, int, int, error) {
	g := t.app.Game()
	if g == nil || !g.OK {
		return nil, 0, 0, fmt.Errorf("游戏窗口未就绪，无法截屏")
	}
	img, err := capture.Frame(win.HWND(g.HWND))
	if err != nil {
		return nil, 0, 0, fmt.Errorf("capture.Frame: %w", err)
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, 0, 0, fmt.Errorf("png.Encode: %w", err)
	}
	return buf.Bytes(), img.Rect.Dx(), img.Rect.Dy(), nil
}
