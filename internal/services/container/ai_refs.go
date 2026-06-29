package container

// AIConnectionRef 引用某 AI 连接的节点定位(删连接前列给用户, 避免静默失效)。
type AIConnectionRef struct {
	ContainerID   string `json:"containerId"`
	ContainerName string `json:"containerName"`
	NodeID        string `json:"nodeId"`
}

// ScanAIConnectionRefs 扫容器列表, 返回所有 Kind=="AI" 且 Connection 字面 == connectionID
// 的节点引用。settings 删 AI 连接前调, 让用户知情被哪些节点引用(不静默失效)。
// 只扫容器主图; 全局子图池里的 AI 节点不在此(子图引用另算)。
func ScanAIConnectionRefs(containers []Container, connectionID string) []AIConnectionRef {
	if connectionID == "" {
		return nil
	}
	var refs []AIConnectionRef
	for i := range containers {
		c := &containers[i]
		for j := range c.Graph.Nodes {
			n := &c.Graph.Nodes[j]
			if n.Kind == "AI" && PinString(n, "Connection") == connectionID {
				refs = append(refs, AIConnectionRef{
					ContainerID:   c.ID,
					ContainerName: c.Name,
					NodeID:        n.ID,
				})
			}
		}
	}
	return refs
}
