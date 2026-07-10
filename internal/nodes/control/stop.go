// internal/nodes/control/stop.go
// Stop — terminate graph dispatch cleanly. Has exec-in, no exec-out.
// Run returns the errStopRun sentinel; the runner catches it and halts the
// dispatch loop without emitting container:error.
package control

import "github.com/yottaapp/yotta/internal/node"

func init() { node.Register(&Stop{}) }

type Stop struct{}

const (
	stopInExec = "In"
)

func (Stop) Spec() node.Spec {
	return node.Spec{
		Kind:     "Stop",
		Category: "Control",
		Inputs: []node.InputSpec{
			{Name: stopInExec, Type: "Exec"},
		},
		// no Outputs — terminal node
	}
}

func (Stop) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	return nil, errStopRun
}
