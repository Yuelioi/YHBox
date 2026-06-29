package container

import (
	"strings"

	"yotta/internal/automation/target"
)

// win32WindowTargetForNode 求编辑期工具该用的窗口 = 该节点最近上游 Win32WindowTarget.
// 规则: 沿 exec 边从 nodeID 反向 BFS, 第一层遇到的 Win32WindowTarget; 唯一 → 用它;
// 0/多个 或 nodeID 空 → 回落主窗口 (Graph.Nodes 数组序第一个 Win32WindowTarget).
// 已知限制: 被编辑节点在子图里时, 反向 BFS 只遍历主图 c.Graph, 子图节点会回落主窗口
// (编辑期工具截图限制, 非目标范围).
func win32WindowTargetForNode(c *Container, nodeID string) *GraphNode {
	mainWT := firstMainWin32WindowTarget(c)
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
			if n != nil && n.Kind == "Win32WindowTarget" {
				found[id] = n
				continue // 不越过 Win32WindowTarget 继续往上
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

func firstMainWin32WindowTarget(c *Container) *GraphNode {
	for i := range c.Graph.Nodes {
		if c.Graph.Nodes[i].Kind == "Win32WindowTarget" {
			return &c.Graph.Nodes[i]
		}
	}
	return nil
}

func editorTargetKindForNode(c *Container, nodeID string) (string, bool) {
	if c == nil {
		return target.KindWin32Window, true
	}
	nodeByID := graphNodeByID(c.Graph)
	if nodeID != "" {
		if n := nodeByID[nodeID]; n != nil {
			if kind, ok := targetKindForSelectionNode(n.Kind); ok {
				return kind, true
			}
		}
		if kind, ok := nearestUpstreamTargetKind(c.Graph, nodeByID, nodeID); ok {
			return kind, true
		}
	}
	if kind, ok := firstTargetKind(c.Graph); ok {
		return kind, true
	}
	return target.KindWin32Window, true
}

func firstTargetKind(g Graph) (string, bool) {
	for i := range g.Nodes {
		if kind, ok := targetKindForSelectionNode(g.Nodes[i].Kind); ok {
			return kind, true
		}
	}
	return "", false
}

func splitPinNode(pin string) string {
	if i := strings.IndexByte(pin, '.'); i >= 0 {
		return pin[:i]
	}
	return pin
}
