// internal/nodes/variable/set_var.go
// SetVar — 按 scope=auto/local/global 写一个容器变量.
package variable

import (
	"fmt"

	"yotta/internal/node"
)

func init() { node.Register(&SetVar{}) }

type SetVar struct{}

const (
	svInExec    = "In"
	svInVarName = "VarName"
	svInScope   = "Scope"
	svInValue   = "Value"
	svOutOut    = "Done"
)

func (SetVar) Spec() node.Spec {
	return node.Spec{
		Kind:     "SetVar",
		Category: "Variable",
		Inputs: []node.InputSpec{
			{Name: svInExec, Type: "Exec"},
			{Name: svInVarName, Type: "String", Required: true,
				Widget: node.WidgetSpec{Kind: "text"}},
			{Name: svInScope, Type: "String", Default: "auto",
				Widget: node.WidgetSpec{Kind: "dropdown",
					Props: node.MarshalProps(node.DropdownProps{
						Options: []node.EnumOption{
							{Value: "auto"},
							{Value: "local"},
							{Value: "global"},
						}})}},
			{Name: svInValue, Type: "*", Required: true,
				Widget: node.WidgetSpec{Kind: "text"}},
		},
		Outputs: []node.OutputSpec{
			{Name: svOutOut, Type: "Exec"},
		},
	}
}

func (SetVar) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	name := in.String(svInVarName)
	if name == "" {
		return nil, fmt.Errorf("SetVar: missing varName")
	}
	val := in.Raw(svInValue)
	scope := in.String(svInScope)
	if scope == "" {
		scope = "auto"
	}
	ctx.Vars().SetScoped(name, scope, val)
	return ctx.Out(svOutOut).Fire(), nil
}

func (SetVar) Display(in node.Inputs, exitName string, out node.OutputData) string {
	return fmt.Sprintf("%s := %v", in.String(svInVarName), in.Raw(svInValue))
}
