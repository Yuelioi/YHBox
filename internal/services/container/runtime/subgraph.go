package runtime

import (
	"errors"
	"fmt"

	"yotta/internal/services/container"
)

// ResolveSubgraphCall 解析一个 Subgraph 调用节点: 从 node.config["SubgraphID"] 拿目标子图,
// 在 container.Subgraphs 里找. 失败给清晰错误信息.
func ResolveSubgraphCall(c *container.Container, callNode *container.GraphNode) (*container.Subgraph, error) {
	if callNode == nil || (callNode.Kind != "Subgraph" && callNode.Kind != "CollapsedNode") {
		return nil, errors.New("not a Subgraph / CollapsedNode call node")
	}
	sgID := container.PinString(callNode, "SubgraphID")
	if sgID == "" {
		return nil, fmt.Errorf("Subgraph 节点 %s 缺 config.SubgraphID", callNode.ID)
	}
	for i := range c.Subgraphs {
		if c.Subgraphs[i].ID == sgID {
			return &c.Subgraphs[i], nil
		}
	}
	return nil, fmt.Errorf("subgraph %q not found in container %q", sgID, c.ID)
}
