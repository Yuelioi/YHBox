// Package input 定义 admitted Run 使用的 input backend contract。
// package-level 原语 (Click/KeyDown/KeyUp/ReleaseAll... input.go) 由 PostMessageBackend 调。
// Backend 由 automation provider 按 Run 创建和持有。
//
// Backend 必须 stateful — KeyDown/Up/MouseDown/Up 后必须能 ReleaseAll 知道放谁.
// SendInput backend 不是 hwnd-scoped (全局 OS state), 所以 ReleaseAll 不带 hwnd 参数.
package input

// Capabilities 让 runtime / 高级节点判断 backend 能不能干某事.
type Capabilities struct {
	BackgroundInput bool // false = 必须前台窗口 (SendInput); true = 后台也能注入 (PostMessage)
	RelativeMouse   bool // 是否支持原生相对鼠标移动 (RawInput / SendInput=yes, PostMessage 弱)
	GlobalInput     bool // true = 全局 OS 级注入抢其他 app 鼠标 (SendInput); false = 仅 targeted hwnd
}

// Backend 每个 Run 实例化一次；Close 释放资源。
type Backend interface {
	Name() string
	Capabilities() Capabilities

	// 输入操作 — hwnd 是当前 admitted automation target 的窗口。
	// xRatio/yRatio 是 0-1 客户区比例, backend 自己 * ClientSize 拿像素.
	Click(hwnd Handle, xRatio, yRatio float64, button string, durMs int) error
	KeyDown(hwnd Handle, vk string) error
	KeyUp(hwnd Handle, vk string) error
	KeyDownCode(hwnd Handle, vk uint32) error
	KeyUpCode(hwnd Handle, vk uint32) error
	MouseDown(hwnd Handle, xRatio, yRatio float64, button string) error
	MouseUp(hwnd Handle, button string) error
	MouseMoveRel(hwnd Handle, dx, dy, durMs int) error
	Scroll(hwnd Handle, xRatio, yRatio float64, notches int, horizontal bool) error

	// TypeText 向目标窗口注入文本字符串 (unicode, 逐 rune 拆 UTF-16 code unit).
	// postmessage 实现走 PostMessage WM_CHAR 投递到 hwnd (targeted, 后台可用);
	// sendinput 实现走全局 SendInput KEYEVENTF_UNICODE (注入到真实前台焦点窗口, hwnd 忽略).
	TypeText(hwnd Handle, s string) error

	// MoveTo 瞬时把光标移到客户区比例 (xRatio,yRatio) 并发 hover. 运动规划由 controller 模块统一完成.
	MoveTo(hwnd Handle, xRatio, yRatio float64) error
	// CursorRatio 读当前光标在该 hwnd 客户区的比例坐标. 分帧滑动取起点用. client rect 为 0 时返 error.
	CursorRatio(hwnd Handle) (xRatio, yRatio float64, err error)

	// ReleaseAll 释放 backend 跟踪的所有 down 集合；Run stop/panic/cancel 时 defer 调用。
	// 无 hwnd 参数 — SendInput 是全局 OS state, backend 内部知道自己按下了什么.
	ReleaseAll() error

	// Close 释放 backend 持有的 OS 资源. 1.0 PostMessage 返 nil.
	Close() error
}
