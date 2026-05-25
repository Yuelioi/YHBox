package container

import "strings"

// GraphRewriter 统一 graph 改写抽象。
//
// 调用点:
//   - Copy-on-use 库 → 容器: subgraph clone 时所有内部节点 id 重发；template key 冲突时 rename + patch；
//     SubgraphOutputDecl ID 也要重发 → SubgraphOutput 节点 config.declID 必须同步 patch
//   - 折叠为 Subgraph: 父图选中节点搬进新 subgraph, 新 subgraph 内 id 重发
//   - 粘贴: 复制的 nodes 在目标 graph 内可能 id 冲突, 重发 id
//
// 一次性记录所有要 rename 的映射, 最后 Apply 到 graph 一次性应用.
// 集中改写避免散落字符串 patch 容易漏 + 难单测.
type GraphRewriter struct {
	nodeIDMap      map[string]string // oldID → newID
	templateKeyMap map[string]string // oldKey → newKey
	declIDMap      map[string]string // SubgraphOutputDecl.ID oldID → newID（B-3 修复）
}

// NewGraphRewriter 创建空 rewriter。
func NewGraphRewriter() *GraphRewriter {
	return &GraphRewriter{
		nodeIDMap:      map[string]string{},
		templateKeyMap: map[string]string{},
		declIDMap:      map[string]string{},
	}
}

// RenameNodeID 注册一个节点 id rename。
func (r *GraphRewriter) RenameNodeID(old, new string) {
	if old == new || old == "" || new == "" {
		return
	}
	r.nodeIDMap[old] = new
}

// RenameTemplateKey 注册一个 template key rename。
func (r *GraphRewriter) RenameTemplateKey(old, new string) {
	if old == new || old == "" || new == "" {
		return
	}
	r.templateKeyMap[old] = new
}

// RenameDeclID 注册一个 SubgraphOutputDecl ID rename.
// 库 → 容器 copy-on-use 时, OutputPins 的 ID 全部重发,
// 内部 SubgraphOutput 节点 config.declID 必须跟着改, 否则父图调用边引用旧 ID 找不到下游.
func (r *GraphRewriter) RenameDeclID(old, new string) {
	if old == new || old == "" || new == "" {
		return
	}
	r.declIDMap[old] = new
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
		// 2. 节点 config 里 template key 改写
		if newKey, ok := r.templateKeyMap[stringFromConfig(g.Nodes[i].Config, "template")]; ok {
			if g.Nodes[i].Config != nil {
				g.Nodes[i].Config["template"] = newKey
			}
		}
		// 3. SubgraphOutput 节点 config.declID 改写
		if g.Nodes[i].Kind == "SubgraphOutput" {
			if newDeclID, ok := r.declIDMap[stringFromConfig(g.Nodes[i].Config, "DeclID")]; ok {
				if g.Nodes[i].Config != nil {
					g.Nodes[i].Config["DeclID"] = newDeclID
				}
			}
		}
	}

	// 4. edges 的 from/to 引用 nodeID 改写
	for i := range g.Edges {
		g.Edges[i].From = rewriteEdgeRef(g.Edges[i].From, r.nodeIDMap)
		g.Edges[i].To = rewriteEdgeRef(g.Edges[i].To, r.nodeIDMap)
	}
}

// stringFromConfig 取 config[key] 的字符串值，缺失返空串。
func stringFromConfig(cfg map[string]any, key string) string {
	if cfg == nil {
		return ""
	}
	if v, ok := cfg[key].(string); ok {
		return v
	}
	return ""
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
