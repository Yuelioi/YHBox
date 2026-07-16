// internal/services/script/assetdeps.go
// Script static dependency extraction for explicit Subgraph calls.
package script

import (
	"regexp"

	"github.com/yottaapp/yotta/internal/node"
)

// subgraphCallRe extracts literal Subgraph({SubgraphID: "<id>"}) calls.
var subgraphCallRe = regexp.MustCompile(`Subgraph\(\s*\{[^)]*?["']?SubgraphID["']?\s*:\s*["']([^"']+)["']`)

// Dependencies extracts literal Subgraph IDs. Blob-backed assets are values in
// the 3.1 graph contract and are never inferred from arbitrary UUID strings.
func Dependencies(code string) []node.Dependency {
	if code == "" {
		return nil
	}
	seen := map[string]bool{}
	var deps []node.Dependency
	for _, match := range subgraphCallRe.FindAllStringSubmatch(code, -1) {
		key := match[1]
		if seen[key] {
			continue
		}
		seen[key] = true
		deps = append(deps, node.Dependency{Kind: "subgraph", Key: key})
	}
	return deps
}
