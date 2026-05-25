// internal/nodes/control/stop.go
// Stop — terminate graph dispatch cleanly. Has exec-in, no exec-out.
// Run returns errStopRun sentinel; Phase 5 runner catches it and halts the
// dispatch loop without emitting container:error (same behavior as old
// runtime.errStopRun — see internal/services/container/runtime/nodes.go::execNode
// case "Stop" + runSubFlow's errors.Is(err, errStopRun) check).
package control

import "yhbox/internal/node"

func init() { node.Register(&Stop{}) }

type Stop struct{}

const (
	stopInExec = "in"
)

func (Stop) Spec() node.Spec {
	return node.Spec{
		Kind:        "Stop",
		Category:    "Control",
		DisplayName: "终点",
		Description: "终止图执行. 框架捕获 sentinel 后停止 dispatch, 不报错.",
		Inputs: []node.InputSpec{
			{Name: stopInExec, Type: "Exec"},
		},
		// no Outputs — terminal node
	}
}

func (Stop) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	return nil, errStopRun
}
