package dependency

import (
	"testing"

	_ "github.com/yottaapp/yotta/internal/nodes/script"
	_ "github.com/yottaapp/yotta/internal/nodes/system"
)

func TestScanSubgraphDependenciesRecursive(t *testing.T) {
	nodes := map[string][]NodeInfo{
		"root": {
			{Kind: "Subgraph", Config: map[string]any{"literal": map[string]any{"SubgraphID": "callee"}}},
		},
	}
	got, err := ScanSubgraphDependencies("root", func(id string) ([]NodeInfo, error) {
		return nodes[id], nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !containsDep(got, Dependency{Kind: KindSubgraph, Key: "root"}) ||
		!containsDep(got, Dependency{Kind: KindSubgraph, Key: "callee"}) {
		t.Fatalf("dependencies = %+v", got)
	}
}

func TestScanSubgraphDependenciesCyclic(t *testing.T) {
	nodes := map[string][]NodeInfo{
		"A": {{Kind: "Subgraph", Config: map[string]any{"SubgraphID": "B"}}},
		"B": {{Kind: "Subgraph", Config: map[string]any{"SubgraphID": "A"}}},
	}
	got, err := ScanSubgraphDependencies("A", func(id string) ([]NodeInfo, error) {
		return nodes[id], nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !containsDep(got, Dependency{Kind: KindSubgraph, Key: "A"}) ||
		!containsDep(got, Dependency{Kind: KindSubgraph, Key: "B"}) {
		t.Fatalf("cyclic dependency missing: %+v", got)
	}
}

func TestScanSubgraphDependenciesFollowsScriptCalls(t *testing.T) {
	nodes := map[string][]NodeInfo{
		"root": {
			{Kind: "Script", Config: map[string]any{"literal": map[string]any{
				"Code": `return Subgraph({SubgraphID: "press_esc"}).exit`,
			}}},
		},
	}
	got, err := ScanSubgraphDependencies("root", func(id string) ([]NodeInfo, error) {
		return nodes[id], nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !containsDep(got, Dependency{Kind: KindSubgraph, Key: "press_esc"}) {
		t.Fatalf("script subgraph dependency missing: %+v", got)
	}
}

func containsDep(dependencies []Dependency, want Dependency) bool {
	for _, dependency := range dependencies {
		if dependency == want {
			return true
		}
	}
	return false
}
