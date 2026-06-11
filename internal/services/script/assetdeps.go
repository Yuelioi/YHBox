// internal/services/script/assetdeps.go
// 脚本静态资产依赖提取 — 让依赖扫描器 / 资产 GC / 安全删除看见脚本 Code 里引用的资产 GUID。
// Script 节点没有结构化 pin 存 GUID(都在 Code 字符串里),故静态扫文本字面量。
package script

import (
	"regexp"

	"yotta/internal/node"
)

// guidPat 标准 uuid。clip GUID 形如 clip-<uuid>(与 clip 记录文件名同款)。
const guidPat = `[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`

// blobGUIDRe 抓 Code 里所有 (clip-)?<uuid> 字面量。template / clip 是带 blob 的资产,
// GC 漏标会误删字节 → 全文扫(覆盖 `const X = "guid"` 常量声明等任意位置,不限 pin 值位)。
// 贪婪的 (clip-)? 前缀保证 clip-<uuid> 整体算一条,内层 uuid 不再被当 template 重复命中。
var blobGUIDRe = regexp.MustCompile(`(clip-)?` + guidPat)

// AssetDeps 从脚本源码静态抽资产依赖。
//   - (clip-)?<uuid> 字面量:clip- 前缀 → clip;否则 → template。
//     over-approx 安全侧:多标只会少删一个 blob(无害),漏标才会误删(有害)。
//   - 静态正则不解析 JS 语义:动态拼接出来的 GUID 扫不到(约定:模板/clip GUID 用字面量,别字符串拼)。
//
// subgraph 引用(Subgraph({SubgraphID:...}))在 Stage 2 脚本调子图落地时按 pin 名扩 ——
// 子图无 blob、非 GC 关键,且 SubgraphID 可为非 uuid 人名,与本处全文 uuid 扫不同路。
func AssetDeps(code string) []node.Dependency {
	if code == "" {
		return nil
	}
	seen := map[string]bool{}
	var deps []node.Dependency
	for _, m := range blobGUIDRe.FindAllStringSubmatch(code, -1) {
		full, prefix := m[0], m[1]
		kind := "template"
		if prefix != "" {
			kind = "clip"
		}
		d := node.Dependency{Kind: kind, Key: full}
		k := kind + ":" + full
		if seen[k] {
			continue
		}
		seen[k] = true
		deps = append(deps, d)
	}
	return deps
}
