// Package input/backend 定义 Backend 接口给 ContainerRunner per-container 选 input impl.
// package-level 原语 (Click/KeyDown/KeyUp/ReleaseAll... input.go) 由 PostMessageBackend 调.
// Backend interface 是 ContainerRunner 唯一通路.
//
// Backend 必须 stateful — KeyDown/Up/MouseDown/Up 后必须能 ReleaseAll 知道放谁.
// SendInput backend 不是 hwnd-scoped (全局 OS state), 所以 ReleaseAll 不带 hwnd 参数.
package input

import (
	"fmt"

	"github.com/lxn/win"
)

// Capabilities 让 runtime / 高级节点判断 backend 能不能干某事.
// 当前只 PostMessage backend, 都填 {true, false, false}; 加 SendInput backend 后这些字段才有差异.
type Capabilities struct {
	BackgroundInput bool // false = 必须前台窗口 (SendInput); true = 后台也能注入 (PostMessage)
	RelativeMouse   bool // 是否支持原生相对鼠标移动 (RawInput / SendInput=yes, PostMessage 弱)
	GlobalInput     bool // true = 全局 OS 级注入抢其他 app 鼠标 (SendInput); false = 仅 targeted hwnd
}

// Backend per-container 实例化, 一个 run 一个. Close 释放资源.
type Backend interface {
	Name() string
	Capabilities() Capabilities

	// 输入操作 — hwnd 是当前 container 的目标窗口.
	// xRatio/yRatio 是 0-1 客户区比例, backend 自己 * ClientSize 拿像素.
	Click(hwnd win.HWND, xRatio, yRatio float64, button string, durMs int) error
	KeyDown(hwnd win.HWND, vk string) error
	KeyUp(hwnd win.HWND, vk string) error
	MouseDown(hwnd win.HWND, xRatio, yRatio float64, button string) error
	MouseUp(hwnd win.HWND, button string) error
	MouseMoveRel(hwnd win.HWND, dx, dy, durMs int) error
	Scroll(hwnd win.HWND, xRatio, yRatio float64, notches int) error

	// MoveTo 瞬时把光标移到客户区比例 (xRatio,yRatio) 并发 hover. 无 sleep —— 分帧由 caller(节点层) 控.
	MoveTo(hwnd win.HWND, xRatio, yRatio float64) error
	// CursorRatio 读当前光标在该 hwnd 客户区的比例坐标. 分帧滑动取起点用. client rect 为 0 时返 error.
	CursorRatio(hwnd win.HWND) (xRatio, yRatio float64, err error)

	// ReleaseAll 释放 backend 跟踪的所有 down 集合. ContainerRunner stop/panic/cancel 时 defer 调.
	// 无 hwnd 参数 — SendInput 是全局 OS state, backend 内部知道自己按下了什么.
	ReleaseAll() error

	// Close 释放 backend 持有的 OS 资源. 1.0 PostMessage 返 nil.
	Close() error
}

// NewBackend 按 name 拿 impl. 1.0 ship 只有 "postmessage".
func NewBackend(name string) (Backend, error) {
	switch name {
	case "", "postmessage":
		return newPostMessageBackend(), nil
	case "sendinput":
		return newSendInputBackend(), nil
	default:
		return nil, fmt.Errorf("unknown input backend %q (supported: postmessage, sendinput)", name)
	}
}
