// internal/services/script/assetdeps.go
// 脚本静态资产依赖提取 — 让依赖扫描器 / 资产 GC / 安全删除看见脚本 Code 里引用的资产 GUID。
// Script 节点没有结构化 pin 存 GUID(都在 Code 字符串里),故静态扫文本字面量。
package script

import (
	"regexp"

	"github.com/yottaapp/yotta/internal/node"
)

// guidPat 标准 uuid。clip GUID 形如 clip-<uuid>(与 clip 记录文件名同款)。
const guidPat = `[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`

// blobGUIDRe 抓 Code 里所有 (clip-)?<uuid> 字面量。template / clip 是带 blob 的资产,
// GC 漏标会误删字节 → 全文扫(覆盖 `const X = "guid"` 常量声明等任意位置,不限 pin 值位)。
// 贪婪的 (clip-)? 前缀保证 clip-<uuid> 整体算一条,内层 uuid 不再被当 template 重复命中。
var blobGUIDRe = regexp.MustCompile(`(clip-)?` + guidPat)

// subgraphCallRe 抓 Subgraph({SubgraphID: "<id>"}) 调用里的字面 SubgraphID(单双引号均可,
// key 可带引号)。SubgraphID 可为非 uuid 人名, 与全文 uuid 扫不同路, 按调用形态抽。
var subgraphCallRe = regexp.MustCompile(`Subgraph\(\s*\{[^)]*?["']?SubgraphID["']?\s*:\s*["']([^"']+)["']`)

// AssetDeps 从脚本源码静态抽资产依赖。
//   - (clip-)?<uuid> 字面量:clip- 前缀 → clip;否则 → template。
//     over-approx 安全侧:多标只会少删一个 blob(无害),漏标才会误删(有害)。
//   - Subgraph({SubgraphID: "<id>"}) → subgraph 依赖(分享导出闭包 / 删除 referrer 警告用;
//     uuid 形态的 SubgraphID 会同时被全文扫多标一条 template — over-approx 无害)。
//   - 静态正则不解析 JS 语义:动态拼接出来的 GUID / SubgraphID 扫不到(约定:用字面量,别字符串拼)。
func AssetDeps(code string) []node.Dependency {
	if code == "" {
		return nil
	}
	seen := map[string]bool{}
	var deps []node.Dependency
	add := func(kind, key string) {
		k := kind + ":" + key
		if seen[k] {
			return
		}
		seen[k] = true
		deps = append(deps, node.Dependency{Kind: kind, Key: key})
	}
	for _, m := range blobGUIDRe.FindAllStringSubmatchIndex(code, -1) {
		if m[0] >= 3 && code[m[0]-3:m[0]] == "sg-" {
			continue
		}
		kind := "template"
		key := code[m[0]:m[1]]
		if m[2] != -1 {
			kind = "clip"
		}
		add(kind, key)
	}
	for _, m := range subgraphCallRe.FindAllStringSubmatch(code, -1) {
		add("subgraph", m[1])
	}
	return deps
}
