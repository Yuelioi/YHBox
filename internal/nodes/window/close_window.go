package window

import "yotta/internal/node"

func init() { node.Register(&CloseWindow{}) }

type CloseWindow struct{}

const (
	cwInExec = "In"
	cwDone   = "Done"
)

func (CloseWindow) Spec() node.Spec {
	return node.Spec{
		Kind: "CloseWindow", Category: "Window", NeedsWindow: true,
		Inputs:  append([]node.InputSpec{{Name: cwInExec, Type: "Exec"}}, node.WindowInputSpec()),
		Outputs: []node.OutputSpec{{Name: cwDone, Type: "Exec"}},
	}
}

func (CloseWindow) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	if err := ctx.Window().Close(); err != nil {
		return nil, err
	}
	return ctx.Out(cwDone).Fire(), nil
}
