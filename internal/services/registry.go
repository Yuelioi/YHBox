package services

import "github.com/wailsapp/wails/v3/pkg/application"

// BotKind 是强类型 bot 标识，替代分散在各处的 string literal ("fish", "rhythm" 等)。
// 仅长跑型 bot（fish / cook / piano / rhythm）走 BotKind + registry。
// battle 是 hotkey-driven 形态不同，保留原 BattleService 单独 wire。
type BotKind string

const (
	KindFish   BotKind = "fish"
	KindCook   BotKind = "cook"
	KindPiano  BotKind = "piano"
	KindRhythm BotKind = "rhythm"
)

// BotService 是所有长跑型 bot service 必须实现的接口。
// App.Shutdown 通过它统一停所有 bot。
type BotService interface {
	StopSync()
}

// BotMeta 是注册一个长跑型 bot 需要的最小元信息。
//
// Construct 返回两个东西：
//   - BotService：给 App 注册到 Shutdown 队列
//   - application.Service：给 wails3 Services 列表
//
// 之所以一个 Construct 返回两份，是因为 application.NewService[T] 需要具体类型推断，
// 不能接 interface（BotService）。在 init() 闭包里调时 T 是已知的具体 service 类型。
type BotMeta struct {
	Kind      BotKind
	Construct func(*App) (BotService, application.Service)
}

// botRegistry 全局注册表，由各 service 文件的 init() 写入。
// 顺序无关；main.go 遍历构造、注册事件、加入 application.Services 列表。
var botRegistry []BotMeta

// RegisterBot 把 bot 元信息加入全局注册表。
// 调用约定：每个 bot service 文件在 init() 中调一次。
func RegisterBot(m BotMeta) {
	botRegistry = append(botRegistry, m)
}

// RegisteredBots 返回所有已注册 bot 的快照（顺序为 init 注册顺序）。
// main.go 主路径用；调用频率低，每次返回切片副本是可接受的。
func RegisteredBots() []BotMeta {
	out := make([]BotMeta, len(botRegistry))
	copy(out, botRegistry)
	return out
}

// EventStateName 从 BotKind 派生标准 state 事件名："rhythm" → "rhythm:state"。
// 替代之前散落在 events.go / botrun.go / frontend/events.ts 的 5 套常量。
func EventStateName(k BotKind) string {
	return string(k) + ":state"
}
