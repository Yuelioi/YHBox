// internal/nodes/variable/get_sys.go
// GetSys pure-data — pull-based evaluator reads system constant by dot-path
// (e.g. "now_ms", "lastBarTrack.cursorX"). Phase 5 wire 加 pull-eval.
package variable

import "yhbox/internal/node"

func init() { node.Register(&GetSys{}) }

type GetSys struct{}

const (
	gsInPath   = "path"
	gsOutValue = "value"
)

func (GetSys) Spec() node.Spec {
	return node.Spec{
		Kind:        "GetSys",
		Version:     1,
		Category:    "Variable",
		DisplayName: "读系统值",
		Description: "pure-data 节点 — 读 sys 路径 (e.g. now_ms / lastBarTrack.cursorX). 数据流求值.",
		Inputs: []node.InputSpec{
			{Name: gsInPath, Type: "String", Required: true,
				DisplayName: "路径",
				Doc:         "点路径, e.g. now_ms / lastBarTrack.cursorX",
				Widget:      node.WidgetSpec{Kind: "text"}},
		},
		Outputs: []node.OutputSpec{
			{Name: gsOutValue, Type: "*", DisplayName: "值"},
		},
		IsPureData: true,
	}
}

func (GetSys) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	return nil, errPureDataNotEvaluatable
}
