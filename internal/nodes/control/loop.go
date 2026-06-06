// internal/nodes/control/loop.go
// Loop — RegionRunner 节点. Body 子图执行 N 次 / forever.
// Body 内 Break sentinel → 跳完成出口; Continue sentinel → 下一轮.
//
// 只有 count + forever 两 mode, 无 while mode — while 用 If + Break 拼接更直白.
package control

import (
	"encoding/json"
	"errors"
	"fmt"

	"yotta/internal/node"
)

func init() { node.Register(&Loop{}) }

type Loop struct{}

const (
	loopInExec  = "In" // exec pin 跟其他节点一起后续 batch 迁
	loopInMode  = "Mode"
	loopInCount = "Count"

	loopOutBody = "Body"
	loopOutDone = "Done"

	loopCapIndex = "CaptureIndex"
)

func (Loop) Spec() node.Spec {
	return node.Spec{
		Kind:     "Loop",
		Category: "Control",
		Inputs: []node.InputSpec{
			{Name: loopInExec, Type: "Exec"},
			{Name: loopInMode, Type: "String", Default: "count",
				Widget: node.WidgetSpec{Kind: "dropdown",
					Props: node.MarshalProps(node.DropdownProps{
						Options: []node.EnumOption{
							{Value: "count"},
							{Value: "forever"},
						},
					})}},
			{Name: loopInCount, Type: "Integer", Default: json.Number("10"),
				Widget:      node.WidgetSpec{Kind: "number"},
				VisibleWhen: &node.VisibleRule{Field: loopInMode, Equals: "count"}},
			{Name: loopCapIndex, Type: "String", Advanced: true, Semantic: "capture",
				CaptureType: "number", Widget: node.WidgetSpec{Kind: "text"}},
		},
		Outputs: []node.OutputSpec{
			{Name: loopOutBody, Type: "Exec"},
			{Name: loopOutDone, Type: "Exec"},
		},
	}
}

// RunRegion — body() 调 N 次 / forever. break/continue sentinel 截获.
func (Loop) RunRegion(ctx node.Ctx, in node.Inputs, body func(node.Ctx) error) (node.Outputs, error) {
	mode := in.String(loopInMode)
	switch mode {
	case "count":
		count := in.Int(loopInCount)
		for i := 0; i < count; i++ {
			node.Capture(ctx, in, loopCapIndex, i)
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
		for i := 0; ; i++ {
			node.Capture(ctx, in, loopCapIndex, i)
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
