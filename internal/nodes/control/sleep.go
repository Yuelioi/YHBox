// Package control 控制流节点 (基础). Phase 5 加 Loop/Switch full/Parallel/Race.
package control

import (
	"fmt"
	"time"

	"yhbox/internal/node"
)

func init() { node.Register(&Sleep{}) }

type Sleep struct{}

const (
	sleepInExec     = "In"
	sleepInDuration = "Duration"
	sleepOutDone    = "Done"
)

func (Sleep) Spec() node.Spec {
	return node.Spec{
		Kind:        "Sleep",
		Category:    "Control",
		DisplayName: "等待",
		Description: "阻塞当前 exec 流指定时长. MVP 不支持 cancel (ctx.Context() 接口在但 noop).",
		Inputs: []node.InputSpec{
			{Name: sleepInExec, Type: "Exec"},
			{Name: sleepInDuration, Type: "Duration", Required: true,
				DisplayName: "时长",
				Widget:      node.WidgetSpec{Kind: "duration"}},
		},
		Outputs: []node.OutputSpec{
			{Name: sleepOutDone, Type: "Exec", DisplayName: "完成"},
		},
	}
}

func (Sleep) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	d := in.Duration(sleepInDuration)
	if d <= 0 {
		return nil, fmt.Errorf("Sleep Duration 必须 > 0, got %v", d)
	}
	// MVP: 简单 time.Sleep. Phase 5 加 select { case <-ctx.Done(): / case <-time.After(d): }
	time.Sleep(d)
	return ctx.Out(sleepOutDone).Fire(), nil
}

func (Sleep) Display(in node.Inputs, exitName string, out node.OutputData) string {
	return fmt.Sprintf("sleep %v", in.Duration(sleepInDuration))
}
