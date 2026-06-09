// wire_asset.go — 全局资产子系统接线辅助: 引用扫描 + 存在性查询.
package main

import (
	nodepkg "yotta/internal/node"
	"yotta/internal/services/asset"
	"yotta/internal/services/container"
)

// assetExistence 返回查全局 asset 库某 GUID 是否为指定 kind 的闭包.
// 给 container.Service.SetAssetExistence 注入 (validator 存在性检查).
func assetExistence(store *asset.Store, kind string) func(guid string) bool {
	return func(guid string) bool {
		rec, ok := store.Get(guid)
		return ok && rec.Kind == kind
	}
}

// scanAssetReferrers 扫全部容器 (顶层图 + 所有子图) 找引用某 GUID 的节点位置.
// 给 asset.Service.SetReferrerScanner 注入 (删资产前的 usage 警告).
// 复用节点的 Dependencies 抽取: 任一节点的依赖 Key == guid 即记一处引用.
func scanAssetReferrers(store *container.Store) func(guid string) []asset.Referrer {
	return func(guid string) []asset.Referrer {
		var refs []asset.Referrer
		nodeRefs := func(containerID, subgraphID string, nodes []container.GraphNode) {
			for _, n := range nodes {
				rn, ok := nodepkg.Get(n.Kind)
				if !ok || rn.Dependencies == nil {
					continue
				}
				in := nodepkg.NewInputsFromConfig(n.Config)
				for _, d := range rn.Dependencies(in) {
					if (d.Kind == "template" || d.Kind == "clip") && d.Key == guid {
						refs = append(refs, asset.Referrer{
							ContainerID: containerID,
							SubgraphID:  subgraphID,
							NodeID:      n.ID,
							NodeKind:    n.Kind,
						})
					}
				}
			}
		}
		for _, c := range store.List() {
			nodeRefs(c.ID, "", c.Graph.Nodes)
			for _, sg := range c.Subgraphs {
				nodeRefs(c.ID, sg.ID, sg.Graph.Nodes)
			}
		}
		return refs
	}
}
