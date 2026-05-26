// internal/nodes/variable/get_sys.go
// GetSys pure-data — Evaluator capability. ctx.Sys() 在 EvaluatePureData 入口
// 已被 framework wrap 成 frozen SysStore (now_ms / varLastChange.X 仍 live escape).
package variable

import (
	"fmt"
	"strings"

	"yhbox/internal/node"
	"yhbox/internal/services/container/sys"
)

func init() { node.Register(&GetSys{}) }

type GetSys struct{}

const (
	gsInPath   = "Path"
	gsOutValue = "Value"
)

func (GetSys) Spec() node.Spec {
	return node.Spec{
		Kind:        "GetSys",
		Category:    "Variable",
		DisplayName: "读系统值",
		Description: "pure-data 节点 — 读 sys 路径 (e.g. now_ms / lastDualBarTrack.innerX). 数据流求值.",
		Inputs: []node.InputSpec{
			{Name: gsInPath, Type: "String", Required: true,
				DisplayName: "路径",
				Doc:         "点路径, e.g. now_ms / lastDualBarTrack.innerX",
				Widget:      node.WidgetSpec{Kind: "text"}},
		},
		Outputs: []node.OutputSpec{
			{Name: gsOutValue, Type: "*", DisplayName: "值"},
		},
		IsPureData: true,
	}
}

func (GetSys) Evaluate(ctx node.Ctx, in node.Inputs) (any, error) {
	path := in.String(gsInPath)
	if path == "" {
		return nil, fmt.Errorf("GetSys: missing Path")
	}
	// schema 校验 — varLastChange.X / now_ms 是 live escape (不在 schema 表枚举 / 但 schema 含
	// now_ms 占位); 其他 path 必须命中 schema, 否则 silent nil 会让前端拿到 untyped nil.
	if !strings.HasPrefix(path, "varLastChange.") && path != "now_ms" {
		if _, ok := sys.PathSchema[path]; !ok {
			return nil, fmt.Errorf("GetSys: unknown sys path %q", path)
		}
	}
	v, _ := ctx.Sys().Get(path)
	return v, nil
}

