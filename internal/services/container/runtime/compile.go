// compile.go — Compile 阶段.
//
// CompileContainer 把 edge index / data edge index / node ID 查表构造一次跑完;
// runtime dispatch 读 r.compiled 字段, 不在热路径重建.
package runtime

import (
	"github.com/yottaapp/yotta/internal/node"
	"github.com/yottaapp/yotta/internal/services/container"
)

// CompiledGraph 单 graph 的编译产物 — main graph 或一个 subgraph.
// 三件套: exec edge / data edge / 节点 ID 查表. 一次构建, runtime dispatch 直接读.
//
// subgraph 还含 entry/output 虚拟 marker ID (main graph 字段为零). dispatch 走 metadata
// 不 scan Graph.Nodes 找 SubgraphInput/Output kind.
type CompiledGraph struct {
	Edges     *edgeIndex
	DataEdges *dataEdgeIndex
	NodesByID map[string]*container.GraphNode

	// subgraph-only (main graph 留零值):
	EntryNodeID     string                                   // sg.Entry.NodeID; main graph = ""
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
	return CompileGraphWithRegistry(node.DefaultRegistrySnapshot(), g)
}

func CompileGraphWithRegistry(registry node.RegistryReader, g container.Graph) *CompiledGraph {
	nodes := make(map[string]*container.GraphNode, len(g.Nodes))
	for i := range g.Nodes {
		n := &g.Nodes[i]
		nodes[n.ID] = n
	}
	return &CompiledGraph{
		Edges:     buildEdgeIndex(g),
		DataEdges: buildDataEdgeIndexWithRegistry(registry, g),
		NodesByID: nodes,
	}
}

// CompileSubgraph 单 subgraph 编译 — 多带 Entry / OutputDeclsByID metadata 给 dispatch 用.
func CompileSubgraph(sg *container.Subgraph) *CompiledGraph {
	return CompileSubgraphWithRegistry(node.DefaultRegistrySnapshot(), sg)
}

func CompileSubgraphWithRegistry(registry node.RegistryReader, sg *container.Subgraph) *CompiledGraph {
	cg := CompileGraphWithRegistry(registry, sg.Graph)
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

// CompileContainer 完整 compile — main + 解析闭包内全部 subgraphs + MainCalibCounts.
// Subgraphs map key 是 sg.ID, value CompiledGraph 内含的 *GraphNode 绑 subgraphs[i].Graph.Nodes
// 元素地址 — caller (NewContainerRunner) 持有的 rt.Subgraphs 切片在 run 生命周期内稳定.
func CompileContainer(c *container.Container, subgraphs []container.Subgraph) *CompiledContainer {
	return CompileContainerWithRegistry(node.DefaultRegistrySnapshot(), c, subgraphs)
}

func CompileContainerWithRegistry(registry node.RegistryReader, c *container.Container, subgraphs []container.Subgraph) *CompiledContainer {
	sgs := make(map[string]*CompiledGraph, len(subgraphs))
	for i := range subgraphs {
		sg := &subgraphs[i]
		sgs[sg.ID] = CompileSubgraphWithRegistry(registry, sg)
	}
	return &CompiledContainer{
		Main:            CompileGraphWithRegistry(registry, c.Graph),
		Subgraphs:       sgs,
		MainCalibCounts: snapshotMainCalibCounts(c),
	}
}
