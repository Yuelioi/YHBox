// internal/nodes/control/switch.go
// Switch — 多 case 路由. Value 跟 CaseNValue (N=1..16) 逐一比较, 命中走对应 CaseN exec out;
// 全不命中走 Default. 前 4 case 默认展开, 5-16 折叠到 Advanced.
//
// Phase 5: 替换 Phase 2 stub (3 case). dynamic cases (n8n SwitchV3 风格) 未来另立 spec.
// 16 上限选定理由: fishing-v2 main swState 最多 9 case (IDLE/SETUP/WAITING/FISHING/RESULT/
// RECOVERING/SHOPSELL/BUYBAIT/CHANGEBAIT) + default, 留余量给后续容器.
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
	swInExec     = "in"
	swInValue    = "Value"
	swOutDefault = "Default"
)

func (Switch) Spec() node.Spec {
	inputs := []node.InputSpec{
		{Name: swInExec, Type: "Exec"},
		{Name: swInValue, Type: "String", Required: true,
			DisplayName: "输入值",
			Widget:      node.WidgetSpec{Kind: "text"}},
	}
	outputs := []node.OutputSpec{}

	for i := 1; i <= switchMaxCases; i++ {
		inputs = append(inputs, node.InputSpec{
			Name:        fmt.Sprintf("Case%dValue", i),
			Type:        "String",
			DisplayName: fmt.Sprintf("Case %d 值", i),
			Advanced:    i > 4, // 前 4 默认显示, 5-16 折叠到 Advanced
			Widget:      node.WidgetSpec{Kind: "text"},
		})
		outputs = append(outputs, node.OutputSpec{
			Name:        fmt.Sprintf("Case%d", i),
			Type:        "Exec",
			DisplayName: fmt.Sprintf("Case %d", i),
		})
	}
	outputs = append(outputs, node.OutputSpec{
		Name: swOutDefault, Type: "Exec", DisplayName: "Default",
	})

	return node.Spec{
		Kind:        "Switch",
		Version:     1,
		Category:    "Control",
		DisplayName: "分支 (多 Case)",
		Description: "Value 跟 Case1..Case16 值逐一匹配 (first-match-wins), 命中走对应出口; 全不命中走 Default. 前 4 case 默认显示, 5-16 在 Advanced.",
		Inputs:      inputs,
		Outputs:     outputs,
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
