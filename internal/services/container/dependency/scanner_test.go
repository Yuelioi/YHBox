package dependency

import (
	"reflect"
	"sort"
	"testing"

	// blank import 真节点 — scanner 现走 nodepkg.Get(kind).Dependencies, 不再有 fake Extractor.
	_ "yotta/internal/nodes/detect" // CheckTemplate / ClickTemplate / WaitTemplate
	_ "yotta/internal/nodes/io"     // PlayClip
	_ "yotta/internal/nodes/script" // Script (资产依赖走 Code 文本扫描)
	_ "yotta/internal/nodes/system" // Subgraph / CollapsedNode
)

func TestScanSubgraphDependencies_FlatDeps(t *testing.T) {
	nodes := map[string][]NodeInfo{
		"sg1": {
			{Kind: "CheckTemplate", Config: map[string]any{"literal": map[string]any{"Templates": []any{"ns.a"}}}},
			{Kind: "PlayClip", Config: map[string]any{"literal": map[string]any{"ClipID": "c1"}}},
		},
	}
	get := func(id string) ([]NodeInfo, error) { return nodes[id], nil }

	got, err := ScanSubgraphDependencies("sg1", get)
	if err != nil {
		t.Fatal(err)
	}
	want := []Dependency{
		{Kind: KindSubgraph, Key: "sg1"},
		{Kind: KindTemplate, Key: "ns.a"},
		{Kind: KindClip, Key: "c1"},
	}
	sortDeps(got)
	sortDeps(want)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestScanSubgraphDependencies_RecursiveSubgraph(t *testing.T) {
	nodes := map[string][]NodeInfo{
		"root": {
			{Kind: "Subgraph", Config: map[string]any{"literal": map[string]any{"SubgraphID": "callee"}}},
		},
		"callee": {
			{Kind: "CheckTemplate", Config: map[string]any{"literal": map[string]any{"Templates": []any{"ns.x"}}}},
		},
	}
	get := func(id string) ([]NodeInfo, error) { return nodes[id], nil }
	got, err := ScanSubgraphDependencies("root", get)
	if err != nil {
		t.Fatal(err)
	}
	if !containsDep(got, Dependency{Kind: KindSubgraph, Key: "root"}) ||
		!containsDep(got, Dependency{Kind: KindSubgraph, Key: "callee"}) ||
		!containsDep(got, Dependency{Kind: KindTemplate, Key: "ns.x"}) {
		t.Errorf("missing expected deps in %v", got)
	}
}

func TestScanSubgraphDependencies_Cyclic(t *testing.T) {
	nodes := map[string][]NodeInfo{
		"A": {{Kind: "Subgraph", Config: map[string]any{"SubgraphID": "B"}}},
		"B": {{Kind: "Subgraph", Config: map[string]any{"SubgraphID": "A"}}},
	}
	get := func(id string) ([]NodeInfo, error) { return nodes[id], nil }
	got, err := ScanSubgraphDependencies("A", get)
	if err != nil {
		t.Fatal(err)
	}
	if !containsDep(got, Dependency{Kind: KindSubgraph, Key: "A"}) ||
		!containsDep(got, Dependency{Kind: KindSubgraph, Key: "B"}) {
		t.Errorf("cyclic ref dropped, got %v", got)
	}
}

func TestScanSubgraphDependencies_ScriptNode(t *testing.T) {
	// Script 节点的资产引用藏在 Code 字符串里 — Dependencies 走 AssetDeps 全文扫。
	nodes := map[string][]NodeInfo{
		"sg": {
			{Kind: "Script", Config: map[string]any{"literal": map[string]any{
				"Code": `const T = "3680b3d2-d31d-461c-b697-0d9c3e6a87ed";
CheckTemplate({Templates:[T]});
PlayClip({ClipID:"clip-2ba73f97-2820-4090-958a-c07dd3f8f48c"});`,
			}}},
		},
	}
	get := func(id string) ([]NodeInfo, error) { return nodes[id], nil }
	got, err := ScanSubgraphDependencies("sg", get)
	if err != nil {
		t.Fatal(err)
	}
	if !containsDep(got, Dependency{Kind: KindTemplate, Key: "3680b3d2-d31d-461c-b697-0d9c3e6a87ed"}) ||
		!containsDep(got, Dependency{Kind: KindClip, Key: "clip-2ba73f97-2820-4090-958a-c07dd3f8f48c"}) {
		t.Errorf("script asset deps not scanned, got %v", got)
	}
}

func TestScanSubgraphDependencies_ScriptCallsSubgraph(t *testing.T) {
	// 脚本 Subgraph({SubgraphID}) 调用要被扫成 subgraph 依赖, 且 BFS 跟进 callee 的资产.
	nodes := map[string][]NodeInfo{
		"sg": {
			{Kind: "Script", Config: map[string]any{"literal": map[string]any{
				"Code": `let r = Subgraph({SubgraphID: "press_esc"}); return r.exit`,
			}}},
		},
		"press_esc": {
			{Kind: "CheckTemplate", Config: map[string]any{"literal": map[string]any{"Templates": []any{"ns.esc"}}}},
		},
	}
	get := func(id string) ([]NodeInfo, error) { return nodes[id], nil }
	got, err := ScanSubgraphDependencies("sg", get)
	if err != nil {
		t.Fatal(err)
	}
	if !containsDep(got, Dependency{Kind: KindSubgraph, Key: "press_esc"}) {
		t.Errorf("script subgraph call not scanned, got %v", got)
	}
	if !containsDep(got, Dependency{Kind: KindTemplate, Key: "ns.esc"}) {
		t.Errorf("callee assets not followed (BFS), got %v", got)
	}
}

func sortDeps(d []Dependency) {
	sort.Slice(d, func(i, j int) bool { return d[i].String() < d[j].String() })
}
func containsDep(s []Dependency, d Dependency) bool {
	for _, x := range s {
		if x == d {
			return true
		}
	}
	return false
}
