// cmd/mousetest 测哪种方法能让异环游戏的**摄像机/视角**转向。
//
// 背景：游戏读 mouse 输入有 3 种主流方式：
//  1. cursor position (旧 UI / Slate widget)        —— SetCursorPos 起作用
//  2. WM_MOUSEMOVE 消息 (老游戏)                    —— PostMessage 起作用
//  3. Raw Input / DirectInput (FPS / 摄像机控制)    —— SendInput 起作用
//
// 用法：
//
//	# 先把异环开到能转视角的状态（比如野外）
//	go run ./cmd/mousetest setcursor      # 方法 1：SetCursorPos 移光标 delta
//	go run ./cmd/mousetest mouse_event    # 方法 2：user32!mouse_event 旧 API
//	go run ./cmd/mousetest send_input     # 方法 3：SendInput INPUT_MOUSE
//	go run ./cmd/mousetest postmessage    # 方法 4：PostMessage WM_MOUSEMOVE (现有 pkg/input 的方式)
//
// 启动后 3 秒倒计时（用户切回游戏），然后向右扫 20 帧 × dx=15px（共约 300px）。
// 哪个能让相机转就是答案。运行期不会 click，只动光标。
package main

import (
	"fmt"
	"os"
	"syscall"
	"time"
	"unsafe"

	"github.com/lxn/win"

	"yhbox/pkg/winutil"
)

var (
	user32             = syscall.NewLazyDLL("user32.dll")
	procSetCursorPos   = user32.NewProc("SetCursorPos")
	procGetCursorPos   = user32.NewProc("GetCursorPos")
	procMouseEvent     = user32.NewProc("mouse_event")
	procSendInput      = user32.NewProc("SendInput")
	procSetForeground  = user32.NewProc("SetForegroundWindow")
	procPostMessageW   = user32.NewProc("PostMessageW")
	procSendMessageW   = user32.NewProc("SendMessageW")
	procClientToScreen = user32.NewProc("ClientToScreen")
)

const (
	wmActivate    = 0x0006
	waActive      = 1
	wmMouseMove   = 0x0200
	mouseEventfMove        uint32 = 0x0001
	mouseEventfAbsolute    uint32 = 0x8000
	mouseEventfVirtualDesk uint32 = 0x4000

	inputTypeMouse uint32 = 0

	// SendInput MOUSEINPUT flags
	miMoveRelative uint32 = 0x0000 // 默认是相对
	miMoveAbsolute uint32 = 0x8000
)

type point struct{ X, Y int32 }

func getCursor() (int32, int32) {
	var p point
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&p)))
	return p.X, p.Y
}

func setCursor(x, y int32) {
	procSetCursorPos.Call(uintptr(x), uintptr(y))
}

// MOUSEINPUT struct for SendInput
type mouseInput struct {
	Dx        int32
	Dy        int32
	MouseData uint32
	Flags     uint32
	Time      uint32
	ExtraInfo uintptr
}

// INPUT union（mouse 时前面是 type=0，然后 MOUSEINPUT 内嵌）
// Win32 INPUT 结构在 64-bit 下是 type(4) + pad(4) + MOUSEINPUT(24) + tail pad = 40 bytes
type inputUnion struct {
	Type uint32
	_    uint32 // padding on amd64
	Mi   mouseInput
}

func sendInputMouseDelta(dx, dy int32) {
	in := inputUnion{
		Type: inputTypeMouse,
		Mi: mouseInput{
			Dx:    dx,
			Dy:    dy,
			Flags: mouseEventfMove, // 相对移动
		},
	}
	procSendInput.Call(1, uintptr(unsafe.Pointer(&in)), unsafe.Sizeof(in))
}

func mouseEventDelta(dx, dy int32) {
	procMouseEvent.Call(uintptr(mouseEventfMove), uintptr(dx), uintptr(dy), 0, 0)
}

func makeLParam(x, y int32) uintptr {
	return uintptr(uint32(uint16(x)) | uint32(uint16(y))<<16)
}

func postMouseMove(hwnd win.HWND, clientX, clientY int32) {
	procPostMessageW.Call(uintptr(hwnd), wmMouseMove, 0, makeLParam(clientX, clientY))
}

func activateWindow(hwnd win.HWND) {
	procSendMessageW.Call(uintptr(hwnd), wmActivate, uintptr(waActive), 0)
	procSetForeground.Call(uintptr(hwnd))
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: mousetest <setcursor|mouse_event|send_input|postmessage>")
		os.Exit(1)
	}
	method := os.Args[1]

	games := winutil.FindGame()
	if len(games) == 0 {
		fmt.Println("找不到异环窗口；请先启动游戏")
		os.Exit(1)
	}
	g := games[0]
	fmt.Printf("找到窗口: %s (hwnd=%v)\n", g.Title, g.HWND)
	fmt.Println("3 秒后开始向右扫光标。把游戏切到前台 + 鼠标点游戏窗口内...")
	time.Sleep(3 * time.Second)

	activateWindow(g.HWND)
	time.Sleep(100 * time.Millisecond)

	// 把光标定位到屏幕中心（游戏窗口中心更稳）作为起点
	var rect win.RECT
	win.GetClientRect(g.HWND, &rect)
	cx := rect.Right / 2
	cy := rect.Bottom / 2
	sx, sy := clientToScreen(g.HWND, cx, cy)
	setCursor(sx, sy)
	time.Sleep(100 * time.Millisecond)

	const frames = 20
	const dx int32 = 15 // 每帧向右 15 px

	fmt.Printf("方法 [%s]：%d 帧 × dx=%d\n", method, frames, dx)

	for i := 0; i < frames; i++ {
		switch method {
		case "setcursor":
			x, y := getCursor()
			setCursor(x+dx, y)
		case "mouse_event":
			mouseEventDelta(dx, 0)
		case "send_input":
			sendInputMouseDelta(dx, 0)
		case "postmessage":
			// PostMessage WM_MOUSEMOVE 需要 client 坐标（不是 delta）；模拟"客户区内光标位置 +dx"
			postMouseMove(g.HWND, cx+int32(i+1)*dx, cy)
		default:
			fmt.Println("未知方法:", method)
			os.Exit(1)
		}
		time.Sleep(15 * time.Millisecond)
	}

	fmt.Println("完成。看相机有没有向右转。没转 → 这方法对异环无效。")
}

func clientToScreen(hwnd win.HWND, x, y int32) (int32, int32) {
	p := point{X: x, Y: y}
	procClientToScreen.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&p)))
	return p.X, p.Y
}
