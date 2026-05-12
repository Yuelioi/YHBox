// Package cook 锤子连点工具。
package cook

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/lxn/win"

	"yhbox"
	"yhbox/pkg/capture"
	"yhbox/pkg/input"
	"yhbox/pkg/log"
	"yhbox/pkg/runctl"
	"yhbox/pkg/vision"
)

// Run 启动锤子连点主循环。
func Run(ctx context.Context, hwnd win.HWND, cfg *Config, logger *log.Logger, ctrl runctl.Control) {
	// 1. 加载 templates.toml
	tomlBytes, err := yhbox.AssetBytes("templates.toml")
	if err != nil {
		logger.Log(log.SYSTEM, "读 embed templates.toml 失败: %v", err)
		return
	}
	tplsCfg, warns, err := vision.LoadTemplatesConfig(tomlBytes)
	if err != nil {
		logger.Log(log.SYSTEM, "解析 templates.toml 失败: %v", err)
		return
	}
	for _, w := range warns {
		logger.Log(log.SYSTEM, "WARN embed templates: %s", w)
	}

	// 2. 可选：扫 templates/cook.toml 追加用户模板
	userPath := filepath.Join("templates", "cook.toml")
	if userBytes, err := os.ReadFile(userPath); err == nil {
		uwarns, uerr := tplsCfg.MergeUserOverrides("cook", userBytes)
		if uerr != nil {
			logger.Log(log.SYSTEM, "WARN 解析 %s: %v（外挂层失效）", userPath, uerr)
		}
		for _, w := range uwarns {
			logger.Log(log.SYSTEM, "WARN user templates: %s", w)
		}
	}

	hammerSpecs := tplsCfg["cook"]["hammer"]
	if len(hammerSpecs) == 0 {
		logger.Log(log.SYSTEM, "没有 hammer 模板候选（templates.toml 里 [cook.hammer] 为空）")
		return
	}

	// 3. 抓一帧获取分辨率，按 H 挑最匹配的 spec
	firstFrame, ferr := capture.Frame(hwnd)
	if ferr != nil {
		logger.Log(log.SYSTEM, "抓帧失败: %v", ferr)
		return
	}
	frameW, frameH := firstFrame.Bounds().Dx(), firstFrame.Bounds().Dy()
	bestSpec := pickBestSpec(hammerSpecs, frameW, frameH)
	if bestSpec == nil {
		logger.Log(log.SYSTEM, "无可用 hammer 模板（spec 为空）")
		return
	}

	// 4. 加载 PNG + 灰度化
	pngBytes, err := vision.LoadPNGForSpec(*bestSpec)
	if err != nil {
		logger.Log(log.SYSTEM, "加载 hammer PNG 失败: %v", err)
		return
	}
	tpl, err := vision.LoadPNG(bytes.NewReader(pngBytes))
	if err != nil {
		logger.Log(log.SYSTEM, "解码 hammer PNG 失败: %v", err)
		return
	}

	// 按当前 H 缩放（如果 spec 不是 native H）
	if bestSpec.Resolution[1] != frameH {
		scale := float32(frameH) / float32(bestSpec.Resolution[1])
		if scaled := vision.ScaleTemplate(tpl, scale); scaled != nil {
			tpl = scaled
		}
	}
	logger.Log(log.SYSTEM, "锤子模板: %dx%d (file=%s, baseRes=%v)",
		tpl.W, tpl.H, bestSpec.File, bestSpec.Resolution)

	// 5. KeepAlive goroutine（保留之前的修复）
	kaCtx, kaCancel := context.WithCancel(ctx)
	defer kaCancel()
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-kaCtx.Done():
				return
			case <-ticker.C:
				input.FakeActivate(hwnd)
			}
		}
	}()

	clickCount := 0
	missCount := 0

	logger.Log(log.SYSTEM, "开始检测...")
	for {
		select {
		case <-ctx.Done():
			logger.Log(log.STOP, "用户退出，共点击 %d 次", clickCount)
			return
		default:
		}

		if ctrl.IsPaused() {
			logger.Log(log.PAUSE, "已暂停（GUI 点[继续]恢复 / [停止]退出）")
			if !ctrl.WaitUnpause() {
				logger.Log(log.STOP, "用户退出，共点击 %d 次", clickCount)
				return
			}
			logger.Log(log.SYSTEM, "恢复运行")
			missCount = 0
			continue
		}

		frame, err := capture.Frame(hwnd)
		if err != nil {
			logger.Log(log.MISS, "抓帧失败: %v，重试中", err)
			time.Sleep(cfg.Interval)
			continue
		}

		roiGray, roiX, roiY, roiW, roiH := vision.CropROI(frame,
			cfg.HammerROI[0], cfg.HammerROI[1], cfg.HammerROI[2], cfg.HammerROI[3])

		x, y, conf := vision.MatchFast(roiGray, roiW, roiH, tpl, vision.DefaultParallel())

		if conf >= cfg.ConfGate {
			missCount = 0
			cx := roiX + x + tpl.W/2
			cy := roiY + y + tpl.H/2

			input.Click(hwnd, cx, cy, cfg.ClickHold, cfg.ActivateDelay, cfg.CursorSettleDelay)
			clickCount++
			logger.Log(log.CLICK, "#%d  conf=%.3f  pos=(%d,%d)", clickCount, conf, cx, cy)
		} else {
			missCount++
			logger.Log(log.MISS, "conf=%.3f  miss=%d/%d", conf, missCount, cfg.MaxMiss)
			if missCount >= cfg.MaxMiss {
				logger.Log(log.PAUSE, "连续 %d 次未检测到锤子，暂停（GUI 点[继续]恢复 / [停止]退出）", cfg.MaxMiss)
				ctrl.Pause()
			}
		}

		if missCount > 0 {
			time.Sleep(cfg.MissInterval)
		} else {
			time.Sleep(cfg.Interval)
		}
	}
}

// pickBestSpec 在 specs 候选里挑 H 最接近 frameH 的 spec（aspect 优先）。
// 不复用 vision.PickBest 是因为 PickBest 接收 *NamedTemplate，cook 这里只有 spec。
func pickBestSpec(specs []vision.TemplateSpec, frameW, frameH int) *vision.TemplateSpec {
	if len(specs) == 0 {
		return nil
	}
	frameAspect := float64(frameW) / float64(frameH)
	bestIdx := 0
	bestScore := scoreSpec(specs[0], frameH, frameAspect)
	for i := 1; i < len(specs); i++ {
		s := scoreSpec(specs[i], frameH, frameAspect)
		if s > bestScore {
			bestScore = s
			bestIdx = i
		}
	}
	return &specs[bestIdx]
}

func scoreSpec(s vision.TemplateSpec, frameH int, frameAspect float64) float64 {
	// aspect 一致 + 1000；H 越接近分越高（用 -|diff| 让大的更优）
	specAspect := float64(s.Resolution[0]) / float64(s.Resolution[1])
	aspectMatch := 0.0
	if absF(specAspect-frameAspect)/frameAspect < 0.05 {
		aspectMatch = 1000.0
	}
	hDiff := s.Resolution[1] - frameH
	if hDiff < 0 {
		hDiff = -hDiff
	}
	return aspectMatch - float64(hDiff)
}

func absF(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
