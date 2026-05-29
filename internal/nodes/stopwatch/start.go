// Package stopwatch 秒表节点 (Start / Stop / Read). 走 ctx.Stopwatches() 服务.
//
// 语义:
//   - Start: 已存在 key 视为 reset.
//   - Stop:  不存在 key 视为 no-op (validator static-warn).
//   - Read:  不存在 key 返 0; running 返 now-start; stopped 返 stoppedAt-start.
//
// key 命名空间独立于 $vars.* (同名 key/var 不冲突).
package stopwatch

import (
	"errors"

	"yhbox/internal/node"
)

func init() { node.Register(&Start{}) }

// Start 启动 / reset 指定 key 的秒表. 老 runtime: stopwatch_nodes.go::execStopwatchStart.
type Start struct{}

const (
	swStartInExec = "In"
	swStartInKey  = "Key"
	swStartOutOut = "Done"
)

func (Start) Spec() node.Spec {
	return node.Spec{
		Kind:     "StopwatchStart",
		Category: "Stopwatch",
		Inputs: []node.InputSpec{
			{Name: swStartInExec, Type: "Exec"},
			{Name: swStartInKey, Type: "String", Required: true, Default: "default",
				Widget: node.WidgetSpec{Kind: "text"}},
		},
		Outputs: []node.OutputSpec{
			{Name: swStartOutOut, Type: "Exec"},
		},
	}
}

func (Start) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	key := in.String(swStartInKey)
	if key == "" {
		return nil, errStopwatchEmptyKey
	}
	ctx.Stopwatches().Start(key)
	return ctx.Out(swStartOutOut).Fire(), nil
}

func (Start) Display(in node.Inputs, exitName string, out node.OutputData) string {
	return "sw▶ " + in.String(swStartInKey)
}

// errStopwatchEmptyKey — Required 已防 nil, 这是 defensive.
var errStopwatchEmptyKey = errors.New("stopwatch: empty key")
