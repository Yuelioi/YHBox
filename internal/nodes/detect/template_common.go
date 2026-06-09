// template_common.go — WaitTemplate / ClickTemplate / CheckTemplate 共用的多模板辅助.
// 三个节点的 Template 字段是 Templates 列表 (GUID, + MatchMode any/all); 依赖抽取逻辑一致,
// 抽这里避免三处重复. 资产改 GUID 后无节点级格式校验 (合法性=存在性, 走 container validator_deps).
package detect

import (
	"yotta/internal/node"
)

// templateDeps 把模板 GUID 列表转成 library scanner 用的 template 依赖 (每 GUID 一条).
func templateDeps(guids []string) []node.Dependency {
	deps := make([]node.Dependency, 0, len(guids))
	for _, guid := range guids {
		if guid != "" {
			deps = append(deps, node.Dependency{Kind: "template", Key: guid})
		}
	}
	return deps
}
