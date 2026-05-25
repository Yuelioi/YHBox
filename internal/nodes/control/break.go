// internal/nodes/control/break.go
// Break — request innermost Loop region to terminate iteration.
// Phase 4 stub: returns errBreakRequested. Phase 5 Loop region catches it.
package control

import "yhbox/internal/node"

func init() { node.Register(&Break{}) }

type Break struct{}

const (
	breakInExec = "In"
)

func (Break) Spec() node.Spec {
	return node.Spec{
		Kind:        "Break",
		Category:    "Control",
		DisplayName: "跳出循环",
		Description: "退出最近的 Loop 区域. Phase 5 Loop region 截获 sentinel, 否则 framework 当 error 上报.",
		Inputs: []node.InputSpec{
			{Name: breakInExec, Type: "Exec"},
		},
		// no Outputs — control transfer
	}
}

func (Break) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	return nil, errBreakRequested
}
