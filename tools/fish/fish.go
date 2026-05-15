// Package fish 钓鱼工具：状态机驱动的全自动钓鱼。
package fish

import (
	"context"

	"github.com/lxn/win"
	"github.com/rs/zerolog"

	"yhbox/pkg/runctl"
)

// Run 启动钓鱼状态机主循环。
// statsHook 在 stats 每次变化时同步调用（在 bot goroutine 上）；可传 nil。
func Run(ctx context.Context, hwnd win.HWND, cfg *Config, logger zerolog.Logger, ctrl runctl.Control, statsHook StatsHook) Stats {
	det, err := NewDetector()
	if err != nil {
		logger.Error().Err(err).Str("tag", "SYSTEM").Msg("Detector 初始化失败")
		return Stats{}
	}

	m := newMachine(hwnd, cfg, det, logger, ctrl, statsHook)
	return m.run(ctx)
}
