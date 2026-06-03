package container

import "strings"

// windowTargetForNode 求编辑期工具该用的窗口 = 该节点最近上游 WindowTarget.
// 规则: 沿 exec 边从 nodeID 反向 BFS, 第一层遇到的 WindowTarget; 唯一 → 用它;
// 0/多个 或 nodeID 空 → 回落主窗口 (Graph.Nodes 数组序第一个 WindowTarget).
// 已知限制: 被编辑节点在子图里时, 反向 BFS 只遍历主图 c.Graph, 子图节点会回落主窗口
// (编辑期工具截图限制, 非目标范围).
func windowTargetForNode(c *Container, nodeID string) *GraphNode {
	mainWT := firstMainWindowTarget(c)
	if nodeID == "" {
		return mainWT
	}
	// 反向 exec 邻接: toNode → []fromNode.
	rev := map[string][]string{}
	for _, e := range c.Graph.Edges {
		rev[splitPinNode(e.To)] = append(rev[splitPinNode(e.To)], splitPinNode(e.From))
	}
	byID := map[string]*GraphNode{}
	for i := range c.Graph.Nodes {
		byID[c.Graph.Nodes[i].ID] = &c.Graph.Nodes[i]
	}
	seen := map[string]bool{nodeID: true}
	frontier := append([]string{}, rev[nodeID]...)
	found := map[string]*GraphNode{}
	for len(frontier) > 0 {
		next := []string{}
		for _, id := range frontier {
			if seen[id] {
				continue
			}
			seen[id] = true
			n := byID[id]
			if n != nil && n.Kind == "WindowTarget" {
				found[id] = n
				continue // 不越过 WindowTarget 继续往上
			}
			next = append(next, rev[id]...)
		}
		if len(found) > 0 {
			break
		}
		frontier = next
	}
	if len(found) == 1 {
		for _, n := range found {
			return n
		}
	}
	return mainWT
}

func firstMainWindowTarget(c *Container) *GraphNode {
	for i := range c.Graph.Nodes {
		if c.Graph.Nodes[i].Kind == "WindowTarget" {
			return &c.Graph.Nodes[i]
		}
	}
	return nil
}

func splitPinNode(pin string) string {
	if i := strings.IndexByte(pin, '.'); i >= 0 {
		return pin[:i]
	}
	return pin
}
