// internal/nodes/event/event_tick.go
// EventTick — 定时后台触发节点 (listener-style). runtime listener.go::EventListener
// kind=tick: 主图启动起后台 goroutine, 每 IntervalMs 触发一次 Out 后裔子图 (带 DeltaMs
// 数据出口), 不挡主流程. 检测/改变量靠子图组合 (EventTick→检测→If→SetVar(global))。
// 节点本身在主 dispatch 中不被调用 (没 exec-in), 直调 Run 返 sentinel。
package event

import (
	"encoding/json"
	"errors"

	"yotta/internal/node"
)

func init() { node.Register(&EventTick{}) }

type EventTick struct{}

const (
	etInIntervalMs      = "IntervalMs"
	etInMaxConcurrent   = "MaxConcurrent"
	etInRetriggerPolicy = "RetriggerPolicy"
	etOutOut            = "Out"
	etOutDataDeltaMs    = "DeltaMs"
)

// errEventTickNotWired — stub 防御性报错 (主 dispatch 不会调到 EventTick, 它没 exec-in,
// 由 runtime listener spawn 驱动)。
var errEventTickNotWired = errors.New("EventTick 尚未接线: 需 listener spawn 机制 (无 exec-in, 主 dispatch 不调)")

func (EventTick) Spec() node.Spec {
	return node.Spec{
		Kind:     "EventTick",
		Category: "Event",
		Inputs: []node.InputSpec{
			{Name: etInIntervalMs, Type: "Number", Default: json.Number("100"),
				Widget: node.WidgetSpec{Kind: "number"}},
			{Name: etInMaxConcurrent, Type: "Number", Default: json.Number("1"),
				Advanced: true,
				Widget:   node.WidgetSpec{Kind: "number"}},
			{Name: etInRetriggerPolicy, Type: "String", Default: "drop",
				Advanced: true,
				Widget: node.WidgetSpec{Kind: "dropdown",
					Props: node.MarshalProps(node.DropdownProps{
						Options: []node.EnumOption{
							{Value: "drop"},
							{Value: "queue"},
							{Value: "restart"},
						}})}},
		},
		Outputs: []node.OutputSpec{
			{Name: etOutOut, Type: "Exec",
				Data: []node.DataField{
					{Name: etOutDataDeltaMs, Type: "Number"},
				}},
		},
	}
}

func (EventTick) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	return nil, errEventTickNotWired
}
