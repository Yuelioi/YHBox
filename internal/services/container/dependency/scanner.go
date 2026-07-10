package dependency

import (
	"fmt"

	nodepkg "github.com/yottaapp/yotta/internal/node"
)

// NodeInfo 节点的最小描述, 供 ScanSubgraphDependencies 迭代. 跟 container.GraphNode 解耦
// (避免 dependency → container → … 闭环), 由 caller 适配.
type NodeInfo struct {
	Kind   string
	Config map[string]any
}

// ScanContainerDependencies 从容器顶层图节点出发递归扫全部依赖. BFS, dedupe, cyclic-safe.
// 用于容器级导出: 闭包根 = 顶层图节点引用集, 顶层的 Subgraph-call 节点递归进所有子图,
// 收齐整容器的 template/clip GUID + subgraph 闭包.
//
// topNodes: 容器顶层图 (Container.Graph) 的节点. getNodes: sgID → 子图节点 (同 ScanSubgraphDependencies).
// 返 []Dependency 含顶层引用的全部 subgraph / template / clip + 它们 transitively 引用的.
func ScanContainerDependencies(
	topNodes []NodeInfo,
	getNodes func(sgID string) ([]NodeInfo, error),
) ([]Dependency, error) {
	visited := map[string]bool{}
	seenDeps := map[string]bool{}
	var allDeps []Dependency
	var queue []string

	// 先扫顶层节点, 收直接依赖 + 把 subgraph-call 入队.
	emit := func(nodes []NodeInfo) {
		for _, n := range nodes {
			rn, ok := nodepkg.Get(n.Kind)
			if !ok || rn.Dependencies == nil {
				continue
			}
			in := nodepkg.NewInputsFromConfig(n.Config)
			for _, raw := range rn.Dependencies(in) {
				d := Dependency{Kind: Kind(raw.Kind), Key: raw.Key}
				if seenDeps[d.String()] {
					continue
				}
				seenDeps[d.String()] = true
				allDeps = append(allDeps, d)
				if d.Kind == KindSubgraph {
					queue = append(queue, d.Key)
				}
			}
		}
	}
	emit(topNodes)

	// BFS 子图闭包.
	for len(queue) > 0 {
		sgID := queue[0]
		queue = queue[1:]
		if visited[sgID] {
			continue
		}
		visited[sgID] = true
		nodes, err := getNodes(sgID)
		if err != nil {
			return nil, fmt.Errorf("get subgraph %q: %w", sgID, err)
		}
		emit(nodes)
	}
	return allDeps, nil
}

// ScanSubgraphDependencies 递归扫子图依赖. BFS, dedupe, cyclic-safe.
//
// getNodes: 给定 sgID 返回该子图的全部节点; 不存在时返 nil, nil — 跳过, 不报错.
// 节点 kind → deps 走 nodepkg.Get(kind).Dependencies(NewInputsFromConfig(cfg)) — 单一源
// 跟 framework Dependencies(in) 接口对齐. 节点未注册 (kind 未知) 或未实现 Dependencer 接口
// → 0 依赖.
//
// 返 []Dependency 含 root subgraph 自身 + 所有 transitively 引用的 subgraph / template / clip.
func ScanSubgraphDependencies(
	rootSgID string,
	getNodes func(sgID string) ([]NodeInfo, error),
) ([]Dependency, error) {
	visited := map[string]bool{}
	seenDeps := map[string]bool{}
	var allDeps []Dependency
	queue := []string{rootSgID}

	for len(queue) > 0 {
		sgID := queue[0]
		queue = queue[1:]
		if visited[sgID] {
			continue
		}
		visited[sgID] = true

		// 自身也是 KindSubgraph 依赖
		selfDep := Dependency{Kind: KindSubgraph, Key: sgID}
		if !seenDeps[selfDep.String()] {
			seenDeps[selfDep.String()] = true
			allDeps = append(allDeps, selfDep)
		}

		nodes, err := getNodes(sgID)
		if err != nil {
			return nil, fmt.Errorf("get subgraph %q: %w", sgID, err)
		}
		for _, n := range nodes {
			rn, ok := nodepkg.Get(n.Kind)
			if !ok || rn.Dependencies == nil {
				continue
			}
			in := nodepkg.NewInputsFromConfig(n.Config)
			for _, raw := range rn.Dependencies(in) {
				d := Dependency{Kind: Kind(raw.Kind), Key: raw.Key}
				if seenDeps[d.String()] {
					continue
				}
				seenDeps[d.String()] = true
				allDeps = append(allDeps, d)
				if d.Kind == KindSubgraph {
					queue = append(queue, d.Key)
				}
			}
		}
	}
	return allDeps, nil
}
