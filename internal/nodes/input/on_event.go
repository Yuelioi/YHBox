// internal/nodes/input/on_event.go
// OnEvent — 事件监听节点 (listener-style). runtime listener.go::EventListener
// 在主图启动期 spawn 长跑 goroutine, 周期性 Detect, 命中条件 → spawn 子 runner 跑
// out pin 后裔子图 (maxConcurrent / retriggerPolicy=drop|queue|restart /
// cooldownMs 决定 spawn 节奏). 节点本身在主 dispatch 中不被调用 (没 exec-in),
// 直调 Run 返 sentinel.
//
// Dependencies(in) 在 kind=template_appeared 时提 template 依赖, library scanner 走这个.
package input

import (
	"encoding/json"
	"errors"

	"yotta/internal/node"
)

func init() { node.Register(&OnEvent{}) }

type OnEvent struct{}

const (
	oeInKind            = "Kind"
	oeInTemplates       = "Templates"
	oeInMatchMode       = "MatchMode"
	oeInThreshold       = "Threshold"
	oeInPollIntervalMs  = "PollIntervalMs"
	oeInMaxConcurrent   = "MaxConcurrent"
	oeInCooldownMs      = "CooldownMs"
	oeInRetriggerPolicy = "RetriggerPolicy"
	oeOutOut            = "Out"
)

// errOnEventNotWired — stub 防御性报错 (主 dispatch 不会调到 OnEvent, 它没
// exec-in, 但 region runner 误触可被 catch).
var errOnEventNotWired = errors.New("OnEvent 尚未接线: 需 EventBus + listener spawn 机制")

func (OnEvent) Spec() node.Spec {
	return node.Spec{
		Kind:     "OnEvent",
		Category: "Input",
		Inputs: []node.InputSpec{
			{Name: oeInKind, Type: "String", Default: "template_appeared",
				Widget: node.WidgetSpec{Kind: "dropdown",
					Props: node.MarshalProps(node.DropdownProps{
						Options: []node.EnumOption{
							{Value: "template_appeared"},
						}})}},
			{Name: oeInTemplates, Type: "String", Semantic: "TemplateKey",
				Widget: node.WidgetSpec{Kind: "template-picker"}},
			{Name: oeInMatchMode, Type: "String", Default: "any", Advanced: true,
				Widget: node.WidgetSpec{Kind: "dropdown",
					Props: node.MarshalProps(node.DropdownProps{
						Options: []node.EnumOption{{Value: "any"}, {Value: "all"}}})}},
			{Name: oeInThreshold, Type: "Number", Default: json.Number("0.85"),
				Widget: node.WidgetSpec{Kind: "slider",
					Props: node.MarshalProps(node.SliderProps{Min: 0, Max: 1, Step: 0.01})}},
			{Name: oeInPollIntervalMs, Type: "Number", Default: json.Number("100"),
				Widget: node.WidgetSpec{Kind: "number"}},
			{Name: oeInMaxConcurrent, Type: "Number", Default: json.Number("1"),
				Advanced: true,
				Widget:   node.WidgetSpec{Kind: "number"}},
			{Name: oeInCooldownMs, Type: "Number", Default: json.Number("0"),
				Advanced: true,
				Widget:   node.WidgetSpec{Kind: "number"}},
			{Name: oeInRetriggerPolicy, Type: "String", Default: "drop",
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
			// 没 exec-in, 单 exec-out (listener spawn 子 runner 入口).
			{Name: oeOutOut, Type: "Exec"},
		},
	}
}

func (OnEvent) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	return nil, errOnEventNotWired
}

func (OnEvent) Dependencies(in node.Inputs) []node.Dependency {
	if in.String(oeInKind) != "template_appeared" {
		return nil
	}
	var deps []node.Dependency
	for _, key := range in.StringList(oeInTemplates) {
		if key != "" {
			deps = append(deps, node.Dependency{Kind: "template", Key: key})
		}
	}
	return deps
}
