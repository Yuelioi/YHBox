// internal/nodes/control/loop.go
// Loop — 第一个 RegionRunner 节点. Body 子图执行 N 次 / forever.
// Body 内 Break sentinel → 跳完成出口; Continue sentinel → 下一轮.
//
// 跟老 runtime nodes.go::execLoop 行为对齐 (count + forever 两 mode), 但 while mode
// 老有, 新版砍掉 — while 用 If + Break 拼接更直白, mode dropdown 不再列.
package control

import (
	"encoding/json"
	"errors"
	"fmt"

	"yhbox/internal/node"
)

func init() { node.Register(&Loop{}) }

type Loop struct{}

const (
	loopInExec  = "in" // exec pin 跟其他节点一起后续 batch 迁
	loopInMode  = "Mode"
	loopInCount = "Count"

	loopOutBody = "body" // exec out 同 in, 后续 batch 迁
	loopOutDone = "done"
)

func (Loop) Spec() node.Spec {
	return node.Spec{
		Kind:        "Loop",
		Category:    "Control",
		DisplayName: "循环",
		Description: "Body 子图执行 N 次 / forever. Body 内 Break sentinel 跳出, Continue sentinel 跳下一轮.",
		Inputs: []node.InputSpec{
			{Name: loopInExec, Type: "Exec"},
			{Name: loopInMode, Type: "String", Default: "count",
				DisplayName: "模式",
				Widget: node.WidgetSpec{Kind: "dropdown",
					Props: node.MarshalProps(node.DropdownProps{
						Options: []node.EnumOption{
							{Value: "count", Label: "次数"},
							{Value: "forever", Label: "永远"},
						},
					})}},
			{Name: loopInCount, Type: "Integer", Default: json.Number("10"),
				DisplayName: "次数 (mode=count)",
				Widget:      node.WidgetSpec{Kind: "number"},
				VisibleWhen: &node.VisibleRule{Field: loopInMode, Equals: "count"}},
		},
		Outputs: []node.OutputSpec{
			{Name: loopOutBody, Type: "Exec", DisplayName: "循环体 (每轮触发)"},
			{Name: loopOutDone, Type: "Exec", DisplayName: "完成"},
		},
	}
}

// Run — 防御性. 框架走 RunNodeAsRegion → RunRegion. 直调 Run 是 framework bug.
func (Loop) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	return nil, errLoopMustUseRegion
}

// RunRegion — body() 调 N 次 / forever. break/continue sentinel 截获.
func (Loop) RunRegion(ctx node.Ctx, in node.Inputs, body func(node.Ctx) error) (node.Outputs, error) {
	mode := in.String(loopInMode)
	switch mode {
	case "count":
		count := in.Int(loopInCount)
		for i := 0; i < count; i++ {
			if err := body(ctx); err != nil {
				if errors.Is(err, errBreakRequested) {
					return ctx.Out(loopOutDone).Fire(), nil
				}
				if errors.Is(err, errContinueRequested) {
					continue
				}
				return nil, err
			}
		}
		return ctx.Out(loopOutDone).Fire(), nil
	case "forever":
		for {
			if err := body(ctx); err != nil {
				if errors.Is(err, errBreakRequested) {
					return ctx.Out(loopOutDone).Fire(), nil
				}
				if errors.Is(err, errContinueRequested) {
					continue
				}
				return nil, err
			}
		}
	default:
		return nil, fmt.Errorf("Loop: unknown mode %q", mode)
	}
}
