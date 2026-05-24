// Package stopwatch 秒表节点 (Start / Stop / Read). 老 runtime: stopwatch_nodes.go +
// stopwatch.go stopwatchTable per-ContainerRunner. 新框架走 ctx.Stopwatches() 服务
// (interfaces.go::StopwatchStore), services.go stubStopwatchStore 测试用; Phase 5
// wire 时 main.go 注入 RuntimeContext.stopwatches 适配.
//
// 语义 (镜像老 stopwatchTable):
//   - Start: 已存在 key 视为 reset.
//   - Stop:  不存在 key 视为 no-op (validator static-warn).
//   - Read:  不存在 key 返 0; running 返 now-start; stopped 返 stoppedAt-start.
//
// key 命名空间独立于 $vars.* (同名 key/var 不冲突, 见老 stopwatch.go 注释).
package stopwatch

import (
	"errors"

	"yhbox/internal/node"
)

func init() { node.Register(&Start{}) }

// Start 启动 / reset 指定 key 的秒表. 老 runtime: stopwatch_nodes.go::execStopwatchStart.
type Start struct{}

const (
	swStartInExec = "in"
	swStartInKey  = "Key"
	swStartOutOut = "Out"
)

func (Start) Spec() node.Spec {
	return node.Spec{
		Kind:        "StopwatchStart",
		Version:     1,
		Category:    "Stopwatch",
		DisplayName: "秒表 启动",
		Description: "启动或 reset 指定 key 的秒表 (已存在 → reset).",
		Inputs: []node.InputSpec{
			{Name: swStartInExec, Type: "Exec"},
			{Name: swStartInKey, Type: "String", Required: true, Default: "default",
				DisplayName: "key",
				Doc:         "秒表 key (命名空间独立于 $vars.*)",
				Widget:      node.WidgetSpec{Kind: "text"}},
		},
		Outputs: []node.OutputSpec{
			{Name: swStartOutOut, Type: "Exec", DisplayName: "完成"},
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

// errStopwatchEmptyKey 跟老 runtime 一致 — Required 已防 nil, 这是 defensive.
var errStopwatchEmptyKey = errors.New("stopwatch: empty key")
