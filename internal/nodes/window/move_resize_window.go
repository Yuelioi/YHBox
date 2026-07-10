package window

import (
	"encoding/json"

	"github.com/yottaapp/yotta/internal/node"
)

func init() { node.Register(&MoveResizeWindow{}) }

type MoveResizeWindow struct{}

const (
	mrInExec = "In"
	mrInX    = "X"
	mrInY    = "Y"
	mrInW    = "Width"
	mrInH    = "Height"
	mrDone   = "Done"
)

func (MoveResizeWindow) Spec() node.Spec {
	num := func(name string) node.InputSpec {
		return node.InputSpec{Name: name, Type: "Number", Default: json.Number("0"), Widget: node.WidgetSpec{Kind: "number"}}
	}
	return node.Spec{
		Kind: "MoveResizeWindow", Category: "Window", NeedsWindow: true,
		Inputs: append([]node.InputSpec{
			{Name: mrInExec, Type: "Exec"}, num(mrInX), num(mrInY), num(mrInW), num(mrInH),
		}, node.WindowInputSpec()),
		Outputs: []node.OutputSpec{
			{Name: mrDone, Type: "Exec", Data: []node.DataField{{Name: "Window", Type: "Window"}}},
		},
	}
}

func (MoveResizeWindow) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	if err := ctx.Window().MoveResize(in.Int(mrInX), in.Int(mrInY), in.Int(mrInW), in.Int(mrInH)); err != nil {
		return nil, err
	}
	w, err := ctx.Window().Snapshot()
	if err != nil {
		return nil, err
	}
	return ctx.Out(mrDone).Set("Window", w).Fire(), nil
}
