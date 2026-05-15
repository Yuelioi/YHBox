// Package services 是 wails3 service 层 + 共享基础设施。
// 所有 service 通过 wails3 binding 暴露 RPC 给前端；事件用 application.Event.Emit 单向推送。
package services

import "time"

// 事件名常量 —— 跟 frontend/src/constants/events.ts 一一对应。
// State 事件（fish:state / cook:state / piano:state / rhythm:state）由 BotKind 派生，
// 见 EventStateName()。此处只列 bot-specific 单独事件。
const (
	EventFishStats     = "fish:stats"
	EventPianoProgress = "piano:progress"
	EventBattleState   = "battle:state"
	EventGameStatus    = "game:status"
	EventLogLines      = "log:lines"
)

// Bot 运行态字符串 —— 跟前端 store 的 'idle'|'running'|'paused' 同步。
const (
	BotStateIdle    = "idle"
	BotStateRunning = "running"
	BotStatePaused  = "paused"
)

// BotStateEvent 是 fish:state / cook:state / piano:state 共用 payload。
type BotStateEvent struct {
	State string `json:"state"` // idle | running | paused
}

// FishStatsEvent 鱼 bot 的实时统计。Phase 2 由 FishService 推。
type FishStatsEvent struct {
	CommonCount int       `json:"commonCount"`
	PurpleCount int       `json:"purpleCount"`
	GoldenCount int       `json:"goldenCount"`
	StartedAt   time.Time `json:"startedAt"`
}

// PianoProgressEvent 钢琴播放进度。Phase 3 由 PianoService 推。
type PianoProgressEvent struct {
	CurMs   int64 `json:"curMs"`
	TotalMs int64 `json:"totalMs"`
}

// BattleStateEvent 战斗 tab 热键启用状态。Phase 3 由 BattleService 推。
type BattleStateEvent struct {
	Enabled bool   `json:"enabled"`
	Mods    string `json:"mods"`
}

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
