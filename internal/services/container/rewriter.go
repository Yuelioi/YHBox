package container

import "strings"

// GraphRewriter 统一 graph 改写抽象。
//
// 调用点:
//   - Copy-on-use 库 → 容器: subgraph clone 时所有内部节点 id 重发.
//   - 折叠为 Subgraph: 父图选中节点搬进新 subgraph, 新 subgraph 内 id 重发.
//   - 粘贴: 复制的 nodes 在目标 graph 内可能 id 冲突, 重发 id.
//
// 一次性记录所有要 rename 的映射, 最后 Apply 到 graph 一次性应用.
// 集中改写避免散落字符串 patch 容易漏 + 难单测.
//
// SubgraphOutput 节点 config.DeclID 不经此 rewriter (节点不在 Graph.Nodes; OutputPins 的 ID
// 由 caller 直接 mutate).
type GraphRewriter struct {
	nodeIDMap map[string]string // oldID → newID
}

// NewGraphRewriter 创建空 rewriter。
func NewGraphRewriter() *GraphRewriter {
	return &GraphRewriter{
		nodeIDMap: map[string]string{},
	}
}

// RenameNodeID 注册一个节点 id rename。
func (r *GraphRewriter) RenameNodeID(old, new string) {
	if old == new || old == "" || new == "" {
		return
	}
	r.nodeIDMap[old] = new
}

// Apply 应用所有 rename 到给定 graph。原地修改。
func (r *GraphRewriter) Apply(g *Graph) {
	if g == nil {
		return
	}
	// 1. 节点 id 改写
	for i := range g.Nodes {
		if newID, ok := r.nodeIDMap[g.Nodes[i].ID]; ok {
			g.Nodes[i].ID = newID
		}
	}

	// 2. edges 的 from/to 引用 nodeID 改写
	for i := range g.Edges {
		g.Edges[i].From = rewriteEdgeRef(g.Edges[i].From, r.nodeIDMap)
		g.Edges[i].To = rewriteEdgeRef(g.Edges[i].To, r.nodeIDMap)
	}
}

// rewriteEdgeRef "<nodeID>.<pin>" 格式里把 nodeID 改写。
func rewriteEdgeRef(ref string, idMap map[string]string) string {
	dot := strings.IndexByte(ref, '.')
	if dot < 0 {
		return ref
	}
	nodeID := ref[:dot]
	pin := ref[dot:]
	if newID, ok := idMap[nodeID]; ok {
		return newID + pin
	}
	return ref
}
