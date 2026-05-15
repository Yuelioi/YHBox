// Package actiontransform 把录制器的 raw 事件流清洗成 actions.Step 序列。
// 多 pass pipeline 独立于 recorder.go，方便：
//  1) v2 "重新生成 steps" 按钮用更激进/保守策略再算一次
//  2) 未来导入外部 macro 复用同一套清洗
//  3) 每 pass 单测覆盖
package actiontransform

import "yhbox/internal/services/actions"

type EventType int

const (
	EventKeyboard       EventType = iota
	EventMouse                    // 鼠标按键 down/up
	EventMouseMove                // 纯移动（30ms 节流过）— 给 drag 检测做轨迹
	EventMouseScroll              // 滚轮
	EventRawMouseDelta            // Raw Input 相对位移（camera 转向）—— PairEvents 聚合成 mouse_move_rel
)

type MouseBtn int

const (
	BtnLeft MouseBtn = iota
	BtnMiddle
	BtnRight
)

// Event 录制器 raw 事件 schema。
// TimestampMs 来自 hook 自带 timestamp（KBDLLHOOKSTRUCT.time / MSLLHOOKSTRUCT.time，
// uint32 ms tick）—— 比 time.Now() 少 goroutine scheduling jitter。
type Event struct {
	TimestampMs uint32
	Type        EventType
	MouseBtn    MouseBtn // EventMouse 时用
	Vk          string   // EventKeyboard 时用
	X, Y        int      // 鼠标系列 event 都是 client 坐标（recorder 已转）
	IsDown      bool     // EventMouse 时用
	// EventMouseScroll 时用。正=滚轮上推（页面上滑），负=下。
	WheelNotches int
	// EventRawMouseDelta 时用：设备 mickeys。+x 右、+y 下。
	Dx int
	Dy int
}

// EventsToSteps 把 raw events 转 step 序列：
// Normalize → PairEvents → MergeSleeps → MergeMouseMoveRel → CollapseCombos。
// clientW/clientH 用来把 mouse client 坐标转 ratio。
func EventsToSteps(events []Event, clientW, clientH int) []actions.Step {
	return CollapseCombos(MergeMouseMoveRel(MergeSleeps(PairEvents(Normalize(events), clientW, clientH))))
}
