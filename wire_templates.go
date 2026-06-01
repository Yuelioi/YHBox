// wire_templates.go 模板相关 wiring: 截屏 adapter.
package main

import (
	"bytes"
	"fmt"
	"image/png"

	"github.com/lxn/win"

	"yhbox/internal/services/container"
	"yhbox/pkg/capture"
)

// templateCaptureAdapter 实现 template.CaptureAdapter interface.
// 用户在 UI 截"模板"时, 按 containerID 解析目标窗口抓一帧 → PNG bytes.
// 容器不存在 / 无 WindowTarget / 窗口没开则 error.
type templateCaptureAdapter struct {
	containers *container.Service
}

// Capture 解析 containerID 目标窗口抓一帧, 返 PNG bytes. width/height 调用方从 PNG header 自己读.
func (t *templateCaptureAdapter) Capture(containerID string) ([]byte, error) {
	wh, err := t.containers.ResolveWindow(containerID)
	if err != nil {
		return nil, fmt.Errorf("解析目标窗口: %w", err)
	}
	img, err := capture.Frame(win.HWND(wh.HWND))
	if err != nil {
		return nil, fmt.Errorf("capture.Frame: %w", err)
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("png.Encode: %w", err)
	}
	return buf.Bytes(), nil
}
