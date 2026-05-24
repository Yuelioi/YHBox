// internal/nodes/control/switch.go
// Switch 基础版 — 3 个固定 case 出口 + default. Phase 5 真做 dynamic cases (n8n SwitchV3 风格).
package control

import (
	"fmt"

	"yhbox/internal/node"
)

func init() { node.Register(&Switch{}) }

type Switch struct{}

const (
	swInExec     = "in"
	swInValue    = "Value"
	swInCase1    = "Case1Value"
	swInCase2    = "Case2Value"
	swInCase3    = "Case3Value"
	swOutCase1   = "Case1"
	swOutCase2   = "Case2"
	swOutCase3   = "Case3"
	swOutDefault = "Default"
)

func (Switch) Spec() node.Spec {
	return node.Spec{
		Kind:        "Switch",
		Version:     1,
		Category:    "Control",
		DisplayName: "分支",
		Description: "Value 跟 Case1Value/Case2Value/Case3Value 逐一匹配, 命中走对应出口, 否则走 Default. Phase 2 基础版仅 3 case + default; Phase 5 加 dynamic cases.",
		Inputs: []node.InputSpec{
			{Name: swInExec, Type: "Exec"},
			{Name: swInValue, Type: "String", Required: true,
				DisplayName: "输入值",
				Widget:      node.WidgetSpec{Kind: "text"}},
			{Name: swInCase1, Type: "String", DisplayName: "Case 1 值",
				Widget: node.WidgetSpec{Kind: "text"}},
			{Name: swInCase2, Type: "String", DisplayName: "Case 2 值",
				Widget: node.WidgetSpec{Kind: "text"}},
			{Name: swInCase3, Type: "String", DisplayName: "Case 3 值",
				Widget: node.WidgetSpec{Kind: "text"}},
		},
		Outputs: []node.OutputSpec{
			{Name: swOutCase1, Type: "Exec", DisplayName: "Case 1"},
			{Name: swOutCase2, Type: "Exec", DisplayName: "Case 2"},
			{Name: swOutCase3, Type: "Exec", DisplayName: "Case 3"},
			{Name: swOutDefault, Type: "Exec", DisplayName: "Default"},
		},
	}
}

func (Switch) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	v := in.String(swInValue)
	switch {
	case in.Has(swInCase1) && v == in.String(swInCase1):
		return ctx.Out(swOutCase1).Fire(), nil
	case in.Has(swInCase2) && v == in.String(swInCase2):
		return ctx.Out(swOutCase2).Fire(), nil
	case in.Has(swInCase3) && v == in.String(swInCase3):
		return ctx.Out(swOutCase3).Fire(), nil
	}
	return ctx.Out(swOutDefault).Fire(), nil
}

func (Switch) Display(in node.Inputs, exitName string, out node.OutputData) string {
	return fmt.Sprintf("switch %q → %s", in.String(swInValue), exitName)
}
