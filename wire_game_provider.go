package main

import (
	"github.com/lxn/win"

	"yhbox/internal/services"
	containerruntime "yhbox/internal/services/container/runtime"
	"yhbox/pkg/input"
)

// gameProviderAdapter 把 services.App.Game() 适配到 container/runtime.GameProvider。
// Plan A Task 1.15 留 nil 占位；Plan B Task E.10 真实接入：
//   - HWND() 取当前游戏窗口句柄
//   - BringToForeground 调 pkg/input.BringToForeground（含 ShowWindow restore + SetForegroundWindow）
type gameProviderAdapter struct {
	app *services.App
}

func newGameProviderAdapter(app *services.App) *gameProviderAdapter {
	return &gameProviderAdapter{app: app}
}

func (a *gameProviderAdapter) HWND() (uintptr, bool) {
	g := a.app.Game()
	if g == nil || !g.OK {
		return 0, false
	}
	return uintptr(g.HWND), true
}

func (a *gameProviderAdapter) BringToForeground(hwnd uintptr) bool {
	return input.BringToForeground(win.HWND(hwnd))
}

var _ containerruntime.GameProvider = (*gameProviderAdapter)(nil)
