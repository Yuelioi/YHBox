package runtime

import (
	"errors"
	"fmt"

	"yhbox/internal/services/container"
)

// ResolveSubgraphCall 解析一个 Subgraph 调用节点：从 node.config["subgraphId"] 拿目标子图，
// 在 container.Subgraphs 里找。失败给清晰的错误信息。
func ResolveSubgraphCall(c *container.Container, callNode *container.GraphNode) (*container.Subgraph, error) {
	if callNode == nil || (callNode.Kind != "Subgraph" && callNode.Kind != "CollapsedNode") {
		return nil, errors.New("not a Subgraph / CollapsedNode call node")
	}
	sgID, _ := callNode.Config["subgraphId"].(string)
	if sgID == "" {
		return nil, fmt.Errorf("Subgraph 节点 %s 缺 config.subgraphId", callNode.ID)
	}
	for i := range c.Subgraphs {
		if c.Subgraphs[i].ID == sgID {
			return &c.Subgraphs[i], nil
		}
	}
	return nil, fmt.Errorf("subgraph %q not found in container %q", sgID, c.ID)
}

// FindParentDownstreamByDeclID 子图里 SubgraphOutput 节点跑完后，
// 要找父图里"调用方那条边"对应的下游节点入 pin。
// parentEdges：父图边列表。callNodeID：父图里 Subgraph 调用节点 id。declID：子图 OutputPin Decl 的 id。
// 返回 "<nextNodeID>.<inPin>"；找不到返回空串。
func FindParentDownstreamByDeclID(parentEdges []container.GraphEdge, callNodeID, declID string) string {
	key := callNodeID + "." + declID
	for _, e := range parentEdges {
		if e.From == key {
			return e.To
		}
	}
	return ""
}
