// compile.go — B3 三阶段中 Compile 阶段实现.
//
// 把以前散在 NewContainerRunner / dispatch_v5.go subgraph swap / listener.go newEventListener 的
// edge index / data edge index / node ID 查表构造统一到 CompileContainer 一次跑完.
// runtime dispatch 改读 r.compiled 字段, 不再 hot rebuild.
package runtime

import (
	"yhbox/internal/services/container"
)

// CompiledGraph 单 graph 的编译产物 — main graph 或一个 subgraph.
// 三件套: exec edge / data edge / 节点 ID 查表. 一次构建, runtime dispatch 直接读, 不再 hot rebuild.
//
// B2: subgraph 还含 entry/output 虚拟 marker ID (main graph 字段为零). dispatch 走 metadata
// 不再 scan Graph.Nodes 找 SubgraphInput/Output kind.
type CompiledGraph struct {
	Edges     *edgeIndex
	DataEdges *dataEdgeIndex
	NodesByID map[string]*container.GraphNode

	// B2 subgraph-only:
	EntryNodeID     string                                       // sg.Entry.NodeID; main graph = ""
	OutputDeclsByID map[string]*container.SubgraphOutputDecl // virtualNodeID → decl; main graph = nil
}

// CompiledContainer 整 container 的编译产物 — main + 所有 subgraphs + MouseCalibration snapshot.
type CompiledContainer struct {
	Main            *CompiledGraph
	Subgraphs       map[string]*CompiledGraph // sg.ID → CompiledGraph
	MainCalibCounts int
}

// CompileGraph 单 graph 编译 (main graph 用; subgraph 走 CompileSubgraph). 假设上游 ValidateContainer 已过.
//
// 注意: NodesByID 的 *GraphNode 绑回 g.Nodes 元素地址. caller 不能拷 g 之后还用此返回值
// (拷贝后 g.Nodes 是新 slice, 指针会 dangling). runtime 持的 container 不会拷, 安全.
func CompileGraph(g container.Graph) *CompiledGraph {
	nodes := make(map[string]*container.GraphNode, len(g.Nodes))
	for i := range g.Nodes {
		n := &g.Nodes[i]
		nodes[n.ID] = n
	}
	return &CompiledGraph{
		Edges:     buildEdgeIndex(g),
		DataEdges: buildDataEdgeIndex(g),
		NodesByID: nodes,
	}
}

// CompileSubgraph 单 subgraph 编译 — 多带 Entry / OutputDeclsByID metadata 给 dispatch 用.
func CompileSubgraph(sg *container.Subgraph) *CompiledGraph {
	cg := CompileGraph(sg.Graph)
	cg.EntryNodeID = sg.Entry.NodeID
	cg.OutputDeclsByID = make(map[string]*container.SubgraphOutputDecl, len(sg.OutputPins))
	for i := range sg.OutputPins {
		decl := &sg.OutputPins[i]
		if decl.NodeID != "" {
			cg.OutputDeclsByID[decl.NodeID] = decl
		}
	}
	return cg
}

// CompileContainer 完整 compile — main + 全部 subgraphs + MainCalibCounts.
// Subgraphs map key 是 sg.ID, value CompiledGraph 内含的 *GraphNode 绑 c.Subgraphs[i].Graph.Nodes 元素地址.
func CompileContainer(c *container.Container) *CompiledContainer {
	sgs := make(map[string]*CompiledGraph, len(c.Subgraphs))
	for i := range c.Subgraphs {
		sg := &c.Subgraphs[i]
		sgs[sg.ID] = CompileSubgraph(sg)
	}
	return &CompiledContainer{
		Main:            CompileGraph(c.Graph),
		Subgraphs:       sgs,
		MainCalibCounts: snapshotMainCalibCounts(c),
	}
}
