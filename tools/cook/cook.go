// Package cook 锤子连点工具。
//
// 锤子在烹饪界面位置固定 → 直接 click 固定坐标，不跑模板匹配。
// 用户切到非烹饪界面就别开 cook（GUI 现在没有自动检测烹饪界面，等用户报需求再加）。
package cook

import (
	"context"
	"time"

	"github.com/lxn/win"
	"github.com/rs/zerolog"

	"yhbox/pkg/capture"
	"yhbox/pkg/input"
	"yhbox/pkg/runctl"
)

// Run 启动锤子连点主循环。
func Run(ctx context.Context, hwnd win.HWND, cfg *Config, logger zerolog.Logger, ctrl runctl.Control) {
	w, h, err := capture.ClientSize(hwnd)
	if err != nil {
		logger.Error().Err(err).Str("tag", "SYSTEM").Msg("获取客户区尺寸失败")
		return
	}
	hammer, ok := PickHammer(w, h)
	if !ok {
		logger.Error().Str("tag", "SYSTEM").Msgf("不支持的分辨率 %dx%d（仅 1920×1080 / 1280×720）", w, h)
		return
	}
	logger.Info().Str("tag", "SYSTEM").Msgf("cook 启用 %dx%d  锤子坐标 (%d,%d)  间隔 %v",
		w, h, hammer.X, hammer.Y, cfg.Interval)

	// KeepAlive：周期 FakeActivate 让游戏 IsActive 保持 true，避免 PostMessage 被丢
	kaCtx, kaCancel := context.WithCancel(ctx)
	defer kaCancel()
	go func() {
		t := time.NewTicker(500 * time.Millisecond)
		defer t.Stop()
		for {
			select {
			case <-kaCtx.Done():
				return
			case <-t.C:
				input.FakeActivate(hwnd)
			}
		}
	}()

	clickCount := 0
	logger.Info().Str("tag", "SYSTEM").Msg("开始连点...")

	for {
		select {
		case <-ctx.Done():
			logger.Info().Str("tag", "STOP").Msgf("用户退出，共点击 %d 次", clickCount)
			return
		default:
		}

		if ctrl.IsPaused() {
			logger.Info().Str("tag", "PAUSE").Msg("已暂停（GUI 点[继续]恢复 / [停止]退出）")
			if !ctrl.WaitUnpause() {
				logger.Info().Str("tag", "STOP").Msgf("用户退出，共点击 %d 次", clickCount)
				return
			}
			logger.Info().Str("tag", "SYSTEM").Msg("恢复运行")
		}

		input.Click(hwnd, hammer.X, hammer.Y, cfg.ClickHold, cfg.ActivateDelay, cfg.CursorSettleDelay)
		clickCount++
		if clickCount%20 == 0 {
			logger.Info().Str("tag", "CLICK").Msgf("已点击 %d 次", clickCount)
		}

		// ctx-aware sleep：Stop / cancel 立刻中断，不必等满 Interval
		select {
		case <-ctx.Done():
			logger.Info().Str("tag", "STOP").Msgf("用户退出，共点击 %d 次", clickCount)
			return
		case <-time.After(cfg.Interval):
		}
	}
}
