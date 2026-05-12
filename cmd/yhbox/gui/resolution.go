package gui

import (
	"fmt"

	"github.com/lxn/win"

	"yhbox/pkg/capture"
	"yhbox/pkg/winutil"
)

// GameStatus 描述游戏窗口检测结果。
type GameStatus struct {
	HWND   win.HWND
	Width  int
	Height int
	OK     bool
	Title  string
}

// String 给 label 显示用。
func (s GameStatus) String() string {
	if !s.OK {
		return "未检测到异环窗口，请先开游戏后点[重新检测]"
	}
	return fmt.Sprintf("状态: %dx%d ✓ %s", s.Width, s.Height, s.Title)
}

// DetectGameWindow 枚举窗口找异环，并抓一帧获取客户区分辨率。
// 没找到或抓帧失败 → OK=false。
func DetectGameWindow() GameStatus {
	targets := winutil.FindGame()
	if len(targets) == 0 {
		return GameStatus{}
	}
	tgt := targets[0]

	frame, err := capture.Frame(tgt.HWND)
	if err != nil {
		return GameStatus{}
	}
	b := frame.Bounds()
	return GameStatus{
		HWND:   tgt.HWND,
		Width:  b.Dx(),
		Height: b.Dy(),
		OK:     true,
		Title:  tgt.Title,
	}
}
