// internal/nodes/system/mouse_calibration.go
// MouseCalibration — 声明式校准节点. 老 spec (nodekind/specs/system.go) ExecIn=nil,
// ExecOut=nil — runtime 启动期消费 config.counts360 供 MouseMoveRel scaling 用,
// 执行流直通 (runtime/nodes.go::case "MouseCalibration" → return nil, nil).
//
// 新框架保持 declarative 形态: 没 exec-in, 单 exec-out "Fire" 表达 declarative
// passthrough 语义 (跟老 Defaults.counts360 一致, 仅一个 counts360 数值字段).
//
// 主图唯一性约束 (validator MOUSE_CAL_DUPLICATE) 不在节点内做 — Phase 5
// graph validator 加 cross-node check.
package system

import (
	"encoding/json"

	"yhbox/internal/node"
)

func init() { node.Register(&MouseCalibration{}) }

type MouseCalibration struct{}

const (
	mcInCounts360 = "Counts360"
	mcOutFire     = "Fire"
)

func (MouseCalibration) Spec() node.Spec {
	return node.Spec{
		Kind:        "MouseCalibration",
		Category:    "System",
		DisplayName: "鼠标校准",
		Description: "声明式 — runtime 启动期读 counts360 供 MouseMoveRel scaling 用. 节点本体 no-op, 走 Fire 出口表达 passthrough.",
		Inputs: []node.InputSpec{
			{Name: mcInCounts360, Type: "Number", Default: json.Number("0"),
				DisplayName: "counts/360°",
				Doc:         "鼠标 360° 转一圈所需 counts. MouseMoveRel 用来 scale dx/dy.",
				Widget:      node.WidgetSpec{Kind: "text"}},
		},
		Outputs: []node.OutputSpec{
			{Name: mcOutFire, Type: "Exec", DisplayName: "Fire"},
		},
	}
}

func (MouseCalibration) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	// declarative node — runtime 已在启动期读 counts360, 这里 no-op.
	return ctx.Out(mcOutFire).Fire(), nil
}
