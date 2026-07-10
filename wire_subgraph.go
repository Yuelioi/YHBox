// wire_subgraph.go — 全局子图池接线辅助 (2026-06-12 全局化):
// 正向闭包解析 / 反向 referrer 扫描 / 匿名 GC 的引用集.
// 都建立在 dependency 包的节点级提取 primitive 上 (单一咽喉, 不内联递归).
package main

import (
	"github.com/yottaapp/yotta/internal/services/container"
	"github.com/yottaapp/yotta/internal/services/container/dependency"
)

// depNodeInfos container.GraphNode → dependency.NodeInfo 适配.
func depNodeInfos(nodes []container.GraphNode) []dependency.NodeInfo {
	out := make([]dependency.NodeInfo, 0, len(nodes))
	for i := range nodes {
		out = append(out, dependency.NodeInfo{Kind: nodes[i].Kind, Config: nodes[i].Config})
	}
	return out
}

// subgraphClosureFor 解析容器引用闭包 → 子图快照集.
// 运行时 bundle / 容器校验共用. 闭包缺失的 ID (池里没有) 自然跳过 — validator 报 MISSING_SUBGRAPH.
func subgraphClosureFor(c *container.Container, sgStore *container.SubgraphStore) []container.Subgraph {
	res, err := dependency.Closure(depNodeInfos(c.Graph.Nodes), func(sgID string) ([]dependency.NodeInfo, error) {
		sg, ok := sgStore.Get(sgID)
		if !ok {
			return nil, nil
		}
		return depNodeInfos(sg.Graph.Nodes), nil
	})
	if err != nil {
		return nil
	}
	out := make([]container.Subgraph, 0, len(res.Subgraphs))
	for _, id := range res.Subgraphs {
		if sg, ok := sgStore.Get(id); ok {
			out = append(out, sg)
		}
	}
	return out
}

// scanSubgraphReferrers 反向 referrer: 谁引用了子图 guid — 全部容器顶层图 + 全池子图图体,
// 内存遍历 (非磁盘解析). 给 SubgraphService.SetReferrerScanner (删除前警告 + 库页引用计数).
func scanSubgraphReferrers(cStore *container.Store, sgStore *container.SubgraphStore) func(sgID string) []container.SubgraphReferrer {
	return func(sgID string) []container.SubgraphReferrer {
		var refs []container.SubgraphReferrer
		match := func(containerID, inSubgraphID string, nodes []container.GraphNode) {
			for i := range nodes {
				n := &nodes[i]
				for _, dep := range nodeSubgraphDeps(n) {
					if dep == sgID {
						refs = append(refs, container.SubgraphReferrer{
							ContainerID: containerID, SubgraphID: inSubgraphID,
							NodeID: n.ID, NodeKind: n.Kind,
						})
					}
				}
			}
		}
		for _, c := range cStore.List() {
			match(c.ID, "", c.Graph.Nodes)
		}
		for _, sg := range sgStore.List() {
			if sg.ID == sgID {
				continue // 自引用不算外部 referrer (环由 validator 拦)
			}
			match("", sg.ID, sg.Graph.Nodes)
		}
		return refs
	}
}

// collectReferencedSubgraphIDs 匿名 GC 的 mark 集: 全部容器图 + 全池子图图体直接引用到的
// 子图 ID (扫描时点计算, 无持久计数).
func collectReferencedSubgraphIDs(cStore *container.Store, sgStore *container.SubgraphStore) map[string]bool {
	referenced := map[string]bool{}
	scan := func(nodes []container.GraphNode) {
		for i := range nodes {
			for _, dep := range nodeSubgraphDeps(&nodes[i]) {
				referenced[dep] = true
			}
		}
	}
	for _, c := range cStore.List() {
		scan(c.Graph.Nodes)
	}
	for _, sg := range sgStore.List() {
		scan(sg.Graph.Nodes)
	}
	return referenced
}

// nodeSubgraphDeps 单节点引用的子图 ID 列表 — 走 dependency 包同款节点级提取
// (图节点 Dependencies + 脚本 assetdeps 单一源).
func nodeSubgraphDeps(n *container.GraphNode) []string {
	infos := []dependency.NodeInfo{{Kind: n.Kind, Config: n.Config}}
	res, err := dependency.Closure(infos, func(string) ([]dependency.NodeInfo, error) {
		return nil, nil // 只要直接引用, 不递归
	})
	if err != nil {
		return nil
	}
	return res.Subgraphs
}
