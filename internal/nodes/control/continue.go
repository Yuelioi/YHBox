// internal/nodes/control/continue.go
// Continue — request innermost Loop region to skip rest of body + start next iteration.
// Returns errContinueRequested; the enclosing Loop region catches it.
package control

import "yotta/internal/node"

func init() { node.Register(&Continue{}) }

type Continue struct{}

const (
	continueInExec = "In"
)

func (Continue) Spec() node.Spec {
	return node.Spec{
		Kind:     "Continue",
		Category: "Control",
		Inputs: []node.InputSpec{
			{Name: continueInExec, Type: "Exec"},
		},
		// no Outputs — control transfer
	}
}

func (Continue) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	return nil, errContinueRequested
}
