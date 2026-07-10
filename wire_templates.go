// wire_templates.go 模板相关 wiring: 截屏 adapter.
package main

import (
	"bytes"
	"context"
	"fmt"
	"image/png"
	"time"

	"github.com/lxn/win"

	"github.com/yottaapp/yotta/internal/automation/controller"
	"github.com/yottaapp/yotta/internal/automation/target"
	"github.com/yottaapp/yotta/internal/services/container"
	"github.com/yottaapp/yotta/pkg/capture"
)

// templateCaptureAdapter 实现 asset.CaptureAdapter interface.
// 用户在 UI 截"模板"时, 按 containerID 解析目标窗口抓一帧 → PNG bytes.
// 容器不存在 / 无 Win32WindowTarget / 窗口没开则 error.
type templateCaptureAdapter struct {
	containers *container.Service
	adbRunner  controller.ADBRunner
}

// Capture 解析 containerID 目标窗口抓一帧, 返 PNG bytes. width/height 调用方从 PNG header 自己读.
func (t *templateCaptureAdapter) Capture(containerID, nodeID string) ([]byte, error) {
	tg, err := t.containers.ResolveEditorTargetForNode(containerID, nodeID)
	if err != nil {
		return nil, err
	}
	if tg.Kind == target.KindAndroidADB {
		return t.captureAndroid(tg)
	}
	wh, err := t.containers.ResolveWindowForNode(containerID, nodeID)
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

// Resolution 解析 containerID 目标窗口客户区分辨率 [宽,高], 不截帧 (走 GetClientRect).
// 与 Capture 的帧尺寸同源 (gdiFrame/wgcFrame 都用 GetClientRect 定尺寸), 故可拿来精确匹配变体档.
// 容器不存在 / 无 Win32WindowTarget / 窗口没开 / 客户区为 0 → error.
func (t *templateCaptureAdapter) Resolution(containerID string) ([2]int, error) {
	tg, err := t.containers.ResolveEditorTargetForNode(containerID, "")
	if err != nil {
		return [2]int{}, err
	}
	if tg.Kind == target.KindAndroidADB {
		if tg.Resolution.W <= 0 || tg.Resolution.H <= 0 {
			return [2]int{}, fmt.Errorf("android 目标分辨率无效: %dx%d", tg.Resolution.W, tg.Resolution.H)
		}
		return [2]int{tg.Resolution.W, tg.Resolution.H}, nil
	}
	wh, err := t.containers.ResolveWindow(containerID)
	if err != nil {
		return [2]int{}, fmt.Errorf("解析目标窗口: %w", err)
	}
	if wh.ClientW <= 0 || wh.ClientH <= 0 {
		return [2]int{}, fmt.Errorf("目标窗口客户区为 %d×%d (可能最小化或不可见)", wh.ClientW, wh.ClientH)
	}
	return [2]int{wh.ClientW, wh.ClientH}, nil
}

func (t *templateCaptureAdapter) captureAndroid(tg target.Target) ([]byte, error) {
	ctrl, err := controller.NewAndroidADBController(tg, controller.AndroidADBDeps{Runner: t.adbRunner})
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	frame, err := ctrl.Screenshot(ctx, controller.ScreenshotRequest{Space: target.SpaceAndroidDevice})
	if err != nil {
		return nil, fmt.Errorf("android 截图: %w", err)
	}
	if frame.Image == nil {
		return nil, fmt.Errorf("android 截图为空")
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, frame.Image); err != nil {
		return nil, fmt.Errorf("png.Encode: %w", err)
	}
	return buf.Bytes(), nil
}
