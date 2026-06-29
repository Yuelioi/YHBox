// MouseCalibration — 声明式校准节点. 没 exec-in, 单 exec-out "Done" 表达
// declarative passthrough 语义 (仅一个 counts360 数值字段).
//
// 主图唯一性约束不在节点内做 — 由 graph validator 做 cross-node check.
package system

import (
	"encoding/json"

	"yotta/internal/node"
)

func init() { node.Register(&MouseCalibration{}) }

type MouseCalibration struct{}

const (
	mcInCounts360 = "Counts360"
	mcOutDone     = "Done"
)

func (MouseCalibration) Spec() node.Spec {
	return node.Spec{
		Kind:     "MouseCalibration",
		Category: "System",
		Inputs: []node.InputSpec{
			{Name: mcInCounts360, Type: "Number", Default: json.Number("0"),
				Widget: node.WidgetSpec{Kind: "number"}},
		},
		Outputs: []node.OutputSpec{
			{Name: mcOutDone, Type: "Exec"},
		},
	}
}

func (MouseCalibration) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	// declarative node — runtime 已在启动期读 counts360, 这里 no-op.
	return ctx.Out(mcOutDone).Fire(), nil
}
