package main

import (
	"github.com/lxn/win"

	containerruntime "yotta/internal/services/container/runtime"
	"yotta/pkg/input"
)

// gameProviderAdapter 实现 runtime.GameProvider — 只做跨进程窗口置前.
type gameProviderAdapter struct{}

func newGameProviderAdapter() *gameProviderAdapter { return &gameProviderAdapter{} }

func (a *gameProviderAdapter) BringToForeground(hwnd uintptr) bool {
	return input.BringToForeground(win.HWND(hwnd))
}

var _ containerruntime.GameProvider = (*gameProviderAdapter)(nil)
