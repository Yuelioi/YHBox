// Package fish 钓鱼工具：状态机驱动的全自动钓鱼。
package fish

import (
	"context"

	"github.com/lxn/win"

	"yhbox/pkg/log"
	"yhbox/pkg/runctl"
)

type Logger = log.Logger

// Run 启动钓鱼状态机主循环。
// statsHook 在 stats 每次变化时同步调用（在 bot goroutine 上）；可传 nil。
func Run(ctx context.Context, hwnd win.HWND, cfg *Config, logger *Logger, ctrl runctl.Control, statsHook StatsHook) Stats {
	det, err := NewDetector()
	if err != nil {
		logger.Log(log.SYSTEM, "Detector 初始化失败: %v", err)
		return Stats{}
	}

	m := newMachine(hwnd, cfg, det, logger, ctrl, statsHook)
	return m.run(ctx)
}
