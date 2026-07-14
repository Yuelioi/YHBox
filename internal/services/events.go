// Package services 是 wails3 service 层 + 共享基础设施。
// RPC service 通过 presentation binding 暴露给前端；事件经注入的 emitter 单向推送。
package services

// 事件名常量 —— 跟 frontend/src/constants/events.ts 一一对应。
const (
	EventLogBatch = "log:batch"
)

// LogEntry is the normalized presentation contract shared by system logs,
// runtime logs, node dumps, and action traces. File persistence keeps the
// original JSONL representation; only the UI transport uses this shape.
type LogEntry struct {
	Time    string `json:"time"`
	Level   string `json:"level"`
	Source  string `json:"source"`
	Kind    string `json:"kind"`
	Tag     string `json:"tag,omitempty"`
	Message string `json:"message"`
	Fields  any    `json:"fields,omitempty"`

	NodeID  string `json:"nodeId,omitempty"`
	LineKey string `json:"lineKey,omitempty"`
	Count   int    `json:"count,omitempty"`
	Final   bool   `json:"final,omitempty"`
	Trace   any    `json:"trace,omitempty"`
}

// LogBatchEvent is the only backend-to-frontend diagnostic transport.
// Seq is monotonic. Dropped reports entries discarded behind a slow consumer.
type LogBatchEvent struct {
	Seq     uint64     `json:"seq"`
	Entries []LogEntry `json:"entries"`
	Dropped uint64     `json:"dropped,omitempty"`
}
