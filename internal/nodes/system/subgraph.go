// Package system 系统类节点 (Subgraph / OnEvent / Throw / Try / WindowTarget / MouseCalibration / CommentBox).
// Phase 5 加 RegionRunner 真做 Subgraph + Try.
package system

import (
	"errors"

	"yhbox/internal/node"
)

func init() { node.Register(&Subgraph{}) }

// Subgraph 占位节点 — Phase 5 用 RegionRunner 接口真做.
// 此版本 Run 直接返 error, 阻止用户在 graph 里实际跑.
type Subgraph struct{}

const (
	sgInExec       = "in"
	sgInSubgraphID = "SubgraphID"
	sgOutDone      = "Done"
)

func (Subgraph) Spec() node.Spec {
	return node.Spec{
		Kind:        "Subgraph",
		Version:     1,
		Category:    "System",
		DisplayName: "子图调用",
		Description: "(Phase 2 stub) Phase 5 真做 — 调用另一组节点. 现阶段在 graph 里加这节点会 runtime error.",
		Inputs: []node.InputSpec{
			{Name: sgInExec, Type: "Exec"},
			{Name: sgInSubgraphID, Type: "String", Required: true,
				DisplayName: "子图 ID",
				Doc:         "目标子图标识符, Phase 5 实做后才生效",
				Widget:      node.WidgetSpec{Kind: "text"}},
		},
		Outputs: []node.OutputSpec{
			{Name: sgOutDone, Type: "Exec", DisplayName: "完成 (Phase 5)"},
		},
	}
}

var errSubgraphNotImplemented = errors.New("Subgraph node 未实现 — Phase 5 加 RegionRunner 接口后才能跑")

func (Subgraph) Run(ctx node.Ctx, in node.Inputs) (node.Outputs, error) {
	return nil, errSubgraphNotImplemented
}
