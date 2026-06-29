package container

// GraphRef 结构化引用一个 graph, 替代散落的 "main" magic string.
// 所有 runtime / debugger / 日志 / 序列化里指向 graph 的地方都用这个, 不要直接传字符串.
type GraphRef struct {
	Kind string `json:"kind"` // "main" | "subgraph"
	ID   string `json:"id"`   // Graph.ID（subgraph 时 = subgraph.Graph.ID）
}

// GraphRefKindMain 主图常量。
const GraphRefKindMain = "main"

// GraphRefKindSubgraph 子图常量。
const GraphRefKindSubgraph = "subgraph"

// MainGraphRef 主图引用（ID 留空，因为主图没有独立 ID——它由所在 Container.ID 唯一标识）。
func MainGraphRef() GraphRef {
	return GraphRef{Kind: GraphRefKindMain, ID: ""}
}

// SubgraphGraphRef 子图引用。
func SubgraphGraphRef(subgraphID string) GraphRef {
	return GraphRef{Kind: GraphRefKindSubgraph, ID: subgraphID}
}

// IsMain 当前 ref 是否指向主图。
func (r GraphRef) IsMain() bool { return r.Kind == GraphRefKindMain }

// IsSubgraph 当前 ref 是否指向子图。
func (r GraphRef) IsSubgraph() bool { return r.Kind == GraphRefKindSubgraph }
