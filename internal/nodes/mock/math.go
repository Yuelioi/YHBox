// internal/nodes/mock/math.go
// Phase 0 用 — 验证 Inspector schema 的 number/slider widget 渲染. 无 backend.
package mock

import (
	"encoding/json"

	"yhbox/internal/node"
)

func init() {
	node.Register(&Math{})
}

type Math struct{}

const (
	mathInExec  = "in"
	mathInA     = "A"
	mathInB     = "B"
	mathInRound = "Round"
	mathOutDone = "Done"
	mathDataSum = "Sum"
)

func (Math) Spec() node.Spec {
	return node.Spec{
		Kind:        "Mock_Math",
		Version:     1,
		Category:    "Mock",
		DisplayName: "Mock 数学",
		Description: "Phase 0 测试节点 — 验证 number/slider widget. 无 backend.",
		Inputs: []node.InputSpec{
			{Name: mathInExec, Type: "Exec"},
			{Name: mathInA, Type: "Number", Default: json.Number("1"),
				DisplayName: "操作数 A",
				Widget:      node.WidgetSpec{Kind: "number"}},
			{Name: mathInB, Type: "Number", Default: json.Number("2"),
				DisplayName: "操作数 B",
				Widget: node.WidgetSpec{Kind: "slider",
					Props: node.MarshalProps(node.SliderProps{Min: 0, Max: 100, Step: 1})}},
			{Name: mathInRound, Type: "Bool", Default: false,
				DisplayName: "舍入",
				Widget:      node.WidgetSpec{Kind: "checkbox"}},
		},
		Outputs: []node.OutputSpec{
			{Name: mathOutDone, Type: "Exec", DisplayName: "完成",
				Data: []node.DataField{
					{Name: mathDataSum, Type: "Number", Doc: "A + B"},
				}},
		},
	}
}

// Run — Phase 0 mock 不真跑, 直接 Fire 第一个 exec 出口. 用于 framework 单测 / FE 视觉验证.
func (Math) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	return ctx.Out(mathOutDone).Fire(), nil
}
