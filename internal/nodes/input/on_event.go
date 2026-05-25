// internal/nodes/input/on_event.go
// OnEvent — IsYield 事件监听节点 (listener-style). 老 runtime:
// listener.go::EventListener — 主图启动期 spawn 长跑 goroutine, 周期性 Detect,
// 命中条件 → spawn 子 runner 跑 out pin 后裔子图 (maxConcurrent /
// retriggerPolicy=drop|queue|restart / cooldownMs 决定 spawn 节奏).
// 节点本身在主 dispatch 中不被调用 (没 exec-in).
//
// Phase 5 wire 需要新增 EventBus service (Subscribe(filter) + Fire), 让节点
// Run 走 RegionRunner / yield 机制. 本 Phase 仅注册 Spec (FE 节点列表可见),
// Run 返 errOnEventPhase5 sentinel — 用户在 graph 里放 OnEvent 跑会立刻报错.
//
// dependency.RegisterExtractor (老 specs/input.go) 在 kind=template_appeared
// 时提 template 依赖, 新框架走 Dependencies(in) 复刻同逻辑.
package input

import (
	"errors"

	"yhbox/internal/node"
)

func init() { node.Register(&OnEvent{}) }

type OnEvent struct{}

const (
	oeInKind              = "Kind"
	oeInTemplate          = "Template"
	oeInThreshold         = "Threshold"
	oeInPollIntervalMs    = "PollIntervalMs"
	oeInMaxConcurrent     = "MaxConcurrent"
	oeInCooldownMs        = "CooldownMs"
	oeInRetriggerPolicy   = "RetriggerPolicy"
	oeOutOut              = "Out"
)

// errOnEventPhase5 sentinel returned by OnEvent.Run. Phase 5 wire 加 EventBus +
// RegionRunner 后 Run 走真订阅; 当前 stub 防御性报错 (实际主 dispatch 不会
// 调到 OnEvent — 它没 exec-in, 但 region runner 误触可被 catch).
var errOnEventPhase5 = errors.New("OnEvent — Phase 5 wire 需要 EventBus + listener spawn 机制")

func (OnEvent) Spec() node.Spec {
	return node.Spec{
		Kind:        "OnEvent",
		Category:    "Input",
		DisplayName: "事件监听",
		Description: "(Phase 5 stub) listener 节点 — 周期性 Detect 命中条件 → spawn 子 runner 跑 Out 后裔. 没 exec-in.",
		Inputs: []node.InputSpec{
			{Name: oeInKind, Type: "String", Default: "template_appeared",
				DisplayName: "事件类型",
				Widget: node.WidgetSpec{Kind: "dropdown",
					Props: node.MarshalProps(node.DropdownProps{
						Options: []node.EnumOption{
							{Value: "template_appeared", Label: "Template Appeared"},
						}})}},
			{Name: oeInTemplate, Type: "String", Semantic: "TemplateKey",
				DisplayName: "模板",
				Doc:         "命名空间.名 格式, kind=template_appeared 时必填",
				Widget: node.WidgetSpec{Kind: "async-dropdown",
					Props: node.MarshalProps(node.AsyncDropdownProps{AsyncSource: "templateKeys"})}},
			{Name: oeInThreshold, Type: "Number", Default: 0.85,
				DisplayName: "阈值",
				Widget: node.WidgetSpec{Kind: "slider",
					Props: node.MarshalProps(node.SliderProps{Min: 0, Max: 1, Step: 0.01})}},
			{Name: oeInPollIntervalMs, Type: "Number", Default: 100,
				DisplayName: "轮询间隔 (ms)",
				Widget:      node.WidgetSpec{Kind: "number"}},
			{Name: oeInMaxConcurrent, Type: "Number", Default: 1,
				DisplayName: "并发上限",
				Advanced:    true,
				Widget:      node.WidgetSpec{Kind: "number"}},
			{Name: oeInCooldownMs, Type: "Number", Default: 0,
				DisplayName: "冷却 (ms)",
				Advanced:    true,
				Widget:      node.WidgetSpec{Kind: "number"}},
			{Name: oeInRetriggerPolicy, Type: "String", Default: "drop",
				DisplayName: "重触发策略",
				Advanced:    true,
				Widget: node.WidgetSpec{Kind: "dropdown",
					Props: node.MarshalProps(node.DropdownProps{
						Options: []node.EnumOption{
							{Value: "drop", Label: "Drop (忽略, 等当前完)"},
							{Value: "queue", Label: "Queue (FIFO 排队)"},
							{Value: "restart", Label: "Restart (取消旧, 跑新)"},
						}})}},
		},
		Outputs: []node.OutputSpec{
			// 没 exec-in, 单 exec-out (listener spawn 子 runner 入口).
			{Name: oeOutOut, Type: "Exec", DisplayName: "事件触发 (Phase 5)"},
		},
	}
}

func (OnEvent) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	return nil, errOnEventPhase5
}

func (OnEvent) Dependencies(in node.Inputs) []node.Dependency {
	if in.String(oeInKind) != "template_appeared" {
		return nil
	}
	key := in.String(oeInTemplate)
	if key == "" {
		return nil
	}
	return []node.Dependency{{Kind: "template", Key: key}}
}
