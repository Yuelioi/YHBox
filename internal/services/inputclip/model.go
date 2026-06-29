package inputclip

// EventType uint8 枚举 — 跨 Go / TS / binary 一致.
// 不要改值, binary codec 直接存这些数字.
type EventType uint8

const (
	EventTypeNone        EventType = 0
	EventTypeKeyDown     EventType = 1 // a=vk
	EventTypeKeyUp       EventType = 2 // a=vk
	EventTypeMouseBtnDown EventType = 3 // a=btn(0/1/2), b=x, c=y
	EventTypeMouseBtnUp   EventType = 4 // 同 MouseBtnDown
	EventTypeMouseMove    EventType = 5 // b=x, c=y (simple 模式过滤)
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
	_    [4]byte // 总 32 字节对齐 (blank field, JSON 跳过)
}

// Less 排序谓词: (TUs, Seq) 主次键
func (e Event) Less(o Event) bool {
	if e.TUs != o.TUs {
		return e.TUs < o.TUs
	}
	return e.Seq < o.Seq
}

// Range 保留区间 — 注意: keepRanges 不在 InputClip 里, 在 PlayClipNodeConfig 里.
// 此 type 共用. fromUs inclusive, toUs exclusive.
type Range struct {
	FromUs uint64 `json:"fromUs"`
	ToUs   uint64 `json:"toUs"`
}

// ClipMeta 录制环境快照.
type ClipMeta struct {
	MouseMode      string `json:"mouseMode"`      // 'relative' | 'absolute' | 'mixed'
	BaseResolution [2]int `json:"baseResolution"` // [w, h]
	MouseCounts360 int    `json:"mouseCounts360"` // 相机转向缩放分母
	FilterMode     string `json:"filterMode"`     // 'precise' | 'simple'
	StopHotkeyVK   uint32 `json:"stopHotkeyVK"`   // 默认 0x7B (F12)
}

// InputClip immutable 资产. 一旦录完, 字段都不变.
// keepRanges 不在这里 — 在 PlayClipNodeConfig 里 (允许不同 PlayClip 各自 trim).
type InputClip struct {
	ID          string    `json:"id"`
	Label       string    `json:"label"`
	Description string    `json:"description,omitempty"`
	Category    string    `json:"category,omitempty"` // 库分组用; 同子图 Category 语义
	Tags        []string  `json:"tags,omitempty"`
	DurationUs  uint64    `json:"durationUs"` // = Events[last].TUs (Events[0].TUs ≡ 0)
	CreatedAt   string    `json:"createdAt"`  // RFC3339
	Meta        ClipMeta  `json:"meta"`
	Events      []Event   `json:"-"` // 序列化走 binary chunk 上盘, 不进 JSON RPC. 前端不需要逐 event — 回放走 ClipResolver, timeline 只画 durationUs.
}

// UpdateDuration 录制 Stop 时 / 加载校验时调.
func (c *InputClip) UpdateDuration() {
	if len(c.Events) == 0 {
		c.DurationUs = 0
		return
	}
	c.DurationUs = c.Events[len(c.Events)-1].TUs
}
