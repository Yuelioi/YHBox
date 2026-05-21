package dependency

import (
	"reflect"
	"sort"
	"testing"

	"yhbox/internal/services/container"
)

type fakeExtractor struct{ deps []Dependency }

func (e fakeExtractor) Extract(n *container.GraphNode) []Dependency { return e.deps }

func TestScanSubgraphDependencies_FlatDeps(t *testing.T) {
	sgs := map[string]*container.Subgraph{
		"sg1": {
			Graph: container.Graph{Nodes: []container.GraphNode{
				{ID: "n1", Kind: "CheckTemplate", Config: map[string]any{"template": "ns.a"}},
				{ID: "n2", Kind: "PlayClip", Config: map[string]any{"clipID": "c1"}},
			}},
		},
	}
	get := func(id string) (*container.Subgraph, error) { return sgs[id], nil }

	extractors := map[string]Extractor{
		"CheckTemplate": fakeExtractor{deps: []Dependency{{Kind: KindTemplate, Key: "ns.a"}}},
		"PlayClip":      fakeExtractor{deps: []Dependency{{Kind: KindClip, Key: "c1"}}},
	}
	got, err := ScanSubgraphDependenciesWithExtractors("sg1", get, extractors)
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
	sgs := map[string]*container.Subgraph{
		"root": {
			Graph: container.Graph{Nodes: []container.GraphNode{
				{ID: "n1", Kind: "Subgraph", Config: map[string]any{"subgraphId": "callee"}},
			}},
		},
		"callee": {
			Graph: container.Graph{Nodes: []container.GraphNode{
				{ID: "n2", Kind: "CheckTemplate", Config: map[string]any{"template": "ns.x"}},
			}},
		},
	}
	get := func(id string) (*container.Subgraph, error) { return sgs[id], nil }
	extractors := map[string]Extractor{
		"Subgraph":      fakeExtractor{deps: []Dependency{{Kind: KindSubgraph, Key: "callee"}}},
		"CheckTemplate": fakeExtractor{deps: []Dependency{{Kind: KindTemplate, Key: "ns.x"}}},
	}
	got, err := ScanSubgraphDependenciesWithExtractors("root", get, extractors)
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
	sgs := map[string]*container.Subgraph{
		"A": {Graph: container.Graph{Nodes: []container.GraphNode{
			{ID: "n", Kind: "Subgraph", Config: map[string]any{"subgraphId": "B"}},
		}}},
		"B": {Graph: container.Graph{Nodes: []container.GraphNode{
			{ID: "n", Kind: "Subgraph", Config: map[string]any{"subgraphId": "A"}},
		}}},
	}
	get := func(id string) (*container.Subgraph, error) {
		sg, ok := sgs[id]
		if !ok {
			return nil, nil
		}
		return sg, nil
	}
	realExt := subgraphExtractorTestOnly{}
	extractors := map[string]Extractor{
		"Subgraph": realExt,
	}
	got, err := ScanSubgraphDependenciesWithExtractors("A", get, extractors)
	if err != nil {
		t.Fatal(err)
	}
	if !containsDep(got, Dependency{Kind: KindSubgraph, Key: "A"}) ||
		!containsDep(got, Dependency{Kind: KindSubgraph, Key: "B"}) {
		t.Errorf("cyclic ref dropped, got %v", got)
	}
}

type subgraphExtractorTestOnly struct{}

func (e subgraphExtractorTestOnly) Extract(n *container.GraphNode) []Dependency {
	id, _ := n.Config["subgraphId"].(string)
	if id == "" {
		return nil
	}
	return []Dependency{{Kind: KindSubgraph, Key: id}}
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
