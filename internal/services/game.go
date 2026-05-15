package services

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

// DetectGameWindow 枚举窗口找异环并读客户区尺寸。
// 没找到 → OK=false。**只用 GetClientRect，不抓帧**——抓帧依赖 capture backend
// 是否就绪（WGC 第一帧需要等 vsync，刚启动游戏时可能 timeout），而我们这里只
// 想报告"窗口在不在 + 分辨率多少"，没必要走截屏路径。
func DetectGameWindow() GameStatus {
	targets := winutil.FindGame()
	if len(targets) == 0 {
		return GameStatus{}
	}
	tgt := targets[0]

	w, h, err := capture.ClientSize(tgt.HWND)
	if err != nil {
		return GameStatus{}
	}
	return GameStatus{
		HWND:   tgt.HWND,
		Width:  w,
		Height: h,
		OK:     true,
		Title:  tgt.Title,
	}
}

// ToEvent 给 GameService.Detect emit 用。把内部 win.HWND 转 uint64。
func (g GameStatus) ToEvent() GameStatusEvent {
	return GameStatusEvent{
		OK:    g.OK,
		HWND:  uint64(g.HWND),
		Title: g.Title,
		W:     g.Width,
		H:     g.Height,
	}
}
