// internal/nodes/control/continue.go
// Continue — request innermost Loop region to skip rest of body + start next iteration.
// Phase 4 stub: returns errContinueRequested. Phase 5 Loop region catches it.
package control

import "yhbox/internal/node"

func init() { node.Register(&Continue{}) }

type Continue struct{}

const (
	continueInExec = "In"
)

func (Continue) Spec() node.Spec {
	return node.Spec{
		Kind:        "Continue",
		Category:    "Control",
		DisplayName: "跳过本轮",
		Description: "跳到最近 Loop 的下一次迭代. Phase 5 Loop region 截获 sentinel.",
		Inputs: []node.InputSpec{
			{Name: continueInExec, Type: "Exec"},
		},
		// no Outputs — control transfer
	}
}

func (Continue) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	return nil, errContinueRequested
}
