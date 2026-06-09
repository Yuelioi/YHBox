// wire_templates.go 模板相关 wiring: 截屏 adapter.
package main

import (
	"bytes"
	"fmt"
	"image/png"

	"github.com/lxn/win"

	"yotta/internal/services/container"
	"yotta/pkg/capture"
)

// templateCaptureAdapter 实现 asset.CaptureAdapter interface.
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
	backend, warning, err := capture.NewIBackend(t.containers.CaptureBackendFor(containerID))
	if err != nil {
		return nil, fmt.Errorf("capture backend: %w", err)
	}
	_ = warning // 制作工具单帧, fallback warning 不冒泡
	defer backend.Close()
	img, err := backend.Frame(win.HWND(wh.HWND))
	if err != nil {
		return nil, fmt.Errorf("capture: %w", err)
	}
	// 窗口最小化/不可见时客户区是 0×0, png.Encode 会报 "invalid image size: 0x0" 这种天书.
	// 提前拦下给人话, 让用户还原窗口重试 (跟录制取不到客户区判失败同思路).
	if b := img.Bounds(); b.Dx() <= 0 || b.Dy() <= 0 {
		return nil, fmt.Errorf("目标窗口客户区为 %d×%d (可能最小化或不可见), 无法截图, 请还原窗口后重试", b.Dx(), b.Dy())
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("png.Encode: %w", err)
	}
	return buf.Bytes(), nil
}
