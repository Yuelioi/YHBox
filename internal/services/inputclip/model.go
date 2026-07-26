package inputclip

import "github.com/yottaapp/yotta/internal/blob"

// RecordingMode identifies the user-selected capture policy persisted with a
// clip. It is never inferred from whichever native events happened to arrive.
type RecordingMode string

const (
	RecordingModeSimple  RecordingMode = "simple"
	RecordingModePrecise RecordingMode = "precise"
)

func (mode RecordingMode) Valid() bool {
	return mode == RecordingModeSimple || mode == RecordingModePrecise
}

// EventType uint8 枚举 — 跨 Go / TS / binary 一致.
// 不要改值, binary codec 直接存这些数字.
type EventType uint8

const (
	EventTypeNone         EventType = 0
	EventTypeKeyDown      EventType = 1 // a=vk
	EventTypeKeyUp        EventType = 2 // a=vk
	EventTypeMouseBtnDown EventType = 3 // a=btn(0/1/2), b=x, c=y
	EventTypeMouseBtnUp   EventType = 4 // 同 MouseBtnDown
	EventTypeMouseMove    EventType = 5 // b=x, c=y
	EventTypeRawDelta     EventType = 6 // b=dx, c=dy (相机转向)
	EventTypeScroll       EventType = 7 // a=notches, b=x, c=y
)

// Event 固定布局 — cache locality + binary mmap 友好.
// 字段 a/b/c 含义按 Type 解释, 看 decoder.
// 32 字节: TUs(8) + Seq(4) + Type(1) + 3 字节 padding + A(4) + B(4) + C(4) + 4 字节 padding = 32
type Event struct {
	TUs  uint64    `json:"tUs"`  // 微秒, 相对 events[0].TUs (永远是 0)
	Seq  uint32    `json:"seq"`  // 同 TUs 内单调递增, 排序 tie-break
	Type EventType `json:"type"` // 1 字节, 后跟 3 字节 padding 对齐 4
	_    [3]byte   // 显式 padding (blank field, JSON 跳过)
	A    int32     `json:"a"`
	B    int32     `json:"b"`
	C    int32     `json:"c"`
	_    [4]byte   // 总 32 字节对齐 (blank field, JSON 跳过)
}

// Less 排序谓词: (TUs, Seq) 主次键
func (e Event) Less(o Event) bool {
	if e.TUs != o.TUs {
		return e.TUs < o.TUs
	}
	return e.Seq < o.Seq
}

// ClipMeta 录制环境快照.
type ClipMeta struct {
	RecordingMode  RecordingMode `json:"recordingMode"`  // 'simple' | 'precise'
	MouseMode      string        `json:"mouseMode"`      // 'relative' | 'absolute' | 'mixed'
	BaseResolution [2]int        `json:"baseResolution"` // [w, h]
	MouseCounts360 int           `json:"mouseCounts360"` // 相机转向缩放分母
	StopHotkeyVK   uint32        `json:"stopHotkeyVK"`   // 默认 0x7B (F12)
}

// InputClip carrier content is immutable after recording. Presentation metadata
// lives in the asset record and does not participate in content identity.
type InputClip struct {
	ID          string       `json:"id"`
	Label       string       `json:"label"`
	Description string       `json:"description,omitempty"`
	Category    string       `json:"category,omitempty"` // 库分组用; 同子图 Category 语义
	Tags        []string     `json:"tags,omitempty"`
	DurationUs  uint64       `json:"durationUs"` // = Events[last].TUs (Events[0].TUs ≡ 0)
	CreatedAt   string       `json:"createdAt"`  // RFC3339
	Meta        ClipMeta     `json:"meta"`
	Blob        blob.BlobRef `json:"blob"`
	Events      []Event      `json:"-"` // binary carrier only; RPC exposes metadata and the nominal BlobRef.
}

// UpdateDuration 录制 Stop 时 / 加载校验时调.
func (c *InputClip) UpdateDuration() {
	if len(c.Events) == 0 {
		c.DurationUs = 0
		return
	}
	c.DurationUs = c.Events[len(c.Events)-1].TUs
}
