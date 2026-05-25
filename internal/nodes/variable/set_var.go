// internal/nodes/variable/set_var.go
// SetVar — 写一个容器变量. 老 runtime nodes.go::execSetVar 1:1.
package variable

import (
	"fmt"

	"yhbox/internal/node"
)

func init() { node.Register(&SetVar{}) }

type SetVar struct{}

const (
	svInExec    = "in" // exec pin 暂保 lowercase (P2.1 后续 batch 跟 100+ edges 一起迁)
	svInVarName = "VarName"
	svInScope   = "Scope"
	svInValue   = "Value"
	svOutOut    = "out" // 同 svInExec
)

func (SetVar) Spec() node.Spec {
	return node.Spec{
		Kind:        "SetVar",
		Category:    "Variable",
		DisplayName: "写变量",
		Description: "写一个容器变量. scope=auto/local/global (auto: 当前 frame 有就 local, 否则 global).",
		Inputs: []node.InputSpec{
			{Name: svInExec, Type: "Exec"},
			{Name: svInVarName, Type: "String", Required: true,
				DisplayName: "变量名",
				Widget:      node.WidgetSpec{Kind: "text"}},
			{Name: svInScope, Type: "String", Default: "auto",
				DisplayName: "作用域",
				Widget: node.WidgetSpec{Kind: "dropdown",
					Props: node.MarshalProps(node.DropdownProps{
						Options: []node.EnumOption{
							{Value: "auto", Label: "auto"},
							{Value: "local", Label: "local"},
							{Value: "global", Label: "global"},
						}})}},
			{Name: svInValue, Type: "*", Required: true,
				DisplayName: "值",
				Doc:         "wildcard — 任意类型",
				Widget:      node.WidgetSpec{Kind: "text"}},
		},
		Outputs: []node.OutputSpec{
			{Name: svOutOut, Type: "Exec", DisplayName: "完成"},
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
