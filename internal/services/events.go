// Package services 是 wails3 service 层 + 共享基础设施。
// 所有 service 通过 wails3 binding 暴露 RPC 给前端；事件用 application.Event.Emit 单向推送。
package services

// 事件名常量 —— 跟 frontend/src/constants/events.ts 一一对应。
const (
	EventGameStatus = "game:status"
	EventLogLines   = "log:lines"
)

// GameStatusEvent 游戏窗口检测结果。GameService 主动推。
type GameStatusEvent struct {
	OK    bool   `json:"ok"`
	HWND  uint64 `json:"hwnd"`
	Title string `json:"title"`
	W     int    `json:"w"`
	H     int    `json:"h"`
}

// LogLinesEvent 日志批量推送。
// Seq 单调递增（LogSink.flush 自增），前端检查 seq+1 是否连续，否则告警丢包。
// Lines 每行是 zerolog JSON 输出原文（前端 JSON.parse + 兜底 fallback）。
type LogLinesEvent struct {
	Seq   uint64   `json:"seq"`
	Lines []string `json:"lines"`
}
