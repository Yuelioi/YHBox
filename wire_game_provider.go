package main

import (
	containerruntime "github.com/yottaapp/yotta/internal/services/container/runtime"
	"github.com/yottaapp/yotta/pkg/winutil"
)

// gameProviderAdapter 实现 runtime.GameProvider — 只做跨进程窗口置前.
type gameProviderAdapter struct{}

func newGameProviderAdapter() *gameProviderAdapter { return &gameProviderAdapter{} }

func (a *gameProviderAdapter) BringToForeground(hwnd uintptr) bool {
	return winutil.BringToFront(hwnd) == nil
}

var _ containerruntime.GameProvider = (*gameProviderAdapter)(nil)
