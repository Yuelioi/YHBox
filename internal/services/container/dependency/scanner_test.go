package dependency

import (
	"reflect"
	"sort"
	"testing"

	// blank import 真节点 — scanner 现走 nodepkg.Get(kind).Dependencies, 不再有 fake Extractor.
	_ "yotta/internal/nodes/detect" // CheckTemplate / ClickTemplate / WaitTemplate
	_ "yotta/internal/nodes/io"     // PlayClip
	_ "yotta/internal/nodes/system" // Subgraph / CollapsedNode / Try
)

func TestScanSubgraphDependencies_FlatDeps(t *testing.T) {
	nodes := map[string][]NodeInfo{
		"sg1": {
			{Kind: "CheckTemplate", Config: map[string]any{"Template": "ns.a"}},
			{Kind: "PlayClip", Config: map[string]any{"ClipID": "c1"}},
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
			{Kind: "Subgraph", Config: map[string]any{"SubgraphID": "callee"}},
		},
		"callee": {
			{Kind: "CheckTemplate", Config: map[string]any{"Template": "ns.x"}},
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
