// StopwatchStop — 停止指定 key 的秒表 (不存在 key 静默 no-op).
// 由 container validator 处理 (key 不在 Start 列表) — 节点本身 runtime 不报错.
package stopwatch

import (
	"yotta/internal/node"
)

func init() { node.Register(&Stop{}) }

type Stop struct{}

const (
	swStopInExec = "In"
	swStopInKey  = "Key"
	swStopOutOut = "Done"
)

func (Stop) Spec() node.Spec {
	return node.Spec{
		Kind:     "StopwatchStop",
		Category: "Stopwatch",
		Inputs: []node.InputSpec{
			{Name: swStopInExec, Type: "Exec"},
			{Name: swStopInKey, Type: "String", Required: true, Default: "default",
				Widget: node.WidgetSpec{Kind: "text"}},
		},
		Outputs: []node.OutputSpec{
			{Name: swStopOutOut, Type: "Exec"},
		},
	}
}

func (Stop) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	key := in.String(swStopInKey)
	if key == "" {
		return nil, errStopwatchEmptyKey
	}
	ctx.Stopwatches().Stop(key) // no-op on missing
	return ctx.Out(swStopOutOut).Fire(), nil
}

func (Stop) Display(in node.Inputs, exitName string, out node.OutputData) string {
	return "sw■ " + in.String(swStopInKey)
}
