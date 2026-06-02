// Package backends 定义 InputClip 回放的注入后端抽象.
package backends

import "yotta/internal/services/inputclip"

// IInputBackend 抽象注入层. ClipPlayer 不直接调 SendInput, 走这个接口.
type IInputBackend interface {
	// Send 投递单个 event. 同步完成. Caller 已经精确 wait 到 target time.
	Send(ev inputclip.Event) error

	// SendBatch 同 tick 多 event 一次投递. backend 决定是否真合批.
	SendBatch(evs []inputclip.Event) error

	// ReleaseHeld 取消时调. 释放所有 backend 认为"按下"的 keys / mouseBtns.
	// ⚠ 防键卡住.
	ReleaseHeld() error

	// Name backend 名 (日志 + UI 显示).
	Name() string
}
