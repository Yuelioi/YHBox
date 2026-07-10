package runtime

import (
	"errors"
	"fmt"

	"github.com/yottaapp/yotta/internal/services/container"
)

// ResolveSubgraphCall 解析一个 Subgraph 调用节点: 从 node.config["SubgraphID"] 拿目标子图,
// 在解析闭包快照 sgs 里找. 失败给清晰错误信息.
func ResolveSubgraphCall(sgs []container.Subgraph, callNode *container.GraphNode) (*container.Subgraph, error) {
	if callNode == nil || (callNode.Kind != "Subgraph" && callNode.Kind != "CollapsedNode") {
		return nil, errors.New("call node must be Subgraph or CollapsedNode")
	}
	sgID := container.PinString(callNode, "SubgraphID")
	if sgID == "" {
		return nil, fmt.Errorf("subgraph 节点 %s 缺 config.SubgraphID", callNode.ID)
	}
	for i := range sgs {
		if sgs[i].ID == sgID {
			return &sgs[i], nil
		}
	}
	return nil, fmt.Errorf("subgraph %q not found", sgID)
}
