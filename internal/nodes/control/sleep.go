// Package control 控制流节点.
package control

import (
	"encoding/json"
	"fmt"
	"time"

	"yhbox/internal/node"
)

func init() { node.Register(&Sleep{}) }

type Sleep struct{}

const (
	sleepInExec      = "In"
	sleepInDuration  = "Duration"
	sleepInJitterPct = "JitterPct"
	sleepOutDone     = "Done"
)

func (Sleep) Spec() node.Spec {
	return node.Spec{
		Kind:     "Sleep",
		Category: "Control",
		Inputs: []node.InputSpec{
			{Name: sleepInExec, Type: "Exec"},
			{Name: sleepInDuration, Type: "Duration", Required: true,
				Widget: node.WidgetSpec{Kind: "duration"}},
			{Name: sleepInJitterPct, Type: "Number", Default: json.Number("0"),
				Widget: node.WidgetSpec{Kind: "number"}},
		},
		Outputs: []node.OutputSpec{
			{Name: sleepOutDone, Type: "Exec"},
		},
	}
}

func (Sleep) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	d := in.Duration(sleepInDuration)
	if d <= 0 {
		return nil, fmt.Errorf("Sleep Duration 必须 > 0, got %v", d)
	}
	d = node.JitterDuration(d, in.Int(sleepInJitterPct)) // ±% 近正态抖动 (pct=0 → 原值)
	// 可取消 sleep: graph stop/cancel 时立即中断, 不等满 d. 返 ctx.Err()
	// (context.Canceled) — dispatch 把它当优雅 halt, 不当节点失败高亮.
	select {
	case <-ctx.Context().Done():
		return nil, ctx.Context().Err()
	case <-time.After(d):
		return ctx.Out(sleepOutDone).Fire(), nil
	}
}

func (Sleep) Display(in node.Inputs, exitName string, out node.OutputData) string {
	return fmt.Sprintf("sleep %v", in.Duration(sleepInDuration))
}
