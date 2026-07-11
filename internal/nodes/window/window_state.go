package window

import "github.com/yottaapp/yotta/internal/node"

func init() { node.Register(&WindowState{}) }

type WindowState struct{}

const (
	wsInExec  = "In"
	wsInState = "State"
	wsDone    = "Done"
)

func (WindowState) Spec() node.Spec {
	return node.Spec{
		Kind: "WindowState", Category: "Window", NeedsWindow: true,
		RuntimeCapabilities: []node.RuntimeCapability{node.RuntimeCapabilityWindow},
		Inputs: append([]node.InputSpec{
			{Name: wsInExec, Type: "Exec"},
			{Name: wsInState, Type: "String", Default: "maximize",
				Widget: node.WidgetSpec{Kind: "dropdown", Props: node.MarshalProps(node.DropdownProps{
					Options: []node.EnumOption{
						{Value: "maximize"}, {Value: "minimize"}, {Value: "restore"},
						{Value: "borderlessFullscreen"}, {Value: "restoreBorders"},
					}})}},
		}, node.WindowInputSpec()),
		Outputs: []node.OutputSpec{
			{Name: wsDone, Type: "Exec", Data: []node.DataField{{Name: "Window", Type: "Window"}}},
		},
	}
}

func (WindowState) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	var err error
	switch in.String(wsInState) {
	case "maximize":
		err = ctx.Window().Maximize()
	case "minimize":
		err = ctx.Window().Minimize()
	case "restore":
		err = ctx.Window().Restore()
	case "borderlessFullscreen":
		err = ctx.Window().BorderlessFullscreen()
	case "restoreBorders":
		err = ctx.Window().RestoreBorders()
	default:
		return nil, node.Failf(node.CodeError, nil, "WindowState: 未知 State %q", in.String(wsInState))
	}
	if err != nil {
		return nil, err
	}
	w, err := ctx.Window().Snapshot()
	if err != nil {
		return nil, err
	}
	return ctx.Out(wsDone).Set("Window", w).Fire(), nil
}
