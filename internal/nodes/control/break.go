// internal/nodes/control/break.go
// Break — request innermost Loop region to terminate iteration.
// Returns errBreakRequested; the enclosing Loop region catches it.
package control

import "yotta/internal/node"

func init() { node.Register(&Break{}) }

type Break struct{}

const (
	breakInExec = "In"
)

func (Break) Spec() node.Spec {
	return node.Spec{
		Kind:     "Break",
		Category: "Control",
		Inputs: []node.InputSpec{
			{Name: breakInExec, Type: "Exec"},
		},
		// no Outputs — control transfer
	}
}

func (Break) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	return nil, errBreakRequested
}
