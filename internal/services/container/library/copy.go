package library

import (
	"fmt"
	"maps"

	"github.com/google/uuid"

	"yhbox/internal/services/container"
)

// CopyResult copy-on-use 一次拷贝产生的所有改写映射。
type CopyResult struct {
	NewSubgraph    *container.Subgraph
	TemplateKeyMap map[string]string
}

// genIDFn 抽出来方便单元测注入确定性 id。生产用 uuid.NewString。
type genIDFn func(b []byte) string

// CopySubgraph 把库 subgraph 克隆一份给目标容器。
func CopySubgraph(
	src *LibrarySubgraph,
	target *container.Container,
	existingKeys map[string]struct{},
	idGen genIDFn,
) (*CopyResult, error) {
	if src == nil || target == nil {
		return nil, fmt.Errorf("nil src or target")
	}
	if idGen == nil {
		idGen = func(b []byte) string { return uuid.NewString() }
	}

	cp := src.Subgraph
	cp.Graph.Nodes = append([]container.GraphNode(nil), src.Graph.Nodes...)
	// 深拷贝每个 node 的 Config map，避免共享底层引用导致 Apply 改 cp 时也改了 src
	for i := range cp.Graph.Nodes {
		if cp.Graph.Nodes[i].Config != nil {
			cp.Graph.Nodes[i].Config = maps.Clone(cp.Graph.Nodes[i].Config)
		}
	}
	cp.Graph.Edges = append([]container.GraphEdge(nil), src.Graph.Edges...)

	rewriter := container.NewGraphRewriter()

	for _, n := range src.Graph.Nodes {
		rewriter.RenameNodeID(n.ID, idGen(nil))
	}
	cp.ID = "sg-" + idGen(nil)[:8]
	cp.Graph.ID = idGen(nil)
	// OutputPins ID 全部重发；内部 SubgraphOutput 节点 config.declID 通过 rewriter.RenameDeclID
	// 跟随改写（否则父图调用边的 declID 引用对不上 → runtime 找不到下游，B-3 修复）
	newOutputPins := make([]container.SubgraphOutputDecl, len(src.OutputPins))
	for i, d := range src.OutputPins {
		newID := idGen(nil)
		newOutputPins[i] = container.SubgraphOutputDecl{ID: newID, Name: d.Name}
		rewriter.RenameDeclID(d.ID, newID)
	}
	cp.OutputPins = newOutputPins

	templateKeyMap := map[string]string{}
	referencedTemplates := collectReferencedTemplates(&cp.Graph)
	for oldKey := range referencedTemplates {
		if _, conflict := existingKeys[oldKey]; conflict {
			newKey := pickNonConflictKey(oldKey, existingKeys)
			rewriter.RenameTemplateKey(oldKey, newKey)
			templateKeyMap[oldKey] = newKey
			existingKeys[newKey] = struct{}{}
		}
	}

	rewriter.Apply(&cp.Graph)

	return &CopyResult{
		NewSubgraph:    &cp,
		TemplateKeyMap: templateKeyMap,
	}, nil
}

// collectReferencedTemplates 扫描子图节点，找出所有 config.template 引用的 key。
func collectReferencedTemplates(g *container.Graph) map[string]struct{} {
	out := map[string]struct{}{}
	for _, n := range g.Nodes {
		switch n.Kind {
		case "WaitTemplate", "CheckTemplate", "ClickTemplate", "OnEvent":
			if k, _ := n.Config["template"].(string); k != "" {
				out[k] = struct{}{}
			}
		}
	}
	return out
}

// pickNonConflictKey 找 oldKey_2 / oldKey_3 / ... 直到不冲突。
func pickNonConflictKey(oldKey string, existing map[string]struct{}) string {
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s_%d", oldKey, i)
		if _, ok := existing[candidate]; !ok {
			return candidate
		}
	}
}
