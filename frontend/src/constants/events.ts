// 跟 Go 端 cmd/yhbox/services/events.go 的事件名常量一一对应。
// 长跑 bot 的 state 事件（fish:state 等）由 bot-registry.ts 的 botStateEventName 派生，不在此列。
export const EVENT_FISH_STATS = 'fish:stats'
export const EVENT_PIANO_PROGRESS = 'piano:progress'
export const EVENT_BATTLE_STATE = 'battle:state'
export const EVENT_GAME_STATUS = 'game:status'
export const EVENT_LOG_LINES = 'log:lines'
