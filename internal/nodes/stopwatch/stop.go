// internal/nodes/stopwatch/stop.go
// StopwatchStop — 停止指定 key 的秒表 (不存在 key 静默 no-op, 镜像老 runtime).
// 老 runtime: stopwatch_nodes.go::execStopwatchStop. validator static-warn 路径
// 由 container validator 处理 (key 不在 Start 列表) — 节点本身 runtime 不报错.
package stopwatch

import (
	"yhbox/internal/node"
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
		Kind:        "StopwatchStop",
		Category:    "Stopwatch",
		DisplayName: "秒表 停止",
		Description: "停止指定 key 的秒表 (不存在 key 静默 no-op, validator 已 static-warn).",
		Inputs: []node.InputSpec{
			{Name: swStopInExec, Type: "Exec"},
			{Name: swStopInKey, Type: "String", Required: true, Default: "default",
				DisplayName: "key",
				Doc:         "跟之前 StopwatchStart 同一个 key",
				Widget:      node.WidgetSpec{Kind: "text"}},
		},
		Outputs: []node.OutputSpec{
			{Name: swStopOutOut, Type: "Exec", DisplayName: "完成"},
		},
	}
}

func (Stop) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	key := in.String(swStopInKey)
	if key == "" {
		return nil, errStopwatchEmptyKey
	}
	ctx.Stopwatches().Stop(key) // no-op on missing — 老 runtime 一致
	return ctx.Out(swStopOutOut).Fire(), nil
}

func (Stop) Display(in node.Inputs, exitName string, out node.OutputData) string {
	return "sw■ " + in.String(swStopInKey)
}
