// internal/nodes/control/switch.go
// Switch — 多 case 路由. Value 跟 CaseNValue (N=1..16) 逐一比较, 命中走对应 CaseN exec out;
// 全不命中走 Default. 前 4 case 默认展开, 5-16 折叠到 Advanced.
//
// 16 case 上限: 留足典型状态机 (~9 状态 + default) 的余量.
//
// 跟 Loop 不同 — Switch 不是 region (无 body), 走 Run() 而非 RunRegion().
package control

import (
	"fmt"

	"yhbox/internal/node"
)

func init() { node.Register(&Switch{}) }

type Switch struct{}

// Switch 最多支持 N 个 case 出口. 改这个值要同时调 Spec 生成 + tests.
const switchMaxCases = 16

const (
	swInExec     = "In"
	swInValue    = "Value"
	swOutDefault = "Default"
)

func (Switch) Spec() node.Spec {
	inputs := []node.InputSpec{
		{Name: swInExec, Type: "Exec"},
		{Name: swInValue, Type: "String", Required: true,
			Widget: node.WidgetSpec{Kind: "text"}},
	}
	outputs := []node.OutputSpec{}

	for i := 1; i <= switchMaxCases; i++ {
		inputs = append(inputs, node.InputSpec{
			Name:     fmt.Sprintf("Case%dValue", i),
			Type:     "String",
			Advanced: i > 4, // 前 4 默认显示, 5-16 折叠到 Advanced
			Widget:   node.WidgetSpec{Kind: "text"},
		})
		outputs = append(outputs, node.OutputSpec{
			Name: fmt.Sprintf("Case%d", i),
			Type: "Exec",
		})
	}
	outputs = append(outputs, node.OutputSpec{
		Name: swOutDefault, Type: "Exec",
	})

	return node.Spec{
		Kind:     "Switch",
		Category: "Control",
		Inputs:   inputs,
		Outputs:  outputs,
	}
}

func (Switch) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	v := in.String(swInValue)
	for i := 1; i <= switchMaxCases; i++ {
		caseField := fmt.Sprintf("Case%dValue", i)
		if in.Has(caseField) && v == in.String(caseField) {
			return ctx.Out(fmt.Sprintf("Case%d", i)).Fire(), nil
		}
	}
	return ctx.Out(swOutDefault).Fire(), nil
}

func (Switch) Display(in node.Inputs, exitName string, out node.OutputData) string {
	return fmt.Sprintf("switch %q → %s", in.String(swInValue), exitName)
}
